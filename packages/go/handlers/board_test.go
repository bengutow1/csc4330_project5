// Author: Caden Lund
package handlers_test

import (
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/alexedwards/scs/pgxstore"
	"github.com/alexedwards/scs/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bengutow1/csc4330_project5/packages/go/apigen/account/v1/accountv1connect"
	boardv1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/board/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/apigen/board/v1/boardv1connect"
	"github.com/bengutow1/csc4330_project5/packages/go/auth"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers/account"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers/board"
	"github.com/bengutow1/csc4330_project5/packages/go/points"
	"github.com/bengutow1/csc4330_project5/packages/go/repository"
)

const (
	chefs       = boardv1.Board_BOARD_CHEFS
	hallOfFries = boardv1.Board_BOARD_HALL_OF_FRIES
	thisWeek    = boardv1.Range_RANGE_THIS_WEEK
	allTime     = boardv1.Range_RANGE_ALL_TIME
)

// wednesday is when most of these tests stand: midweek, so this week's runs are behind them.
var wednesday = time.Date(2026, 10, 21, 18, 0, 0, 0, points.Campus())

// boardKinds is each board, how a test gives a player n of what that board counts, and whether
// its rank-1 rows wear the crown. The tests below run on both, so the two boards cannot drift.
var boardKinds = []struct {
	name   string
	board  boardv1.Board
	give   func(s boardServer, t *testing.T, handle string, n int, at time.Time)
	crowns bool
}{
	{name: "chefs", board: chefs, give: boardServer.cooks, crowns: false},
	{name: "hall of fries", board: hallOfFries, give: boardServer.runs, crowns: true},
}

// TestTheBoardRanksTiesTogetherAndCrownsTheLeader: players with as much share a rank, the next
// rank skips past them, ties go by handle, and every rank-1 player on the Hall of Fries wears the
// crown, never on Chefs.
func TestTheBoardRanksTiesTogetherAndCrownsTheLeader(t *testing.T) {
	t.Parallel()
	for _, k := range boardKinds {
		t.Run(k.name, func(t *testing.T) {
			t.Parallel()
			srv := boardStack(t, wednesday)
			maya := boardPlayer(t, srv, "test_maya")
			k.give(srv, t, "test_maya", 3, wednesday.AddDate(0, 0, -2))
			k.give(srv, t, "test_ben", 3, wednesday.AddDate(0, 0, -2))
			k.give(srv, t, "test_cal", 2, wednesday.AddDate(0, 0, -2))

			got := getBoard(t, maya, &boardv1.GetBoardRequest{Board: k.board, Range: thisWeek})

			want := crowned(k.crowns, "1 test_ben 3", "1 test_maya 3", "3 test_cal 2")
			rows := boardRows(got.Rows)
			if !slices.Equal(rows, want) {
				t.Errorf("rows = %v, want %v", rows, want)
			}
			if got.NextCursor != "" {
				t.Errorf("next_cursor = %q, want \"\": the board fits one page", got.NextCursor)
			}
		})
	}
}

// TestPagingTheBoardWalksToTheEndWithNoGapsOrRepeats: pages of two split each tie across a page
// boundary, the ranks stay the whole board's, and the walk ends.
func TestPagingTheBoardWalksToTheEndWithNoGapsOrRepeats(t *testing.T) {
	t.Parallel()
	for _, k := range boardKinds {
		t.Run(k.name, func(t *testing.T) {
			t.Parallel()
			srv := boardStack(t, wednesday)
			maya := boardPlayer(t, srv, "test_maya")
			for handle, n := range map[string]int{"test_ann": 3, "test_bob": 3, "test_cat": 2, "test_dan": 2, "test_eli": 1} {
				k.give(srv, t, handle, n, wednesday.AddDate(0, 0, -2))
			}
			k.give(srv, t, "test_maya", 1, wednesday.AddDate(0, 0, -2))
			want := crowned(k.crowns,
				"1 test_ann 3", "1 test_bob 3", "3 test_cat 2", "3 test_dan 2", "5 test_eli 1", "5 test_maya 1",
			)

			var walked []string
			req := &boardv1.GetBoardRequest{Board: k.board, Range: thisWeek, Limit: 2}
			for pages := 0; ; pages++ {
				if pages > 5 {
					t.Fatalf("still paging after %d pages: %v", pages, walked)
				}
				got := getBoard(t, maya, req)
				walked = append(walked, boardRows(got.Rows)...)
				if got.NextCursor == "" {
					break
				}
				req.Cursor = got.NextCursor
			}

			if !slices.Equal(walked, want) {
				t.Errorf("walked %v, want %v", walked, want)
			}
		})
	}
}

