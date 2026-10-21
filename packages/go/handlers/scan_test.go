// Author: Caden Lund
package handlers_test

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"

	"github.com/bengutow1/csc4330_project5/packages/go/apigen/account/v1/accountv1connect"
	scanv1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/scan/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/apigen/scan/v1/scanv1connect"
	"github.com/bengutow1/csc4330_project5/packages/go/auth"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers/account"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers/scan"
	"github.com/bengutow1/csc4330_project5/packages/go/repository"
	"github.com/bengutow1/csc4330_project5/packages/go/vision"
)

// testJPEG is enough of a JPEG for the server's check: its start-of-image marker, then filler.
// Tests never send a real photo.
var testJPEG = []byte("\xFF\xD8\xFF\xE0 a test photo")

// chefFridge is the fake model's list unless a test writes its own: a protein beside two
// vegetables and a dairy.
var chefFridge = vision.Fridge{
	Items: []vision.Item{
		{Name: "Chicken", Category: "protein", Quantity: "2 breasts"},
		{Name: "Spinach", Category: "vegetable", Quantity: "1 bag"},
		{Name: "Bell peppers", Category: "vegetable", Quantity: "x3"},
		{Name: "Cheddar", Category: "dairy", Quantity: ""},
	},
	Roast: "Chicken, greens and cheddar. Who are you trying to impress?",
}

// TestEachMockFridgeScansToItsTier: the mock's fridges are what the screens are built against,
// one per tier and an empty one, so each must come back as its tier, read back the same way.
func TestEachMockFridgeScansToItsTier(t *testing.T) {
	t.Parallel()
	cases := []struct {
		fridge string
		score  int32
		tier   scanv1.Tier
		items  int
	}{
		{fridge: "certified_chef", score: 91, tier: scanv1.Tier_TIER_CERTIFIED_CHEF, items: 6},
		{fridge: "home_cook", score: 79, tier: scanv1.Tier_TIER_HOME_COOK, items: 6},
		{fridge: "scrappy", score: 39, tier: scanv1.Tier_TIER_SCRAPPY, items: 5},
		{fridge: "fries_in_the_bag", score: 29, tier: scanv1.Tier_TIER_FRIES_IN_THE_BAG, items: 5},
		{fridge: "empty", score: 0, tier: scanv1.Tier_TIER_FRIES_IN_THE_BAG, items: 0},
	}
	for _, c := range cases {
		t.Run(c.fridge, func(t *testing.T) {
			t.Parallel()
			mock, err := vision.NewMock(c.fridge, "")
			if err != nil {
				t.Fatalf("mock: %v", err)
			}
			srv := scanStack(t, scanFixture{vision: mock})
			maya := scanner(t, srv, "test_maya")

			got, err := maya.CreateScan(t.Context(), scanRequest(fridgePhoto()))
			if err != nil {
				t.Fatalf("create scan: %v", err)
			}
			v := got.Msg.Verdict
			read, err := maya.GetScan(t.Context(), connect.NewRequest(&scanv1.GetScanRequest{ScanId: v.GetScanId()}))
			if err != nil {
				t.Fatalf("get scan: %v", err)
			}
			last, err := maya.GetLastScan(t.Context(), connect.NewRequest(&scanv1.GetLastScanRequest{}))
			if err != nil {
				t.Fatalf("get last scan: %v", err)
			}

			if v.GetScore() != c.score || v.GetTier() != c.tier || len(v.GetIngredients()) != c.items {
				t.Errorf("score %d, tier %v, %d ingredients; want %d, %v, %d",
					v.GetScore(), v.GetTier(), len(v.GetIngredients()), c.score, c.tier, c.items)
			}
			if v.GetScanId() == 0 || v.GetRoast() == "" || v.GetCreatedAt() == nil {
				t.Errorf("scan id %d, roast %q, created at %v; want all three", v.GetScanId(), v.GetRoast(), v.GetCreatedAt())
			}
			if !proto.Equal(read.Msg.Verdict, v) || !proto.Equal(last.Msg.Verdict, v) {
				t.Errorf("read back as %v and %v, want %v", read.Msg.Verdict, last.Msg.Verdict, v)
			}
		})
	}
}

