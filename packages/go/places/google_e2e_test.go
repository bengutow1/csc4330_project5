// Author: Arthur

//go:build e2e

package places_test

import (
	"os"
	"testing"

	"github.com/bengutow1/csc4330_project5/packages/go/places"
)

// TestTheLiveAPIListsSpotsNearCampus spends one real call, so a person can see the client read
// Google's actual answer. It runs only through `moon run go-places:e2e`, with PLACES_API_KEY
// exported in the shell, never written to a file. The log prints the spots and never the key.
func TestTheLiveAPIListsSpotsNearCampus(t *testing.T) {
	key := os.Getenv("PLACES_API_KEY")
	if key == "" {
		t.Fatal("PLACES_API_KEY is not set: export it in this shell, never in a file")
	}

	got, err := places.New(key, "").Nearby(t.Context(), campusLat, campusLng, "")
	if err != nil {
		t.Fatalf("nearby: %v", err)
	}

	if len(got) == 0 {
		t.Fatal("no spots near campus, want some")
	}
	for _, p := range got {
		t.Logf("%s | %d m | open=%v closes=%q | price %d | fast food %v | %v",
			p.Name, p.DistanceMeters, p.OpenNow, p.ClosesAt, p.PriceLevel, p.FastFood, p.Hours)
		if p.ID == "" || p.Name == "" {
			t.Errorf("a spot without its id or name: %+v", p)
		}
		if p.PriceLevel < 0 || p.PriceLevel > 4 {
			t.Errorf("%s has price %d, want 0 to 4", p.Name, p.PriceLevel)
		}
	}
}
