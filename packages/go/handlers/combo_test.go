// Author: LAPTOP-BPMQJP10\andyk
package handlers_test

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"

	"github.com/bengutow1/csc4330_project5/packages/go/apigen/account/v1/accountv1connect"
	combov1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/combo/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/apigen/combo/v1/combov1connect"
	scanv1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/scan/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/apigen/scan/v1/scanv1connect"
	"github.com/bengutow1/csc4330_project5/packages/go/auth"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers/account"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers/combo"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers/scan"
	"github.com/bengutow1/csc4330_project5/packages/go/repository"
	"github.com/bengutow1/csc4330_project5/packages/go/vision"
)

// TestComboMergesBothFridgesWithoutCountingDuplicateFoodsTwice pins provenance and server scoring.
func TestComboMergesBothFridgesWithoutCountingDuplicateFoodsTwice(t *testing.T) {
	t.Parallel()
	eyes := newFakeVision(vision.Fridge{Items: []vision.Item{{Name: "Eggs", Category: "protein"}, {Name: "Peas", Category: "vegetable"}}})
	srv := comboStack(t, combo.Config{}, eyes)
	host, guest := comboPlayer(t, srv, "test_host"), comboPlayer(t, srv, "test_guest")
	c := startCombo(t, host)
	if c.HostId <= 0 || c.Players[0].Handle != "test_host" || c.State != combov1.ComboState_COMBO_STATE_OPEN || c.Verdict != nil {
		t.Fatalf("started = %v, want open, host only, no verdict", c)
	}
	if c.Code != "000001" || !c.ExpiresAt.AsTime().Equal(comboNow().Add(2*time.Hour)) {
		t.Errorf("code/expiry = %s/%v", c.Code, c.ExpiresAt)
	}
	joinCombo(t, guest, " "+c.Code+" ")
	one := comboScan(t, host, c.ComboId)
	if getCombo(t, guest, c.ComboId).Verdict != nil {
		t.Error("combined verdict exists before both scans")
	}
	two := comboScan(t, guest, c.ComboId)
	got := getCombo(t, host, c.ComboId)
	if !proto.Equal(got, getCombo(t, guest, c.ComboId)) {
		t.Error("phones disagree on combined verdict")
	}
	if got.Verdict.GetScore() != 58 || got.Verdict.GetTier() != scanv1.Tier_TIER_SCRAPPY {
		t.Errorf("combined = %v, want score 58, scrappy", got.Verdict)
	}
	if len(got.Verdict.GetIngredients()) != 4 {
		t.Fatalf("owned ingredients = %v, want both lists", got.Verdict)
	}
	for i, ingredient := range got.Verdict.Ingredients {
		owner := got.Players[i/2].AccountId
		if ingredient.AccountId != owner {
			t.Errorf("ingredient %d owner = %d, want %d", i, ingredient.AccountId, owner)
		}
	}
	if !proto.Equal(got.Players[0].Scan, one) || !proto.Equal(got.Players[1].Scan, two) {
		t.Error("member scan differs from stored verdict")
	}
	// A guest's removal hides their old food even if they rejoin under the same code.
	_, err := host.host.RemovePlayer(t.Context(), connect.NewRequest(&combov1.RemovePlayerRequest{ComboId: c.ComboId}))
	if err != nil {
		t.Fatalf("remove: %v", err)
	}
	_, err = guest.combo.GetCombo(t.Context(), connect.NewRequest(&combov1.GetComboRequest{ComboId: c.ComboId}))
	comboError(t, err, connect.CodeNotFound, "not found")
	joinCombo(t, guest, c.Code)
	after := getCombo(t, host, c.ComboId)
	if after.Verdict != nil || after.Players[1].Scan != nil {
		t.Errorf("old food survived removal: %v", after)
	}
	read, err := guest.scan.GetScan(t.Context(), connect.NewRequest(&scanv1.GetScanRequest{ScanId: two.ScanId}))
	if err != nil {
		t.Fatalf("read personal scan: %v", err)
	}
	if !proto.Equal(read.Msg.GetVerdict(), two) {
		t.Errorf("guest's personal scan lost: %v", read)
	}
	t.Logf("two phones: score=%d tier=%s owned_items=%d; removed guest retains personal scan", got.Verdict.Score, got.Verdict.Tier, len(got.Verdict.Ingredients))
}

