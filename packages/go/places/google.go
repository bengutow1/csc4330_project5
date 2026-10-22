// Author: Arthur
package places

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"slices"
	"strings"
	"time"
)

// GoogleBaseURL is where Google serves Places API (New). New uses it when given no base URL.
const GoogleBaseURL = "https://places.googleapis.com"

const (
	// placesTimeout bounds one call to Google. A search answers in well under a second, and a
	// player on the spots screen should see "try again" long before they give up on it.
	placesTimeout = 6 * time.Second
	// searchRadius is how far from the player a search looks, in meters: a short drive. Twenty
	// spots ranked by distance near campus fall well inside it.
	searchRadius = 5000.0
	// pageSize is the most places one call asks for, which is Google's own cap.
	pageSize = 20
	// maxBody caps how much of an answer is read. Twenty places with their hours come to tens of
	// kilobytes, so anything near this is not an answer to our question.
	maxBody = 1 << 20
)

// fieldMask asks Google for only what Place needs. Google bills a call at its costliest field, and
// the hours and the price already put every call in its Enterprise tier, so nothing else rides
// along.
const fieldMask = "places.id,places.displayName,places.location,places.shortFormattedAddress," +
	"places.types,places.priceLevel,places.currentOpeningHours,places.utcOffsetMinutes"

// foodTypes are the Google place types a search around the player asks for: somewhere to eat.
var foodTypes = []string{"restaurant", "meal_takeaway"}

// Client finds food near a point through Google Places API (New). The API serves it once
// PLACES_API_KEY is set.
type Client struct {
	key     string
	baseURL string
	client  *http.Client
	now     func() time.Time
}

// New builds a client that sends key with every call. baseURL is "" for Google itself; a test
// points it at its own server.
func New(key, baseURL string) *Client {
	if baseURL == "" {
		baseURL = GoogleBaseURL
	}
	return &Client{
		key:     key,
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: placesTimeout},
		now:     time.Now,
	}
}

// Nearby finds places to eat around lat, lng, each with its straight-line distance from that
// point. An empty query asks for any food nearby, closest first; otherwise query says what to look
// for and Google looks around the point first. Every failure, a timeout included, is
// ErrUnavailable.
func (c *Client) Nearby(ctx context.Context, lat, lng float64, query string) ([]Place, error) {
	around := area{Circle: circle{Center: latLng{Latitude: lat, Longitude: lng}, Radius: searchRadius}}
	var (
		path string
		body any
	)
	if query == "" {
		path, body = "/v1/places:searchNearby", nearbyRequest{
			IncludedTypes:       foodTypes,
			MaxResultCount:      pageSize,
			RankPreference:      "DISTANCE",
			LanguageCode:        "en",
			LocationRestriction: around,
		}
	} else {
		// Ranked by relevance, Google's advice for words like "near the stadium". The handler
		// orders by distance anyway.
		path, body = "/v1/places:searchText", textRequest{
			TextQuery:    query,
			IncludedType: "restaurant",
			PageSize:     pageSize,
			LanguageCode: "en",
			LocationBias: around,
		}
	}

	found, err := c.search(ctx, path, body)
	if err != nil {
		return nil, err
	}
	now := c.now()
	out := make([]Place, 0, len(found))
	for i, g := range found {
		var p Place
		p, err = toPlace(g, lat, lng, now)
		if err != nil {
			return nil, unavailable(fmt.Sprintf("%s place %d", path, i), err)
		}
		out = append(out, p)
	}
	return out, nil
}

