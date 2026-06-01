package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	databaseURL := getEnv("DATABASE_URL", "postgres://galaxy:galaxy_dev@localhost:5432/galaxy_empire?sslmode=disable")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		slog.Error("connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := runMigrations(ctx, pool); err != nil {
		slog.Error("run migrations", "error", err)
		os.Exit(1)
	}

	repo := NewPostgresRepository(pool)
	svc := NewEventService(repo)
	h := NewHandler(svc)

	eventCtx, eventCancel := context.WithCancel(context.Background())
	defer eventCancel()
	go svc.StartBackgroundTicker(eventCtx)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"event"}`))
	})
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"event"}`))
	})
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		pingCtx, pingCancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer pingCancel()
		if err := pool.Ping(pingCtx); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"status":"unavailable","error":err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"event"}`))
	})

	r.Get("/api/event/active", h.GetActiveEvents)
	r.Get("/api/event/all", h.GetAllEvents)
	r.Post("/api/event/{id}/join", h.JoinEvent)
	r.Post("/api/event/{id}/claim", h.ClaimRewards)
	r.Post("/internal/event/check", h.InternalCheck)
	r.Post("/internal/event/create", h.InternalCreate)

	srv := &http.Server{Addr: ":8095", Handler: r}
	go func() {
		slog.Info("event service starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("event service fatal", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("event service shutting down")
	shutdownTimeout := getEnvDuration("SHUTDOWN_TIMEOUT", 15*time.Second)
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()
	srv.Shutdown(shutdownCtx)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		d, err := time.ParseDuration(v)
		if err == nil {
			return d
		}
	}
	return fallback
}

func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, `
		CREATE SCHEMA IF NOT EXISTS event;
		CREATE TABLE IF NOT EXISTS event.events (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			description TEXT,
			event_type VARCHAR(50) NOT NULL,
			modifiers JSONB NOT NULL DEFAULT '{}',
			starts_at TIMESTAMPTZ NOT NULL,
			ends_at TIMESTAMPTZ NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'upcoming',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE TABLE IF NOT EXISTS event.event_participation (
			id SERIAL PRIMARY KEY,
			player_id INT NOT NULL,
			event_id INT NOT NULL REFERENCES event.events(id),
			progress JSONB NOT NULL DEFAULT '{}',
			completed BOOLEAN NOT NULL DEFAULT FALSE,
			rewards_claimed BOOLEAN NOT NULL DEFAULT FALSE,
			joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE(player_id, event_id)
		);
	`); err != nil {
		return err
	}
	return nil
}
