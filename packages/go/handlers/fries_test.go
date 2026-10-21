// Author: Caden Lund
package handlers_test

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"math"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/alexedwards/scs/pgxstore"
	"github.com/alexedwards/scs/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/proto"

	"github.com/bengutow1/csc4330_project5/packages/go/apigen/account/v1/accountv1connect"
	friesv1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/fries/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/apigen/fries/v1/friesv1connect"
	"github.com/bengutow1/csc4330_project5/packages/go/auth"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers/account"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers/fries"
	"github.com/bengutow1/csc4330_project5/packages/go/logging"
	"github.com/bengutow1/csc4330_project5/packages/go/places"
	"github.com/bengutow1/csc4330_project5/packages/go/points"
	"github.com/bengutow1/csc4330_project5/packages/go/repository"
)

// campusLat and campusLng are where the players stand. The mock sets its street around any point.
const (
	campusLat = 30.4133
	campusLng = -91.1800
)

// The chips, by the names the app gives them.
const (
	openNow   = friesv1.SpotFilter_SPOT_FILTER_OPEN_NOW
	lateNight = friesv1.SpotFilter_SPOT_FILTER_LATE_NIGHT
	cheap     = friesv1.SpotFilter_SPOT_FILTER_CHEAP
	fastFood  = friesv1.SpotFilter_SPOT_FILTER_FAST_FOOD
	walkable  = friesv1.SpotFilter_SPOT_FILTER_WALKABLE
)

// byDistance is the mock's street as ListSpots must list it: closest first, and Main Street Pizza
// (mock-06) before Green Bowl (mock-07), both 2,400 m away, by spot id. The mock answers in
// neither order.
var byDistance = []string{
	"mock-01", "mock-02", "mock-03", "mock-04", "mock-05",
	"mock-06", "mock-07", "mock-08", "mock-09", "mock-10",
}

// TestListSpotsListsTheStreetClosestFirst: the whole order, with its tie, and two spots read in
// full, so every field the server works out reaches the wire, boundaries included.
func TestListSpotsListsTheStreetClosestFirst(t *testing.T) {
	t.Parallel()
	srv := friesStack(t, usualStreet(t), fries.Limits{}, nil)
	maya := friesPlayer(t, srv, "test_maya")

	got := listSpots(t, maya, atCampus(50))

	ids := spotIDs(got.Spots)

	if !slices.Equal(ids, byDistance) {
		t.Fatalf("order = %v, want %v", ids, byDistance)
	}
	if got.NextCursor != "" {
		t.Errorf("next_cursor = %q, want \"\": all ten fit", got.NextCursor)
	}
	owl := got.Spots[2]
	if owl.Name != "Night Owl Fries" || owl.DistanceMeters != 1200 || !owl.OpenNow ||
		owl.ClosesAt != "02:00" || owl.PriceLevel != 1 || !owl.FastFood ||
		owl.Address != "77 Maple Dr" || owl.Category != "Fast food" {
		t.Errorf("night owl = %v", owl)
	}
	if !owl.LateNight || !owl.Walkable {
		t.Errorf("night owl: late night %v, walkable %v, want both (open to 2 am, exactly 1,200 m)",
			owl.LateNight, owl.Walkable)
	}
	if math.Abs(owl.Lat-campusLat) > 0.05 || math.Abs(owl.Lng-campusLng) > 0.05 || owl.Lat == campusLat {
		t.Errorf("night owl at %v, %v, want a point near campus but not on it", owl.Lat, owl.Lng)
	}
	drumstick := got.Spots[4]
	if drumstick.Name != "Drumstick Drive-Thru" || drumstick.LateNight || drumstick.Walkable {
		t.Errorf("%s: late night %v, walkable %v, want neither (closes at 23:00, 1,900 m away)",
			drumstick.Name, drumstick.LateNight, drumstick.Walkable)
	}
}

