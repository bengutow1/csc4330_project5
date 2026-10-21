// Author: Caden Lund
package account

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"strings"

	accountv1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/account/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers"
	"github.com/bengutow1/csc4330_project5/packages/go/repository"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// The closed sets only the account speaks, from the words the database stores to the wire's
// enums. A word missing here fails the rpc with a logged internal error rather than reaching the
// app as unspecified. Equipment and diets are in the handlers root, because recipes speak them
// too.
var (
	roastLevels = map[string]accountv1.RoastLevel{
		repository.RoastMild:        accountv1.RoastLevel_ROAST_LEVEL_MILD,
		repository.RoastMedium:      accountv1.RoastLevel_ROAST_LEVEL_MEDIUM,
		repository.RoastExtraCrispy: accountv1.RoastLevel_ROAST_LEVEL_EXTRA_CRISPY,
	}
	allergies = map[string]accountv1.Allergy{
		repository.AllergyPeanuts:   accountv1.Allergy_ALLERGY_PEANUTS,
		repository.AllergyTreeNuts:  accountv1.Allergy_ALLERGY_TREE_NUTS,
		repository.AllergyMilk:      accountv1.Allergy_ALLERGY_MILK,
		repository.AllergyEggs:      accountv1.Allergy_ALLERGY_EGGS,
		repository.AllergyWheat:     accountv1.Allergy_ALLERGY_WHEAT,
		repository.AllergySoy:       accountv1.Allergy_ALLERGY_SOY,
		repository.AllergyFish:      accountv1.Allergy_ALLERGY_FISH,
		repository.AllergyShellfish: accountv1.Allergy_ALLERGY_SHELLFISH,
		repository.AllergySesame:    accountv1.Allergy_ALLERGY_SESAME,
	}
	staples = map[string]accountv1.Staple{
		repository.StapleSalt:   accountv1.Staple_STAPLE_SALT,
		repository.StaplePepper: accountv1.Staple_STAPLE_PEPPER,
		repository.StapleOil:    accountv1.Staple_STAPLE_OIL,
		repository.StapleButter: accountv1.Staple_STAPLE_BUTTER,
		repository.StapleRice:   accountv1.Staple_STAPLE_RICE,
		repository.StaplePasta:  accountv1.Staple_STAPLE_PASTA,
		repository.StapleFlour:  accountv1.Staple_STAPLE_FLOUR,
		repository.StapleSugar:  accountv1.Staple_STAPLE_SUGAR,
		repository.StapleSpices: accountv1.Staple_STAPLE_SPICES,
	}
)

// The root's equipment and diets, keyed by stored word like the sets above so a refusal can name
// every word. They are built from the root's own helpers, which leaves no second copy of either.
var (
	equipment = setOf(handlers.EquipmentWord)
	diets     = setOf(handlers.DietWord)
)

// toProtoAccount maps a stored account to the wire.
func toProtoAccount(a repository.Account) (*accountv1.Account, error) {
	roastLevel, err := wire(roastLevels, a.RoastLevel, "roast level")
	if err != nil {
		return nil, err
	}
	diet, err := handlers.WireDiet(a.Diet)
	if err != nil {
		return nil, err
	}
	kitchenEquipment, err := handlers.WireEquipment(a.Equipment)
	if err != nil {
		return nil, err
	}
	kitchenAllergies, err := wireEach(allergies, a.Allergies, "allergy")
	if err != nil {
		return nil, err
	}
	kitchenStaples, err := wireEach(staples, a.Staples, "staple")
	if err != nil {
		return nil, err
	}
	return &accountv1.Account{
		Handle: a.Handle,
		Avatar: a.Avatar,
		Kitchen: &accountv1.Kitchen{
			RoastLevel: roastLevel,
			Equipment:  kitchenEquipment,
			Diet:       diet,
			Allergies:  kitchenAllergies,
			Staples:    kitchenStaples,
		},
	}, nil
}

