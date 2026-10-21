// Author: Caden Lund
// Package cook serves cook.v1 CookService: the recipes for one of a player's scans, on two
// shelves, and one recipe at a time. Plating (#23) answers unimplemented until it lands.
package cook

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"connectrpc.com/connect"

	cookv1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/cook/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/apigen/cook/v1/cookv1connect"
	"github.com/bengutow1/csc4330_project5/packages/go/auth"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers"
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

// Vision writes recipes for a list and the player's kitchen. The vision mock serves it until the
// real client (#25) exists.
type Vision interface {
	Recipes(ctx context.Context, items []vision.Item, kitchen vision.Kitchen) ([]vision.Recipe, error)
}

// Cook implements cook.v1 CookService.
type Cook struct {
	// PlateMeal is #23's. Until it lands, the generated stub answers unimplemented.
	cookv1connect.UnimplementedCookServiceHandler

	repo   *repository.Repository
	gate   handlers.Gate
	vision Vision
	limit  *handlers.Limiter
	log    *slog.Logger
}

// Config is what Cook needs to exist.
type Config struct {
	Repo   *repository.Repository
	Auth   *auth.Auth
	Vision Vision
	// Limit caps how many sets of recipes each player's lists get. main.go passes DefaultLimit;
	// nil limits nothing.
	Limit *handlers.Limiter
	Log   *slog.Logger
}

// DefaultLimit is the limit the API serves with: setsPerHour for each player.
func DefaultLimit() *handlers.Limiter {
	return handlers.NewLimiter(setsPerHour, setWindow)
}

// NewCook builds the service. A nil Log discards.
func NewCook(cfg Config) *Cook {
	log := cfg.Log
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	return &Cook{
		repo:   cfg.Repo,
		gate:   handlers.Gate{Auth: cfg.Auth},
		vision: cfg.Vision,
		limit:  cfg.Limit,
		log:    log,
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

	// 3. Load the scan: another player's answers as a missing one does
	scan, err := h.repo.ScanByID(ctx, me.AccountID, in.scanID)
	if err != nil {
		return nil, h.fail(op, err)
	}

	// 4. Each version of the list gets its set once. A list with nothing on it has nothing to
	// cook, so it never reaches the model
	if len(scan.Items) > 0 {
		var has bool
		has, err = h.repo.HasRecipes(ctx, me.AccountID, scan.ScanID, scan.ItemsVersion)
		if err != nil {
			return nil, h.fail(op, err)
		}
		if !has {
			err = h.makeSet(ctx, me, scan)
			if err != nil {
				return nil, h.fail(op, err)
			}
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

// makeSet asks the model for recipes for the scan's list and stores the ones the player can cook.
// It spends one of the player's sets, and the call takes seconds, so no transaction is open while
// it runs.
func (h *Cook) makeSet(ctx context.Context, me repository.Account, scan repository.Scan) error {
	// 1. The model's time costs money, so each player gets setsPerHour
	err := h.limit.Allow(strconv.FormatInt(me.AccountID, 10))
	if err != nil {
		return err
	}

	// 2. The model hears the list and the kitchen, and nothing else about the player
	written, err := h.vision.Recipes(ctx, visionItems(scan.Items), vision.Kitchen{
		Equipment: me.Equipment,
		Diet:      me.Diet,
		Allergies: me.Allergies,
	})
	if err != nil {
		return err
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

	// 5. Store the set. A request that raced this one may have stored its own first, which then
	// stands, so either way the shelves read one set
	_, err = h.repo.CreateRecipes(ctx, me.AccountID, scan.ScanID, scan.ItemsVersion, set)
	return err
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
