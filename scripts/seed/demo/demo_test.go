// Author: Caden Lund
package demo_test

import (
	"maps"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bengutow1/csc4330_project5/packages/go/points"
	"github.com/bengutow1/csc4330_project5/packages/go/repository"
	"github.com/bengutow1/csc4330_project5/packages/go/scoring"
	"github.com/bengutow1/csc4330_project5/scripts/seed/demo"
)

// wednesday is when most of these tests seed: midweek, so Monday and Tuesday are behind it.
var wednesday = time.Date(2026, 10, 21, 18, 0, 0, 0, points.Campus())

// TestTheDemoHasFifteenPlayersAndEveryTierBothWeeks: the board and the verdict screens have a full
// cast to show, with every tier scanned this week and last, and the report says what was written.
func TestTheDemoHasFifteenPlayersAndEveryTierBothWeeks(t *testing.T) {
	t.Parallel()
	pool := newPool(t)

	report, err := demo.Seed(t.Context(), repository.New(pool), wednesday)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	handles := texts(t, pool, `select handle from app.account`)
	for _, want := range []string{"chef_maya", "gymbro_tyler", "meal_prep_sam", "ramen_king", "dorm_fridge_dan"} {
		if !slices.Contains(handles, want) {
			t.Errorf("players %v, want %s among them", handles, want)
		}
	}
	if len(handles) != 15 {
		t.Errorf("%d players, want 15", len(handles))
	}
	weekStart := points.WeekStart(wednesday)
	for _, week := range []struct {
		name     string
		from, to time.Time
	}{
		{"this week", weekStart, wednesday},
		{"last week", weekStart.AddDate(0, 0, -7), weekStart},
	} {
		tiers := map[scoring.Tier]bool{}
		for _, score := range numbers(t, pool, `
			select s.score
			  from app.scan s
			 where s.created_at >= $1
			   and s.created_at < $2`, week.from, week.to) {
			tiers[scoring.TierOf(score)] = true
		}
		for _, tier := range []scoring.Tier{
			scoring.TierCertifiedChef, scoring.TierHomeCook, scoring.TierScrappy, scoring.TierFriesInTheBag,
		} {
			if !tiers[tier] {
				t.Errorf("%s: no %s scan, want every tier", week.name, tier)
			}
		}
	}
	want := demo.Report{
		Players: 15,
		Scans:   count(t, pool, `select count(*) from app.scan`),
		FryRuns: count(t, pool, `select count(*) from app.fry_run`),
	}
	if report != want || want.Scans == 0 || want.FryRuns == 0 {
		t.Errorf("report = %+v, want %+v, which counts what is stored", report, want)
	}
}

// TestTheDemoHasOneClearFridgeOfTheWeek: chef_maya's Monday fridge beats every other scan this
// week outright, so the card never flips between two players, and last week's 100 does not count.
func TestTheDemoHasOneClearFridgeOfTheWeek(t *testing.T) {
	t.Parallel()
	pool := newPool(t)

	_, err := demo.Seed(t.Context(), repository.New(pool), wednesday)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	weekStart := points.WeekStart(wednesday)
	whose := texts(t, pool, `
		select a.handle
		  from app.scan s
		  join app.account a on a.account_id = s.account_id
		 where s.created_at >= $1
		 order by s.score desc
		 limit 1`, weekStart)
	scores := numbers(t, pool, `
		select s.score
		  from app.scan s
		 where s.created_at >= $1
		 order by s.score desc
		 limit 2`, weekStart)
	if !slices.Equal(whose, []string{"chef_maya"}) || len(scores) != 2 || scores[0] != 100 || scores[1] >= 100 {
		t.Errorf("this week's best is %v's, scores %v, want chef_maya's 100 alone at the top", whose, scores)
	}
}

// TestDormFridgeDanLeadsTheHallOfFriesThisWeek: the Hall of Fries has a clear leader and enough
// players with a run that a newcomer's first run lands below the fold.
func TestDormFridgeDanLeadsTheHallOfFriesThisWeek(t *testing.T) {
	t.Parallel()
	pool := newPool(t)

	_, err := demo.Seed(t.Context(), repository.New(pool), wednesday)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	weekStart := points.WeekStart(wednesday)
	leader := texts(t, pool, `
		select a.handle
		  from app.fry_run f
		  join app.account a on a.account_id = f.account_id
		 where f.created_at >= $1
		 group by a.handle
		 order by count(*) desc, a.handle
		 limit 1`, weekStart)
	most := numbers(t, pool, `
		select count(*)
		  from app.fry_run f
		 where f.created_at >= $1
		 group by f.account_id
		 order by count(*) desc`, weekStart)
	if !slices.Equal(leader, []string{"dorm_fridge_dan"}) || len(most) < 2 || most[0] != 5 || most[1] >= 5 {
		t.Errorf("this week's leader is %v, run counts %v, want dorm_fridge_dan's 5 alone at the top", leader, most)
	}
	if len(most) != 11 {
		t.Errorf("%d players with a run this week, want 11", len(most))
	}
}

