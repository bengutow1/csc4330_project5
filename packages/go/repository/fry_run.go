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
