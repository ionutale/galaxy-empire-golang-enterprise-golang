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
	svc := NewPlanetService(repo)
	h := NewHandler(svc)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"planet"}`))
	})

	r.Get("/api/planet/mine", h.GetMyPlanet)
	r.Post("/api/buildings/{type}/upgrade", h.StartUpgrade)
	r.Post("/api/buildings/{type}/cancel", h.CancelUpgrade)
	r.Post("/api/buildings/{type}/deconstruct", h.DeconstructBuilding)

	srv := &http.Server{Addr: ":8082", Handler: r}
	go func() {
		slog.Info("planet service starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("planet service fatal", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("planet service shutting down")
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
		CREATE SCHEMA IF NOT EXISTS planet;
		CREATE TABLE IF NOT EXISTS planet.planets (
			id SERIAL PRIMARY KEY,
			user_id INTEGER UNIQUE NOT NULL,
			name VARCHAR(100) NOT NULL DEFAULT 'Homeworld',
			galaxy INTEGER NOT NULL DEFAULT 1,
			system INTEGER NOT NULL DEFAULT 1,
			position INTEGER NOT NULL DEFAULT 7,
			metal INTEGER NOT NULL DEFAULT 500,
			crystal INTEGER NOT NULL DEFAULT 300,
			gas INTEGER NOT NULL DEFAULT 200,
			energy INTEGER NOT NULL DEFAULT 50,
			max_fields INTEGER NOT NULL DEFAULT 40,
			resources_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`); err != nil {
		return err
	}

	if _, err := pool.Exec(ctx, `
		ALTER TABLE planet.planets
		ADD COLUMN IF NOT EXISTS resources_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
	`); err != nil {
		return err
	}

	if _, err := pool.Exec(ctx, `
		ALTER TABLE planet.planets
		ADD COLUMN IF NOT EXISTS max_fields INTEGER NOT NULL DEFAULT 40;
	`); err != nil {
		return err
	}

	if _, err := pool.Exec(ctx, `
		ALTER TABLE planet.planets
		ADD COLUMN IF NOT EXISTS type VARCHAR(20) NOT NULL DEFAULT 'terran';
	`); err != nil {
		return err
	}

	if _, err := pool.Exec(ctx, `
		ALTER TABLE planet.planets
		ADD COLUMN IF NOT EXISTS temperature INTEGER NOT NULL DEFAULT 20;
	`); err != nil {
		return err
	}

	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS planet.player_progress (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES planet.planets(user_id) ON DELETE CASCADE UNIQUE,
			vip_points INTEGER NOT NULL DEFAULT 0,
			total_resources_produced BIGINT NOT NULL DEFAULT 0
		);
	`); err != nil {
		return err
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO planet.player_progress (user_id, vip_points, total_resources_produced)
		SELECT p.user_id, 0, 0
		FROM planet.planets p
		WHERE NOT EXISTS (
			SELECT 1 FROM planet.player_progress pp
			WHERE pp.user_id = p.user_id
		);
	`); err != nil {
		return err
	}

	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS planet.buildings (
			id SERIAL PRIMARY KEY,
			planet_id INTEGER NOT NULL REFERENCES planet.planets(id) ON DELETE CASCADE,
			type VARCHAR(50) NOT NULL,
			level INTEGER NOT NULL DEFAULT 0,
			UNIQUE(planet_id, type)
		);
	`); err != nil {
		return err
	}

	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS planet.construction_queue (
			id SERIAL PRIMARY KEY,
			planet_id INTEGER NOT NULL REFERENCES planet.planets(id) ON DELETE CASCADE,
			building_type VARCHAR(50) NOT NULL,
			target_level INTEGER NOT NULL,
			started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			completes_at TIMESTAMPTZ NOT NULL,
			completed BOOLEAN NOT NULL DEFAULT FALSE
		);
	`); err != nil {
		return err
	}

	if _, err := pool.Exec(ctx, `
		ALTER TABLE planet.construction_queue
		ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'upgrade';
	`); err != nil {
		return err
	}

	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS planet.player_technologies (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES planet.planets(user_id) ON DELETE CASCADE,
			type VARCHAR(50) NOT NULL,
			level INTEGER NOT NULL DEFAULT 0,
			UNIQUE(user_id, type)
		);
	`); err != nil {
		return err
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO planet.player_technologies (user_id, type, level)
		SELECT p.user_id, 'energy_tech', 3
		FROM planet.planets p
		WHERE NOT EXISTS (
			SELECT 1 FROM planet.player_technologies t
			WHERE t.user_id = p.user_id AND t.type = 'energy_tech'
		);
	`); err != nil {
		return err
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO planet.buildings (planet_id, type, level)
		SELECT p.id, btype, 1
		FROM planet.planets p
		CROSS JOIN (VALUES ('robotics_factory'), ('nanite_factory'), ('terraformer'), ('fusion_reactor')) AS t(btype)
		WHERE NOT EXISTS (
			SELECT 1 FROM planet.buildings b
			WHERE b.planet_id = p.id AND b.type = t.btype
		);
	`); err != nil {
		return err
	}

	return nil
}