// TestMyRowRidesEveryPage: the caller's own row comes back on every page, ranked on the whole
// board, even when the page doesn't show it; a player with nothing in the range has none.
func TestMyRowRidesEveryPage(t *testing.T) {
	t.Parallel()
	for _, k := range boardKinds {
		t.Run(k.name, func(t *testing.T) {
			t.Parallel()
			srv := boardStack(t, wednesday)
			maya := boardPlayer(t, srv, "test_maya")
			eve := boardPlayer(t, srv, "test_eve")
			for handle, n := range map[string]int{"test_ann": 3, "test_bob": 2} {
				k.give(srv, t, handle, n, wednesday.AddDate(0, 0, -2))
			}
			k.give(srv, t, "test_maya", 1, wednesday.AddDate(0, 0, -2))

			first := getBoard(t, maya, &boardv1.GetBoardRequest{Board: k.board, Range: thisWeek, Limit: 1})
			second := getBoard(t, maya, &boardv1.GetBoardRequest{
				Board: k.board, Range: thisWeek, Limit: 1, Cursor: first.NextCursor,
			})
			evesFirst := getBoard(t, eve, &boardv1.GetBoardRequest{Board: k.board, Range: thisWeek, Limit: 1})

			for name, page := range map[string]*boardv1.GetBoardResponse{"page 1": first, "page 2": second} {
				mine := boardRows([]*boardv1.Row{page.MyRow})
				if page.MyRow == nil || !slices.Equal(mine, []string{"3 test_maya 1"}) {
					t.Errorf("%s: my_row = %v, want 3 test_maya 1", name, mine)
				}
			}
			if evesFirst.MyRow != nil {
				t.Errorf("eve's my_row = %v, want none: she has nothing this week", evesFirst.MyRow)
			}
		})
	}
}

// TestTheWeekStartsMondayAtMidnightCampusTime: something earned at Sunday 23:59 counts for the
// week it ends, and at Monday 00:00 the week starts fresh. All time counts both.
func TestTheWeekStartsMondayAtMidnightCampusTime(t *testing.T) {
	t.Parallel()
	sunday := time.Date(2026, 10, 25, 23, 59, 0, 0, points.Campus())
	monday := time.Date(2026, 10, 26, 0, 0, 0, 0, points.Campus())
	cases := []struct {
		name string
		now  time.Time
		rng  boardv1.Range
		want []string
	}{
		{"Sunday night, this week", sunday.Add(30 * time.Second), thisWeek, []string{"1 test_sun 1"}},
		{"Monday morning, this week", monday.Add(time.Minute), thisWeek, []string{"1 test_mon 1"}},
		{"Monday morning, all time", monday.Add(time.Minute), allTime, []string{"1 test_mon 1", "1 test_sun 1"}},
	}
	for _, k := range boardKinds {
		for _, c := range cases {
			t.Run(k.name+", "+c.name, func(t *testing.T) {
				t.Parallel()
				srv := boardStack(t, c.now)
				maya := boardPlayer(t, srv, "test_maya")
				k.give(srv, t, "test_sun", 1, sunday)
				if c.now.After(monday) {
					k.give(srv, t, "test_mon", 1, monday)
				}

				got := getBoard(t, maya, &boardv1.GetBoardRequest{Board: k.board, Range: c.rng})

				want := crowned(k.crowns, c.want...)
				rows := boardRows(got.Rows)
				if !slices.Equal(rows, want) {
					t.Errorf("rows = %v, want %v", rows, want)
				}
			})
		}
	}
}

