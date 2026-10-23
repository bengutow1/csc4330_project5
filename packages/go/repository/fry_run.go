// Author: Caden Lund
package repository

import (
	"context"
	"fmt"
	"time"
)

// fry_run.go owns app.fry_run, one row per trip a player logged from Fries mode. A run belongs to
// the account that logged it and goes with it (on delete cascade). The spot is the places
// provider's, kept as the app sent it, with its id and name checked for length. A player counts
// one run per meal window (fry_run_meal_window_key).

// FryRun is a trip to log. MealDay and MealWindow are the campus day and meal window At falls in,
// which the handler works out with points.MealWindow.
type FryRun struct {
	SpotID     string
	SpotName   string
	MealDay    time.Time // the campus day; only its date is stored
	MealWindow string    // morning, midday, evening or late_night
	At         time.Time
}

// LogFryRun stores a trip for accountID and returns its id. When the player already has a run in
// that meal window, it stores nothing and returns that run's id, so tapping again, or logging a
// second spot over the same lunch, cannot pad the Hall of Fries.
func (r *Repository) LogFryRun(ctx context.Context, accountID int64, run FryRun) (int64, error) {
	op := fmt.Sprintf("log fry run account=%d", accountID)
	day := run.MealDay.Format(time.DateOnly)
	var id int64
	err := r.db.QueryRow(ctx, `
		   insert into app.fry_run (account_id, spot_id, spot_name, meal_day, meal_window, created_at)
		   values ($1, $2, $3, $4::date, $5, $6)
		       on conflict on constraint fry_run_meal_window_key do nothing
		returning fry_run_id`,
		accountID, run.SpotID, run.SpotName, day, run.MealWindow, run.At).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !isNoRows(err) {
		return 0, known(op, err)
	}
	// The window already has a run. A second statement reads it: under read committed it sees a
	// run another tap committed a moment ago, which the insert's own snapshot could not.
	err = r.db.QueryRow(ctx, `
		select f.fry_run_id
		  from app.fry_run f
		 where f.account_id = $1
		   and f.meal_day = $2::date
		   and f.meal_window = $3`, accountID, day, run.MealWindow).Scan(&id)
	if err != nil {
		return 0, known(op, err)
	}
	return id, nil
}

// FryRunCount is how many fry runs accountID has logged, all time.
func (r *Repository) FryRunCount(ctx context.Context, accountID int64) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `
		select count(*)
		  from app.fry_run f
		 where f.account_id = $1`, accountID).Scan(&n)
	if err != nil {
		return 0, known(fmt.Sprintf("fry run count account=%d", accountID), err)
	}
	return n, nil
}

// HallOfFriesWeeks is how many finished weeks accountID ended on top of the Hall of Fries: the
// weeks before before, each drawn Monday 00:00 to Monday 00:00 in campus, the zone the board's
// weeks run on. A shared top counts for everyone on it. It reads across accounts, as the board
// does, because topping a week is a comparison.
func (r *Repository) HallOfFriesWeeks(ctx context.Context, accountID int64, before time.Time, campus *time.Location) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `
		  with weekly as (
		select f.account_id, date_trunc('week', f.created_at at time zone $3) as week, count(*) as runs
		  from app.fry_run f
		 where f.created_at < $2
		 group by f.account_id, week
		), ranked as (
		select w.account_id, rank() over (partition by w.week order by w.runs desc) as rank
		  from weekly w
		)
		select count(*)
		  from ranked r
		 where r.account_id = $1
		   and r.rank = 1`, accountID, before, campus.String()).Scan(&n)
	if err != nil {
		return 0, known(fmt.Sprintf("hall of fries weeks account=%d", accountID), err)
	}
	return n, nil
}