// TestEachFilterKeepsOnlyTheSpotsThatPassIt: each chip keeps exactly these spots of the usual
// street, and several chips keep only what passes every one.
func TestEachFilterKeepsOnlyTheSpotsThatPassIt(t *testing.T) {
	t.Parallel()
	srv := friesStack(t, usualStreet(t), fries.Limits{}, nil)
	maya := friesPlayer(t, srv, "test_maya")
	cases := []struct {
		name    string
		filters []friesv1.SpotFilter
		want    []string
	}{
		{"open now: all but Early Bird Burgers", []friesv1.SpotFilter{openNow}, byDistance[1:]},
		{"late night: open past 23:00", []friesv1.SpotFilter{lateNight}, []string{"mock-03", "mock-06"}},
		{"cheap: price level 1", []friesv1.SpotFilter{cheap}, []string{"mock-03", "mock-04"}},
		{"fast food, open or not", []friesv1.SpotFilter{fastFood}, []string{"mock-01", "mock-03", "mock-05"}},
		{"walkable: 1,200 m or nearer", []friesv1.SpotFilter{walkable}, []string{"mock-01", "mock-02", "mock-03"}},
		{"open fast food", []friesv1.SpotFilter{openNow, fastFood}, []string{"mock-03", "mock-05"}},
		{"open and walkable", []friesv1.SpotFilter{openNow, walkable}, []string{"mock-02", "mock-03"}},
		{"every chip", []friesv1.SpotFilter{openNow, lateNight, cheap, fastFood, walkable}, []string{"mock-03"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			req := atCampus(50)
			req.Filters = c.filters

			got := spotIDs(listSpots(t, maya, req).Spots)

			if !slices.Equal(got, c.want) {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

// TestPagingWalksTheListToTheEndWithNoRepeats: pages of three split the 2,400 m tie across a page
// boundary, and the nearest fries ride every page, the ones without them included.
func TestPagingWalksTheListToTheEndWithNoRepeats(t *testing.T) {
	t.Parallel()
	srv := friesStack(t, usualStreet(t), fries.Limits{}, nil)
	maya := friesPlayer(t, srv, "test_maya")
	want := [][]string{
		{"mock-01", "mock-02", "mock-03"},
		{"mock-04", "mock-05", "mock-06"},
		{"mock-07", "mock-08", "mock-09"},
		{"mock-10"},
	}

	var pages [][]string
	req := atCampus(3)
	for {
		if len(pages) > 5 {
			t.Fatalf("still paging after %d pages: %v", len(pages), pages)
		}
		got := listSpots(t, maya, req)
		pages = append(pages, spotIDs(got.Spots))
		if got.NearestFries.GetSpotId() != "mock-03" {
			t.Errorf("page %d: nearest fries = %q, want mock-03", len(pages), got.NearestFries.GetSpotId())
		}
		if got.NextCursor == "" {
			break
		}
		req.Cursor = got.NextCursor
	}

	if !slices.EqualFunc(pages, want, slices.Equal) {
		t.Errorf("pages = %v, want %v", pages, want)
	}
}

// TestAPagePastTheEndIsEmpty: a cursor sent with filters that leave nothing after it answers an
// empty last page, rather than starting over and looping an infinite scroll.
func TestAPagePastTheEndIsEmpty(t *testing.T) {
	t.Parallel()
	srv := friesStack(t, usualStreet(t), fries.Limits{}, nil)
	maya := friesPlayer(t, srv, "test_maya")
	first := listSpots(t, maya, atCampus(9))
	if first.NextCursor == "" {
		t.Fatal("no cursor after nine of ten spots")
	}

	// Both late-night spots sit before the ninth.
	req := atCampus(9)
	req.Filters = []friesv1.SpotFilter{lateNight}
	req.Cursor = first.NextCursor
	got := listSpots(t, maya, req)

	if len(got.Spots) != 0 || got.NextCursor != "" {
		t.Errorf("got %v and next_cursor %q, want an empty last page", spotIDs(got.Spots), got.NextCursor)
	}
}

// TestNearestFriesIsTheClosestOpenFastFood: Early Bird Burgers is fast food and nearer, but it
// has closed, so the chip goes to Night Owl Fries; on a street where everything has closed there
// are no nearest fries at all.
func TestNearestFriesIsTheClosestOpenFastFood(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		street *fakeSpots
		want   string
	}{
		{"the usual street", usualStreet(t), "mock-03"},
		{"everything closed", closedStreet(t), ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			srv := friesStack(t, c.street, fries.Limits{}, nil)
			maya := friesPlayer(t, srv, "test_maya")
			req := atCampus(50)
			// The filters never hide the nearest fries.
			req.Filters = []friesv1.SpotFilter{walkable}

			got := listSpots(t, maya, req)

			if got.NearestFries.GetSpotId() != c.want {
				t.Errorf("nearest fries = %q, want %q", got.NearestFries.GetSpotId(), c.want)
			}
		})
	}
}

// TestTheFiltersNeverHideTheNearestFries: the Nearest fries chip shows whichever chips are on, so
// a burger joint the Cheap chip leaves out of the list is still the nearest fries.
func TestTheFiltersNeverHideTheNearestFries(t *testing.T) {
	t.Parallel()
	street := &fakeSpots{found: []places.Place{
		{ID: "fake-burgers", Name: "Pricey Burgers", DistanceMeters: 100, OpenNow: true, FastFood: true, PriceLevel: 3},
		{ID: "fake-noodles", Name: "Cheap Noodles", DistanceMeters: 200, OpenNow: true, PriceLevel: 1},
	}}
	srv := friesStack(t, street, fries.Limits{}, nil)
	maya := friesPlayer(t, srv, "test_maya")

	got := listSpots(t, maya, withFilters(atCampus(0), cheap))

	ids := spotIDs(got.Spots)

	if !slices.Equal(ids, []string{"fake-noodles"}) {
		t.Errorf("spots = %v, want only fake-noodles", ids)
	}
	if got.NearestFries.GetSpotId() != "fake-burgers" {
		t.Errorf("nearest fries = %q, want fake-burgers (filtered out of the list, but nearest)",
			got.NearestFries.GetSpotId())
	}
}

// TestWithLocationOffTheQuerySaysWhereToLook: no point, so the provider searches by the words
// alone. Nothing has a distance, nothing is walkable, the list runs in spot-id order, and there
// are no nearest fries, because nothing is near anything.
func TestWithLocationOffTheQuerySaysWhereToLook(t *testing.T) {
	t.Parallel()
	street := usualStreet(t)
	srv := friesStack(t, street, fries.Limits{}, nil)
	maya := friesPlayer(t, srv, "test_maya")

	got := listSpots(t, maya, &friesv1.ListSpotsRequest{Query: "  tacos near the stadium ", Limit: 50})

	calls := street.calls()

	if !slices.Equal(calls, []string{"search tacos near the stadium"}) {
		t.Errorf("provider calls = %q, want one trimmed search", calls)
	}
	ids := spotIDs(got.Spots)
	if !slices.Equal(ids, byDistance) {
		t.Errorf("order = %v, want spot-id order %v", ids, byDistance)
	}
	for _, s := range got.Spots {
		if s.DistanceMeters != 0 || s.Walkable {
			t.Errorf("%s: %d m away, walkable %v, want 0 m and not walkable", s.Name, s.DistanceMeters, s.Walkable)
		}
	}
	if got.NearestFries != nil {
		t.Errorf("nearest fries = %s, want none without a point", got.NearestFries.GetSpotId())
	}
}

// TestABadListSpotsRequestIsRefusedFieldByField: each problem comes back on its own field, all of
// them in one answer, and a refused request never reaches the provider.
func TestABadListSpotsRequestIsRefusedFieldByField(t *testing.T) {
	t.Parallel()
	const choices = "filter must be open_now, late_night, cheap, fast_food or walkable"
	cases := []struct {
		name string
		req  *friesv1.ListSpotsRequest
		want map[string]string
	}{
		{"a latitude past 90", at(90.5, 0), map[string]string{"lat": "lat must be between -90 and 90"}},
		{"a latitude past -90", at(-91, 0), map[string]string{"lat": "lat must be between -90 and 90"}},
		{"a longitude past 180", at(0, 180.5), map[string]string{"lng": "lng must be between -180 and 180"}},
		{"a longitude past -180", at(0, -181), map[string]string{"lng": "lng must be between -180 and 180"}},
		{"a latitude that is not a number", at(math.NaN(), 0), map[string]string{"lat": "lat must be between -90 and 90"}},
		{"an infinite longitude", at(0, math.Inf(1)), map[string]string{"lng": "lng must be between -180 and 180"}},
		{
			"lat without lng",
			&friesv1.ListSpotsRequest{Lat: proto.Float64(campusLat)},
			map[string]string{"lng": "lng is required with lat"},
		},
		{
			"lng without lat",
			&friesv1.ListSpotsRequest{Lng: proto.Float64(campusLng)},
			map[string]string{"lat": "lat is required with lng"},
		},
		{
			"no point and no query",
			&friesv1.ListSpotsRequest{},
			map[string]string{"query": "query is required when lat and lng are not sent"},
		},
		{
			"no point and a blank query",
			&friesv1.ListSpotsRequest{Query: "   "},
			map[string]string{"query": "query is required when lat and lng are not sent"},
		},
		{
			"a query of 61 characters",
			withQuery(atCampus(0), strings.Repeat("é", 61)),
			map[string]string{"query": "query must be at most 60 characters"},
		},
		{"an unspecified filter", withFilters(atCampus(0), friesv1.SpotFilter_SPOT_FILTER_UNSPECIFIED), map[string]string{"filters[0]": choices}},
		{"a filter this server does not know", withFilters(atCampus(0), friesv1.SpotFilter(99)), map[string]string{"filters[0]": choices}},
		{"a filter twice", withFilters(atCampus(0), cheap, openNow, cheap), map[string]string{"filters[2]": "filter is listed twice"}},
		{
			"walkable with location off",
			withFilters(&friesv1.ListSpotsRequest{Query: "fries"}, walkable),
			map[string]string{"filters[0]": "walkable needs lat and lng"},
		},
		{"a negative limit", atCampus(-1), map[string]string{"limit": "limit must be at least 0"}},
		{
			"a cursor this server never gave out",
			&friesv1.ListSpotsRequest{Lat: proto.Float64(campusLat), Lng: proto.Float64(campusLng), Cursor: "!!!"},
			map[string]string{"cursor": "cursor is not one this server gave out"},
		},
		{
			"several problems at once",
			&friesv1.ListSpotsRequest{
				Lat: proto.Float64(91), Lng: proto.Float64(0), Query: strings.Repeat("a", 61), Limit: -5,
			},
			map[string]string{
				"lat":   "lat must be between -90 and 90",
				"query": "query must be at most 60 characters",
				"limit": "limit must be at least 0",
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			street := usualStreet(t)
			srv := friesStack(t, street, fries.Limits{}, nil)
			maya := friesPlayer(t, srv, "test_maya")

			_, err := maya.ListSpots(t.Context(), connect.NewRequest(c.req))

			if connect.CodeOf(err) != connect.CodeInvalidArgument {
				t.Fatalf("got code %v (%v), want invalid_argument", connect.CodeOf(err), err)
			}
			got := fieldViolations(t, err)
			if !maps.Equal(got, c.want) {
				t.Errorf("violations = %v, want %v", got, c.want)
			}
			calls := street.calls()
			if len(calls) != 0 {
				t.Errorf("provider calls = %q, want none for a refused request", calls)
			}
		})
	}
}

// TestTheEdgesOfTheMapAreAllowed: the bounds are inclusive, so a player at a pole or on the date
// line still gets spots.
func TestTheEdgesOfTheMapAreAllowed(t *testing.T) {
	t.Parallel()
	srv := friesStack(t, usualStreet(t), fries.Limits{}, nil)
	maya := friesPlayer(t, srv, "test_maya")
	for _, req := range []*friesv1.ListSpotsRequest{at(90, 180), at(-90, -180)} {
		_, err := maya.ListSpots(t.Context(), connect.NewRequest(req))
		if err != nil {
			t.Errorf("lat %v, lng %v: %v", req.GetLat(), req.GetLng(), err)
		}
	}
}

// TestTheLimitDefaultsToTwentyAndStopsAtFifty: a page has a size whatever the client sends, and
// a cursor follows whenever there is more.
func TestTheLimitDefaultsToTwentyAndStopsAtFifty(t *testing.T) {
	t.Parallel()
	srv := friesStack(t, longStreet(60), fries.Limits{}, nil)
	maya := friesPlayer(t, srv, "test_maya")
	cases := []struct {
		limit int32
		want  int
	}{
		{0, 20},
		{7, 7},
		{50, 50},
		{51, 50},
		{1000, 50},
	}
	for _, c := range cases {
		got := listSpots(t, maya, atCampus(c.limit))
		if len(got.Spots) != c.want || got.NextCursor == "" {
			t.Errorf("limit %d: %d spots, next_cursor %q, want %d and a cursor", c.limit, len(got.Spots),
				got.NextCursor, c.want)
		}
	}
}

// TestFriesModeNeedsAnAccount: both rpcs answer a fresh install with unauthenticated, which the
// app reads as "back to onboarding", and neither asks the provider or stores a thing.
func TestFriesModeNeedsAnAccount(t *testing.T) {
	t.Parallel()
	street := usualStreet(t)
	srv := friesStack(t, street, fries.Limits{}, nil)
	stranger := friesNewcomer(t, srv)

	_, listErr := stranger.ListSpots(t.Context(), connect.NewRequest(atCampus(0)))
	_, logErr := stranger.LogFryRun(t.Context(), connect.NewRequest(nightOwlRun()))

	for name, err := range map[string]error{"list spots": listErr, "log fry run": logErr} {
		if connect.CodeOf(err) != connect.CodeUnauthenticated {
			t.Errorf("%s: got code %v (%v), want unauthenticated", name, connect.CodeOf(err), err)
		}
	}
	calls := street.calls()
	if len(calls) != 0 {
		t.Errorf("provider calls = %q, want none", calls)
	}
	n := fryRunCount(t, srv.pool)
	if n != 0 {
		t.Errorf("%d fry runs stored, want 0", n)
	}
}

// TestPlacesBeingDownIsUnavailable: the player reads words they can act on and can tap again;
// what went wrong with the provider stays out of the answer.
func TestPlacesBeingDownIsUnavailable(t *testing.T) {
	t.Parallel()
	down := failingSpots(fmt.Errorf("places: nearby: status 503 from the provider: %w", places.ErrUnavailable))
	srv := friesStack(t, down, fries.Limits{}, nil)
	maya := friesPlayer(t, srv, "test_maya")

	for name, req := range map[string]*friesv1.ListSpotsRequest{
		"nearby":       atCampus(0),
		"location off": {Query: "fries"},
	} {
		_, err := maya.ListSpots(t.Context(), connect.NewRequest(req))

		cerr := asConnectErr(t, err)
		if cerr.Code() != connect.CodeUnavailable {
			t.Errorf("%s: code = %v, want unavailable", name, cerr.Code())
		}
		if cerr.Message() != places.ErrUnavailable.Error() {
			t.Errorf("%s: message = %q, want %q", name, cerr.Message(), places.ErrUnavailable.Error())
		}
	}
}

// TestListSpotsIsLimitedPerAccount: every call spends a request to the provider, so one player's
// calls are capped, the cap is theirs alone, and the refused call never reaches the provider.
func TestListSpotsIsLimitedPerAccount(t *testing.T) {
	t.Parallel()
	street := usualStreet(t)
	srv := friesStack(t, street, fries.Limits{Spots: handlers.NewLimiter(2, time.Hour)}, nil)
	maya := friesPlayer(t, srv, "test_maya")
	ben := friesPlayer(t, srv, "test_ben")

	var errs []error
	for range 3 {
		_, err := maya.ListSpots(t.Context(), connect.NewRequest(atCampus(0)))
		errs = append(errs, err)
	}
	_, bensFirst := ben.ListSpots(t.Context(), connect.NewRequest(atCampus(0)))

	if errs[0] != nil || errs[1] != nil {
		t.Fatalf("the first two calls: %v, %v, want both answered", errs[0], errs[1])
	}
	assertTooManyRequests(t, "maya's third call", errs[2])
	if bensFirst != nil {
		t.Errorf("ben's first call: %v, want it answered (the limit is per account)", bensFirst)
	}
	n := len(street.calls())
	if n != 3 {
		t.Errorf("provider calls = %d, want 3 (the refused call spends none)", n)
	}
}

// TestNoCoordinateReachesALogLine: the player's location is used for one call and never logged.
// Every way a request can end (answered, refused as invalid, unauthenticated, the provider down,
// the provider failing in a way nobody expected) runs under the request logger and the service's
// logger, with coordinates no other number in the run could spell.
func TestNoCoordinateReachesALogLine(t *testing.T) {
	t.Parallel()
	const lat, lng = 12.3456789, -98.7654321
	var logs lockedBuffer
	log := slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	street := &fakeSpots{street: newPlacesMock(t, "")}
	srv := friesStack(t, street, fries.Limits{}, log)
	maya := friesPlayer(t, srv, "test_maya")
	stranger := friesNewcomer(t, srv)

	ask := func(f fryer, req *friesv1.ListSpotsRequest) {
		_, _ = f.ListSpots(t.Context(), connect.NewRequest(req)) // any outcome: only the log matters
	}
	ask(maya, at(lat, lng))
	ask(maya, at(lat+100, lng))
	ask(stranger, at(lat, lng))
	street.fail(fmt.Errorf("places: nearby: %w", places.ErrUnavailable))
	ask(maya, at(lat, lng))
	street.fail(fmt.Errorf("places: nearby: the provider sent a body that is not JSON"))
	ask(maya, at(lat, lng))
	// The server finishes its log lines before Close returns.
	srv.srv.Close()

	got := logs.String()
	if !strings.Contains(got, "/fries.v1.FriesService/ListSpots") || !strings.Contains(got, `"level":"ERROR"`) {
		t.Fatalf("the log is missing the request lines or the failure's error line:\n%s", got)
	}
	for _, digits := range []string{"12.345", "98.765", "112.345"} {
		if strings.Contains(got, digits) {
			t.Errorf("the log holds %q, part of the player's location:\n%s", digits, got)
		}
	}
}

// TestAFryRunIsStoredForTheCallerAlone: identity comes from the session. The run belongs to the
// player whose cookie sent it, under the trimmed name, a second tap in the same meal window is the
// same run, and nobody else's runs move.
func TestAFryRunIsStoredForTheCallerAlone(t *testing.T) {
	t.Parallel()
	srv := friesStack(t, usualStreet(t), fries.Limits{}, nil)
	maya := friesPlayer(t, srv, "test_maya")
	ben := friesPlayer(t, srv, "test_ben")

	first := logFryRun(t, maya, &friesv1.LogFryRunRequest{SpotId: "mock-03", SpotName: "  Night Owl Fries "})
	again := logFryRun(t, maya, nightOwlRun())

	if first <= 0 || again != first {
		t.Errorf("fry run ids %d then %d, want one run for both taps", first, again)
	}
	runs := fryRunsOf(t, srv.pool, maya.handle)
	if runs != 1 {
		t.Errorf("maya has %d fry runs, want 1", runs)
	}
	var handle, spotID, spotName string
	err := srv.pool.QueryRow(t.Context(), `
		select a.handle, f.spot_id, f.spot_name
		  from app.fry_run f
		  join app.account a on a.account_id = f.account_id
		 where f.fry_run_id = $1`, first).Scan(&handle, &spotID, &spotName)
	if err != nil {
		t.Fatalf("read fry run %d back: %v", first, err)
	}
	if handle != maya.handle || spotID != "mock-03" || spotName != "Night Owl Fries" {
		t.Errorf("stored %s at %s (%q), want test_maya at mock-03 (\"Night Owl Fries\")", handle, spotID, spotName)
	}
	n := fryRunsOf(t, srv.pool, ben.handle)
	if n != 0 {
		t.Errorf("ben has %d fry runs, want 0", n)
	}
}

// TestTheMealWindowComesFromCampusTime: the server's clock decides the window, in campus time.
// Half past midnight is still the night before's late-night run, and five in the morning starts
// a new day's run.
func TestTheMealWindowComesFromCampusTime(t *testing.T) {
	t.Parallel()
	clock := &testClock{at: time.Date(2026, 10, 20, 22, 30, 0, 0, points.Campus())}
	srv := friesStackWith(t, fries.Config{Spots: usualStreet(t), Now: clock.now})
	maya := friesPlayer(t, srv, "test_maya")

	lateNight := logFryRun(t, maya, nightOwlRun())
	clock.set(time.Date(2026, 10, 21, 0, 30, 0, 0, points.Campus()))
	afterMidnight := logFryRun(t, maya, nightOwlRun())
	clock.set(time.Date(2026, 10, 21, 5, 0, 0, 0, points.Campus()))
	breakfast := logFryRun(t, maya, nightOwlRun())

	if afterMidnight != lateNight {
		t.Errorf("00:30 got run %d, want 22:30's %d: both are Tuesday's late night", afterMidnight, lateNight)
	}
	if breakfast == lateNight {
		t.Errorf("05:00 got late night's run %d, want a new one", breakfast)
	}
	for id, want := range map[int64]string{lateNight: "2026-10-20 late_night", breakfast: "2026-10-21 morning"} {
		var got string
		err := srv.pool.QueryRow(t.Context(), `
			select f.meal_day::text || ' ' || f.meal_window
			  from app.fry_run f
			 where f.fry_run_id = $1`, id).Scan(&got)
		if err != nil {
			t.Fatalf("read fry run %d back: %v", id, err)
		}
		if got != want {
			t.Errorf("run %d fell in %s, want %s", id, got, want)
		}
	}
}

// TestLogFryRunIsLimitedPerAccount: a run counts once a window anyway, so the limit only stops a
// loop from hammering the table. It is the caller's alone.
func TestLogFryRunIsLimitedPerAccount(t *testing.T) {
	t.Parallel()
	srv := friesStackWith(t, fries.Config{
		Spots:  usualStreet(t),
		Limits: fries.Limits{FryRuns: handlers.NewLimiter(2, time.Hour)},
	})
	maya := friesPlayer(t, srv, "test_maya")
	ben := friesPlayer(t, srv, "test_ben")

	var errs []error
	for range 3 {
		_, err := maya.LogFryRun(t.Context(), connect.NewRequest(nightOwlRun()))
		errs = append(errs, err)
	}
	_, bensFirst := ben.LogFryRun(t.Context(), connect.NewRequest(nightOwlRun()))

	if errs[0] != nil || errs[1] != nil {
		t.Fatalf("the first two runs: %v, %v, want both logged", errs[0], errs[1])
	}
	assertTooManyRequests(t, "maya's third run", errs[2])
	if bensFirst != nil {
		t.Errorf("ben's first run: %v, want it logged (the limit is per account)", bensFirst)
	}
}

// TestABadFryRunIsRefusedAndNothingIsStored: the spot's bounds match app.fry_run's, so a bad
// value is a field message instead of a redacted internal error, and nothing is written.
func TestABadFryRunIsRefusedAndNothingIsStored(t *testing.T) {
	t.Parallel()
	srv := friesStack(t, usualStreet(t), fries.Limits{}, nil)
	maya := friesPlayer(t, srv, "test_maya")
	cases := []struct {
		name string
		req  *friesv1.LogFryRunRequest
		want map[string]string
	}{
		{
			"nothing sent",
			&friesv1.LogFryRunRequest{SpotName: "  "},
			map[string]string{"spot_id": "spot id is required", "spot_name": "spot name is required"},
		},
		{
			"an id of 257 characters",
			&friesv1.LogFryRunRequest{SpotId: strings.Repeat("i", 257), SpotName: "Night Owl Fries"},
			map[string]string{"spot_id": "spot id must be at most 256 characters"},
		},
		{
			"a name of 121 characters",
			&friesv1.LogFryRunRequest{SpotId: "mock-03", SpotName: strings.Repeat("ñ", 121)},
			map[string]string{"spot_name": "spot name must be at most 120 characters"},
		},
	}
	for _, c := range cases {
		_, err := maya.LogFryRun(t.Context(), connect.NewRequest(c.req))

		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Fatalf("%s: got code %v (%v), want invalid_argument", c.name, connect.CodeOf(err), err)
		}
		got := fieldViolations(t, err)
		if !maps.Equal(got, c.want) {
			t.Errorf("%s: violations = %v, want %v", c.name, got, c.want)
		}
	}
	n := fryRunCount(t, srv.pool)
	if n != 0 {
		t.Errorf("%d fry runs stored, want 0", n)
	}
}

// fakeSpots stands in for the places provider, this service's one true edge. It answers from a
// real places.Mock (tested in packages/go/places), from a fixed list, or with an error, and notes
// each call by method and query, never the point.
type fakeSpots struct {
	street *places.Mock   // when set, answers come from it
	found  []places.Place // otherwise this list

	mu    sync.Mutex
	err   error // when set, every call fails with it
	asked []string
}

func (f *fakeSpots) Nearby(ctx context.Context, lat, lng float64, query string) ([]places.Place, error) {
	err := f.note("nearby " + query)
	switch {
	case err != nil:
		return nil, err
	case f.street != nil:
		return f.street.Nearby(ctx, lat, lng, query)
	default:
		return slices.Clone(f.found), nil
	}
}

func (f *fakeSpots) Search(ctx context.Context, query string) ([]places.Place, error) {
	err := f.note("search " + query)
	switch {
	case err != nil:
		return nil, err
	case f.street != nil:
		return f.street.Search(ctx, query)
	default:
		return slices.Clone(f.found), nil
	}
}

// note records a call and returns the error it should fail with, if any.
func (f *fakeSpots) note(call string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.asked = append(f.asked, call)
	return f.err
}

// fail makes every later call fail with err.
func (f *fakeSpots) fail(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.err = err
}

func (f *fakeSpots) calls() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.asked)
}

