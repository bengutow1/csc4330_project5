// Author: Caden Lund
// Command seed fills the local database with the demo week (scripts/seed/demo), so the board, the
// profiles and the Fridge of the Week look alive while screens are built and at the demo. Run it
// with `moon run db:seed`. It is local only: it never runs in CI, at boot or from a migration,
// and a second run changes nothing.
package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bengutow1/csc4330_project5/packages/go/logging"
	"github.com/bengutow1/csc4330_project5/packages/go/repository"
	"github.com/bengutow1/csc4330_project5/scripts/seed/demo"
)

func main() {
	os.Exit(run())
}

// run seeds and says what it wrote, returning the exit code, so the pool closes before the exit.
func run() int {
	log := logging.New("seed")
	slog.SetDefault(log)

	// db/moon.yml's seed task sets it to the local database from docker/compose.yml.
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Error("DATABASE_URL is not set")
		return 1
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Error("open database", "err", err)
		return 1
	}
	defer pool.Close()

	report, err := demo.Seed(ctx, repository.New(pool), time.Now())
	if err != nil {
		log.Error("seed", "err", err)
		return 1
	}
	if report.AlreadySeeded {
		log.Info("the demo is already here; nothing changed")
		return 0
	}
	log.Info("seeded the demo week", "players", report.Players, "scans", report.Scans, "fry_runs", report.FryRuns)
	return 0
}