// TestAModelOutageAnswersUnavailable: the judging screen offers a retry on unavailable, so a
// failed read must arrive as exactly that, in words for the player, with nothing stored.
func TestAModelOutageAnswersUnavailable(t *testing.T) {
	t.Parallel()
	mock, err := vision.NewMock("error", "")
	if err != nil {
		t.Fatalf("mock: %v", err)
	}
	srv := scanStack(t, scanFixture{vision: mock})
	maya := scanner(t, srv, "test_maya")

	_, err = maya.CreateScan(t.Context(), scanRequest(fridgePhoto()))

	cerr := asConnectErr(t, err)
	if cerr.Code() != connect.CodeUnavailable || cerr.Message() != vision.ErrUnavailable.Error() {
		t.Errorf("got %v %q, want unavailable %q", cerr.Code(), cerr.Message(), vision.ErrUnavailable.Error())
	}
	stored := count(t, srv, `select count(*) from app.scan`)
	if stored != 0 {
		t.Errorf("scans stored = %d, want 0", stored)
	}
}

// TestAnotherPlayersScanIsNotFoundAndStaysUntouched: every rpc that names a scan is fenced. A
// stranger's scan answers exactly like one that does not exist, and a refused edit moves nothing.
func TestAnotherPlayersScanIsNotFoundAndStaysUntouched(t *testing.T) {
	t.Parallel()
	srv := scanStack(t, scanFixture{vision: newFakeVision(chefFridge)})
	maya := scanner(t, srv, "test_maya")
	sam := scanner(t, srv, "test_sam")
	mine := createScan(t, maya)

	rpcs := map[string]func(id int64) error{
		"get scan": func(id int64) error {
			_, err := sam.GetScan(t.Context(), connect.NewRequest(&scanv1.GetScanRequest{ScanId: id}))
			return err
		},
		"edit ingredients": func(id int64) error {
			_, err := sam.EditIngredients(t.Context(), connect.NewRequest(&scanv1.EditIngredientsRequest{
				ScanId:      id,
				Ingredients: []*scanv1.Ingredient{{Name: "Ketchup", Category: scanv1.Category_CATEGORY_CONDIMENT}},
			}))
			return err
		},
	}
	for name, call := range rpcs {
		for _, target := range []struct {
			what string
			id   int64
		}{{"maya's scan", mine.GetScanId()}, {"a scan that does not exist", mine.GetScanId() + 1000}} {
			cerr := asConnectErr(t, call(target.id))
			if cerr.Code() != connect.CodeNotFound || cerr.Message() != repository.ErrNotFound.Error() {
				t.Errorf("%s on %s: got %v %q, want not_found", name, target.what, cerr.Code(), cerr.Message())
			}
		}
	}
	last, err := sam.GetLastScan(t.Context(), connect.NewRequest(&scanv1.GetLastScanRequest{}))
	if err != nil {
		t.Fatalf("sam's last scan: %v", err)
	}

	if last.Msg.Verdict != nil {
		t.Errorf("sam's last scan = %v, want none (maya's is not his)", last.Msg.Verdict)
	}
	after, err := maya.GetScan(t.Context(), connect.NewRequest(&scanv1.GetScanRequest{ScanId: mine.GetScanId()}))
	if err != nil {
		t.Fatalf("re-read maya's scan: %v", err)
	}
	if !proto.Equal(after.Msg.Verdict, mine) {
		t.Errorf("maya's scan moved: %v, want %v", after.Msg.Verdict, mine)
	}
	if v := itemsVersion(t, srv, mine.GetScanId()); v != 1 {
		t.Errorf("items version = %d, want 1", v)
	}
}