// TestAnEmptyWeekIsAnEmptyBoard: before anyone earns anything this week, the board is empty, with
// no cursor and no row of the caller's, though last week still counts all time.
func TestAnEmptyWeekIsAnEmptyBoard(t *testing.T) {
	t.Parallel()
	for _, k := range boardKinds {
		t.Run(k.name, func(t *testing.T) {
			t.Parallel()
			srv := boardStack(t, wednesday)
			maya := boardPlayer(t, srv, "test_maya")
			k.give(srv, t, "test_maya", 2, wednesday.AddDate(0, 0, -7))

			week := getBoard(t, maya, &boardv1.GetBoardRequest{Board: k.board, Range: thisWeek})
			ever := getBoard(t, maya, &boardv1.GetBoardRequest{Board: k.board, Range: allTime})

			if len(week.Rows) != 0 || week.NextCursor != "" || week.MyRow != nil {
				t.Errorf("this week = %v, want an empty board", week)
			}
			rows := boardRows(ever.Rows)
			if !slices.Equal(rows, crowned(k.crowns, "1 test_maya 2")) {
				t.Errorf("all time = %v, want test_maya's 2", rows)
			}
		})
	}
}

// TestABadBoardRequestIsRefusedFieldByField: each problem comes back on its own field.
func TestABadBoardRequestIsRefusedFieldByField(t *testing.T) {
	t.Parallel()
	srv := boardStack(t, wednesday)
	maya := boardPlayer(t, srv, "test_maya")
	cases := []struct {
		name string
		req  *boardv1.GetBoardRequest
		want map[string]string
	}{
		{
			"no board and no range",
			&boardv1.GetBoardRequest{},
			map[string]string{"board": "board must be chefs or hall_of_fries", "range": "range must be this_week or all_time"},
		},
		{
			"a board this server does not know",
			&boardv1.GetBoardRequest{Board: boardv1.Board(9), Range: thisWeek},
			map[string]string{"board": "board must be chefs or hall_of_fries"},
		},
		{
			"a cursor this server never gave out",
			&boardv1.GetBoardRequest{Board: hallOfFries, Range: thisWeek, Cursor: "!!!"},
			map[string]string{"cursor": "cursor is not one this server gave out"},
		},
		{
			"a negative limit",
			&boardv1.GetBoardRequest{Board: hallOfFries, Range: thisWeek, Limit: -1},
			map[string]string{"limit": "limit must be at least 0"},
		},
	}
	for _, c := range cases {
		_, err := maya.GetBoard(t.Context(), connect.NewRequest(c.req))

		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Fatalf("%s: got code %v (%v), want invalid_argument", c.name, connect.CodeOf(err), err)
		}
		got := fieldViolations(t, err)
		if !maps.Equal(got, c.want) {
			t.Errorf("%s: violations = %v, want %v", c.name, got, c.want)
		}
	}
}

// TestTheFridgeOfTheWeekIsTheWeeksBestScan: the week's highest score comes back with whose it was
// and its roast, and last week's better scan does not count.
func TestTheFridgeOfTheWeekIsTheWeeksBestScan(t *testing.T) {
	t.Parallel()
	srv := boardStack(t, wednesday)
	maya := boardPlayer(t, srv, "test_maya")
	srv.scan(t, "test_cal", 99, wednesday.AddDate(0, 0, -7))
	srv.scan(t, "test_ben", 91, wednesday.AddDate(0, 0, -1))
	srv.scan(t, "test_dee", 64, wednesday.Add(-time.Hour))

	got := getFridge(t, maya)

	if got.Fridge == nil || fridgeLine(got.Fridge) != "test_ben 🍟 91 a 91 fridge" {
		t.Errorf("fridge = %v, want test_ben's 91 with its roast", got.Fridge)
	}
}

// TestTheFridgeOfTheWeekStartsFreshMondayAtMidnight: a scan at Sunday 23:59 campus time is the
// fridge until the week ends, and from Monday 00:00 there is none until someone scans.
func TestTheFridgeOfTheWeekStartsFreshMondayAtMidnight(t *testing.T) {
	t.Parallel()
	sunday := time.Date(2026, 10, 25, 23, 59, 0, 0, points.Campus())
	monday := time.Date(2026, 10, 26, 0, 0, 0, 0, points.Campus())
	cases := []struct {
		name string
		now  time.Time
		want string // "" for no fridge
	}{
		{"Sunday night", sunday.Add(30 * time.Second), "test_sun 🍟 70 a 70 fridge"},
		{"Monday morning", monday.Add(time.Minute), ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			srv := boardStack(t, c.now)
			maya := boardPlayer(t, srv, "test_maya")
			srv.scan(t, "test_sun", 70, sunday)

			got := getFridge(t, maya)

			if fridgeLine(got.Fridge) != c.want {
				t.Errorf("fridge = %v, want %q", got.Fridge, c.want)
			}
		})
	}
}

