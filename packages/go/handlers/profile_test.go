// Author: Caden Lund
package handlers_test

import (
	"fmt"
	"maps"
	"net/http"
	"slices"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/bengutow1/csc4330_project5/packages/go/apigen/account/v1/accountv1connect"
	profilev1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/profile/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/apigen/profile/v1/profilev1connect"
	"github.com/bengutow1/csc4330_project5/packages/go/auth"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers/account"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers/profile"
	"github.com/bengutow1/csc4330_project5/packages/go/points"
	"github.com/bengutow1/csc4330_project5/packages/go/repository"
)

// profileMonday starts the week the profile tests stand in, and profileNow is Wednesday evening
// of it.
var (
	profileMonday = time.Date(2026, 10, 19, 0, 0, 0, 0, points.Campus())
	profileNow    = profileMonday.AddDate(0, 0, 2).Add(18 * time.Hour)
)

// TestTheProfileReadsAKnownHistory: points, rank, stats and badges all come from the player's own
// history: two scans, three days of plates with a streak running to today, and last week's lone
// fry run, which topped that week's Hall of Fries.
func TestTheProfileReadsAKnownHistory(t *testing.T) {
	t.Parallel()
	srv := profileStack(t, profileNow)
	maya := profileOf(t, srv, "test_maya")
	low := scannedAt(t, srv, "test_maya", 29, profileMonday.Add(9*time.Hour))
	high := scannedAt(t, srv, "test_maya", 91, profileMonday.AddDate(0, 0, 1).Add(9*time.Hour))
	platedFrom(t, srv, "test_maya", low, profileMonday.Add(12*time.Hour), "monday",
		award(repository.ReasonCook, 100), award(repository.ReasonAgainstAllOdds, 100), award(repository.ReasonStreak, 10))
	platedFrom(t, srv, "test_maya", high, profileMonday.AddDate(0, 0, 1).Add(12*time.Hour), "tuesday",
		award(repository.ReasonCook, 100), award(repository.ReasonStreak, 20))
	platedFrom(t, srv, "test_maya", high, profileMonday.AddDate(0, 0, 2).Add(12*time.Hour), "wednesday",
		award(repository.ReasonCook, 100), award(repository.ReasonStreak, 30))
	ranOnce(t, srv, "test_maya", profileMonday.AddDate(0, 0, -4).Add(12*time.Hour))

	got := getProfile(t, maya)

	if got.GetHandle() != "test_maya" || got.GetAvatar() != "🍟" || got.GetPoints() != 460 {
		t.Errorf("profile = %s %s with %d points, want test_maya 🍟 with 460", got.GetHandle(), got.GetAvatar(), got.GetPoints())
	}
	rank := got.GetRank()
	if rank.GetName() != "Fry Cook" || rank.GetNext() != "Line Cook" || rank.GetToNext() != 40 || rank.GetProgress() != 0.92 {
		t.Errorf("rank = %v, want Fry Cook, 40 to Line Cook, 0.92 of the way", rank)
	}
	stats := got.GetStats()
	if stats.GetCooks() != 3 || stats.GetStreakDays() != 3 || stats.GetFryRuns() != 1 ||
		stats.GetFryRatio() != 0.25 || stats.GetBestScore() != 91 {
		t.Errorf("stats = %v, want 3 cooks, a 3-day streak, 1 fry run, 0.25 fry ratio, best 91", stats)
	}
	earned := map[string]bool{}
	var names []string
	for _, b := range got.GetBadges() {
		names = append(names, b.GetName())
		earned[b.GetName()] = b.GetEarned()
	}
	wantNames := []string{
		"First Roast", "Certified Chef", "On the Board", "Underdog", "Three's a Meal", "Combo Meal",
		"Clean Plate Club", "Late Night Legend", "Glow Up", "Microwave Maestro", "Bag Secured",
	}
	if !slices.Equal(names, wantNames) {
		t.Errorf("badges = %v, want all eleven in the concept's order", names)
	}
	wantEarned := map[string]bool{
		"First Roast": true, "Certified Chef": true, "On the Board": true, "Underdog": true,
		"Three's a Meal": true, "Combo Meal": false, "Clean Plate Club": false,
		"Late Night Legend": false, "Glow Up": true, "Microwave Maestro": false, "Bag Secured": true,
	}
	if !maps.Equal(earned, wantEarned) {
		t.Errorf("earned = %v\nwant %v", earned, wantEarned)
	}
}

