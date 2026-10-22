// Author: Caden Lund
package handlers_test

import (
	"crypto/sha256"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	cookv1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/cook/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/apigen/cook/v1/cookv1connect"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers"
	"github.com/bengutow1/csc4330_project5/packages/go/points"
	"github.com/bengutow1/csc4330_project5/packages/go/repository"
	"github.com/bengutow1/csc4330_project5/packages/go/vision"
)

// A week of campus times: Monday 19 October 2026 onward.
var (
	mondayNoon    = campusTime("2026-10-19 12:00")
	mondayEvening = campusTime("2026-10-19 20:00")
)

// TestEachBonusFollowsTheScansScore: the worse the fridge, the more cooking out of it pays, and
// the day's first plate adds its streak. The scores are the mock fridges'.
func TestEachBonusFollowsTheScansScore(t *testing.T) {
	t.Parallel()
	cook, scrappy, odds, streak := cookv1.PointReason_POINT_REASON_COOK,
		cookv1.PointReason_POINT_REASON_SCRAPPY_BONUS, cookv1.PointReason_POINT_REASON_AGAINST_ALL_ODDS,
		cookv1.PointReason_POINT_REASON_STREAK
	cases := []struct {
		pin   string
		lines []line
		total int32
	}{
		{pin: "certified_chef", lines: []line{{cook, 100}, {streak, 10}}, total: 110},
		{pin: "home_cook", lines: []line{{cook, 100}, {streak, 10}}, total: 110},
		{pin: "scrappy", lines: []line{{cook, 100}, {scrappy, 50}, {streak, 10}}, total: 160},
		{pin: "fries_in_the_bag", lines: []line{{cook, 100}, {odds, 100}, {streak, 10}}, total: 210},
	}
	for _, c := range cases {
		t.Run(c.pin, func(t *testing.T) {
			t.Parallel()
			mock, err := vision.NewMock(c.pin, "")
			if err != nil {
				t.Fatalf("new mock: %v", err)
			}
			clock := &fakeClock{t: mondayEvening}
			srv := cookStack(t, cookFixture{eyes: mock, chef: mock, now: clock.now})
			maya := cookPlayerFor(t, srv, "test_maya")
			shelves := listRecipes(t, maya, &cookv1.ListRecipesRequest{ScanId: scanFridge(t, maya).GetScanId()})
			recipe := firstOn(t, shelves.GetMakeItNow())

			got := plateMeal(t, maya, recipe.GetRecipeId(), "dinner")

			if !got.GetPassed() || got.GetMessage() != "Looks like "+recipe.GetTitle() {
				t.Errorf("receipt = %v %q, want passed, saying it looks like %s", got.GetPassed(), got.GetMessage(), recipe.GetTitle())
			}
			lines := linesOf(got)
			if !slices.Equal(lines, c.lines) || got.GetTotal() != c.total {
				t.Errorf("lines = %v, total %d; want %v, total %d", lines, got.GetTotal(), c.lines, c.total)
			}
			// The preview on the card was the receipt without its streak.
			if recipe.GetPointsPreview() != c.total-10 {
				t.Errorf("points preview = %d, want %d", recipe.GetPointsPreview(), c.total-10)
			}
		})
	}
}

