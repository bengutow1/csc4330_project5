// Author: LAPTOP-BPMQJP10\andyk
package cook

import (
	"testing"

	"github.com/bengutow1/csc4330_project5/packages/go/repository"
)

// TestOnlyTogetherNeedsExclusiveFoodFromBothFridges prevents a shared staple earning the tag.
func TestOnlyTogetherNeedsExclusiveFoodFromBothFridges(t *testing.T) {
	t.Parallel()
	c := repository.Combo{Members: []repository.ComboMember{
		{AccountID: 1, Scan: &repository.Scan{Items: []repository.ScanItem{{Name: "Eggs"}, {Name: "Rice"}}}},
		{AccountID: 2, Scan: &repository.Scan{Items: []repository.ScanItem{{Name: "Peas"}, {Name: "Rice"}}}},
	}}
	cases := []struct {
		name   string
		uses   []string
		want   bool
		owners int
	}{
		{"both exclusive", []string{"Eggs", "Peas"}, true, 2},
		{"one fridge", []string{"Eggs"}, false, 1},
		{"shared food", []string{"Rice"}, false, 2},
		{"one exclusive and shared", []string{"Eggs", "Rice"}, false, 3},
		{"unscanned staple", []string{"Eggs", "oil"}, false, 1},
		{"normalized food", []string{" eggs ", "PEAS"}, true, 2},
	}
	for _, cse := range cases {
		t.Run(cse.name, func(t *testing.T) {
			t.Parallel()
			owners, tag := ingredientOwners(c, cse.uses)
			if tag != cse.want || len(owners) != cse.owners {
				t.Errorf("tag=%v owners=%v, want %v and %d", tag, owners, cse.want, cse.owners)
			}
		})
	}
}