// TestTheFryRatioWithZeroCooks: a player who has only gone for fries is all fries, and a newcomer
// with nothing at all is at 0, with no division by zero and the first title ahead of them. Topping
// a Hall of Fries week that is not over yet earns nothing.
func TestTheFryRatioWithZeroCooks(t *testing.T) {
	t.Parallel()
	srv := profileStack(t, profileNow)
	dan := profileOf(t, srv, "test_dan")
	eve := profileOf(t, srv, "test_eve")
	ranOnce(t, srv, "test_dan", profileMonday.Add(12*time.Hour))
	ranOnce(t, srv, "test_dan", profileMonday.AddDate(0, 0, 1).Add(12*time.Hour))

	dans := getProfile(t, dan)
	eves := getProfile(t, eve)

	danStats, eveStats, eveRank := dans.GetStats(), eves.GetStats(), eves.GetRank()
	if danStats.GetCooks() != 0 || danStats.GetFryRuns() != 2 || danStats.GetFryRatio() != 1 {
		t.Errorf("dan's stats = %v, want no cooks, 2 fry runs, a fry ratio of 1", danStats)
	}
	if eveStats.GetCooks() != 0 || eveStats.GetFryRuns() != 0 || eveStats.GetFryRatio() != 0 {
		t.Errorf("eve's stats = %v, want nothing and a fry ratio of 0", eveStats)
	}
	if eveRank.GetName() != "Fry Cook" || eveRank.GetToNext() != 500 || eveRank.GetProgress() != 0 {
		t.Errorf("eve's rank = %v, want Fry Cook with all 500 to go", eveRank)
	}
	for _, b := range dans.GetBadges() {
		if b.GetName() == "Bag Secured" && b.GetEarned() {
			t.Error("dan earned Bag Secured for topping a week that is not over, want it locked")
		}
	}
}

// TestRankProgressAtABoundary: 490 points is nearly a Line Cook, and 500 is one, at the start of
// the way to Sous Chef.
func TestRankProgressAtABoundary(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		lines    [][]repository.Award // each plate's awards
		wantRank string
		wantNext string
		toNext   int32
		progress float64
	}{
		{
			"490, ten short",
			[][]repository.Award{
				{award(repository.ReasonCook, 100), award(repository.ReasonScrappyBonus, 50)},
				{award(repository.ReasonCook, 100), award(repository.ReasonStreak, 40)},
				{award(repository.ReasonCook, 100)},
				{award(repository.ReasonCook, 100)},
			},
			"Fry Cook", "Line Cook", 10, 0.98,
		},
		{
			"500 exactly",
			[][]repository.Award{
				{award(repository.ReasonCook, 100)}, {award(repository.ReasonCook, 100)},
				{award(repository.ReasonCook, 100)}, {award(repository.ReasonCook, 100)},
				{award(repository.ReasonCook, 100)},
			},
			"Line Cook", "Sous Chef", 1000, 0,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			srv := profileStack(t, profileNow)
			maya := profileOf(t, srv, "test_maya")
			s := scannedAt(t, srv, "test_maya", 70, profileMonday)
			for i, lines := range c.lines {
				// Each plate in a meal window of its own, six hours apart.
				platedFrom(t, srv, "test_maya", s, profileMonday.Add(time.Duration(6*i+6)*time.Hour), fmt.Sprint("plate ", i), lines...)
			}

			r := getProfile(t, maya).GetRank()

			if r.GetName() != c.wantRank || r.GetNext() != c.wantNext || r.GetToNext() != c.toNext || r.GetProgress() != c.progress {
				t.Errorf("rank = %v, want %s, %d to %s, %v of the way", r, c.wantRank, c.toNext, c.wantNext, c.progress)
			}
		})
	}
}

