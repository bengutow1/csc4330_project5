// Author: Caden Lund
package repository

import (
	"context"
	"fmt"
	"time"
)

// board.go reads the boards. A board ranks every player at once, so these queries read across
// accounts by design: a row is a handle, an avatar and a count, and the Fridge of the Week is one
// scan's score and roast, which every player may see. Ranks come from a window function over the
// whole board, so ties share a rank and a page never renumbers them. A page resumes after its last
// row with a keyset over (value, handle).

// BoardRow is one player's place on a board.
type BoardRow struct {
	Rank      int // 1 for the most; ties share it, and the next rank skips past them
	AccountID int64
	Handle    string
	Avatar    string
	Value     int64 // fry runs, or points, over the range
}

// BoardKey is where a page of a board ended: its last row's value and handle.
type BoardKey struct {
	Value  int64
	Handle string
}

// HallOfFries ranks players by their fry runs since since (the zero time counts them all) and
// returns up to limit rows, best first, after after (nil for the first page). Only players with a
// run in the range are ranked.
func (r *Repository) HallOfFries(ctx context.Context, since time.Time, after *BoardKey, limit int) ([]BoardRow, error) {
	// A null key reads the first page. Value runs high to low and handle low to high, so the row
	// comparison negates the value to put both in one direction.
	var afterValue *int64
	var afterHandle *string
	if after != nil {
		afterValue, afterHandle = &after.Value, &after.Handle
	}
	rows, err := r.db.Query(ctx, `
		  with ranked as (
		select f.account_id, count(*) as value, rank() over (order by count(*) desc) as rank
		  from app.fry_run f
		 where f.created_at >= $1
		 group by f.account_id
		)
		select r.rank, r.account_id, a.handle, a.avatar, r.value
		  from ranked r
		  join app.account a on a.account_id = r.account_id
		 where $2::bigint is null or (-r.value, a.handle) > (-$2::bigint, $3::text)
		 order by r.value desc, a.handle
		 limit $4`, since, afterValue, afterHandle, limit)
	if err != nil {
		return nil, known("hall of fries", err)
	}
	defer rows.Close()
	var out []BoardRow
	for rows.Next() {
		var b BoardRow
		err = rows.Scan(&b.Rank, &b.AccountID, &b.Handle, &b.Avatar, &b.Value)
		if err != nil {
			return nil, known("hall of fries", err)
		}
		out = append(out, b)
	}
	err = rows.Err()
	if err != nil {
		return nil, known("hall of fries", err)
	}
	return out, nil
}

// HallOfFriesRow is accountID's own row on the Hall of Fries since since, ranked on the whole
// board; false when they have no run in the range.
func (r *Repository) HallOfFriesRow(ctx context.Context, accountID int64, since time.Time) (BoardRow, bool, error) {
	var b BoardRow
	err := r.db.QueryRow(ctx, `
		  with ranked as (
		select f.account_id, count(*) as value, rank() over (order by count(*) desc) as rank
		  from app.fry_run f
		 where f.created_at >= $2
		 group by f.account_id
		)
		select r.rank, r.account_id, a.handle, a.avatar, r.value
		  from ranked r
		  join app.account a on a.account_id = r.account_id
		 where r.account_id = $1`, accountID, since).Scan(&b.Rank, &b.AccountID, &b.Handle, &b.Avatar, &b.Value)
	if isNoRows(err) {
		return BoardRow{}, false, nil
	}
	if err != nil {
		return BoardRow{}, false, known(fmt.Sprintf("hall of fries row account=%d", accountID), err)
	}
	return b, true, nil
}

// FridgeRow is the scan the Fridge of the Week shows, and whose it was.
type FridgeRow struct {
	Handle string
	Avatar string
	Score  int // 0 to 100
	Roast  string
}

// FridgeOfTheWeek is the best scan made since since: the highest score, and the first made of the
// scans tied on it, so matching a score later never takes the fridge. False when nobody has
// scanned since since.
func (r *Repository) FridgeOfTheWeek(ctx context.Context, since time.Time) (FridgeRow, bool, error) {
	var f FridgeRow
	err := r.db.QueryRow(ctx, `
		select a.handle, a.avatar, s.score, s.roast
		  from app.scan s
		  join app.account a on a.account_id = s.account_id
		 where s.created_at >= $1
		 order by s.score desc, s.created_at, s.scan_id
		 limit 1`, since).Scan(&f.Handle, &f.Avatar, &f.Score, &f.Roast)
	if isNoRows(err) {
		return FridgeRow{}, false, nil
	}
	if err != nil {
		return FridgeRow{}, false, known("fridge of the week", err)
	}
	return f, true, nil
}
