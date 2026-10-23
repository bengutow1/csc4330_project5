// Author: LAPTOP-BPMQJP10\andyk
package repository

import (
	"context"
	"fmt"
	"time"
)

// combo.go owns app.combo and app.combo_member. Fixed slots enforce the two-player cap;
// the combo row serializes joins, removal, ending and attaching a completed scan.
const (
	ComboOpen  = "open"
	ComboEnded = "ended"
)

// Combo is an invite and the latest scan for each current member, host first.
type Combo struct {
	ComboID   int64
	Code      string
	HostID    int64
	State     string
	ExpiresAt time.Time
	Members   []ComboMember
}

// ComboMember is public identity and scan status; Scan is nil until this member scans.
type ComboMember struct {
	AccountID int64
	Handle    string
	Avatar    string
	Scan      *Scan
}

// CreateCombo stores the invite and host membership together. A live code collision is ErrDuplicate.
func (r *Repository) CreateCombo(ctx context.Context, accountID int64, code string, now time.Time) (Combo, error) {
	var c Combo
	err := r.InTx(ctx, func(tx *Repository) error {
		// Retire this expired code before reusing it; the unique index arbitrates concurrent starts.
		_, err := tx.db.Exec(ctx, `
			update app.combo set state = 'ended'
			 where code = $1 and state = 'open' and expires_at <= $2`, code, now)
		if err != nil {
			return known("retire expired combo code", err)
		}
		var id int64
		err = tx.db.QueryRow(ctx, `
			insert into app.combo (host_id, code, created_at, expires_at)
			values ($1, $2, $3, $3::timestamptz + interval '2 hours') returning combo_id`, accountID, code, now).Scan(&id)
		if err != nil {
			return known(fmt.Sprintf("create combo account=%d", accountID), err)
		}
		_, err = tx.db.Exec(ctx, `
			insert into app.combo_member (combo_id, account_id, slot) values ($1, $2, 1)`, id, accountID)
		if err != nil {
			return known(fmt.Sprintf("add combo host combo=%d", id), err)
		}
		c, err = tx.ComboByID(ctx, accountID, id)
		return err
	})
	return c, err
}

// JoinCombo claims the guest seat in one guarded statement. Rejoining is idempotent.
func (r *Repository) JoinCombo(ctx context.Context, accountID int64, code string, now time.Time) (int64, error) {
	var id int64
	var state string
	var expired, joined bool
	err := r.db.QueryRow(ctx, `
		with target as materialized (
			select c.combo_id, c.host_id, c.state, c.expires_at
			  from app.combo c where c.code = $2
			 order by (c.state = 'open') desc, c.created_at desc, c.combo_id desc limit 1 for update
		), seat as (
			insert into app.combo_member as m (combo_id, account_id, slot)
			select t.combo_id, $1, 2 from target t
			 where t.state = 'open' and t.expires_at > $3 and t.host_id <> $1
			on conflict (combo_id, slot) do update set account_id = m.account_id
			 where m.account_id = excluded.account_id
			returning combo_id
		)
		select t.combo_id, t.state, t.expires_at <= $3,
		       t.host_id = $1 or exists (select 1 from seat)
		  from target t`, accountID, code, now).Scan(&id, &state, &expired, &joined)
	if isNoRows(err) {
		return 0, ErrComboCodeMissing
	}
	if err != nil {
		return 0, known(fmt.Sprintf("join combo account=%d", accountID), err)
	}
	switch {
	case state == ComboEnded:
		return 0, ErrComboEnded
	case expired:
		return 0, ErrComboExpired
	case !joined:
		return 0, ErrComboFull
	default:
		return id, nil
	}
}

// ComboByID reads an entire lobby in one indexed query, fenced to its current members.
func (r *Repository) ComboByID(ctx context.Context, accountID, comboID int64) (Combo, error) {
	op := fmt.Sprintf("combo by id account=%d combo=%d", accountID, comboID)
	rows, err := r.db.Query(ctx, `
		select c.combo_id, c.code, c.host_id, c.state, c.expires_at,
		       a.account_id, a.handle, a.avatar, s.scan_id, s.score, s.roast,
		       s.items_version, s.created_at, i.name, i.category, i.quantity
		  from app.combo c
		  join app.combo_member mine on mine.combo_id = c.combo_id and mine.account_id = $1
		  join app.combo_member m on m.combo_id = c.combo_id
		  join app.account a on a.account_id = m.account_id
		  left join lateral (
			select s.scan_id, s.score, s.roast, s.items_version, s.created_at from app.scan s
			 where s.combo_id = c.combo_id and s.account_id = m.account_id
			 order by s.created_at desc, s.scan_id desc limit 1
		  ) s on true
		  left join app.scan_item i on i.scan_id = s.scan_id
		 where c.combo_id = $2
		 order by m.slot, i.position`, accountID, comboID)
	if err != nil {
		return Combo{}, known(op, err)
	}
	defer rows.Close()
	var c Combo
	for rows.Next() {
		var m ComboMember
		var scanID *int64
		var score, version *int
		var roast, name, category, quantity *string
		var created *time.Time
		err = rows.Scan(&c.ComboID, &c.Code, &c.HostID, &c.State, &c.ExpiresAt,
			&m.AccountID, &m.Handle, &m.Avatar, &scanID, &score, &roast, &version, &created,
			&name, &category, &quantity)
		if err != nil {
			return Combo{}, known(op, err)
		}
		if len(c.Members) == 0 || c.Members[len(c.Members)-1].AccountID != m.AccountID {
			if scanID != nil {
				m.Scan = &Scan{ScanID: *scanID, Score: *score, Roast: *roast, ItemsVersion: *version, CreatedAt: *created}
			}
			c.Members = append(c.Members, m)
		}
		if name != nil {
			s := c.Members[len(c.Members)-1].Scan
			s.Items = append(s.Items, ScanItem{Name: *name, Category: *category, Quantity: *quantity})
		}
	}
	err = rows.Err()
	if err != nil {
		return Combo{}, known(op, err)
	}
	if c.ComboID == 0 {
		return Combo{}, ErrNotFound
	}
	return c, nil
}

