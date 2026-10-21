// Author: Caden Lund
package vision_test

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/bengutow1/csc4330_project5/packages/go/vision"
)

// fridge is one canned fridge as the tests expect it: the VISION_MOCK value that pins it, what it
// lists and its medium roast, the concept's sample line for its tier.
type fridge struct {
	pin    string
	items  []vision.Item
	medium string
}

// wantFridges is every mock fridge written out by hand, in the order the cycle serves them.
func wantFridges() []fridge {
	return []fridge{
		{
			pin: "certified_chef",
			items: []vision.Item{
				{Name: "Chicken", Category: "protein", Quantity: "x2"},
				{Name: "Spinach", Category: "vegetable", Quantity: "1 bag"},
				{Name: "Bell peppers", Category: "vegetable", Quantity: "x3"},
				{Name: "Greek yogurt", Category: "dairy", Quantity: "1 tub"},
				{Name: "Eggs", Category: "protein", Quantity: "x12"},
				{Name: "Cheddar", Category: "dairy", Quantity: "1 block"},
			},
			medium: "Fresh veggies, real protein, labeled leftovers. Your fridge has its life together.",
		},
		{
			pin: "home_cook",
			items: []vision.Item{
				{Name: "Ground beef", Category: "protein", Quantity: "1 lb"},
				{Name: "Tortillas", Category: "grain", Quantity: "x10"},
				{Name: "Onion", Category: "vegetable", Quantity: "x1"},
				{Name: "Shredded cheese", Category: "dairy", Quantity: "1 bag"},
				{Name: "Salsa", Category: "condiment", Quantity: "1 jar"},
				{Name: "Orange juice", Category: "drink", Quantity: "half a carton"},
			},
			medium: "There's a real dinner in here. Go find it.",
		},
		{
			pin: "scrappy",
			items: []vision.Item{
				{Name: "Instant ramen", Category: "grain", Quantity: "x3"},
				{Name: "Frozen peas", Category: "vegetable", Quantity: "half a bag"},
				{Name: "Hot sauce", Category: "condiment", Quantity: ""},
				{Name: "Apple", Category: "fruit", Quantity: "x1"},
				{Name: "Soda", Category: "drink", Quantity: "x2"},
			},
			medium: "It's giving creative. We can work with this.",
		},
		{
			pin: "fries_in_the_bag",
			items: []vision.Item{
				{Name: "Ketchup", Category: "condiment", Quantity: ""},
				{Name: "Egg", Category: "protein", Quantity: "x1"},
				{Name: "Lemon", Category: "fruit", Quantity: "x1"},
				{Name: "Energy drinks", Category: "drink", Quantity: "x4"},
				{Name: "Mystery container", Category: "other", Quantity: ""},
			},
			medium: "Ketchup, one egg and a lemon from last semester. Put the fries in the bag.",
		},
		{
			pin:    "empty",
			items:  []vision.Item{},
			medium: "Either your fridge is empty or the photo is.",
		},
	}
}

// TestTheMockCyclesThroughItsFridgesInOrder: with nothing pinned, each scan gets the next fridge
// and the cycle starts over after the last, so a demo walks every tier in turn.
func TestTheMockCyclesThroughItsFridgesInOrder(t *testing.T) {
	t.Parallel()
	mock := quickMock(t, "")
	want := wantFridges()

	// Two laps, so the wrap back to the first fridge is covered.
	for i := range 2 * len(want) {
		w := want[i%len(want)]

		got := read(t, mock, vision.RoastMedium)

		if got.Roast != w.medium {
			t.Errorf("read %d roast = %q, want %s's %q", i, got.Roast, w.pin, w.medium)
		}
		if !slices.Equal(got.Items, w.items) {
			t.Errorf("read %d items = %v, want %s's %v", i, got.Items, w.pin, w.items)
		}
	}
}