// testClock is the service's clock, moved by hand, so a test can stand in any meal window.
type testClock struct {
	mu sync.Mutex
	at time.Time
}

func (c *testClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.at
}

func (c *testClock) set(at time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.at = at
}

func usualStreet(t *testing.T) *fakeSpots {
	t.Helper()
	return &fakeSpots{street: newPlacesMock(t, "")}
}

func closedStreet(t *testing.T) *fakeSpots {
	t.Helper()
	return &fakeSpots{street: newPlacesMock(t, "closed")}
}

func failingSpots(err error) *fakeSpots {
	return &fakeSpots{err: err}
}

// longStreet is n open spots, one every 10 m, for pages past the limits.
func longStreet(n int) *fakeSpots {
	found := make([]places.Place, n)
	for i := range found {
		found[i] = places.Place{
			ID:             fmt.Sprintf("long-%03d", i),
			Name:           fmt.Sprintf("Spot %d", i),
			DistanceMeters: 10 * (i + 1),
			OpenNow:        true,
		}
	}
	return &fakeSpots{found: found}
}

func newPlacesMock(t *testing.T, mode string) *places.Mock {
	t.Helper()
	mock, err := places.NewMock(mode)
	if err != nil {
		t.Fatalf("places mock %q: %v", mode, err)
	}
	return mock
}

