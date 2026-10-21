// Author: Caden Lund
package account

import (
	"maps"
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"

	accountv1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/account/v1"
)

// TestEveryWireValueHasExactlyOneStoredWord: a value added to a closed set on one side only would
// either never reach the app, or fail every WhoAmI of a player who picked it.
func TestEveryWireValueHasExactlyOneStoredWord(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		enum   protoreflect.EnumDescriptor
		mapped map[protoreflect.EnumNumber]int
	}{
		{name: "roast level", enum: accountv1.RoastLevel(0).Descriptor(), mapped: counts(roastLevels)},
		{name: "equipment", enum: accountv1.Equipment(0).Descriptor(), mapped: counts(equipment)},
		{name: "diet", enum: accountv1.Diet(0).Descriptor(), mapped: counts(diets)},
		{name: "allergy", enum: accountv1.Allergy(0).Descriptor(), mapped: counts(allergies)},
		{name: "staple", enum: accountv1.Staple(0).Descriptor(), mapped: counts(staples)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			// Every value but UNSPECIFIED, once each.
			want := map[protoreflect.EnumNumber]int{}
			values := c.enum.Values()
			for i := range values.Len() {
				n := values.Get(i).Number()
				if n != 0 {
					want[n] = 1
				}
			}
			if !maps.Equal(c.mapped, want) {
				t.Errorf("stored words per wire value = %v, want %v", c.mapped, want)
			}
		})
	}
}

// counts is how many stored words map to each wire value.
func counts[E interface {
	Number() protoreflect.EnumNumber
}](set map[string]E) map[protoreflect.EnumNumber]int {
	out := map[protoreflect.EnumNumber]int{}
	for _, e := range set {
		out[e.Number()]++
	}
	return out
}
