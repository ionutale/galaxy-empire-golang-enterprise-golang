package main

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	service *AdminService
}

func NewHandler(service *AdminService) *Handler {
	return &Handler{service: service}
}

// internalSecretMiddleware rejects any request that does not carry the correct
// X-Internal-Secret header (#21). This prevents direct port access from
// bypassing the gateway's JWT validation — the gateway sets this header on
// every forwarded request, and the value is shared via ADMIN_INTERNAL_SECRET.
func internalSecretMiddleware(next http.Handler) http.Handler {
	secret := os.Getenv("ADMIN_INTERNAL_SECRET")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if secret == "" {
			// Secret is not configured — fail closed to avoid silent bypass.
			slog.Error("ADMIN_INTERNAL_SECRET is not set; rejecting all admin requests")
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "service misconfigured"})
			return
		}
		if r.Header.Get("X-Internal-Secret") != secret {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) adminOnly(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userIDStr := r.Header.Get("X-User-ID")
		if userIDStr == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		userID, err := strconv.Atoi(userIDStr)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid user"})
			return
		}
		if err := h.service.RequireAdmin(r.Context(), userID); err != nil {
			if errors.Is(err, ErrNotAdmin) {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin access required"})
				return
			}
			slog.Error("admin check failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}
		next(w, r)
	}
}

func (h *Handler) SearchUsers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	users, total, err := h.service.SearchUsers(r.Context(), q, page, limit)
	if err != nil {
		slog.Error("search users failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	// Strip sensitive fields (email, tokens) before sending to the caller.
	sanitized := make([]UserSearchResponse, len(users))
	for i, u := range users {
		sanitized[i] = UserSearchResponse{
			ID:        u.ID,
			Username:  u.Username,
			IsBanned:  u.IsBanned,
			CreatedAt: u.CreatedAt,
		}
	}

	totalPages := (total + limit - 1) / limit
	writeJSON(w, http.StatusOK, map[string]any{
		"users":       sanitized,
		"total":       total,
		"page":        page,
		"total_pages": totalPages,
	})
}

func (h *Handler) GetPlanets(w http.ResponseWriter, r *http.Request) {
	playerIDStr := r.URL.Query().Get("player_id")
	if playerIDStr == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "player_id is required"})
		return
	}
	playerID, err := strconv.Atoi(playerIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid player_id"})
		return
	}

	planets, err := h.service.GetPlanetsByUser(r.Context(), playerID)
	if err != nil {
		slog.Error("get planets failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"player_id": playerID,
		"planets":   planets,
	})
}

func (h *Handler) OverrideResources(w http.ResponseWriter, r *http.Request) {
	planetIDStr := chi.URLParam(r, "id")
	planetID, err := strconv.Atoi(planetIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid planet id"})
		return
	}

	var req ResourceOverrideRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Metal < 0 || req.Crystal < 0 || req.Gas < 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "resources cannot be negative"})
		return
	}

	if err := h.service.OverrideResources(r.Context(), planetID, req.Metal, req.Crystal, req.Gas); err != nil {
		slog.Error("override resources failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) GrantDM(w http.ResponseWriter, r *http.Request) {
	playerIDStr := chi.URLParam(r, "id")
	playerID, err := strconv.Atoi(playerIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid player id"})
		return
	}

	var req DMGrantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if err := h.service.GrantDM(r.Context(), playerID, req.Amount, req.Reason); err != nil {
		slog.Error("grant DM failed", "error", err)
		code := http.StatusInternalServerError
		msg := "internal error"
		if err.Error() == "reason is required" {
			code = http.StatusBadRequest
			msg = err.Error()
		}
		writeJSON(w, code, map[string]string{"error": msg})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) GrantCredits(w http.ResponseWriter, r *http.Request) {
	playerIDStr := chi.URLParam(r, "id")
	playerID, err := strconv.Atoi(playerIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid player id"})
		return
	}

	var req CreditsGrantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if err := h.service.GrantCredits(r.Context(), playerID, req.Amount, req.Reason); err != nil {
		slog.Error("grant credits failed", "error", err)
		code := http.StatusInternalServerError
		msg := "internal error"
		if err.Error() == "reason is required" {
			code = http.StatusBadRequest
			msg = err.Error()
		}
		writeJSON(w, code, map[string]string{"error": msg})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) BanPlayer(w http.ResponseWriter, r *http.Request) {
	playerIDStr := chi.URLParam(r, "id")
	playerID, err := strconv.Atoi(playerIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid player id"})
		return
	}

	var req BanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if err := h.service.BanPlayer(r.Context(), playerID, req.Banned, req.Reason); err != nil {
		slog.Error("ban player failed", "error", err)
		code := http.StatusInternalServerError
		msg := "internal error"
		if err.Error() == "reason is required" {
			code = http.StatusBadRequest
			msg = err.Error()
		}
		writeJSON(w, code, map[string]string{"error": msg})
		return
	}

	action := "banned"
	if !req.Banned {
		action = "unbanned"
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": action})
}

func (h *Handler) SendGMMessage(w http.ResponseWriter, r *http.Request) {
	var req GMMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if err := h.service.SendGMMessage(r.Context(), req.PlayerID, req.Subject, req.Message); err != nil {
		slog.Error("send GM message failed", "error", err)
		code := http.StatusInternalServerError
		msg := "internal error"
		if err.Error() == "subject and message are required" {
			code = http.StatusBadRequest
			msg = err.Error()
		}
		writeJSON(w, code, map[string]string{"error": msg})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	var req EventCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if err := h.service.CreateEvent(r.Context(), req); err != nil {
		slog.Error("create event failed", "error", err)
		code := http.StatusInternalServerError
		msg := "internal error"
		switch {
		case err.Error() == "name and event_type are required",
			err.Error() == "starts_at and ends_at are required",
			err.Error() == "ends_at must be after starts_at",
			errors.Is(err, ErrInvalidEventType):
			code = http.StatusBadRequest
			msg = err.Error()
		}
		writeJSON(w, code, map[string]string{"error": msg})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