// TestARefusedPhotoSetIsAViolationOnPhotos: the camera screen shows its error under the photos,
// and a refused set must never reach the model or store anything.
func TestARefusedPhotoSetIsAViolationOnPhotos(t *testing.T) {
	t.Parallel()
	eyes := newFakeVision(chefFridge)
	srv := scanStack(t, scanFixture{vision: eyes})
	maya := scanner(t, srv, "test_maya")
	tooBig := append(slices.Clone(testJPEG), make([]byte, 2<<20)...)
	cases := []struct {
		name   string
		photos []*scanv1.Photo
		want   string
	}{
		{name: "none", photos: nil, want: "a scan takes 1 to 3 photos"},
		{
			name: "four",
			photos: []*scanv1.Photo{
				fridgePhoto(), photo(scanv1.PhotoKind_PHOTO_KIND_FREEZER),
				photo(scanv1.PhotoKind_PHOTO_KIND_PANTRY), fridgePhoto(),
			},
			want: "a scan takes 1 to 3 photos",
		},
		{
			name:   "not a JPEG",
			photos: []*scanv1.Photo{{Jpeg: []byte("GIF89a, a test photo"), Kind: scanv1.PhotoKind_PHOTO_KIND_FRIDGE}},
			want:   "photo 1 isn't a JPEG",
		},
		{
			name:   "too big",
			photos: []*scanv1.Photo{fridgePhoto(), {Jpeg: tooBig, Kind: scanv1.PhotoKind_PHOTO_KIND_PANTRY}},
			want:   "photo 2 is over 2 MB",
		},
		{
			name:   "no kind",
			photos: []*scanv1.Photo{{Jpeg: testJPEG}},
			want:   "photo 1 must be of the fridge, freezer or pantry",
		},
		{
			name:   "a kind newer than the server",
			photos: []*scanv1.Photo{{Jpeg: testJPEG, Kind: scanv1.PhotoKind(9)}},
			want:   "photo 1 must be of the fridge, freezer or pantry",
		},
	}
	// The group returns once every parallel case has finished, so the checks below see them all.
	t.Run("cases", func(t *testing.T) {
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()

				_, err := maya.CreateScan(t.Context(), scanRequest(c.photos...))

				if connect.CodeOf(err) != connect.CodeInvalidArgument {
					t.Fatalf("got code %v (%v), want invalid_argument", connect.CodeOf(err), err)
				}
				got := fieldViolations(t, err)
				want := map[string]string{"photos": c.want}
				if !maps.Equal(got, want) {
					t.Errorf("violations = %v, want %v", got, want)
				}
			})
		}
	})

	asked := len(eyes.calls())
	stored := count(t, srv, `select count(*) from app.scan`)
	if asked != 0 || stored != 0 {
		t.Errorf("the model was asked %d times and %d scans were stored, want neither", asked, stored)
	}
}

// TestScansAreLimitedPerPlayer: each scan spends model time, so a player stops at the limit,
// while another player on the same network keeps their own. A refused photo set spends nothing.
func TestScansAreLimitedPerPlayer(t *testing.T) {
	t.Parallel()
	srv := scanStack(t, scanFixture{
		vision: newFakeVision(chefFridge),
		limit:  handlers.NewLimiter(2, time.Hour),
	})
	maya := scanner(t, srv, "test_maya")
	sam := scanner(t, srv, "test_sam")

	// Every client here calls from 127.0.0.1, so a limit by address would stop sam too.
	_, refused := maya.CreateScan(t.Context(), scanRequest())
	var mayas []error
	for range 3 {
		_, err := maya.CreateScan(t.Context(), scanRequest(fridgePhoto()))
		mayas = append(mayas, err)
	}
	_, sams := sam.CreateScan(t.Context(), scanRequest(fridgePhoto()))

	if connect.CodeOf(refused) != connect.CodeInvalidArgument {
		t.Fatalf("an empty photo set: got code %v (%v), want invalid_argument", connect.CodeOf(refused), refused)
	}
	if mayas[0] != nil || mayas[1] != nil {
		t.Fatalf("maya's scans within the limit: %v, %v", mayas[0], mayas[1])
	}
	cerr := asConnectErr(t, mayas[2])
	if cerr.Code() != connect.CodeResourceExhausted || cerr.Message() != handlers.ErrTooManyRequests.Error() {
		t.Errorf("maya's third scan: got %v %q, want resource_exhausted", cerr.Code(), cerr.Message())
	}
	if sams != nil {
		t.Errorf("sam's first scan: %v, want it judged", sams)
	}
}