// TestComboRefusalsKeepTheirCodesAndSentences checks wrong, expired, full and ended invitations.
func TestComboRefusalsKeepTheirCodesAndSentences(t *testing.T) {
	t.Parallel()
	srv := comboStack(t, combo.Config{}, newFakeVision(chefFridge))
	host, guest, extra := comboPlayer(t, srv, "test_host"), comboPlayer(t, srv, "test_guest"), comboPlayer(t, srv, "test_extra")
	c := startCombo(t, host)
	_, err := guest.combo.JoinCombo(t.Context(), connect.NewRequest(&combov1.JoinComboRequest{Code: "999999"}))
	comboError(t, err, connect.CodeNotFound, "no combo has that code")
	joinCombo(t, guest, c.Code)
	joinCombo(t, guest, c.Code)
	joinCombo(t, host, c.Code)
	_, err = extra.combo.JoinCombo(t.Context(), connect.NewRequest(&combov1.JoinComboRequest{Code: c.Code}))
	comboError(t, err, connect.CodeResourceExhausted, "that combo is full")
	_, err = host.host.EndCombo(t.Context(), connect.NewRequest(&combov1.EndComboRequest{ComboId: c.ComboId}))
	if err != nil {
		t.Fatalf("end: %v", err)
	}
	if getCombo(t, guest, c.ComboId).State != combov1.ComboState_COMBO_STATE_ENDED {
		t.Error("ended state not stored")
	}
	_, err = extra.combo.JoinCombo(t.Context(), connect.NewRequest(&combov1.JoinComboRequest{Code: c.Code}))
	comboError(t, err, connect.CodeFailedPrecondition, "that combo has already ended")
	_, err = host.scan.CreateScan(t.Context(), comboScanRequest(c.ComboId))
	comboError(t, err, connect.CodeFailedPrecondition, "that combo has already ended")

	for _, delta := range []time.Duration{-time.Microsecond, 0, time.Microsecond} {
		t.Run(fmt.Sprint(delta), func(t *testing.T) {
			t.Parallel()
			clock := func() time.Time { return comboNow().Add(2*time.Hour + delta) }
			s := comboStack(t, combo.Config{Now: clock}, newFakeVision(chefFridge))
			comboPlayer(t, s, "test_host")
			g := comboPlayer(t, s, "test_guest")
			var id int64
			err := s.pool.QueryRow(t.Context(), `select account_id from app.account where handle='test_host'`).Scan(&id)
			if err != nil {
				t.Fatalf("host id: %v", err)
			}
			_, err = repository.New(s.pool).CreateCombo(t.Context(), id, "012345", comboNow())
			if err != nil {
				t.Fatalf("setup invite: %v", err)
			}
			result, err := g.combo.JoinCombo(t.Context(), connect.NewRequest(&combov1.JoinComboRequest{Code: "012345"}))
			if delta < 0 {
				if err != nil {
					t.Fatalf("before expiry: %v", err)
				}
				if len(result.Msg.Combo.Players) != 2 {
					t.Errorf("join = %v", result.Msg)
				}
			} else {
				comboError(t, err, connect.CodeFailedPrecondition, "that code has expired, ask for a new one")
			}
		})
	}
}

// TestComboHostControlsAndEveryIDFence checks rights after visibility and proves refusals write nothing.
func TestComboHostControlsAndEveryIDFence(t *testing.T) {
	t.Parallel()
	srv := comboStack(t, combo.Config{}, newFakeVision(chefFridge))
	host, guest, outsider := comboPlayer(t, srv, "test_host"), comboPlayer(t, srv, "test_guest"), comboPlayer(t, srv, "test_outside")
	c := startCombo(t, host)
	joinCombo(t, guest, c.Code)
	before := getCombo(t, host, c.ComboId)
	for _, p := range []comboPhone{guest, outsider} {
		for _, id := range []int64{c.ComboId, c.ComboId + 1000} {
			want := connect.CodeNotFound
			message := "not found"
			if p.combo == guest.combo && id == c.ComboId {
				want = connect.CodePermissionDenied
				message = "not allowed"
			}
			_, err := p.host.EndCombo(t.Context(), connect.NewRequest(&combov1.EndComboRequest{ComboId: id}))
			comboError(t, err, want, message)
			_, err = p.host.RemovePlayer(t.Context(), connect.NewRequest(&combov1.RemovePlayerRequest{ComboId: id}))
			comboError(t, err, want, message)
		}
	}
	for _, id := range []int64{c.ComboId, c.ComboId + 1000} {
		_, err := outsider.combo.GetCombo(t.Context(), connect.NewRequest(&combov1.GetComboRequest{ComboId: id}))
		comboError(t, err, connect.CodeNotFound, "not found")
		_, err = outsider.scan.CreateScan(t.Context(), comboScanRequest(id))
		comboError(t, err, connect.CodeNotFound, "not found")
	}
	if !proto.Equal(before, getCombo(t, host, c.ComboId)) {
		t.Error("refused caller changed the combo")
	}
	if count(t, srv, `select count(*) from app.scan`) != 0 {
		t.Error("outsider scan was stored")
	}
}

