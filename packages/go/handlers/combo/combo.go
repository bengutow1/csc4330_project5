// Author: LAPTOP-BPMQJP10\andyk
// Package combo serves the two-player lobby and the host's controls.
package combo

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"connectrpc.com/connect"

	combov1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/combo/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/auth"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers"
	"github.com/bengutow1/csc4330_project5/packages/go/repository"
)

// Combo implements combo.v1 ComboService and ComboHostService.
type Combo struct {
	repo  *repository.Repository
	gate  handlers.Gate
	limit *handlers.Limiter
	now   func() time.Time
	code  func() (string, error)
	log   *slog.Logger
}

// Config supplies sessions, persistence and an injectable clock and invite source.
type Config struct {
	Repo  *repository.Repository
	Auth  *auth.Auth
	Limit *handlers.Limiter      // nil disables limits in tests.
	Now   func() time.Time       // nil uses the wall clock.
	Code  func() (string, error) // nil uses crypto/rand.
	Log   *slog.Logger
}

// DefaultLimit slows invite guessing across accounts from the same client IP.
func DefaultLimit() *handlers.Limiter { return handlers.NewLimiter(12, 10*time.Minute) }

// NewCombo builds both surfaces with the same gates and clock.
func NewCombo(cfg Config) *Combo {
	if cfg.Log == nil {
		cfg.Log = slog.New(slog.DiscardHandler)
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.Code == nil {
		cfg.Code = auth.NewCode
	}
	return &Combo{repo: cfg.Repo, gate: handlers.Gate{Auth: cfg.Auth, Repo: cfg.Repo}, limit: cfg.Limit, now: cfg.Now, code: cfg.Code, log: cfg.Log}
}

func (h *Combo) fail(op string, err error) error { return handlers.ConnectErr(h.log, op, err) }

// StartCombo retries only invite collisions; every successful invite includes its host.
func (h *Combo) StartCombo(
	ctx context.Context, _ *connect.Request[combov1.StartComboRequest],
) (*connect.Response[combov1.StartComboResponse], error) {
	const op = "start combo"
	me, err := h.gate.Account(ctx)
	if err != nil {
		return nil, h.fail(op, err)
	}
	for range 10 {
		code, err := h.code()
		if err != nil {
			return nil, h.fail(op, err)
		}
		c, err := h.repo.CreateCombo(ctx, me.AccountID, code, h.now())
		if errors.Is(err, repository.ErrDuplicate) {
			continue
		}
		if err != nil {
			return nil, h.fail(op, err)
		}
		wire, err := toProtoCombo(c)
		if err != nil {
			return nil, h.fail(op, err)
		}
		return connect.NewResponse(&combov1.StartComboResponse{Combo: wire}), nil
	}
	return nil, h.fail(op, fmt.Errorf("combo: invite collisions exhausted"))
}

// JoinCombo validates the invite and spends the client's guess before the guarded write.
func (h *Combo) JoinCombo(
	ctx context.Context, req *connect.Request[combov1.JoinComboRequest],
) (*connect.Response[combov1.JoinComboResponse], error) {
	const op = "join combo"
	me, err := h.gate.Account(ctx)
	if err != nil {
		return nil, h.fail(op, err)
	}
	err = h.limit.Allow(handlers.ClientIP(req.Peer()))
	if err != nil {
		return nil, h.fail(op, err)
	}
	code := strings.TrimSpace(req.Msg.Code)
	err = validateCode(code)
	if err != nil {
		return nil, h.fail(op, err)
	}
	id, err := h.repo.JoinCombo(ctx, me.AccountID, code, h.now())
	if err != nil {
		return nil, h.fail(op, err)
	}
	c, err := h.repo.ComboByID(ctx, me.AccountID, id)
	if err != nil {
		return nil, h.fail(op, err)
	}
	wire, err := toProtoCombo(c)
	if err != nil {
		return nil, h.fail(op, err)
	}
	return connect.NewResponse(&combov1.JoinComboResponse{Combo: wire}), nil
}

// GetCombo reads both scan statuses from the same database snapshot.
func (h *Combo) GetCombo(
	ctx context.Context, req *connect.Request[combov1.GetComboRequest],
) (*connect.Response[combov1.GetComboResponse], error) {
	const op = "get combo"
	_, c, err := h.gate.ComboMember(ctx, req.Msg.ComboId)
	if err != nil {
		return nil, h.fail(op, err)
	}
	wire, err := toProtoCombo(c)
	if err != nil {
		return nil, h.fail(op, err)
	}
	return connect.NewResponse(&combov1.GetComboResponse{Combo: wire}), nil
}

// EndCombo keeps its history visible to members while closing the invite and scanning.
func (h *Combo) EndCombo(
	ctx context.Context, req *connect.Request[combov1.EndComboRequest],
) (*connect.Response[combov1.EndComboResponse], error) {
	const op = "end combo"
	me, _, err := h.gate.ComboHost(ctx, req.Msg.ComboId)
	if err != nil {
		return nil, h.fail(op, err)
	}
	changed, err := h.repo.EndCombo(ctx, me.AccountID, req.Msg.ComboId)
	if err != nil {
		return nil, h.fail(op, err)
	}
	if !changed {
		return nil, h.fail(op, repository.ErrNotFound)
	}
	return connect.NewResponse(&combov1.EndComboResponse{}), nil
}

// RemovePlayer frees the guest seat without deleting either player's own scans.
func (h *Combo) RemovePlayer(
	ctx context.Context, req *connect.Request[combov1.RemovePlayerRequest],
) (*connect.Response[combov1.RemovePlayerResponse], error) {
	const op = "remove combo player"
	me, _, err := h.gate.ComboHost(ctx, req.Msg.ComboId)
	if err != nil {
		return nil, h.fail(op, err)
	}
	changed, err := h.repo.RemoveComboPlayer(ctx, me.AccountID, req.Msg.ComboId)
	if err != nil {
		return nil, h.fail(op, err)
	}
	if !changed {
		return nil, h.fail(op, repository.ErrNotFound)
	}
	return connect.NewResponse(&combov1.RemovePlayerResponse{}), nil
}