// TestEditingTheListChangesItAndNeverTheScore: fixing the list re-runs the recipes, never the
// score. An edit is stored trimmed, bumps the version the recipes regenerate on, and sending the
// same list again changes nothing at all.
func TestEditingTheListChangesItAndNeverTheScore(t *testing.T) {
	t.Parallel()
	srv := scanStack(t, scanFixture{vision: newFakeVision(chefFridge)})
	maya := scanner(t, srv, "test_maya")
	mine := createScan(t, maya)
	fix := func(ingredients ...*scanv1.Ingredient) *scanv1.Verdict {
		t.Helper()
		got, err := maya.EditIngredients(t.Context(), connect.NewRequest(&scanv1.EditIngredientsRequest{
			ScanId:      mine.GetScanId(),
			Ingredients: ingredients,
		}))
		if err != nil {
			t.Fatalf("edit ingredients: %v", err)
		}
		return got.Msg.Verdict
	}

	edited := fix(
		&scanv1.Ingredient{Name: " Ketchup ", Category: scanv1.Category_CATEGORY_CONDIMENT, Quantity: " 1 bottle "},
		&scanv1.Ingredient{Name: "Chicken", Category: scanv1.Category_CATEGORY_PROTEIN, Quantity: "2 breasts"},
	)
	editedVersion := itemsVersion(t, srv, mine.GetScanId())
	again := fix(edited.GetIngredients()...)
	againVersion := itemsVersion(t, srv, mine.GetScanId())
	emptied := fix()
	read, err := maya.GetScan(t.Context(), connect.NewRequest(&scanv1.GetScanRequest{ScanId: mine.GetScanId()}))
	if err != nil {
		t.Fatalf("get scan: %v", err)
	}

	want := []*scanv1.Ingredient{
		{Name: "Ketchup", Category: scanv1.Category_CATEGORY_CONDIMENT, Quantity: "1 bottle"},
		{Name: "Chicken", Category: scanv1.Category_CATEGORY_PROTEIN, Quantity: "2 breasts"},
	}
	if !slices.EqualFunc(edited.GetIngredients(), want, equalIngredient) {
		t.Errorf("edited list = %v, want %v", edited.GetIngredients(), want)
	}
	if editedVersion != 2 || againVersion != 2 {
		t.Errorf("items version after the edit %d, after the same list again %d; want 2 and 2",
			editedVersion, againVersion)
	}
	if !proto.Equal(again, edited) {
		t.Errorf("the same list again answered %v, want %v", again, edited)
	}
	if len(emptied.GetIngredients()) != 0 || !proto.Equal(read.Msg.Verdict, emptied) {
		t.Errorf("emptied list = %v, read back as %v; want none, the same", emptied.GetIngredients(), read.Msg.Verdict)
	}
	for _, v := range []*scanv1.Verdict{edited, again, emptied} {
		if v.GetScore() != mine.GetScore() || v.GetTier() != mine.GetTier() || v.GetRoast() != mine.GetRoast() {
			t.Errorf("score %d, tier %v, roast %q after an edit; want %d, %v, %q",
				v.GetScore(), v.GetTier(), v.GetRoast(), mine.GetScore(), mine.GetTier(), mine.GetRoast())
		}
	}
}

