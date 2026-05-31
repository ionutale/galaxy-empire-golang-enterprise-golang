package main

import (
	"context"
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

	planetBaseURL := getEnv("PLANET_SERVICE_URL", "http://localhost:8082")

	repo := NewPostgresRepository(pool)
	svc := NewFleetService(repo, planetBaseURL)
	h := NewFleetHandler(svc)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"fleet"}`))
	})

	r.Get("/api/fleet/my-fleets", h.MyFleets)
	r.Post("/api/fleet/dispatch", h.Dispatch)

	srv := &http.Server{Addr: ":8083", Handler: r}
	go func() {
		slog.Info("fleet service starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("fleet service fatal", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("fleet service shutting down")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	srv.Shutdown(shutdownCtx)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, `
		CREATE SCHEMA IF NOT EXISTS fleet;
		CREATE TABLE IF NOT EXISTS fleet.fleets (
			id SERIAL PRIMARY KEY,
			player_id INT NOT NULL,
			origin_planet_id INT NOT NULL,
			target_galaxy INT NOT NULL,
			target_system INT NOT NULL,
			target_position INT NOT NULL,
			mission VARCHAR(20) NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'stationed',
			speed_pct INT NOT NULL DEFAULT 100,
			ships JSONB NOT NULL DEFAULT '{}',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`	); err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, `
		ALTER TABLE fleet.fleets ADD COLUMN IF NOT EXISTS arrives_at TIMESTAMPTZ;
	`); err != nil {
		return err
	}
	return nil
}