// friesServer is AccountService and FriesService behind the middleware main.go puts in front of
// them, over the test's own database.
type friesServer struct {
	url  string
	pool *pgxpool.Pool
	srv  *httptest.Server
}

// friesStack serves both services with one session manager, so an account made through
// AccountService is the session FriesService reads. A nil log discards.
func friesStack(t *testing.T, spots fries.Spots, limits fries.Limits, log *slog.Logger) friesServer {
	t.Helper()
	return friesStackWith(t, fries.Config{Spots: spots, Limits: limits, Log: log})
}

// friesStackWith is friesStack for a test that sets more of the service's Config, such as its
// clock. The stack fills in Repo and Auth.
func friesStackWith(t *testing.T, cfg fries.Config) friesServer {
	t.Helper()
	log := cfg.Log
	if log == nil {
		log = discard()
	}
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
	mux.Handle(accountv1connect.NewAccountServiceHandler(account.NewAccount(account.Config{
		Repo: repo,
		Auth: authn,
	})))
	cfg.Repo, cfg.Auth, cfg.Log = repo, authn, log
	mux.Handle(friesv1connect.NewFriesServiceHandler(fries.NewFries(cfg)))
	srv := httptest.NewServer(logging.RequestLogger(log)(sessions.LoadAndSave(mux)))
	t.Cleanup(srv.Close)
	return friesServer{url: srv.URL, pool: pool, srv: srv}
}

