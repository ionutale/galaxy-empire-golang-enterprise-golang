package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	service *RankingService
}

func NewHandler(service *RankingService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetTop(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 100
	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 {
			limit = v
		}
	}

	offset := 0
	if offsetStr != "" {
		if v, err := strconv.Atoi(offsetStr); err == nil && v >= 0 {
			offset = v
		}
	}

	scores, total, err := h.service.GetTop(r.Context(), limit, offset)
	if err != nil {
		slog.Error("get top failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	page := (offset / limit) + 1
	if page < 1 {
		page = 1
	}

	entries := make([]RankEntry, 0, len(scores))
	for i, s := range scores {
		rank := offset + i + 1
		entries = append(entries, RankEntry{
			Rank:           rank,
			PlayerID:       s.PlayerID,
			PlayerName:     s.PlayerName,
			TotalScore:     s.TotalScore,
			FleetScore:     s.FleetScore,
			BuildingsScore: s.BuildingsScore,
			ResearchScore:  s.ResearchScore,
			DefenseScore:   s.DefenseScore,
			UpdatedAt:      s.UpdatedAt,
		})
	}

	writeJSON(w, http.StatusOK, TopResponse{
		Page:    page,
		PerPage: limit,
		Total:   total,
		Ranking: entries,
	})
}

func (h *Handler) GetPlayer(w http.ResponseWriter, r *http.Request) {
	playerIDStr := chi.URLParam(r, "playerId")
	playerID, err := strconv.Atoi(playerIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid player id"})
		return
	}

	score, rank, err := h.service.GetByPlayerID(r.Context(), playerID)
	if err != nil {
		slog.Error("get player failed", "player_id", playerID, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	if score == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "player not found"})
		return
	}

	writeJSON(w, http.StatusOK, PlayerRankResponse{
		Rank:       rank,
		PlayerID:   score.PlayerID,
		PlayerName: score.PlayerName,
		PlayerScore: *score,
	})
}

func (h *Handler) UpdateScore(w http.ResponseWriter, r *http.Request) {
	var req UpdateScoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if req.PlayerID == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "player_id is required"})
		return
	}
	if req.PlayerName == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "player_name is required"})
		return
	}

	if err := h.service.UpdateScore(r.Context(), req); err != nil {
		slog.Error("update score failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) Recalculate(w http.ResponseWriter, r *http.Request) {
	var req RecalcRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if req.PlayerID != nil {
		if err := h.service.RecalculateForPlayer(r.Context(), *req.PlayerID); err != nil {
			slog.Error("recalculate failed", "player_id", *req.PlayerID, "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "player_id": *req.PlayerID})
		return
	}

	if err := h.service.RecalculateAll(r.Context()); err != nil {
		slog.Error("recalculate all failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
