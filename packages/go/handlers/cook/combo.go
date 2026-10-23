// Author: LAPTOP-BPMQJP10\andyk
package cook

import (
	"context"
	"crypto/sha256"
	"fmt"
	"slices"
	"strings"
	"time"

	"connectrpc.com/connect"

	cookv1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/cook/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers"
	"github.com/bengutow1/csc4330_project5/packages/go/points"
	"github.com/bengutow1/csc4330_project5/packages/go/repository"
	"github.com/bengutow1/csc4330_project5/packages/go/scoring"
)

func comboReady(c repository.Combo, now time.Time) error {
	if c.State == repository.ComboEnded {
		return repository.ErrComboEnded
	}
	if !c.ExpiresAt.After(now) {
		return repository.ErrComboExpired
	}
	if len(c.Members) != 2 || c.Members[0].Scan == nil || c.Members[1].Scan == nil {
		return repository.ErrComboNotReady
	}
	return nil
}

func (h *Cook) listComboRecipes(ctx context.Context, me repository.Account, id int64, filter repository.RecipeFilter) (*connect.Response[cookv1.ListRecipesResponse], error) {
	const op = "list combo recipes"
	_, c, err := h.gate.ComboMember(ctx, id)
	if err != nil {
		return nil, h.fail(op, err)
	}
	err = comboReady(c, h.now())
	if err != nil {
		return nil, h.fail(op, err)
	}
	a, b := c.Members[0].Scan, c.Members[1].Scan
	key := setKey{scanID: a.ScanID, version: a.ItemsVersion, partnerScanID: b.ScanID, partnerVersion: b.ItemsVersion}
	err = h.ensureRecipes(ctx, key, func(ctx context.Context) (bool, error) { return h.repo.HasComboRecipes(ctx, c) }, func(ctx context.Context) error { return h.makeComboSet(ctx, me, c) })
	if err != nil {
		return nil, h.fail(op, err)
	}
	set, err := h.repo.ListComboRecipes(ctx, me.AccountID, c, filter)
	if err != nil {
		return nil, h.fail(op, err)
	}
	out, err := toProtoShelves(set)
	if err != nil {
		return nil, h.fail(op, err)
	}
	return connect.NewResponse(out), nil
}

func (h *Cook) makeComboSet(ctx context.Context, me repository.Account, c repository.Combo) error {
	accounts, err := h.repo.ComboAccounts(ctx, c.ComboID)
	if err != nil {
		return err
	}
	if len(accounts) != 2 {
		return repository.ErrComboNotReady
	}
	merged := repository.Scan{ScanID: c.Members[0].Scan.ScanID}
	for _, m := range c.Members {
		merged.Items = append(merged.Items, m.Scan.Items...)
	}
	counted, err := handlers.ScoringItems(merged.Items)
	if err != nil {
		return err
	}
	merged.Score = scoring.Fridge(counted)
	me.Staples = unionWords(accounts[0].Staples, accounts[1].Staples)
	me.Equipment = unionWords(accounts[0].Equipment, accounts[1].Equipment)
	me.Allergies = unionWords(accounts[0].Allergies, accounts[1].Allergies)
	me.Diet, err = combinedDiet(accounts[0].Diet, accounts[1].Diet)
	if err != nil {
		return err
	}
	var set []repository.NewRecipe
	if len(merged.Items) > 0 {
		set, err = h.checkedRecipes(ctx, me, merged)
		if err != nil {
			return err
		}
	}
	for i := range set {
		set[i].IngredientOwners, set[i].OnlyTogether = ingredientOwners(c, set[i].Uses)
	}
	return h.repo.CreateComboRecipes(ctx, me.AccountID, c, merged.Score, set, h.now())
}

func unionWords(a, b []string) []string {
	out := slices.Clone(a)
	for _, word := range b {
		if !slices.Contains(out, word) {
			out = append(out, word)
		}
	}
	return out
}

func combinedDiet(a, b string) (string, error) {
	ranks := map[string]int{repository.DietNone: 0, repository.DietPescatarian: 1, repository.DietVegetarian: 2, repository.DietVegan: 3}
	one, ok := ranks[a]
	if !ok {
		return "", fmt.Errorf("cook: unknown stored diet %q", a)
	}
	two, ok := ranks[b]
	if !ok {
		return "", fmt.Errorf("cook: unknown stored diet %q", b)
	}
	if two > one {
		return b, nil
	}
	return a, nil
}

