// Author: Caden Lund
// Package board serves board.v1 BoardService: the boards, ranked by the server and paged, the
// caller's own row, and the Fridge of the Week. A week starts Monday at 00:00, campus time.
package board

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect"

	boardv1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/board/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/auth"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers"
	"github.com/bengutow1/csc4330_project5/packages/go/points"
	"github.com/bengutow1/csc4330_project5/packages/go/repository"
)

// Board implements board.v1 BoardService.
type Board struct {
	repo *repository.Repository
	gate handlers.Gate
	now  func() time.Time
	log  *slog.Logger
}

// Config is what Board needs to exist.
type Config struct {
	Repo *repository.Repository
	Auth *auth.Auth
	// Now is the clock that says which week it is. Nil means time.Now.
	Now func() time.Time
	Log *slog.Logger
}

// NewBoard builds the service. A nil Log discards.
func NewBoard(cfg Config) *Board {
	log := cfg.Log
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	return &Board{repo: cfg.Repo, gate: handlers.Gate{Auth: cfg.Auth}, now: now, log: log}
}

// fail is the one way an error leaves this service.
func (h *Board) fail(op string, err error) error {
	return handlers.ConnectErr(h.log, op, err)
}

// GetBoard returns a page of one board, best first, and the caller's own row on every page.
func (h *Board) GetBoard(
	ctx context.Context, req *connect.Request[boardv1.GetBoardRequest],
) (*connect.Response[boardv1.GetBoardResponse], error) {
	const op = "get board"

	// 1. Who is asking: the account comes from the session, for the caller's own row
	me, err := h.gate.Account(ctx)
	if err != nil {
		return nil, h.fail(op, err)
	}

	// 2. Normalize, then validate everything the request carries
	in, invalid := buildGetBoard(req.Msg)
	if invalid != nil {
		return nil, h.fail(op, invalid)
	}
	// All time counts from the zero time; this week from Monday 00:00, campus time.
	var since time.Time
	if in.rng == boardv1.Range_RANGE_THIS_WEEK {
		since = points.WeekStart(h.now())
	}

	// 3. The page, one row past it to know whether there is more, and the caller's own row
	var rows []repository.BoardRow
	var mine repository.BoardRow
	var ranked bool
	switch in.board {
	case boardv1.Board_BOARD_CHEFS:
		rows, err = h.repo.Chefs(ctx, since, in.after, in.limit+1)
		if err != nil {
			return nil, h.fail(op, err)
		}
		mine, ranked, err = h.repo.ChefsRow(ctx, me.AccountID, since)
		if err != nil {
			return nil, h.fail(op, err)
		}
	case boardv1.Board_BOARD_HALL_OF_FRIES:
		rows, err = h.repo.HallOfFries(ctx, since, in.after, in.limit+1)
		if err != nil {
			return nil, h.fail(op, err)
		}
		mine, ranked, err = h.repo.HallOfFriesRow(ctx, me.AccountID, since)
		if err != nil {
			return nil, h.fail(op, err)
		}
	case boardv1.Board_BOARD_UNSPECIFIED:
		fallthrough
	default:
		// buildGetBoard refused every other value, so reaching here is a bug, not a request.
		return nil, h.fail(op, fmt.Errorf("board %v passed validation", in.board))
	}

	return connect.NewResponse(toProtoBoard(in, rows, mine, ranked)), nil
}

// GetFridgeOfTheWeek returns the week's best scan and whose it was, or no fridge before anyone
// scans this week.
func (h *Board) GetFridgeOfTheWeek(
	ctx context.Context, _ *connect.Request[boardv1.GetFridgeOfTheWeekRequest],
) (*connect.Response[boardv1.GetFridgeOfTheWeekResponse], error) {
	const op = "get fridge of the week"

	// 1. Who is asking: the board is for players
	_, err := h.gate.Account(ctx)
	if err != nil {
		return nil, h.fail(op, err)
	}

	// 2. The best scan since Monday 00:00, campus time
	fridge, found, err := h.repo.FridgeOfTheWeek(ctx, points.WeekStart(h.now()))
	if err != nil {
		return nil, h.fail(op, err)
	}
	return connect.NewResponse(toProtoFridge(fridge, found)), nil
}