// TestTheStreakLineCountsMealWindowDaysInARow: the day's first plate pays 10 a streak day, today
// included, and 00:30 belongs to the night before, so it neither starts a day nor pays again.
func TestTheStreakLineCountsMealWindowDaysInARow(t *testing.T) {
	t.Parallel()
	clock := &fakeClock{}
	f, _ := kitchenFixture()
	f.now = clock.now
	srv := cookStack(t, f)
	maya := cookPlayerFor(t, srv, "test_maya")
	recipeID, score := aRecipe(t, maya)
	steps := []struct {
		at         string
		streakLine int32 // 0 for none
		days       int32
	}{
		{at: "2026-10-19 20:00", streakLine: 10, days: 1}, // Monday's first
		{at: "2026-10-20 00:30", streakLine: 0, days: 1},  // still Monday's late night
		{at: "2026-10-20 12:00", streakLine: 20, days: 2}, // Tuesday's first
		{at: "2026-10-20 18:00", streakLine: 0, days: 2},  // Tuesday again
		{at: "2026-10-22 08:00", streakLine: 10, days: 1}, // Wednesday skipped
	}
	for i, step := range steps {
		clock.set(campusTime(step.at))

		got := plateMeal(t, maya, recipeID, fmt.Sprintf("plate %d", i))

		want := []line{{cookv1.PointReason_POINT_REASON_COOK, 100}}
		want = append(want, bonusFor(score)...)
		if step.streakLine > 0 {
			want = append(want, line{cookv1.PointReason_POINT_REASON_STREAK, step.streakLine})
		}
		lines := linesOf(got)
		if !slices.Equal(lines, want) || got.GetStreakDays() != step.days {
			t.Errorf("%s: lines %v and a %d-day streak, want %v and %d", step.at, lines, got.GetStreakDays(), want, step.days)
		}
	}
}

// TestAPhotoSeenBeforeIsRefused: the same photo must never count twice, even in a later window,
// and the refusal says so in words for the player.
func TestAPhotoSeenBeforeIsRefused(t *testing.T) {
	t.Parallel()
	clock := &fakeClock{t: mondayNoon}
	f, _ := kitchenFixture()
	f.now = clock.now
	srv := cookStack(t, f)
	maya := cookPlayerFor(t, srv, "test_maya")
	recipeID, _ := aRecipe(t, maya)
	plateMeal(t, maya, recipeID, "lunch")
	clock.set(mondayEvening)

	_, err := maya.cook.PlateMeal(t.Context(), plateRequest(recipeID, "lunch"))

	cerr := asConnectErr(t, err)
	if cerr.Code() != connect.CodeAlreadyExists || cerr.Message() != "we've seen this plate before" {
		t.Errorf("got %v %q, want already_exists, we've seen this plate before", cerr.Code(), cerr.Message())
	}
	assertLedger(t, srv, 1, 1)
}

// TestASecondPlateInOneMealWindowIsRefused: one cook credit per window, so a second plate in it
// is refused, and its plate is not kept either.
func TestASecondPlateInOneMealWindowIsRefused(t *testing.T) {
	t.Parallel()
	clock := &fakeClock{t: mondayEvening}
	f, _ := kitchenFixture()
	f.now = clock.now
	srv := cookStack(t, f)
	maya := cookPlayerFor(t, srv, "test_maya")
	recipeID, _ := aRecipe(t, maya)
	plateMeal(t, maya, recipeID, "dinner")
	clock.set(mondayEvening.Add(90 * time.Minute))

	_, err := maya.cook.PlateMeal(t.Context(), plateRequest(recipeID, "seconds"))

	cerr := asConnectErr(t, err)
	if cerr.Code() != connect.CodeFailedPrecondition || cerr.Message() != "you already cooked this meal window" {
		t.Errorf("got %v %q, want failed_precondition, you already cooked this meal window", cerr.Code(), cerr.Message())
	}
	assertLedger(t, srv, 1, 1)
}

// TestTwentyPlatesRacingForOneWindowEarnOneCredit: twenty plates at once in one window must earn
// exactly one cook credit, and the other nineteen hear why.
func TestTwentyPlatesRacingForOneWindowEarnOneCredit(t *testing.T) {
	t.Parallel()
	clock := &fakeClock{t: mondayEvening}
	f, _ := kitchenFixture()
	f.now = clock.now
	srv := cookStack(t, f)
	maya := cookPlayerFor(t, srv, "test_maya")
	recipeID, _ := aRecipe(t, maya)

	var wg sync.WaitGroup
	errs := make([]error, 20)
	for i := range errs {
		wg.Go(func() {
			_, errs[i] = maya.cook.PlateMeal(t.Context(), plateRequest(recipeID, fmt.Sprintf("racer %d", i)))
		})
	}
	wg.Wait()

	won := 0
	for i, err := range errs {
		code := connect.CodeOf(err)
		switch {
		case err == nil:
			won++
		case code != connect.CodeFailedPrecondition:
			t.Errorf("racer %d: code %v (%v), want failed_precondition", i, code, err)
		}
	}
	if won != 1 {
		t.Errorf("%d racers earned the credit, want exactly 1", won)
	}
	assertLedger(t, srv, 1, 1)
}

