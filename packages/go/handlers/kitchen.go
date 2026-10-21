// Author: Caden Lund
package handlers

import (
	"fmt"

	accountv1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/account/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/repository"
)

// The kitchen's closed sets that more than one service speaks (the account's settings, the
// recipes' equipment and filters), from the word the database stores to account.v1's enum. A word
// missing here fails the rpc with a logged internal error rather than reaching the app as
// unspecified.
var (
	equipmentWire = map[string]accountv1.Equipment{
		repository.EquipmentMicrowave: accountv1.Equipment_EQUIPMENT_MICROWAVE,
		repository.EquipmentStovetop:  accountv1.Equipment_EQUIPMENT_STOVETOP,
		repository.EquipmentOven:      accountv1.Equipment_EQUIPMENT_OVEN,
		repository.EquipmentAirFryer:  accountv1.Equipment_EQUIPMENT_AIR_FRYER,
	}
	dietWire = map[string]accountv1.Diet{
		repository.DietNone:        accountv1.Diet_DIET_NONE,
		repository.DietVegetarian:  accountv1.Diet_DIET_VEGETARIAN,
		repository.DietVegan:       accountv1.Diet_DIET_VEGAN,
		repository.DietPescatarian: accountv1.Diet_DIET_PESCATARIAN,
	}
)

// WireEquipment maps stored equipment to the wire, in order.
func WireEquipment(words []string) ([]accountv1.Equipment, error) {
	out := make([]accountv1.Equipment, 0, len(words))
	for _, word := range words {
		e, ok := equipmentWire[word]
		if !ok {
			return nil, fmt.Errorf("handlers: no wire equipment for %q", word)
		}
		out = append(out, e)
	}
	return out, nil
}

// WireDiet is the wire's enum for a stored diet.
func WireDiet(word string) (accountv1.Diet, error) {
	d, ok := dietWire[word]
	if !ok {
		return accountv1.Diet_DIET_UNSPECIFIED, fmt.Errorf("handlers: no wire diet for %q", word)
	}
	return d, nil
}

// EquipmentWord is the stored word for equipment from the wire, and false for unspecified or a
// value newer than this server.
func EquipmentWord(e accountv1.Equipment) (string, bool) {
	return wordFor(equipmentWire, e)
}

// DietWord is the stored word for a diet from the wire, and false for unspecified or a value
// newer than this server.
func DietWord(d accountv1.Diet) (string, bool) {
	return wordFor(dietWire, d)
}

// wordFor is the stored word that maps to e in set.
func wordFor[E comparable](set map[string]E, e E) (string, bool) {
	for word, candidate := range set {
		if candidate == e {
			return word, true
		}
	}
	return "", false
}
