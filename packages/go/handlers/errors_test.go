// Author: Caden Lund
package handlers_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"maps"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"github.com/cadenlund/fries-in-the-bag/packages/go/handlers"
	"github.com/cadenlund/fries-in-the-bag/packages/go/repository"
)

// TestConnectErrMapsEverySentinelToItsCode: a sentinel that loses its arm reaches the client as a
// redacted internal error. Every sentinel ConnectErr knows belongs in this table.
func TestConnectErrMapsEverySentinelToItsCode(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		sentinel error
		code     connect.Code
	}{
		{name: "not found", sentinel: repository.ErrNotFound, code: connect.CodeNotFound},
		{name: "duplicate", sentinel: repository.ErrDuplicate, code: connect.CodeAlreadyExists},
		{name: "not ready", sentinel: handlers.ErrNotReady, code: connect.CodeUnavailable},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			// Wrapped the way the repository wraps, with ids the caller must never see.
			wrapped := fmt.Errorf("scan by id account=7 scan=3: %w", c.sentinel)

			err := handlers.ConnectErr(discard(), "load scan", wrapped)

			cerr := asConnectErr(t, err)
			if cerr.Code() != c.code {
				t.Errorf("code = %v, want %v", cerr.Code(), c.code)
			}
			if cerr.Message() != c.sentinel.Error() {
				t.Errorf("message = %q, want the sentinel's own %q", cerr.Message(), c.sentinel.Error())
			}
		})
	}
}

// TestConnectErrRedactsAnUnknownError: a driver error names hosts and users. The caller gets
// "internal error" and the log gets the cause.
func TestConnectErrRedactsAnUnknownError(t *testing.T) {
	t.Parallel()
	var logs bytes.Buffer
	cause := errors.New("dial tcp 10.0.0.4:5432: password authentication failed for user fries")

	err := handlers.ConnectErr(jsonLog(&logs), "load scan", fmt.Errorf("scan by id account=7: %w", cause))

	cerr := asConnectErr(t, err)
	if cerr.Code() != connect.CodeInternal {
		t.Errorf("code = %v, want internal", cerr.Code())
	}
	if cerr.Message() != "internal error" {
		t.Errorf("message = %q, want %q", cerr.Message(), "internal error")
	}
	if len(cerr.Details()) != 0 {
		t.Errorf("details = %v, want none", cerr.Details())
	}
	if !strings.Contains(logs.String(), "10.0.0.4") {
		t.Errorf("the log lost the cause: %s", logs.String())
	}
}

// TestConnectErrCarriesEveryFieldViolation: a form shows all its problems in one round trip,
// keyed by the proto field names the client gave its inputs.
func TestConnectErrCarriesEveryFieldViolation(t *testing.T) {
	t.Parallel()
	invalid := handlers.FieldErrors{
		{Field: "handle", Message: "handle is required"},
		{Field: "avatar", Message: "avatar is required"},
	}

	err := handlers.ConnectErr(discard(), "create account", invalid)

	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("got code %v (%v), want invalid_argument", connect.CodeOf(err), err)
	}
	got := fieldViolations(t, err)
	want := map[string]string{
		"handle": "handle is required",
		"avatar": "avatar is required",
	}
	if !maps.Equal(got, want) {
		t.Errorf("violations = %v, want %v", got, want)
	}
}

// TestConnectErrTreatsAHangUpAsCanceledWithoutAnErrorLine: a closed app is not an outage.
func TestConnectErrTreatsAHangUpAsCanceledWithoutAnErrorLine(t *testing.T) {
	t.Parallel()
	var logs bytes.Buffer

	err := handlers.ConnectErr(jsonLog(&logs), "load scan", fmt.Errorf("scan by id account=7: %w", context.Canceled))

	cerr := asConnectErr(t, err)
	if cerr.Code() != connect.CodeCanceled {
		t.Errorf("code = %v, want canceled", cerr.Code())
	}
	if strings.Contains(cerr.Message(), "account=7") {
		t.Errorf("message leaks the wrap: %q", cerr.Message())
	}
	if logs.Len() != 0 {
		t.Errorf("a hang-up was logged: %s", logs.String())
	}
}

// TestConnectErrReportsADeadlineWithoutTheWrap: the deadline is the caller's to know, the ids in
// the wrap are not.
func TestConnectErrReportsADeadlineWithoutTheWrap(t *testing.T) {
	t.Parallel()

	err := handlers.ConnectErr(discard(), "load scan", fmt.Errorf("scan by id account=7: %w", context.DeadlineExceeded))

	cerr := asConnectErr(t, err)
	if cerr.Code() != connect.CodeDeadlineExceeded {
		t.Errorf("code = %v, want deadline_exceeded", cerr.Code())
	}
	if strings.Contains(cerr.Message(), "account=7") {
		t.Errorf("message leaks the wrap: %q", cerr.Message())
	}
}

// TestConnectErrPassesADeliberateRefusalThrough: a handler that built its own Connect error meant
// that code and that sentence.
func TestConnectErrPassesADeliberateRefusalThrough(t *testing.T) {
	t.Parallel()
	refusal := connect.NewError(connect.CodeFailedPrecondition, errors.New("that combo has already ended"))

	err := handlers.ConnectErr(discard(), "join combo", fmt.Errorf("join combo=7: %w", refusal))

	cerr := asConnectErr(t, err)
	if cerr.Code() != connect.CodeFailedPrecondition {
		t.Errorf("code = %v, want failed_precondition", cerr.Code())
	}
	if cerr.Message() != "that combo has already ended" {
		t.Errorf("message = %q, want the refusal's own", cerr.Message())
	}
}