// TestPlatesInOneDaysWindowsTakeTurnsForTheStreak: plates at once in the day's four windows each
// earn a cook credit, but only one is the day's first, so only one may pay the streak.
func TestPlatesInOneDaysWindowsTakeTurnsForTheStreak(t *testing.T) {
	t.Parallel()
	clock := &tickingClock{times: []time.Time{
		campusTime("2026-10-19 08:00"), campusTime("2026-10-19 12:00"),
		campusTime("2026-10-19 18:00"), campusTime("2026-10-19 23:00"),
	}}
	f, chef := kitchenFixture()
	f.now = clock.now
	srv := cookStack(t, f)
	maya := cookPlayerFor(t, srv, "test_maya")
	recipeID, _ := aRecipe(t, maya)
	// The four checks finish together, so the plates' transactions overlap.
	var together sync.WaitGroup
	together.Add(4)
	chef.together = &together

	var wg sync.WaitGroup
	receipts := make([]*cookv1.Receipt, 4)
	errs := make([]error, 4)
	for i := range receipts {
		wg.Go(func() {
			got, err := maya.cook.PlateMeal(t.Context(), plateRequest(recipeID, fmt.Sprintf("window %d", i)))
			errs[i] = err
			if err == nil {
				receipts[i] = got.Msg.GetReceipt()
			}
		})
	}
	wg.Wait()

	streakLines := 0
	for i, r := range receipts {
		if errs[i] != nil {
			t.Fatalf("plate %d: %v", i, errs[i])
		}
		for _, l := range linesOf(r) {
			if l.reason == cookv1.PointReason_POINT_REASON_STREAK {
				streakLines++
			}
		}
	}
	if streakLines != 1 {
		t.Errorf("%d plates paid the streak, want only the day's first", streakLines)
	}
	assertLedger(t, srv, 4, 4)
}

// TestARejectedPlateWritesNothing: a plate the model does not pass earns nothing and keeps
// nothing, and the player reads our sentence, never the model's reason.
func TestARejectedPlateWritesNothing(t *testing.T) {
	t.Parallel()
	chef := &fakeChef{recipes: kitchenRecipes, rejectPlates: true}
	var logs lockedBuffer
	srv := cookStack(t, cookFixture{eyes: newFakeVision(kitchenFridge), chef: chef, log: jsonLog(&logs)})
	maya := cookPlayerFor(t, srv, "test_maya")
	recipeID, _ := aRecipe(t, maya)

	got := plateMeal(t, maya, recipeID, "a blur")

	want := "That doesn't look like Spinach Cheddar Omelette. Try another angle."
	if got.GetPassed() || got.GetMessage() != want {
		t.Errorf("receipt = %v %q, want not passed, %q", got.GetPassed(), got.GetMessage(), want)
	}
	if len(got.GetLines()) != 0 || got.GetTotal() != 0 || len(got.GetNewBadges()) != 0 {
		t.Errorf("receipt paid %v (%d) and %v, want nothing", got.GetLines(), got.GetTotal(), got.GetNewBadges())
	}
	if got.GetRankBefore().GetName() != "Fry Cook" || got.GetRankAfter().GetName() != "Fry Cook" {
		t.Errorf("ranks = %v, %v; want Fry Cook both", got.GetRankBefore(), got.GetRankAfter())
	}
	assertLedger(t, srv, 0, 0)
	if !strings.Contains(logs.String(), "a photo of the floor") {
		t.Errorf("log = %s, want the model's reason for the server", logs.String())
	}
}