// search posts one request to Google and decodes the places it answers with.
func (c *Client) search(ctx context.Context, path string, body any) ([]googlePlace, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, unavailable(path, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return nil, unavailable(path, err)
	}
	req.Header.Set("Content-Type", "application/json")
	// The key rides a header, never the URL, so no error message can carry it.
	req.Header.Set("X-Goog-Api-Key", c.key)
	req.Header.Set("X-Goog-FieldMask", fieldMask)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, unavailable(path, err)
	}
	defer func() { _ = resp.Body.Close() }() // read-only: closing it has nothing to report

	if resp.StatusCode != http.StatusOK {
		// The status alone: Google's error text can repeat the request, the player's point
		// included.
		return nil, unavailable(path, fmt.Errorf("status %d", resp.StatusCode))
	}
	var out searchResponse
	err = json.NewDecoder(io.LimitReader(resp.Body, maxBody)).Decode(&out)
	if err != nil {
		return nil, unavailable(path, fmt.Errorf("decode: %w", err))
	}
	return out.Places, nil
}

// unavailable makes any failed call ErrUnavailable. The cause rides along as text only, so a
// timeout reaches the player as "try again" and not as their own deadline running out.
func unavailable(op string, err error) error {
	return fmt.Errorf("places: google %s: %w (%s)", op, ErrUnavailable, err.Error())
}

// toPlace turns one of Google's places into a Place measured from lat, lng. now decides which day
// is today where the place is.
func toPlace(g googlePlace, lat, lng float64, now time.Time) (Place, error) {
	if g.ID == "" || g.DisplayName.Text == "" || g.Location == nil {
		return Place{}, errors.New("a place without its id, name or location")
	}
	price, err := priceLevel(g.PriceLevel)
	if err != nil {
		return Place{}, err
	}
	p := Place{
		ID:             g.ID,
		Name:           g.DisplayName.Text,
		DistanceMeters: distanceMeters(lat, lng, g.Location.Latitude, g.Location.Longitude),
		PriceLevel:     price,
		FastFood:       slices.Contains(g.Types, "fast_food_restaurant"),
		Lat:            g.Location.Latitude,
		Lng:            g.Location.Longitude,
		Address:        g.ShortFormattedAddress,
	}

	h := g.CurrentOpeningHours
	if h == nil {
		return p, nil // no posted hours: not known to be open
	}
	p.OpenNow = h.OpenNow
	// Without the place's offset from UTC there is no telling which day is today there.
	if g.UTCOffsetMinutes == nil {
		return p, nil
	}
	zone := time.FixedZone("", *g.UTCOffsetMinutes*60)
	hours, allDay := openingsToday(h.Periods, now.In(zone))
	p.Hours = hours
	if p.OpenNow && !allDay && h.NextCloseTime != nil {
		p.ClosesAt = h.NextCloseTime.In(zone).Format("15:04")
	}
	return p, nil
}

// openingsToday picks the periods that open on now's day, measured from that day's midnight, with
// a close after midnight counted past 24h. allDay reports a place open around the clock, which is
// one opening from 0h to 24h.
func openingsToday(periods []period, now time.Time) (out []Opening, allDay bool) {
	// Wall-clock times on a plain UTC axis, so every day is 24h long: now already carries the
	// place's offset.
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	dayEnd := midnight.Add(24 * time.Hour)
	aroundTheClock := []Opening{{Open: 0, Close: 24 * time.Hour}}

	for _, pd := range periods {
		// A period with no close is how Google writes a place that never closes.
		if pd.Close == nil {
			return aroundTheClock, true
		}
		opens := pd.Open.wallClock(midnight)
		closes := pd.Close.wallClock(startOfDay(opens))
		switch {
		case !closes.After(opens):
			// Regular hours carry no dates, so a close that lands at or before its open is the
			// same weekday a week on: open around the clock.
			return aroundTheClock, true
		case !opens.After(midnight) && !closes.Before(dayEnd):
			// Open through the whole of today, which is also how current hours write a place that
			// never closes: one span cut off at the edges of the week they cover.
			return aroundTheClock, true
		case !opens.Before(midnight) && opens.Before(dayEnd):
			out = append(out, Opening{Open: opens.Sub(midnight), Close: closes.Sub(midnight)})
		}
	}
	slices.SortFunc(out, func(a, b Opening) int { return cmp.Compare(a.Open, b.Open) })
	return out, false
}

