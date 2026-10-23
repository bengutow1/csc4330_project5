// Author: Ben Gutowski
package repository

import (
	"context"
	"fmt"
)

// crew.go owns app.crew and app.crew_member: friend groups that get a board of their own, joined
// by a six-digit code that is one crew's alone (crew_code_key). Every read is fenced by the
// caller's membership in the SQL, so a crew a player is not in reads as missing. A player is in at
// most a handful of crews: making and joining lock their account row before reading anything, so
// two taps at once can neither slip under the cap nor both read "not a member yet". The last
// member out ends the crew, by leaving or by deleting their account: a trigger in 00008_crews.sql
// deletes a crew once its last membership goes.

// Crew is one crew, as a member sees it.
type Crew struct {
	CrewID  int64
	Name    string
	Code    string // six digits
	Members int    // the caller included
}

// CreateCrew makes a crew named name with code, owned by accountID and with them as its first
// member. It answers ErrTooManyCrews when they are already in maxCrews, and ErrDuplicate when
// another crew holds code, so the caller can draw a fresh one.
func (r *Repository) CreateCrew(ctx context.Context, accountID int64, name, code string, maxCrews int) (Crew, error) {
	op := fmt.Sprintf("create crew account=%d", accountID)
	var c Crew
	err := r.InTx(ctx, func(tx *Repository) error {
		err := tx.lockForCrews(ctx, accountID)
		if err != nil {
			return err
		}
		err = tx.underCrewCap(ctx, accountID, maxCrews)
		if err != nil {
			return err
		}
		err = tx.db.QueryRow(ctx, `
			   insert into app.crew (name, code, owner_id)
			   values ($2, $3, $1)
			returning crew_id, name, code`, accountID, name, code).Scan(&c.CrewID, &c.Name, &c.Code)
		if err != nil {
			return known(op, err)
		}
		_, err = tx.db.Exec(ctx, `
			insert into app.crew_member (crew_id, account_id)
			values ($2, $1)`, accountID, c.CrewID)
		if err != nil {
			return known(op, err)
		}
		c.Members = 1
		return nil
	})
	if err != nil {
		return Crew{}, err
	}
	return c, nil
}

// JoinCrew adds accountID to the crew whose code is code and returns it, counting them. A code no
// crew holds is ErrNoCrewForCode. Joining a crew they are already in changes nothing and returns
// it all the same; joining another past maxCrews is ErrTooManyCrews.
func (r *Repository) JoinCrew(ctx context.Context, accountID int64, code string, maxCrews int) (Crew, error) {
	op := fmt.Sprintf("join crew account=%d", accountID)
	var c Crew
	err := r.InTx(ctx, func(tx *Repository) error {
		// 1. The player's lock first, so whether they are in the crew already is read after any
		// join of theirs that got there first: two taps at once both answer with the crew
		err := tx.lockForCrews(ctx, accountID)
		if err != nil {
			return err
		}

		// 2. The crew, and whether they are in it already
		var member bool
		err = tx.db.QueryRow(ctx, `
			select c.crew_id,
			       exists (select 1
			                 from app.crew_member m
			                where m.crew_id = c.crew_id
			                  and m.account_id = $1)
			  from app.crew c
			 where c.code = $2
			   for share of c`, accountID, code).Scan(&c.CrewID, &member)
		if isNoRows(err) {
			return fmt.Errorf("%s: %w", op, ErrNoCrewForCode)
		}
		if err != nil {
			return known(op, err)
		}

		// 3. A new member, under the cap
		if !member {
			err = tx.underCrewCap(ctx, accountID, maxCrews)
			if err != nil {
				return err
			}
			_, err = tx.db.Exec(ctx, `
				insert into app.crew_member (crew_id, account_id)
				values ($2, $1)`, accountID, c.CrewID)
			if err != nil {
				return known(op, err)
			}
		}

		// 4. The crew as its members now see it
		c, err = tx.CrewByID(ctx, accountID, c.CrewID)
		return err
	})
	if err != nil {
		return Crew{}, err
	}
	return c, nil
}

