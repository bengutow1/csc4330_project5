// Author: Caden Lund
// Package handlers holds what every Connect service shares: the one error boundary, field
// violations, validation, the gate and the per-IP limiter. Each product surface is a subpackage
// (healthz and account now, then scan, cook, fries, combo and board).
package handlers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"

	"github.com/bengutow1/csc4330_project5/packages/go/auth"
	"github.com/bengutow1/csc4330_project5/packages/go/repository"
)

var (
	// ErrNotReady fires when a readiness check fails. The probe is public, so the real failure
	// stays in the server log.
	ErrNotReady = errors.New("not ready")
	// ErrTooManyRequests fires when one client IP has spent a public rpc's limit for now.
	ErrTooManyRequests = errors.New("too many tries, wait a few minutes and try again")
)

// ConnectErr is the only place an error becomes a wire code; each service reaches it through its
// fail method. A sentinel answers with its own text, never the wrap, which carries ids and driver
// detail. Anything else is logged in full and answers "internal error".
//
// A new known condition is three things in one change: a sentinel in its package, an arm here,
// and a test that pins the mapping.
func ConnectErr(log *slog.Logger, op string, err error) error {
	var refusal *connect.Error
	var fieldErrs FieldErrors
	switch {
	// A refusal the handler built on purpose. Rare: prefer a sentinel.
	case errors.As(err, &refusal):
		return refusal
	// The client hung up. Not a server fault, so no ERROR line.
	case errors.Is(err, context.Canceled):
		return connect.NewError(connect.CodeCanceled, context.Canceled)
	case errors.Is(err, context.DeadlineExceeded):
		return connect.NewError(connect.CodeDeadlineExceeded, context.DeadlineExceeded)
	case errors.As(err, &fieldErrs):
		return invalidArgument(log, op, fieldErrs)

	// repository
	case errors.Is(err, repository.ErrNotFound):
		return connect.NewError(connect.CodeNotFound, repository.ErrNotFound)
	case errors.Is(err, repository.ErrDuplicate):
		return connect.NewError(connect.CodeAlreadyExists, repository.ErrDuplicate)

	// auth
	// The app reads Unauthenticated as "back to onboarding", with no toast.
	case errors.Is(err, auth.ErrNotAuthenticated):
		return connect.NewError(connect.CodeUnauthenticated, auth.ErrNotAuthenticated)
	// PermissionDenied, not NotFound: the caller can see the thing, they just may not do this.
	case errors.Is(err, auth.ErrForbidden):
		return connect.NewError(connect.CodePermissionDenied, auth.ErrForbidden)

	// handlers
	// Unavailable, not Internal: the process is fine and a dependency is not, so a probe and a
	// retrying client both read it as "try again".
	case errors.Is(err, ErrNotReady):
		return connect.NewError(connect.CodeUnavailable, ErrNotReady)
	case errors.Is(err, ErrTooManyRequests):
		return connect.NewError(connect.CodeResourceExhausted, ErrTooManyRequests)

	default:
		log.Error(op, "err", err)
		return connect.NewError(connect.CodeInternal, errors.New("internal error"))
	}
}

// invalidArgument answers with every violation in an error.v1.BadRequest detail. A detail that
// fails to marshal degrades to the code and message, never to a failed response.
func invalidArgument(log *slog.Logger, op string, fieldErrs FieldErrors) error {
	cerr := connect.NewError(connect.CodeInvalidArgument, fieldErrs)
	detail, err := connect.NewErrorDetail(fieldErrs.detail())
	if err != nil {
		log.Error(op, "err", fmt.Errorf("marshal field violations: %w", err))
		return cerr
	}
	cerr.AddDetail(detail)
	return cerr
}
