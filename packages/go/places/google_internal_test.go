// Author: Arthur
package places

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"
)

// campus is where these tests measure from.
var campus = &latLng{Latitude: 30.4133, Longitude: -91.18}

// campusTime is the offset of a place near campus in late October (CDT), as Google sends it.
var campusTime = time.FixedZone("CDT", -5*60*60)

// TestTheClientGivesUpOnItsOwnClock: a hung Google must not hold the player's request open even
// when the caller set no deadline, so the client carries its own timeout, and running out of it is
// ErrUnavailable.
func TestTheClientGivesUpOnItsOwnClock(t *testing.T) {
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
	c := New("test-key-not-real", srv.URL)
	if c.client.Timeout != placesTimeout {
		t.Fatalf("timeout = %v, want placesTimeout (%v)", c.client.Timeout, placesTimeout)
	}
	c.client.Timeout = 50 * time.Millisecond // placesTimeout itself would slow the run by seconds

	_, err := c.Nearby(t.Context(), 30.4133, -91.18, "")
	if !errors.Is(err, ErrUnavailable) {
		t.Errorf("got %v, want ErrUnavailable", err)
	}
}

// TestOpeningsTodayAreMeasuredFromLocalMidnight: the late-night filter reads these openings, so a
// 2 am close must count as 26h, the day must be the place's own, and a place that never closes must
// read as open through the day. Each row is a shape Google really sends.
func TestOpeningsTodayAreMeasuredFromLocalMidnight(t *testing.T) {
	t.Parallel()
	wednesdayEvening := time.Date(2026, 10, 21, 19, 0, 0, 0, campusTime)
	saturdayEvening := time.Date(2026, 10, 24, 21, 0, 0, 0, campusTime)
	cases := []struct {
		name       string
		now        time.Time
		periods    []period
		want       []Opening
		wantAllDay bool
	}{
		{
			name: "an evening that closes after midnight", now: wednesdayEvening,
			periods: []period{span(on(21, 10, 30), on(22, 2, 0))},
			want:    []Opening{{Open: hm(10, 30), Close: hm(26, 0)}},
		},
		{
			name: "a lunch and a dinner, sent dinner first", now: wednesdayEvening,
			periods: []period{span(on(21, 17, 0), on(21, 22, 0)), span(on(21, 11, 0), on(21, 14, 30))},
			want:    []Opening{{Open: hm(11, 0), Close: hm(14, 30)}, {Open: hm(17, 0), Close: hm(22, 0)}},
		},
		{
			name: "last night cut off at midnight, then tonight", now: wednesdayEvening,
			periods: []period{span(on(21, 0, 0), on(21, 2, 0)), span(on(21, 18, 0), on(22, 3, 0))},
			want:    []Opening{{Open: 0, Close: hm(2, 0)}, {Open: hm(18, 0), Close: hm(27, 0)}},
		},
		{
			name: "a day it does not open", now: wednesdayEvening,
			periods: []period{span(on(20, 8, 0), on(20, 16, 0)), span(on(22, 8, 0), on(22, 16, 0))},
		},
		{
			name: "a place that is never open", now: wednesdayEvening,
			periods: []period{},
		},
		{
			name: "a period with no close", now: wednesdayEvening,
			periods:    []period{{Open: point{}}},
			want:       []Opening{{Open: 0, Close: 24 * time.Hour}},
			wantAllDay: true,
		},
		{
			name: "one span over the whole week shown", now: wednesdayEvening,
			periods:    []period{span(on(21, 0, 0), on(28, 0, 0))},
			want:       []Opening{{Open: 0, Close: 24 * time.Hour}},
			wantAllDay: true,
		},
		{
			name: "regular hours, which carry weekdays and no dates", now: wednesdayEvening,
			periods: []period{
				{Open: point{Day: 3, Hour: 9}, Close: &point{Day: 3, Hour: 17}},
				{Open: point{Day: 3, Hour: 22}, Close: &point{Day: 4, Hour: 1}},
				{Open: point{Day: 4, Hour: 9}, Close: &point{Day: 4, Hour: 17}},
			},
			want: []Opening{{Open: hm(9, 0), Close: hm(17, 0)}, {Open: hm(22, 0), Close: hm(25, 0)}},
		},
		{
			name: "regular hours that close a week after they open", now: wednesdayEvening,
			periods:    []period{{Open: point{Day: 0}, Close: &point{Day: 0}}},
			want:       []Opening{{Open: 0, Close: 24 * time.Hour}},
			wantAllDay: true,
		},
		{
			name: "Saturday night into Sunday, by weekday", now: saturdayEvening,
			periods: []period{{Open: point{Day: 6, Hour: 20}, Close: &point{Day: 0, Hour: 2}}},
			want:    []Opening{{Open: hm(20, 0), Close: hm(26, 0)}},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			got, allDay := openingsToday(c.periods, c.now)

			if !slices.Equal(got, c.want) {
				t.Errorf("openings = %v, want %v", got, c.want)
			}
			if allDay != c.wantAllDay {
				t.Errorf("all day = %v, want %v", allDay, c.wantAllDay)
			}
		})
	}
}

