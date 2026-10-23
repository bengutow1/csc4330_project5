// Author: Caden Lund
// Package cook serves cook.v1 CookService: the recipes for one of a player's scans, on two
// shelves, one recipe at a time, and the plate that turns a cooked recipe into points.
package cook

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"sync"
	"time"

	"connectrpc.com/connect"

	cookv1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/cook/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/apigen/cook/v1/cookv1connect"
	"github.com/bengutow1/csc4330_project5/packages/go/auth"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers"
	"github.com/bengutow1/csc4330_project5/packages/go/points"
	"github.com/bengutow1/csc4330_project5/packages/go/recipes"
	"github.com/bengutow1/csc4330_project5/packages/go/repository"
	"github.com/bengutow1/csc4330_project5/packages/go/vision"
)

// Each set of recipes spends model time, so each player gets setsPerHour of them in an hour. A
// scan and each edit of its list need one; a changed filter rereads the stored set and spends
// none.
const (
	setsPerHour = 30
	setWindow   = time.Hour
)

// Each plate check spends model time too, so each player gets platesPerHour of them. Four meal
// windows a day leave room for a retake or two in each.
const (
	platesPerHour = 10
	plateWindow   = time.Hour
)

// maxReasonRunes bounds the model's reason for a failed plate in the log line, since its words
// are the vendor's.
const maxReasonRunes = 200

// Vision writes recipes for a list and the player's kitchen, and checks a photo of a cooked one.
// The vision mock serves it until the real client (#25) exists.
type Vision interface {
	Recipes(ctx context.Context, items []vision.Item, kitchen vision.Kitchen) ([]vision.Recipe, error)
	CheckPlate(ctx context.Context, photo []byte, recipeTitle string) (vision.PlateCheck, error)
}

// Cook implements cook.v1 CookService.
type Cook struct {
	repo        *repository.Repository
	gate        handlers.Gate
	vision      Vision
	limit       *handlers.Limiter
	plateLimit  *handlers.Limiter
	making      *making
	setDeadline time.Duration
	now         func() time.Time
	log         *slog.Logger
}

// Every rpc is served here, so a new one in cook.proto fails the build until it is.
var _ cookv1connect.CookServiceHandler = (*Cook)(nil)

// making holds the sets being made right now, so each is made by one request at a time. Another
// request for the same set (a chip tapped while the first shelves load) waits for that one
// instead of asking the model again and spending another of the player's sets. It covers one
// process; recipe_set's unique key still settles a race between two.
type making struct {
	mu   sync.Mutex
	sets map[setKey]chan struct{}
}

// setDeadline bounds the time one request spends getting its set: a wait on another request's
// claim, then one model call and two writes, well inside the server's 90 s WriteTimeout.
const setDeadline = 60 * time.Second

// setKey names one version of one scan's list, which gets one set.
type setKey struct {
	scanID         int64
	version        int
	partnerScanID  int64
	partnerVersion int
}

// claim makes this request the one making key's set, or returns what closes when the request
// making it is done.
func (m *making) claim(key setKey) (done <-chan struct{}, mine bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if ch, ok := m.sets[key]; ok {
		return ch, false
	}
	m.sets[key] = make(chan struct{})
	return nil, true
}

// release ends this request's claim on key and wakes every request waiting on it.
func (m *making) release(key setKey) {
	m.mu.Lock()
	defer m.mu.Unlock()
	close(m.sets[key])
	delete(m.sets, key)
}

// Config is what Cook needs to exist.
type Config struct {
	Repo   *repository.Repository
	Auth   *auth.Auth
	Vision Vision
	// Limit caps how many sets of recipes each player's lists get. main.go passes DefaultLimit;
	// nil limits nothing.
	Limit *handlers.Limiter
	// PlateLimit caps how many plates each player has checked. main.go passes
	// DefaultPlateLimit; nil limits nothing.
	PlateLimit *handlers.Limiter
	// SetDeadline bounds how long one request spends getting its set. main.go leaves it zero,
	// which is setDeadline; a test shortens it to watch one pass.
	SetDeadline time.Duration
	// Now is the clock the meal windows are read from; nil is time.Now.
	Now func() time.Time
	Log *slog.Logger
}

// DefaultLimit is the limit the API serves with: setsPerHour for each player.
func DefaultLimit() *handlers.Limiter {
	return handlers.NewLimiter(setsPerHour, setWindow)
}

// DefaultPlateLimit is the plate limit the API serves with: platesPerHour for each player.
func DefaultPlateLimit() *handlers.Limiter {
	return handlers.NewLimiter(platesPerHour, plateWindow)
}

