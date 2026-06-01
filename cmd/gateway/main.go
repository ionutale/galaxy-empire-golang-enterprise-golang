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
	"sync"
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
	planetAddr := getEnv("PLANET_SERVICE_ADDR", "http://localhost:8082")
	authAddr := getEnv("AUTH_SERVICE_ADDR", "http://localhost:8081")
	espionageAddr := getEnv("ESPIONAGE_SERVICE_ADDR", "http://localhost:8086")
	researchAddr := getEnv("RESEARCH_SERVICE_ADDR", "http://localhost:8085")
	nebulaAddr := getEnv("NEBULA_SERVICE_ADDR", "http://localhost:8088")
	allianceAddr := getEnv("ALLIANCE_SERVICE_ADDR", "http://localhost:8087")
	chatAddr := getEnv("CHAT_SERVICE_ADDR", "http://localhost:8090")
	notificationAddr := getEnv("NOTIFICATION_SERVICE_ADDR", "http://localhost:8093")
	questAddr := getEnv("QUEST_SERVICE_ADDR", "http://localhost:8094")
	eventAddr := getEnv("EVENT_SERVICE_ADDR", "http://localhost:8095")
	radarAddr := getEnv("RADAR_SERVICE_ADDR", "http://localhost:8089")
	fleetAddr := getEnv("FLEET_SERVICE_ADDR", "http://localhost:8083")
	friendAddr := getEnv("FRIEND_SERVICE_ADDR", "http://localhost:8091")
	rankingAddr := getEnv("RANKING_SERVICE_ADDR", "http://localhost:8092")
	tutorialAddr := getEnv("TUTORIAL_SERVICE_ADDR", "http://localhost:8097")
	adminAddr := getEnv("ADMIN_SERVICE_ADDR", "http://localhost:8096")
	jwtKey := []byte(getEnv("JWT_SECRET", "dev-secret-change-in-production"))
	shutdownTimeout := getEnvDuration("SHUTDOWN_TIMEOUT", 15*time.Second)

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	rl := NewRateLimiter(100, 60*time.Second)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(rl.Middleware)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"gateway"}`))
	})
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"gateway"}`))
	})
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"gateway"}`))
	})

	r.Route("/api", func(r chi.Router) {
		r.Post("/auth/register", proxyToService(authAddr))
		r.Post("/auth/login", proxyToService(authAddr))

		r.Group(func(r chi.Router) {
			r.Use(jwtMiddleware(jwtKey))
			r.Get("/auth/me", proxyToService(authAddr))
			r.Post("/auth/vacation/enable", proxyToService(authAddr))
			r.Post("/auth/vacation/confirm", proxyToService(authAddr))
			r.Post("/auth/vacation/disable", proxyToService(authAddr))
			r.Get("/auth/vacation/status", proxyToService(authAddr))
			r.Get("/planet/mine", proxyToService(planetAddr))
			r.Post("/buildings/{type}/upgrade", proxyToService(planetAddr))
			r.Post("/espionage/probe", proxyToService(espionageAddr))
			r.Get("/espionage/reports", proxyToService(espionageAddr))
			r.Get("/espionage/reports/{id}", proxyToService(espionageAddr))
			r.Delete("/espionage/reports/{id}", proxyToService(espionageAddr))
			r.Get("/research", proxyToService(researchAddr))
			r.Post("/research/{type}/start", proxyToService(researchAddr))
			r.Post("/research/{type}/cancel", proxyToService(researchAddr))
			r.Get("/research/queue", proxyToService(researchAddr))
			r.Get("/moon/{galaxy}/{system}/{position}/buildings", proxyToService(planetAddr))
			r.Post("/moon/{galaxy}/{system}/{position}/buildings/{type}/upgrade", proxyToService(planetAddr))
			r.Post("/moon/{galaxy}/{system}/{position}/build-iron-behemoth", proxyToService(planetAddr))
			r.Post("/wormhole/link", proxyToService(planetAddr))
			r.Post("/planet/{id}/rename", proxyToService(planetAddr))
			r.Post("/planet/{id}/stargate/link", proxyToService(planetAddr))
			r.Post("/planet/{id}/stargate/unlink", proxyToService(planetAddr))
			r.Get("/planet/{id}/stargate/links", proxyToService(planetAddr))
			// Galaxy map, shipyard and defense (served by the planet service)
			r.Get("/galaxy", proxyToService(planetAddr))
			r.Get("/galaxy/systems/{galaxyID}", proxyToService(planetAddr))
			r.Get("/galaxy/positions/{systemID}", proxyToService(planetAddr))
			r.Get("/shipyard", proxyToService(planetAddr))
			r.Post("/shipyard/build", proxyToService(planetAddr))
			r.Get("/defense", proxyToService(planetAddr))
			r.Post("/defense/build", proxyToService(planetAddr))
			r.Post("/alliance/create", proxyToService(allianceAddr))
			r.Post("/alliance/apply", proxyToService(allianceAddr))
			r.Post("/alliance/leave", proxyToService(allianceAddr))
			r.Post("/alliance/transfer", proxyToService(allianceAddr))
			r.Get("/alliance/my", proxyToService(allianceAddr))
			r.Post("/alliance/bank/deposit", proxyToService(allianceAddr))
			r.Post("/alliance/bank/withdraw", proxyToService(allianceAddr))
			r.Get("/alliance/bank", proxyToService(allianceAddr))
			r.Post("/alliance/bulletin", proxyToService(allianceAddr))
			r.Get("/alliance/bulletins", proxyToService(allianceAddr))
			r.Delete("/alliance/bulletins/{id}", proxyToService(allianceAddr))
			r.Post("/alliance/share-report", proxyToService(allianceAddr))
			r.Get("/alliance/shared-reports", proxyToService(allianceAddr))
			r.Post("/alliance/unshare-report", proxyToService(allianceAddr))
			r.Post("/radar/scan", proxyToService(radarAddr))
			r.Post("/radar/events", proxyToService(radarAddr))
			r.Post("/radar/events/resolve", proxyToService(radarAddr))
			r.Post("/radar/planet-status", proxyToService(radarAddr))
			r.Post("/radar/eu-scan", proxyToService(radarAddr))
			r.Post("/nebula/dm/speed-up", proxyToService(nebulaAddr))
			r.Post("/nebula/dm/estimate-cost", proxyToService(nebulaAddr))
			r.Get("/nebula/dm/transactions", proxyToService(nebulaAddr))
			r.Post("/nebula/commanders/hire", proxyToService(nebulaAddr))
			r.Get("/nebula/commanders", proxyToService(nebulaAddr))
			r.Get("/nebula/commanders/available", proxyToService(nebulaAddr))
			r.Post("/nebula/credits-balance", proxyToService(nebulaAddr))
			r.Post("/nebula/credits-spend", proxyToService(nebulaAddr))
			r.Get("/nebula/credits-transactions", proxyToService(nebulaAddr))
			r.Post("/nebula/daily-gift/claim", proxyToService(nebulaAddr))
			r.Get("/nebula/daily-gift/status", proxyToService(nebulaAddr))
			r.Get("/nebula/daily-tasks", proxyToService(nebulaAddr))
			r.Post("/nebula/daily-tasks/{id}/progress", proxyToService(nebulaAddr))
			r.Post("/nebula/daily-tasks/{id}/claim", proxyToService(nebulaAddr))
			r.Post("/nebula/daily-tasks/reroll", proxyToService(nebulaAddr))
			r.Post("/nebula/daily-tasks/claim-all", proxyToService(nebulaAddr))
			r.Get("/nebula/store/items", proxyToService(nebulaAddr))
			r.Post("/nebula/store/buy/{itemId}", proxyToService(nebulaAddr))
			r.Get("/nebula/discoverer", proxyToService(nebulaAddr))
			r.Post("/nebula/discoverer/upgrade", proxyToService(nebulaAddr))
			r.Get("/planet/{id}/gems", proxyToService(planetAddr))
			r.Post("/planet/{id}/gems/equip", proxyToService(planetAddr))
			r.Post("/planet/{id}/gems/unequip", proxyToService(planetAddr))
			r.Post("/planet/{id}/gems/combine", proxyToService(planetAddr))
			r.Post("/chat/send", proxyToService(chatAddr))
			r.Get("/chat/messages", proxyToService(chatAddr))
			r.Post("/chat/private/send", proxyToService(chatAddr))
			r.Get("/chat/private/inbox", proxyToService(chatAddr))
			r.Get("/chat/private/outbox", proxyToService(chatAddr))
			r.Post("/chat/private/read", proxyToService(chatAddr))
			r.Delete("/chat/private/messages/{id}", proxyToService(chatAddr))
			r.Get("/chat/private/unread-count", proxyToService(chatAddr))
			r.Post("/friend/add", proxyToService(friendAddr))
			r.Post("/friend/accept", proxyToService(friendAddr))
			r.Get("/fleets", fleetProxy(fleetAddr))
			r.Post("/fleets/dispatch", fleetProxy(fleetAddr))
			r.Post("/fleet/merge", proxyToService(fleetAddr))
			r.Post("/fleet/{id}/recall", proxyToService(fleetAddr))
			r.Post("/fleet/{id}/split", proxyToService(fleetAddr))
			r.Post("/friend/remove", proxyToService(friendAddr))
			r.Get("/friend/list", proxyToService(friendAddr))
			r.Get("/ranking/top", proxyToService(rankingAddr))
			r.Get("/ranking/{playerId}", proxyToService(rankingAddr))
			r.Get("/notification/list", proxyToService(notificationAddr))
			r.Get("/notification/unread-count", proxyToService(notificationAddr))
			r.Post("/notification/{id}/read", proxyToService(notificationAddr))
			r.Post("/notification/read-all", proxyToService(notificationAddr))
			r.Post("/quest/list", proxyToService(questAddr))
			r.Post("/quest/{id}/claim", proxyToService(questAddr))
			r.Get("/quest/completed", proxyToService(questAddr))
			r.Get("/event/active", proxyToService(eventAddr))
			r.Get("/event/all", proxyToService(eventAddr))
			r.Post("/event/{id}/join", proxyToService(eventAddr))
			r.Post("/event/{id}/claim", proxyToService(eventAddr))
			r.Get("/admin/users", proxyToService(adminAddr))
			r.Get("/admin/planets", proxyToService(adminAddr))
			r.Post("/admin/planet/{id}/resources", proxyToService(adminAddr))
			r.Post("/admin/player/{id}/dm", proxyToService(adminAddr))
			r.Post("/admin/player/{id}/credits", proxyToService(adminAddr))
			r.Post("/admin/player/{id}/ban", proxyToService(adminAddr))
			r.Post("/admin/gm-message", proxyToService(adminAddr))
			r.Post("/admin/event/create", proxyToService(adminAddr))
			r.Get("/tutorial/status", proxyToService(tutorialAddr))
			r.Post("/tutorial/{step}/claim", proxyToService(tutorialAddr))
			r.Post("/tutorial/skip", proxyToService(tutorialAddr))
		})
		r.Get("/chat/stream", proxyToService(chatAddr))
		r.Get("/notification/stream", proxyToService(notificationAddr))
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
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
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

func fleetProxy(addr string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		switch p {
		case "/api/fleets":
			p = "/api/fleet/my-fleets"
		case "/api/fleets/dispatch":
			p = "/api/fleet/dispatch"
		}
		targetURL := fmt.Sprintf("%s%s", addr, p)
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
			slog.Error("proxy to fleet failed", "error", err)
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

type RateLimiter struct {
	mu       sync.Mutex
	requests map[string]*tokenBucket
	limit    int
	window   time.Duration
}

type tokenBucket struct {
	count    int
	windowStart time.Time
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string]*tokenBucket),
		limit:    limit,
		window:   window,
	}
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		key := ip + ":" + r.Method + ":" + r.URL.Path

		rl.mu.Lock()
		bucket, exists := rl.requests[key]
		now := time.Now()

		if !exists || now.Sub(bucket.windowStart) > rl.window {
			rl.requests[key] = &tokenBucket{count: 1, windowStart: now}
			rl.mu.Unlock()
			next.ServeHTTP(w, r)
			return
		}

		if bucket.count >= rl.limit {
			rl.mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":"rate limit exceeded","retry_after":60}`))
			return
		}

		bucket.count++
		rl.mu.Unlock()
		next.ServeHTTP(w, r)
	})
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

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		d, err := time.ParseDuration(v)
		if err == nil {
			return d
		}
	}
	return fallback
}
