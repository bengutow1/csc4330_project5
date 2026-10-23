// Author: Caden Lund
// Package profile serves profile.v1 ProfileService: the caller's rank, stats and badges, their
// plates a page at a time, and any player's public card. Every number comes from the ledger and
// the rules in packages/go/points, never from the app.
package profile

import (
	"context"
	"log/slog"
	"time"

	"connectrpc.com/connect"

	profilev1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/profile/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/auth"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers"
	"github.com/bengutow1/csc4330_project5/packages/go/points"
	"github.com/bengutow1/csc4330_project5/packages/go/repository"
)

// Profile implements profile.v1 ProfileService.
type Profile struct {
	repo *repository.Repository
	gate handlers.Gate
	now  func() time.Time
	log  *slog.Logger
}

// Config is what Profile needs to exist.
type Config struct {
	Repo *repository.Repository
	Auth *auth.Auth
	// Now is the clock that says which day a streak ends on and which weeks are over. Nil means
	// time.Now.
	Now func() time.Time
	Log *slog.Logger
}

// NewProfile builds the service. A nil Log discards.
func NewProfile(cfg Config) *Profile {
	log := cfg.Log
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	return &Profile{repo: cfg.Repo, gate: handlers.Gate{Auth: cfg.Auth}, now: now, log: log}
}

// fail is the one way an error leaves this service.
func (h *Profile) fail(op string, err error) error {
	return handlers.ConnectErr(h.log, op, err)
}

// GetProfile returns the caller's rank, stats and every badge, earned or locked.
func (h *Profile) GetProfile(
	ctx context.Context, _ *connect.Request[profilev1.GetProfileRequest],
) (*connect.Response[profilev1.GetProfileResponse], error) {
	const op = "get profile"

	// 1. Who is asking: the account comes from the session
	me, err := h.gate.Account(ctx)
	if err != nil {
		return nil, h.fail(op, err)
	}

	// 2. Where they stand, from their own history
	p, err := h.standingOf(ctx, me.AccountID)
	if err != nil {
		return nil, h.fail(op, err)
	}
	return connect.NewResponse(&profilev1.GetProfileResponse{Profile: toProtoProfile(me, p)}), nil
}

// ListPlates returns a page of the caller's plates, newest first.
func (h *Profile) ListPlates(
	ctx context.Context, req *connect.Request[profilev1.ListPlatesRequest],
) (*connect.Response[profilev1.ListPlatesResponse], error) {
	const op = "list plates"

	// 1. Who is asking: the account comes from the session, and only their plates come back
	me, err := h.gate.Account(ctx)
	if err != nil {
		return nil, h.fail(op, err)
	}

	// 2. Normalize, then validate everything the request carries
	in, invalid := buildListPlates(req.Msg)
	if invalid != nil {
		return nil, h.fail(op, invalid)
	}

	// 3. The page, and one plate past it to know whether there is more
	plates, err := h.repo.ListPlates(ctx, me.AccountID, in.after, in.limit+1)
	if err != nil {
		return nil, h.fail(op, err)
	}
	return connect.NewResponse(toProtoPlates(in, plates)), nil
}

// recentPlates is how many plates a player card carries: two rows of the card's grid, few enough
// that the photos keep the card light, so it needs no cursor.
const recentPlates = 6

// GetPlayerCard returns one player's public card, found by their handle: their rank, stats, newest
// plates and every badge. It never reads their kitchen settings, so it cannot send them.
func (h *Profile) GetPlayerCard(
	ctx context.Context, req *connect.Request[profilev1.GetPlayerCardRequest],
) (*connect.Response[profilev1.GetPlayerCardResponse], error) {
	const op = "get player card"

	// 1. Who is asking: any signed-in player may open any card, their own included
	_, err := h.gate.Account(ctx)
	if err != nil {
		return nil, h.fail(op, err)
	}

	// 2. Normalize, then validate the handle by the same rule that made it
	handle := handlers.NormalizeHandle(req.Msg.Handle)
	invalid := handlers.Validate(handlers.HandleInput{Handle: handle})
	if invalid != nil {
		return nil, h.fail(op, invalid)
	}

	// 3. The player it names: a handle nobody holds is not_found
	player, err := h.repo.PlayerByHandle(ctx, handle)
	if err != nil {
		return nil, h.fail(op, err)
	}

	// 4. Where they stand and their newest plates, each count fenced by their account
	p, err := h.standingOf(ctx, player.AccountID)
	if err != nil {
		return nil, h.fail(op, err)
	}
	plates, err := h.repo.ListPlates(ctx, player.AccountID, nil, recentPlates)
	if err != nil {
		return nil, h.fail(op, err)
	}
	return connect.NewResponse(&profilev1.GetPlayerCardResponse{Card: toProtoPlayerCard(player, p, plates)}), nil
}

// standingOf reads where accountID stands: their points, the stats the badges read, their streak
// ending today by the meal window's day, and their fry runs. Every count is fenced by accountID.
func (h *Profile) standingOf(ctx context.Context, accountID int64) (standing, error) {
	now := h.now()
	total, err := h.repo.PointsTotal(ctx, accountID)
	if err != nil {
		return standing{}, err
	}
	played, err := h.repo.PlayerStats(ctx, accountID)
	if err != nil {
		return standing{}, err
	}
	cooked, err := h.repo.CookDays(ctx, accountID)
	if err != nil {
		return standing{}, err
	}
	fryRuns, err := h.repo.FryRunCount(ctx, accountID)
	if err != nil {
		return standing{}, err
	}
	topped, err := h.repo.HallOfFriesWeeks(ctx, accountID, points.WeekStart(now), points.Campus())
	if err != nil {
		return standing{}, err
	}
	days, err := handlers.CampusDays(cooked)
	if err != nil {
		return standing{}, err
	}
	_, today := points.MealWindow(now)
	return standing{
		total:   total,
		stats:   handlers.StatsOf(played, topped),
		streak:  points.Streak(days, today),
		fryRuns: fryRuns,
	}, nil
}