// NewCook builds the service. A nil Log discards.
func NewCook(cfg Config) *Cook {
	log := cfg.Log
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	deadline := cfg.SetDeadline
	if deadline == 0 {
		deadline = setDeadline
	}
	return &Cook{
		repo:        cfg.Repo,
		gate:        handlers.Gate{Auth: cfg.Auth, Repo: cfg.Repo},
		vision:      cfg.Vision,
		limit:       cfg.Limit,
		plateLimit:  cfg.PlateLimit,
		making:      &making{sets: map[setKey]chan struct{}{}},
		setDeadline: deadline,
		now:         now,
		log:         log,
	}
}

// fail is the one way an error leaves this service.
func (h *Cook) fail(op string, err error) error {
	return handlers.ConnectErr(h.log, op, err)
}

// ListRecipes suggests recipes for one of the caller's scans, on two shelves.
func (h *Cook) ListRecipes(
	ctx context.Context, req *connect.Request[cookv1.ListRecipesRequest],
) (*connect.Response[cookv1.ListRecipesResponse], error) {
	const op = "list recipes"

	// 1. Who is asking: the account comes from the session
	me, err := h.gate.Account(ctx)
	if err != nil {
		return nil, h.fail(op, err)
	}

	// 2. Validate the scan and every filter, so the chips mark each problem
	in, invalid := buildListRecipes(req.Msg)
	if invalid != nil {
		return nil, h.fail(op, invalid)
	}
	if req.Msg.ComboId != nil {
		return h.listComboRecipes(ctx, me, req.Msg.GetComboId(), in.filter)
	}

	// 3. Load the scan: another player's answers as a missing one does
	scan, err := h.repo.ScanByID(ctx, me.AccountID, in.scanID)
	if err != nil {
		return nil, h.fail(op, err)
	}

	// 4. Each version of the list gets its set once. A list with nothing on it has nothing to
	// cook, so it never reaches the model
	if len(scan.Items) > 0 {
		err = h.ensureSet(ctx, me, scan)
		if err != nil {
			return nil, h.fail(op, err)
		}
	}

	// 5. The shelves, narrowed in SQL, with the player's allergies read fresh
	set, err := h.repo.ListRecipes(ctx, me.AccountID, scan.ScanID, scan.ItemsVersion, in.filter)
	if err != nil {
		return nil, h.fail(op, err)
	}
	resp, err := toProtoShelves(set)
	if err != nil {
		return nil, h.fail(op, err)
	}
	return connect.NewResponse(resp), nil
}

// ensureSet sees that the scan's list has its set, making it if no request has. A request that
// finds another making it waits for that one and spends nothing, and makes the set itself if that
// one failed. One deadline covers all of it, so a request that waited never starts a fresh one.
func (h *Cook) ensureSet(ctx context.Context, me repository.Account, scan repository.Scan) error {
	key := setKey{scanID: scan.ScanID, version: scan.ItemsVersion}
	return h.ensureRecipes(ctx, key, func(ctx context.Context) (bool, error) {
		return h.repo.HasRecipes(ctx, me.AccountID, scan.ScanID, scan.ItemsVersion)
	}, func(ctx context.Context) error { return h.makeSet(ctx, me, scan) })
}

