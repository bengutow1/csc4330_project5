// Author: LAPTOP-BPMQJP10\andyk
package repository

import (
	"context"
	"fmt"
	"time"
)

// LockCombo serializes writes against membership changes, then re-reads the fence after any
// wait. Call only inside InTx, so the lock lasts through the dependent writes.
func (r *Repository) LockCombo(ctx context.Context, accountID, comboID int64) (Combo, error) {
	var id int64
	err := r.db.QueryRow(ctx, `
		select c.combo_id from app.combo c
		join app.combo_member m on m.combo_id=c.combo_id and m.account_id=$1
		where c.combo_id=$2 for update of c`, accountID, comboID).Scan(&id)
	if err != nil {
		return Combo{}, known(fmt.Sprintf("lock combo account=%d combo=%d", accountID, comboID), err)
	}
	return r.ComboByID(ctx, accountID, comboID)
}

// ComboAccounts reads both members' current kitchen settings in one bounded read.
func (r *Repository) ComboAccounts(ctx context.Context, comboID int64) ([]Account, error) {
	rows, err := r.db.Query(ctx, `
		select a.account_id,a.handle,a.avatar,a.roast_level,a.equipment,a.diet,a.allergies,a.staples
		from app.combo_member m join app.account a on a.account_id=m.account_id
		where m.combo_id=$1 order by m.slot`, comboID)
	if err != nil {
		return nil, known(fmt.Sprintf("combo kitchens combo=%d", comboID), err)
	}
	defer rows.Close()
	var out []Account
	for rows.Next() {
		var a Account
		err = rows.Scan(&a.AccountID, &a.Handle, &a.Avatar, &a.RoastLevel, &a.Equipment, &a.Diet, &a.Allergies, &a.Staples)
		if err != nil {
			return nil, known("combo kitchen row", err)
		}
		out = append(out, a)
	}
	err = rows.Err()
	if err != nil {
		return nil, known("combo kitchen rows", err)
	}
	return out, nil
}

// HasComboRecipes includes empty sets and keys both players' copies by both list versions.
func (r *Repository) HasComboRecipes(ctx context.Context, c Combo) (bool, error) {
	if len(c.Members) != 2 || c.Members[0].Scan == nil || c.Members[1].Scan == nil {
		return false, nil
	}
	a, b := c.Members[0].Scan, c.Members[1].Scan
	var n int
	err := r.db.QueryRow(ctx, `
		select count(*) from app.recipe_set rs join app.scan s on s.scan_id=rs.scan_id
		join app.combo_member m on m.combo_id=s.combo_id and m.account_id=s.account_id
		where s.combo_id=$1 and (rs.scan_id,rs.items_version,rs.partner_scan_id,rs.partner_items_version)
		in (($2,$3,$4,$5),($4,$5,$2,$3))`, c.ComboID, a.ScanID, a.ItemsVersion, b.ScanID, b.ItemsVersion).Scan(&n)
	if err != nil {
		return false, known(fmt.Sprintf("has combo recipes combo=%d", c.ComboID), err)
	}
	return n == 2, nil
}

