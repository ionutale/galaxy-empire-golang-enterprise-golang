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
	planetBaseURL := getEnv("PLANET_SERVICE_URL", "http://localhost:8082")
	fleetBaseURL := getEnv("FLEET_SERVICE_URL", "http://localhost:8083")

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
	svc := NewRadarService(repo, planetBaseURL, fleetBaseURL)
	h := NewHandler(svc)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"radar"}`))
	})
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"radar"}`))
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
		w.Write([]byte(`{"status":"ok","service":"radar"}`))
	})

	r.Post("/api/radar/scan", h.Scan)
	r.Post("/api/radar/events", h.GetEvents)
	r.Post("/api/radar/events/resolve", h.ResolveEvent)
	r.Post("/api/radar/planet-status", h.PlanetStatus)
	r.Post("/api/radar/eu-scan", h.EUXScan)

	r.Post("/internal/radar/detect", h.InternalDetect)

	srv := &http.Server{Addr: ":8089", Handler: r}
	go func() {
		slog.Info("radar service starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("radar service fatal", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("radar service shutting down")
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
		CREATE SCHEMA IF NOT EXISTS radar;

		CREATE TABLE IF NOT EXISTS radar.radar_events (
			id SERIAL PRIMARY KEY,
			player_id INT NOT NULL,
			event_type VARCHAR(30) NOT NULL,
			source_player_id INT,
			fleet_id INT,
			target_galaxy INT NOT NULL,
			target_system INT NOT NULL,
			target_position INT NOT NULL,
			origin_galaxy INT,
			origin_system INT,
			origin_position INT,
			arrival_time TIMESTAMPTZ,
			detected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			resolved BOOLEAN NOT NULL DEFAULT FALSE
		);

		CREATE TABLE IF NOT EXISTS radar.eu_x_radars (
			id SERIAL PRIMARY KEY,
			player_id INT NOT NULL UNIQUE,
			moon_galaxy INT NOT NULL,
			moon_system INT NOT NULL,
			moon_position INT NOT NULL,
			level INT NOT NULL DEFAULT 1
		);
	`); err != nil {
		return err
	}
	return nil
}
