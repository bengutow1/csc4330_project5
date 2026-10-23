// Author: Caden Lund
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// recipe.go owns app.recipe_set and app.recipe: one set per version of a scan's list, recorded
// even when it is empty, and the dishes kept in it, written once and never updated. Neither has an
// account of its own, so every query reaches the owner through the scan, and every read also hides
// a recipe holding one of the owner's allergies, which can have changed since the set was made.

// The difficulties, mirroring app.recipe's check.
const (
	DifficultyEasy   = "easy"
	DifficultyMedium = "medium"
	DifficultyHard   = "hard"
)

// NewRecipe is one checked dish to store in a set.
type NewRecipe struct {
	Title            string
	Minutes          int      // the whole time to the plate
	Difficulty       string   // a Difficulty const
	Equipment        []string // Equipment consts; empty needs none
	Uses             []string // the food it takes that the player has
	Missing          []string // the food it takes that the player lacks: none, or one
	Allergens        []string // the Allergy consts it contains
	Diets            []string // the Diet consts besides none that it fits
	Steps            []string
	StepTimers       []int // aligned seconds; 0 is untimed, empty means all untimed
	IngredientOwners []IngredientOwner
	OnlyTogether     bool
}

// IngredientOwner is the stored provenance of a scanned food used in a combo dish.
type IngredientOwner struct {
	AccountID  int64  `json:"account_id"`
	Ingredient string `json:"ingredient"`
}

// Recipe is one stored dish, as the shelves and the recipe page read it.
type Recipe struct {
	RecipeID            int64
	Title               string
	Minutes             int
	Difficulty          string // a Difficulty const
	Equipment           []string
	Uses                []string
	Missing             []string // none, or one
	Steps               []string
	StepTimers          []int // aligned seconds; 0 is untimed, empty means all untimed
	ScanScore           int   // the score of the recipe's scan, which sets what plating it pays
	ScanID              int64
	ItemsVersion        int
	PartnerScanID       int64 // 0 for a solo set; otherwise the other list's snapshot identity.
	PartnerItemsVersion int
	Position            int
	ComboID             int64
	IngredientOwners    []IngredientOwner
	OnlyTogether        bool
}

// RecipeFilter narrows a set. Each zero value filters nothing.
type RecipeFilter struct {
	MaxMinutes int      // the longest a recipe may take; 0 is any
	Equipment  []string // keep recipes that need nothing beyond these; empty is any
	Diet       string   // keep recipes that fit this Diet const; "" and none are any
}

// HasRecipes reports whether one of the account's scans already has its set for itemsVersion,
// an empty one included.
func (r *Repository) HasRecipes(
	ctx context.Context, accountID, scanID int64, itemsVersion int,
) (bool, error) {
	var has bool
	err := r.db.QueryRow(ctx, `
		select exists (
		       select 1
		         from app.recipe_set rs
		         join app.scan s on s.scan_id = rs.scan_id
		        where s.account_id = $1
		          and rs.scan_id = $2
		          and rs.items_version = $3
		          and rs.partner_scan_id = 0)`, accountID, scanID, itemsVersion).Scan(&has)
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
	Position         int               `json:"position"`
	Title            string            `json:"title"`
	Minutes          int               `json:"minutes"`
	Difficulty       string            `json:"difficulty"`
	Equipment        []string          `json:"equipment"`
	Uses             []string          `json:"uses"`
	Missing          []string          `json:"missing"`
	Allergens        []string          `json:"allergens"`
	Diets            []string          `json:"diets"`
	Steps            []string          `json:"steps"`
	StepTimers       []int             `json:"step_timers"`
	IngredientOwners []IngredientOwner `json:"ingredient_owners"`
	OnlyTogether     bool              `json:"only_together"`
}

// errSetTaken unwinds CreateRecipes' transaction when the version already has its set.
var errSetTaken = errors.New("repository: the version already has its set")

// CreateRecipes makes the set for one version of one of the account's scans and stores its
// recipes, numbered in the order given. An empty set is made too, so the version is never asked
// for again. stored is false when the scan is not the account's, or when the version already has
// its set: the unique key on app.recipe_set decides, so of two requests that raced, one stores its
// whole set and the other stores nothing.
func (r *Repository) CreateRecipes(
	ctx context.Context, accountID, scanID int64, itemsVersion int, set []NewRecipe,
) (stored bool, err error) {
	return r.createRecipes(ctx, accountID, scanID, itemsVersion, 0, 0, -1, set)
}

