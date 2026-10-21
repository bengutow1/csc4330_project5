// Author: Caden Lund
package handlers

import (
	"maps"
	"slices"
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"

	accountv1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/account/v1"
)

// TestEveryKitchenWireValueHasExactlyOneStoredWord: a value added to a closed set on one side only
// would either never reach the app, or fail every rpc that maps a player's pick of it.
func TestEveryKitchenWireValueHasExactlyOneStoredWord(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		enum   protoreflect.EnumDescriptor
		mapped map[protoreflect.EnumNumber]int
	}{
		{name: "equipment", enum: accountv1.Equipment(0).Descriptor(), mapped: counts(equipmentWire)},
		{name: "diet", enum: accountv1.Diet(0).Descriptor(), mapped: counts(dietWire)},
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

// TestKitchenWordsGoBothWays: a filter reads a word off the wire and a recipe writes one back, so
// each value must come back as itself, and unspecified must never pass as a word.
func TestKitchenWordsGoBothWays(t *testing.T) {
	t.Parallel()
	for word, e := range equipmentWire {
		back, ok := EquipmentWord(e)
		wired, err := WireEquipment([]string{word})
		if !ok || back != word || err != nil || !slices.Equal(wired, []accountv1.Equipment{e}) {
			t.Errorf("equipment %q: word %q (%v), wire %v (%v); want it back both ways", word, back, ok, wired, err)
		}
	}
	for word, d := range dietWire {
		back, ok := DietWord(d)
		wired, err := WireDiet(word)
		if !ok || back != word || err != nil || wired != d {
			t.Errorf("diet %q: word %q (%v), wire %v (%v); want it back both ways", word, back, ok, wired, err)
		}
	}
	word, ok := EquipmentWord(accountv1.Equipment_EQUIPMENT_UNSPECIFIED)
	if ok {
		t.Errorf("unspecified equipment = %q, want no word", word)
	}
	word, ok = DietWord(accountv1.Diet_DIET_UNSPECIFIED)
	if ok {
		t.Errorf("unspecified diet = %q, want no word", word)
	}
	_, err := WireEquipment([]string{"grill"})
	if err == nil {
		t.Error("equipment \"grill\" mapped to the wire, want an error")
	}
	_, err = WireDiet("keto")
	if err == nil {
		t.Error("diet \"keto\" mapped to the wire, want an error")
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
