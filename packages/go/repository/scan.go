// Author: Caden Lund
package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// scan.go owns app.scan and app.scan_item: a verdict, and the ingredient list it was judged on.
// The score is set once, when the scan is made. Editing the list replaces its items and bumps
// items_version, and never touches the score. Every query is fenced by the owner's account id.

// The categories an ingredient falls in, mirroring app.scan_item's check.
const (
	CategoryProtein   = "protein"
	CategoryVegetable = "vegetable"
	CategoryFruit     = "fruit"
	CategoryGrain     = "grain"
	CategoryDairy     = "dairy"
	CategoryCondiment = "condiment"
	CategoryDrink     = "drink"
	CategorySnack     = "snack"
	CategoryLeftover  = "leftover"
	CategoryOther     = "other"
)

// Scan is one verdict: the score, the roast and the list it was judged on. It has no tier,
// because the tier comes from the score on every read.
type Scan struct {
	ScanID       int64
	Score        int // 0 to 100, set when the scan was made
	Roast        string
	ItemsVersion int // 1 for the model's list, and one more for each edit
	CreatedAt    time.Time
	Items        []ScanItem // in display order; empty when the scan found no food
}

// ScanItem is one food on a scan's list.
type ScanItem struct {
	Name     string
	Category string // a Category const
	Quantity string // free text, "" when unknown
}

// CreateScan stores a verdict and its list as one act, and returns the scan as stored.
func (r *Repository) CreateScan(
	ctx context.Context, accountID int64, score int, roast string, items []ScanItem,
) (Scan, error) {
	var s Scan
	err := r.InTx(ctx, func(tx *Repository) error {
		var scanID int64
		err := tx.db.QueryRow(ctx, `
			   insert into app.scan (account_id, score, roast)
			   values ($1, $2, $3)
			returning scan_id`, accountID, score, roast).Scan(&scanID)
		if err != nil {
			return known(fmt.Sprintf("create scan account=%d", accountID), err)
		}
		err = tx.insertItems(ctx, accountID, scanID, items)
		if err != nil {
			return err
		}
		s, err = tx.ScanByID(ctx, accountID, scanID)
		return err
	})
	if err != nil {
		return Scan{}, err
	}
	return s, nil
}

// ScanByID reads one of the account's scans with its list in order. A scan that is another
// player's, or that does not exist, is ErrNotFound.
func (r *Repository) ScanByID(ctx context.Context, accountID, scanID int64) (Scan, error) {
	op := fmt.Sprintf("scan by id account=%d scan=%d", accountID, scanID)
	// One query, so the scan and its list come from the same moment.
	rows, err := r.db.Query(ctx, `
		select s.scan_id, s.score, s.roast, s.items_version, s.created_at,
		       i.name, i.category, i.quantity
		  from app.scan s
		  left join app.scan_item i on i.scan_id = s.scan_id
		 where s.account_id = $1
		   and s.scan_id = $2
		 order by i.position`, accountID, scanID)
	if err != nil {
		return Scan{}, known(op, err)
	}
	defer rows.Close()

	var s Scan
	found := false
	for rows.Next() {
		// Null on the one row of a scan with no items.
		var name, category, quantity *string
		err = rows.Scan(&s.ScanID, &s.Score, &s.Roast, &s.ItemsVersion, &s.CreatedAt,
			&name, &category, &quantity)
		if err != nil {
			return Scan{}, known(op, err)
		}
		found = true
		if name != nil {
			s.Items = append(s.Items, ScanItem{Name: *name, Category: *category, Quantity: *quantity})
		}
	}
	err = rows.Err()
	if err != nil {
		return Scan{}, known(op, err)
	}
	if !found {
		return Scan{}, known(op, pgx.ErrNoRows)
	}
	return s, nil
}

// LastScan reads the account's newest scan. found is false before their first scan, which is an
// answer, not an error.
func (r *Repository) LastScan(ctx context.Context, accountID int64) (Scan, bool, error) {
	var scanID int64
	err := r.db.QueryRow(ctx, `
		select s.scan_id
		  from app.scan s
		 where s.account_id = $1
		 order by s.created_at desc, s.scan_id desc
		 limit 1`, accountID).Scan(&scanID)
	if isNoRows(err) {
		return Scan{}, false, nil
	}
	if err != nil {
		return Scan{}, false, known(fmt.Sprintf("last scan account=%d", accountID), err)
	}
	s, err := r.ScanByID(ctx, accountID, scanID)
	if err != nil {
		return Scan{}, false, err
	}
	return s, true, nil
}

// ReplaceItems swaps the list on one of the account's scans for items, in order, and bumps
// items_version so the recipes regenerate. The score stays as it was. False means the account
// has no such scan, and nothing changed.
func (r *Repository) ReplaceItems(
	ctx context.Context, accountID, scanID int64, items []ScanItem,
) (bool, error) {
	op := fmt.Sprintf("replace scan items account=%d scan=%d", accountID, scanID)
	changed := false
	err := r.InTx(ctx, func(tx *Repository) error {
		// 1. The bump, which also locks the scan, so two edits to it take turns
		tag, err := tx.db.Exec(ctx, `
			update app.scan s
			   set items_version = s.items_version + 1
			 where s.account_id = $1
			   and s.scan_id = $2`, accountID, scanID)
		if err != nil {
			return known(op, err)
		}
		if tag.RowsAffected() == 0 {
			return nil
		}

		// 2. Out with the old list, in with the new
		_, err = tx.db.Exec(ctx, `
			delete from app.scan_item i
			 using app.scan s
			 where s.scan_id = i.scan_id
			   and s.account_id = $1
			   and i.scan_id = $2`, accountID, scanID)
		if err != nil {
			return known(op, err)
		}
		err = tx.insertItems(ctx, accountID, scanID, items)
		if err != nil {
			return err
		}
		changed = true
		return nil
	})
	if err != nil {
		return false, err
	}
	return changed, nil
}

// insertItems adds items to one of the account's scans, numbered from 1 in the order given, in
// one statement. The join to app.scan is the fence: it cannot add to another player's scan.
func (r *Repository) insertItems(
	ctx context.Context, accountID, scanID int64, items []ScanItem,
) error {
	names := make([]string, len(items))
	categories := make([]string, len(items))
	quantities := make([]string, len(items))
	for i, item := range items {
		names[i], categories[i], quantities[i] = item.Name, item.Category, item.Quantity
	}
	_, err := r.db.Exec(ctx, `
		insert into app.scan_item (scan_id, name, category, quantity, position)
		select s.scan_id, i.name, i.category, i.quantity, i.position
		  from app.scan s
		 cross join unnest($3::text[], $4::text[], $5::text[])
		       with ordinality as i (name, category, quantity, position)
		 where s.account_id = $1
		   and s.scan_id = $2`, accountID, scanID, names, categories, quantities)
	if err != nil {
		return known(fmt.Sprintf("insert scan items account=%d scan=%d", accountID, scanID), err)
	}
	return nil
}
