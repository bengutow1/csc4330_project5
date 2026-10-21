// Author: Caden Lund
// Command api is the server every Connect service is mounted on. It only wires: it reads the
// environment, builds the logger and the pool, mounts the services and drains on shutdown.
// Anything that decides something lives in packages/go.
//
// Adding a service is three lines: build it from its handlers subpackage, then mount it with
// readCap below.
package main

import (
	"cmp"
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"github.com/alexedwards/scs/pgxstore"
	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bengutow1/csc4330_project5/packages/go/apigen/account/v1/accountv1connect"
	"github.com/bengutow1/csc4330_project5/packages/go/apigen/fries/v1/friesv1connect"
	"github.com/bengutow1/csc4330_project5/packages/go/apigen/health/v1/healthv1connect"
	"github.com/bengutow1/csc4330_project5/packages/go/auth"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers/account"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers/fries"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers/healthz"
	"github.com/bengutow1/csc4330_project5/packages/go/logging"
	"github.com/bengutow1/csc4330_project5/packages/go/places"
	"github.com/bengutow1/csc4330_project5/packages/go/repository"
	"github.com/bengutow1/csc4330_project5/packages/go/vision"
)

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
	// PORT lets a second checkout (an Omni task) run its own API beside this one.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port
	// Read once for every vendor below and the session cookie: "production" makes a missing key
	// stop the boot and the cookie Secure. Anything else is local dev.
	appEnv := os.Getenv("APP_ENV")

	// The vision model. Without a key the API serves the mock's canned fridges, except in
	// production, where a missing key stops the boot: a real deployment never hands out a fake
	// verdict.
	visionKey := os.Getenv("VISION_API_KEY")
	visionPin := os.Getenv("VISION_MOCK")
	switch {
	case visionKey != "":
		// The real client is still to come (#20), and serving the mock anyway would ignore the key
		// without a word.
		log.Error("VISION_API_KEY is set, but there is no vision client yet; unset it to serve the mock")
		os.Exit(1)
	case appEnv == "production":
		log.Error("VISION_API_KEY is required in production")
		os.Exit(1)
	}
	platePin := os.Getenv("VISION_MOCK_PLATE")
	visionMock, err := vision.NewMock(visionPin, platePin)
	if err != nil {
		// err says which of the two names a pin the mock does not have.
		log.Error("VISION_MOCK or VISION_MOCK_PLATE names no mock pin", "err", err)
		os.Exit(1)
	}
	log.Info("vision mock is live", "serving", cmp.Or(visionPin, "every fridge in turn"),
		"plates", cmp.Or(platePin, "every plate passes"))
	_ = visionMock // nothing calls the model until the scan (#17) and cook (#21) services mount

	// The places provider. Without a key the API serves the mock's made-up spots, except in
	// production, where a missing key stops the boot: a real deployment never lists fake spots.
	placesKey := os.Getenv("PLACES_API_KEY")
	placesMock := os.Getenv("PLACES_MOCK")
	switch {
	case placesKey != "":
		// The real client is still to come (#32), and serving the mock anyway would ignore the key
		// without a word.
		log.Error("PLACES_API_KEY is set, but there is no places client yet; unset it to serve the mock")
		os.Exit(1)
	case appEnv == "production":
		log.Error("PLACES_API_KEY is required in production")
		os.Exit(1)
	}
	spots, err := places.NewMock(placesMock)
	if err != nil {
		log.Error("PLACES_MOCK names no mock mode", "err", err)
		os.Exit(1)
	}
	log.Info("places mock is live", "mode", cmp.Or(placesMock, "default"))

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Error("open database pool", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	// One repository over the one pool, shared by every service. Migrations never run here:
	// `moon run db:migrate` applies them.
	repo := repository.New(pool)

	// Sessions live in Postgres, and the cookie is the device account's only credential, so it
	// lasts a year. It is Secure only in production: local dev serves plain http to the simulator
	// and to phones on the LAN, which would never send a Secure cookie back.
	sessionStore := pgxstore.New(pool)
	defer sessionStore.StopCleanup()
	sessions := scs.New()
	sessions.Store = sessionStore
	sessions.Lifetime = auth.Lifetime
	sessions.Cookie.Secure = appEnv == "production"
	sessions.Cookie.SameSite = http.SameSiteLaxMode
	authn := auth.New(sessions, repo)

	accountSvc := account.NewAccount(account.Config{
		Repo:   repo,
		Auth:   authn,
		Limits: account.DefaultLimits(),
		Log:    log,
	})
	friesSvc := fries.NewFries(fries.Config{
		Repo:   repo,
		Auth:   authn,
		Spots:  spots,
		Limits: fries.DefaultLimits(),
		Log:    log,
	})

	// Every service reads request bodies under the same cap.
	readCap := connect.WithReadMaxBytes(4 << 20)

	r := chi.NewRouter()
	r.Use(logging.RequestLogger(log))
	r.Use(logging.Recoverer(log))
	r.Use(sessions.LoadAndSave)

	r.Mount(healthv1connect.NewHealthServiceHandler(healthz.NewHealth(log, repo.Ping), readCap))
	r.Mount(accountv1connect.NewAccountServiceHandler(accountSvc, readCap))
	r.Mount(friesv1connect.NewFriesServiceHandler(friesSvc, readCap))

	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		// Three scan photos over a slow phone connection still fit.
		ReadTimeout: 60 * time.Second,
		// Longer than any vendor call, which has its own shorter deadline (go-vendors.md).
		WriteTimeout: 90 * time.Second,
		IdleTimeout:  2 * time.Minute,
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