// TestPlatesComeNewestFirstAPageAtATime: a walk over pages of two returns every plate once, newest
// first, each with its photo and its recipe, and ends.
func TestPlatesComeNewestFirstAPageAtATime(t *testing.T) {
	t.Parallel()
	srv := profileStack(t, profileNow)
	maya := profileOf(t, srv, "test_maya")
	s := scannedAt(t, srv, "test_maya", 70, profileMonday)
	for i, day := range []int{0, 1, 2} {
		platedFrom(t, srv, "test_maya", s, profileMonday.AddDate(0, 0, day).Add(12*time.Hour), fmt.Sprint("day ", i),
			award(repository.ReasonCook, 100))
	}

	var walked []string
	req := &profilev1.ListPlatesRequest{Limit: 2}
	for pages := 0; ; pages++ {
		if pages > 5 {
			t.Fatalf("still paging after %d pages: %v", pages, walked)
		}
		got := listPlates(t, maya, req)
		for _, p := range got.GetPlates() {
			walked = append(walked, fmt.Sprintf("%s %s %s",
				p.GetPlatedAt().AsTime().In(points.Campus()).Format("Mon"), p.GetRecipeTitle(), p.GetPhoto()[4:]))
		}
		if got.GetNextCursor() == "" {
			break
		}
		req.Cursor = got.GetNextCursor()
	}

	want := []string{"Wed Egg fried rice day 2", "Tue Egg fried rice day 1", "Mon Egg fried rice day 0"}
	if !slices.Equal(walked, want) {
		t.Errorf("walked %v, want %v", walked, want)
	}
}

// TestAnotherPlayersHistoryIsNeverShown: the account in the session is the fence. Ben's plates
// never reach maya's grid, and his plates and runs never count toward her stats.
func TestAnotherPlayersHistoryIsNeverShown(t *testing.T) {
	t.Parallel()
	srv := profileStack(t, profileNow)
	maya := profileOf(t, srv, "test_maya")
	profileOf(t, srv, "test_ben")
	bens := scannedAt(t, srv, "test_ben", 88, profileMonday)
	platedFrom(t, srv, "test_ben", bens, profileMonday.Add(12*time.Hour), "ben's", award(repository.ReasonCook, 100))
	ranOnce(t, srv, "test_ben", profileMonday.Add(19*time.Hour))

	plates := listPlates(t, maya, &profilev1.ListPlatesRequest{})
	mine := getProfile(t, maya)

	if len(plates.GetPlates()) != 0 || plates.GetNextCursor() != "" {
		t.Errorf("maya's grid = %d plates, want none of ben's", len(plates.GetPlates()))
	}
	stats := mine.GetStats()
	if mine.GetPoints() != 0 || stats.GetCooks() != 0 || stats.GetFryRuns() != 0 || stats.GetBestScore() != 0 {
		t.Errorf("maya's profile = %d points and %v, want nothing of ben's", mine.GetPoints(), stats)
	}
}

// TestABadPlatesRequestIsRefusedFieldByField: each problem comes back on its own field.
func TestABadPlatesRequestIsRefusedFieldByField(t *testing.T) {
	t.Parallel()
	srv := profileStack(t, profileNow)
	maya := profileOf(t, srv, "test_maya")

	_, err := maya.ListPlates(t.Context(), connect.NewRequest(&profilev1.ListPlatesRequest{Cursor: "!!!", Limit: -1}))

	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("got code %v (%v), want invalid_argument", connect.CodeOf(err), err)
	}
	want := map[string]string{"cursor": "cursor is not one this server gave out", "limit": "limit must be at least 0"}
	got := fieldViolations(t, err)
	if !maps.Equal(got, want) {
		t.Errorf("violations = %v, want %v", got, want)
	}
}