// lockForCrews locks accountID's row, so their makes and joins take turns. Run it inside a
// transaction and before reading their crews: the lock lasts until the transaction ends, and every
// read after it sees what the one before committed.
func (r *Repository) lockForCrews(ctx context.Context, accountID int64) error {
	_, err := r.db.Exec(ctx, `
		select 1
		  from app.account a
		 where a.account_id = $1
		   for update`, accountID)
	if err != nil {
		return known(fmt.Sprintf("lock for crews account=%d", accountID), err)
	}
	return nil
}

// underCrewCap answers ErrTooManyCrews when accountID is already in maxCrews. Call it after
// lockForCrews, so two at once cannot both count under the cap.
func (r *Repository) underCrewCap(ctx context.Context, accountID int64, maxCrews int) error {
	op := fmt.Sprintf("crew cap account=%d", accountID)
	var n int
	err := r.db.QueryRow(ctx, `
		select count(*)
		  from app.crew_member m
		 where m.account_id = $1`, accountID).Scan(&n)
	if err != nil {
		return known(op, err)
	}
	if n >= maxCrews {
		return fmt.Errorf("%s: in %d: %w", op, n, ErrTooManyCrews)
	}
	return nil
}

// CrewByID is crewID as accountID sees it, or ErrNotFound when they are not in it: another
// crew's board and a crew that never was read the same.
func (r *Repository) CrewByID(ctx context.Context, accountID, crewID int64) (Crew, error) {
	var c Crew
	err := r.db.QueryRow(ctx, `
		select c.crew_id, c.name, c.code,
		       (select count(*) from app.crew_member n where n.crew_id = c.crew_id)
		  from app.crew c
		  join app.crew_member m on m.crew_id = c.crew_id
		 where m.account_id = $1
		   and c.crew_id = $2`, accountID, crewID).Scan(&c.CrewID, &c.Name, &c.Code, &c.Members)
	if err != nil {
		return Crew{}, known(fmt.Sprintf("crew by id account=%d crew=%d", accountID, crewID), err)
	}
	return c, nil
}

// ListMyCrews is every crew accountID is in, the one they joined first first. A player is in a
// handful at most, so it is not paged.
func (r *Repository) ListMyCrews(ctx context.Context, accountID int64) ([]Crew, error) {
	op := fmt.Sprintf("list my crews account=%d", accountID)
	rows, err := r.db.Query(ctx, `
		select c.crew_id, c.name, c.code,
		       (select count(*) from app.crew_member n where n.crew_id = c.crew_id)
		  from app.crew_member m
		  join app.crew c on c.crew_id = m.crew_id
		 where m.account_id = $1
		 order by m.created_at, c.crew_id`, accountID)
	if err != nil {
		return nil, known(op, err)
	}
	defer rows.Close()
	var out []Crew
	for rows.Next() {
		var c Crew
		err = rows.Scan(&c.CrewID, &c.Name, &c.Code, &c.Members)
		if err != nil {
			return nil, known(op, err)
		}
		out = append(out, c)
	}
	err = rows.Err()
	if err != nil {
		return nil, known(op, err)
	}
	return out, nil
}

// LeaveCrew takes accountID out of crewID and says whether they were in it. The last member out
// ends the crew, through the crew_member_end_empty_crew trigger, so its code goes free.
func (r *Repository) LeaveCrew(ctx context.Context, accountID, crewID int64) (bool, error) {
	op := fmt.Sprintf("leave crew account=%d crew=%d", accountID, crewID)
	var left bool
	err := r.InTx(ctx, func(tx *Repository) error {
		// The crew's row first. A join holds it shared, so the last leave and a join take turns:
		// the join lands first and keeps the crew, or it finds the code gone.
		_, err := tx.db.Exec(ctx, `
			select 1
			  from app.crew c
			 where c.crew_id = $1
			   for update`, crewID)
		if err != nil {
			return known(op, err)
		}
		tag, err := tx.db.Exec(ctx, `
			delete from app.crew_member m
			 where m.account_id = $1
			   and m.crew_id = $2`, accountID, crewID)
		if err != nil {
			return known(op, err)
		}
		left = tag.RowsAffected() > 0
		return nil
	})
	if err != nil {
		return false, err
	}
	return left, nil
}