// TestPlatingAnotherPlayersRecipeIsNotFound: the recipe is fenced through its scan, so another
// player's answers as a missing one does, before the model is asked or anything is written.
func TestPlatingAnotherPlayersRecipeIsNotFound(t *testing.T) {
	t.Parallel()
	f, chef := kitchenFixture()
	srv := cookStack(t, f)
	maya := cookPlayerFor(t, srv, "test_maya")
	sam := cookPlayerFor(t, srv, "test_sam")
	mayas, _ := aRecipe(t, maya)

	_, err := sam.cook.PlateMeal(t.Context(), plateRequest(mayas, "sam's"))

	code := connect.CodeOf(err)
	if code != connect.CodeNotFound {
		t.Errorf("code %v (%v), want not_found", code, err)
	}
	checked := chef.plateChecks()
	if len(checked) != 0 {
		t.Errorf("the model checked %v, want nothing", checked)
	}
	assertLedger(t, srv, 0, 0)
}

// TestANewBadgeAppearsOnce: the receipt celebrates a badge on the plate that earned it, and
// never again.
func TestANewBadgeAppearsOnce(t *testing.T) {
	t.Parallel()
	clock := &fakeClock{t: mondayEvening}
	f, _ := kitchenFixture()
	f.now = clock.now
	srv := cookStack(t, f)
	maya := cookPlayerFor(t, srv, "test_maya")
	recipeID, score := aRecipe(t, maya)

	first := plateMeal(t, maya, recipeID, "first")
	clock.set(mondayEvening.Add(24 * time.Hour))
	second := plateMeal(t, maya, recipeID, "second")

	// The omelette takes four things, so of the plate's badges only these can be new.
	want := []string{"On the Board"}
	if score < 35 {
		want = append(want, "Underdog")
	}
	names := badgeNames(first)
	if !slices.Equal(names, want) {
		t.Errorf("first plate's new badges = %v, want %v", names, want)
	}
	again := badgeNames(second)
	if len(again) != 0 {
		t.Errorf("second plate's new badges = %v, want none", again)
	}
}

// TestTheReceiptShowsTheRankBeforeAndAfter: a plate that crosses a title's floor must show both
// titles, and how far the player was from the next.
func TestTheReceiptShowsTheRankBeforeAndAfter(t *testing.T) {
	t.Parallel()
	clock := &fakeClock{t: mondayEvening}
	f, _ := kitchenFixture()
	f.now = clock.now
	srv := cookStack(t, f)
	maya := cookPlayerFor(t, srv, "test_maya")
	recipeID, _ := aRecipe(t, maya)
	// 450 points from an earlier week, written straight to the ledger.
	repo := repository.New(srv.pool)
	accountID := accountIDOf(t, srv, "test_maya")
	old, err := repo.CreatePlate(t.Context(), accountID, repository.NewPlate{
		RecipeID: recipeID, Photo: platePhoto("old"), PhotoSHA256: platePhotoHash("old"), At: campusTime("2026-10-05 20:00"),
	})
	if err != nil {
		t.Fatalf("seed a plate: %v", err)
	}
	err = repo.AwardPoints(t.Context(), accountID, old, "2026-10-05 evening", campusTime("2026-10-05 20:00"),
		[]repository.Award{{Reason: repository.ReasonCook, Points: 450}})
	if err != nil {
		t.Fatalf("seed the ledger: %v", err)
	}

	got := plateMeal(t, maya, recipeID, "dinner")

	before, after := got.GetRankBefore(), got.GetRankAfter()
	if before.GetName() != "Fry Cook" || before.GetNext() != "Line Cook" || before.GetToNext() != 50 {
		t.Errorf("rank before = %v, want Fry Cook, 50 from Line Cook", before)
	}
	if after.GetName() != "Line Cook" || after.GetNext() != "Sous Chef" {
		t.Errorf("rank after = %v, want Line Cook, next Sous Chef", after)
	}
}