// TestAnInvalidEditIsRefusedIngredientByIngredient: the edit sheet marks each bad row, so every
// problem arrives at once under its ingredient's index, and the stored list does not move.
func TestAnInvalidEditIsRefusedIngredientByIngredient(t *testing.T) {
	t.Parallel()
	srv := scanStack(t, scanFixture{vision: newFakeVision(chefFridge)})
	maya := scanner(t, srv, "test_maya")
	mine := createScan(t, maya)
	tooMany := make([]*scanv1.Ingredient, 41)
	for i := range tooMany {
		tooMany[i] = &scanv1.Ingredient{Name: "Egg", Category: scanv1.Category_CATEGORY_PROTEIN}
	}
	cases := []struct {
		name string
		req  *scanv1.EditIngredientsRequest
		want map[string]string
	}{
		{
			name: "no scan id",
			req:  &scanv1.EditIngredientsRequest{},
			want: map[string]string{"scan_id": "scan id is required"},
		},
		{
			name: "forty-one ingredients",
			req:  &scanv1.EditIngredientsRequest{ScanId: mine.GetScanId(), Ingredients: tooMany},
			want: map[string]string{"ingredients": "ingredients must be at most 40 items"},
		},
		{
			name: "a problem in each of five ingredients",
			req: &scanv1.EditIngredientsRequest{ScanId: mine.GetScanId(), Ingredients: []*scanv1.Ingredient{
				{Name: "  ", Category: scanv1.Category_CATEGORY_CONDIMENT},
				{Name: "Eggs"},
				{Name: "Eggs", Category: scanv1.Category(99)},
				{Name: "Eggs", Category: scanv1.Category_CATEGORY_PROTEIN, Quantity: strings.Repeat("x", 21)},
				{Name: strings.Repeat("a", 41), Category: scanv1.Category_CATEGORY_PROTEIN},
			}},
			want: map[string]string{
				"ingredients[0].name":     "name is required",
				"ingredients[1].category": "category is required",
				"ingredients[2].category": "category must be protein or vegetable or fruit or grain or dairy or condiment or drink or snack or leftover or other",
				"ingredients[3].quantity": "quantity must be at most 20 characters",
				"ingredients[4].name":     "name must be at most 40 characters",
			},
		},
	}
	// The group returns once every parallel case has finished, so the checks below see them all.
	t.Run("cases", func(t *testing.T) {
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()

				_, err := maya.EditIngredients(t.Context(), connect.NewRequest(c.req))

				if connect.CodeOf(err) != connect.CodeInvalidArgument {
					t.Fatalf("got code %v (%v), want invalid_argument", connect.CodeOf(err), err)
				}
				got := fieldViolations(t, err)
				if !maps.Equal(got, c.want) {
					t.Errorf("violations = %v, want %v", got, c.want)
				}
			})
		}
	})
	_, noID := maya.GetScan(t.Context(), connect.NewRequest(&scanv1.GetScanRequest{}))

	got := fieldViolations(t, noID)
	want := map[string]string{"scan_id": "scan id is required"}
	if !maps.Equal(got, want) {
		t.Errorf("get scan without an id: violations = %v, want %v", got, want)
	}
	after, err := maya.GetScan(t.Context(), connect.NewRequest(&scanv1.GetScanRequest{ScanId: mine.GetScanId()}))
	if err != nil {
		t.Fatalf("re-read the scan: %v", err)
	}
	if !proto.Equal(after.Msg.Verdict, mine) || itemsVersion(t, srv, mine.GetScanId()) != 1 {
		t.Errorf("the scan moved: %v, want %v", after.Msg.Verdict, mine)
	}
}

// TestJunkFromTheModelIsDropped: the model's list is untrusted input. What breaks the request rules
// is dropped rather than failing the scan, the rest is kept and scored, and the log gets the count,
// never the content.
func TestJunkFromTheModelIsDropped(t *testing.T) {
	t.Parallel()
	listed := []vision.Item{
		{Name: "Eggs", Category: "protein", Quantity: "x6"},
		{Name: "   ", Category: "vegetable"},
		{Name: strings.Repeat("a", 41), Category: "dairy"},
		{Name: "Steak", Category: "meat"},
		{Name: "  Spinach ", Category: " Vegetable ", Quantity: " 1 bag "},
		{Name: "Rice", Category: "grain", Quantity: strings.Repeat("x", 21)},
	}
	// Forty snacks after those: the two good items above leave room for 38 of them.
	for i := range 40 {
		listed = append(listed, vision.Item{Name: fmt.Sprintf("Snack %d", i+1), Category: "snack"})
	}
	var logs lockedBuffer
	srv := scanStack(t, scanFixture{
		vision: newFakeVision(vision.Fridge{Items: listed, Roast: "A lot of snacks."}),
		log:    jsonLog(&logs),
	})
	maya := scanner(t, srv, "test_maya")

	v := createScan(t, maya)

	kept := v.GetIngredients()
	if len(kept) != 40 {
		t.Fatalf("kept %d ingredients, want 40", len(kept))
	}
	first := []*scanv1.Ingredient{
		{Name: "Eggs", Category: scanv1.Category_CATEGORY_PROTEIN, Quantity: "x6"},
		{Name: "Spinach", Category: scanv1.Category_CATEGORY_VEGETABLE, Quantity: "1 bag"},
		{Name: "Snack 1", Category: scanv1.Category_CATEGORY_SNACK},
	}
	if !slices.EqualFunc(kept[:3], first, equalIngredient) || kept[39].GetName() != "Snack 38" {
		t.Errorf("kept %v ... %v, want %v ... Snack 38", kept[:3], kept[39], first)
	}
	// Eggs 15 and spinach 15, the first snack 1, two kinds of real food 8, and the meal bonus 20.
	if v.GetScore() != 59 || v.GetTier() != scanv1.Tier_TIER_SCRAPPY {
		t.Errorf("score %d, tier %v; want 59, scrappy", v.GetScore(), v.GetTier())
	}
	line := logs.String()
	if !strings.Contains(line, `"msg":"dropped items the model listed"`) || !strings.Contains(line, `"dropped":6`) {
		t.Errorf("log = %s, want the dropped count, 6", line)
	}
	if strings.Contains(line, "Steak") || strings.Contains(line, "Rice") {
		t.Errorf("log = %s, want no item from the list in it", line)
	}
}

