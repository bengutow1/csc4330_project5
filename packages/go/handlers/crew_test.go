// Author: Ben Gutowski
package handlers_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/alexedwards/scs/pgxstore"
	"github.com/alexedwards/scs/v2"

	"github.com/bengutow1/csc4330_project5/packages/go/apigen/account/v1/accountv1connect"
	boardv1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/board/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/apigen/board/v1/boardv1connect"
	crewv1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/crew/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/apigen/crew/v1/crewv1connect"
	"github.com/bengutow1/csc4330_project5/packages/go/auth"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers/account"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers/board"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers/crew"
	"github.com/bengutow1/csc4330_project5/packages/go/repository"
)

// sixDigits is a crew code's shape.
var sixDigits = regexp.MustCompile(`^[0-9]{6}$`)

// TestMakingACrewGivesItsMakerASixDigitCode: the maker is its first member, and the code is the
// six digits a friend types to join.
func TestMakingACrewGivesItsMakerASixDigitCode(t *testing.T) {
	t.Parallel()
	srv := crewStack(t, crew.Limits{})
	maya := crewPlayer(t, srv, "test_maya")

	made := createCrew(t, maya, "  Dorm 4B  ")

	if made.GetName() != "Dorm 4B" || made.GetMemberCount() != 1 || made.GetCrewId() <= 0 {
		t.Errorf("made %v, want Dorm 4B (trimmed) with 1 member and an id", made)
	}
	if !sixDigits.MatchString(made.GetCode()) {
		t.Errorf("code = %q, want six digits", made.GetCode())
	}
	if got := myCrews(t, maya); !slices.Equal(got, []string{"Dorm 4B 1"}) {
		t.Errorf("maya's crews = %v, want [Dorm 4B 1]", got)
	}
}

// TestJoiningACrewByItsCodePutsItOnTheJoinersList: a friend with the code is in, counted once
// however often they tap, and spaces typed in the code do not matter.
func TestJoiningACrewByItsCodePutsItOnTheJoinersList(t *testing.T) {
	t.Parallel()
	srv := crewStack(t, crew.Limits{})
	maya := crewPlayer(t, srv, "test_maya")
	ben := crewPlayer(t, srv, "test_ben")
	made := createCrew(t, maya, "Dorm 4B")
	spaced := made.GetCode()[:3] + " " + made.GetCode()[3:]

	for try := range 2 {
		resp, err := ben.crews.JoinCrew(t.Context(), connect.NewRequest(&crewv1.JoinCrewRequest{Code: spaced}))
		if err != nil {
			t.Fatalf("join %d: %v", try, err)
		}
		if resp.Msg.GetCrew().GetCrewId() != made.GetCrewId() || resp.Msg.GetCrew().GetMemberCount() != 2 {
			t.Errorf("join %d answered %v, want crew %d with 2 members", try, resp.Msg.GetCrew(), made.GetCrewId())
		}
	}
	for name, p := range map[string]crewPerson{"maya": maya, "ben": ben} {
		if got := myCrews(t, p); !slices.Equal(got, []string{"Dorm 4B 2"}) {
			t.Errorf("%s's crews = %v, want [Dorm 4B 2]", name, got)
		}
	}
}