// TestToPlaceReadsTheDayWhereThePlaceIs: at 10 pm in Baton Rouge it is already tomorrow in UTC.
// Reading the day or the closing time in UTC would show tomorrow's hours and a 04:00 close.
func TestToPlaceReadsTheDayWhereThePlaceIs(t *testing.T) {
	t.Parallel()
	tenPMLocal := time.Date(2026, 10, 22, 3, 0, 0, 0, time.UTC)
	closes := time.Date(2026, 10, 22, 4, 0, 0, 0, time.UTC)
	g := aPlace()
	g.CurrentOpeningHours = &openingHours{
		OpenNow: true,
		Periods: []period{
			span(on(21, 17, 0), on(21, 23, 0)),
			span(on(22, 6, 0), on(22, 9, 0)),
		},
		NextCloseTime: &closes,
	}

	got, err := toPlace(g, campus, tenPMLocal)
	if err != nil {
		t.Fatalf("to place: %v", err)
	}
	want := []Opening{{Open: hm(17, 0), Close: hm(23, 0)}}
	if !slices.Equal(got.Hours, want) {
		t.Errorf("hours = %v, want %v (the 21st, the place's own day)", got.Hours, want)
	}
	if !got.OpenNow || got.ClosesAt != "23:00" {
		t.Errorf("open=%v closes %q, want open until 23:00", got.OpenNow, got.ClosesAt)
	}
}

// TestAPlaceThatNeverClosesShowsNoClosingTime: current hours write a place open around the clock
// as one span to the edge of the week they cover, with that edge as the next close. "Open until
// 00:00" would be a lie on six days out of seven.
func TestAPlaceThatNeverClosesShowsNoClosingTime(t *testing.T) {
	t.Parallel()
	weekEdge := time.Date(2026, 10, 28, 5, 0, 0, 0, time.UTC) // midnight on the 28th, campus time
	g := aPlace()
	g.CurrentOpeningHours = &openingHours{
		OpenNow:       true,
		Periods:       []period{span(on(21, 0, 0), on(28, 0, 0))},
		NextCloseTime: &weekEdge,
	}

	got, err := toPlace(g, campus, time.Date(2026, 10, 21, 19, 0, 0, 0, campusTime))
	if err != nil {
		t.Fatalf("to place: %v", err)
	}
	if !got.OpenNow || got.ClosesAt != "" {
		t.Errorf("open=%v closes %q, want open with no closing time", got.OpenNow, got.ClosesAt)
	}
	want := []Opening{{Open: 0, Close: 24 * time.Hour}}
	if !slices.Equal(got.Hours, want) {
		t.Errorf("hours = %v, want %v", got.Hours, want)
	}
}

// TestAPlaceWithNoOffsetShowsNoHours: without its offset from UTC there is no knowing which day is
// today at the place, so it shows no hours and no closing time rather than guessing. Whether it is
// open now still comes from Google.
func TestAPlaceWithNoOffsetShowsNoHours(t *testing.T) {
	t.Parallel()
	closes := time.Date(2026, 10, 22, 4, 0, 0, 0, time.UTC)
	g := aPlace()
	g.UTCOffsetMinutes = nil
	g.CurrentOpeningHours = &openingHours{
		OpenNow:       true,
		Periods:       []period{span(on(21, 17, 0), on(21, 23, 0))},
		NextCloseTime: &closes,
	}

	got, err := toPlace(g, campus, time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("to place: %v", err)
	}
	if got.Hours != nil || got.ClosesAt != "" {
		t.Errorf("hours %v, closes %q; want none of either", got.Hours, got.ClosesAt)
	}
	if !got.OpenNow {
		t.Error("open = false, want Google's openNow")
	}
}