// fryer is one phone talking to FriesService, with the cookie its account came with.
type fryer struct {
	friesv1connect.FriesServiceClient
	handle string
}

// friesPlayer is a player, made through the real CreateAccount, on FriesService.
func friesPlayer(t *testing.T, srv friesServer, handle string) fryer {
	t.Helper()
	p := player(t, srv.url, handle)
	return fryer{FriesServiceClient: friesv1connect.NewFriesServiceClient(&http.Client{Jar: p.jar}, srv.url), handle: handle}
}

// friesNewcomer is a fresh install: no account, no cookie.
func friesNewcomer(t *testing.T, srv friesServer) fryer {
	t.Helper()
	p := newcomer(t, srv.url)
	return fryer{FriesServiceClient: friesv1connect.NewFriesServiceClient(&http.Client{Jar: p.jar}, srv.url)}
}

func listSpots(t *testing.T, f fryer, req *friesv1.ListSpotsRequest) *friesv1.ListSpotsResponse {
	t.Helper()
	resp, err := f.ListSpots(t.Context(), connect.NewRequest(req))
	if err != nil {
		t.Fatalf("list spots: %v", err)
	}
	return resp.Msg
}

func logFryRun(t *testing.T, f fryer, req *friesv1.LogFryRunRequest) int64 {
	t.Helper()
	resp, err := f.LogFryRun(t.Context(), connect.NewRequest(req))
	if err != nil {
		t.Fatalf("log fry run: %v", err)
	}
	return resp.Msg.FryRunId
}

