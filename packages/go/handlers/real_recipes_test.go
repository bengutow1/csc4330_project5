// Author: LAPTOP-BPMQJP10\andyk
package handlers_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"

	cookv1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/cook/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/repository"
	"github.com/bengutow1/csc4330_project5/packages/go/vision"
)

// TestRealRecipeClientStillUsesTheServersAllergyFence exercises the real client and serving stack
// over a recorded vendor response, including the control where the same recipe must be visible.
func TestRealRecipeClientStillUsesTheServersAllergyFence(t *testing.T) {
	t.Parallel()
	for _, allergic := range []bool{false, true} {
		t.Run(map[bool]string{false: "safe", true: "eggs allergy"}[allergic], func(t *testing.T) {
			t.Parallel()
			recording, err := os.ReadFile("../vision/testdata/good_recipes.json")
			if err != nil {
				t.Fatalf("recording: %v", err)
			}
			contexts := make(chan map[string]json.RawMessage, 1)
			// This fake is only the external model; the client, handler, sessions and database are real.
			vendor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				raw, err := io.ReadAll(r.Body)
				if err != nil {
					http.Error(w, "read request", http.StatusBadRequest)
					return
				}
				var req struct {
					Messages []struct {
						Content []struct {
							Text string `json:"text"`
						} `json:"content"`
					} `json:"messages"`
				}
				err = json.Unmarshal(raw, &req)
				if err != nil || len(req.Messages) != 1 || len(req.Messages[0].Content) != 1 {
					http.Error(w, "bad prompt", http.StatusBadRequest)
					return
				}
				var input map[string]json.RawMessage
				err = json.Unmarshal([]byte(req.Messages[0].Content[0].Text), &input)
				if err != nil {
					http.Error(w, "bad context", http.StatusBadRequest)
					return
				}
				contexts <- input
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write(recording) // the client reports any truncated response
			}))
			t.Cleanup(vendor.Close)
			client := vision.NewClient(vision.Config{APIKey: "test_key", Model: "test-model", BaseURL: vendor.URL})
			srv := cookStack(t, cookFixture{eyes: newFakeVision(kitchenFridge), chef: client})
			maya := cookPlayerFor(t, srv, "test_maya")
			var allergies []string
			if allergic {
				allergies = []string{repository.AllergyEggs}
			}
			setKitchen(t, srv, "test_maya", []string{repository.EquipmentStovetop}, repository.DietNone, allergies)
			verdict := scanFridge(t, maya)
			got := listRecipes(t, maya, &cookv1.ListRecipesRequest{ScanId: verdict.ScanId})
			want := 1
			if allergic {
				want = 0
			}
			if len(got.MakeItNow) != want || len(got.OneItemAway) != 0 {
				t.Errorf("shelves = %v and %v, want %d safe recipes", got.MakeItNow, got.OneItemAway, want)
			}
			select {
			case input := <-contexts:
				var staples []string
				err = json.Unmarshal(input["staples"], &staples)
				if err != nil || !slices.Contains(staples, "oil") {
					t.Errorf("staples = %v, %v, want enabled oil", staples, err)
				}
				for key, value := range input {
					if key == "handle" || key == "location" || strings.Contains(string(value), "test_maya") {
						t.Error("player identity reached model")
					}
				}
			default:
				t.Fatal("model request was never recorded")
			}
		})
	}
}
