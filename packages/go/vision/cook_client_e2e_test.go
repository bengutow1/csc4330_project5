// Author: LAPTOP-BPMQJP10\andyk

//go:build e2e

package vision_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bengutow1/csc4330_project5/packages/go/vision"
)

// TestE2ECookRecipesForASampleFridge spends one call, with peanuts excluded, and logs no key.
func TestE2ECookRecipesForASampleFridge(t *testing.T) {
	t.Parallel()
	client := liveCookClient(t)
	items := []vision.Item{{Name: "Eggs", Category: "protein", Quantity: "x6"}, {Name: "Spinach", Category: "vegetable", Quantity: "1 bag"}, {Name: "Tomatoes", Category: "vegetable", Quantity: "x2"}}
	got, err := client.Recipes(t.Context(), items, vision.Kitchen{Staples: []string{"salt", "pepper", "oil", "rice"}, Equipment: []string{"stovetop", "microwave"}, Diet: "vegetarian", Allergies: []string{"peanuts"}})
	if err != nil {
		t.Fatalf("recipes: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("no recipes returned for the sample fridge")
	}
	for _, r := range got {
		t.Logf("recipe=%q uses=%v missing=%v allergens=%v steps=%v timers=%v", r.Title, r.Uses, r.Missing, r.Allergens, r.Steps, r.StepTimers)
	}
}

// TestE2ECookChecksOneRealPlate spends one call using an owner-supplied photo of scrambled eggs.
func TestE2ECookChecksOneRealPlate(t *testing.T) {
	t.Parallel()
	client := liveCookClient(t)
	photo, err := os.ReadFile(filepath.Join("testdata", "e2e", "scrambled_eggs.jpg"))
	if err != nil {
		t.Fatalf("add a scrambled-eggs JPEG at testdata/e2e/scrambled_eggs.jpg: %v", err)
	}
	got, err := client.CheckPlate(t.Context(), photo, "Scrambled eggs")
	if err != nil {
		t.Fatalf("plate check: %v", err)
	}
	t.Logf("passed=%v reason=%s", got.Passed, got.Reason)
}

func liveCookClient(t *testing.T) *vision.Client {
	t.Helper()
	key := os.Getenv("VISION_API_KEY")
	if key == "" {
		t.Fatal("VISION_API_KEY is not set; the owner must run this live check with a real key")
	}
	return vision.NewClient(vision.Config{APIKey: key, Model: os.Getenv("VISION_MODEL")})
}
