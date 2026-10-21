// Author: Caden Lund
package auth_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/alexedwards/scs/pgxstore"
	"github.com/alexedwards/scs/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // the database/sql driver "pgx", which pgtestdb opens
	"github.com/peterldowns/pgtestdb"
	"github.com/peterldowns/pgtestdb/migrators/goosemigrator"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/bengutow1/csc4330_project5/packages/go/auth"
)

// The tag docker/compose.yml runs: a suite that passes on a different Postgres than the one we
// develop on proves little.
const postgresImage = "postgres:18-alpine"

// base is the container's admin database. pgtestdb creates each test's database from it and never
// runs a test's queries on it.
var base pgtestdb.Config

// TestMain starts one Postgres for the whole package. Sessions live in Postgres, so the tests
// store them there too, in each test's own freshly migrated database.
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

// newSessions is the session manager main.go builds, over the test's database. The cleanup
// goroutine is off: each test's database is gone before it would first run.
func newSessions(pool *pgxpool.Pool) *scs.SessionManager {
	sessions := scs.New()
	sessions.Store = pgxstore.NewWithCleanupInterval(pool, 0)
	sessions.Lifetime = auth.Lifetime
	return sessions
}
