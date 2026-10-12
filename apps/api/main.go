// Author: Caden Lund
// Command api is the server every Connect service is mounted on. It only wires: it reads the
// environment, builds the logger and the pool, mounts the services and drains on shutdown.
// Anything that decides something lives in packages/go.
//
// Adding a service is three lines: build it from its handlers subpackage, then mount it with
// readCap below.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/cadenlund/fries-in-the-bag/packages/go/apigen/health/v1/healthv1connect"
	"github.com/cadenlund/fries-in-the-bag/packages/go/handlers/healthz"
	"github.com/cadenlund/fries-in-the-bag/packages/go/logging"
	"github.com/cadenlund/fries-in-the-bag/packages/go/repository"
)

const addr = ":8080"

func main() {
	log := logging.New("api")
	slog.SetDefault(log)
	// SIGINT and SIGTERM cancel ctx, and the server drains below.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Error("DATABASE_URL is required")
		os.Exit(1)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Error("open database pool", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	// One repository over the one pool, shared by every service. Migrations never run here:
	// `moon run db:migrate` applies them.
	repo := repository.New(pool)

	// Every service reads request bodies under the same cap.
	readCap := connect.WithReadMaxBytes(4 << 20)

	r := chi.NewRouter()
	r.Use(logging.RequestLogger(log))
	r.Use(logging.Recoverer(log))

	r.Mount(healthv1connect.NewHealthServiceHandler(healthz.NewHealth(log, repo.Ping), readCap))

	// No read or write timeout on purpose: a server stream stays open for minutes.
	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}
	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	log.Info("api listening", "addr", addr)

	select {
	case err := <-errc:
		log.Error("server stopped", "err", err)
		os.Exit(1)
	case <-ctx.Done():
	}
	stop() // a second SIGTERM force-quits
	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = srv.Shutdown(shutdownCtx)
	if err != nil {
		_ = srv.Close() // the drain timed out; drop what is left
	}
}