// CreateComboRecipes stores both players' recipe copies together, so a second phone reads the
// same dishes without paying for another model call. A changed snapshot stores nothing.
func (r *Repository) CreateComboRecipes(ctx context.Context, accountID int64, snapshot Combo, score int, set []NewRecipe, now time.Time) error {
	return r.InTx(ctx, func(tx *Repository) error {
		current, err := tx.LockCombo(ctx, accountID, snapshot.ComboID)
		if err != nil {
			return err
		}
		if current.State == ComboEnded {
			return ErrComboEnded
		}
		if !current.ExpiresAt.After(now) {
			return ErrComboExpired
		}
		if len(current.Members) != 2 || len(snapshot.Members) != 2 {
			return ErrNotFound
		}
		for i, m := range current.Members {
			old := snapshot.Members[i]
			if m.AccountID != old.AccountID || m.Scan == nil || old.Scan == nil || m.Scan.ScanID != old.Scan.ScanID || m.Scan.ItemsVersion != old.Scan.ItemsVersion {
				return ErrNotFound
			}
		}
		for i, m := range current.Members {
			buddy := current.Members[1-i]
			_, err = tx.createRecipes(ctx, m.AccountID, m.Scan.ScanID, m.Scan.ItemsVersion, buddy.Scan.ScanID, buddy.Scan.ItemsVersion, score, set)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// ListComboRecipes applies filters and every current member's allergies in SQL.
func (r *Repository) ListComboRecipes(ctx context.Context, accountID int64, c Combo, f RecipeFilter) ([]Recipe, error) {
	if len(c.Members) != 2 {
		return nil, ErrNotFound
	}
	index := -1
	for i, m := range c.Members {
		if m.AccountID == accountID {
			index = i
		}
	}
	if index < 0 {
		return nil, ErrNotFound
	}
	own, other := c.Members[index].Scan, c.Members[1-index].Scan
	if own == nil || other == nil {
		return nil, ErrNotFound
	}
	equipment := f.Equipment
	if equipment == nil {
		equipment = []string{}
	}
	rows, err := r.db.Query(ctx, `
		select r.recipe_id,r.title,r.minutes,r.difficulty,r.equipment,r.uses,r.missing,
		       r.steps,r.step_timers,rs.combo_score,s.scan_id,r.items_version,
		       rs.partner_scan_id,rs.partner_items_version,r.position,s.combo_id,r.ingredient_owners,r.only_together
		from app.recipe r join app.recipe_set rs on rs.recipe_set_id=r.recipe_set_id
		join app.scan s on s.scan_id=r.scan_id
		join app.combo_member mine on mine.combo_id=s.combo_id and mine.account_id=s.account_id
		where s.account_id=$1 and s.combo_id=$2 and rs.scan_id=$3 and rs.items_version=$4
		and rs.partner_scan_id=$5 and rs.partner_items_version=$6
		and not exists(select 1 from app.combo_member m join app.account a on a.account_id=m.account_id
		               where m.combo_id=$2 and (r.allergens && a.allergies or
		               (a.diet<>'none' and not a.diet=any(r.diets))))
		and ($7::int=0 or r.minutes<=$7) and (cardinality($8::text[])=0 or r.equipment<@$8::text[])
		and ($9::text in ('','none') or $9=any(r.diets)) order by r.position`, accountID, c.ComboID, own.ScanID, own.ItemsVersion, other.ScanID, other.ItemsVersion, f.MaxMinutes, equipment, f.Diet)
	if err != nil {
		return nil, known(fmt.Sprintf("list combo recipes combo=%d account=%d", c.ComboID, accountID), err)
	}
	defer rows.Close()
	var out []Recipe
	for rows.Next() {
		var rec Recipe
		err = rows.Scan(&rec.RecipeID, &rec.Title, &rec.Minutes, &rec.Difficulty, &rec.Equipment, &rec.Uses, &rec.Missing, &rec.Steps, &rec.StepTimers, &rec.ScanScore, &rec.ScanID, &rec.ItemsVersion, &rec.PartnerScanID, &rec.PartnerItemsVersion, &rec.Position, &rec.ComboID, &rec.IngredientOwners, &rec.OnlyTogether)
		if err != nil {
			return nil, known("combo recipe row", err)
		}
		out = append(out, rec)
	}
	err = rows.Err()
	if err != nil {
		return nil, known("combo recipe rows", err)
	}
	return out, nil
}

// ComboPartnerRecipe finds the matching dish in the other member's own cached set.
func (r *Repository) ComboPartnerRecipe(ctx context.Context, accountID, recipeID int64) (int64, int64, error) {
	var owner, id int64
	err := r.db.QueryRow(ctx, `
		select partner.account_id,other.recipe_id
		from app.recipe mine join app.recipe_set rs on rs.recipe_set_id=mine.recipe_set_id
		join app.scan s on s.scan_id=mine.scan_id
		join app.scan partner on partner.scan_id=rs.partner_scan_id and partner.combo_id=s.combo_id
		join app.combo_member m on m.combo_id=s.combo_id and m.account_id=s.account_id
		join app.combo_member buddy on buddy.combo_id=s.combo_id and buddy.account_id=partner.account_id
		join app.recipe_set theirs on theirs.scan_id=partner.scan_id and theirs.items_version=rs.partner_items_version
		 and theirs.partner_scan_id=s.scan_id and theirs.partner_items_version=rs.items_version
		join app.recipe other on other.recipe_set_id=theirs.recipe_set_id and other.position=mine.position
		where s.account_id=$1 and mine.recipe_id=$2`, accountID, recipeID).Scan(&owner, &id)
	if err != nil {
		return 0, 0, known(fmt.Sprintf("combo partner recipe account=%d recipe=%d", accountID, recipeID), err)
	}
	return owner, id, nil
}

// HasCookCredit reads one account's meal-window credit after the caller locked that account.
func (r *Repository) HasCookCredit(ctx context.Context, accountID int64, window string) (bool, error) {
	var found bool
	err := r.db.QueryRow(ctx, `select exists(select 1 from app.point_event where account_id=$1 and meal_window=$2 and reason='cook')`, accountID, window).Scan(&found)
	if err != nil {
		return false, known(fmt.Sprintf("cook credit account=%d", accountID), err)
	}
	return found, nil
}

// CreateComboPlateCopy gives the other credited player their own photo record without permitting
// a second independent submission of that photo. All input identity comes from membership.
func (r *Repository) CreateComboPlateCopy(ctx context.Context, accountID, sourcePlateID, recipeID int64) (int64, error) {
	var id int64
	err := r.db.QueryRow(ctx, `
		insert into app.plate(account_id,recipe_id,photo,photo_sha256,created_at,combo_plate_id)
		select target.account_id,rec.recipe_id,p.photo,p.photo_sha256,p.created_at,p.plate_id
		from app.plate p join app.recipe source on source.recipe_id=p.recipe_id
		join app.scan original on original.scan_id=source.scan_id
		join app.recipe rec on rec.recipe_id=$3 join app.scan target on target.scan_id=rec.scan_id
		join app.combo_member mine on mine.combo_id=target.combo_id and mine.account_id=target.account_id
		join app.combo_member buddy on buddy.combo_id=mine.combo_id and buddy.account_id=p.account_id
		where target.account_id=$1 and p.plate_id=$2 and original.combo_id=target.combo_id
		and p.combo_plate_id is null returning plate_id`, accountID, sourcePlateID, recipeID).Scan(&id)
	if err != nil {
		return 0, known(fmt.Sprintf("copy combo plate account=%d plate=%d", accountID, sourcePlateID), err)
	}
	return id, nil
}
