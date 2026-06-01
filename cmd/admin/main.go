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
	svc := NewAdminService(repo)
	h := NewHandler(svc)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"admin"}`))
	})

	r.Route("/api/admin", func(r chi.Router) {
		r.Get("/users", h.adminOnly(h.SearchUsers))
		r.Get("/planets", h.adminOnly(h.GetPlanets))
		r.Post("/planet/{id}/resources", h.adminOnly(h.OverrideResources))
		r.Post("/player/{id}/dm", h.adminOnly(h.GrantDM))
		r.Post("/player/{id}/credits", h.adminOnly(h.GrantCredits))
		r.Post("/player/{id}/ban", h.adminOnly(h.BanPlayer))
		r.Post("/gm-message", h.adminOnly(h.SendGMMessage))
		r.Post("/event/create", h.adminOnly(h.CreateEvent))
	})

	srv := &http.Server{Addr: ":8096", Handler: r}
	go func() {
		slog.Info("admin service starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("admin service fatal", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("admin service shutting down")
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
		CREATE SCHEMA IF NOT EXISTS admin;
		CREATE TABLE IF NOT EXISTS admin.admins (
			player_id INT PRIMARY KEY,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		INSERT INTO admin.admins (player_id) VALUES (1) ON CONFLICT DO NOTHING;
	`); err != nil {
		return err
	}

	if _, err := pool.Exec(ctx, `
		ALTER TABLE auth.users
		ADD COLUMN IF NOT EXISTS banned BOOLEAN NOT NULL DEFAULT FALSE
	`); err != nil {
		return err
	}

	return nil
}


