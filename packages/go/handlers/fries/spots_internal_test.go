// Author: Caden Lund
package fries

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/bengutow1/csc4330_project5/packages/go/places"
)

// TestLateNightMeansOpenPastEleven: the Late night chip and its badge, on every boundary. Closing
// at 23:00 is not open late; a minute later is.
func TestLateNightMeansOpenPastEleven(t *testing.T) {
	t.Parallel()
	open := func(from, to time.Duration) places.Opening { return places.Opening{Open: from, Close: to} }
	cases := []struct {
		name  string
		hours []places.Opening
		want  bool
	}{
		{"no hours known", nil, false},
		{"closes at 22:59", []places.Opening{open(11*time.Hour, 22*time.Hour+59*time.Minute)}, false},
		{"closes at 23:00", []places.Opening{open(11*time.Hour, 23*time.Hour)}, false},
		{"closes at 23:01", []places.Opening{open(11*time.Hour, 23*time.Hour+time.Minute)}, true},
		{"closes at midnight", []places.Opening{open(11*time.Hour, 24*time.Hour)}, true},
		{"closes at 2 am", []places.Opening{open(10*time.Hour+30*time.Minute, 26*time.Hour)}, true},
		{"lunch, then dinner to 22:00", []places.Opening{open(11*time.Hour, 14*time.Hour), open(17*time.Hour, 22*time.Hour)}, false},
		{"lunch, then dinner to 23:30", []places.Opening{open(11*time.Hour, 14*time.Hour), open(17*time.Hour, 23*time.Hour+30*time.Minute)}, true},
		{"opens at 23:30", []places.Opening{open(23*time.Hour+30*time.Minute, 27*time.Hour)}, true},
	}
	for _, c := range cases {
		got := lateNight(c.hours)
		if got != c.want {
			t.Errorf("%s: late night = %v, want %v", c.name, got, c.want)
		}
	}
}

// TestACursorReadsBackAsItsKey: whatever a page ends on, its cursor resumes after exactly that
// spot, ids with colons and accents included.
func TestACursorReadsBackAsItsKey(t *testing.T) {
	t.Parallel()
	for _, want := range []key{
		{meters: 0, spotID: "mock-01"},
		{meters: 2400, spotID: "mock-06"},
		{meters: 2147483647, spotID: "places/ChIJ:with:colons"},
		{meters: 15, spotID: "café-ñ"},
		{meters: 3, spotID: strings.Repeat("i", maxSpotID)},
	} {
		got, ok := decodeCursor(encodeCursor(want))
		if !ok || got != want {
			t.Errorf("%+v came back as %+v (ok %v)", want, got, ok)
		}
	}
}

// TestACursorThisServerNeverWroteIsRefused: a cursor read as "start over" would loop an infinite
// scroll, so anything encodeCursor could not have written is refused.
func TestACursorThisServerNeverWroteIsRefused(t *testing.T) {
	t.Parallel()
	encode := func(raw string) string { return base64.RawURLEncoding.EncodeToString([]byte(raw)) }
	cases := []struct {
		name, cursor string
	}{
		{"not base64", "!!!"},
		{"padded base64", base64.URLEncoding.EncodeToString([]byte("12:mock-01"))},
		{"no colon", encode("1200")},
		{"no distance", encode(":mock-03")},
		{"a negative distance", encode("-5:mock-03")},
		{"a plus sign", encode("+5:mock-03")},
		{"leading zeros", encode("007:mock-03")},
		{"not a number", encode("12x:mock-03")},
		{"no spot id", encode("5:")},
		{"a spot id too long", encode("5:" + strings.Repeat("i", maxSpotID+1))},
		{"a spot id that is not UTF-8", encode("5:\xff\xfe")},
	}
	for _, c := range cases {
		got, ok := decodeCursor(c.cursor)
		if ok {
			t.Errorf("%s: %q read as %+v, want it refused", c.name, c.cursor, got)
		}
	}
}