// TestTheDemoLeavesOutWhatHasNotHappenedYet: seeded on Monday afternoon, nothing is dated after
// now, the report counts only what was written, and the Fridge of the Week is already there.
func TestTheDemoLeavesOutWhatHasNotHappenedYet(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	monday := time.Date(2026, 10, 19, 15, 0, 0, 0, points.Campus())

	report, err := demo.Seed(t.Context(), repository.New(pool), monday)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	ahead := count(t, pool, `select count(*) from app.scan where created_at > $1`, monday) +
		count(t, pool, `select count(*) from app.fry_run where created_at > $1`, monday)
	if ahead != 0 {
		t.Errorf("%d scans and fry runs after now, want none", ahead)
	}
	best := texts(t, pool, `
		select a.handle
		  from app.scan s
		  join app.account a on a.account_id = s.account_id
		 where s.created_at >= $1
		 order by s.score desc
		 limit 1`, points.WeekStart(monday))
	if !slices.Equal(best, []string{"chef_maya"}) {
		t.Errorf("Monday's best scan is %v, want chef_maya's", best)
	}
	want := demo.Report{
		Players: 15,
		Scans:   count(t, pool, `select count(*) from app.scan`),
		FryRuns: count(t, pool, `select count(*) from app.fry_run`),
	}
	if report != want {
		t.Errorf("report = %+v, want all 15 players and only what was written, %+v", report, want)
	}
}

// TestSeedingTwiceChangesNothing: a second run finds the demo and leaves every table as it was.
func TestSeedingTwiceChangesNothing(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	repo := repository.New(pool)

	first, err := demo.Seed(t.Context(), repo, wednesday)
	if err != nil {
		t.Fatalf("first seed: %v", err)
	}
	before := tableCounts(t, pool)
	second, err := demo.Seed(t.Context(), repo, wednesday.Add(time.Hour))
	if err != nil {
		t.Fatalf("second seed: %v", err)
	}
	after := tableCounts(t, pool)

	if first.AlreadySeeded || first.Players != 15 {
		t.Errorf("first run = %+v, want it to write all 15 players", first)
	}
	if second != (demo.Report{AlreadySeeded: true}) {
		t.Errorf("second run = %+v, want it to find the demo and write nothing", second)
	}
	if !maps.Equal(before, after) {
		t.Errorf("rows after the second run = %v, want %v", after, before)
	}
}

// TestAFailedSeedWritesNothing: one transaction holds the whole demo, so a run that fails partway
// (here a handle someone already took) leaves no half-seeded database for the next run to skip.
func TestAFailedSeedWritesNothing(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	repo := repository.New(pool)
	_, err := repo.CreateAccount(t.Context(), "dorm_fridge_dan", "🍟")
	if err != nil {
		t.Fatalf("take dorm_fridge_dan's handle: %v", err)
	}

	_, err = demo.Seed(t.Context(), repo, wednesday)

	if err == nil {
		t.Fatal("seed with a handle taken: no error, want one")
	}
	got := tableCounts(t, pool)
	want := map[string]int{"app.account": 1, "app.scan": 0, "app.scan_item": 0, "app.fry_run": 0}
	if !maps.Equal(got, want) {
		t.Errorf("rows after the failed run = %v, want only the account that was there, %v", got, want)
	}
}

// tableCounts is the row count of every table the demo writes.
func tableCounts(t *testing.T, pool *pgxpool.Pool) map[string]int {
	t.Helper()
	return map[string]int{
		"app.account":   count(t, pool, `select count(*) from app.account`),
		"app.scan":      count(t, pool, `select count(*) from app.scan`),
		"app.scan_item": count(t, pool, `select count(*) from app.scan_item`),
		"app.fry_run":   count(t, pool, `select count(*) from app.fry_run`),
	}
}

func count(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) int {
	t.Helper()
	var n int
	err := pool.QueryRow(t.Context(), sql, args...).Scan(&n)
	if err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	return n
}

func texts(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) []string {
	t.Helper()
	rows, err := pool.Query(t.Context(), sql, args...)
	if err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		err = rows.Scan(&s)
		if err != nil {
			t.Fatalf("scan %s: %v", sql, err)
		}
		out = append(out, s)
	}
	err = rows.Err()
	if err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	return out
}

func numbers(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) []int {
	t.Helper()
	rows, err := pool.Query(t.Context(), sql, args...)
	if err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	defer rows.Close()
	var out []int
	for rows.Next() {
		var n int
		err = rows.Scan(&n)
		if err != nil {
			t.Fatalf("scan %s: %v", sql, err)
		}
		out = append(out, n)
	}
	err = rows.Err()
	if err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	return out
}
