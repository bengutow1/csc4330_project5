// Author: Caden Lund
// Package scan serves scan.v1 ScanService: judging a player's kitchen from its photos, and
// reading and fixing the verdicts it keeps.
package scan

import (
	"context"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"

	scanv1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/scan/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/auth"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers"
	"github.com/bengutow1/csc4330_project5/packages/go/repository"
	"github.com/bengutow1/csc4330_project5/packages/go/scoring"
	"github.com/bengutow1/csc4330_project5/packages/go/vision"
)

// Every scan spends model time, so each player gets scansPerHour of them in an hour.
const (
	scansPerHour = 12
	scanWindow   = time.Hour
)

// Vision reads a kitchen's photos into the foods it can see and a roast. The vision mock serves
// it until the real client (#20) exists.
type Vision interface {
	ReadFridge(ctx context.Context, photos [][]byte, level vision.RoastLevel) (vision.Fridge, error)
}

// Scan implements scan.v1 ScanService.
type Scan struct {
	repo   *repository.Repository
	gate   handlers.Gate
	vision Vision
	limit  *handlers.Limiter
	log    *slog.Logger
	now    func() time.Time
}

// Config is what Scan needs to exist.
type Config struct {
	Repo   *repository.Repository
	Auth   *auth.Auth
	Vision Vision
	// Limit caps scans per account. main.go passes DefaultLimit; nil limits nothing.
	Limit *handlers.Limiter
	Log   *slog.Logger
	Now   func() time.Time // nil uses the wall clock; combo expiry tests fix it.
}

// DefaultLimit is the limit the API serves with: scansPerHour for each player.
func DefaultLimit() *handlers.Limiter {
	return handlers.NewLimiter(scansPerHour, scanWindow)
}

// NewScan builds the service. A nil Log discards.
func NewScan(cfg Config) *Scan {
	log := cfg.Log
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &Scan{
		repo:   cfg.Repo,
		gate:   handlers.Gate{Auth: cfg.Auth, Repo: cfg.Repo},
		vision: cfg.Vision,
		limit:  cfg.Limit,
		log:    log,
		now:    cfg.Now,
	}
}

// fail is the one way an error leaves this service.
func (h *Scan) fail(op string, err error) error {
	return handlers.ConnectErr(h.log, op, err)
}

// CreateScan judges a kitchen's photos and stores the verdict.
func (h *Scan) CreateScan(
	ctx context.Context, req *connect.Request[scanv1.CreateScanRequest],
) (*connect.Response[scanv1.CreateScanResponse], error) {
	const op = "create scan"

	// 1. Who is asking: the account comes from the session
	me, err := h.gate.Account(ctx)
	if err != nil {
		return nil, h.fail(op, err)
	}

	// 2. The photos, checked before anything spends one of the player's scans
	photos, invalid := buildPhotos(req.Msg.Photos)
	if invalid != nil {
		return nil, h.fail(op, invalid)
	}
	if req.Msg.ComboId != nil {
		_, c, err := h.gate.ComboMember(ctx, req.Msg.GetComboId())
		if err != nil {
			return nil, h.fail(op, err)
		}
		if c.State == repository.ComboEnded {
			return nil, h.fail(op, repository.ErrComboEnded)
		}
		if !c.ExpiresAt.After(h.now()) {
			return nil, h.fail(op, repository.ErrComboExpired)
		}
	}

	// 3. The model's time costs money, so each player gets scansPerHour
	err = h.limit.Allow(strconv.FormatInt(me.AccountID, 10))
	if err != nil {
		return nil, h.fail(op, err)
	}

	// 4. The model reads the photos at the player's roast level. It takes seconds, so no
	// transaction is open while it does
	level, err := roastLevel(me.RoastLevel)
	if err != nil {
		return nil, h.fail(op, err)
	}
	fridge, err := h.vision.ReadFridge(ctx, photos, level)
	if err != nil {
		return nil, h.fail(op, err)
	}

	// 5. What the model listed is as untrusted as a request: keep what passes, count the rest
	items, dropped := keepModelItems(fridge.Items)
	if dropped > 0 {
		h.log.Warn("dropped items the model listed", "account_id", me.AccountID, "dropped", dropped)
	}

	// 6. Our code judges, so the same list always scores the same
	counted, err := scoringItems(items)
	if err != nil {
		return nil, h.fail(op, err)
	}
	score := scoring.Fridge(counted)

	// 7. Store the verdict, and answer with it as stored
	// Postgres text cannot hold a NUL; the roast's line breaks may be meant, so only NULs go
	roast := strings.ReplaceAll(fridge.Roast, "\x00", "")
	var scan repository.Scan
	if req.Msg.ComboId == nil {
		scan, err = h.repo.CreateScan(ctx, me.AccountID, score, roast, items)
	} else {
		scan, err = h.repo.CreateComboScan(ctx, me.AccountID, req.Msg.GetComboId(), score, roast, items, h.now())
	}
	if err != nil {
		return nil, h.fail(op, err)
	}
	verdict, err := toProtoVerdict(scan)
	if err != nil {
		return nil, h.fail(op, err)
	}
	return connect.NewResponse(&scanv1.CreateScanResponse{Verdict: verdict}), nil
}

