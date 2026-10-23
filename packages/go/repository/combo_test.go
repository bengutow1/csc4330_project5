// Author: LAPTOP-BPMQJP10\andyk
package repository_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/bengutow1/csc4330_project5/packages/go/repository"
)

// TestARemovedGuestCannotAttachAScanAfterWaitingForTheComboLock prevents a stale join snapshot.
func TestARemovedGuestCannotAttachAScanAfterWaitingForTheComboLock(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	r := repository.New(pool)
	host, guest := createAccount(t, r, "test_host"), createAccount(t, r, "test_guest")
	now := time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)
	c, err := r.CreateCombo(t.Context(), host.AccountID, "123456", now)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	_, err = r.JoinCombo(t.Context(), guest.AccountID, c.Code, now)
	if err != nil {
		t.Fatalf("join: %v", err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin removal: %v", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }() // no-op once committed
	_, err = tx.Exec(ctx, `select combo_id from app.combo where combo_id=$1 for update`, c.ComboID)
	if err != nil {
		t.Fatalf("hold parent lock: %v", err)
	}
	result := make(chan error, 1)
	go func() {
		_, err := r.CreateComboScan(ctx, guest.AccountID, c.ComboID, 0, "Empty.", nil, now)
		result <- err
	}()
	blocked := false
	for range 200 {
		var n int
		err = pool.QueryRow(ctx, `select count(*) from pg_stat_activity where datname=current_database()
			and wait_event_type='Lock' and query like '%for update of c%'`).Scan(&n)
		if err != nil {
			t.Fatalf("wait for scan lock: %v", err)
		}
		if n > 0 {
			blocked = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !blocked {
		t.Fatal("scan never waited on the combo lock")
	}
	_, err = tx.Exec(ctx, `delete from app.combo_member where combo_id=$1 and account_id=$2`, c.ComboID, guest.AccountID)
	if err != nil {
		t.Fatalf("remove guest: %v", err)
	}
	err = tx.Commit(ctx)
	if err != nil {
		t.Fatalf("commit removal: %v", err)
	}
	select {
	case err := <-result:
		if !errors.Is(err, repository.ErrNotFound) {
			t.Errorf("attach after removal = %v, want not found", err)
		}
	case <-ctx.Done():
		t.Fatalf("scan never finished: %v", ctx.Err())
	}
	var scans int
	err = pool.QueryRow(ctx, `select count(*) from app.scan`).Scan(&scans)
	if err != nil {
		t.Fatalf("scan count: %v", err)
	}
	if scans != 0 {
		t.Errorf("stored scans = %d, want 0", scans)
	}
}

// TestTwentyGuestsRaceForOneComboSeat prevents a count-then-insert from overfilling a combo.
func TestTwentyGuestsRaceForOneComboSeat(t *testing.T) {
	t.Parallel()
	r := newRepo(t)
	host := createAccount(t, r, "test_host")
	now := time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)
	c, err := r.CreateCombo(t.Context(), host.AccountID, "012345", now)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	guests := make([]int64, 20)
	for i := range guests {
		guests[i] = createAccount(t, r, fmt.Sprintf("test_guest%d", i)).AccountID
	}
	results := make(chan error, len(guests))
	start := make(chan struct{})
	var wg sync.WaitGroup
	for _, id := range guests {
		wg.Go(func() { <-start; _, err := r.JoinCombo(t.Context(), id, c.Code, now); results <- err })
	}
	close(start)
	wg.Wait()
	close(results)
	winners := 0
	for err := range results {
		if err == nil {
			winners++
		} else if !errors.Is(err, repository.ErrComboFull) {
			t.Errorf("join = %v, want full", err)
		}
	}
	if winners != 1 {
		t.Errorf("winners = %d, want 1", winners)
	}
	got, err := r.ComboByID(t.Context(), host.AccountID, c.ComboID)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(got.Members) != 2 {
		t.Errorf("members = %d, want 2", len(got.Members))
	}
	if !got.ExpiresAt.Equal(now.Add(2 * time.Hour)) {
		t.Errorf("expiry = %v, want two hours", got.ExpiresAt)
	}
	_, err = r.ComboByID(t.Context(), guests[0]+1000, c.ComboID)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("outsider read = %v, want not found", err)
	}
}

// TestComboCodeCollisionsRollbackAndExpiredCodesCanBeReused checks the invite's actual unique index.
func TestComboCodeCollisionsRollbackAndExpiredCodesCanBeReused(t *testing.T) {
	t.Parallel()
	r := newRepo(t)
	host := createAccount(t, r, "test_host")
	other := createAccount(t, r, "test_other")
	now := time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)
	first, err := r.CreateCombo(t.Context(), host.AccountID, "000000", now)
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	_, err = r.CreateCombo(t.Context(), other.AccountID, "000000", now)
	if !errors.Is(err, repository.ErrDuplicate) {
		t.Fatalf("collision = %v, want duplicate", err)
	}
	before, err := r.ComboByID(t.Context(), host.AccountID, first.ComboID)
	if err != nil {
		t.Fatalf("read first: %v", err)
	}
	if len(before.Members) != 1 || before.State != repository.ComboOpen {
		t.Errorf("collision changed first: %+v", before)
	}
	reused, err := r.CreateCombo(t.Context(), other.AccountID, "000000", now.Add(2*time.Hour))
	if err != nil {
		t.Fatalf("reuse: %v", err)
	}
	if reused.ComboID == first.ComboID || reused.HostID != other.AccountID {
		t.Errorf("reused = %+v, want new invite", reused)
	}
	after, err := r.ComboByID(t.Context(), host.AccountID, first.ComboID)
	if err != nil {
		t.Fatalf("read retired: %v", err)
	}
	if after.State != repository.ComboEnded {
		t.Errorf("old state = %s, want ended", after.State)
	}
}