// TestTheBoardNeedsAnAccount: a fresh install gets unauthenticated from every board RPC, which the
// app reads as "back to onboarding".
func TestTheBoardNeedsAnAccount(t *testing.T) {
	t.Parallel()
	srv := boardStack(t, wednesday)
	stranger := boardNewcomer(t, srv)
	calls := map[string]func() error{
		"GetBoard": func() error {
			_, err := stranger.GetBoard(t.Context(), connect.NewRequest(&boardv1.GetBoardRequest{Board: hallOfFries, Range: thisWeek}))
			return err
		},
		"GetFridgeOfTheWeek": func() error {
			_, err := stranger.GetFridgeOfTheWeek(t.Context(), connect.NewRequest(&boardv1.GetFridgeOfTheWeekRequest{}))
			return err
		},
	}

	for name, call := range calls {
		err := call()
		if connect.CodeOf(err) != connect.CodeUnauthenticated {
			t.Errorf("%s: got code %v (%v), want unauthenticated", name, connect.CodeOf(err), err)
		}
	}
}

// boardServer is AccountService and BoardService behind the session middleware, over the test's
// own database, with the board's clock stopped at now.
type boardServer struct {
	url  string
	pool *pgxpool.Pool
	repo *repository.Repository
}

func boardStack(t *testing.T, now time.Time) boardServer {
	t.Helper()
	pool := newPool(t)
	repo := repository.New(pool)
	sessions := scs.New()
	// The cleanup goroutine is off: the test's database is gone before it would first run.
	sessions.Store = pgxstore.NewWithCleanupInterval(pool, 0)
	sessions.Lifetime = auth.Lifetime
	// httptest serves plain http, which a Secure cookie never comes back over.
	sessions.Cookie.Secure = false
	authn := auth.New(sessions, repo)
	mux := http.NewServeMux()
	mux.Handle(accountv1connect.NewAccountServiceHandler(account.NewAccount(account.Config{Repo: repo, Auth: authn})))
	mux.Handle(boardv1connect.NewBoardServiceHandler(board.NewBoard(board.Config{
		Repo: repo,
		Auth: authn,
		Now:  func() time.Time { return now },
	})))
	srv := httptest.NewServer(sessions.LoadAndSave(mux))
	t.Cleanup(srv.Close)
	return boardServer{url: srv.URL, pool: pool, repo: repo}
}

// runs logs n fry runs for handle, one a day from start, making the account if it is new. Each
// falls in its own meal window, so each one counts.
func (s boardServer) runs(t *testing.T, handle string, n int, start time.Time) {
	t.Helper()
	accountID := s.account(t, handle)
	for i := range n {
		at := start.AddDate(0, 0, i)
		window, day := points.MealWindow(at)
		_, err := s.repo.LogFryRun(t.Context(), accountID, repository.FryRun{
			SpotID:     "mock-03",
			SpotName:   "Night Owl Fries",
			MealDay:    day,
			MealWindow: string(window),
			At:         at,
		})
		if err != nil {
			t.Fatalf("log fry run %d for %s: %v", i, handle, err)
		}
	}
}

// cooks gives handle one plate worth n points, plated at at, making the account if it is new. The
// plate's cook credit sits in at's own meal window.
func (s boardServer) cooks(t *testing.T, handle string, n int, at time.Time) {
	t.Helper()
	accountID := s.account(t, handle)
	made, err := s.repo.CreateScan(t.Context(), accountID, 70, "a fridge with dinner in it", nil)
	if err != nil {
		t.Fatalf("create scan for %s: %v", handle, err)
	}
	_, err = s.repo.CreateRecipes(t.Context(), accountID, made.ScanID, made.ItemsVersion, []repository.NewRecipe{{
		Title: "Egg fried rice", Minutes: 15, Difficulty: repository.DifficultyEasy,
		Uses: []string{"Eggs", "Rice"}, Steps: []string{"Fry the rice, then the eggs."},
	}})
	if err != nil {
		t.Fatalf("create a recipe for %s: %v", handle, err)
	}
	recipes, err := s.repo.ListRecipes(t.Context(), accountID, made.ScanID, made.ItemsVersion, repository.RecipeFilter{})
	if err != nil || len(recipes) != 1 {
		t.Fatalf("list %s's recipe: %v (%d recipes)", handle, err, len(recipes))
	}
	photo := handle + " " + at.Format(time.RFC3339)
	plateID, err := s.repo.CreatePlate(t.Context(), accountID, repository.NewPlate{
		RecipeID: recipes[0].RecipeID, Photo: platePhoto(photo), PhotoSHA256: platePhotoHash(photo), At: at,
	})
	if err != nil {
		t.Fatalf("plate for %s: %v", handle, err)
	}
	window, day := points.MealWindow(at)
	err = s.repo.AwardPoints(t.Context(), accountID, plateID, day.Format(time.DateOnly)+" "+string(window), at,
		[]repository.Award{{Reason: repository.ReasonCook, Points: n}})
	if err != nil {
		t.Fatalf("award %s's points: %v", handle, err)
	}
}

