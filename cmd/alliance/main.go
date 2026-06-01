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
	svc := NewAllianceService(repo, planetBaseURL)
	h := NewHandler(svc)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"alliance"}`))
	})
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"alliance"}`))
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
		w.Write([]byte(`{"status":"ok","service":"alliance"}`))
	})

	r.Post("/api/alliance/create", h.CreateAlliance)
	r.Post("/api/alliance/apply", h.ApplyToAlliance)
	r.Post("/api/alliance/leave", h.LeaveAlliance)
	r.Post("/api/alliance/transfer", h.TransferFounder)
	r.Get("/api/alliance/my", h.GetMyAlliance)
	r.Post("/api/alliance/bank/deposit", h.BankDeposit)
	r.Post("/api/alliance/bank/withdraw", h.BankWithdraw)
	r.Get("/api/alliance/bank", h.GetBank)

	r.Post("/api/alliance/bulletin", h.PostBulletin)
	r.Get("/api/alliance/bulletins", h.GetBulletins)
	r.Delete("/api/alliance/bulletins/{id}", h.DeleteBulletin)
	r.Post("/api/alliance/share-report", h.ShareReport)
	r.Get("/api/alliance/shared-reports", h.GetSharedReports)
	r.Post("/api/alliance/unshare-report", h.UnshareReport)

	internalSecret := getEnv("INTERNAL_SECRET", "internal-secret")
	r.With(internalSecretMiddleware(internalSecret)).Post("/internal/alliance/player", h.InternalGetPlayerAlliance)
	r.With(internalSecretMiddleware(internalSecret)).Post("/internal/alliance/ping", h.InternalPing)

	srv := &http.Server{Addr: ":8087", Handler: r}
	go func() {
		slog.Info("alliance service starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("alliance service fatal", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("alliance service shutting down")
	shutdownTimeout := getEnvDuration("SHUTDOWN_TIMEOUT", 15*time.Second)
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()
	srv.Shutdown(shutdownCtx)
}

func internalSecretMiddleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-Internal-Secret") != secret {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
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
		CREATE SCHEMA IF NOT EXISTS alliance;

		CREATE TABLE IF NOT EXISTS alliance.alliances (
			id SERIAL PRIMARY KEY,
			name VARCHAR(50) NOT NULL UNIQUE,
			tag VARCHAR(10) NOT NULL UNIQUE,
			founder_id INT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS alliance.members (
			id SERIAL PRIMARY KEY,
			alliance_id INT NOT NULL REFERENCES alliance.alliances(id) ON DELETE CASCADE,
			player_id INT NOT NULL UNIQUE,
			role VARCHAR(20) NOT NULL DEFAULT 'member',
			joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE(alliance_id, player_id)
		);

		CREATE TABLE IF NOT EXISTS alliance.bank (
			id SERIAL PRIMARY KEY,
			alliance_id INT NOT NULL REFERENCES alliance.alliances(id) ON DELETE CASCADE UNIQUE,
			metal BIGINT NOT NULL DEFAULT 0,
			crystal BIGINT NOT NULL DEFAULT 0,
			gas BIGINT NOT NULL DEFAULT 0
		);

		CREATE TABLE IF NOT EXISTS alliance.audit_log (
			id SERIAL PRIMARY KEY,
			alliance_id INT NOT NULL REFERENCES alliance.alliances(id) ON DELETE CASCADE,
			player_id INT NOT NULL,
			action VARCHAR(100) NOT NULL,
			details JSONB,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		ALTER TABLE alliance.members ADD COLUMN IF NOT EXISTS last_active_at TIMESTAMPTZ;

		CREATE TABLE IF NOT EXISTS alliance.bulletins (
			id SERIAL PRIMARY KEY,
			alliance_id INT NOT NULL REFERENCES alliance.alliances(id) ON DELETE CASCADE,
			author_player_id INT NOT NULL,
			title TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS alliance.shared_reports (
			id SERIAL PRIMARY KEY,
			alliance_id INT NOT NULL REFERENCES alliance.alliances(id) ON DELETE CASCADE,
			report_id INT NOT NULL,
			shared_by INT NOT NULL,
			shared_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`); err != nil {
		return err
	}
	return nil
}