func (r *Repository) createRecipes(
	ctx context.Context, accountID, scanID int64, itemsVersion int, partnerScanID int64, partnerVersion, comboScore int, set []NewRecipe,
) (stored bool, err error) {
	op := fmt.Sprintf("create recipes account=%d scan=%d version=%d", accountID, scanID, itemsVersion)
	rows := make([]recipeRow, len(set))
	for i, n := range set {
		rows[i] = recipeRow{
			Position: i + 1, Title: n.Title, Minutes: n.Minutes, Difficulty: n.Difficulty,
			Equipment: n.Equipment, Uses: n.Uses, Missing: n.Missing, Allergens: n.Allergens,
			Diets: n.Diets, Steps: n.Steps, StepTimers: n.StepTimers,
			IngredientOwners: n.IngredientOwners, OnlyTogether: n.OnlyTogether,
		}
	}
	payload, err := json.Marshal(rows)
	if err != nil {
		return false, fmt.Errorf("%s: encode the set: %w", op, err)
	}

	err = r.InTx(ctx, func(tx *Repository) error {
		// 1. The set itself. The join to app.scan is the fence: it cannot make a set on another
		// player's scan
		var setID int64
		err := tx.db.QueryRow(ctx, `
			insert into app.recipe_set (scan_id, items_version,partner_scan_id,partner_items_version,combo_score)
			select s.scan_id, $3,$4,$5,nullif($6,-1)
			  from app.scan s
			 where s.account_id = $1
			   and s.scan_id = $2 returning recipe_set_id`, accountID, scanID, itemsVersion, partnerScanID, partnerVersion, comboScore).Scan(&setID)
		if isNoRows(err) {
			return nil
		}
		if err != nil {
			err = known(op, err)
			if errors.Is(err, ErrDuplicate) {
				return errSetTaken
			}
			return err
		}

		// 2. Its recipes, in one statement. An empty list arrives as JSON null, so the optional
		// ones fall back to empty; uses and steps stay null, which the table refuses
		_, err = tx.db.Exec(ctx, `
			insert into app.recipe (scan_id, items_version, recipe_set_id,position, title, minutes, difficulty,
			                        equipment, uses, missing, allergens, diets, steps, step_timers,ingredient_owners,only_together)
			select $1, $2,$4, x.position, x.title, x.minutes, x.difficulty,
			       coalesce(x.equipment, '{}'), x.uses, coalesce(x.missing, '{}'),
			       coalesce(x.allergens, '{}'), coalesce(x.diets, '{}'), x.steps, coalesce(x.step_timers, '{}'),coalesce(x.ingredient_owners,'[]'::jsonb),x.only_together
			  from jsonb_to_recordset($3::jsonb)
			       as x (position int, title text, minutes int, difficulty text, equipment text[],
			             uses text[], missing text[], allergens text[], diets text[], steps text[], step_timers int[],ingredient_owners jsonb,only_together boolean)`,
			scanID, itemsVersion, payload, setID)
		if err != nil {
			return known(op, err)
		}
		stored = true
		return nil
	})
	if errors.Is(err, errSetTaken) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return stored, nil
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
		       r.steps, r.step_timers, s.score
		  from app.recipe r
		  join app.scan s on s.scan_id = r.scan_id
		  join app.account a on a.account_id = s.account_id
		  join app.recipe_set rs on rs.recipe_set_id=r.recipe_set_id and rs.partner_scan_id=0
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
			&rec.Uses, &rec.Missing, &rec.Steps, &rec.StepTimers, &rec.ScanScore)
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
		       r.steps, r.step_timers, coalesce(rs.combo_score,s.score),s.scan_id,r.items_version,
		       rs.partner_scan_id,rs.partner_items_version,r.position,
		       case when rs.partner_scan_id=0 then 0 else s.combo_id end,r.ingredient_owners,r.only_together
		  from app.recipe r
		  join app.scan s on s.scan_id = r.scan_id
		  join app.account a on a.account_id = s.account_id
		  join app.recipe_set rs on rs.recipe_set_id=r.recipe_set_id
		 where s.account_id = $1
		   and r.recipe_id = $2
		   and not r.allergens && a.allergies
		   and (rs.partner_scan_id=0 or exists (
		       select 1 from app.scan partner
		       join app.combo_member mine on mine.combo_id=s.combo_id and mine.account_id=$1
		       join app.combo_member other on other.combo_id=mine.combo_id and other.account_id=partner.account_id
		       join app.account buddy on buddy.account_id=other.account_id
		       where partner.scan_id=rs.partner_scan_id and partner.combo_id=s.combo_id
		       and not r.allergens && buddy.allergies))`, accountID, recipeID).Scan(
		&rec.RecipeID, &rec.Title, &rec.Minutes, &rec.Difficulty, &rec.Equipment, &rec.Uses,
		&rec.Missing, &rec.Steps, &rec.StepTimers, &rec.ScanScore, &rec.ScanID, &rec.ItemsVersion,
		&rec.PartnerScanID, &rec.PartnerItemsVersion, &rec.Position, &rec.ComboID, &rec.IngredientOwners, &rec.OnlyTogether)
	if err != nil {
		return Recipe{}, known(fmt.Sprintf("recipe by id account=%d recipe=%d", accountID, recipeID), err)
	}
	return rec, nil
}
