// Author: LAPTOP-BPMQJP10\andyk
package vision_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/bengutow1/csc4330_project5/packages/go/vision"
)

// TestRecordedRecipesKeepTheirIngredientsAndTimers pins the model shape before the handler checks it.
func TestRecordedRecipesKeepTheirIngredientsAndTimers(t *testing.T) {
	t.Parallel()
	_, client := newVendor(t, http.StatusOK, "good_recipes.json")
	got, err := client.Recipes(t.Context(), []vision.Item{{Name: "Eggs", Category: "protein", Quantity: "x6"}}, vision.Kitchen{Staples: []string{"oil"}, Equipment: []string{"stovetop"}, Diet: "none", Allergies: []string{"peanuts"}})
	if err != nil {
		t.Fatalf("recipes: %v", err)
	}
	want := []vision.Recipe{{Title: "Scrambled eggs", Minutes: 5, Difficulty: "easy", Equipment: []string{"stovetop"}, Uses: []string{"Eggs", "oil"}, Missing: []string{}, Allergens: []string{"eggs"}, Steps: []string{"Warm the oil.", "Stir the eggs until set."}, StepTimers: []int{0, 120}}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("recipes = %+v, want %+v", got, want)
	}
}

// TestRecipePromptCarriesOnlyFoodAndKitchenSettings prevents accidental identity or location sharing.
func TestRecipePromptCarriesOnlyFoodAndKitchenSettings(t *testing.T) {
	t.Parallel()
	v, client := newVendor(t, http.StatusOK, "good_recipes.json")
	_, err := client.Recipes(t.Context(), []vision.Item{{Name: "Eggs", Category: "protein", Quantity: "x6"}}, vision.Kitchen{Staples: []string{"oil"}, Equipment: []string{"stovetop"}, Diet: "vegetarian", Allergies: []string{"peanuts"}})
	if err != nil {
		t.Fatalf("recipes: %v", err)
	}
	calls := v.recorded()
	if len(calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(calls))
	}
	request := calls[0].body
	msgs := request["messages"].([]any)
	blocks := msgs[0].(map[string]any)["content"].([]any)
	if len(blocks) != 1 {
		t.Fatalf("blocks = %d, want only ingredient context", len(blocks))
	}
	var context map[string]any
	err = json.Unmarshal([]byte(blocks[0].(map[string]any)["text"].(string)), &context)
	if err != nil {
		t.Fatalf("prompt context: %v", err)
	}
	want := map[string]any{"ingredients": []any{map[string]any{"name": "Eggs", "category": "protein", "quantity": "x6"}}, "staples": []any{"oil"}, "equipment": []any{"stovetop"}, "diet": "vegetarian", "allergies": []any{"peanuts"}}
	if !reflect.DeepEqual(context, want) {
		t.Errorf("context = %v, want exactly %v", context, want)
	}
	assertStructuredCookRequest(t, request, "uses", "missing", "allergens", "step_timers")
}

// TestRecordedPlateChecksRequireFoodAndTheRecipeMatch prevents a non-food match from earning points.
func TestRecordedPlateChecksRequireFoodAndTheRecipeMatch(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, file string
		pass       bool
		reason     string
	}{
		{"matches", "good_plate.json", true, "The photo shows scrambled eggs."},
		{"wrong dish", "wrong_plate.json", false, "The photo shows soup instead of scrambled eggs."},
		{"not food", "not_food_plate.json", false, "The photo shows a printed recipe card."},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			_, client := newVendor(t, http.StatusOK, c.file)
			got, err := client.CheckPlate(t.Context(), []byte("plate jpeg"), "Scrambled eggs")
			if err != nil {
				t.Fatalf("plate: %v", err)
			}
			if got.Passed != c.pass || got.Reason != c.reason {
				t.Errorf("plate = %+v, want %v, %q", got, c.pass, c.reason)
			}
		})
	}
}