// wallClock is when a point falls on the wall-clock axis openingsToday uses. A point with a date
// is exact. One without (regular hours carry none) lands on the first such weekday from day on.
func (pt point) wallClock(day time.Time) time.Time {
	if pt.Date != nil {
		day = time.Date(pt.Date.Year, time.Month(pt.Date.Month), pt.Date.Day, 0, 0, 0, 0, time.UTC)
	} else {
		day = day.AddDate(0, 0, (pt.Day-int(day.Weekday())+7)%7)
	}
	return day.Add(time.Duration(pt.Hour)*time.Hour + time.Duration(pt.Minute)*time.Minute)
}

// startOfDay is midnight on t's day.
func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// priceLevel turns Google's price level into 1 (cheap) through 4, or 0 when Google does not know.
func priceLevel(level string) (int, error) {
	switch level {
	case "", "PRICE_LEVEL_UNSPECIFIED":
		return 0, nil
	case "PRICE_LEVEL_FREE", "PRICE_LEVEL_INEXPENSIVE":
		// Free is as cheap as it gets, so it shares the cheapest level.
		return 1, nil
	case "PRICE_LEVEL_MODERATE":
		return 2, nil
	case "PRICE_LEVEL_EXPENSIVE":
		return 3, nil
	case "PRICE_LEVEL_VERY_EXPENSIVE":
		return 4, nil
	default:
		return 0, fmt.Errorf("unknown price level %q", level)
	}
}

// distanceMeters is the great-circle distance between two points, to the nearest meter.
func distanceMeters(lat1, lng1, lat2, lng2 float64) int {
	const rad = math.Pi / 180
	dLat := (lat2 - lat1) * rad
	dLng := (lng2 - lng1) * rad
	h := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLng/2)*math.Sin(dLng/2)
	return int(math.Round(2 * earthRadius * math.Asin(math.Sqrt(h))))
}

// The wire shapes below are Google's, cut down to what this client sends and reads.

type nearbyRequest struct {
	IncludedTypes       []string `json:"includedTypes"`
	MaxResultCount      int      `json:"maxResultCount"`
	RankPreference      string   `json:"rankPreference"`
	LanguageCode        string   `json:"languageCode"`
	LocationRestriction area     `json:"locationRestriction"`
}

type textRequest struct {
	TextQuery    string `json:"textQuery"`
	IncludedType string `json:"includedType"`
	PageSize     int    `json:"pageSize"`
	LanguageCode string `json:"languageCode"`
	LocationBias area   `json:"locationBias"`
}

type area struct {
	Circle circle `json:"circle"`
}

type circle struct {
	Center latLng  `json:"center"`
	Radius float64 `json:"radius"` // meters
}

type latLng struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type searchResponse struct {
	Places []googlePlace `json:"places"` // absent when nothing was found
}

type googlePlace struct {
	ID          string `json:"id"`
	DisplayName struct {
		Text string `json:"text"`
	} `json:"displayName"`
	Location              *latLng       `json:"location"`
	ShortFormattedAddress string        `json:"shortFormattedAddress"`
	Types                 []string      `json:"types"`
	PriceLevel            string        `json:"priceLevel"`
	CurrentOpeningHours   *openingHours `json:"currentOpeningHours"`
	// UTCOffsetMinutes is a pointer because 0 is a real offset (London in winter).
	UTCOffsetMinutes *int `json:"utcOffsetMinutes"`
}

type openingHours struct {
	OpenNow bool     `json:"openNow"`
	Periods []period `json:"periods"`
	// NextCloseTime is set only while the place is open.
	NextCloseTime *time.Time `json:"nextCloseTime"`
}

type period struct {
	Open  point  `json:"open"`
	Close *point `json:"close"`
}

// point is a moment in a period, in the place's own time. Google leaves out any field that is 0.
type point struct {
	Day    int        `json:"day"` // 0 is Sunday
	Hour   int        `json:"hour"`
	Minute int        `json:"minute"`
	Date   *civilDate `json:"date"`
}

type civilDate struct {
	Year  int `json:"year"`
	Month int `json:"month"`
	Day   int `json:"day"`
}