// TestAWrongCodeIsNotFoundInWordsAndJoinsNothing: a mistyped code must say so in words the join
// screen can show, and add no crew.
func TestAWrongCodeIsNotFoundInWordsAndJoinsNothing(t *testing.T) {
	t.Parallel()
	srv := crewStack(t, crew.Limits{})
	maya := crewPlayer(t, srv, "test_maya")
	ben := crewPlayer(t, srv, "test_ben")
	made := createCrew(t, maya, "Dorm 4B")
	wrong := "000000"
	if made.GetCode() == wrong {
		wrong = "000001"
	}

	_, err := ben.crews.JoinCrew(t.Context(), connect.NewRequest(&crewv1.JoinCrewRequest{Code: wrong}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("got code %v (%v), want not_found", connect.CodeOf(err), err)
	}
	if msg := asConnectErr(t, err).Message(); msg != repository.ErrNoCrewForCode.Error() {
		t.Errorf("message = %q, want %q", msg, repository.ErrNoCrewForCode.Error())
	}
	if got := myCrews(t, ben); len(got) != 0 {
		t.Errorf("ben's crews = %v, want none", got)
	}
}

// TestACrewBoardRanksItsMembersAmongThemselves: a crew's board shows its members alone, ranked
// among themselves, with the caller's own row ranked the same way; the campus board is unchanged.
func TestACrewBoardRanksItsMembersAmongThemselves(t *testing.T) {
	t.Parallel()
	for _, k := range boardKinds {
		t.Run(k.name, func(t *testing.T) {
			t.Parallel()
			srv := crewStack(t, crew.Limits{})
			maya := crewPlayer(t, srv, "test_maya")
			ben := crewPlayer(t, srv, "test_ben")
			made := createCrew(t, maya, "Dorm 4B")
			joinCrew(t, ben, made.GetCode())
			k.give(srv.boardServer, t, "test_cal", 5, wednesday.AddDate(0, 0, -2)) // not in the crew
			k.give(srv.boardServer, t, "test_ben", 3, wednesday.AddDate(0, 0, -2))
			k.give(srv.boardServer, t, "test_maya", 2, wednesday.AddDate(0, 0, -2))

			crewBoard := getBoard(t, maya.board, &boardv1.GetBoardRequest{
				Board: k.board, Range: thisWeek, CrewId: made.GetCrewId(),
			})
			campus := getBoard(t, maya.board, &boardv1.GetBoardRequest{Board: k.board, Range: thisWeek})

			if rows := boardRows(crewBoard.Rows); !slices.Equal(rows, crowned(k.crowns, "1 test_ben 3", "2 test_maya 2")) {
				t.Errorf("crew board = %v, want ben then maya, cal left out", rows)
			}
			if mine := boardRows([]*boardv1.Row{crewBoard.MyRow}); !slices.Equal(mine, []string{"2 test_maya 2"}) {
				t.Errorf("my row on the crew board = %v, want 2 test_maya 2", mine)
			}
			want := crowned(k.crowns, "1 test_cal 5", "2 test_ben 3", "3 test_maya 2")
			if rows := boardRows(campus.Rows); !slices.Equal(rows, want) {
				t.Errorf("campus board = %v, want %v", rows, want)
			}
		})
	}
}

// TestACrewBoardIsNotFoundToANonMember: a crew's board is its members' alone, and a crew someone
// is not in answers exactly like one that never was, so a crew id is no way in.
func TestACrewBoardIsNotFoundToANonMember(t *testing.T) {
	t.Parallel()
	srv := crewStack(t, crew.Limits{})
	maya := crewPlayer(t, srv, "test_maya")
	eve := crewPlayer(t, srv, "test_eve")
	made := createCrew(t, maya, "Dorm 4B")

	for name, crewID := range map[string]int64{"maya's crew": made.GetCrewId(), "no crew": made.GetCrewId() + 1000} {
		_, err := eve.board.GetBoard(t.Context(), connect.NewRequest(&boardv1.GetBoardRequest{
			Board: chefs, Range: thisWeek, CrewId: crewID,
		}))
		if connect.CodeOf(err) != connect.CodeNotFound {
			t.Errorf("%s: got code %v (%v), want not_found", name, connect.CodeOf(err), err)
		}
	}
	// And maya's crew is untouched: still just her.
	if got := myCrews(t, maya); !slices.Equal(got, []string{"Dorm 4B 1"}) {
		t.Errorf("maya's crews = %v, want [Dorm 4B 1]", got)
	}
}

// TestLeavingTakesTheCrewOffYourListAndYourBoard: after leaving, the crew is gone from the
// player's list and its board is closed to them, the others keep it, and leaving twice is
// not_found.
func TestLeavingTakesTheCrewOffYourListAndYourBoard(t *testing.T) {
	t.Parallel()
	srv := crewStack(t, crew.Limits{})
	maya := crewPlayer(t, srv, "test_maya")
	ben := crewPlayer(t, srv, "test_ben")
	made := createCrew(t, maya, "Dorm 4B")
	joinCrew(t, ben, made.GetCode())

	leave := &crewv1.LeaveCrewRequest{CrewId: made.GetCrewId()}
	_, err := ben.crews.LeaveCrew(t.Context(), connect.NewRequest(leave))
	if err != nil {
		t.Fatalf("ben leaves: %v", err)
	}

	if got := myCrews(t, ben); len(got) != 0 {
		t.Errorf("ben's crews = %v, want none", got)
	}
	if got := myCrews(t, maya); !slices.Equal(got, []string{"Dorm 4B 1"}) {
		t.Errorf("maya's crews = %v, want [Dorm 4B 1]", got)
	}
	_, err = ben.board.GetBoard(t.Context(), connect.NewRequest(&boardv1.GetBoardRequest{
		Board: chefs, Range: thisWeek, CrewId: made.GetCrewId(),
	}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("ben's view of the crew board: got code %v (%v), want not_found", connect.CodeOf(err), err)
	}
	_, err = ben.crews.LeaveCrew(t.Context(), connect.NewRequest(leave))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("leaving again: got code %v (%v), want not_found", connect.CodeOf(err), err)
	}
}

// TestAPlayerAtTheCapIsToldToLeaveOne: the eleventh crew is refused in words that say what to do.
func TestAPlayerAtTheCapIsToldToLeaveOne(t *testing.T) {
	t.Parallel()
	srv := crewStack(t, crew.Limits{})
	maya := crewPlayer(t, srv, "test_maya")
	for i := range 10 {
		createCrew(t, maya, fmt.Sprintf("Crew %d", i))
	}

	_, err := maya.crews.CreateCrew(t.Context(), connect.NewRequest(&crewv1.CreateCrewRequest{Name: "One more"}))
	if connect.CodeOf(err) != connect.CodeResourceExhausted {
		t.Fatalf("got code %v (%v), want resource_exhausted", connect.CodeOf(err), err)
	}
	if msg := asConnectErr(t, err).Message(); msg != repository.ErrTooManyCrews.Error() {
		t.Errorf("message = %q, want %q", msg, repository.ErrTooManyCrews.Error())
	}
}

// TestABadCrewRequestIsRefusedFieldByField: each bad field is named, keyed by its proto name, so
// the form shows the message under the right input.
func TestABadCrewRequestIsRefusedFieldByField(t *testing.T) {
	t.Parallel()
	srv := crewStack(t, crew.Limits{})
	maya := crewPlayer(t, srv, "test_maya")
	cases := []struct {
		name  string
		call  func() error
		field string
	}{
		{name: "blank name", field: "name", call: func() error {
			_, err := maya.crews.CreateCrew(t.Context(), connect.NewRequest(&crewv1.CreateCrewRequest{Name: "   "}))
			return err
		}},
		{name: "long name", field: "name", call: func() error {
			_, err := maya.crews.CreateCrew(t.Context(), connect.NewRequest(&crewv1.CreateCrewRequest{Name: strings.Repeat("a", 31)}))
			return err
		}},
		{name: "letters in the code", field: "code", call: func() error {
			_, err := maya.crews.JoinCrew(t.Context(), connect.NewRequest(&crewv1.JoinCrewRequest{Code: "12a456"}))
			return err
		}},
		{name: "short code", field: "code", call: func() error {
			_, err := maya.crews.JoinCrew(t.Context(), connect.NewRequest(&crewv1.JoinCrewRequest{Code: "12345"}))
			return err
		}},
		{name: "no crew to leave", field: "crew_id", call: func() error {
			_, err := maya.crews.LeaveCrew(t.Context(), connect.NewRequest(&crewv1.LeaveCrewRequest{}))
			return err
		}},
		{name: "a negative crew board", field: "crew_id", call: func() error {
			_, err := maya.board.GetBoard(t.Context(), connect.NewRequest(&boardv1.GetBoardRequest{
				Board: chefs, Range: thisWeek, CrewId: -1,
			}))
			return err
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			err := c.call()
			if connect.CodeOf(err) != connect.CodeInvalidArgument {
				t.Fatalf("got code %v (%v), want invalid_argument", connect.CodeOf(err), err)
			}
			if _, ok := fieldViolations(t, err)[c.field]; !ok {
				t.Errorf("violations = %v, want one on %s", fieldViolations(t, err), c.field)
			}
		})
	}
}

// TestJoiningIsLimitedPerClientIP: six digits are cheap to guess, so one address gets only so many
// tries, wrong codes included.
func TestJoiningIsLimitedPerClientIP(t *testing.T) {
	t.Parallel()
	srv := crewStack(t, crew.Limits{Join: handlers.NewLimiter(2, time.Hour)})
	ben := crewPlayer(t, srv, "test_ben")

	for try := range 2 {
		_, err := ben.crews.JoinCrew(t.Context(), connect.NewRequest(&crewv1.JoinCrewRequest{Code: "000000"}))
		if connect.CodeOf(err) != connect.CodeNotFound {
			t.Fatalf("try %d: got code %v (%v), want not_found", try, connect.CodeOf(err), err)
		}
	}
	_, err := ben.crews.JoinCrew(t.Context(), connect.NewRequest(&crewv1.JoinCrewRequest{Code: "000000"}))
	if connect.CodeOf(err) != connect.CodeResourceExhausted {
		t.Errorf("third try: got code %v (%v), want resource_exhausted", connect.CodeOf(err), err)
	}
}

// TestCrewsNeedAnAccount: every crew rpc is for a signed-in player, so a fresh install is sent to
// onboarding.
func TestCrewsNeedAnAccount(t *testing.T) {
	t.Parallel()
	srv := crewStack(t, crew.Limits{})
	p := newcomer(t, srv.url)
	crews := crewv1connect.NewCrewServiceClient(&http.Client{Jar: p.jar}, srv.url)
	calls := map[string]func() error{
		"create": func() error {
			_, err := crews.CreateCrew(t.Context(), connect.NewRequest(&crewv1.CreateCrewRequest{Name: "Dorm 4B"}))
			return err
		},
		"join": func() error {
			_, err := crews.JoinCrew(t.Context(), connect.NewRequest(&crewv1.JoinCrewRequest{Code: "123456"}))
			return err
		},
		"leave": func() error {
			_, err := crews.LeaveCrew(t.Context(), connect.NewRequest(&crewv1.LeaveCrewRequest{CrewId: 1}))
			return err
		},
		"list": func() error {
			_, err := crews.ListMyCrews(t.Context(), connect.NewRequest(&crewv1.ListMyCrewsRequest{}))
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

// crewServer is the board's test server with the crew service mounted beside it.
type crewServer struct {
	boardServer
}

// crewStack serves accounts, crews and the board the way main.go does, on wednesday, with
// joining limited by limits.
func crewStack(t *testing.T, limits crew.Limits) crewServer {
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
	mux.Handle(crewv1connect.NewCrewServiceHandler(crew.NewCrew(crew.Config{Repo: repo, Auth: authn, Limits: limits})))
	mux.Handle(boardv1connect.NewBoardServiceHandler(board.NewBoard(board.Config{
		Repo: repo,
		Auth: authn,
		Now:  func() time.Time { return wednesday },
	})))
	srv := httptest.NewServer(sessions.LoadAndSave(mux))
	t.Cleanup(srv.Close)
	return crewServer{boardServer{url: srv.URL, pool: pool, repo: repo}}
}

// crewPerson is one signed-in player, with a client for crews and one for the board.
type crewPerson struct {
	crews crewv1connect.CrewServiceClient
	board boardv1connect.BoardServiceClient
}

func crewPlayer(t *testing.T, srv crewServer, handle string) crewPerson {
	t.Helper()
	p := player(t, srv.url, handle)
	client := &http.Client{Jar: p.jar}
	return crewPerson{
		crews: crewv1connect.NewCrewServiceClient(client, srv.url),
		board: boardv1connect.NewBoardServiceClient(client, srv.url),
	}
}

func createCrew(t *testing.T, p crewPerson, name string) *crewv1.Crew {
	t.Helper()
	resp, err := p.crews.CreateCrew(t.Context(), connect.NewRequest(&crewv1.CreateCrewRequest{Name: name}))
	if err != nil {
		t.Fatalf("create crew %s: %v", name, err)
	}
	return resp.Msg.GetCrew()
}

func joinCrew(t *testing.T, p crewPerson, code string) {
	t.Helper()
	_, err := p.crews.JoinCrew(t.Context(), connect.NewRequest(&crewv1.JoinCrewRequest{Code: code}))
	if err != nil {
		t.Fatalf("join crew %s: %v", code, err)
	}
}

// myCrews writes the player's crews as "name members", in the order the server gave.
func myCrews(t *testing.T, p crewPerson) []string {
	t.Helper()
	resp, err := p.crews.ListMyCrews(t.Context(), connect.NewRequest(&crewv1.ListMyCrewsRequest{}))
	if err != nil {
		t.Fatalf("list my crews: %v", err)
	}
	out := make([]string, 0, len(resp.Msg.GetCrews()))
	for _, c := range resp.Msg.GetCrews() {
		out = append(out, fmt.Sprintf("%s %d", c.GetName(), c.GetMemberCount()))
	}
	return out
}