// EndCombo ends only the caller's hosted combo. False means this caller hosts no such combo.
func (r *Repository) EndCombo(ctx context.Context, accountID, comboID int64) (bool, error) {
	tag, err := r.db.Exec(ctx, `
		update app.combo set state = 'ended' where host_id = $1 and combo_id = $2`, accountID, comboID)
	if err != nil {
		return false, known(fmt.Sprintf("end combo account=%d combo=%d", accountID, comboID), err)
	}
	return tag.RowsAffected() != 0, nil
}

// RemoveComboPlayer removes the only guest and detaches their scans, so rejoining starts fresh.
func (r *Repository) RemoveComboPlayer(ctx context.Context, accountID, comboID int64) (bool, error) {
	changed := false
	op := fmt.Sprintf("remove combo player account=%d combo=%d", accountID, comboID)
	err := r.InTx(ctx, func(tx *Repository) error {
		var id int64
		err := tx.db.QueryRow(ctx, `select combo_id from app.combo
			 where host_id = $1 and combo_id = $2 for update`, accountID, comboID).Scan(&id)
		if isNoRows(err) {
			return nil
		}
		if err != nil {
			return known(op, err)
		}
		_, err = tx.db.Exec(ctx, `
			update app.scan s set combo_id = null from app.combo_member m
			 where m.combo_id = $1 and m.slot = 2 and s.combo_id = m.combo_id and s.account_id = m.account_id`, comboID)
		if err != nil {
			return known(op, err)
		}
		_, err = tx.db.Exec(ctx, `delete from app.combo_member where combo_id = $1 and slot = 2`, comboID)
		if err != nil {
			return known(op, err)
		}
		changed = true
		return nil
	})
	return changed, err
}

// CreateComboScan attaches a verdict only while the caller is still an active member.
// The model ran before this transaction; removal or ending during that call cannot attach food.
func (r *Repository) CreateComboScan(ctx context.Context, accountID, comboID int64, score int, roast string, items []ScanItem, now time.Time) (Scan, error) {
	var s Scan
	err := r.InTx(ctx, func(tx *Repository) error {
		var lockedID int64
		err := tx.db.QueryRow(ctx, `
			select c.combo_id from app.combo c
			 join app.combo_member m on m.combo_id = c.combo_id and m.account_id = $1
			 where c.combo_id = $2 for update of c`, accountID, comboID).Scan(&lockedID)
		if err != nil {
			return known(fmt.Sprintf("lock combo scan account=%d combo=%d", accountID, comboID), err)
		}
		// Waiting for the parent lock can outlive removal's commit. This new statement must
		// re-read membership rather than trust the snapshot that started waiting.
		var state string
		var expires time.Time
		err = tx.db.QueryRow(ctx, `
			select c.state, c.expires_at from app.combo c
			 join app.combo_member m on m.combo_id = c.combo_id and m.account_id = $1
			 where c.combo_id = $2`, accountID, comboID).Scan(&state, &expires)
		if err != nil {
			return known(fmt.Sprintf("attach scan account=%d combo=%d", accountID, comboID), err)
		}
		if state == ComboEnded {
			return ErrComboEnded
		}
		if !expires.After(now) {
			return ErrComboExpired
		}
		s, err = tx.CreateScan(ctx, accountID, score, roast, items)
		if err != nil {
			return err
		}
		_, err = tx.db.Exec(ctx, `update app.scan set combo_id = $3 where account_id = $1 and scan_id = $2`, accountID, s.ScanID, comboID)
		if err != nil {
			return known(fmt.Sprintf("attach scan account=%d combo=%d", accountID, comboID), err)
		}
		return nil
	})
	return s, err
}
