// Author: LAPTOP-BPMQJP10\andyk
package vision_test

import (
	"slices"
	"testing"

	"github.com/bengutow1/csc4330_project5/packages/go/vision"
)

// TestTheMockKeepsStepTimersWithTheirInstruction covers the demo's timed and untimed steps.
func TestTheMockKeepsStepTimersWithTheirInstruction(t *testing.T) {
	t.Parallel()
	mock := quickMock(t, "certified_chef")
	items := []vision.Item{{Name: "Chicken"}, {Name: "Bell peppers"}}
	first := recipes(t, mock, items)
	if len(first) == 0 {
		t.Fatal("no recipes")
	}
	if !slices.Equal(first[0].StepTimers, []int{0, 0, 480, 180}) {
		t.Fatalf("timers = %v, want [0 0 480 180]", first[0].StepTimers)
	}
	first[0].StepTimers[2] = 1
	again := recipes(t, mock, items)
	if again[0].StepTimers[2] != 480 {
		t.Errorf("timer after caller edit = %d, want 480", again[0].StepTimers[2])
	}
}
