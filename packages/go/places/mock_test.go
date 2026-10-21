// Author: Caden Lund
package places_test

import (
	"cmp"
	"context"
	"errors"
	"math"
	"slices"
	"testing"
	"time"

	"github.com/bengutow1/csc4330_project5/packages/go/places"
)

// campusLat and campusLng are where most tests ask from. Any point would do: the mock sets its
// spots around whatever point it is given.
const (
	campusLat = 30.4133
	campusLng = -91.1800
)

// TestTheMockSpotsSitAtTheirDistancesFromAnyPoint: the mock sets its street around the point it is
// asked about, so the list reads right anywhere. A missing cos(lat) would squash the spots far
// south, and an unwrapped longitude would push them off the map at the date line.
func TestTheMockSpotsSitAtTheirDistancesFromAnyPoint(t *testing.T) {
	t.Parallel()
	want := map[string]int{
		"Early Bird Burgers":    300,
		"Campus Corner Deli":    650,
		"Night Owl Fries":       1200,
		"Sunny Side Taco Truck": 1500,
		"Drumstick Drive-Thru":  1900,
		"Main Street Pizza":     2400,
		"Green Bowl":            2400,
		"Lucky Noodle House":    3200,
		"Slow Smoke BBQ":        4100,
		"Ember Steakhouse":      5000,
	}
	cases := []struct {
		name     string
		lat, lng float64
	}{
		{name: "campus", lat: campusLat, lng: campusLng},
		{name: "far south", lat: -54.8019, lng: -68.3030},
		{name: "beside the date line", lat: 64.8378, lng: 179.9990},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			got := nearby(t, newMock(t, ""), c.lat, c.lng)

			if len(got) != len(want) {
				t.Fatalf("got %d spots, want %d", len(got), len(want))
			}
			for _, spot := range got {
				meters, ok := want[spot.Name]
				if !ok {
					t.Errorf("got a spot nobody expected: %q", spot.Name)
					continue
				}
				if spot.DistanceMeters != meters {
					t.Errorf("%s says %d m, want %d m", spot.Name, spot.DistanceMeters, meters)
				}
				// The coordinates must put the spot where its distance says it is.
				actual := haversine(c.lat, c.lng, spot.Lat, spot.Lng)
				if math.Abs(actual-float64(meters)) > 1 {
					t.Errorf("%s sits %.1f m from the point, want %d m", spot.Name, actual, meters)
				}
				if spot.Lat < -90 || spot.Lat > 90 || spot.Lng < -180 || spot.Lng > 180 {
					t.Errorf("%s is off the map at %v, %v", spot.Name, spot.Lat, spot.Lng)
				}
			}
		})
	}
}

// TestClosedClosesEverySpot: PLACES_MOCK=closed is how the "Even the fries are closed." screen
// gets shown, so not one spot may stay open, the fries included.
func TestClosedClosesEverySpot(t *testing.T) {
	t.Parallel()

	got := nearby(t, newMock(t, "closed"), campusLat, campusLng)

	if len(got) != 10 {
		t.Fatalf("got %d spots, want all 10, closed", len(got))
	}
	for _, spot := range got {
		if spot.OpenNow {
			t.Errorf("%s is open, want every spot closed", spot.Name)
		}
		if spot.ClosesAt != "" {
			t.Errorf("%s closes at %q, want \"\" (closed now)", spot.Name, spot.ClosesAt)
		}
	}
}

// TestTheUsualStreetHasOneClosedSpotAndFriesOpenUntilTwo: the Nearest fries chip has to skip a
// closed fast-food spot that is nearer, so exactly one spot is closed, it is fast food, and the
// fries farther out stay open until 2 am.
func TestTheUsualStreetHasOneClosedSpotAndFriesOpenUntilTwo(t *testing.T) {
	t.Parallel()

	got := nearby(t, newMock(t, ""), campusLat, campusLng)

	var closed []string
	for _, spot := range got {
		if !spot.OpenNow {
			closed = append(closed, spot.Name)
		}
	}
	if !slices.Equal(closed, []string{"Early Bird Burgers"}) {
		t.Fatalf("closed = %v, want only Early Bird Burgers", closed)
	}
	fries := got[index(t, got, "Night Owl Fries")]
	if !fries.FastFood || !fries.OpenNow || fries.ClosesAt != "02:00" {
		t.Errorf("Night Owl Fries is fast food %v, open %v, closing %q; want fast food open until 02:00",
			fries.FastFood, fries.OpenNow, fries.ClosesAt)
	}
	burgers := got[index(t, got, "Early Bird Burgers")]
	if !burgers.FastFood || burgers.DistanceMeters >= fries.DistanceMeters {
		t.Errorf("Early Bird Burgers is fast food %v at %d m; want fast food nearer than the fries at %d m",
			burgers.FastFood, burgers.DistanceMeters, fries.DistanceMeters)
	}
}

// TestTheUsualStreetHasTwoCheapSpotsAndThreeWithinWalkingDistance: the Cheap and Walkable chips
// each need a short list to show, so exactly two spots have the lowest price and exactly three
// sit within 1,200 m. The rest are farther out.
func TestTheUsualStreetHasTwoCheapSpotsAndThreeWithinWalkingDistance(t *testing.T) {
	t.Parallel()

	got := nearby(t, newMock(t, ""), campusLat, campusLng)

	var cheap, walkable []string
	for _, spot := range got {
		if spot.PriceLevel == 1 {
			cheap = append(cheap, spot.Name)
		}
		if spot.DistanceMeters <= 1200 {
			walkable = append(walkable, spot.Name)
		}
	}
	slices.Sort(cheap)
	slices.Sort(walkable)
	wantCheap := []string{"Night Owl Fries", "Sunny Side Taco Truck"}
	if !slices.Equal(cheap, wantCheap) {
		t.Errorf("cheap = %v, want %v", cheap, wantCheap)
	}
	wantWalkable := []string{"Campus Corner Deli", "Early Bird Burgers", "Night Owl Fries"}
	if !slices.Equal(walkable, wantWalkable) {
		t.Errorf("within 1,200 m = %v, want %v", walkable, wantWalkable)
	}
}

