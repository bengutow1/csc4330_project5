// Author: Caden Lund
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// recipe.go owns app.recipe: the dishes kept for one version of a scan's list, written once as a
// set and never updated. A recipe has no account of its own, so every query reaches the owner
// through the recipe's scan, and every read also hides a recipe holding one of the owner's
// allergies, which can have changed since the set was made.

// The difficulties, mirroring app.recipe's check.
const (
	DifficultyEasy   = "easy"
	DifficultyMedium = "medium"
	DifficultyHard   = "hard"
)

// NewRecipe is one checked dish to store in a set.
type NewRecipe struct {
	Title      string
	Minutes    int      // the whole time to the plate
	Difficulty string   // a Difficulty const
	Equipment  []string // Equipment consts; empty needs none
	Uses       []string // the food it takes that the player has
	Missing    []string // the food it takes that the player lacks: none, or one
	Allergens  []string // the Allergy consts it contains
	Diets      []string // the Diet consts besides none that it fits
	Steps      []string
}

// Recipe is one stored dish, as the shelves and the recipe page read it.
type Recipe struct {
	RecipeID   int64
	Title      string
	Minutes    int
	Difficulty string // a Difficulty const
	Equipment  []string
	Uses       []string
	Missing    []string // none, or one
	Steps      []string
	ScanScore  int // the score of the recipe's scan, which sets what plating it pays
}

// RecipeFilter narrows a set. Each zero value filters nothing.
type RecipeFilter struct {
	MaxMinutes int      // the longest a recipe may take; 0 is any
	Equipment  []string // keep recipes that need nothing beyond these; empty is any
	Diet       string   // keep recipes that fit this Diet const; "" and none are any
}

// HasRecipes reports whether one of the account's scans already has a set for itemsVersion.
func (r *Repository) HasRecipes(
	ctx context.Context, accountID, scanID int64, itemsVersion int,
) (bool, error) {
	var has bool
	err := r.db.QueryRow(ctx, `
		select exists (
		       select 1
		         from app.recipe r
		         join app.scan s on s.scan_id = r.scan_id
		        where s.account_id = $1
		          and r.scan_id = $2
		          and r.items_version = $3)`, accountID, scanID, itemsVersion).Scan(&has)
	if err != nil {
		return false, known(fmt.Sprintf("has recipes account=%d scan=%d version=%d",
			accountID, scanID, itemsVersion), err)
	}
	return has, nil
}

// recipeRow is one NewRecipe as CreateRecipes hands it to Postgres. The lists differ in length
// from one recipe to the next, which a two-dimensional array cannot hold, so the set travels as
// JSON.
type recipeRow struct {
	Position   int      `json:"position"`
	Title      string   `json:"title"`
	Minutes    int      `json:"minutes"`
	Difficulty string   `json:"difficulty"`
	Equipment  []string `json:"equipment"`
	Uses       []string `json:"uses"`
	Missing    []string `json:"missing"`
	Allergens  []string `json:"allergens"`
	Diets      []string `json:"diets"`
	Steps      []string `json:"steps"`
}

