// Author: Caden Lund
package cook

import (
	"fmt"

	cookv1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/cook/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers"
	"github.com/bengutow1/csc4330_project5/packages/go/points"
	"github.com/bengutow1/csc4330_project5/packages/go/repository"
	"github.com/bengutow1/csc4330_project5/packages/go/vision"
)

// difficulties is the closed set in one table: the word app.recipe stores and the wire's enum. A
// word missing here fails the rpc with a logged internal error rather than reaching the app as
// unspecified.
var difficulties = map[string]cookv1.Difficulty{
	repository.DifficultyEasy:   cookv1.Difficulty_DIFFICULTY_EASY,
	repository.DifficultyMedium: cookv1.Difficulty_DIFFICULTY_MEDIUM,
	repository.DifficultyHard:   cookv1.Difficulty_DIFFICULTY_HARD,
}

// toProtoShelves splits a set into its two shelves, keeping its order on each: what the player
// can make now, and what lacks one thing.
func toProtoShelves(set []repository.Recipe) (*cookv1.ListRecipesResponse, error) {
	resp := &cookv1.ListRecipesResponse{}
	for _, r := range set {
		recipe, err := toProtoRecipe(r)
		if err != nil {
			return nil, err
		}
		if len(r.Missing) == 0 {
			resp.MakeItNow = append(resp.MakeItNow, recipe)
		} else {
			resp.OneItemAway = append(resp.OneItemAway, recipe)
		}
	}
	return resp, nil
}

// toProtoRecipe maps a stored recipe to the wire, with what plating it would pay.
func toProtoRecipe(r repository.Recipe) (*cookv1.Recipe, error) {
	difficulty, ok := difficulties[r.Difficulty]
	if !ok {
		return nil, fmt.Errorf("cook: no wire difficulty for %q", r.Difficulty)
	}
	equipment, err := handlers.WireEquipment(r.Equipment)
	if err != nil {
		return nil, err
	}
	return &cookv1.Recipe{
		RecipeId:      r.RecipeID,
		Title:         r.Title,
		Minutes:       int32(r.Minutes), // 1 to 240, held there by app.recipe's check
		Difficulty:    difficulty,
		Equipment:     equipment,
		Uses:          r.Uses,
		Missing:       r.Missing,
		Steps:         r.Steps,
		PointsPreview: pointsPreview(r.ScanScore),
	}, nil
}

// pointsPreview is what plating a recipe from a scan with this score pays before any streak: the
// cook, and the bonus a bad fridge earns. The points rules decide it, so the card and the receipt
// cannot disagree.
func pointsPreview(score int) int32 {
	total := 0
	for _, line := range points.PlateAward(points.Plate{Score: score}) {
		total += line.Points
	}
	return int32(total) // at most 200 before a streak or a combo
}

// visionItems is a stored list as the model reads it.
func visionItems(items []repository.ScanItem) []vision.Item {
	out := make([]vision.Item, 0, len(items))
	for _, item := range items {
		out = append(out, vision.Item{Name: item.Name, Category: item.Category, Quantity: item.Quantity})
	}
	return out
}
