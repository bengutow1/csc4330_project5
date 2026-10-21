// Author: Caden Lund
// Package fries serves fries.v1 FriesService: spots near the player, narrowed the way they asked,
// and the fry runs they log. The player's location goes to the places provider for one call and
// nowhere else: it is never stored and never logged.
package fries

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"

	friesv1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/fries/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/auth"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers"
	"github.com/bengutow1/csc4330_project5/packages/go/places"
	"github.com/bengutow1/csc4330_project5/packages/go/repository"
)

// The ListSpots limit per account. Every call spends a request to the places provider, and a
// player flipping every chip and scrolling to the end stays well under it.
const (
	spotsPerAccount = 30
	spotsWindow     = time.Minute
)

// Spots is the places provider as this service uses it: places.Mock until a key exists, the real
// client after #32. A test hands in a fake.
type Spots interface {
	// Nearby finds spots around a point, each with its distance from it.
	Nearby(ctx context.Context, lat, lng float64, query string) ([]places.Place, error)
	// Search finds spots from words alone, for a player whose location is off.
	Search(ctx context.Context, query string) ([]places.Place, error)
}

// Fries implements fries.v1 FriesService.
type Fries struct {
	repo   *repository.Repository
	gate   handlers.Gate
	spots  Spots
	limits Limits
	log    *slog.Logger
}

// Config is what Fries needs to exist.
type Config struct {
	Repo  *repository.Repository
	Auth  *auth.Auth
	Spots Spots
	// Limits caps ListSpots per account. main.go passes DefaultLimits; the zero value limits
	// nothing.
	Limits Limits
	Log    *slog.Logger
}

// Limits holds the ListSpots limiter, keyed by account. A nil one limits nothing, which is how
// test fixtures turn limits off.
type Limits struct {
	Spots *handlers.Limiter
}

// DefaultLimits are the limits the API serves with.
func DefaultLimits() Limits {
	return Limits{Spots: handlers.NewLimiter(spotsPerAccount, spotsWindow)}
}

// NewFries builds the service. A nil Log discards.
func NewFries(cfg Config) *Fries {
	log := cfg.Log
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	return &Fries{
		repo:   cfg.Repo,
		gate:   handlers.Gate{Auth: cfg.Auth},
		spots:  cfg.Spots,
		limits: cfg.Limits,
		log:    log,
	}
}

// fail is the one way an error leaves this service.
func (h *Fries) fail(op string, err error) error {
	return handlers.ConnectErr(h.log, op, err)
}

// ListSpots finds spots near the player, or where the query says when their location is off,
// narrowed by the filters they sent, closest first, a page at a time.
func (h *Fries) ListSpots(
	ctx context.Context, req *connect.Request[friesv1.ListSpotsRequest],
) (*connect.Response[friesv1.ListSpotsResponse], error) {
	const op = "list spots"

	// 1. Who is asking: the account comes from the session
	me, err := h.gate.Account(ctx)
	if err != nil {
		return nil, h.fail(op, err)
	}

	// 2. Normalize, then validate everything the request carries
	in, invalid := buildListSpots(req.Msg)
	if invalid != nil {
		return nil, h.fail(op, invalid)
	}

	// 3. Each call spends a request to the provider, so each player has a limit
	err = h.limits.Spots.Allow(strconv.FormatInt(me.AccountID, 10))
	if err != nil {
		return nil, h.fail(op, err)
	}

	// 4. The provider, outside any transaction. The point rides this one call: no error, log line
	// or row ever carries it.
	found, err := h.find(ctx, in)
	if err != nil {
		return nil, h.fail(op, fmt.Errorf("find spots account=%d: %w", me.AccountID, err))
	}

	// 5. The provider ranks and filters its own way, so the order, the filters and the page are
	// applied here
	return connect.NewResponse(toProtoListSpots(pageOf(found, in))), nil
}

// find asks the provider around the player's point, or from the query alone when their location
// is off.
func (h *Fries) find(ctx context.Context, in listSpotsInput) ([]places.Place, error) {
	if in.at == nil {
		return h.spots.Search(ctx, in.query)
	}
	return h.spots.Nearby(ctx, in.at.lat, in.at.lng, in.query)
}

// LogFryRun records that the caller went to a spot. It earns no points: the Hall of Fries counts
// it.
func (h *Fries) LogFryRun(
	ctx context.Context, req *connect.Request[friesv1.LogFryRunRequest],
) (*connect.Response[friesv1.LogFryRunResponse], error) {
	const op = "log fry run"

	// 1. Who is asking: the run is theirs, whatever the request says
	me, err := h.gate.Account(ctx)
	if err != nil {
		return nil, h.fail(op, err)
	}

	// 2. Normalize, then validate both fields
	run := fryRunInput{
		SpotID:   strings.TrimSpace(req.Msg.SpotId),
		SpotName: strings.TrimSpace(req.Msg.SpotName),
	}
	invalid := handlers.Validate(run)
	if invalid != nil {
		return nil, h.fail(op, invalid)
	}

	// 3. The write
	id, err := h.repo.CreateFryRun(ctx, me.AccountID, run.SpotID, run.SpotName)
	if err != nil {
		return nil, h.fail(op, err)
	}
	return connect.NewResponse(&friesv1.LogFryRunResponse{FryRunId: id}), nil
}