// TestEveryComboRpcNeedsASessionAndValidInput prevents a new surface from forgetting its gate.
func TestEveryComboRpcNeedsASessionAndValidInput(t *testing.T) {
	t.Parallel()
	srv := comboStack(t, combo.Config{}, newFakeVision(chefFridge))
	anon := comboClients(http.DefaultClient, srv.url)
	for name, call := range comboCalls(anon, 0) {
		err := call(t.Context())
		comboError(t, err, connect.CodeUnauthenticated, "not signed in")
		t.Logf("%s without session: unauthenticated", name)
	}
	p := comboPlayer(t, srv, "test_player")
	for _, code := range []string{"", "12345", "1234567", "abcdef", "１２３４５６"} {
		_, err := p.combo.JoinCombo(t.Context(), connect.NewRequest(&combov1.JoinComboRequest{Code: code}))
		comboError(t, err, connect.CodeInvalidArgument, "enter the six-digit code")
		if fieldViolations(t, err)["code"] == "" {
			t.Errorf("code %q has no field violation", code)
		}
	}
	for name, call := range comboCalls(p, 0) {
		if name == "start" || name == "join" {
			continue
		}
		err := call(t.Context())
		if connect.CodeOf(err) != connect.CodeInvalidArgument || fieldViolations(t, err)["combo_id"] == "" {
			t.Errorf("%s: %v, want combo_id violation", name, err)
		}
	}
	_, err := p.scan.CreateScan(t.Context(), comboScanRequest(0))
	if connect.CodeOf(err) != connect.CodeInvalidArgument || fieldViolations(t, err)["combo_id"] == "" {
		t.Errorf("scan: %v, want combo_id violation", err)
	}
}

// TestTwentyPhonesRaceForTheLastComboSeat drives the guarded SQL through real generated clients.
func TestTwentyPhonesRaceForTheLastComboSeat(t *testing.T) {
	t.Parallel()
	srv := comboStack(t, combo.Config{}, newFakeVision(chefFridge))
	host := comboPlayer(t, srv, "test_host")
	c := startCombo(t, host)
	phones := make([]comboPhone, 20)
	for i := range phones {
		phones[i] = comboPlayer(t, srv, fmt.Sprintf("test_guest%d", i))
	}
	results := make(chan error, 20)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for _, p := range phones {
		wg.Go(func() {
			<-start
			_, err := p.combo.JoinCombo(t.Context(), connect.NewRequest(&combov1.JoinComboRequest{Code: c.Code}))
			results <- err
		})
	}
	close(start)
	wg.Wait()
	close(results)
	winners := 0
	for err := range results {
		if err == nil {
			winners++
		} else {
			comboError(t, err, connect.CodeResourceExhausted, "that combo is full")
		}
	}
	if winners != 1 || len(getCombo(t, host, c.ComboId).Players) != 2 {
		t.Errorf("winners = %d, want 1 with two members", winners)
	}
	t.Logf("20 simultaneous joins: winners=%d, refusals=%d", winners, 20-winners)
}

