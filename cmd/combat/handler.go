package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type CombatHandler struct {
	service *CombatService
}

func NewCombatHandler(service *CombatService) *CombatHandler {
	return &CombatHandler{service: service}
}

func (h *CombatHandler) Resolve(w http.ResponseWriter, r *http.Request) {
	var req resolveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if req.AttackerID == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "attacker_player_id required"})
		return
	}
	if len(req.AttackerShips) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "attacker_ships required"})
		return
	}

	resp, err := h.service.Resolve(r.Context(), req)
	if err != nil {
		slog.Error("combat resolve failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *CombatHandler) GetReport(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	report, err := h.service.GetReport(r.Context(), id)
	if err != nil {
		slog.Error("get report failed", "id", id, "error", err)
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "report not found"})
		return
	}

	writeJSON(w, http.StatusOK, report)
}

func (h *CombatHandler) ListPlayerReports(w http.ResponseWriter, r *http.Request) {
	playerIDStr := r.URL.Query().Get("player_id")
	playerID, err := strconv.Atoi(playerIDStr)
	if err != nil || playerID == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "player_id required"})
		return
	}

	reports, err := h.service.ListPlayerReports(r.Context(), playerID)
	if err != nil {
		slog.Error("list player reports failed", "player_id", playerID, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	type reportSummary struct {
		ID           int    `json:"id"`
		TargetGalaxy int    `json:"target_galaxy"`
		TargetSystem int    `json:"target_system"`
		TargetPos    int    `json:"target_position"`
		AttackerWon  bool   `json:"attacker_won"`
		RoundCount   int    `json:"round_count"`
		CreatedAt    string `json:"created_at"`
	}

	summaries := make([]reportSummary, 0, len(reports))
	for _, r := range reports {
		summaries = append(summaries, reportSummary{
			ID:           r.ID,
			TargetGalaxy: r.TargetGalaxy,
			TargetSystem: r.TargetSystem,
			TargetPos:    r.TargetPosition,
			AttackerWon:  r.AttackerWon,
			RoundCount:   len(r.Rounds),
			CreatedAt:    r.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	writeJSON(w, http.StatusOK, summaries)
}

type moonInfoRequest struct {
	Galaxy   int `json:"galaxy"`
	System   int `json:"system"`
	Position int `json:"position"`
}

func (h *CombatHandler) MissileStrike(w http.ResponseWriter, r *http.Request) {
	var req MissileStrikeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	result, err := h.service.MissileStrike(r.Context(), req)
	if err != nil {
		slog.Error("missile strike failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *CombatHandler) MoonInfo(w http.ResponseWriter, r *http.Request) {
	var req moonInfoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	moon, err := h.service.GetMoonInfo(r.Context(), req.Galaxy, req.System, req.Position)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "moon not found"})
		return
	}

	writeJSON(w, http.StatusOK, moon)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
