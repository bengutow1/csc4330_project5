// Author: Caden Lund
// Package healthz serves health.v1, the liveness and readiness probes every binary mounts.
package healthz

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"

	healthv1 "github.com/cadenlund/fries-in-the-bag/packages/go/apigen/health/v1"
	"github.com/cadenlund/fries-in-the-bag/packages/go/handlers"
)

// Check reports whether one dependency can serve. repository.Repository's Ping is one.
type Check func(ctx context.Context) error

// Health implements health.v1 HealthService.
type Health struct {
	checks []Check
	log    *slog.Logger
}

// NewHealth builds the probes over the checks Ready runs. No checks is valid: a binary with no
// dependencies is always ready. A nil log discards.
func NewHealth(log *slog.Logger, checks ...Check) *Health {
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	return &Health{checks: checks, log: log}
}

// fail is the one way an error leaves this service.
func (h *Health) fail(op string, err error) error {
	return handlers.ConnectErr(h.log, op, err)
}

// Live answers while the process can serve a request. It runs no check on purpose: an outage
// must not get the process restarted, which would only deepen it.
func (h *Health) Live(
	context.Context, *connect.Request[healthv1.LiveRequest],
) (*connect.Response[healthv1.LiveResponse], error) {
	return connect.NewResponse(&healthv1.LiveResponse{}), nil
}

// Ready answers only when every check passes, and stops at the first failure.
func (h *Health) Ready(
	ctx context.Context, _ *connect.Request[healthv1.ReadyRequest],
) (*connect.Response[healthv1.ReadyResponse], error) {
	const op = "ready"

	// Public by design: a probe holds no session. So the cause goes to the log, and the caller
	// hears only "not ready".
	for i, check := range h.checks {
		err := check(ctx)
		if err != nil {
			h.log.Error(op, "check", i, "err", err)
			return nil, h.fail(op, handlers.ErrNotReady)
		}
	}
	return connect.NewResponse(&healthv1.ReadyResponse{}), nil
}
