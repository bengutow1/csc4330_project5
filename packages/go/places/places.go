// Author: Caden Lund
// Package places finds food near a point. A provider is one small type with a Nearby method that
// answers in Place, which is no provider's shape and none of ours: no proto types, no repository,
// no logger. Mock is the provider the API serves when it has no PLACES_API_KEY.
package places

import "time"

// Place is one place to eat, as any provider describes it. Whether it is walkable or open late is
// the server's call, made from DistanceMeters and Hours.
type Place struct {
	// ID is the provider's own id for the place, the same from one call to the next.
	ID   string
	Name string
	// DistanceMeters is the straight line from the point Nearby was asked about.
	DistanceMeters int
	OpenNow        bool
	// ClosesAt is when an open place closes, as 24-hour "15:04" in its local time. "" when there is
	// no time to show: closed now, open around the clock, or unknown.
	ClosesAt string
	// PriceLevel runs from 1 (cheap) to 4; 0 when unknown.
	PriceLevel int
	FastFood   bool
	Lat, Lng   float64 // degrees
	// Address is one line, as the provider writes it; "" when unknown.
	Address string
	// Category is the kind of food in a word or two ("Mexican", "Pizza"), as the provider names
	// it; "" when unknown.
	Category string
	// Hours are the day's openings, earliest first; none when unknown.
	Hours []Opening
}

// Opening is one stretch of a day when a place is open, measured from its local midnight. A place
// that closes after midnight closes past 24h: open until 2 am is a Close of 26h.
type Opening struct {
	Open, Close time.Duration
}
