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
		name   string
		fridge string
		plate  string
		pass   bool
	}{
		{name: "unpinned", fridge: "", plate: "", pass: true},
		{name: "reject", fridge: "", plate: "reject", pass: false},
		// The two pins are separate settings, so pinning a fridge keeps the plate pin.
		{name: "reject beside a pinned fridge", fridge: "home_cook", plate: "reject", pass: false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			mock := quickPlateMock(t, c.fridge, c.plate)

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

// TestAMisspelledPlatePinIsRefused: VISION_MOCK_PLATE=rejct must stop the boot, with or without
// a fridge pin. Passing every plate instead would look like the pin is broken.
func TestAMisspelledPlatePinIsRefused(t *testing.T) {
	t.Parallel()
	for _, fridge := range []string{"", "scrappy"} {
		for _, plate := range []string{"rejct", "Reject", " reject", "fail", "error"} {
			mock, err := vision.NewMock(fridge, plate)
			if err == nil {
				t.Errorf("NewMock(%q, %q) = %v, want an error", fridge, plate, mock)
			}
		}
	}
}

// TestTheErrorPinFailsPlateChecks: VISION_MOCK=error is a model outage for plates too, which
// the player sees as "try again" and never as a rejected plate, even beside the reject pin.
func TestTheErrorPinFailsPlateChecks(t *testing.T) {
	t.Parallel()
	for _, plate := range []string{"", "reject"} {
		mock := quickPlateMock(t, "error", plate)

		got, err := mock.CheckPlate(t.Context(), aPlate(), "Beef Tacos")

		if !errors.Is(err, vision.ErrUnavailable) {
			t.Errorf("plate pin %q: err = %v, want ErrUnavailable", plate, err)
		}
		if got != (vision.PlateCheck{}) {
			t.Errorf("plate pin %q: check = %+v, want none beside the error", plate, got)
		}
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

// quickPlateMock is the mock built with both pins, without the pause.
func quickPlateMock(t *testing.T, fridgePin, platePin string) *vision.Mock {
	t.Helper()
	mock, err := vision.NewMock(fridgePin, platePin)
	if err != nil {
		t.Fatalf("new mock %q, %q: %v", fridgePin, platePin, err)
	}
	return mock.WithoutWaitForTest()
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

// aPlate stands in for a plate photo: a JPEG's opening bytes. The mock never looks at it.
func aPlate() []byte {
	return []byte{0xFF, 0xD8, 0xFF, 0xE0}
}
