// Author: Caden Lund
package fries

import (
	"cmp"
	"encoding/base64"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/bengutow1/csc4330_project5/packages/go/places"
)

// What the server works out about a spot, as the contract's comments define it.
const (
	// walkableMeters is how near counts as walkable: about fifteen minutes on foot.
	walkableMeters = 1200
	// lateNightFrom is the time of day after which a spot counts as open late.
	lateNightFrom = 23 * time.Hour
	// cheapPrice is the price level the Cheap chip keeps: the provider's lowest.
	cheapPrice = 1
)

// spot is a place as this service lists it: the provider's answer and what the server works out.
type spot struct {
	places.Place
	lateNight bool
	walkable  bool
}

// toSpot works out what the provider leaves to the server. Without a point there is nothing to
// measure from, so no spot has a distance and none is walkable.
func toSpot(p places.Place, hasPoint bool) spot {
	if !hasPoint {
		p.DistanceMeters = 0
	}
	return spot{
		Place:     p,
		lateNight: lateNight(p.Hours),
		walkable:  hasPoint && p.DistanceMeters <= walkableMeters,
	}
}

// lateNight reports whether a spot's hours run past 23:00, a 2 am close being 26h. Closing at
// 23:00 is not open late, and a spot with no hours is not known to be.
func lateNight(hours []places.Opening) bool {
	return slices.ContainsFunc(hours, func(o places.Opening) bool { return o.Close > lateNightFrom })
}

// filters are the chips the player turned on. A spot is listed only if it passes every one.
type filters struct {
	openNow, lateNight, cheap, fastFood, walkable bool
}

// keep reports whether s passes every filter that is on.
func (f filters) keep(s spot) bool {
	return (!f.openNow || s.OpenNow) &&
		(!f.lateNight || s.lateNight) &&
		(!f.cheap || s.PriceLevel == cheapPrice) &&
		(!f.fastFood || s.FastFood) &&
		(!f.walkable || s.walkable)
}

// key is a spot's place in the list: closest first, then by spot id, so a tie always breaks the
// same way. A cursor carries the key of the last spot a page showed.
type key struct {
	meters int
	spotID string
}

func keyOf(s spot) key {
	return key{meters: s.DistanceMeters, spotID: s.ID}
}

func (k key) compare(o key) int {
	return cmp.Or(cmp.Compare(k.meters, o.meters), strings.Compare(k.spotID, o.spotID))
}

// page is one page of spots and what rides along with it.
type page struct {
	spots []spot
	// next is the key to page on from; nil on the last page.
	next *key
	// nearest is the closest open fast-food spot whatever the filters; nil when there is none or
	// when the request sent no point to be near.
	nearest *spot
}

// pageOf turns what the provider found into the page the player asked for: the spots every
// filter keeps, closest first, after the cursor's key, at most limit of them.
func pageOf(found []places.Place, in listSpotsInput) page {
	all := make([]spot, 0, len(found))
	for _, p := range found {
		all = append(all, toSpot(p, in.at != nil))
	}
	slices.SortFunc(all, func(a, b spot) int { return keyOf(a).compare(keyOf(b)) })

	var out page
	if in.at != nil {
		i := slices.IndexFunc(all, func(s spot) bool { return s.OpenNow && s.FastFood })
		if i >= 0 {
			out.nearest = &all[i]
		}
	}
	for _, s := range all {
		if !in.filters.keep(s) || (in.after != nil && keyOf(s).compare(*in.after) <= 0) {
			continue
		}
		// One spot past the page means there is more: the cursor resumes after the page's last.
		if len(out.spots) == in.limit {
			last := keyOf(out.spots[len(out.spots)-1])
			out.next = &last
			break
		}
		out.spots = append(out.spots, s)
	}
	return out
}

// encodeCursor writes a key as the next_cursor the client sends back. It is opaque to the client.
func encodeCursor(k key) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.Itoa(k.meters) + ":" + k.spotID))
}

// decodeCursor reads a cursor back into its key. Anything encodeCursor could not have written is
// refused, rather than read as "start over", which would loop an infinite scroll.
func decodeCursor(cursor string) (key, bool) {
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return key{}, false
	}
	meters, spotID, found := strings.Cut(string(raw), ":")
	if !found || spotID == "" || !utf8.ValidString(spotID) || utf8.RuneCountInString(spotID) > maxSpotID {
		return key{}, false
	}
	m, err := strconv.Atoi(meters)
	if err != nil || m < 0 || strconv.Itoa(m) != meters {
		return key{}, false
	}
	return key{meters: m, spotID: spotID}, true
}
