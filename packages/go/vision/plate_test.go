// Author: Caden Lund
package vision_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bengutow1/csc4330_project5/packages/go/vision"
)

// TestPlateChecksPassUnlessPinnedToReject: plating has to work against the mock, and
// VISION_MOCK_PLATE=reject gives the rejected-plate screen something to show.
func TestPlateChecksPassUnlessPinnedToReject(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		pin  string
		pass bool
	}{
		{name: "unpinned", pin: "", pass: true},
		{name: "reject", pin: "reject", pass: false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			mock := quickMock(t, "")
			err := mock.PinPlates(c.pin)
			if err != nil {
				t.Fatalf("pin plates %q: %v", c.pin, err)
			}

			for _, title := range []string{"Chicken Fajita Bowl", "Lemon Water (Fancy)"} {
				got := checkPlate(t, mock, title)

				if got.Passed != c.pass {
					t.Errorf("%s passed = %v, want %v", title, got.Passed, c.pass)
				}
				// A pass needs no reason; a failure needs one for the server log.
				if (got.Reason == "") != c.pass {
					t.Errorf("%s reason = %q, want one only on a failure", title, got.Reason)
				}
			}
		})
	}
}

// TestUnpinningPlatesPassesThemAgain: "" is the default, so pinning it after reject must undo
// the reject rather than leave plates failing.
func TestUnpinningPlatesPassesThemAgain(t *testing.T) {
	t.Parallel()
	mock := quickMock(t, "")
	pinPlates(t, mock, "reject")
	pinPlates(t, mock, "")

	got := checkPlate(t, mock, "Beef Tacos")

	if !got.Passed {
		t.Errorf("passed = false (%q), want true once unpinned", got.Reason)
	}
}

// TestAMisspelledPlatePinIsRefused: VISION_MOCK_PLATE=rejct must stop the boot. Passing every
// plate instead would look like the pin is broken.
func TestAMisspelledPlatePinIsRefused(t *testing.T) {
	t.Parallel()
	for _, pin := range []string{"rejct", "Reject", " reject", "fail", "error"} {
		err := quickMock(t, "").PinPlates(pin)
		if err == nil {
			t.Errorf("PinPlates(%q) = nil, want an error", pin)
		}
	}
}

// TestTheErrorPinFailsPlateChecks: VISION_MOCK=error is a model outage for plates too, which
// the player sees as "try again" and never as a rejected plate.
func TestTheErrorPinFailsPlateChecks(t *testing.T) {
	t.Parallel()
	mock := quickMock(t, "error")

	got, err := mock.CheckPlate(t.Context(), aPlate(), "Beef Tacos")

	if !errors.Is(err, vision.ErrUnavailable) {
		t.Errorf("err = %v, want ErrUnavailable", err)
	}
	if got != (vision.PlateCheck{}) {
		t.Errorf("check = %+v, want none beside the error", got)
	}
}

// TestPlateChecksTakeTwoSecondsAndStopWhenCancelled: the pause makes the check real on screen,
// and a player who leaves frees the request at once.
func TestPlateChecksTakeTwoSecondsAndStopWhenCancelled(t *testing.T) {
	t.Parallel()
	t.Run("the pause", func(t *testing.T) {
		t.Parallel()
		start := time.Now()

		checkPlate(t, newMock(t, ""), "Beef Tacos")

		elapsed := time.Since(start)
		if elapsed < 2*time.Second {
			t.Errorf("check took %v, want at least 2s", elapsed)
		}
	})
	t.Run("a cancelled context", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(t.Context())
		done := make(chan error, 1)
		go func() {
			_, err := newMock(t, "").CheckPlate(ctx, aPlate(), "Beef Tacos")
			done <- err
		}()

		cancel()

		select {
		case err := <-done:
			if !errors.Is(err, context.Canceled) {
				t.Errorf("err = %v, want context.Canceled", err)
			}
		case <-time.After(time.Second):
			t.Fatal("CheckPlate was still waiting a second after its context was cancelled")
		}
	})
}

// checkPlate is one CheckPlate call that must succeed.
func checkPlate(t *testing.T, mock *vision.Mock, title string) vision.PlateCheck {
	t.Helper()
	got, err := mock.CheckPlate(t.Context(), aPlate(), title)
	if err != nil {
		t.Fatalf("check plate %q: %v", title, err)
	}
	return got
}

// pinPlates is one PinPlates that must succeed.
func pinPlates(t *testing.T, mock *vision.Mock, pin string) {
	t.Helper()
	err := mock.PinPlates(pin)
	if err != nil {
		t.Fatalf("pin plates %q: %v", pin, err)
	}
}

// aPlate stands in for a plate photo: a JPEG's opening bytes. The mock never looks at it.
func aPlate() []byte {
	return []byte{0xFF, 0xD8, 0xFF, 0xE0}
}
