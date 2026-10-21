// Author: Caden Lund
package cook

import (
	"fmt"
	"strings"

	accountv1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/account/v1"
	cookv1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/cook/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers"
	"github.com/bengutow1/csc4330_project5/packages/go/repository"
	"github.com/bengutow1/csc4330_project5/packages/go/vision"
)

// maxRecipes is the most a set keeps from the model, which is asked for a handful.
const maxRecipes = 12

// recipeInput holds one recipe the model wrote, normalized, to app.recipe's checks (keep the two
// in step), with each name and step bounded too, which the table cannot do for an array.
type recipeInput struct {
	Title      string   `pfield:"title"      validate:"required,max=80,plaintext"`
	Minutes    int      `pfield:"minutes"    validate:"min=1,max=240"`
	Difficulty string   `pfield:"difficulty" validate:"oneof=easy medium hard"`
	Equipment  []string `pfield:"equipment"  validate:"max=4,dive,oneof=microwave stovetop oven air_fryer"`
	Uses       []string `pfield:"uses"       validate:"min=1,max=20,dive,required,max=40,plaintext"`
	// More than one is the recipe check's to refuse, so that the log says why.
	Missing   []string `pfield:"missing"   validate:"max=20,dive,required,max=40,plaintext"`
	Allergens []string `pfield:"allergens" validate:"max=9,dive,oneof=peanuts tree_nuts milk eggs wheat soy fish shellfish sesame"`
	Steps     []string `pfield:"steps"     validate:"min=1,max=20,dive,required,max=300,plaintext"`
}

// listInput is a ListRecipes request, validated: the scan and the filters the SQL applies.
type listInput struct {
	scanID int64
	filter repository.RecipeFilter
}

// The filters' rules, as the chips name them.
const (
	equipmentRule = "equipment must be microwave, stovetop, oven or air fryer"
	dietRule      = "diet must be none, vegetarian, vegan or pescatarian"
)

// buildListRecipes validates a ListRecipes request. Every problem comes back at once, each under
// its field (equipment[1]), so the screen can mark them all.
func buildListRecipes(req *cookv1.ListRecipesRequest) (listInput, handlers.FieldErrors) {
	invalid := handlers.Validate(struct {
		ScanID     int64 `pfield:"scan_id"     validate:"gt=0"`
		MaxMinutes int32 `pfield:"max_minutes" validate:"oneof=0 10 20 45"`
	}{req.GetScanId(), req.GetMaxMinutes()})
	refuse := func(field, msg string) {
		invalid = append(invalid, handlers.FieldError{Field: field, Message: msg})
	}

	equipment := make([]string, 0, len(req.GetEquipment()))
	for i, e := range req.GetEquipment() {
		word, ok := handlers.EquipmentWord(e)
		if !ok {
			refuse(fmt.Sprintf("equipment[%d]", i), equipmentRule)
			continue
		}
		equipment = append(equipment, word)
	}

	// Every dish fits no diet at all, so unspecified and none filter nothing.
	diet := ""
	d := req.GetDiet()
	if d != accountv1.Diet_DIET_UNSPECIFIED && d != accountv1.Diet_DIET_NONE {
		word, ok := handlers.DietWord(d)
		if !ok {
			refuse("diet", dietRule)
		}
		diet = word
	}

	if invalid != nil {
		return listInput{}, invalid
	}
	return listInput{
		scanID: req.GetScanId(),
		filter: repository.RecipeFilter{
			MaxMinutes: int(req.GetMaxMinutes()),
			Equipment:  equipment,
			Diet:       diet,
		},
	}, nil
}

// keepModelRecipes holds what the model wrote to the shape a stored recipe takes. A recipe that
// fails is dropped, as is anything past maxRecipes, and how many went comes back for the log.
func keepModelRecipes(in []vision.Recipe) ([]repository.NewRecipe, int) {
	kept := make([]repository.NewRecipe, 0, min(len(in), maxRecipes))
	for _, r := range in {
		if len(kept) == maxRecipes {
			break
		}
		row := recipeInput{
			Title:      strings.TrimSpace(r.Title),
			Minutes:    r.Minutes,
			Difficulty: word(r.Difficulty),
			Equipment:  eachWord(r.Equipment),
			Uses:       eachTrimmed(r.Uses),
			Missing:    eachTrimmed(r.Missing),
			Allergens:  eachWord(r.Allergens),
			Steps:      eachTrimmed(r.Steps),
		}
		if handlers.Validate(row) != nil {
			continue
		}
		kept = append(kept, repository.NewRecipe{
			Title: row.Title, Minutes: row.Minutes, Difficulty: row.Difficulty,
			Equipment: row.Equipment, Uses: row.Uses, Missing: row.Missing,
			Allergens: row.Allergens, Steps: row.Steps,
		})
	}
	return kept, len(in) - len(kept)
}

// word is a closed-set value as the model may write it ("Easy ") as the database spells it.
func word(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func eachWord(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		out = append(out, word(s))
	}
	return out
}

func eachTrimmed(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		out = append(out, strings.TrimSpace(s))
	}
	return out
}