// TestTheMockLeavesTheOrderToTheServer: a provider ranks its own way, and sorting by distance with
// ties broken by spot id is the server's job. The mock answers out of distance order, with its
// one tie listed against spot-id order, so a server that skips either step shows it.
func TestTheMockLeavesTheOrderToTheServer(t *testing.T) {
	t.Parallel()

	got := nearby(t, newMock(t, ""), campusLat, campusLng)

	if slices.IsSortedFunc(got, func(a, b places.Place) int {
		return cmp.Compare(a.DistanceMeters, b.DistanceMeters)
	}) {
		t.Errorf("the spots came closest first; want them out of order, for the server to sort")
	}
	bowl := index(t, got, "Green Bowl")
	pizza := index(t, got, "Main Street Pizza")
	if got[bowl].DistanceMeters != got[pizza].DistanceMeters {
		t.Errorf("Green Bowl at %d m and Main Street Pizza at %d m, want a tie",
			got[bowl].DistanceMeters, got[pizza].DistanceMeters)
	}
	if bowl > pizza || got[bowl].ID < got[pizza].ID {
		t.Errorf("Green Bowl (%q) comes at %d and Main Street Pizza (%q) at %d; want Green Bowl "+
			"first but with the greater id", got[bowl].ID, bowl, got[pizza].ID, pizza)
	}
}

// TestEveryMockSpotIsOneTheServerCanList: the server breaks ties and logs fry runs by spot id, so
// no two spots share one, and every spot has a name and a price from 0 to 4.
func TestEveryMockSpotIsOneTheServerCanList(t *testing.T) {
	t.Parallel()

	got := nearby(t, newMock(t, ""), campusLat, campusLng)

	if len(got) != 10 {
		t.Fatalf("got %d spots, want 10", len(got))
	}
	seen := map[string]string{}
	for _, spot := range got {
		if spot.ID == "" || spot.Name == "" {
			t.Errorf("spot %q named %q, want both an id and a name", spot.ID, spot.Name)
		}
		other, taken := seen[spot.ID]
		if taken {
			t.Errorf("%s and %s share the id %q", other, spot.Name, spot.ID)
		}
		seen[spot.ID] = spot.Name
		if spot.PriceLevel < 0 || spot.PriceLevel > 4 {
			t.Errorf("%s has price level %d, want 0 to 4", spot.Name, spot.PriceLevel)
		}
	}
}

// TestAnUnknownMockModeIsRefused: a typo in PLACES_MOCK must stop the boot, not quietly serve open
// spots to a run that expected every one closed.
func TestAnUnknownMockModeIsRefused(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"close", "Closed", " closed", "open"} {
		mock, err := places.NewMock(mode)
		if err == nil {
			t.Errorf("NewMock(%q) = %v, want an error", mode, mock)
		}
	}
}

// TestACanceledRequestGetsNoSpots: the mock fails a request its caller gave up on, as a provider
// would, so the server's hang-up path runs against the mock too.
func TestACanceledRequestGetsNoSpots(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	got, err := newMock(t, "").Nearby(ctx, campusLat, campusLng, "")

	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
	if got != nil {
		t.Errorf("got %d spots, want none", len(got))
	}
}

// TestChangingAnAnswerLeavesTheMockAlone: the server trims and sorts what Nearby returns, and the
// next request must still find the street as it was.
func TestChangingAnAnswerLeavesTheMockAlone(t *testing.T) {
	t.Parallel()
	mock := newMock(t, "")
	first := nearby(t, mock, campusLat, campusLng)
	first[index(t, first, "Night Owl Fries")].Hours[0].Close = 0

	again := nearby(t, mock, campusLat, campusLng)

	got := again[index(t, again, "Night Owl Fries")].Hours[0].Close
	if got != 26*time.Hour {
		t.Errorf("Night Owl Fries closes at %v after a caller edited its copy, want 26h (2 am)", got)
	}
}

func newMock(t *testing.T, mode string) *places.Mock {
	t.Helper()
	mock, err := places.NewMock(mode)
	if err != nil {
		t.Fatalf("new mock %q: %v", mode, err)
	}
	return mock
}

func nearby(t *testing.T, mock *places.Mock, lat, lng float64) []places.Place {
	t.Helper()
	got, err := mock.Nearby(t.Context(), lat, lng, "")
	if err != nil {
		t.Fatalf("nearby %v, %v: %v", lat, lng, err)
	}
	return got
}

// index is where the spot with that name sits in the list. A missing spot fails the test.
func index(t *testing.T, spots []places.Place, name string) int {
	t.Helper()
	i := slices.IndexFunc(spots, func(p places.Place) bool { return p.Name == name })
	if i < 0 {
		t.Fatalf("no spot named %q", name)
	}
	return i
}

// haversine is the great-circle distance in meters, worked out apart from the mock's own math so
// the test does not grade the mock against itself.
func haversine(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadius = 6_371_000.0
	rad := math.Pi / 180
	dLat := (lat2 - lat1) * rad
	dLng := (lng2 - lng1) * rad
	h := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * earthRadius * math.Asin(math.Sqrt(h))
}