// ensureRecipes shares one deadline and one in-flight claim across solo and combo shelves.
func (h *Cook) ensureRecipes(ctx context.Context, key setKey, has func(context.Context) (bool, error), makeSet func(context.Context) error) error {
	ctx, cancel := context.WithTimeout(ctx, h.setDeadline)
	defer cancel()
	for {
		found, err := has(ctx)
		if err != nil || found {
			return err
		}
		done, mine := h.making.claim(key)
		if mine {
			return h.makeClaimed(ctx, key, has, makeSet)
		}
		select {
		case <-done:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// makeClaimed makes the set this request claimed, unless another request stored it between the
// look and the claim. It finishes even if this request's caller hangs up, which a chip tapped
// while the shelves load makes the app do, because the requests waiting on the claim still need
// the set. It keeps the request's deadline, which ensureRecipes set.
func (h *Cook) makeClaimed(ctx context.Context, key setKey, has func(context.Context) (bool, error), makeSet func(context.Context) error) error {
	defer h.making.release(key)
	deadline, _ := ctx.Deadline()
	ctx, cancel := context.WithDeadline(context.WithoutCancel(ctx), deadline)
	defer cancel()
	found, err := has(ctx)
	if err != nil || found {
		return err
	}
	return makeSet(ctx)
}

// makeSet asks the model for recipes for the scan's list and stores the ones the player can cook.
// It spends one of the player's sets, and the call takes seconds, so no transaction is open while
// it runs.
func (h *Cook) makeSet(ctx context.Context, me repository.Account, scan repository.Scan) error {
	set, err := h.checkedRecipes(ctx, me, scan)
	if err != nil {
		return err
	}
	_, err = h.repo.CreateRecipes(ctx, me.AccountID, scan.ScanID, scan.ItemsVersion, set)
	return err
}

// checkedRecipes keeps the same validation and food checks for either kind of fridge list.
func (h *Cook) checkedRecipes(ctx context.Context, me repository.Account, scan repository.Scan) ([]repository.NewRecipe, error) {
	// 1. The model's time costs money, so each player gets setsPerHour
	err := h.limit.Allow(strconv.FormatInt(me.AccountID, 10))
	if err != nil {
		return nil, err
	}

	// 2. The model hears the list and the kitchen, and nothing else about the player
	written, err := h.vision.Recipes(ctx, visionItems(scan.Items), vision.Kitchen{
		Equipment: me.Equipment,
		Diet:      me.Diet,
		Allergies: me.Allergies,
	})
	if err != nil {
		return nil, err
	}

	// 3. What the model wrote is as untrusted as a request: keep what passes, count the rest
	shaped, dropped := keepModelRecipes(written)
	if dropped > 0 {
		h.log.Warn("dropped recipes the model wrote", "account_id", me.AccountID,
			"scan_id", scan.ScanID, "dropped", dropped)
	}

	// 4. Our code judges what the player can cook, and what the dish contains and fits
	names := make([]string, 0, len(scan.Items))
	for _, item := range scan.Items {
		names = append(names, item.Name)
	}
	set := make([]repository.NewRecipe, 0, len(shaped))
	for _, r := range shaped {
		dish := recipes.Recipe{
			Title: r.Title, Uses: r.Uses, Missing: r.Missing, Allergens: r.Allergens, Steps: r.Steps,
		}
		kept, why := recipes.Check(dish, names, me.Staples, me.Allergies)
		if !kept {
			h.log.Info("dropped a recipe the player cannot cook", "account_id", me.AccountID,
				"scan_id", scan.ScanID, "why", why)
			continue
		}
		r.Allergens = recipes.Allergens(dish)
		r.Diets = recipes.Diets(dish)
		set = append(set, r)
	}

	return set, nil
}

// GetRecipe reads one of the caller's recipes.
func (h *Cook) GetRecipe(
	ctx context.Context, req *connect.Request[cookv1.GetRecipeRequest],
) (*connect.Response[cookv1.GetRecipeResponse], error) {
	const op = "get recipe"

	// 1. Who is asking: the account comes from the session
	me, err := h.gate.Account(ctx)
	if err != nil {
		return nil, h.fail(op, err)
	}

	// 2. Validate the id
	invalid := handlers.Validate(struct {
		RecipeID int64 `pfield:"recipe_id" validate:"gt=0"`
	}{req.Msg.GetRecipeId()})
	if invalid != nil {
		return nil, h.fail(op, invalid)
	}

	// 3. Load it through its scan: another player's answers as a missing one does
	stored, err := h.repo.RecipeByID(ctx, me.AccountID, req.Msg.GetRecipeId())
	if err != nil {
		return nil, h.fail(op, err)
	}
	recipe, err := toProtoRecipe(stored)
	if err != nil {
		return nil, h.fail(op, err)
	}
	return connect.NewResponse(&cookv1.GetRecipeResponse{Recipe: recipe}), nil
}

// PlateMeal checks a photo of the finished dish and, when it passes, awards its points.
func (h *Cook) PlateMeal(
	ctx context.Context, req *connect.Request[cookv1.PlateMealRequest],
) (*connect.Response[cookv1.PlateMealResponse], error) {
	const op = "plate meal"

	// 1. Who is asking: the account comes from the session
	me, err := h.gate.Account(ctx)
	if err != nil {
		return nil, h.fail(op, err)
	}

	// 2. Validate the recipe id and the photo, before anything spends a check
	invalid := buildPlate(req.Msg)
	if invalid != nil {
		return nil, h.fail(op, invalid)
	}

	// 3. Each check spends model time, so each player gets platesPerHour
	err = h.plateLimit.Allow(strconv.FormatInt(me.AccountID, 10))
	if err != nil {
		return nil, h.fail(op, err)
	}

	// 4. Load the recipe through its scan: another player's answers as a missing one does
	recipe, err := h.repo.RecipeByID(ctx, me.AccountID, req.Msg.GetRecipeId())
	if err != nil {
		return nil, h.fail(op, err)
	}

	// 5. The model looks at the photo. It takes seconds, so no transaction is open while it does
	if recipe.ComboID > 0 {
		_, c, err := h.gate.ComboMember(ctx, recipe.ComboID)
		if err != nil {
			return nil, h.fail(op, err)
		}
		err = comboReady(c, h.now())
		if err != nil {
			return nil, h.fail(op, err)
		}
	}
	check, err := h.vision.CheckPlate(ctx, req.Msg.GetPhoto(), recipe.Title)
	if err != nil {
		return nil, h.fail(op, err)
	}
	now := h.now()

	// 6. A plate that fails earns nothing and stores nothing. The model's reason is for the log,
	// and the player reads our sentence
	if !check.Passed {
		h.log.Info("plate check failed", "account_id", me.AccountID, "recipe_id", recipe.RecipeID,
			"reason", clip(check.Reason, maxReasonRunes))
		receipt, err := h.failedReceipt(ctx, me.AccountID, recipe.Title, now)
		if err != nil {
			return nil, h.fail(op, err)
		}
		return connect.NewResponse(&cookv1.PlateMealResponse{Receipt: receipt}), nil
	}

	// 7. The plate and its points, as one act
	receipt, err := h.plate(ctx, me.AccountID, recipe, req.Msg.GetPhoto(), now)
	if err != nil {
		return nil, h.fail(op, err)
	}
	return connect.NewResponse(&cookv1.PlateMealResponse{Receipt: receipt}), nil
}

// plate stores a passed plate and its awards in one transaction and returns the receipt.
func (h *Cook) plate(
	ctx context.Context, accountID int64, recipe repository.Recipe, photo []byte, now time.Time,
) (*cookv1.Receipt, error) {
	if recipe.ComboID > 0 {
		return h.plateCombo(ctx, accountID, recipe, photo, now)
	}
	window, day := points.MealWindow(now)
	today := day.Format(time.DateOnly)
	hash := sha256.Sum256(photo)

	var paid paidPlate
	err := h.repo.InTx(ctx, func(tx *repository.Repository) error {
		// 1. This player's plates take turns, so the day's first is read before it is written
		err := tx.LockAccount(ctx, accountID)
		if err != nil {
			return err
		}

		// 2. Where they stood before this plate
		paid.before, err = tx.PlayerStats(ctx, accountID)
		if err != nil {
			return err
		}
		paid.totalBefore, err = tx.PointsTotal(ctx, accountID)
		if err != nil {
			return err
		}
		cooked, err := tx.CookDays(ctx, accountID)
		if err != nil {
			return err
		}

		// 3. The plate. A photo seen before is refused here
		plateID, err := tx.CreatePlate(ctx, accountID, repository.NewPlate{
			RecipeID: recipe.RecipeID, Photo: photo, PhotoSHA256: hash[:], At: now,
		})
		if err != nil {
			return err
		}

		// 4. The awards. The streak counts today, and only the day's first plate pays it
		days, err := handlers.CampusDays(cooked)
		if err != nil {
			return err
		}
		paid.streak = points.Streak(append(days, day), day)
		paid.lines = points.PlateAward(points.Plate{
			Score:      recipe.ScanScore,
			FirstOfDay: !slices.Contains(cooked, today),
			Streak:     paid.streak,
		})

		// 5. The ledger. The meal window's one cook credit is claimed here, and a second refused
		err = tx.AwardPoints(ctx, accountID, plateID, today+" "+string(window), now, toAwards(paid.lines))
		if err != nil {
			return err
		}

		// 6. Re-read where they stand, so the receipt shows what was stored
		paid.after, err = tx.PlayerStats(ctx, accountID)
		if err != nil {
			return err
		}
		paid.totalAfter, err = tx.PointsTotal(ctx, accountID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return toProtoReceipt(fmt.Sprintf("Looks like %s", recipe.Title), paid)
}

// failedReceipt is the receipt for a plate the model did not pass: nothing earned, with the
// player's streak and rank as they stand.
func (h *Cook) failedReceipt(
	ctx context.Context, accountID int64, title string, now time.Time,
) (*cookv1.Receipt, error) {
	total, err := h.repo.PointsTotal(ctx, accountID)
	if err != nil {
		return nil, err
	}
	cooked, err := h.repo.CookDays(ctx, accountID)
	if err != nil {
		return nil, err
	}
	days, err := handlers.CampusDays(cooked)
	if err != nil {
		return nil, err
	}
	_, today := points.MealWindow(now)
	rank := handlers.ToProtoRank(points.RankOf(total))
	return &cookv1.Receipt{
		Passed:     false,
		Message:    fmt.Sprintf("That doesn't look like %s. Try another angle.", title),
		StreakDays: int32(points.Streak(days, today)), // a count of days, far under int32
		RankBefore: rank,
		RankAfter:  rank,
	}, nil
}

// clip shortens s to at most n runes.
func clip(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n])
}