// A food in both fridges does not require both: only exclusive food on each side earns the tag.
func ingredientOwners(c repository.Combo, uses []string) ([]repository.IngredientOwner, bool) {
	var out []repository.IngredientOwner
	exclusive := 0
	for _, use := range uses {
		mask := 0
		for i, m := range c.Members {
			for _, item := range m.Scan.Items {
				if strings.EqualFold(strings.TrimSpace(item.Name), strings.TrimSpace(use)) {
					mask |= 1 << i
					tag := repository.IngredientOwner{AccountID: m.AccountID, Ingredient: use}
					if !slices.Contains(out, tag) {
						out = append(out, tag)
					}
					break
				}
			}
		}
		if mask == 1 || mask == 2 {
			exclusive |= mask
		}
	}
	return out, exclusive == 3
}

// plateCombo locks membership and both accounts before checking each independent credit.
// Each credited player owns their recipe and meal record, sharing the one validated photo.
func (h *Cook) plateCombo(ctx context.Context, accountID int64, recipe repository.Recipe, photo []byte, now time.Time) (*cookv1.Receipt, error) {
	window, day := points.MealWindow(now)
	today := day.Format(time.DateOnly)
	mealWindow := today + " " + string(window)
	hash := sha256.Sum256(photo)
	var callerPaid paidPlate
	err := h.repo.InTx(ctx, func(tx *repository.Repository) error {
		c, err := tx.LockCombo(ctx, accountID, recipe.ComboID)
		if err != nil {
			return err
		}
		err = comboReady(c, h.now())
		if err != nil {
			return err
		}
		// Re-read the recipe's allergy fence after membership is locked and the model has finished.
		recipe, err = tx.RecipeByID(ctx, accountID, recipe.RecipeID)
		if err != nil {
			return err
		}
		partnerID, partnerRecipeID, err := tx.ComboPartnerRecipe(ctx, accountID, recipe.RecipeID)
		if err != nil {
			return err
		}
		ids := []int64{accountID, partnerID}
		slices.Sort(ids)
		for _, id := range ids {
			err = tx.LockAccount(ctx, id)
			if err != nil {
				return err
			}
		}
		var originalPlateID int64
		credited := 0
		for _, id := range ids {
			paid := paidPlate{}
			paid.before, err = tx.PlayerStats(ctx, id)
			if err != nil {
				return err
			}
			paid.totalBefore, err = tx.PointsTotal(ctx, id)
			if err != nil {
				return err
			}
			cooked, err := tx.CookDays(ctx, id)
			if err != nil {
				return err
			}
			days, err := handlers.CampusDays(cooked)
			if err != nil {
				return err
			}
			used, err := tx.HasCookCredit(ctx, id, mealWindow)
			if err != nil {
				return err
			}
			paid.streak = points.Streak(days, day)
			if !used {
				paid.streak = points.Streak(append(days, day), day)
				paid.lines = points.PlateAward(points.Plate{Score: recipe.ScanScore, Combo: true, FirstOfDay: !slices.Contains(cooked, today), Streak: paid.streak})
				recipeID := recipe.RecipeID
				if id == partnerID {
					recipeID = partnerRecipeID
				}
				// Re-check both settings, including an allergy changed while the vendor was running.
				_, err = tx.RecipeByID(ctx, id, recipeID)
				if err != nil {
					return err
				}
				var plateID int64
				if originalPlateID == 0 {
					plateID, err = tx.CreatePlate(ctx, id, repository.NewPlate{RecipeID: recipeID, Photo: photo, PhotoSHA256: hash[:], At: now})
					originalPlateID = plateID
				} else {
					plateID, err = tx.CreateComboPlateCopy(ctx, id, originalPlateID, recipeID)
				}
				if err != nil {
					return err
				}
				err = tx.AwardPoints(ctx, id, plateID, mealWindow, now, toAwards(paid.lines))
				if err != nil {
					return err
				}
				credited++
			}
			paid.after, err = tx.PlayerStats(ctx, id)
			if err != nil {
				return err
			}
			paid.totalAfter, err = tx.PointsTotal(ctx, id)
			if err != nil {
				return err
			}
			if id == accountID {
				callerPaid = paid
			}
		}
		if credited == 0 {
			return repository.ErrMealWindowCooked
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return toProtoReceipt(fmt.Sprintf("Looks like %s", recipe.Title), callerPaid)
}
