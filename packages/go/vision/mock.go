// Author: Caden Lund
package vision

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync/atomic"
	"time"
)

// mockWait is how long every mock call takes, about what the real model takes, so the judging
// screen plays as it will against the real thing.
const mockWait = 2 * time.Second

// mockFailPin pins the mock to fail every call, so the screen for a model outage can be built
// without one.
const mockFailPin = "error"

// mockRejectPin pins the mock to fail every plate check, so the retry screen can be built
// without a bad photo.
const mockRejectPin = "reject"

// Mock stands in for the vision model when the API has no key. It serves the canned fridges in
// fridges.go, one per read: each in turn, or always the one it is pinned to. Each fridge's
// recipes are in recipes.go, and plate checks pass unless pinned otherwise.
type Mock struct {
	fridges      []mockFridge // the cycle; a pinned mock's holds its one fridge
	fail         bool         // pinned to "error": every call answers ErrUnavailable
	rejectPlates bool         // pinned to "reject": every plate check fails
	wait         time.Duration
	reads        atomic.Uint64 // verdicts served so far, which picks the next fridge
}

// NewMock builds the mock. pin is "" to serve every fridge in turn, a fridge's name
// (certified_chef, home_cook, scrappy, fries_in_the_bag or empty) to serve only that one, or
// "error" to fail every call. Any other pin is refused.
func NewMock(pin string) (*Mock, error) {
	m := &Mock{fridges: mockFridges, wait: mockWait}
	switch pin {
	case "":
		// Every fridge, in turn.
	case mockFailPin:
		m.fail = true
	default:
		i := slices.IndexFunc(mockFridges, func(f mockFridge) bool { return f.name == pin })
		if i < 0 {
			names := make([]string, 0, len(mockFridges)+1)
			for _, f := range mockFridges {
				names = append(names, f.name)
			}
			names = append(names, mockFailPin)
			return nil, fmt.Errorf("vision: no mock fridge is named %q, want one of %s", pin, strings.Join(names, ", "))
		}
		m.fridges = mockFridges[i : i+1]
	}
	return m, nil
}

// ReadFridge answers as the model would, after the pause the model takes. It never looks at the
// photos: the fridge is the next one in turn, or the pinned one.
func (m *Mock) ReadFridge(ctx context.Context, photos [][]byte, level RoastLevel) (Fridge, error) {
	// 1. The pause, cut short when the caller gives up
	err := m.pause(ctx, "read fridge")
	if err != nil {
		return Fridge{}, err
	}

	// 2. Pinned to fail, like a model that is down
	if m.fail {
		return Fridge{}, fmt.Errorf("vision: mock pinned to %s: %w", mockFailPin, ErrUnavailable)
	}

	// 3. The fridge whose turn it is. Each read claims its own turn, so scans at the same moment
	// still get consecutive fridges.
	turn := m.reads.Add(1) - 1
	f := m.fridges[turn%uint64(len(m.fridges))]
	roast, err := f.roasts.at(level)
	if err != nil {
		return Fridge{}, err
	}
	// A copy, so a caller that edits its list cannot edit what every later read serves.
	return Fridge{Items: slices.Clone(f.items), Roast: roast}, nil
}

// PinPlates sets how the mock judges plates: "" passes every plate, and "reject" fails every one.
// Any other pin is refused. Call it before the first check.
func (m *Mock) PinPlates(pin string) error {
	switch pin {
	case "":
		m.rejectPlates = false
	case mockRejectPin:
		m.rejectPlates = true
	default:
		return fmt.Errorf("vision: no plate pin is named %q, want %q or none", pin, mockRejectPin)
	}
	return nil
}

// Recipes answers as the model would, after the pause the model takes: the canned recipes of the
// mock fridge whose list shares the most names with items. It ignores the kitchen, so every
// recipe reaches the server's checks and filters, which are what a player's settings narrow.
func (m *Mock) Recipes(ctx context.Context, items []Item, kitchen Kitchen) ([]Recipe, error) {
	err := m.pause(ctx, "recipes")
	if err != nil {
		return nil, err
	}
	if m.fail {
		return nil, fmt.Errorf("vision: mock pinned to %s: %w", mockFailPin, ErrUnavailable)
	}
	return recipesFor(items), nil
}

// CheckPlate answers as the model would, after the pause: the plate passes, unless PinPlates
// pinned the mock to reject every one.
func (m *Mock) CheckPlate(ctx context.Context, photo []byte, recipeTitle string) (PlateCheck, error) {
	err := m.pause(ctx, "check plate")
	if err != nil {
		return PlateCheck{}, err
	}
	if m.fail {
		return PlateCheck{}, fmt.Errorf("vision: mock pinned to %s: %w", mockFailPin, ErrUnavailable)
	}
	if m.rejectPlates {
		return PlateCheck{Passed: false, Reason: "pinned to reject every plate"}, nil
	}
	return PlateCheck{Passed: true}, nil
}

// pause is the wait every call makes, about what the model takes, cut short when the caller
// gives up.
func (m *Mock) pause(ctx context.Context, op string) error {
	timer := time.NewTimer(m.wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return fmt.Errorf("vision: mock %s: %w", op, ctx.Err())
	case <-timer.C:
		return nil
	}
}
