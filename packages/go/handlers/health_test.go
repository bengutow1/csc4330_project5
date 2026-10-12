// Author: Caden Lund
package handlers_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"connectrpc.com/connect"

	healthv1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/health/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/apigen/health/v1/healthv1connect"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers/healthz"
	"github.com/bengutow1/csc4330_project5/packages/go/repository"
)

// TestReadyAnswersWhenTheDatabaseDoes: the probe passes over a real, migrated Postgres.
func TestReadyAnswersWhenTheDatabaseDoes(t *testing.T) {
	t.Parallel()
	client, _ := healthStack(t, nil, newRepo(t).Ping)

	_, err := client.Ready(t.Context(), connect.NewRequest(&healthv1.ReadyRequest{}))

	if err != nil {
		t.Fatalf("ready: %v", err)
	}
}

// TestReadyAnswersUnavailableOnceTheDatabaseIsGone keeps traffic away from a replica that cannot
// reach Postgres.
func TestReadyAnswersUnavailableOnceTheDatabaseIsGone(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	repo := repository.New(pool)
	pool.Close()
	client, _ := healthStack(t, nil, repo.Ping)

	_, err := client.Ready(t.Context(), connect.NewRequest(&healthv1.ReadyRequest{}))

	if connect.CodeOf(err) != connect.CodeUnavailable {
		t.Fatalf("got code %v (%v), want unavailable", connect.CodeOf(err), err)
	}
}

// TestReadyNeverTellsTheCallerWhy: the probe is public, and a driver error names hosts, ports and
// users. The cause belongs in the log.
func TestReadyNeverTellsTheCallerWhy(t *testing.T) {
	t.Parallel()
	var logs lockedBuffer
	leak := errors.New("failed to connect to `user=fries database=fries`: dial tcp 10.0.0.4:5432: connection refused")
	client, _ := healthStack(t, jsonLog(&logs), func(context.Context) error { return leak })

	_, err := client.Ready(t.Context(), connect.NewRequest(&healthv1.ReadyRequest{}))

	cerr := asConnectErr(t, err)
	if cerr.Code() != connect.CodeUnavailable {
		t.Errorf("code = %v, want unavailable", cerr.Code())
	}
	if cerr.Message() != "not ready" {
		t.Errorf("message = %q, want %q", cerr.Message(), "not ready")
	}
	if !strings.Contains(logs.String(), "10.0.0.4") {
		t.Errorf("the log lost the cause: %s", logs.String())
	}
}

// TestReadyStopsAtTheFirstFailingCheck: the verdict is already decided, and a later check may be
// slow.
func TestReadyStopsAtTheFirstFailingCheck(t *testing.T) {
	t.Parallel()
	var ran atomic.Bool
	client, _ := healthStack(t, nil, down, func(context.Context) error {
		ran.Store(true)
		return nil
	})

	_, err := client.Ready(t.Context(), connect.NewRequest(&healthv1.ReadyRequest{}))

	if connect.CodeOf(err) != connect.CodeUnavailable {
		t.Fatalf("got code %v (%v), want unavailable", connect.CodeOf(err), err)
	}
	if ran.Load() {
		t.Error("a check ran after an earlier one failed")
	}
}

// TestLiveIgnoresEveryDependency: if a database outage failed liveness, the orchestrator would
// restart the process, which only deepens the outage.
func TestLiveIgnoresEveryDependency(t *testing.T) {
	t.Parallel()
	client, _ := healthStack(t, nil, down)

	_, err := client.Live(t.Context(), connect.NewRequest(&healthv1.LiveRequest{}))

	if err != nil {
		t.Fatalf("live failed with a dependency down: %v", err)
	}
}

// TestProbesAnswerAPlainGet: an orchestrator's probe sends a plain GET and reads only the status,
// so both rpcs must answer one, and a failing Ready must not be a 2xx.
func TestProbesAnswerAPlainGet(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		rpc    string
		check  healthz.Check
		status int
	}{
		{name: "live, with a dependency down", rpc: "Live", check: down, status: http.StatusOK},
		{name: "ready", rpc: "Ready", check: up, status: http.StatusOK},
		{name: "ready, with a dependency down", rpc: "Ready", check: down, status: http.StatusServiceUnavailable},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			_, baseURL := healthStack(t, nil, c.check)
			url := baseURL + "/health.v1.HealthService/" + c.rpc + "?encoding=json&message=%7B%7D"
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
			if err != nil {
				t.Fatalf("build request: %v", err)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("GET %s: %v", c.rpc, err)
			}
			_ = resp.Body.Close() // only the status matters here

			if resp.StatusCode != c.status {
				t.Errorf("GET %s = %d, want %d", c.rpc, resp.StatusCode, c.status)
			}
		})
	}
}

func up(context.Context) error   { return nil }
func down(context.Context) error { return errors.New("database down") }

// healthStack serves the probes over checks on a real HTTP server and returns the generated client
// and the server's base URL.
func healthStack(t *testing.T, log *slog.Logger, checks ...healthz.Check) (healthv1connect.HealthServiceClient, string) {
	t.Helper()
	path, h := healthv1connect.NewHealthServiceHandler(healthz.NewHealth(log, checks...))
	baseURL := serve(t, path, h)
	return healthv1connect.NewHealthServiceClient(http.DefaultClient, baseURL), baseURL
}
