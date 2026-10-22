// Author: Arthur
package places_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bengutow1/csc4330_project5/packages/go/places"
)

// testKey stands in for PLACES_API_KEY. No test calls Google, so it is never a real key.
const testKey = "test-key-not-real"

// fieldMask is every field a search may ask Google for, written out by hand: Google bills a call
// by its costliest field, so a change here is a change to the bill.
const fieldMask = "places.id,places.displayName,places.location,places.shortFormattedAddress," +
	"places.types,places.priceLevel,places.currentOpeningHours,places.utcOffsetMinutes"

// TestAGoodPageBecomesPlacesMeasuredFromThePlayer: every field the spots screen shows comes from
// Google's answer, the distance from the player's point, the price on our 1 to 4 scale and the
// closing time on the place's own clock. A wrong mapping would list a steakhouse as cheap or say a
// late-night spot closes at 07:00.
func TestAGoodPageBecomesPlacesMeasuredFromThePlayer(t *testing.T) {
	t.Parallel()
	client, _ := replaying(t, http.StatusOK, "google_nearby.json")

	got, err := client.Nearby(t.Context(), campusLat, campusLng, "")
	if err != nil {
		t.Fatalf("nearby: %v", err)
	}

	want := []places.Place{
		{
			ID: "test-fries", Name: "Midnight Fry Shack", DistanceMeters: 1200,
			OpenNow: true, ClosesAt: "02:00", PriceLevel: 1, FastFood: true,
			Lat: 30.424092, Lng: -91.18, Address: "77 Maple Dr, Baton Rouge",
		},
		{
			ID: "test-noodles", Name: "Corner Noodle Bar", DistanceMeters: 650,
			OpenNow: true, ClosesAt: "22:00", PriceLevel: 2,
			Lat: 30.407454, Lng: -91.18, Address: "402 Oak Ave, Baton Rouge",
		},
		{
			// Closed, and Google sends no price level for it.
			ID: "test-diner", Name: "Early Bird Diner", DistanceMeters: 300,
			Lat: 30.415998, Lng: -91.18, Address: "18 Elm St, Baton Rouge",
		},
		{
			// Open around the clock, so there is no closing time to show.
			ID: "test-grill", Name: "Round The Clock Grill", DistanceMeters: 2400,
			OpenNow: true, PriceLevel: 3,
			Lat: 30.391716, Lng: -91.18, Address: "560 Main St, Baton Rouge",
			Hours: []places.Opening{{Open: 0, Close: 24 * time.Hour}},
		},
		{
			// No posted hours and no address: not known to be open.
			ID: "test-cart", Name: "Sunny Taco Cart", DistanceMeters: 1500, PriceLevel: 1,
			Lat: 30.42679, Lng: -91.18,
		},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d places, want %d", len(got), len(want))
	}
	for i := range want {
		samePlace(t, got[i], want[i])
	}
	// The grill's hours hold on any day. The others depend on today's date, which
	// google_internal_test.go pins.
	if !slices.Equal(got[3].Hours, want[3].Hours) {
		t.Errorf("grill hours = %v, want %v", got[3].Hours, want[3].Hours)
	}
}