// GetScan reads one of the caller's verdicts.
func (h *Scan) GetScan(
	ctx context.Context, req *connect.Request[scanv1.GetScanRequest],
) (*connect.Response[scanv1.GetScanResponse], error) {
	const op = "get scan"

	// 1. Who is asking: the account comes from the session
	me, err := h.gate.Account(ctx)
	if err != nil {
		return nil, h.fail(op, err)
	}

	// 2. Validate the id
	invalid := handlers.Validate(struct {
		ScanID int64 `pfield:"scan_id" validate:"gt=0"`
	}{req.Msg.ScanId})
	if invalid != nil {
		return nil, h.fail(op, invalid)
	}

	// 3. Load it: another player's scan answers as a missing one does
	scan, err := h.repo.ScanByID(ctx, me.AccountID, req.Msg.ScanId)
	if err != nil {
		return nil, h.fail(op, err)
	}
	verdict, err := toProtoVerdict(scan)
	if err != nil {
		return nil, h.fail(op, err)
	}
	return connect.NewResponse(&scanv1.GetScanResponse{Verdict: verdict}), nil
}

// GetLastScan reads the caller's newest verdict, which is unset before their first scan.
func (h *Scan) GetLastScan(
	ctx context.Context, _ *connect.Request[scanv1.GetLastScanRequest],
) (*connect.Response[scanv1.GetLastScanResponse], error) {
	const op = "get last scan"

	// 1. Who is asking: the account comes from the session
	me, err := h.gate.Account(ctx)
	if err != nil {
		return nil, h.fail(op, err)
	}

	// 2. Load it. No scan yet is an answer, not an error
	scan, found, err := h.repo.LastScan(ctx, me.AccountID)
	if err != nil {
		return nil, h.fail(op, err)
	}
	if !found {
		return connect.NewResponse(&scanv1.GetLastScanResponse{}), nil
	}
	verdict, err := toProtoVerdict(scan)
	if err != nil {
		return nil, h.fail(op, err)
	}
	return connect.NewResponse(&scanv1.GetLastScanResponse{Verdict: verdict}), nil
}

// EditIngredients replaces a verdict's list with the player's corrected one. The score stays as
// it was judged; the new list is what the recipes are built from.
func (h *Scan) EditIngredients(
	ctx context.Context, req *connect.Request[scanv1.EditIngredientsRequest],
) (*connect.Response[scanv1.EditIngredientsResponse], error) {
	const op = "edit ingredients"

	// 1. Who is asking: the account comes from the session
	me, err := h.gate.Account(ctx)
	if err != nil {
		return nil, h.fail(op, err)
	}

	// 2. Normalize, then validate the id and every ingredient, so the sheet marks each problem
	items, invalid := buildEdit(req.Msg)
	if invalid != nil {
		return nil, h.fail(op, invalid)
	}

	// 3. Load the scan: another player's answers as a missing one does
	scan, err := h.repo.ScanByID(ctx, me.AccountID, req.Msg.ScanId)
	if err != nil {
		return nil, h.fail(op, err)
	}

	// 4. The write, unless the list is the one already stored: sending it again changes nothing,
	// not even the version the recipes regenerate on
	if !slices.Equal(scan.Items, items) {
		var changed bool
		changed, err = h.repo.ReplaceItems(ctx, me.AccountID, scan.ScanID, items)
		if err != nil {
			return nil, h.fail(op, err)
		}
		if !changed {
			return nil, h.fail(op, repository.ErrNotFound)
		}
		// 5. Re-read, so the player sees the list as stored
		scan, err = h.repo.ScanByID(ctx, me.AccountID, scan.ScanID)
		if err != nil {
			return nil, h.fail(op, err)
		}
	}
	verdict, err := toProtoVerdict(scan)
	if err != nil {
		return nil, h.fail(op, err)
	}
	return connect.NewResponse(&scanv1.EditIngredientsResponse{Verdict: verdict}), nil
}
