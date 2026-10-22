// Author: Caden Lund
package repository

import (
	"context"
	"fmt"
	"time"
)

// plate.go owns app.plate and app.point_event: the plates the model checked, and the points
// ledger, one row per award. A player's points are a sum over their rows, never a column. A photo
// counts once for anyone (plate_photo_sha256_key), and a player earns one cook credit per meal
// window (point_event_account_id_meal_window_key). Rows are written once and never updated.

// The award reasons, mirroring app.point_event's check and spelled as packages/go/points spells
// them.
const (
	ReasonCook           = "cook"
	ReasonScrappyBonus   = "scrappy_bonus"
	ReasonAgainstAllOdds = "against_all_odds"
	ReasonStreak         = "streak"
	ReasonComboCook      = "combo_cook"
)

// NewPlate is a checked plate to store.
type NewPlate struct {
	RecipeID    int64
	Photo       []byte // the JPEG, kept for the profile
	PhotoSHA256 []byte // the photo's hash, which a plate seen before collides on
	At          time.Time
}

// Award is one line of a plate's receipt, and one row of the ledger.
type Award struct {
	Reason string // a Reason const
	Points int
}

// PlayerStats is what the badges read from a player's history. Every field only grows.
type PlayerStats struct {
	Scans                 int
	BestScore             int
	Cooks                 int // checked plates, one cook credit each
	CooksUnder35          int // plates of recipes from a scan that scored under 35
	CooksWithThreeOrFewer int // plates of recipes that take three things or fewer
	ComboCooks            int
	LongestStreak         int // the most meal-window days in a row with a cook credit
	LateNightPlates       int // plates between midnight and 4 am, campus time
	MicrowaveOnlyCooks    int // plates of recipes that need a microwave and nothing else
	BiggestWeeklyRise     int // the most a scan's score rose over an earlier one within 7 days
}

// LockAccount holds the account's row until the transaction ends, so the player's plates take
// turns: the day's first plate and its streak are read before the plate is written. It must run
// inside InTx. A missing account is ErrNotFound.
func (r *Repository) LockAccount(ctx context.Context, accountID int64) error {
	var id int64
	err := r.db.QueryRow(ctx, `
		select a.account_id
		  from app.account a
		 where a.account_id = $1
		   for update`, accountID).Scan(&id)
	if err != nil {
		return known(fmt.Sprintf("lock account=%d", accountID), err)
	}
	return nil
}

// CreatePlate stores a checked plate on one of the account's recipes and returns its id. A photo
// plated before, by anyone, is ErrPlateSeen; a recipe that is not the account's is ErrNotFound.
func (r *Repository) CreatePlate(ctx context.Context, accountID int64, p NewPlate) (int64, error) {
	op := fmt.Sprintf("create plate account=%d recipe=%d", accountID, p.RecipeID)
	// The join through the recipe's scan is the fence: it cannot plate another player's recipe.
	var plateID int64
	err := r.db.QueryRow(ctx, `
		   insert into app.plate (account_id, recipe_id, photo, photo_sha256, created_at)
		   select s.account_id, rec.recipe_id, $3, $4, $5
		     from app.recipe rec
		     join app.scan s on s.scan_id = rec.scan_id
		    where s.account_id = $1
		      and rec.recipe_id = $2
		returning plate_id`, accountID, p.RecipeID, p.Photo, p.PhotoSHA256, p.At).Scan(&plateID)
	if err != nil {
		return 0, known(op, err)
	}
	return plateID, nil
}

// AwardPoints writes a plate's awards to the ledger in one statement, all in mealWindow (its
// campus day and window, as "2026-10-21 evening"). A cook credit the player already holds for
// that window is ErrMealWindowCooked, and a plate that is not the account's is ErrNotFound; either
// way nothing is written.
func (r *Repository) AwardPoints(
	ctx context.Context, accountID, plateID int64, mealWindow string, at time.Time, awards []Award,
) error {
	reasons := make([]string, len(awards))
	points := make([]int, len(awards))
	for i, a := range awards {
		reasons[i], points[i] = a.Reason, a.Points
	}
	// The join to the plate is the fence: it cannot award points on another player's plate.
	op := fmt.Sprintf("award points account=%d plate=%d window=%s", accountID, plateID, mealWindow)
	tag, err := r.db.Exec(ctx, `
		insert into app.point_event (account_id, plate_id, reason, points, meal_window, created_at)
		select p.account_id, p.plate_id, a.reason, a.points, $3, $4
		  from app.plate p
		 cross join unnest($5::text[], $6::int[]) as a (reason, points)
		 where p.account_id = $1
		   and p.plate_id = $2`, accountID, plateID, mealWindow, at, reasons, points)
	if err != nil {
		return known(op, err)
	}
	if len(awards) > 0 && tag.RowsAffected() == 0 {
		return fmt.Errorf("%s: %w", op, ErrNotFound)
	}
	return nil
}