// TestTheProfileNeedsAnAccount: a fresh install gets unauthenticated from every profile rpc, which
// the app reads as "back to onboarding".
func TestTheProfileNeedsAnAccount(t *testing.T) {
	t.Parallel()
	srv := profileStack(t, profileNow)
	stranger := profilev1connect.NewProfileServiceClient(&http.Client{Jar: newcomer(t, srv.url).jar}, srv.url)
	calls := map[string]func() error{
		"GetProfile": func() error {
			_, err := stranger.GetProfile(t.Context(), connect.NewRequest(&profilev1.GetProfileRequest{}))
			return err
		},
		"ListPlates": func() error {
			_, err := stranger.ListPlates(t.Context(), connect.NewRequest(&profilev1.ListPlatesRequest{}))
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

// profileStack is AccountService and ProfileService behind the session middleware, over the test's
// own database, with the profile's clock stopped at now.
func profileStack(t *testing.T, now time.Time) stack {
	t.Helper()
	return sessionStack(t, func(mux *http.ServeMux, repo *repository.Repository, authn *auth.Auth) {
		mux.Handle(accountv1connect.NewAccountServiceHandler(account.NewAccount(account.Config{Repo: repo, Auth: authn})))
		mux.Handle(profilev1connect.NewProfileServiceHandler(profile.NewProfile(profile.Config{
			Repo: repo,
			Auth: authn,
			Now:  func() time.Time { return now },
		})))
	})
}

// profileOf makes handle's account through the real rpc and returns their profile client.
func profileOf(t *testing.T, srv stack, handle string) profilev1connect.ProfileServiceClient {
	t.Helper()
	return profilev1connect.NewProfileServiceClient(&http.Client{Jar: player(t, srv.url, handle).jar}, srv.url)
}

// scannedAt stores a scan of score for handle, stamped at at.
func scannedAt(t *testing.T, srv stack, handle string, score int, at time.Time) repository.Scan {
	t.Helper()
	s, err := repository.New(srv.pool).CreateScan(t.Context(), accountIDOf(t, srv, handle), score, "a fridge", nil)
	if err != nil {
		t.Fatalf("scan for %s: %v", handle, err)
	}
	_, err = srv.pool.Exec(t.Context(), `update app.scan set created_at = $2 where scan_id = $1`, s.ScanID, at)
	if err != nil {
		t.Fatalf("stamp scan %d: %v", s.ScanID, err)
	}
	return s
}

// platedFrom plates the dish on scan s at at, with a photo named photo, and pays it the lines
// given, in at's meal window. The dish takes two things, so it counts toward Three's a Meal.
func platedFrom(t *testing.T, srv stack, handle string, s repository.Scan, at time.Time, photo string, lines ...repository.Award) {
	t.Helper()
	repo := repository.New(srv.pool)
	accountID := accountIDOf(t, srv, handle)
	_, err := repo.CreateRecipes(t.Context(), accountID, s.ScanID, s.ItemsVersion, []repository.NewRecipe{{
		Title: "Egg fried rice", Minutes: 15, Difficulty: repository.DifficultyEasy,
		Equipment: []string{repository.EquipmentStovetop},
		Uses:      []string{"Eggs", "Rice"}, Steps: []string{"Fry the rice, then the eggs."},
	}})
	if err != nil {
		t.Fatalf("recipe for %s: %v", handle, err)
	}
	recipes, err := repo.ListRecipes(t.Context(), accountID, s.ScanID, s.ItemsVersion, repository.RecipeFilter{})
	if err != nil || len(recipes) != 1 {
		t.Fatalf("list %s's recipe: %v (%d recipes)", handle, err, len(recipes))
	}
	plateID, err := repo.CreatePlate(t.Context(), accountID, repository.NewPlate{
		RecipeID: recipes[0].RecipeID, Photo: platePhoto(photo), PhotoSHA256: platePhotoHash(photo), At: at,
	})
	if err != nil {
		t.Fatalf("plate %q for %s: %v", photo, handle, err)
	}
	window, day := points.MealWindow(at)
	err = repo.AwardPoints(t.Context(), accountID, plateID, day.Format(time.DateOnly)+" "+string(window), at, lines)
	if err != nil {
		t.Fatalf("pay plate %q for %s: %v", photo, handle, err)
	}
}

// ranOnce logs one fry run for handle at at.
func ranOnce(t *testing.T, srv stack, handle string, at time.Time) {
	t.Helper()
	window, day := points.MealWindow(at)
	_, err := repository.New(srv.pool).LogFryRun(t.Context(), accountIDOf(t, srv, handle), repository.FryRun{
		SpotID: "mock-03", SpotName: "Night Owl Fries", MealDay: day, MealWindow: string(window), At: at,
	})
	if err != nil {
		t.Fatalf("fry run for %s: %v", handle, err)
	}
}

func award(reason string, points int) repository.Award {
	return repository.Award{Reason: reason, Points: points}
}

func getProfile(t *testing.T, c profilev1connect.ProfileServiceClient) *profilev1.Profile {
	t.Helper()
	resp, err := c.GetProfile(t.Context(), connect.NewRequest(&profilev1.GetProfileRequest{}))
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}
	return resp.Msg.GetProfile()
}

func listPlates(t *testing.T, c profilev1connect.ProfileServiceClient, req *profilev1.ListPlatesRequest) *profilev1.ListPlatesResponse {
	t.Helper()
	resp, err := c.ListPlates(t.Context(), connect.NewRequest(req))
	if err != nil {
		t.Fatalf("list plates: %v", err)
	}
	return resp.Msg
}
