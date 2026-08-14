package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jiachwen99/sf-assignment/api/internal/api"
	"github.com/jiachwen99/sf-assignment/api/internal/events"
	"github.com/jiachwen99/sf-assignment/api/internal/service"
	"github.com/jiachwen99/sf-assignment/api/internal/store"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(log); err != nil {
		log.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	ctx := context.Background()

	addr := env("ADDR", ":8080")
	dsn := env("DATABASE_URL", "postgres://todo:todo@localhost:5432/todo?sslmode=disable")

	st, err := store.New(ctx, dsn)
	if err != nil {
		return err
	}
	defer st.Close()

	// Applied at startup, so `docker compose up` is the only command needed.
	if err := st.Migrate(ctx); err != nil {
		return err
	}
	log.Info("migrations applied")

	hub := events.NewHub()

	srv := &http.Server{
		Addr:              addr,
		Handler:           api.NewServer(service.New(st, hub), hub, log),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errc := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", addr)
		errc <- srv.ListenAndServe()
	}()

	// Compose stops the container with SIGTERM, and an in-flight request
	// should finish rather than being cut off.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errc:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-stop:
		shutdown, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdown)
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