// PointsTotal is the account's all-time points: the sum of its ledger.
func (r *Repository) PointsTotal(ctx context.Context, accountID int64) (int, error) {
	var total int
	err := r.db.QueryRow(ctx, `
		select coalesce(sum(e.points), 0)
		  from app.point_event e
		 where e.account_id = $1`, accountID).Scan(&total)
	if err != nil {
		return 0, known(fmt.Sprintf("points total account=%d", accountID), err)
	}
	return total, nil
}

// CookDays is every campus day the account holds a cook credit for, oldest first, as
// "2026-10-21": the day of each credit's meal window, so a plate at 00:30 counts for the night
// before.
func (r *Repository) CookDays(ctx context.Context, accountID int64) ([]string, error) {
	op := fmt.Sprintf("cook days account=%d", accountID)
	rows, err := r.db.Query(ctx, `
		select distinct split_part(e.meal_window, ' ', 1) as day
		  from app.point_event e
		 where e.account_id = $1
		   and e.reason = 'cook'
		 order by day`, accountID)
	if err != nil {
		return nil, known(op, err)
	}
	defer rows.Close()
	var days []string
	for rows.Next() {
		var day string
		err = rows.Scan(&day)
		if err != nil {
			return nil, known(op, err)
		}
		days = append(days, day)
	}
	err = rows.Err()
	if err != nil {
		return nil, known(op, err)
	}
	return days, nil
}

// PlayerStats counts what the badges read from the account's history, in one statement.
func (r *Repository) PlayerStats(ctx context.Context, accountID int64) (PlayerStats, error) {
	var s PlayerStats
	err := r.db.QueryRow(ctx, `
		with plated as (
		     select p.created_at, sc.score, rec.equipment,
		            cardinality(rec.uses) + cardinality(rec.missing) as takes
		       from app.plate p
		       join app.recipe rec on rec.recipe_id = p.recipe_id
		       join app.scan sc on sc.scan_id = rec.scan_id
		      where p.account_id = $1
		),
		cook_days as (
		     select distinct split_part(e.meal_window, ' ', 1)::date as day
		       from app.point_event e
		      where e.account_id = $1
		        and e.reason = 'cook'
		),
		runs as (
		     -- Consecutive days share day minus their rank, so each run is one group.
		     select count(*) as days
		       from (select day, day - (row_number() over (order by day))::int as run
		               from cook_days) numbered
		      group by run
		)
		select (select count(*) from app.scan s where s.account_id = $1),
		       (select coalesce(max(s.score), 0) from app.scan s where s.account_id = $1),
		       (select count(*) from app.point_event e where e.account_id = $1 and e.reason = 'cook'),
		       (select count(*) from plated where score < 35),
		       (select count(*) from plated where takes <= 3),
		       (select count(*) from app.point_event e where e.account_id = $1 and e.reason = 'combo_cook'),
		       (select coalesce(max(days), 0) from runs),
		       (select count(*) from plated
		         where extract(hour from created_at at time zone 'America/Chicago') < 4),
		       (select count(*) from plated where equipment = array['microwave']),
		       (select greatest(coalesce(max(later.score - earlier.score), 0), 0)
		          from app.scan earlier
		          join app.scan later on later.account_id = earlier.account_id
		                             and later.created_at > earlier.created_at
		                             and later.created_at <= earlier.created_at + interval '7 days'
		         where earlier.account_id = $1)`, accountID).Scan(
		&s.Scans, &s.BestScore, &s.Cooks, &s.CooksUnder35, &s.CooksWithThreeOrFewer, &s.ComboCooks,
		&s.LongestStreak, &s.LateNightPlates, &s.MicrowaveOnlyCooks, &s.BiggestWeeklyRise)
	if err != nil {
		return PlayerStats{}, known(fmt.Sprintf("player stats account=%d", accountID), err)
	}
	return s, nil
}