// CreateRecipes stores the set for one version of one of the account's scans, numbered in the
// order given, in one statement. stored is false when there is nothing to store, when the scan is
// not the account's, or when the version already has its set: the unique key on the position
// decides, so of two requests that raced, one stores its whole set and the other stores nothing.
func (r *Repository) CreateRecipes(
	ctx context.Context, accountID, scanID int64, itemsVersion int, set []NewRecipe,
) (stored bool, err error) {
	if len(set) == 0 {
		return false, nil
	}
	op := fmt.Sprintf("create recipes account=%d scan=%d version=%d", accountID, scanID, itemsVersion)
	rows := make([]recipeRow, len(set))
	for i, n := range set {
		rows[i] = recipeRow{
			Position: i + 1, Title: n.Title, Minutes: n.Minutes, Difficulty: n.Difficulty,
			Equipment: n.Equipment, Uses: n.Uses, Missing: n.Missing, Allergens: n.Allergens,
			Diets: n.Diets, Steps: n.Steps,
		}
	}
	payload, err := json.Marshal(rows)
	if err != nil {
		return false, fmt.Errorf("%s: encode the set: %w", op, err)
	}

	// The join to app.scan is the fence: it cannot add to another player's scan. An empty list
	// arrives as JSON null, so the optional ones fall back to empty; uses and steps stay null,
	// which the table refuses.
	tag, err := r.db.Exec(ctx, `
		insert into app.recipe (scan_id, items_version, position, title, minutes, difficulty,
		                        equipment, uses, missing, allergens, diets, steps)
		select s.scan_id, $3, x.position, x.title, x.minutes, x.difficulty,
		       coalesce(x.equipment, '{}'), x.uses, coalesce(x.missing, '{}'),
		       coalesce(x.allergens, '{}'), coalesce(x.diets, '{}'), x.steps
		  from app.scan s
		 cross join jsonb_to_recordset($4::jsonb)
		       as x (position int, title text, minutes int, difficulty text, equipment text[],
		             uses text[], missing text[], allergens text[], diets text[], steps text[])
		 where s.account_id = $1
		   and s.scan_id = $2`, accountID, scanID, itemsVersion, payload)
	if err != nil {
		err = known(op, err)
		if errors.Is(err, ErrDuplicate) {
			return false, nil
		}
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// ListRecipes reads the set for one version of one of the account's scans, in the model's order,
// narrowed by f. A recipe holding one of the account's allergies never comes back.
func (r *Repository) ListRecipes(
	ctx context.Context, accountID, scanID int64, itemsVersion int, f RecipeFilter,
) ([]Recipe, error) {
	op := fmt.Sprintf("list recipes account=%d scan=%d version=%d", accountID, scanID, itemsVersion)
	equipment := f.Equipment
	if equipment == nil {
		equipment = []string{}
	}
	rows, err := r.db.Query(ctx, `
		select r.recipe_id, r.title, r.minutes, r.difficulty, r.equipment, r.uses, r.missing,
		       r.steps, s.score
		  from app.recipe r
		  join app.scan s on s.scan_id = r.scan_id
		  join app.account a on a.account_id = s.account_id
		 where s.account_id = $1
		   and r.scan_id = $2
		   and r.items_version = $3
		   and not r.allergens && a.allergies
		   and ($4::int = 0 or r.minutes <= $4::int)
		   and (cardinality($5::text[]) = 0 or r.equipment <@ $5::text[])
		   and ($6::text in ('', 'none') or $6::text = any (r.diets))
		 order by r.position`,
		accountID, scanID, itemsVersion, f.MaxMinutes, equipment, f.Diet)
	if err != nil {
		return nil, known(op, err)
	}
	defer rows.Close()

	var out []Recipe
	for rows.Next() {
		var rec Recipe
		err = rows.Scan(&rec.RecipeID, &rec.Title, &rec.Minutes, &rec.Difficulty, &rec.Equipment,
			&rec.Uses, &rec.Missing, &rec.Steps, &rec.ScanScore)
		if err != nil {
			return nil, known(op, err)
		}
		out = append(out, rec)
	}
	err = rows.Err()
	if err != nil {
		return nil, known(op, err)
	}
	return out, nil
}

// RecipeByID reads one of the account's recipes, reached through its scan. A recipe that is
// another player's, that does not exist, or that holds one of the account's allergies is
// ErrNotFound.
func (r *Repository) RecipeByID(ctx context.Context, accountID, recipeID int64) (Recipe, error) {
	var rec Recipe
	err := r.db.QueryRow(ctx, `
		select r.recipe_id, r.title, r.minutes, r.difficulty, r.equipment, r.uses, r.missing,
		       r.steps, s.score
		  from app.recipe r
		  join app.scan s on s.scan_id = r.scan_id
		  join app.account a on a.account_id = s.account_id
		 where s.account_id = $1
		   and r.recipe_id = $2
		   and not r.allergens && a.allergies`, accountID, recipeID).Scan(
		&rec.RecipeID, &rec.Title, &rec.Minutes, &rec.Difficulty, &rec.Equipment, &rec.Uses,
		&rec.Missing, &rec.Steps, &rec.ScanScore)
	if err != nil {
		return Recipe{}, known(fmt.Sprintf("recipe by id account=%d recipe=%d", accountID, recipeID), err)
	}
	return rec, nil
}