// TestPlateMealRefusesABadPhoto: a plate is checked before anything spends a call on the model,
// and every problem comes back under its field.
func TestPlateMealRefusesABadPhoto(t *testing.T) {
	t.Parallel()
	f, chef := kitchenFixture()
	srv := cookStack(t, f)
	maya := cookPlayerFor(t, srv, "test_maya")
	recipeID, _ := aRecipe(t, maya)
	tooBig := append(platePhoto("big"), make([]byte, handlers.MaxPhotoBytes)...)
	cases := []struct {
		name string
		req  *cookv1.PlateMealRequest
		want map[string]string
	}{
		{name: "no recipe", req: &cookv1.PlateMealRequest{Photo: platePhoto("x")}, want: map[string]string{"recipe_id": "recipe id is required"}},
		{name: "no photo", req: &cookv1.PlateMealRequest{RecipeId: recipeID}, want: map[string]string{"photo": "the photo isn't a JPEG"}},
		{name: "not a JPEG", req: &cookv1.PlateMealRequest{RecipeId: recipeID, Photo: []byte("GIF89a")}, want: map[string]string{"photo": "the photo isn't a JPEG"}},
		{name: "over 2 MB", req: &cookv1.PlateMealRequest{RecipeId: recipeID, Photo: tooBig}, want: map[string]string{"photo": "the photo is over 2 MB"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			_, err := maya.cook.PlateMeal(t.Context(), connect.NewRequest(c.req))

			code := connect.CodeOf(err)
			if code != connect.CodeInvalidArgument {
				t.Fatalf("code %v (%v), want invalid_argument", code, err)
			}
			got := fieldViolations(t, err)
			if !maps.Equal(got, c.want) {
				t.Errorf("violations = %v, want %v", got, c.want)
			}
		})
	}
	t.Cleanup(func() {
		checked := chef.plateChecks()
		if len(checked) != 0 {
			t.Errorf("the model checked %v, want nothing for a refused plate", checked)
		}
	})
}

// TestAModelOutageCheckingAPlateAnswersUnavailable: a failed check reaches the player as "try
// again", with nothing stored, so the retry counts.
func TestAModelOutageCheckingAPlateAnswersUnavailable(t *testing.T) {
	t.Parallel()
	chef := &fakeChef{recipes: kitchenRecipes, plateErr: fmt.Errorf("vision: check plate: %w", vision.ErrUnavailable)}
	srv := cookStack(t, cookFixture{eyes: newFakeVision(kitchenFridge), chef: chef})
	maya := cookPlayerFor(t, srv, "test_maya")
	recipeID, _ := aRecipe(t, maya)

	_, err := maya.cook.PlateMeal(t.Context(), plateRequest(recipeID, "dinner"))

	cerr := asConnectErr(t, err)
	if cerr.Code() != connect.CodeUnavailable || cerr.Message() != vision.ErrUnavailable.Error() {
		t.Errorf("got %v %q, want unavailable %q", cerr.Code(), cerr.Message(), vision.ErrUnavailable.Error())
	}
	assertLedger(t, srv, 0, 0)
}

// TestPlatesAreLimitedPerPlayer: every check costs money, so a player who has spent their checks
// is refused until later, while another player keeps their own.
func TestPlatesAreLimitedPerPlayer(t *testing.T) {
	t.Parallel()
	clock := &fakeClock{t: mondayNoon}
	f, _ := kitchenFixture()
	f.now, f.plateLimit = clock.now, handlers.NewLimiter(1, time.Hour)
	srv := cookStack(t, f)
	maya := cookPlayerFor(t, srv, "test_maya")
	sam := cookPlayerFor(t, srv, "test_sam")
	mine, _ := aRecipe(t, maya)
	theirs, _ := aRecipe(t, sam)
	plateMeal(t, maya, mine, "lunch")
	clock.set(mondayEvening)

	_, err := maya.cook.PlateMeal(t.Context(), plateRequest(mine, "dinner"))

	cerr := asConnectErr(t, err)
	if cerr.Code() != connect.CodeResourceExhausted {
		t.Errorf("maya's second check: code %v (%v), want resource_exhausted", cerr.Code(), cerr)
	}
	plateMeal(t, sam, theirs, "sam's dinner")
}

// TestPlatingNeedsAnAccount: points belong to a player, so a phone with no session is refused.
func TestPlatingNeedsAnAccount(t *testing.T) {
	t.Parallel()
	f, _ := kitchenFixture()
	srv := cookStack(t, f)
	stranger := cookv1connect.NewCookServiceClient(http.DefaultClient, srv.url)

	_, err := stranger.PlateMeal(t.Context(), plateRequest(1, "dinner"))

	code := connect.CodeOf(err)
	if code != connect.CodeUnauthenticated {
		t.Errorf("code %v (%v), want unauthenticated", code, err)
	}
}

