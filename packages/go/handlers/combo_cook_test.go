// Author: LAPTOP-BPMQJP10\andyk
package handlers_test

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/bengutow1/csc4330_project5/packages/go/apigen/account/v1/accountv1connect"
	combov1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/combo/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/apigen/combo/v1/combov1connect"
	cookv1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/cook/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/apigen/cook/v1/cookv1connect"
	"github.com/bengutow1/csc4330_project5/packages/go/apigen/scan/v1/scanv1connect"
	"github.com/bengutow1/csc4330_project5/packages/go/auth"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers/account"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers/combo"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers/cook"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers/scan"
	"github.com/bengutow1/csc4330_project5/packages/go/points"
	"github.com/bengutow1/csc4330_project5/packages/go/repository"
	"github.com/bengutow1/csc4330_project5/packages/go/vision"
)

// TestComboShelvesFenceBothPlayersAndBothListVersions prevents stale or unsafe merged recipes.
func TestComboShelvesFenceBothPlayersAndBothListVersions(t *testing.T) {
	t.Parallel()
	srv, chef := comboCookStack(t)
	host, guest, c, recipe := readyComboMeal(t, srv)
	other := firstOn(t, listRecipes(t, guest.cookPlayer, &cookv1.ListRecipesRequest{ComboId: &c.ComboId}).MakeItNow)
	if other.Title != recipe.Title || other.RecipeId == recipe.RecipeId || len(chef.calls()) != 1 {
		t.Errorf("guest recipe=%v calls=%d, want own copy and one call", other, len(chef.calls()))
	}
	_, err := host.cook.GetRecipe(t.Context(), connect.NewRequest(&cookv1.GetRecipeRequest{RecipeId: other.RecipeId}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("partner recipe code=%v (%v), want not_found", connect.CodeOf(err), err)
	}
	outsider := comboCookPlayer(t, srv, "test_outsider")
	_, err = outsider.cook.ListRecipes(t.Context(), connect.NewRequest(&cookv1.ListRecipesRequest{ComboId: &c.ComboId}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("outsider code=%v (%v), want not_found", connect.CodeOf(err), err)
	}
	_, err = host.cook.ListRecipes(t.Context(), connect.NewRequest(&cookv1.ListRecipesRequest{ComboId: &c.ComboId, ScanId: 1}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument || fieldViolations(t, err)["scan_id"] == "" {
		t.Errorf("ambiguous selector=%v, want invalid_argument scan_id", err)
	}
	short := listRecipes(t, host.cookPlayer, &cookv1.ListRecipesRequest{ComboId: &c.ComboId, MaxMinutes: 10})
	if len(short.MakeItNow) != 1 {
		t.Errorf("ten minute shelves=%v", short)
	}
	_, err = srv.pool.Exec(t.Context(), `update app.account set allergies=array['eggs'] where handle='test_guest'`)
	if err != nil {
		t.Fatal(err)
	}
	for _, phone := range []comboCookPhone{host, guest} {
		safe := listRecipes(t, phone.cookPlayer, &cookv1.ListRecipesRequest{ComboId: &c.ComboId})
		if len(safe.MakeItNow)+len(safe.OneItemAway) != 0 {
			t.Errorf("allergy reached shelves: %v", safe)
		}
		_, err = phone.cook.GetRecipe(t.Context(), connect.NewRequest(&cookv1.GetRecipeRequest{RecipeId: map[bool]int64{true: recipe.RecipeId, false: other.RecipeId}[phone.cook == host.cook]}))
		if connect.CodeOf(err) != connect.CodeNotFound {
			t.Errorf("allergy recipe code=%v (%v), want not_found", connect.CodeOf(err), err)
		}
	}
	_, err = srv.pool.Exec(t.Context(), `update app.account set allergies='{}' where handle='test_guest'; update app.scan set items_version=items_version+1 where account_id=(select account_id from app.account where handle='test_guest')`)
	if err != nil {
		t.Fatal(err)
	}
	fresh := firstOn(t, listRecipes(t, host.cookPlayer, &cookv1.ListRecipesRequest{ComboId: &c.ComboId}).MakeItNow)
	if fresh.RecipeId == recipe.RecipeId || len(chef.calls()) != 2 {
		t.Errorf("fresh recipe=%d calls=%d, want new cache and second call", fresh.RecipeId, len(chef.calls()))
	}
}

// TestComboSecondAwardFailureRollsBackBothMeals prevents half-paid shared plates.
func TestComboSecondAwardFailureRollsBackBothMeals(t *testing.T) {
	t.Parallel()
	srv, _ := comboCookStack(t)
	host, _, _, recipe := readyComboMeal(t, srv)
	_, err := srv.pool.Exec(t.Context(), `
		create function app.reject_guest_award() returns trigger language plpgsql as $$
		begin
		 if new.account_id=(select account_id from app.account where handle='test_guest') then
		  raise exception 'second award unavailable';
		 end if;
		 return new;
		end $$;
		create trigger reject_guest_award before insert on app.point_event
		for each row execute function app.reject_guest_award()`)
	if err != nil {
		t.Fatalf("install failure: %v", err)
	}
	_, err = host.cook.PlateMeal(t.Context(), connect.NewRequest(&cookv1.PlateMealRequest{RecipeId: recipe.RecipeId, Photo: testJPEG}))
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("code = %v (%v), want internal", connect.CodeOf(err), err)
	}
	var plates, awards int
	err = srv.pool.QueryRow(t.Context(), `select (select count(*) from app.plate),(select count(*) from app.point_event)`).Scan(&plates, &awards)
	if err != nil {
		t.Fatalf("read rollback: %v", err)
	}
	if plates != 0 || awards != 0 {
		t.Errorf("plates=%d awards=%d, want both zero", plates, awards)
	}
}

// TestConcurrentComboPlatesClaimEachWindowOnce proves either phone can submit without double pay.
func TestConcurrentComboPlatesClaimEachWindowOnce(t *testing.T) {
	t.Parallel()
	srv, _ := comboCookStack(t)
	host, guest, c, recipe := readyComboMeal(t, srv)
	other := firstOn(t, listRecipes(t, guest.cookPlayer, &cookv1.ListRecipesRequest{ComboId: &c.ComboId}).MakeItNow)
	var wg sync.WaitGroup
	results := make(chan error, 20)
	for i := range 20 {
		wg.Go(func() {
			phone, id := host, recipe.RecipeId
			if i%2 == 1 {
				phone, id = guest, other.RecipeId
			}
			_, err := phone.cook.PlateMeal(t.Context(), connect.NewRequest(&cookv1.PlateMealRequest{RecipeId: id, Photo: append(append([]byte{}, testJPEG...), fmt.Sprint(i)...)}))
			results <- err
		})
	}
	wg.Wait()
	close(results)
	succeeded := 0
	for err := range results {
		if err == nil {
			succeeded++
		} else if connect.CodeOf(err) != connect.CodeFailedPrecondition {
			t.Errorf("code=%v (%v), want failed_precondition", connect.CodeOf(err), err)
		}
	}
	if succeeded != 1 {
		t.Errorf("successful plates=%d, want 1", succeeded)
	}
	var cooks, bonuses, plates int
	err := srv.pool.QueryRow(t.Context(), `select (select count(*) from app.point_event where reason='cook'),(select count(*) from app.point_event where reason='combo_cook'),(select count(*) from app.plate)`).Scan(&cooks, &bonuses, &plates)
	if err != nil {
		t.Fatalf("count credits: %v", err)
	}
	if cooks != 2 || bonuses != 2 || plates != 2 {
		t.Errorf("cooks=%d bonuses=%d plates=%d, want 2 each", cooks, bonuses, plates)
	}
}

// TestDeletingEitherComboPlayerKeepsThePartnersMeal prevents linked-photo cascades losing history.
func TestDeletingEitherComboPlayerKeepsThePartnersMeal(t *testing.T) {
	t.Parallel()
	for _, deleteHost := range []bool{true, false} {
		t.Run(fmt.Sprint(deleteHost), func(t *testing.T) {
			t.Parallel()
			srv, _ := comboCookStack(t)
			host, _, c, recipe := readyComboMeal(t, srv)
			plateMeal(t, host.cookPlayer, recipe.RecipeId, "shared deletion")
			deletedID := c.HostId
			if !deleteHost {
				err := srv.pool.QueryRow(t.Context(), `select account_id from app.account where handle='test_guest'`).Scan(&deletedID)
				if err != nil {
					t.Fatal(err)
				}
			}
			deleted, err := repository.New(srv.pool).DeleteAccount(t.Context(), deletedID)
			if err != nil || !deleted {
				t.Fatalf("delete account = %v, %v", deleted, err)
			}
			var plates, cooks, roots int
			err = srv.pool.QueryRow(t.Context(), `select (select count(*) from app.plate),(select count(*) from app.point_event where reason='cook'),(select count(*) from app.plate where combo_plate_id is null)`).Scan(&plates, &cooks, &roots)
			if err != nil {
				t.Fatal(err)
			}
			if plates != 1 || cooks != 1 || roots != 1 {
				t.Errorf("plates=%d cooks=%d roots=%d, want 1 each", plates, cooks, roots)
			}
		})
	}
}

// readyComboMeal gives each player one exclusive food and returns the host's shared recipe.
func readyComboMeal(t *testing.T, srv stack) (comboCookPhone, comboCookPhone, *combov1.Combo, *cookv1.Recipe) {
	t.Helper()
	host, guest := comboCookPlayer(t, srv, "test_host"), comboCookPlayer(t, srv, "test_guest")
	c := startCombo(t, host.comboPhone)
	joinCombo(t, guest.comboPhone, c.Code)
	comboScan(t, host.comboPhone, c.ComboId)
	comboScan(t, guest.comboPhone, c.ComboId)
	_, err := srv.pool.Exec(t.Context(), `update app.scan_item i set name='Peas',category='vegetable' from app.scan s join app.account a on a.account_id=s.account_id where i.scan_id=s.scan_id and a.handle='test_guest'`)
	if err != nil {
		t.Fatalf("guest food: %v", err)
	}
	r := firstOn(t, listRecipes(t, host.cookPlayer, &cookv1.ListRecipesRequest{ComboId: &c.ComboId}).MakeItNow)
	return host, guest, c, r
}

// TestComboRecipeTagsAndOnePlatePaysBothPlayers proves merged inputs and atomic independent credit.
func TestComboRecipeTagsAndOnePlatePaysBothPlayers(t *testing.T) {
	t.Parallel()
	for _, hostAlreadyCooked := range []bool{false, true} {
		t.Run(fmt.Sprint(hostAlreadyCooked), func(t *testing.T) {
			t.Parallel()
			srv, chef := comboCookStack(t)
			host, guest := comboCookPlayer(t, srv, "test_host"), comboCookPlayer(t, srv, "test_guest")
			c := startCombo(t, host.comboPhone)
			joinCombo(t, guest.comboPhone, c.Code)
			hostScan := comboScan(t, host.comboPhone, c.ComboId)
			comboScan(t, guest.comboPhone, c.ComboId)
			var guestID int64
			err := srv.pool.QueryRow(t.Context(), `select account_id from app.account where handle='test_guest'`).Scan(&guestID)
			if err != nil {
				t.Fatalf("guest: %v", err)
			}
			_, err = srv.pool.Exec(t.Context(), `update app.scan_item i set name='Peas',category='vegetable' from app.scan s where i.scan_id=s.scan_id and s.account_id=$1`, guestID)
			if err != nil {
				t.Fatalf("guest food: %v", err)
			}
			shelves := listRecipes(t, host.cookPlayer, &cookv1.ListRecipesRequest{ComboId: &c.ComboId})
			if len(shelves.MakeItNow) != 1 {
				t.Fatalf("shelves = %v, want one combined meal", shelves)
			}
			r := shelves.MakeItNow[0]
			if !r.OnlyTogether || len(r.IngredientOwners) != 2 || r.PointsPreview != 200 {
				t.Errorf("combo recipe = %v, want both owners, Only together and +200", r)
			}
			if hostAlreadyCooked {
				_, err := repository.New(srv.pool).CreateRecipes(t.Context(), c.HostId, hostScan.ScanId, 1, []repository.NewRecipe{{Title: "Eggs", Minutes: 5, Difficulty: "easy", Uses: []string{"Eggs"}, Steps: []string{"Cook the eggs."}}})
				if err != nil {
					t.Fatalf("solo recipe setup: %v", err)
				}
				solo := listRecipes(t, host.cookPlayer, &cookv1.ListRecipesRequest{ScanId: hostScan.ScanId})
				plateMeal(t, host.cookPlayer, firstOn(t, solo.MakeItNow).RecipeId, "solo plate")
			}
			response, err := host.cook.PlateMeal(t.Context(), connect.NewRequest(&cookv1.PlateMealRequest{RecipeId: r.RecipeId, Photo: append(testJPEG, []byte("combo plate")...)}))
			if err != nil {
				t.Fatalf("combo plate: %v", err)
			}
			if !response.Msg.Receipt.Passed {
				t.Fatal("valid combo plate was rejected")
			}
			wantHost := 210
			if hostAlreadyCooked {
				wantHost = 0
			}
			if int(response.Msg.Receipt.Total) != wantHost {
				t.Errorf("host receipt = %d, want %d", response.Msg.Receipt.Total, wantHost)
			}
			rows, err := srv.pool.Query(t.Context(), `select a.handle,e.reason,e.points from app.point_event e join app.account a using(account_id) order by a.handle,e.point_event_id`)
			if err != nil {
				t.Fatalf("ledger: %v", err)
			}
			defer rows.Close()
			counts := map[string]int{}
			for rows.Next() {
				var handle, reason string
				var n int
				err = rows.Scan(&handle, &reason, &n)
				if err != nil {
					t.Fatalf("event: %v", err)
				}
				if reason == "combo_cook" {
					counts[handle] += n
				}
			}
			err = rows.Err()
			if err != nil {
				t.Fatalf("ledger rows: %v", err)
			}
			if counts["test_guest"] != 50 || (!hostAlreadyCooked && counts["test_host"] != 50) || (hostAlreadyCooked && counts["test_host"] != 0) {
				t.Errorf("combo bonuses = %v", counts)
			}
			if len(chef.calls()) != 1 && !hostAlreadyCooked {
				t.Errorf("model calls = %d, want shared set once", len(chef.calls()))
			}
		})
	}
}

// comboCookStack mounts the real Combo and cooking surfaces together over one database.
func comboCookStack(t *testing.T) (stack, *fakeChef) {
	t.Helper()
	chef := &fakeChef{recipes: []vision.Recipe{
		{Title: "Eggs and peas", Minutes: 10, Difficulty: "easy", Uses: []string{"Eggs", "Peas"}, Steps: []string{"Cook the eggs and peas."}, Allergens: []string{"eggs"}},
	}}
	now := func() time.Time { return time.Date(2026, 10, 23, 12, 0, 0, 0, points.Campus()) }
	srv := sessionStack(t, func(mux *http.ServeMux, repo *repository.Repository, authn *auth.Auth) {
		mux.Handle(accountv1connect.NewAccountServiceHandler(account.NewAccount(account.Config{Repo: repo, Auth: authn})))
		c := combo.NewCombo(combo.Config{Repo: repo, Auth: authn, Now: now})
		mux.Handle(combov1connect.NewComboServiceHandler(c))
		mux.Handle(combov1connect.NewComboHostServiceHandler(c))
		mux.Handle(scanv1connect.NewScanServiceHandler(scan.NewScan(scan.Config{Repo: repo, Auth: authn, Vision: newFakeVision(vision.Fridge{Items: []vision.Item{{Name: "Eggs", Category: "protein"}}}), Now: now})))
		mux.Handle(cookv1connect.NewCookServiceHandler(cook.NewCook(cook.Config{Repo: repo, Auth: authn, Vision: chef, Now: now})))
	})
	return srv, chef
}

type comboCookPhone struct {
	comboPhone
	cookPlayer
}

func comboCookPlayer(t *testing.T, srv stack, handle string) comboCookPhone {
	t.Helper()
	p := player(t, srv.url, handle)
	phone := &http.Client{Jar: p.jar}
	return comboCookPhone{comboClients(phone, srv.url), cookPlayer{scan: scanv1connect.NewScanServiceClient(phone, srv.url), cook: cookv1connect.NewCookServiceClient(phone, srv.url)}}
}
