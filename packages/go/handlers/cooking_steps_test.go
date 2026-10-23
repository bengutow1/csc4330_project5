// Author: LAPTOP-BPMQJP10\andyk
package handlers_test

import (
	"testing"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"

	cookv1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/cook/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/vision"
)

// TestCookingStepsReachTheRecipeOverTheWire exercises validation, storage and both RPCs.
func TestCookingStepsReachTheRecipeOverTheWire(t *testing.T) {
	t.Parallel()
	f, _ := kitchenFixture()
	f.chef = &fakeChef{recipes: []vision.Recipe{{Title: "Eggs", Minutes: 3, Difficulty: "easy", Uses: []string{"Eggs"}, Allergens: []string{"eggs"}, Steps: []string{"Whisk.", "Cook."}, StepTimers: []int{0, 120}}}}
	srv := cookStack(t, f)
	player := cookPlayerFor(t, srv, "test_timers")
	scan := scanFridge(t, player)
	shelves := listRecipes(t, player, &cookv1.ListRecipesRequest{ScanId: scan.GetScanId()})
	card := firstOn(t, shelves.GetMakeItNow())
	if len(card.GetCookingSteps()) != 2 {
		t.Fatalf("steps = %v, want two", card.GetCookingSteps())
	}
	if card.CookingSteps[0].TimerSeconds != nil || card.CookingSteps[1].GetTimerSeconds() != 120 {
		t.Errorf("timers = %v, want untimed then 120 seconds", card.GetCookingSteps())
	}
	got, err := player.cook.GetRecipe(t.Context(), connect.NewRequest(&cookv1.GetRecipeRequest{RecipeId: card.GetRecipeId()}))
	if err != nil {
		t.Fatalf("get recipe: %v", err)
	}
	if !proto.Equal(card, got.Msg.GetRecipe()) {
		t.Errorf("detail = %v, want card %v", got.Msg.GetRecipe(), card)
	}
}