// TestASearchAsksGoogleForOnlyWhatPlaceNeeds: Google bills a call by the fields it asks for, so a
// field nobody reads costs money on every search. The key rides a header, never the URL, where a
// proxy or an error message could print it.
func TestASearchAsksGoogleForOnlyWhatPlaceNeeds(t *testing.T) {
	t.Parallel()
	client, wire := replaying(t, http.StatusOK, "google_nearby.json")

	_, err := client.Nearby(t.Context(), campusLat, campusLng, "")
	if err != nil {
		t.Fatalf("nearby: %v", err)
	}

	call := wire.only(t)
	if call.method != http.MethodPost || call.path != "/v1/places:searchNearby" {
		t.Errorf("sent %s %s, want POST /v1/places:searchNearby", call.method, call.path)
	}
	if call.rawQuery != "" {
		t.Errorf("the URL carries %q, want nothing in it", call.rawQuery)
	}
	wantHeader(t, call, "X-Goog-FieldMask", fieldMask)
	wantHeader(t, call, "X-Goog-Api-Key", testKey)
	wantHeader(t, call, "Content-Type", "application/json")

	b := call.body
	if !slices.Equal(b.IncludedTypes, []string{"restaurant", "meal_takeaway"}) {
		t.Errorf("included types = %v, want restaurant and meal_takeaway", b.IncludedTypes)
	}
	if b.MaxResultCount != 20 || b.RankPreference != "DISTANCE" {
		t.Errorf("asked for %d ranked by %q, want 20 ranked by DISTANCE", b.MaxResultCount, b.RankPreference)
	}
	if b.LocationRestriction == nil {
		t.Fatal("no locationRestriction, want a circle around the player")
	}
	aroundCampus(t, *b.LocationRestriction)
}

// TestWordsSearchByTextAroundThePoint: "tacos" or "near the stadium" is a text search, not a
// nearby one, and it still looks around the player first. Sending the words to the nearby search
// would drop them, because that endpoint has no query.
func TestWordsSearchByTextAroundThePoint(t *testing.T) {
	t.Parallel()
	client, wire := replaying(t, http.StatusOK, "google_nearby.json")

	got, err := client.Nearby(t.Context(), campusLat, campusLng, "tacos")
	if err != nil {
		t.Fatalf("nearby: %v", err)
	}
	if len(got) != 5 {
		t.Errorf("got %d places, want the page's 5", len(got))
	}

	call := wire.only(t)
	if call.method != http.MethodPost || call.path != "/v1/places:searchText" {
		t.Errorf("sent %s %s, want POST /v1/places:searchText", call.method, call.path)
	}
	wantHeader(t, call, "X-Goog-Api-Key", testKey)
	wantHeader(t, call, "X-Goog-FieldMask", fieldMask)
	b := call.body
	if b.TextQuery != "tacos" || b.IncludedType != "restaurant" || b.PageSize != 20 {
		t.Errorf("sent query %q, type %q, page size %d; want tacos, restaurant, 20",
			b.TextQuery, b.IncludedType, b.PageSize)
	}
	if b.LocationBias == nil {
		t.Fatal("no locationBias, want a circle around the player")
	}
	aroundCampus(t, *b.LocationBias)
}

// TestAnEmptyPageIsNoSpots: Google answers {} when nothing is near, which is an empty street for
// "Even the fries are closed.", not a failure.
func TestAnEmptyPageIsNoSpots(t *testing.T) {
	t.Parallel()
	client, _ := replaying(t, http.StatusOK, "google_empty.json")

	got, err := client.Nearby(t.Context(), campusLat, campusLng, "")
	if err != nil {
		t.Fatalf("nearby: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %d places, want none", len(got))
	}
}

