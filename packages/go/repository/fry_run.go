// Author: Caden Lund
package repository

import (
	"context"
	"fmt"
)

// fry_run.go owns app.fry_run, one row per trip a player logged from Fries mode. A run belongs to
// the account that logged it and goes with it (on delete cascade). The spot is the places
// provider's, kept as the app sent it, with its id and name checked for length.

// CreateFryRun logs a trip to a spot for accountID and returns the run's id.
func (r *Repository) CreateFryRun(ctx context.Context, accountID int64, spotID, spotName string) (int64, error) {
	var id int64
	err := r.db.QueryRow(ctx, `
		   insert into app.fry_run (account_id, spot_id, spot_name)
		   values ($1, $2, $3)
		returning fry_run_id`, accountID, spotID, spotName).Scan(&id)
	if err != nil {
		return 0, known(fmt.Sprintf("create fry run account=%d", accountID), err)
	}
	return id, nil
}
