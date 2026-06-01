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
	svc := NewNebulaService(repo, planetBaseURL)
	h := NewHandler(svc)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"nebula"}`))
	})
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"nebula"}`))
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
		w.Write([]byte(`{"status":"ok","service":"nebula"}`))
	})

	r.Post("/api/nebula/start", h.StartExpedition)
	r.Get("/api/nebula/expeditions", h.ListExpeditions)
	r.Get("/api/nebula/expeditions/{id}", h.GetExpedition)
	r.Post("/api/nebula/dm-balance", h.DMBalance)
	r.Post("/api/nebula/dm-spend", h.DMSpend)
	r.Post("/api/nebula/dm/speed-up", h.DMSpeedUp)
	r.Post("/api/nebula/dm/estimate-cost", h.DMEstimateCost)
	r.Get("/api/nebula/dm/transactions", h.DMTransactions)
	r.Post("/api/nebula/commanders/hire", h.HireCommander)
	r.Get("/api/nebula/commanders", h.ListCommanders)
	r.Get("/api/nebula/commanders/available", h.AvailableCommanders)
	r.Post("/api/nebula/credits-balance", h.CreditsBalance)
	r.Post("/api/nebula/credits-spend", h.CreditsSpend)
	r.Get("/api/nebula/credits-transactions", h.CreditsTransactions)
	r.Post("/internal/nebula/credits/add", h.InternalAddCredits)
	r.Post("/internal/nebula/commanders/active", h.InternalActiveCommanders)

	r.Post("/api/nebula/daily-gift/claim", h.ClaimDailyGift)
	r.Get("/api/nebula/daily-gift/status", h.GetDailyGiftStatus)
	r.Get("/api/nebula/daily-tasks", h.GetDailyTasks)
	r.Post("/api/nebula/daily-tasks/{id}/progress", h.UpdateTaskProgress)
	r.Post("/api/nebula/daily-tasks/{id}/claim", h.ClaimTaskReward)
	r.Post("/api/nebula/daily-tasks/reroll", h.RerollTask)
	r.Post("/api/nebula/daily-tasks/claim-all", h.ClaimAllTasks)

	r.Get("/api/nebula/discoverer", h.GetGalactoniteDiscoverer)
	r.Post("/api/nebula/discoverer/upgrade", h.UpgradeGalactoniteDiscoverer)

	srv := &http.Server{Addr: ":8088", Handler: r}
	go func() {
		slog.Info("nebula service starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("nebula service fatal", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("nebula service shutting down")
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
		CREATE SCHEMA IF NOT EXISTS nebula;
		CREATE TABLE IF NOT EXISTS nebula.expeditions (
			id SERIAL PRIMARY KEY,
			player_id INT NOT NULL,
			fleet_id INT NOT NULL DEFAULT 0,
			galaxy INT NOT NULL DEFAULT 0,
			system INT NOT NULL DEFAULT 0,
			position INT NOT NULL DEFAULT 0,
			status VARCHAR(20) NOT NULL DEFAULT 'travelling',
			ships_sent JSONB NOT NULL DEFAULT '{}',
			ships_lost JSONB NOT NULL DEFAULT '{}',
			ships_found JSONB NOT NULL DEFAULT '{}',
			resources_found JSONB NOT NULL DEFAULT '{}',
			dark_matter_found INT NOT NULL DEFAULT 0,
			outcome VARCHAR(50),
			started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			travel_duration INT NOT NULL DEFAULT 300,
			explore_duration INT NOT NULL DEFAULT 1800,
			completed_at TIMESTAMPTZ
		);
		CREATE TABLE IF NOT EXISTS nebula.player_dark_matter (
			player_id INT PRIMARY KEY,
			balance INT NOT NULL DEFAULT 0,
			total_earned INT NOT NULL DEFAULT 0
		);
		CREATE TABLE IF NOT EXISTS nebula.dm_transactions (
			id SERIAL PRIMARY KEY,
			player_id INT NOT NULL,
			amount INT NOT NULL,
			balance_after INT NOT NULL,
			reason VARCHAR(100) NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE TABLE IF NOT EXISTS nebula.player_credits (
			player_id INT PRIMARY KEY,
			balance INT NOT NULL DEFAULT 0,
			total_earned INT NOT NULL DEFAULT 0
		);
		CREATE TABLE IF NOT EXISTS nebula.credits_transactions (
			id SERIAL PRIMARY KEY,
			player_id INT NOT NULL,
			amount INT NOT NULL,
			balance_after INT NOT NULL,
			reason VARCHAR(100) NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE TABLE IF NOT EXISTS nebula.player_commanders (
			id SERIAL PRIMARY KEY,
			player_id INT NOT NULL,
			commander_type VARCHAR(20) NOT NULL,
			level INT NOT NULL DEFAULT 1,
			hired_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			expires_at TIMESTAMPTZ NOT NULL,
			UNIQUE(player_id, commander_type)
		);
		CREATE TABLE IF NOT EXISTS nebula.daily_gift_streak (
			player_id INT PRIMARY KEY,
			streak_day INT NOT NULL DEFAULT 0,
			last_claim_date DATE NOT NULL DEFAULT CURRENT_DATE,
			consecutive_days INT NOT NULL DEFAULT 0
		);
		CREATE TABLE IF NOT EXISTS nebula.store_purchases (
			id SERIAL PRIMARY KEY,
			player_id INT NOT NULL,
			item_id VARCHAR(50) NOT NULL,
			cost INT NOT NULL,
			currency VARCHAR(10) NOT NULL,
			purchased_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE TABLE IF NOT EXISTS nebula.player_shifter_uses (
			player_id INT PRIMARY KEY,
			uses INT NOT NULL DEFAULT 0
		);
		CREATE TABLE IF NOT EXISTS nebula.daily_tasks (
			id SERIAL PRIMARY KEY,
			player_id INT NOT NULL,
			task_type VARCHAR(50) NOT NULL,
			description TEXT NOT NULL,
			target_amount INT NOT NULL,
			progress INT NOT NULL DEFAULT 0,
			reward_dm INT NOT NULL DEFAULT 0,
			reward_resources JSONB NOT NULL DEFAULT '{}',
			completed BOOLEAN NOT NULL DEFAULT FALSE,
			claimed BOOLEAN NOT NULL DEFAULT FALSE,
			assigned_date DATE NOT NULL DEFAULT CURRENT_DATE,
			rerolls_used INT NOT NULL DEFAULT 0,
			UNIQUE(player_id, task_type, assigned_date)
		);
		CREATE TABLE IF NOT EXISTS nebula.galactonite_discoverer (
			player_id INT PRIMARY KEY,
			level INT NOT NULL DEFAULT 0
		);
	`); err != nil {
		return err
	}
	return nil
}