// TestPlatePromptCarriesOnlyThePhotoAndRecipeTitle keeps the plate model from deciding rewards.
func TestPlatePromptCarriesOnlyThePhotoAndRecipeTitle(t *testing.T) {
	t.Parallel()
	v, client := newVendor(t, http.StatusOK, "good_plate.json")
	photo := []byte("plate jpeg")
	_, err := client.CheckPlate(t.Context(), photo, "Scrambled eggs")
	if err != nil {
		t.Fatalf("plate: %v", err)
	}
	calls := v.recorded()
	if len(calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(calls))
	}
	msgs := calls[0].body["messages"].([]any)
	blocks := msgs[0].(map[string]any)["content"].([]any)
	if len(blocks) != 2 {
		t.Fatalf("blocks = %d, want photo and title", len(blocks))
	}
	source := blocks[0].(map[string]any)["source"].(map[string]any)
	if source["data"] != base64.StdEncoding.EncodeToString(photo) || source["media_type"] != "image/jpeg" {
		t.Errorf("source is not the supplied JPEG")
	}
	var title map[string]string
	err = json.Unmarshal([]byte(blocks[1].(map[string]any)["text"].(string)), &title)
	if err != nil || !reflect.DeepEqual(title, map[string]string{"recipe_title": "Scrambled eggs"}) {
		t.Errorf("title = %v, %v", title, err)
	}
	assertStructuredCookRequest(t, calls[0].body, "shows_food", "matches_recipe", "reason")
}

// TestCookCallsRejectMalformedResponsesAndNeverRetry refuses partial recipes and ambiguous plate verdicts.
func TestCookCallsRejectMalformedResponsesAndNeverRetry(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"recipes", "plate"} {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()
			for _, c := range []struct {
				name, answer string
				status       int
			}{
				{"html", "<html>Bad gateway</html>", http.StatusOK},
				{"missing fields", "{}", http.StatusOK},
				{"null", "null", http.StatusOK},
				{"unknown field", `{"score":100}`, http.StatusOK},
				{"server outage", `{}`, http.StatusServiceUnavailable},
			} {
				t.Run(c.name, func(t *testing.T) {
					t.Parallel()
					v := answering(t, c.answer)
					v.status = c.status
					_, client := serve(t, v)
					if kind == "recipes" {
						got, err := client.Recipes(t.Context(), nil, vision.Kitchen{})
						if !errors.Is(err, vision.ErrUnavailable) || got != nil {
							t.Errorf("recipes = %v, %v, want unavailable and no partial result", got, err)
						}
					} else {
						got, err := client.CheckPlate(t.Context(), []byte("photo"), "Eggs")
						if !errors.Is(err, vision.ErrUnavailable) || got != (vision.PlateCheck{}) {
							t.Errorf("plate = %+v, %v, want unavailable and no verdict", got, err)
						}
					}
					if len(v.recorded()) != 1 {
						t.Errorf("calls = %d, want 1", len(v.recorded()))
					}
				})
			}
		})
	}
}

// TestCookTimeoutsAndCallerCancellationStayDistinct preserves the shared error boundary's semantics.
func TestCookTimeoutsAndCallerCancellationStayDistinct(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"recipes", "plate"} {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()
			for _, caller := range []bool{false, true} {
				t.Run(map[bool]string{false: "provider timeout", true: "caller timeout"}[caller], func(t *testing.T) {
					t.Parallel()
					v := recorded(t, "good_recipes.json")
					v.delay = 2 * time.Second
					_, client := serve(t, v)
					ctx := t.Context()
					if caller {
						var cancel context.CancelFunc
						ctx, cancel = context.WithTimeout(ctx, 50*time.Millisecond)
						defer cancel()
					} else {
						client.WithTimeoutForTest(50 * time.Millisecond)
					}
					var err error
					if kind == "recipes" {
						_, err = client.Recipes(ctx, nil, vision.Kitchen{})
					} else {
						_, err = client.CheckPlate(ctx, []byte("photo"), "Eggs")
					}
					if caller {
						if !errors.Is(err, context.DeadlineExceeded) || errors.Is(err, vision.ErrUnavailable) {
							t.Errorf("caller error = %v", err)
						}
					} else if !errors.Is(err, vision.ErrUnavailable) || errors.Is(err, context.DeadlineExceeded) {
						t.Errorf("provider timeout = %v", err)
					}
				})
			}
		})
	}
}

func assertStructuredCookRequest(t *testing.T, body map[string]any, fields ...string) {
	t.Helper()
	format := body["output_config"].(map[string]any)["format"].(map[string]any)
	if format["type"] != "json_schema" {
		t.Errorf("format = %v, want json_schema", format["type"])
	}
	raw, err := json.Marshal(format["schema"])
	if err != nil {
		t.Fatalf("schema: %v", err)
	}
	for _, field := range fields {
		if !strings.Contains(string(raw), `"`+field+`"`) {
			t.Errorf("schema missing %s", field)
		}
	}
}
