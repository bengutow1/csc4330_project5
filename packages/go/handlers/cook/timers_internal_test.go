// Author: LAPTOP-BPMQJP10\andyk
package cook

import (
	"slices"
	"testing"

	"github.com/bengutow1/csc4330_project5/packages/go/repository"
	"github.com/bengutow1/csc4330_project5/packages/go/vision"
)

// TestStepTimersKeepTheirInstruction tests the stored and wire timer association.
func TestStepTimersKeepTheirInstruction(t *testing.T) {
	t.Parallel()
	input := vision.Recipe{Title: "Rice", Minutes: 4, Difficulty: "easy", Uses: []string{"Rice"}, Steps: []string{"Fill the bowl.", "Microwave for four minutes."}, StepTimers: []int{0, 240}}
	rows, dropped := keepModelRecipes([]vision.Recipe{input})
	if len(rows) != 1 || dropped != 0 {
		t.Fatalf("kept %d, dropped %d; want 1, 0", len(rows), dropped)
	}
	if !slices.Equal(rows[0].StepTimers, []int{0, 240}) {
		t.Errorf("timers = %v, want [0 240]", rows[0].StepTimers)
	}
	got, err := toProtoRecipe(repository.Recipe{Difficulty: "easy", Steps: rows[0].Steps, StepTimers: rows[0].StepTimers})
	if err != nil {
		t.Fatalf("map recipe: %v", err)
	}
	if len(got.GetCookingSteps()) != 2 {
		t.Fatalf("cooking steps = %v, want two", got.GetCookingSteps())
	}
	if got.CookingSteps[0].TimerSeconds != nil {
		t.Error("first step has a timer, want unset")
	}
	if got.CookingSteps[1].GetTimerSeconds() != 240 || got.CookingSteps[1].GetInstruction() != "Microwave for four minutes." {
		t.Errorf("second step = %v, want instruction and 240 seconds", got.CookingSteps[1])
	}
	if !slices.Equal(got.GetSteps(), input.Steps) {
		t.Errorf("legacy steps = %v, want %v", got.GetSteps(), input.Steps)
	}
}

// TestMalformedStepTimersAreDropped prevents a timer applying to the wrong instruction.
func TestMalformedStepTimersAreDropped(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		timers []int
	}{
		{"negative", []int{0, -1}}, {"too long", []int{0, 14401}}, {"misaligned", []int{5}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			input := vision.Recipe{Title: "Rice", Minutes: 4, Difficulty: "easy", Uses: []string{"Rice"}, Steps: []string{"Fill.", "Cook."}, StepTimers: c.timers}
			kept, dropped := keepModelRecipes([]vision.Recipe{input})
			if len(kept) != 0 || dropped != 1 {
				t.Errorf("kept %d, dropped %d; want 0, 1", len(kept), dropped)
			}
		})
	}
}

// TestRecipesWithoutTimersStillHaveCookingSteps keeps saved recipes usable.
func TestRecipesWithoutTimersStillHaveCookingSteps(t *testing.T) {
	t.Parallel()
	got, err := toProtoRecipe(repository.Recipe{Difficulty: "easy", Steps: []string{"Stir."}})
	if err != nil {
		t.Fatalf("map recipe: %v", err)
	}
	if len(got.GetCookingSteps()) != 1 || got.CookingSteps[0].TimerSeconds != nil {
		t.Errorf("steps = %v, want one untimed step", got.GetCookingSteps())
	}
}
