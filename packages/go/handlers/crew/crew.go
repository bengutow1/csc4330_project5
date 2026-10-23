// Author: Ben Gutowski
// Package crew serves crew.v1 CrewService: making a crew, joining one by its six-digit code,
// leaving, and listing the caller's crews. A crew's board is board.v1 GetBoard with its crew_id.
package crew

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"time"

	"connectrpc.com/connect"

	crewv1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/crew/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/auth"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers"
	"github.com/bengutow1/csc4330_project5/packages/go/repository"
)

// maxCrews is the most crews a player is in, which keeps ListMyCrews one short, unpaged list.
const maxCrews = 10

// Six digits are cheap to guess, so each client IP gets joinsPerIP joins in a window.
const (
	joinsPerIP = 20
	joinWindow = 10 * time.Minute
)

// codeDraws is how many fresh codes CreateCrew tries before giving up. A million codes and a
// class's worth of crews make a second draw rare and a fifth one a bug.
const codeDraws = 5

// Crew implements crew.v1 CrewService.
type Crew struct {
	repo   *repository.Repository
	gate   handlers.Gate
	limits Limits
	log    *slog.Logger
}

// Config is what Crew needs to exist.
type Config struct {
	Repo *repository.Repository
	Auth *auth.Auth
	// Limits caps joining. main.go passes DefaultLimits; the zero value limits nothing.
	Limits Limits
	Log    *slog.Logger
}

// Limits holds one limiter per guessable rpc, keyed by client IP. A nil one limits nothing, which
// is how test fixtures turn limits off.
type Limits struct {
	Join *handlers.Limiter
}

// DefaultLimits are the limits the API serves with.
func DefaultLimits() Limits {
	return Limits{Join: handlers.NewLimiter(joinsPerIP, joinWindow)}
}

// NewCrew builds the service. A nil Log discards.
func NewCrew(cfg Config) *Crew {
	log := cfg.Log
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	return &Crew{repo: cfg.Repo, gate: handlers.Gate{Auth: cfg.Auth}, limits: cfg.Limits, log: log}
}

// fail is the one way an error leaves this service.
func (h *Crew) fail(op string, err error) error {
	return handlers.ConnectErr(h.log, op, err)
}

// CreateCrew makes a crew with the caller as its first member and a fresh code.
func (h *Crew) CreateCrew(
	ctx context.Context, req *connect.Request[crewv1.CreateCrewRequest],
) (*connect.Response[crewv1.CreateCrewResponse], error) {
	const op = "create crew"

	// 1. Who is asking: the account comes from the session
	me, err := h.gate.Account(ctx)
	if err != nil {
		return nil, h.fail(op, err)
	}

	// 2. Normalize, then validate everything the request carries
	name, invalid := buildName(req.Msg.Name)
	if invalid != nil {
		return nil, h.fail(op, invalid)
	}

	// 3. The write, drawing again when another crew holds the code
	for range codeDraws {
		code, err := newCode()
		if err != nil {
			return nil, h.fail(op, err)
		}
		made, err := h.repo.CreateCrew(ctx, me.AccountID, name, code, maxCrews)
		if errors.Is(err, repository.ErrDuplicate) {
			continue
		}
		if err != nil {
			return nil, h.fail(op, err)
		}
		return connect.NewResponse(&crewv1.CreateCrewResponse{Crew: toProtoCrew(made)}), nil
	}
	return nil, h.fail(op, fmt.Errorf("no free crew code in %d draws account=%d", codeDraws, me.AccountID))
}

// JoinCrew adds the caller to the crew with this code. Joining one they are in changes nothing.
func (h *Crew) JoinCrew(
	ctx context.Context, req *connect.Request[crewv1.JoinCrewRequest],
) (*connect.Response[crewv1.JoinCrewResponse], error) {
	const op = "join crew"

	// 1. Who is asking: the account comes from the session
	me, err := h.gate.Account(ctx)
	if err != nil {
		return nil, h.fail(op, err)
	}

	// 2. A code is six guessable digits, so each client IP gets joinsPerIP tries
	err = h.limits.Join.Allow(handlers.ClientIP(req.Peer()))
	if err != nil {
		return nil, h.fail(op, err)
	}

	// 3. Normalize, then validate everything the request carries
	code, invalid := buildCode(req.Msg.Code)
	if invalid != nil {
		return nil, h.fail(op, invalid)
	}

	// 4. The write: a wrong code and a player at the cap are refused inside it
	joined, err := h.repo.JoinCrew(ctx, me.AccountID, code, maxCrews)
	if err != nil {
		return nil, h.fail(op, err)
	}
	return connect.NewResponse(&crewv1.JoinCrewResponse{Crew: toProtoCrew(joined)}), nil
}

// LeaveCrew takes the caller out of a crew they are in. The last one out ends the crew.
func (h *Crew) LeaveCrew(
	ctx context.Context, req *connect.Request[crewv1.LeaveCrewRequest],
) (*connect.Response[crewv1.LeaveCrewResponse], error) {
	const op = "leave crew"

	// 1. Who is asking: the account comes from the session
	me, err := h.gate.Account(ctx)
	if err != nil {
		return nil, h.fail(op, err)
	}

	// 2. Validate the crew the request names
	invalid := handlers.Validate(struct {
		CrewID int64 `pfield:"crew_id" validate:"gt=0"`
	}{req.Msg.CrewId})
	if invalid != nil {
		return nil, h.fail(op, invalid)
	}

	// 3. The write: a crew they are not in left nothing, and says so as not found
	left, err := h.repo.LeaveCrew(ctx, me.AccountID, req.Msg.CrewId)
	if err != nil {
		return nil, h.fail(op, err)
	}
	if !left {
		return nil, h.fail(op, repository.ErrNotFound)
	}
	return connect.NewResponse(&crewv1.LeaveCrewResponse{}), nil
}

// ListMyCrews returns the caller's crews, the one they joined first first.
func (h *Crew) ListMyCrews(
	ctx context.Context, _ *connect.Request[crewv1.ListMyCrewsRequest],
) (*connect.Response[crewv1.ListMyCrewsResponse], error) {
	const op = "list my crews"

	// 1. Who is asking: the account comes from the session
	me, err := h.gate.Account(ctx)
	if err != nil {
		return nil, h.fail(op, err)
	}

	// 2. Their crews, at most maxCrews, so one unpaged list
	crews, err := h.repo.ListMyCrews(ctx, me.AccountID)
	if err != nil {
		return nil, h.fail(op, err)
	}
	return connect.NewResponse(&crewv1.ListMyCrewsResponse{Crews: toProtoCrews(crews)}), nil
}

// newCode draws a six-digit crew code from crypto/rand, uniform over all million (go-auth.md).
func newCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", fmt.Errorf("draw crew code: %w", err)
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}
