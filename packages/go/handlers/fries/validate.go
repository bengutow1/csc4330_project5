// Author: Caden Lund
package fries

import (
	"fmt"
	"math"
	"strings"

	friesv1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/fries/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers"
)

// The bounds a request is held to. The tags below restate the numbers, since a tag cannot name a
// const: keep the two in step.
const (
	// defaultLimit and maxLimit size a page of spots: a phone shows about six at once, and a page
	// past fifty is a scroll nobody makes.
	defaultLimit = 20
	maxLimit     = 50
	// maxSpotID is the longest spot id, the bound on app.fry_run.spot_id.
	maxSpotID = 256
)

// filterChoices names every filter a request may send.
const filterChoices = "filter must be open_now, late_night, cheap, fast_food or walkable"

// point is where the player is, in degrees.
type point struct {
	lat, lng float64
}

// listSpotsInput is a ListSpots request once it is known good.
type listSpotsInput struct {
	at      *point // nil when the player's location is off
	query   string
	filters filters
	after   *key // nil on the first page
	limit   int
}

// listSpotsTags holds the fields a validator tag can judge on its own.
type listSpotsTags struct {
	Query string `pfield:"query" validate:"max=60"`
	Limit int32  `pfield:"limit" validate:"gte=0"`
}

// fryRunInput holds a fry run to the checks on app.fry_run (keep the two in step), and to plain
// text on one line, which the table cannot hold a NUL against anyway.
type fryRunInput struct {
	SpotID   string `pfield:"spot_id" validate:"required,max=256,plaintext"`
	SpotName string `pfield:"spot_name" validate:"required,max=120,plaintext"`
}

// buildListSpots normalizes a ListSpots request, then validates all of it, so the client hears
// about every problem at once.
func buildListSpots(msg *friesv1.ListSpotsRequest) (listSpotsInput, handlers.FieldErrors) {
	in := listSpotsInput{query: strings.TrimSpace(msg.Query)}
	invalid := handlers.Validate(listSpotsTags{Query: in.query, Limit: msg.Limit})
	refuse := func(field, message string) {
		invalid = append(invalid, handlers.FieldError{Field: field, Message: message})
	}

	// 1. The point: both coordinates or neither. Without one, the query says where to look.
	switch {
	case msg.Lat != nil && msg.Lng != nil:
		lat, lng := msg.GetLat(), msg.GetLng()
		// NaN compares false with everything, so it is caught by name.
		if math.IsNaN(lat) || lat < -90 || lat > 90 {
			refuse("lat", "lat must be between -90 and 90")
		}
		if math.IsNaN(lng) || lng < -180 || lng > 180 {
			refuse("lng", "lng must be between -180 and 180")
		}
		in.at = &point{lat: lat, lng: lng}
	case msg.Lat != nil:
		refuse("lng", "lng is required with lat")
	case msg.Lng != nil:
		refuse("lat", "lat is required with lng")
	case in.query == "":
		refuse("query", "query is required when lat and lng are not sent")
	}

	// 2. The filters: each at most once, every one known. Walkable measures from the point, so it
	// needs one.
	seen := map[friesv1.SpotFilter]bool{}
	for i, f := range msg.Filters {
		field := fmt.Sprintf("filters[%d]", i)
		if seen[f] {
			refuse(field, "filter is listed twice")
			continue
		}
		seen[f] = true
		switch f {
		case friesv1.SpotFilter_SPOT_FILTER_OPEN_NOW:
			in.filters.openNow = true
		case friesv1.SpotFilter_SPOT_FILTER_LATE_NIGHT:
			in.filters.lateNight = true
		case friesv1.SpotFilter_SPOT_FILTER_CHEAP:
			in.filters.cheap = true
		case friesv1.SpotFilter_SPOT_FILTER_FAST_FOOD:
			in.filters.fastFood = true
		case friesv1.SpotFilter_SPOT_FILTER_WALKABLE:
			if msg.Lat == nil && msg.Lng == nil {
				refuse(field, "walkable needs lat and lng")
			}
			in.filters.walkable = true
		case friesv1.SpotFilter_SPOT_FILTER_UNSPECIFIED:
			refuse(field, filterChoices)
		default:
			// A value newer than this server.
			refuse(field, filterChoices)
		}
	}

	// 3. The page: a cursor only this server could have written, and a size within bounds
	if msg.Cursor != "" {
		after, ok := decodeCursor(msg.Cursor)
		if !ok {
			refuse("cursor", "cursor is not one this server gave out")
		}
		in.after = &after
	}
	in.limit = defaultLimit
	if msg.Limit > 0 {
		in.limit = min(int(msg.Limit), maxLimit)
	}

	if invalid != nil {
		return listSpotsInput{}, invalid
	}
	return in, nil
}