// line is one receipt line, as a test compares it.
type line struct {
	reason cookv1.PointReason
	points int32
}

func linesOf(r *cookv1.Receipt) []line {
	var out []line
	for _, l := range r.GetLines() {
		out = append(out, line{l.GetReason(), l.GetPoints()})
	}
	return out
}

func badgeNames(r *cookv1.Receipt) []string {
	var out []string
	for _, b := range r.GetNewBadges() {
		out = append(out, b.GetName())
	}
	return out
}

// bonusFor is the bonus a scan's score pays, written out by hand from the points rules.
func bonusFor(score int32) []line {
	switch {
	case score < 35:
		return []line{{cookv1.PointReason_POINT_REASON_AGAINST_ALL_ODDS, 100}}
	case score < 60:
		return []line{{cookv1.PointReason_POINT_REASON_SCRAPPY_BONUS, 50}}
	default:
		return nil
	}
}

// aRecipe scans kitchenFridge for the player and returns the omelette on its shelf, and the
// scan's score.
func aRecipe(t *testing.T, p cookPlayer) (int64, int32) {
	t.Helper()
	verdict := scanFridge(t, p)
	shelves := listRecipes(t, p, &cookv1.ListRecipesRequest{ScanId: verdict.GetScanId()})
	omelette := firstOn(t, shelves.GetMakeItNow())
	if omelette.GetTitle() != omeletteRecipe.Title {
		t.Fatalf("first on the shelf = %q, want the omelette", omelette.GetTitle())
	}
	return omelette.GetRecipeId(), verdict.GetScore()
}

func plateRequest(recipeID int64, photo string) *connect.Request[cookv1.PlateMealRequest] {
	return connect.NewRequest(&cookv1.PlateMealRequest{RecipeId: recipeID, Photo: platePhoto(photo)})
}

// plateMeal plates a photo that must be answered, and returns the receipt.
func plateMeal(t *testing.T, p cookPlayer, recipeID int64, photo string) *cookv1.Receipt {
	t.Helper()
	got, err := p.cook.PlateMeal(t.Context(), plateRequest(recipeID, photo))
	if err != nil {
		t.Fatalf("plate meal %q: %v", photo, err)
	}
	return got.Msg.GetReceipt()
}

// assertLedger checks how many plates and cook credits were stored.
func assertLedger(t *testing.T, srv stack, plates, credits int) {
	t.Helper()
	gotPlates := count(t, srv, `select count(*) from app.plate`)
	gotCredits := count(t, srv, `select count(*) from app.point_event where reason = 'cook'`)
	if gotPlates != plates || gotCredits != credits {
		t.Errorf("%d plates and %d cook credits stored, want %d and %d", gotPlates, gotCredits, plates, credits)
	}
}

func accountIDOf(t *testing.T, srv stack, handle string) int64 {
	t.Helper()
	var id int64
	err := srv.pool.QueryRow(t.Context(), `select account_id from app.account where handle = $1`, handle).Scan(&id)
	if err != nil {
		t.Fatalf("account id of %s: %v", handle, err)
	}
	return id
}

// platePhoto is a JPEG's opening bytes, then a name, so each name is a photo of its own.
func platePhoto(name string) []byte {
	return append([]byte("\xFF\xD8\xFF\xE0"), name...)
}

func platePhotoHash(name string) []byte {
	sum := sha256.Sum256(platePhoto(name))
	return sum[:]
}

func campusTime(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", s, points.Campus())
	if err != nil {
		panic(err)
	}
	return t
}

// tickingClock hands out its times in turn, one per reading, so plates at once land in windows of
// their own.
type tickingClock struct {
	mu    sync.Mutex
	times []time.Time
	next  int
}

func (c *tickingClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	t := c.times[c.next%len(c.times)]
	c.next++
	return t
}

// fakeClock is the campus clock a test moves by hand. Plates read it from the server's
// goroutines, so it is guarded.
type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *fakeClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) set(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = t
}