// TestEveryPriceLevelLandsOnOneToFour: the Cheap filter keeps price 1, so every level Google names
// must land on our scale, and a level nobody expected must fail loudly instead of passing as
// unknown.
func TestEveryPriceLevelLandsOnOneToFour(t *testing.T) {
	t.Parallel()
	cases := []struct {
		level string
		want  int
	}{
		{level: "", want: 0},
		{level: "PRICE_LEVEL_UNSPECIFIED", want: 0},
		{level: "PRICE_LEVEL_FREE", want: 1},
		{level: "PRICE_LEVEL_INEXPENSIVE", want: 1},
		{level: "PRICE_LEVEL_MODERATE", want: 2},
		{level: "PRICE_LEVEL_EXPENSIVE", want: 3},
		{level: "PRICE_LEVEL_VERY_EXPENSIVE", want: 4},
	}
	for _, c := range cases {
		t.Run(c.level, func(t *testing.T) {
			t.Parallel()

			got, err := priceLevel(c.level)

			if err != nil || got != c.want {
				t.Errorf("got %d (%v), want %d", got, err, c.want)
			}
		})
	}

	_, err := priceLevel("PRICE_LEVEL_PRICELESS")
	if err == nil {
		t.Error("an unknown level passed, want an error")
	}
}

// TestCategoryIsTheKindOfFood: the spots list shows the kind of food before the price, so
// Google's type name loses its "Restaurant" and a plain Restaurant, which names no kind, shows
// nothing.
func TestCategoryIsTheKindOfFood(t *testing.T) {
	t.Parallel()
	cases := []struct{ primaryType, want string }{
		{primaryType: "Mexican Restaurant", want: "Mexican"},
		{primaryType: "Fast Food Restaurant", want: "Fast Food"},
		{primaryType: "Pizza Restaurant", want: "Pizza"},
		{primaryType: "Restaurant", want: ""},
		{primaryType: "Sandwich Shop", want: "Sandwich Shop"},
		{primaryType: "", want: ""},
	}
	for _, c := range cases {
		t.Run(c.primaryType, func(t *testing.T) {
			t.Parallel()

			got := category(c.primaryType)

			if got != c.want {
				t.Errorf("category(%q) = %q, want %q", c.primaryType, got, c.want)
			}
		})
	}
}

// TestDistanceMetersHoldsAcrossTheGlobe: the distance is the great circle at any range, from the
// player's own spot to the far side of the world. A flat-map shortcut drifts with distance and
// breaks past the date line.
func TestDistanceMetersHoldsAcrossTheGlobe(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name                   string
		lat1, lng1, lat2, lng2 float64
		want                   int
	}{
		{name: "the same point", lat1: 30.4133, lng1: -91.18, lat2: 30.4133, lng2: -91.18, want: 0},
		{name: "1,200 m due north", lat1: 30.4133, lng1: -91.18, lat2: 30.424092, lng2: -91.18, want: 1200},
		{name: "the far side of the world", lat1: 0, lng1: 0, lat2: 0, lng2: 180, want: 20015087},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			got := distanceMeters(c.lat1, c.lng1, c.lat2, c.lng2)

			if got != c.want {
				t.Errorf("got %d m, want %d m", got, c.want)
			}
		})
	}
}

// aPlace is one of Google's places near campus with no hours yet.
func aPlace() googlePlace {
	offset := -5 * 60
	g := googlePlace{ID: "test-place", Location: &latLng{Latitude: 30.4133, Longitude: -91.18}}
	g.DisplayName.Text = "Test Place"
	g.UTCOffsetMinutes = &offset
	return g
}

// on is a point in current hours: day of October 2026 at hour:minute, the way Google dates it.
func on(day, hour, minute int) point {
	date := time.Date(2026, 10, day, 0, 0, 0, 0, time.UTC)
	return point{
		Day: int(date.Weekday()), Hour: hour, Minute: minute,
		Date: &civilDate{Year: 2026, Month: 10, Day: day},
	}
}

// span is a period from open to close.
func span(open, closes point) period {
	return period{Open: open, Close: &closes}
}

// hm is a time of day as a duration from midnight. Past midnight keeps counting: hm(26, 0) is 2 am.
func hm(hour, minute int) time.Duration {
	return time.Duration(hour)*time.Hour + time.Duration(minute)*time.Minute
}