// TestComboGuessLimitIsSharedByClientIP prevents free accounts from resetting the invite limit.
func TestComboGuessLimitIsSharedByClientIP(t *testing.T) {
	t.Parallel()
	srv := comboStack(t, combo.Config{Limit: handlers.NewLimiter(1, time.Hour)}, newFakeVision(chefFridge))
	a, b := comboPlayer(t, srv, "test_one"), comboPlayer(t, srv, "test_two")
	_, err := a.combo.JoinCombo(t.Context(), connect.NewRequest(&combov1.JoinComboRequest{Code: "999999"}))
	comboError(t, err, connect.CodeNotFound, "no combo has that code")
	_, err = b.combo.JoinCombo(t.Context(), connect.NewRequest(&combov1.JoinComboRequest{Code: "999999"}))
	comboError(t, err, connect.CodeResourceExhausted, handlers.ErrTooManyRequests.Error())
}

// TestComboRemovalDuringVisionCannotAttachTheFinishedScan closes the check-before-vendor race.
func TestComboRemovalDuringVisionCannotAttachTheFinishedScan(t *testing.T) {
	t.Parallel()
	for _, action := range []string{"remove", "end"} {
		t.Run(action, func(t *testing.T) {
			t.Parallel()
			eyes := &waitingComboVision{entered: make(chan struct{}), release: make(chan struct{})}
			srv := comboStack(t, combo.Config{}, eyes)
			var release sync.Once
			t.Cleanup(func() { release.Do(func() { close(eyes.release) }) })
			host, guest := comboPlayer(t, srv, "test_host"), comboPlayer(t, srv, "test_guest")
			c := startCombo(t, host)
			joinCombo(t, guest, c.Code)
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			result := make(chan error, 1)
			go func() { _, err := guest.scan.CreateScan(ctx, comboScanRequest(c.ComboId)); result <- err }()
			select {
			case <-eyes.entered:
			case <-ctx.Done():
				t.Fatalf("vision never entered: %v", ctx.Err())
			}
			var err error
			if action == "remove" {
				_, err = host.host.RemovePlayer(ctx, connect.NewRequest(&combov1.RemovePlayerRequest{ComboId: c.ComboId}))
			} else {
				_, err = host.host.EndCombo(ctx, connect.NewRequest(&combov1.EndComboRequest{ComboId: c.ComboId}))
			}
			if err != nil {
				t.Fatalf("host action: %v", err)
			}
			release.Do(func() { close(eyes.release) })
			select {
			case err := <-result:
				if action == "remove" {
					comboError(t, err, connect.CodeNotFound, "not found")
				} else {
					comboError(t, err, connect.CodeFailedPrecondition, "that combo has already ended")
				}
			case <-ctx.Done():
				t.Fatalf("scan never finished: %v", ctx.Err())
			}
			if count(t, srv, `select count(*) from app.scan`) != 0 {
				t.Error("finished scan was stored after membership ended")
			}
		})
	}
}

// waitingComboVision is safe at the vendor edge; channels let the real host RPC finish mid-call.
type waitingComboVision struct {
	entered chan struct{}
	release chan struct{}
}

func (f *waitingComboVision) ReadFridge(ctx context.Context, _ [][]byte, _ vision.RoastLevel) (vision.Fridge, error) {
	close(f.entered)
	select {
	case <-f.release:
		return chefFridge, nil
	case <-ctx.Done():
		return vision.Fridge{}, ctx.Err()
	}
}

// TestComboStartRetriesACodeCollisionWithoutLeavingHalfACombo pins rollback before retry.
func TestComboStartRetriesACodeCollisionWithoutLeavingHalfACombo(t *testing.T) {
	t.Parallel()
	var calls atomic.Int64
	codes := func() (string, error) {
		if calls.Add(1) <= 2 {
			return "012345", nil
		}
		return "543210", nil
	}
	srv := comboStack(t, combo.Config{Code: codes}, newFakeVision(chefFridge))
	host := comboPlayer(t, srv, "test_host")
	first, second := startCombo(t, host), startCombo(t, host)
	if first.Code != "012345" || second.Code != "543210" || first.ComboId == second.ComboId {
		t.Errorf("collision retry returned wrong invites")
	}
	if count(t, srv, `select count(*) from app.combo`) != 2 || count(t, srv, `select count(*) from app.combo_member`) != 2 {
		t.Error("collision left a half-created combo")
	}
}

// comboNow is fixed so expiry never depends on the test machine or Postgres clock.
func comboNow() time.Time { return time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC) }

