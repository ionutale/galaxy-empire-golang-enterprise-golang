package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	service *QuestService
}

func NewHandler(service *QuestService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListQuests(w http.ResponseWriter, r *http.Request) {
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

	quests, err := h.service.ListQuests(r.Context(), userID)
	if err != nil {
		slog.Error("list quests failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, ListQuestsResponse{Quests: quests})
}

func (h *Handler) ClaimReward(w http.ResponseWriter, r *http.Request) {
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

	questID := chi.URLParam(r, "id")
	if questID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing quest id"})
		return
	}

	def, err := h.service.ClaimReward(r.Context(), userID, questID)
	if err != nil {
		slog.Error("claim reward failed", "quest_id", questID, "error", err)
		code := http.StatusInternalServerError
		msg := "internal error"
		if strings.HasPrefix(err.Error(), "quest") || strings.Contains(err.Error(), "not completed") || strings.Contains(err.Error(), "already claimed") {
			code = http.StatusBadRequest
			msg = err.Error()
		}
		writeJSON(w, code, map[string]string{"error": msg})
		return
	}

	writeJSON(w, http.StatusOK, ClaimRewardResponse{
		QuestID:     questID,
		RewardDM:    def.RewardDM,
		RewardMetal: def.RewardMetal,
		RewardCrystal: def.RewardCrystal,
		RewardGas:   def.RewardGas,
	})
}

func (h *Handler) ProgressUpdate(w http.ResponseWriter, r *http.Request) {
	var req ProgressUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if err := h.service.CheckAndUpdateProgress(r.Context(), req.PlayerID, req.EventType, req.EventData); err != nil {
		slog.Error("progress update failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) CompletedQuests(w http.ResponseWriter, r *http.Request) {
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

	ids, err := h.service.GetCompletedQuests(r.Context(), userID)
	if err != nil {
		slog.Error("get completed quests failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, CompletedQuestsResponse{QuestIDs: ids})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
