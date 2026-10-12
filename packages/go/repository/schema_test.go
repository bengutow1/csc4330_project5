// Author: Caden Lund
package repository_test

import "testing"

// TestSetUpdatedAtStampsARowOnEveryUpdate guards the trigger every mutable table will share: if it
// stopped firing, updated_at would freeze at the insert and nothing else would notice.
func TestSetUpdatedAtStampsARowOnEveryUpdate(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	ctx := t.Context()
	_, err := pool.Exec(ctx, `
		create table app.widget (
		    widget_id  bigint      generated always as identity primary key,
		    label      text        not null,
		    updated_at timestamptz not null default now()
		)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	_, err = pool.Exec(ctx, `
		create trigger set_updated_at before update on app.widget
		    for each row execute function app.set_updated_at()`)
	if err != nil {
		t.Fatalf("attach trigger: %v", err)
	}
	// Stamped an hour back by the database's own clock, which can run a hair behind this machine's.
	_, err = pool.Exec(ctx, `
		insert into app.widget (label, updated_at)
		values ('before', now() - interval '1 hour')`)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	_, err = pool.Exec(ctx, `update app.widget set label = 'after'`)
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	var fresh bool
	err = pool.QueryRow(ctx, `
		select w.updated_at > now() - interval '1 minute'
		  from app.widget w`).Scan(&fresh)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if !fresh {
		t.Error("updated_at kept the insert's stamp after an update")
	}
}
