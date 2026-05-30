package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const (
	ctxKeyUserID contextKey = "user_id"
	ctxKeyEmail  contextKey = "email"
)

type Claims struct {
	UserID int    `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

func main() {
	planetAddr := os.Getenv("PLANET_SERVICE_ADDR")
	if planetAddr == "" {
		planetAddr = "http://localhost:8082"
	}
	authAddr := os.Getenv("AUTH_SERVICE_ADDR")
	if authAddr == "" {
		authAddr = "http://localhost:8081"
	}
	jwtKey := []byte(getEnv("JWT_SECRET", "dev-secret-change-in-production"))

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"gateway"}`))
	})

	r.Route("/api", func(r chi.Router) {
		r.Post("/auth/register", proxyToService(authAddr))
		r.Post("/auth/login", proxyToService(authAddr))

		r.Group(func(r chi.Router) {
			r.Use(jwtMiddleware(jwtKey))
			r.Get("/auth/me", proxyToService(authAddr))
			r.Get("/planet/{id}", proxyToService(planetAddr))
		})
	})

	srv := &http.Server{Addr: ":8080", Handler: r}
	go func() {
		slog.Info("gateway starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("gateway fatal", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("gateway shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}

func jwtMiddleware(jwtKey []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := extractBearerToken(r)
			if tokenStr == "" {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing authorization header"})
				return
			}

			claims := &Claims{}
			token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (any, error) {
				return jwtKey, nil
			})
			if err != nil || !token.Valid {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid or expired token"})
				return
			}

			ctx := context.WithValue(r.Context(), ctxKeyUserID, claims.UserID)
			ctx = context.WithValue(ctx, ctxKeyEmail, claims.Email)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(auth, "Bearer ")
}

func proxyToService(serviceAddr string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		targetURL := fmt.Sprintf("%s%s", serviceAddr, r.URL.Path)
		if r.URL.RawQuery != "" {
			targetURL += "?" + r.URL.RawQuery
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, r.Method, targetURL, r.Body)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		for k, v := range r.Header {
			req.Header[k] = v
		}

		if userID, ok := r.Context().Value(ctxKeyUserID).(int); ok {
			req.Header.Set("X-User-ID", strconv.Itoa(userID))
		}
		if email, ok := r.Context().Value(ctxKeyEmail).(string); ok {
			req.Header.Set("X-User-Email", email)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			slog.Error("proxy failed", "service", serviceAddr, "error", err)
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "service unavailable"})
			return
		}
		defer resp.Body.Close()

		for k, v := range resp.Header {
			w.Header()[k] = v
		}
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
