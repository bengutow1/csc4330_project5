// Author: Caden Lund
package places

import (
	"context"
	"fmt"
	"math"
	"slices"
	"time"
)

// Mock is the places provider the API serves without a PLACES_API_KEY. It knows ten made-up spots
// and sets them at fixed offsets around whatever point it is asked about, so their distances read
// right anywhere. The tests use the same spots, so tests and local dev agree.
type Mock struct {
	closed bool // every spot closed, for "Even the fries are closed."
}

// NewMock builds the mock for a mode: "" for the usual spots, or "closed" to close every one of
// them. Any other mode is an error, so a typo never quietly serves open spots.
func NewMock(mode string) (*Mock, error) {
	switch mode {
	case "":
		return &Mock{}, nil
	case "closed":
		return &Mock{closed: true}, nil
	default:
		return nil, fmt.Errorf("places: unknown mock mode %q, want closed or none", mode)
	}
}

// Nearby sets the ten spots around lat, lng and returns them all, out of distance order, the way
// a provider ranks by its own measure. It ignores query: the ten stand in for any search.
func (c *Mock) Nearby(ctx context.Context, lat, lng float64, query string) ([]Place, error) {
	err := ctx.Err()
	if err != nil {
		return nil, fmt.Errorf("places: mock nearby: %w", err)
	}
	out := make([]Place, 0, len(mockSpots))
	for _, s := range mockSpots {
		spotLat, spotLng := offset(lat, lng, float64(s.meters), s.bearing)
		p := Place{
			ID:             s.id,
			Name:           s.name,
			DistanceMeters: s.meters,
			OpenNow:        s.open && !c.closed,
			PriceLevel:     s.price,
			FastFood:       s.fastFood,
			Lat:            spotLat,
			Lng:            spotLng,
			Address:        s.address,
			// A copy, so a caller that edits its answer cannot change the next one.
			Hours: slices.Clone(s.hours),
		}
		// The street stands at about 7 pm, inside every open spot's last opening of the day.
		if p.OpenNow && len(s.hours) > 0 {
			p.ClosesAt = clock(s.hours[len(s.hours)-1].Close)
		}
		out = append(out, p)
	}
	return out, nil
}

// mockSpot is one of the mock's spots before it is set around a point.
type mockSpot struct {
	id, name, address string
	meters            int     // from the point Nearby is asked about
	bearing           float64 // degrees clockwise from north
	open              bool
	price             int
	fastFood          bool
	hours             []Opening
}

// mockSpots is the street at about 7 pm. Early Bird Burgers has closed for the day, so the
// Nearest fries chip has to skip past it to Night Owl Fries, open until 2 am. Night Owl Fries and
// the taco truck are the two cheap spots, and the first three by distance are the only ones within
// 1,200 m. The list is out of distance order, and Green Bowl ties Main Street Pizza against
// spot-id order, so sorting stays the server's job. The names are made up: no real chain.
var mockSpots = []mockSpot{
	{
		id: "mock-03", name: "Night Owl Fries", address: "77 Maple Dr",
		meters: 1200, bearing: 120, open: true, price: 1, fastFood: true,
		hours: []Opening{{Open: at(10, 30), Close: at(26, 0)}},
	},
	{
		id: "mock-07", name: "Green Bowl", address: "33 Cedar Ln",
		meters: 2400, bearing: 260, open: true, price: 2,
		hours: []Opening{{Open: at(10, 0), Close: at(21, 0)}},
	},
	{
		id: "mock-06", name: "Main Street Pizza", address: "560 Main St",
		meters: 2400, bearing: 80, open: true, price: 2,
		hours: []Opening{{Open: at(11, 0), Close: at(24, 0)}},
	},
	{
		id: "mock-01", name: "Early Bird Burgers", address: "18 Elm St",
		meters: 300, bearing: 45, open: false, price: 0, fastFood: true,
		hours: []Opening{{Open: at(6, 0), Close: at(14, 0)}},
	},
	{
		id: "mock-10", name: "Ember Steakhouse", address: "8 Market Sq",
		meters: 5000, bearing: 230, open: true, price: 4,
		hours: []Opening{{Open: at(17, 0), Close: at(22, 0)}},
	},
	{
		id: "mock-02", name: "Campus Corner Deli", address: "402 Oak Ave",
		meters: 650, bearing: 200, open: true, price: 2,
		hours: []Opening{{Open: at(8, 0), Close: at(21, 0)}},
	},
	{
		id: "mock-05", name: "Drumstick Drive-Thru", address: "1290 Pine St",
		meters: 1900, bearing: 160, open: true, price: 2, fastFood: true,
		hours: []Opening{{Open: at(10, 0), Close: at(23, 0)}},
	},
	{
		id: "mock-08", name: "Lucky Noodle House", address: "905 Birch Rd",
		meters: 3200, bearing: 20, open: true, price: 2,
		hours: []Opening{{Open: at(11, 0), Close: at(14, 30)}, {Open: at(17, 0), Close: at(22, 0)}},
	},
	// A truck: no fixed address and no posted hours.
	{
		id: "mock-04", name: "Sunny Side Taco Truck",
		meters: 1500, bearing: 300, open: true, price: 1,
	},
	{
		id: "mock-09", name: "Slow Smoke BBQ", address: "2104 River Rd",
		meters: 4100, bearing: 340, open: true, price: 3,
		hours: []Opening{{Open: at(11, 0), Close: at(21, 30)}},
	},
}

// at is a time of day for the mock's hours. Past midnight keeps counting: at(26, 0) is 2 am.
func at(hour, minute int) time.Duration {
	return time.Duration(hour)*time.Hour + time.Duration(minute)*time.Minute
}

// clock writes a time of day as 24-hour "15:04", folding a time past midnight back into the day,
// so 26h is "02:00".
func clock(d time.Duration) string {
	d %= 24 * time.Hour
	return fmt.Sprintf("%02d:%02d", d/time.Hour, d%time.Hour/time.Minute)
}

// earthRadius is the Earth's mean radius in meters. Over a few kilometers a sphere is as good as
// the real shape.
const earthRadius = 6_371_000.0

// offset is the point meters away from lat, lng along bearing (degrees clockwise from north), on
// a great circle. The longitude is wrapped into -180 to 180, so a spot past the date line stays on
// the map.
func offset(lat, lng, meters, bearing float64) (float64, float64) {
	const rad = math.Pi / 180
	lat1, lng1, brg := lat*rad, lng*rad, bearing*rad
	arc := meters / earthRadius
	lat2 := math.Asin(math.Sin(lat1)*math.Cos(arc) + math.Cos(lat1)*math.Sin(arc)*math.Cos(brg))
	lng2 := lng1 + math.Atan2(math.Sin(brg)*math.Sin(arc)*math.Cos(lat1),
		math.Cos(arc)-math.Sin(lat1)*math.Sin(lat2))
	return lat2 / rad, math.Mod(lng2/rad+540, 360) - 180
}