// TestTheModelReadsThePhotosAtThePlayersRoastLevel: the player picks how hard the roast hits,
// and every photo they took reaches the model as sent.
func TestTheModelReadsThePhotosAtThePlayersRoastLevel(t *testing.T) {
	t.Parallel()
	eyes := newFakeVision(chefFridge)
	srv := scanStack(t, scanFixture{vision: eyes})
	maya := scanner(t, srv, "test_maya")
	_, err := srv.pool.Exec(t.Context(),
		`update app.account set roast_level = 'extra_crispy' where handle = 'test_maya'`)
	if err != nil {
		t.Fatalf("turn the roast up: %v", err)
	}
	pantry := append(slices.Clone(testJPEG), " of the pantry"...)

	createScan(t, maya, fridgePhoto(), &scanv1.Photo{Jpeg: pantry, Kind: scanv1.PhotoKind_PHOTO_KIND_PANTRY})

	calls := eyes.calls()
	if len(calls) != 1 {
		t.Fatalf("the model was asked %d times, want once", len(calls))
	}
	if calls[0].level != vision.RoastExtraCrispy {
		t.Errorf("roast level = %q, want extra_crispy", calls[0].level)
	}
	if !slices.EqualFunc(calls[0].photos, [][]byte{testJPEG, pantry}, bytes.Equal) {
		t.Errorf("photos = %q, want the two sent", calls[0].photos)
	}
}

// TestGetLastScanIsUnsetUntilTheFirstScanThenTheNewest: the kitchen shows the latest verdict, and
// a first launch has none to show rather than an error.
func TestGetLastScanIsUnsetUntilTheFirstScanThenTheNewest(t *testing.T) {
	t.Parallel()
	srv := scanStack(t, scanFixture{vision: newFakeVision(chefFridge)})
	maya := scanner(t, srv, "test_maya")
	lastScan := func() *scanv1.Verdict {
		t.Helper()
		got, err := maya.GetLastScan(t.Context(), connect.NewRequest(&scanv1.GetLastScanRequest{}))
		if err != nil {
			t.Fatalf("get last scan: %v", err)
		}
		return got.Msg.Verdict
	}

	before := lastScan()
	createScan(t, maya)
	second := createScan(t, maya)
	after := lastScan()

	if before != nil {
		t.Errorf("before any scan: %v, want none", before)
	}
	if !proto.Equal(after, second) {
		t.Errorf("last scan = %v, want the second, %v", after, second)
	}
}

// TestEveryScanRpcNeedsASession: every rpc acts for the account in the session, so one without a
// session is unauthenticated, which the app reads as "back to onboarding", and never reaches the
// model.
func TestEveryScanRpcNeedsASession(t *testing.T) {
	t.Parallel()
	eyes := newFakeVision(chefFridge)
	srv := scanStack(t, scanFixture{vision: eyes})
	stranger := scanv1connect.NewScanServiceClient(http.DefaultClient, srv.url)
	rpcs := map[string]func() error{
		"create scan": func() error {
			_, err := stranger.CreateScan(t.Context(), scanRequest(fridgePhoto()))
			return err
		},
		"get scan": func() error {
			_, err := stranger.GetScan(t.Context(), connect.NewRequest(&scanv1.GetScanRequest{ScanId: 1}))
			return err
		},
		"get last scan": func() error {
			_, err := stranger.GetLastScan(t.Context(), connect.NewRequest(&scanv1.GetLastScanRequest{}))
			return err
		},
		"edit ingredients": func() error {
			_, err := stranger.EditIngredients(t.Context(), connect.NewRequest(&scanv1.EditIngredientsRequest{ScanId: 1}))
			return err
		},
	}

	for name, call := range rpcs {
		cerr := asConnectErr(t, call())
		if cerr.Code() != connect.CodeUnauthenticated || cerr.Message() != auth.ErrNotAuthenticated.Error() {
			t.Errorf("%s: got %v %q, want unauthenticated", name, cerr.Code(), cerr.Message())
		}
	}
	asked := len(eyes.calls())
	if asked != 0 {
		t.Errorf("the model was asked %d times, want never", asked)
	}
}