// at is a request from lat, lng with every other field at its default.
func at(lat, lng float64) *friesv1.ListSpotsRequest {
	return &friesv1.ListSpotsRequest{Lat: proto.Float64(lat), Lng: proto.Float64(lng)}
}

// atCampus is a request from campus for a page of limit spots.
func atCampus(limit int32) *friesv1.ListSpotsRequest {
	req := at(campusLat, campusLng)
	req.Limit = limit
	return req
}

func withQuery(req *friesv1.ListSpotsRequest, query string) *friesv1.ListSpotsRequest {
	req.Query = query
	return req
}

func withFilters(req *friesv1.ListSpotsRequest, filters ...friesv1.SpotFilter) *friesv1.ListSpotsRequest {
	req.Filters = filters
	return req
}

func nightOwlRun() *friesv1.LogFryRunRequest {
	return &friesv1.LogFryRunRequest{SpotId: "mock-03", SpotName: "Night Owl Fries"}
}

func spotIDs(spots []*friesv1.Spot) []string {
	ids := make([]string, 0, len(spots))
	for _, s := range spots {
		ids = append(ids, s.SpotId)
	}
	return ids
}

func fryRunCount(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var n int
	err := pool.QueryRow(t.Context(), `select count(*) from app.fry_run`).Scan(&n)
	if err != nil {
		t.Fatalf("count fry runs: %v", err)
	}
	return n
}

func fryRunsOf(t *testing.T, pool *pgxpool.Pool, handle string) int {
	t.Helper()
	var n int
	err := pool.QueryRow(t.Context(), `
		select count(*)
		  from app.fry_run f
		  join app.account a on a.account_id = f.account_id
		 where a.handle = $1`, handle).Scan(&n)
	if err != nil {
		t.Fatalf("count %s's fry runs: %v", handle, err)
	}
	return n
}