func comboStack(t *testing.T, cfg combo.Config, eyes scan.Vision) stack {
	t.Helper()
	if cfg.Now == nil {
		cfg.Now = comboNow
	}
	if cfg.Code == nil {
		var next atomic.Int64
		cfg.Code = func() (string, error) { return fmt.Sprintf("%06d", next.Add(1)), nil }
	}
	return sessionStack(t, func(mux *http.ServeMux, repo *repository.Repository, authn *auth.Auth) {
		cfg.Repo, cfg.Auth = repo, authn
		c := combo.NewCombo(cfg)
		mux.Handle(accountv1connect.NewAccountServiceHandler(account.NewAccount(account.Config{Repo: repo, Auth: authn})))
		mux.Handle(combov1connect.NewComboServiceHandler(c))
		mux.Handle(combov1connect.NewComboHostServiceHandler(c))
		mux.Handle(scanv1connect.NewScanServiceHandler(scan.NewScan(scan.Config{Repo: repo, Auth: authn, Vision: eyes, Now: cfg.Now})))
	})
}

type comboPhone struct {
	combo combov1connect.ComboServiceClient
	host  combov1connect.ComboHostServiceClient
	scan  scanv1connect.ScanServiceClient
}

func comboClients(phone *http.Client, url string) comboPhone {
	return comboPhone{combov1connect.NewComboServiceClient(phone, url), combov1connect.NewComboHostServiceClient(phone, url), scanv1connect.NewScanServiceClient(phone, url)}
}
func comboPlayer(t *testing.T, srv stack, handle string) comboPhone {
	t.Helper()
	p := player(t, srv.url, handle)
	return comboClients(&http.Client{Jar: p.jar}, srv.url)
}
func startCombo(t *testing.T, p comboPhone) *combov1.Combo {
	t.Helper()
	got, err := p.combo.StartCombo(t.Context(), connect.NewRequest(&combov1.StartComboRequest{}))
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	return got.Msg.Combo
}
func joinCombo(t *testing.T, p comboPhone, code string) *combov1.Combo {
	t.Helper()
	got, err := p.combo.JoinCombo(t.Context(), connect.NewRequest(&combov1.JoinComboRequest{Code: code}))
	if err != nil {
		t.Fatalf("join: %v", err)
	}
	return got.Msg.Combo
}
func getCombo(t *testing.T, p comboPhone, id int64) *combov1.Combo {
	t.Helper()
	got, err := p.combo.GetCombo(t.Context(), connect.NewRequest(&combov1.GetComboRequest{ComboId: id}))
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	return got.Msg.Combo
}
func comboScanRequest(id int64) *connect.Request[scanv1.CreateScanRequest] {
	return connect.NewRequest(&scanv1.CreateScanRequest{Photos: []*scanv1.Photo{fridgePhoto()}, ComboId: &id})
}
func comboScan(t *testing.T, p comboPhone, id int64) *scanv1.Verdict {
	t.Helper()
	got, err := p.scan.CreateScan(t.Context(), comboScanRequest(id))
	if err != nil {
		t.Fatalf("combo scan: %v", err)
	}
	return got.Msg.Verdict
}
func comboError(t *testing.T, err error, code connect.Code, message string) {
	t.Helper()
	cerr := asConnectErr(t, err)
	if cerr.Code() != code || cerr.Message() != message {
		t.Errorf("got %s %q, want %s %q", cerr.Code(), cerr.Message(), code, message)
	}
}
func comboCalls(p comboPhone, id int64) map[string]func(context.Context) error {
	return map[string]func(context.Context) error{
		"start": func(ctx context.Context) error {
			_, err := p.combo.StartCombo(ctx, connect.NewRequest(&combov1.StartComboRequest{}))
			return err
		},
		"join": func(ctx context.Context) error {
			_, err := p.combo.JoinCombo(ctx, connect.NewRequest(&combov1.JoinComboRequest{Code: "000001"}))
			return err
		},
		"get": func(ctx context.Context) error {
			_, err := p.combo.GetCombo(ctx, connect.NewRequest(&combov1.GetComboRequest{ComboId: id}))
			return err
		},
		"end": func(ctx context.Context) error {
			_, err := p.host.EndCombo(ctx, connect.NewRequest(&combov1.EndComboRequest{ComboId: id}))
			return err
		},
		"remove": func(ctx context.Context) error {
			_, err := p.host.RemovePlayer(ctx, connect.NewRequest(&combov1.RemovePlayerRequest{ComboId: id}))
			return err
		},
	}
}