// scanFixture is what a scan stack runs with: the model, the limit (nil limits nothing) and the
// service's log (nil discards).
type scanFixture struct {
	vision scan.Vision
	limit  *handlers.Limiter
	log    *slog.Logger
}

// scanStack mounts ScanService beside AccountService, which makes the players, the way main.go
// mounts it, read cap included.
func scanStack(t *testing.T, f scanFixture) stack {
	t.Helper()
	return sessionStack(t, func(mux *http.ServeMux, repo *repository.Repository, authn *auth.Auth) {
		mux.Handle(accountv1connect.NewAccountServiceHandler(
			account.NewAccount(account.Config{Repo: repo, Auth: authn})))
		svc := scan.NewScan(scan.Config{Repo: repo, Auth: authn, Vision: f.vision, Limit: f.limit, Log: f.log})
		mux.Handle(scanv1connect.NewScanServiceHandler(svc, connect.WithReadMaxBytes(4<<20)))
	})
}

// scanner is a player with an account, holding the scan service as their phone calls it.
func scanner(t *testing.T, srv stack, handle string) scanv1connect.ScanServiceClient {
	t.Helper()
	p := player(t, srv.url, handle)
	return scanv1connect.NewScanServiceClient(&http.Client{Jar: p.jar}, srv.url)
}

// createScan judges photos (one of the fridge, unless the test sends its own) and returns the
// verdict.
func createScan(t *testing.T, c scanv1connect.ScanServiceClient, photos ...*scanv1.Photo) *scanv1.Verdict {
	t.Helper()
	if len(photos) == 0 {
		photos = []*scanv1.Photo{fridgePhoto()}
	}
	got, err := c.CreateScan(t.Context(), scanRequest(photos...))
	if err != nil {
		t.Fatalf("create scan: %v", err)
	}
	return got.Msg.Verdict
}

func scanRequest(photos ...*scanv1.Photo) *connect.Request[scanv1.CreateScanRequest] {
	return connect.NewRequest(&scanv1.CreateScanRequest{Photos: photos})
}

func fridgePhoto() *scanv1.Photo {
	return photo(scanv1.PhotoKind_PHOTO_KIND_FRIDGE)
}

func photo(kind scanv1.PhotoKind) *scanv1.Photo {
	return &scanv1.Photo{Jpeg: testJPEG, Kind: kind}
}

func equalIngredient(a, b *scanv1.Ingredient) bool {
	return proto.Equal(a, b)
}

// itemsVersion is the version the recipes regenerate on, which the wire does not carry.
func itemsVersion(t *testing.T, srv stack, scanID int64) int {
	t.Helper()
	var v int
	err := srv.pool.QueryRow(t.Context(),
		`select items_version from app.scan where scan_id = $1`, scanID).Scan(&v)
	if err != nil {
		t.Fatalf("items version: %v", err)
	}
	return v
}

func count(t *testing.T, srv stack, query string) int {
	t.Helper()
	var n int
	err := srv.pool.QueryRow(t.Context(), query).Scan(&n)
	if err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	return n
}

// fakeVision reads every kitchen as the list a test wrote, at once, and records what it was sent.
// It is safe because the model is the true edge here: the handler, the scoring and the database
// behind them are all real.
type fakeVision struct {
	fridge vision.Fridge

	mu    sync.Mutex
	reads []visionRead
}

// visionRead is one call the fake answered.
type visionRead struct {
	photos [][]byte
	level  vision.RoastLevel
}

func newFakeVision(fridge vision.Fridge) *fakeVision {
	return &fakeVision{fridge: fridge}
}

func (f *fakeVision) ReadFridge(_ context.Context, photos [][]byte, level vision.RoastLevel) (vision.Fridge, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads = append(f.reads, visionRead{photos: photos, level: level})
	return vision.Fridge{Items: slices.Clone(f.fridge.Items), Roast: f.fridge.Roast}, nil
}

func (f *fakeVision) calls() []visionRead {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.reads)
}