// TestComboTablesEnforceSeatsAndCascadeAccountDeletion checks database invariants beyond the handler.
func TestComboTablesEnforceSeatsAndCascadeAccountDeletion(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	r := repository.New(pool)
	host := createAccount(t, r, "test_host")
	guest := createAccount(t, r, "test_guest")
	now := time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)
	c, err := r.CreateCombo(t.Context(), host.AccountID, "123456", now)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	_, err = r.JoinCombo(t.Context(), guest.AccountID, c.Code, now)
	if err != nil {
		t.Fatalf("join: %v", err)
	}
	for _, slot := range []int{0, 1, 2, 3} {
		_, err := pool.Exec(t.Context(), `insert into app.combo_member(combo_id,account_id,slot) values($1,$2,$3)`, c.ComboID, guest.AccountID, slot)
		if err == nil {
			t.Errorf("slot %d allowed a third membership", slot)
		}
	}
	changed, err := r.EndCombo(t.Context(), guest.AccountID, c.ComboID)
	if err != nil || changed {
		t.Errorf("guest end = %v,%v, want false,nil", changed, err)
	}
	changed, err = r.RemoveComboPlayer(t.Context(), guest.AccountID, c.ComboID)
	if err != nil || changed {
		t.Errorf("guest removal = %v,%v, want false,nil", changed, err)
	}
	_, err = r.CreateComboScan(t.Context(), guest.AccountID, c.ComboID, 19, "Eggs.", []repository.ScanItem{{Name: "Eggs", Category: repository.CategoryProtein}}, now)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	_, err = pool.Exec(t.Context(), `delete from app.account where account_id=$1`, host.AccountID)
	if err != nil {
		t.Fatalf("delete host: %v", err)
	}
	var combos, members, scans, attached int
	err = pool.QueryRow(t.Context(), `select (select count(*) from app.combo),(select count(*) from app.combo_member),
		(select count(*) from app.scan),(select count(*) from app.scan where combo_id is not null)`).Scan(&combos, &members, &scans, &attached)
	if err != nil {
		t.Fatalf("cascade counts: %v", err)
	}
	if combos != 0 || members != 0 || scans != 1 || attached != 0 {
		t.Errorf("counts = %d,%d,%d,%d, want 0,0,1,0", combos, members, scans, attached)
	}
}