// wire is the enum one stored word maps to.
func wire[E any](set map[string]E, word, name string) (E, error) {
	e, ok := set[word]
	if !ok {
		var none E
		return none, fmt.Errorf("account: no wire %s for %q", name, word)
	}
	return e, nil
}

// wireEach maps a stored list in order.
func wireEach[E any](set map[string]E, words []string, name string) ([]E, error) {
	out := make([]E, 0, len(words))
	for _, word := range words {
		e, err := wire(set, word, name)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}

// enum is a wire enum: numbered in its closed set's order, with UNSPECIFIED at 0.
type enum interface{ ~int32 }

// fromProtoKitchen maps a kitchen off the wire to the words the database stores. Every value
// outside its closed set, unspecified included, is refused on its own field. Each list comes back
// in the closed set's order with repeats dropped: a kitchen has an oven or it doesn't.
func fromProtoKitchen(k *accountv1.Kitchen) (repository.KitchenEdit, handlers.FieldErrors) {
	var invalid handlers.FieldErrors
	edit := repository.KitchenEdit{
		RoastLevel: stored(roastLevels, k.GetRoastLevel(), "kitchen.roast_level", "roast level", &invalid),
		Equipment:  storedEach(equipment, k.GetEquipment(), "kitchen.equipment", "equipment", &invalid),
		Diet:       stored(diets, k.GetDiet(), "kitchen.diet", "diet", &invalid),
		Allergies:  storedEach(allergies, k.GetAllergies(), "kitchen.allergies", "allergies", &invalid),
		Staples:    storedEach(staples, k.GetStaples(), "kitchen.staples", "staples", &invalid),
	}
	return edit, invalid
}

// stored is the word one wire value is stored as. A value the set doesn't have is refused on field.
func stored[E enum](set map[string]E, value E, field, name string, invalid *handlers.FieldErrors) string {
	word, ok := wordFor(set, value)
	if !ok {
		*invalid = append(*invalid, outsideSet(set, field, name))
	}
	return word
}

// storedEach maps a wire list, refusing each value the set doesn't have on its own index.
func storedEach[E enum](set map[string]E, values []E, field, name string, invalid *handlers.FieldErrors) []string {
	picked := map[string]bool{}
	for i, value := range values {
		word, ok := wordFor(set, value)
		if !ok {
			*invalid = append(*invalid, outsideSet(set, fmt.Sprintf("%s[%d]", field, i), name))
			continue
		}
		picked[word] = true
	}
	out := []string{}
	for _, word := range inOrder(set) {
		if picked[word] {
			out = append(out, word)
		}
	}
	return out
}

// wordFor finds the stored word for a wire value. UNSPECIFIED, and a value from a newer client,
// have none.
func wordFor[E enum](set map[string]E, value E) (string, bool) {
	for word, e := range set {
		if e == value {
			return word, true
		}
	}
	return "", false
}

// outsideSet refuses field for a value its set doesn't have, naming the ones it does.
func outsideSet[E enum](set map[string]E, field, name string) handlers.FieldError {
	return handlers.FieldError{
		Field:   field,
		Message: fmt.Sprintf("%s must be one of %s", name, strings.Join(inOrder(set), ", ")),
	}
}

// inOrder lists a set's words in the wire's order, which is the closed set's order in
// docs/contracts.md.
func inOrder[E enum](set map[string]E) []string {
	words := slices.Collect(maps.Keys(set))
	slices.SortFunc(words, func(a, b string) int { return cmp.Compare(set[a], set[b]) })
	return words
}

// setOf rebuilds a closed set the handlers root keeps to itself, asking the root for the stored
// word of each of the enum's values.
func setOf[E interface {
	enum
	Descriptor() protoreflect.EnumDescriptor
}](word func(E) (string, bool)) map[string]E {
	var zero E
	values := zero.Descriptor().Values()
	set := map[string]E{}
	for i := range values.Len() {
		e := E(values.Get(i).Number())
		w, ok := word(e)
		if ok {
			set[w] = e
		}
	}
	return set
}