// TestAPinnedMockServesOnlyThatFridge: VISION_MOCK=<name> holds one verdict still, read after read,
// so a screen can be built against it.
func TestAPinnedMockServesOnlyThatFridge(t *testing.T) {
	t.Parallel()
	for _, w := range wantFridges() {
		t.Run(w.pin, func(t *testing.T) {
			t.Parallel()
			mock := quickMock(t, w.pin)

			for i := range 3 {
				got := read(t, mock, vision.RoastMedium)

				if got.Roast != w.medium {
					t.Errorf("read %d roast = %q, want %q", i, got.Roast, w.medium)
				}
				if !slices.Equal(got.Items, w.items) {
					t.Errorf("read %d items = %v, want %v", i, got.Items, w.items)
				}
			}
		})
	}
}

// TestTheErrorPinAnswersUnavailable: VISION_MOCK=error stands in for a model outage, which the
// player sees as "try again", on every read.
func TestTheErrorPinAnswersUnavailable(t *testing.T) {
	t.Parallel()
	mock := quickMock(t, "error")

	for i := range 2 {
		got, err := mock.ReadFridge(t.Context(), aPhoto(), vision.RoastMedium)

		if !errors.Is(err, vision.ErrUnavailable) {
			t.Errorf("read %d err = %v, want ErrUnavailable", i, err)
		}
		if got.Roast != "" || got.Items != nil {
			t.Errorf("read %d fridge = %+v, want none beside the error", i, got)
		}
	}
}

// TestACancelledContextEndsTheWait: a player who leaves the judging screen frees the request at
// once instead of holding it for the rest of the pause.
func TestACancelledContextEndsTheWait(t *testing.T) {
	t.Parallel()
	mock := newMock(t, "") // the real two-second pause
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		_, err := mock.ReadFridge(ctx, aPhoto(), vision.RoastMedium)
		done <- err
	}()

	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("err = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("ReadFridge was still waiting a second after its context was cancelled")
	}
}

// TestEveryReadTakesTwoSeconds: the pause is what lets the judging screen play; a mock that
// answered at once would skip it.
func TestEveryReadTakesTwoSeconds(t *testing.T) {
	t.Parallel()
	mock := newMock(t, "scrappy")
	start := time.Now()

	read(t, mock, vision.RoastMedium)

	elapsed := time.Since(start)
	if elapsed < 2*time.Second {
		t.Errorf("read took %v, want at least 2s", elapsed)
	}
}

// TestEveryFridgeHasItsOwnRoastAtEveryLevel: the roast level setting must change the line. A
// missing roast, or one shared between levels or fridges, would make it do nothing.
func TestEveryFridgeHasItsOwnRoastAtEveryLevel(t *testing.T) {
	t.Parallel()
	levels := []vision.RoastLevel{vision.RoastMild, vision.RoastMedium, vision.RoastExtraCrispy}
	servedAt := map[string]string{} // roast -> the fridge and level that served it

	for _, w := range wantFridges() {
		mock := quickMock(t, w.pin)
		for _, level := range levels {
			got := read(t, mock, level)

			where := w.pin + " at " + string(level)
			if got.Roast == "" {
				t.Errorf("%s has no roast", where)
				continue
			}
			prev, repeated := servedAt[got.Roast]
			if repeated {
				t.Errorf("%s repeats the roast of %s: %q", where, prev, got.Roast)
			}
			servedAt[got.Roast] = where
		}
	}
}

// TestEveryMockIngredientIsInAKnownCategory: the scan handler drops an item whose category is
// outside the closed set, so a typo here would quietly shrink a fridge.
func TestEveryMockIngredientIsInAKnownCategory(t *testing.T) {
	t.Parallel()
	// The closed set in docs/contracts.md.
	known := []string{
		"protein", "vegetable", "fruit", "grain", "dairy",
		"condiment", "drink", "snack", "leftover", "other",
	}

	for _, w := range wantFridges() {
		got := read(t, quickMock(t, w.pin), vision.RoastMedium)

		for _, item := range got.Items {
			if !slices.Contains(known, item.Category) {
				t.Errorf("%s lists %q as %q, want one of %v", w.pin, item.Name, item.Category, known)
			}
		}
	}
}