// scan stores a scan of score for handle, with a roast that names the score, made at at. It makes
// the account if it is new.
func (s boardServer) scan(t *testing.T, handle string, score int, at time.Time) {
	t.Helper()
	accountID := s.account(t, handle)
	created, err := s.repo.CreateScan(t.Context(), accountID, score, fmt.Sprintf("a %d fridge", score), nil)
	if err != nil {
		t.Fatalf("create scan for %s: %v", handle, err)
	}
	_, err = s.pool.Exec(t.Context(), `update app.scan set created_at = $2 where scan_id = $1`, created.ScanID, at)
	if err != nil {
		t.Fatalf("stamp scan %d: %v", created.ScanID, err)
	}
}

// account is handle's account id, making the account if it is new.
func (s boardServer) account(t *testing.T, handle string) int64 {
	t.Helper()
	var accountID int64
	err := s.pool.QueryRow(t.Context(), `select account_id from app.account where handle = $1`, handle).Scan(&accountID)
	if err == nil {
		return accountID
	}
	created, err := s.repo.CreateAccount(t.Context(), handle, "🍟")
	if err != nil {
		t.Fatalf("create account %s: %v", handle, err)
	}
	return created.AccountID
}

func boardPlayer(t *testing.T, srv boardServer, handle string) boardv1connect.BoardServiceClient {
	t.Helper()
	p := player(t, srv.url, handle)
	return boardv1connect.NewBoardServiceClient(&http.Client{Jar: p.jar}, srv.url)
}

func boardNewcomer(t *testing.T, srv boardServer) boardv1connect.BoardServiceClient {
	t.Helper()
	p := newcomer(t, srv.url)
	return boardv1connect.NewBoardServiceClient(&http.Client{Jar: p.jar}, srv.url)
}

func getBoard(t *testing.T, c boardv1connect.BoardServiceClient, req *boardv1.GetBoardRequest) *boardv1.GetBoardResponse {
	t.Helper()
	resp, err := c.GetBoard(t.Context(), connect.NewRequest(req))
	if err != nil {
		t.Fatalf("get board: %v", err)
	}
	return resp.Msg
}

// crowned adds the crown to the rank-1 rows when the board crowns its leaders.
func crowned(crowns bool, rows ...string) []string {
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		if crowns && strings.HasPrefix(r, "1 ") {
			r += " crown"
		}
		out = append(out, r)
	}
	return out
}

// boardRows writes rows as "rank handle value", with "crown" after a crowned one.
func boardRows(rows []*boardv1.Row) []string {
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		line := fmt.Sprintf("%d %s %d", r.GetRank(), r.GetHandle(), r.GetValue())
		if r.GetCrown() {
			line += " crown"
		}
		out = append(out, line)
	}
	return out
}

func getFridge(t *testing.T, c boardv1connect.BoardServiceClient) *boardv1.GetFridgeOfTheWeekResponse {
	t.Helper()
	resp, err := c.GetFridgeOfTheWeek(t.Context(), connect.NewRequest(&boardv1.GetFridgeOfTheWeekRequest{}))
	if err != nil {
		t.Fatalf("get fridge of the week: %v", err)
	}
	return resp.Msg
}

// fridgeLine writes a fridge as "handle avatar score roast", and no fridge as "".
func fridgeLine(f *boardv1.FridgeOfTheWeek) string {
	if f == nil {
		return ""
	}
	return fmt.Sprintf("%s %s %d %s", f.GetHandle(), f.GetAvatar(), f.GetScore(), f.GetRoast())
}
