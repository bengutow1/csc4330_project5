// Author: Caden Lund
package handlers_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // the database/sql driver "pgx", which pgtestdb opens
	"github.com/peterldowns/pgtestdb"
	"github.com/peterldowns/pgtestdb/migrators/goosemigrator"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	errorv1 "github.com/cadenlund/fries-in-the-bag/packages/go/apigen/error/v1"
	"github.com/cadenlund/fries-in-the-bag/packages/go/repository"
)

// The tag docker/compose.yml runs: a suite that passes on a different Postgres than the one we
// develop on proves little.
const postgresImage = "postgres:18-alpine"

// base is the container's admin database. pgtestdb creates each test's database from it and never
// runs a test's queries on it.
var base pgtestdb.Config

// TestMain starts one Postgres for the whole package. Handler tests drive the real serving stack:
// the service mounted on an HTTP server, called through the generated client, over a real
// database. Each test gets its own database, so they all run in parallel.
func TestMain(m *testing.M) {
	ctx := context.Background()
	container, err := tcpostgres.Run(ctx, postgresImage,
		tcpostgres.WithDatabase("postgres"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "start postgres (is Docker running?): %v\n", err)
		os.Exit(1)
	}
	host, err := container.Host(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "container host: %v\n", err)
		os.Exit(1)
	}
	port, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		fmt.Fprintf(os.Stderr, "container port: %v\n", err)
		os.Exit(1)
	}
	base = pgtestdb.Config{
		DriverName: "pgx",
		Host:       host,
		Port:       port.Port(),
		User:       "postgres",
		Password:   "postgres",
		Database:   "postgres",
		Options:    "sslmode=disable",
	}

	code := m.Run()

	// Not deferred: os.Exit below would skip it.
	err = testcontainers.TerminateContainer(container)
	if err != nil {
		fmt.Fprintf(os.Stderr, "terminate postgres: %v\n", err)
	}
	os.Exit(code)
}

// newPool gives the calling test its own freshly migrated database and a pool onto it.
func newPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	// io/fs paths may not contain "..", so the walk up to the repo root lives in os.DirFS's argument.
	migrator := goosemigrator.New("migrations", goosemigrator.WithFS(os.DirFS("../../../db")))
	conf := pgtestdb.Custom(t, base, migrator)
	pool, err := pgxpool.New(t.Context(), conf.URL())
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// newRepo is newPool behind a Repository.
func newRepo(t *testing.T) *repository.Repository {
	t.Helper()
	return repository.New(newPool(t))
}

// serve mounts one Connect service on a real HTTP server for the length of the test and returns
// the server's base URL.
func serve(t *testing.T, path string, h http.Handler) string {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(path, h)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv.URL
}

// asConnectErr fails the test unless err is a Connect error, and returns it.
func asConnectErr(t *testing.T, err error) *connect.Error {
	t.Helper()
	var cerr *connect.Error
	if !errors.As(err, &cerr) {
		t.Fatalf("not a connect error: %v", err)
	}
	return cerr
}

// fieldViolations decodes the error.v1.BadRequest detail into field name → message, which is
// what the client renders under each input.
func fieldViolations(t *testing.T, err error) map[string]string {
	t.Helper()
	cerr := asConnectErr(t, err)
	out := map[string]string{}
	for _, d := range cerr.Details() {
		msg, derr := d.Value()
		if derr != nil {
			t.Fatalf("decode error detail: %v", derr)
		}
		br, ok := msg.(*errorv1.BadRequest)
		if !ok {
			continue
		}
		for _, v := range br.FieldViolations {
			out[v.Field] = v.Description
		}
	}
	return out
}

func discard() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func jsonLog(w io.Writer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(w, nil))
}

// lockedBuffer collects log lines the server's goroutines write while the test goroutine reads.
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}