// TestABadAnswerIsUnavailable: anything that is not the answer we asked for fails the whole call
// as ErrUnavailable, which the player sees as "try again". A partial list would hide spots
// without saying so.
func TestABadAnswerIsUnavailable(t *testing.T) {
	t.Parallel()
	malformed, err := os.ReadFile("testdata/google_malformed.json")
	if err != nil {
		t.Fatalf("read the malformed body: %v", err)
	}
	cases := []struct {
		name   string
		status int
		body   string
	}{
		{name: "a malformed body", status: http.StatusOK, body: string(malformed)},
		{name: "places that are not a list", status: http.StatusOK, body: `{"places": {"id": "test-fries"}}`},
		{
			name: "a place with no id", status: http.StatusOK,
			body: `{"places": [{"displayName": {"text": "Nameless"}, "location": {"latitude": 30.41, "longitude": -91.18}}]}`,
		},
		{
			name: "a place with no name", status: http.StatusOK,
			body: `{"places": [{"id": "test-x", "location": {"latitude": 30.41, "longitude": -91.18}}]}`,
		},
		{
			name: "a place with no location", status: http.StatusOK,
			body: `{"places": [{"id": "test-x", "displayName": {"text": "Nowhere"}}]}`,
		},
		{
			name: "a price level nobody expected", status: http.StatusOK,
			body: `{"places": [{"id": "test-x", "displayName": {"text": "X"}, ` +
				`"location": {"latitude": 30.41, "longitude": -91.18}, "priceLevel": "PRICE_LEVEL_PRICELESS"}]}`,
		},
		{name: "a server error", status: http.StatusInternalServerError, body: `{}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			client, _ := answering(t, c.status, []byte(c.body))

			got, err := client.Nearby(t.Context(), campusLat, campusLng, "")
			if !errors.Is(err, places.ErrUnavailable) {
				t.Errorf("got %v, want ErrUnavailable", err)
			}
			if got != nil {
				t.Errorf("got %d places alongside the error, want none", len(got))
			}
		})
	}
}

// TestARefusalNeverRepeatsThePointOrTheKey: the handler logs this error, and Google's error text
// can quote the request it refused. The player's location is never logged, and neither is the
// key. The point's digits are ones nothing else in the run could spell.
func TestARefusalNeverRepeatsThePointOrTheKey(t *testing.T) {
	t.Parallel()
	const lat, lng = 12.3456789, -98.7654321
	echo := `{"error": {"code": 400, "status": "INVALID_ARGUMENT", "message": ` +
		`"Invalid value at 'location_restriction.circle.center' (12.3456789, -98.7654321) for key test-key-not-real"}}`
	for _, status := range []int{http.StatusBadRequest, http.StatusForbidden, http.StatusTooManyRequests} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			t.Parallel()
			client, _ := answering(t, status, []byte(echo))

			_, err := client.Nearby(t.Context(), lat, lng, "")
			if !errors.Is(err, places.ErrUnavailable) {
				t.Fatalf("got %v, want ErrUnavailable", err)
			}
			for _, secret := range []string{"3456789", "7654321", testKey} {
				if strings.Contains(err.Error(), secret) {
					t.Errorf("the error %q carries %q", err, secret)
				}
			}
		})
	}
}

// TestATimeoutIsUnavailable: a Google that never answers becomes ErrUnavailable, so the player is
// told to try again. If the deadline itself stayed in the chain, ConnectErr would match it first
// and answer deadline_exceeded instead.
func TestATimeoutIsUnavailable(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Reading the body first lets the server notice the client hanging up.
		_, _ = io.Copy(io.Discard, r.Body)
		select {
		case <-r.Context().Done(): // the client gave up
		case <-time.After(5 * time.Second): // a backstop, so a broken client cannot hang the run
		}
	}))
	t.Cleanup(srv.Close)
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()

	_, err := places.New(testKey, srv.URL).Nearby(ctx, campusLat, campusLng, "")
	if !errors.Is(err, places.ErrUnavailable) {
		t.Errorf("got %v, want ErrUnavailable", err)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("got %v, want the deadline left out of the chain", err)
	}
}

// replaying starts a fake Google that answers every call with status and the named testdata file.
func replaying(t *testing.T, status int, file string) (*places.Client, *googleWire) {
	t.Helper()
	body, err := os.ReadFile("testdata/" + file)
	if err != nil {
		t.Fatalf("read %s: %v", file, err)
	}
	return answering(t, status, body)
}

// answering starts a fake Google that answers every call with status and body.
func answering(t *testing.T, status int, body []byte) (*places.Client, *googleWire) {
	t.Helper()
	wire := &googleWire{status: status, body: body}
	srv := httptest.NewServer(wire)
	t.Cleanup(srv.Close)
	return places.New(testKey, srv.URL), wire
}

// googleWire stands in for Google, the one true edge of this package: it replays one answer to
// every call and records what each call sent. It is safe because the answers follow the shape
// Google documents, and the e2e task checks the client against the real thing.
type googleWire struct {
	status int
	body   []byte

	mu    sync.Mutex
	calls []sentCall
}

// sentCall is what one request to the fake carried.
type sentCall struct {
	method, path, rawQuery string
	header                 http.Header
	body                   sentBody
}

// sentBody is the request body, read the way Google reads it.
type sentBody struct {
	IncludedTypes       []string  `json:"includedTypes"`
	MaxResultCount      int       `json:"maxResultCount"`
	RankPreference      string    `json:"rankPreference"`
	TextQuery           string    `json:"textQuery"`
	IncludedType        string    `json:"includedType"`
	PageSize            int       `json:"pageSize"`
	LocationRestriction *sentArea `json:"locationRestriction"`
	LocationBias        *sentArea `json:"locationBias"`
}

type sentArea struct {
	Circle struct {
		Center struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		} `json:"center"`
		Radius float64 `json:"radius"`
	} `json:"circle"`
}

func (g *googleWire) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	call := sentCall{method: r.Method, path: r.URL.Path, rawQuery: r.URL.RawQuery, header: r.Header.Clone()}
	raw, err := io.ReadAll(r.Body)
	if err == nil {
		// A body that does not decode leaves call.body empty, which the request tests catch.
		_ = json.Unmarshal(raw, &call.body)
	}
	g.mu.Lock()
	g.calls = append(g.calls, call)
	g.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(g.status)
	_, _ = w.Write(g.body) // a client that hung up has nothing left to read
}

// only is the one call the fake saw.
func (g *googleWire) only(t *testing.T) sentCall {
	t.Helper()
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.calls) != 1 {
		t.Fatalf("got %d calls to Google, want 1", len(g.calls))
	}
	return g.calls[0]
}

// wantHeader checks one header the fake received.
func wantHeader(t *testing.T, call sentCall, name, want string) {
	t.Helper()
	got := call.header.Get(name)
	if got != want {
		t.Errorf("%s = %q, want %q", name, got, want)
	}
}

// aroundCampus checks that a search area is the 5 km circle around the campus point.
func aroundCampus(t *testing.T, a sentArea) {
	t.Helper()
	c := a.Circle
	if c.Center.Latitude != campusLat || c.Center.Longitude != campusLng || c.Radius != 5000 {
		t.Errorf("searched %v, %v within %v m; want %v, %v within 5000 m",
			c.Center.Latitude, c.Center.Longitude, c.Radius, campusLat, campusLng)
	}
}

// samePlace compares every field of a Place but its hours.
func samePlace(t *testing.T, got, want places.Place) {
	t.Helper()
	if got.ID != want.ID || got.Name != want.Name || got.Address != want.Address {
		t.Errorf("got %q %q at %q, want %q %q at %q", got.ID, got.Name, got.Address, want.ID, want.Name, want.Address)
	}
	if got.DistanceMeters != want.DistanceMeters {
		t.Errorf("%s is %d m away, want %d m", want.ID, got.DistanceMeters, want.DistanceMeters)
	}
	if got.OpenNow != want.OpenNow || got.ClosesAt != want.ClosesAt {
		t.Errorf("%s open=%v closes %q, want open=%v closes %q",
			want.ID, got.OpenNow, got.ClosesAt, want.OpenNow, want.ClosesAt)
	}
	if got.PriceLevel != want.PriceLevel || got.FastFood != want.FastFood {
		t.Errorf("%s price %d fast food %v, want price %d fast food %v",
			want.ID, got.PriceLevel, got.FastFood, want.PriceLevel, want.FastFood)
	}
	if got.Lat != want.Lat || got.Lng != want.Lng {
		t.Errorf("%s sits at %v, %v, want %v, %v", want.ID, got.Lat, got.Lng, want.Lat, want.Lng)
	}
}
