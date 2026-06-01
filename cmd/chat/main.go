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
	allianceBaseURL := getEnv("ALLIANCE_SERVICE_URL", "http://localhost:8087")
	jwtSecret := getEnv("JWT_SECRET", "dev-secret-change-in-production")

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
	svc := NewChatService(repo, allianceBaseURL, jwtSecret)
	h := NewHandler(svc)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"chat"}`))
	})
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"chat"}`))
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
		w.Write([]byte(`{"status":"ok","service":"chat"}`))
	})

	r.Post("/api/chat/send", h.Send)
	r.Get("/api/chat/messages", h.Messages)
	r.Get("/api/chat/stream", h.Stream)

	r.Post("/api/chat/private/send", h.SendPrivate)
	r.Get("/api/chat/private/inbox", h.Inbox)
	r.Get("/api/chat/private/outbox", h.Outbox)
	r.Post("/api/chat/private/read", h.MarkRead)
	r.Delete("/api/chat/private/messages/{id}", h.DeleteMessage)
	r.Get("/api/chat/private/unread-count", h.UnreadCount)

	srv := &http.Server{Addr: ":8090", Handler: r}
	go func() {
		slog.Info("chat service starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("chat service fatal", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("chat service shutting down")
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
		CREATE SCHEMA IF NOT EXISTS chat;

		CREATE TABLE IF NOT EXISTS chat.messages (
			id SERIAL PRIMARY KEY,
			channel VARCHAR(20) NOT NULL,
			channel_id INT NOT NULL DEFAULT 0,
			sender_id INT NOT NULL,
			sender_name VARCHAR(100) NOT NULL DEFAULT '',
			content TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_messages_channel ON chat.messages(channel, channel_id, created_at DESC);

		CREATE TABLE IF NOT EXISTS chat.private_messages (
			id SERIAL PRIMARY KEY,
			sender_id INT NOT NULL,
			receiver_id INT NOT NULL,
			content TEXT NOT NULL,
			is_read BOOLEAN NOT NULL DEFAULT FALSE,
			is_system BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_pm_receiver ON chat.private_messages(receiver_id, created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_pm_sender ON chat.private_messages(sender_id, created_at DESC);
	`); err != nil {
		return err
	}
	return nil
}