// TestEditingAReadLeavesTheMockAlone: every scan shares the mock's fridges, so a caller that
// changes the list it got back must not change the next player's.
func TestEditingAReadLeavesTheMockAlone(t *testing.T) {
	t.Parallel()
	mock := quickMock(t, "certified_chef")
	first := read(t, mock, vision.RoastMedium)
	if len(first.Items) == 0 {
		t.Fatal("certified_chef listed nothing to edit")
	}

	first.Items[0].Name = "test_edited"

	second := read(t, mock, vision.RoastMedium)
	if second.Items[0].Name != "Chicken" {
		t.Errorf("first item after an edit = %q, want Chicken", second.Items[0].Name)
	}
}

// TestScansAtOnceStillTakeTurns: players judging at the same moment each get their own turn in
// the cycle, never the same one twice.
func TestScansAtOnceStillTakeTurns(t *testing.T) {
	t.Parallel()
	mock := quickMock(t, "")
	want := wantFridges()
	const laps = 4
	var mu sync.Mutex
	served := map[string]int{} // medium roast -> how many reads got it
	var wg sync.WaitGroup

	for range laps * len(want) {
		wg.Go(func() {
			got, err := mock.ReadFridge(t.Context(), aPhoto(), vision.RoastMedium)
			if err != nil {
				t.Errorf("read: %v", err)
				return
			}
			mu.Lock()
			served[got.Roast]++
			mu.Unlock()
		})
	}
	wg.Wait()

	for _, w := range want {
		if served[w.medium] != laps {
			t.Errorf("%s served %d times, want %d", w.pin, served[w.medium], laps)
		}
	}
}

// TestAMisspelledPinIsRefused: VISION_MOCK=scrapy must stop the boot. Cycling instead would look
// like pinning is broken.
func TestAMisspelledPinIsRefused(t *testing.T) {
	t.Parallel()
	for _, pin := range []string{"scrapy", "Scrappy", " scrappy", "fries", "chef"} {
		mock, err := vision.NewMock(pin, "")
		if err == nil {
			t.Errorf("NewMock(%q, \"\") = %v, want an error", pin, mock)
		}
	}
}

// TestAnUnknownRoastLevelIsRefused: a level outside the three is the caller's bug. Serving medium
// would hide it, and ErrUnavailable would tell the player to retry something that cannot work.
func TestAnUnknownRoastLevelIsRefused(t *testing.T) {
	t.Parallel()
	mock := quickMock(t, "scrappy")

	for _, level := range []vision.RoastLevel{"", "spicy", "MEDIUM"} {
		got, err := mock.ReadFridge(t.Context(), aPhoto(), level)

		if err == nil {
			t.Errorf("level %q served %+v, want an error", level, got)
		}
		if errors.Is(err, vision.ErrUnavailable) {
			t.Errorf("level %q err = %v, want an error that is not ErrUnavailable", level, err)
		}
	}
}

// newMock is the mock as the API builds it with no plate pin, two-second pause included.
func newMock(t *testing.T, pin string) *vision.Mock {
	t.Helper()
	mock, err := vision.NewMock(pin, "")
	if err != nil {
		t.Fatalf("new mock %q: %v", pin, err)
	}
	return mock
}

// quickMock is newMock without the pause.
func quickMock(t *testing.T, pin string) *vision.Mock {
	t.Helper()
	return newMock(t, pin).WithoutWaitForTest()
}

// read is one ReadFridge that must succeed.
func read(t *testing.T, mock *vision.Mock, level vision.RoastLevel) vision.Fridge {
	t.Helper()
	got, err := mock.ReadFridge(t.Context(), aPhoto(), level)
	if err != nil {
		t.Fatalf("read fridge at %q: %v", level, err)
	}
	return got
}

// aPhoto stands in for a scan's photos: a JPEG's opening bytes and nothing more. The mock never
// looks at them.
func aPhoto() [][]byte {
	return [][]byte{{0xFF, 0xD8, 0xFF, 0xE0}}
}
