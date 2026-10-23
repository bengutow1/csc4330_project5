// Author: Caden Lund
package repository

import (
	"context"
	"fmt"
	"time"
)

// profile.go reads the profile's plate grid and finds a player for their public card. The plates
// belong to plate.go; this only reads one player's, newest first, a page at a time, with a keyset
// over (created_at, plate_id) so two plates from the same moment keep one order.

// PlateKey is where a page of plates ended: its last plate's time and id.
type PlateKey struct {
	At      time.Time
	PlateID int64
}

// PlateThumb is one plate on the profile's grid.
type PlateThumb struct {
	PlateID     int64
	Photo       []byte // the JPEG as it was checked
	RecipeTitle string
	At          time.Time // when it was plated
}

// ListPlates reads up to limit of accountID's plates, newest first, after after (nil for the first
// page).
func (r *Repository) ListPlates(ctx context.Context, accountID int64, after *PlateKey, limit int) ([]PlateThumb, error) {
	op := fmt.Sprintf("list plates account=%d", accountID)
	// A null key reads the first page.
	var afterAt *time.Time
	var afterID *int64
	if after != nil {
		afterAt, afterID = &after.At, &after.PlateID
	}
	rows, err := r.db.Query(ctx, `
		select p.plate_id, p.photo, rec.title, p.created_at
		  from app.plate p
		  join app.recipe rec on rec.recipe_id = p.recipe_id
		 where p.account_id = $1
		   and ($2::timestamptz is null or (p.created_at, p.plate_id) < ($2::timestamptz, $3::bigint))
		 order by p.created_at desc, p.plate_id desc
		 limit $4`, accountID, afterAt, afterID, limit)
	if err != nil {
		return nil, known(op, err)
	}
	defer rows.Close()
	var out []PlateThumb
	for rows.Next() {
		var p PlateThumb
		err = rows.Scan(&p.PlateID, &p.Photo, &p.RecipeTitle, &p.At)
		if err != nil {
			return nil, known(op, err)
		}
		out = append(out, p)
	}
	err = rows.Err()
	if err != nil {
		return nil, known(op, err)
	}
	return out, nil
}

// Player is what anyone signed in may see of an account: its handle and avatar, and the id its
// card's counts are fenced by. It never holds the kitchen settings.
type Player struct {
	AccountID int64
	Handle    string
	Avatar    string // one emoji
}

// PlayerByHandle finds the player who holds handle, for their public card, reading only the public
// columns. A handle nobody holds is ErrNotFound.
func (r *Repository) PlayerByHandle(ctx context.Context, handle string) (Player, error) {
	var p Player
	err := r.db.QueryRow(ctx, `
		select a.account_id, a.handle, a.avatar
		  from app.account a
		 where a.handle = $1`, handle).Scan(&p.AccountID, &p.Handle, &p.Avatar)
	if err != nil {
		return Player{}, known(fmt.Sprintf("player by handle handle=%s", handle), err)
	}
	return p, nil
}
