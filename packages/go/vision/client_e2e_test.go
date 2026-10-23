// Author: Ben Gutowski

//go:build e2e

package vision_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bengutow1/csc4330_project5/packages/go/vision"
)

// TestE2EReadsARealFridge spends one real call per bundled photo, at the harshest roast level, and
// prints the verdict for a person to read. cheap_joke.jpg is a fridge that invites a joke about
// whoever owns it (only energy drinks and takeout, diet labels everywhere): its roast must stay on
// the fridge. Run it with `moon run go-vision:test-e2e` and VISION_API_KEY set; never in CI.
func TestE2EReadsARealFridge(t *testing.T) {
	t.Parallel()
	key := os.Getenv("VISION_API_KEY")
	if key == "" {
		t.Fatal("VISION_API_KEY is not set; the live check needs a real key")
	}
	client := vision.NewClient(vision.Config{APIKey: key, Model: os.Getenv("VISION_MODEL")})

	for _, name := range []string{"fridge.jpg", "cheap_joke.jpg"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			photo, err := os.ReadFile(filepath.Join("testdata", "e2e", name))
			if err != nil {
				t.Fatalf("read sample photo (add a JPEG at testdata/e2e/%s): %v", name, err)
			}
			fridge, err := client.ReadFridge(t.Context(), [][]byte{photo}, vision.RoastExtraCrispy)
			if err != nil {
				t.Fatalf("read fridge: %v", err)
			}
			if fridge.Roast == "" {
				t.Errorf("roast is empty, want a line about the fridge")
			}
			t.Logf("%d items:", len(fridge.Items))
			for _, it := range fridge.Items {
				t.Logf("  %s (%s) %s", it.Name, it.Category, it.Quantity)
			}
			t.Logf("roast: %s", fridge.Roast)
		})
	}
}
