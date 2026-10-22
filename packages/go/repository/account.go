// Author: Caden Lund
package repository

import (
	"context"
	"fmt"
)

// account.go owns app.account, one row per device account. The handle is unique
// (account_handle_key), and every kitchen setting is checked against its closed set, spelled by
// the consts below.

// The roast levels. Medium is the default.
const (
	RoastMild        = "mild"
	RoastMedium      = "medium"
	RoastExtraCrispy = "extra_crispy"
)

// The equipment a kitchen can have. A new account has none until onboarding asks.
const (
	EquipmentMicrowave = "microwave"
	EquipmentStovetop  = "stovetop"
	EquipmentOven      = "oven"
	EquipmentAirFryer  = "air_fryer"
)

// The diets. None is the default and means no restriction.
const (
	DietNone        = "none"
	DietVegetarian  = "vegetarian"
	DietVegan       = "vegan"
	DietPescatarian = "pescatarian"
)

// The allergies, each a hard filter on recipes.
const (
	AllergyPeanuts   = "peanuts"
	AllergyTreeNuts  = "tree_nuts"
	AllergyMilk      = "milk"
	AllergyEggs      = "eggs"
	AllergyWheat     = "wheat"
	AllergySoy       = "soy"
	AllergyFish      = "fish"
	AllergyShellfish = "shellfish"
	AllergySesame    = "sesame"
)

// The staples. A new account has every one.
const (
	StapleSalt   = "salt"
	StaplePepper = "pepper"
	StapleOil    = "oil"
	StapleButter = "butter"
	StapleRice   = "rice"
	StaplePasta  = "pasta"
	StapleFlour  = "flour"
	StapleSugar  = "sugar"
	StapleSpices = "spices"
)

// Account is one device account: the player's public name and their kitchen settings.
type Account struct {
	AccountID  int64
	Handle     string
	Avatar     string   // one emoji
	RoastLevel string   // a Roast const
	Equipment  []string // Equipment consts; empty until the player picks
	Diet       string   // a Diet const
	Allergies  []string // Allergy consts; empty means none
	Staples    []string // Staple consts the player always has on hand
}

// CreateAccount makes a device account with the default kitchen settings and returns it. A taken
// handle is ErrDuplicate: the unique constraint decides, so two players racing for one handle
// cannot both get it.
func (r *Repository) CreateAccount(ctx context.Context, handle, avatar string) (Account, error) {
	var a Account
	err := r.db.QueryRow(ctx, `
		   insert into app.account (handle, avatar)
		   values ($1, $2)
		returning account_id, handle, avatar, roast_level, equipment, diet, allergies, staples`,
		handle, avatar).Scan(
		&a.AccountID, &a.Handle, &a.Avatar, &a.RoastLevel, &a.Equipment, &a.Diet, &a.Allergies,
		&a.Staples)
	if err != nil {
		return Account{}, known("create account", err)
	}
	return a, nil
}

// AccountByID reads one account. The id comes from the caller's own session, so this is a player
// reading their own row. A missing row is ErrNotFound.
func (r *Repository) AccountByID(ctx context.Context, accountID int64) (Account, error) {
	var a Account
	err := r.db.QueryRow(ctx, `
		select a.account_id, a.handle, a.avatar, a.roast_level, a.equipment, a.diet, a.allergies,
		       a.staples
		  from app.account a
		 where a.account_id = $1`, accountID).Scan(
		&a.AccountID, &a.Handle, &a.Avatar, &a.RoastLevel, &a.Equipment, &a.Diet, &a.Allergies,
		&a.Staples)
	if err != nil {
		return Account{}, known(fmt.Sprintf("account by id account=%d", accountID), err)
	}
	return a, nil
}

// DeleteAccount deletes the caller's account and says whether there was one to delete.
func (r *Repository) DeleteAccount(ctx context.Context, accountID int64) (bool, error) {
	tag, err := r.db.Exec(ctx, `
		delete from app.account a
		 where a.account_id = $1`, accountID)
	if err != nil {
		return false, known(fmt.Sprintf("delete account account=%d", accountID), err)
	}
	return tag.RowsAffected() > 0, nil
}

// HandleTaken says whether an account already holds handle, so onboarding can say so while the
// player types. CreateAccount never relies on it: between this answer and the insert, someone
// else can take the handle.
func (r *Repository) HandleTaken(ctx context.Context, handle string) (bool, error) {
	var taken bool
	err := r.db.QueryRow(ctx, `
		select exists (
		       select 1
		         from app.account a
		        where a.handle = $1)`, handle).Scan(&taken)
	if err != nil {
		return false, known("handle taken", err)
	}
	return taken, nil
}
