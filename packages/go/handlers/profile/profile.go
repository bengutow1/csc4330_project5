// Author: Caden Lund
// Package profile serves profile.v1 ProfileService: the caller's rank, stats and badges, and their
// plates a page at a time. Every number comes from the ledger and the rules in packages/go/points,
// never from the app.
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

	// 2. Their history, each count fenced by their account
	now := h.now()
	total, err := h.repo.PointsTotal(ctx, me.AccountID)
	if err != nil {
		return nil, h.fail(op, err)
	}
	played, err := h.repo.PlayerStats(ctx, me.AccountID)
	if err != nil {
		return nil, h.fail(op, err)
	}
	cooked, err := h.repo.CookDays(ctx, me.AccountID)
	if err != nil {
		return nil, h.fail(op, err)
	}
	fryRuns, err := h.repo.FryRunCount(ctx, me.AccountID)
	if err != nil {
		return nil, h.fail(op, err)
	}
	topped, err := h.repo.HallOfFriesWeeks(ctx, me.AccountID, points.WeekStart(now), points.Campus())
	if err != nil {
		return nil, h.fail(op, err)
	}

	// 3. What the points rules make of it. The streak ends today, by the meal window's day
	days, err := handlers.CampusDays(cooked)
	if err != nil {
		return nil, h.fail(op, err)
	}
	_, today := points.MealWindow(now)
	p := standing{
		total:   total,
		stats:   handlers.StatsOf(played, topped),
		streak:  points.Streak(days, today),
		fryRuns: fryRuns,
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
