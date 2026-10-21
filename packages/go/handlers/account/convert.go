// Author: Caden Lund
package account

import (
	"fmt"

	accountv1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/account/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/repository"
)

// The closed sets, from the words the database stores to the wire's enums. A word missing here
// fails the rpc with a logged internal error rather than reaching the app as unspecified.
var (
	roastLevels = map[string]accountv1.RoastLevel{
		repository.RoastMild:        accountv1.RoastLevel_ROAST_LEVEL_MILD,
		repository.RoastMedium:      accountv1.RoastLevel_ROAST_LEVEL_MEDIUM,
		repository.RoastExtraCrispy: accountv1.RoastLevel_ROAST_LEVEL_EXTRA_CRISPY,
	}
	equipment = map[string]accountv1.Equipment{
		repository.EquipmentMicrowave: accountv1.Equipment_EQUIPMENT_MICROWAVE,
		repository.EquipmentStovetop:  accountv1.Equipment_EQUIPMENT_STOVETOP,
		repository.EquipmentOven:      accountv1.Equipment_EQUIPMENT_OVEN,
		repository.EquipmentAirFryer:  accountv1.Equipment_EQUIPMENT_AIR_FRYER,
	}
	diets = map[string]accountv1.Diet{
		repository.DietNone:        accountv1.Diet_DIET_NONE,
		repository.DietVegetarian:  accountv1.Diet_DIET_VEGETARIAN,
		repository.DietVegan:       accountv1.Diet_DIET_VEGAN,
		repository.DietPescatarian: accountv1.Diet_DIET_PESCATARIAN,
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

// toProtoAccount maps a stored account to the wire.
func toProtoAccount(a repository.Account) (*accountv1.Account, error) {
	roastLevel, err := wire(roastLevels, a.RoastLevel, "roast level")
	if err != nil {
		return nil, err
	}
	diet, err := wire(diets, a.Diet, "diet")
	if err != nil {
		return nil, err
	}
	kitchenEquipment, err := wireEach(equipment, a.Equipment, "equipment")
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
