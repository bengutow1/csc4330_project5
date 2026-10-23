// Author: LAPTOP-BPMQJP10\andyk
package repository_test

import (
	"slices"
	"testing"

	"github.com/bengutow1/csc4330_project5/packages/go/repository"
)

// TestStepTimersSurviveBothRecipeReads prevents list and detail timer associations drifting.
func TestStepTimersSurviveBothRecipeReads(t *testing.T) {
	t.Parallel()
	repo := repository.New(newPool(t))
	player := createAccount(t, repo, "test_timers")
	scan := createScan(t, repo, player.AccountID, fridge)
	recipe := omelette
	recipe.StepTimers = []int{0, 120, 0}
	createRecipes(t, repo, player.AccountID, scan.ScanID, 1, []repository.NewRecipe{recipe})
	listed := listRecipes(t, repo, player.AccountID, scan.ScanID, 1, repository.RecipeFilter{})
	if len(listed) != 1 {
		t.Fatalf("listed %d, want 1", len(listed))
	}
	got, err := repo.RecipeByID(t.Context(), player.AccountID, listed[0].RecipeID)
	if err != nil {
		t.Fatalf("read recipe: %v", err)
	}
	if !slices.Equal(listed[0].StepTimers, []int{0, 120, 0}) || !slices.Equal(got.StepTimers, []int{0, 120, 0}) {
		t.Errorf("list timers %v, detail timers %v; want [0 120 0]", listed[0].StepTimers, got.StepTimers)
	}
}

// TestTheDatabaseRefusesInvalidStepTimers protects writes that bypass vendor validation.
func TestTheDatabaseRefusesInvalidStepTimers(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		timers []int
	}{
		{"negative", []int{0, -1, 0}}, {"over four hours", []int{0, 14401, 0}}, {"wrong length", []int{120}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			repo := repository.New(newPool(t))
			player := createAccount(t, repo, "test_timers")
			scan := createScan(t, repo, player.AccountID, fridge)
			recipe := omelette
			recipe.StepTimers = c.timers
			_, err := repo.CreateRecipes(t.Context(), player.AccountID, scan.ScanID, 1, []repository.NewRecipe{recipe})
			if err == nil {
				t.Fatal("invalid timers stored, want constraint refusal")
			}
			has, err := repo.HasRecipes(t.Context(), player.AccountID, scan.ScanID, 1)
			if err != nil {
				t.Fatalf("read set: %v", err)
			}
			if has {
				t.Error("a refused recipe left its set behind")
			}
		})
	}
}
