// Author: Caden Lund
package handlers

import (
	"fmt"

	"google.golang.org/protobuf/types/known/timestamppb"

	scanv1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/scan/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/repository"
	"github.com/bengutow1/csc4330_project5/packages/go/scoring"
	"github.com/bengutow1/csc4330_project5/packages/go/vision"
)

// categories is the closed set in one table: the word app.scan_item stores, the wire's enum, and
// the category the score counts it under. A word missing here fails the rpc with a logged
// internal error rather than reaching the app as unspecified.
var categories = []struct {
	word    string
	wire    scanv1.Category
	scoring scoring.Category
}{
	{repository.CategoryProtein, scanv1.Category_CATEGORY_PROTEIN, scoring.Protein},
	{repository.CategoryVegetable, scanv1.Category_CATEGORY_VEGETABLE, scoring.Vegetable},
	{repository.CategoryFruit, scanv1.Category_CATEGORY_FRUIT, scoring.Fruit},
	{repository.CategoryGrain, scanv1.Category_CATEGORY_GRAIN, scoring.Grain},
	{repository.CategoryDairy, scanv1.Category_CATEGORY_DAIRY, scoring.Dairy},
	{repository.CategoryCondiment, scanv1.Category_CATEGORY_CONDIMENT, scoring.Condiment},
	{repository.CategoryDrink, scanv1.Category_CATEGORY_DRINK, scoring.Drink},
	{repository.CategorySnack, scanv1.Category_CATEGORY_SNACK, scoring.Snack},
	{repository.CategoryLeftover, scanv1.Category_CATEGORY_LEFTOVER, scoring.Leftover},
	{repository.CategoryOther, scanv1.Category_CATEGORY_OTHER, scoring.Other},
}

// toProtoVerdict maps a stored scan to the wire, with its tier worked out from its score.
func toProtoVerdict(s repository.Scan) (*scanv1.Verdict, error) {
	return ToProtoVerdict(s)
}

// ToProtoVerdict shares stored-scan mapping between solo and combo responses.
func ToProtoVerdict(s repository.Scan) (*scanv1.Verdict, error) {
	tier, err := wireTier(scoring.TierOf(s.Score))
	if err != nil {
		return nil, err
	}
	ingredients, err := wireIngredients(s.Items)
	if err != nil {
		return nil, err
	}
	return &scanv1.Verdict{
		ScanId:      s.ScanID,
		Score:       int32(s.Score), // 0 to 100, held there by app.scan's check
		Tier:        tier,
		Roast:       s.Roast,
		Ingredients: ingredients,
		CreatedAt:   timestamppb.New(s.CreatedAt),
	}, nil
}

// categoryWord is the stored word for a category from the wire. Unspecified is "", which
// validation refuses as required; a value newer than this server keeps its number, which
// validation refuses naming the categories it knows.
func categoryWord(c scanv1.Category) string {
	if c == scanv1.Category_CATEGORY_UNSPECIFIED {
		return ""
	}
	for _, row := range categories {
		if row.wire == c {
			return row.word
		}
	}
	return c.String()
}

// wireCategory is the wire's enum for a stored word.
func wireCategory(word string) (scanv1.Category, error) {
	for _, row := range categories {
		if row.word == word {
			return row.wire, nil
		}
	}
	return scanv1.Category_CATEGORY_UNSPECIFIED, fmt.Errorf("scan: no wire category for %q", word)
}

// wireIngredients maps a stored list to the wire, in order.
func wireIngredients(items []repository.ScanItem) ([]*scanv1.Ingredient, error) {
	out := make([]*scanv1.Ingredient, 0, len(items))
	for _, item := range items {
		category, err := wireCategory(item.Category)
		if err != nil {
			return nil, err
		}
		out = append(out, &scanv1.Ingredient{
			Name:     item.Name,
			Category: category,
			Quantity: item.Quantity,
		})
	}
	return out, nil
}

// ScoringItems validates the category mapping shared by solo and combined scoring.
func ScoringItems(items []repository.ScanItem) ([]scoring.Item, error) {
	out := make([]scoring.Item, 0, len(items))
	for _, item := range items {
		category, err := scoringCategory(item.Category)
		if err != nil {
			return nil, err
		}
		out = append(out, scoring.Item{Name: item.Name, Category: category})
	}
	return out, nil
}

// scoringCategory is the category the score counts a stored word under.
func scoringCategory(word string) (scoring.Category, error) {
	for _, row := range categories {
		if row.word == word {
			return row.scoring, nil
		}
	}
	return "", fmt.Errorf("scan: no scoring category for %q", word)
}

// wireTier is the wire's enum for a tier.
func wireTier(t scoring.Tier) (scanv1.Tier, error) {
	return WireTier(t)
}

// WireTier keeps combined and individual verdicts on the same closed set.
func WireTier(t scoring.Tier) (scanv1.Tier, error) {
	switch t {
	case scoring.TierCertifiedChef:
		return scanv1.Tier_TIER_CERTIFIED_CHEF, nil
	case scoring.TierHomeCook:
		return scanv1.Tier_TIER_HOME_COOK, nil
	case scoring.TierScrappy:
		return scanv1.Tier_TIER_SCRAPPY, nil
	case scoring.TierFriesInTheBag:
		return scanv1.Tier_TIER_FRIES_IN_THE_BAG, nil
	default:
		return scanv1.Tier_TIER_UNSPECIFIED, fmt.Errorf("scan: no wire tier for %q", t)
	}
}

// roastLevel is the level the model roasts at, from the player's kitchen settings.
func roastLevel(level string) (vision.RoastLevel, error) {
	switch level {
	case repository.RoastMild:
		return vision.RoastMild, nil
	case repository.RoastMedium:
		return vision.RoastMedium, nil
	case repository.RoastExtraCrispy:
		return vision.RoastExtraCrispy, nil
	default:
		return "", fmt.Errorf("scan: no roast level for %q", level)
	}
}

// CategoryWord shares scan category validation with every verdict consumer.
func CategoryWord(c scanv1.Category) string { return categoryWord(c) }

// RoastLevel maps the stored preference to the vision boundary.
func RoastLevel(level string) (vision.RoastLevel, error) { return roastLevel(level) }
