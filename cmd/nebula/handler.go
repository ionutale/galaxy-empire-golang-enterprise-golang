package main

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func getUserID(r *http.Request) (int, error) {
	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		return 0, errors.New("unauthorized")
	}
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return 0, errors.New("invalid user")
	}
	return userID, nil
}

type Handler struct {
	service *NebulaService
}

func NewHandler(service *NebulaService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) StartExpedition(w http.ResponseWriter, r *http.Request) {
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

	var req StartExpeditionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if req.PlanetID == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing planet_id"})
		return
	}

	expedition, err := h.service.StartExpedition(r.Context(), userID, req.PlanetID, req.Ships)
	if err != nil {
		slog.Error("start expedition failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, toExpeditionResponse(expedition))
}

func (h *Handler) GetExpedition(w http.ResponseWriter, r *http.Request) {
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

	expeditionIDStr := chi.URLParam(r, "id")
	if expeditionIDStr == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing expedition id"})
		return
	}
	expeditionID, err := strconv.Atoi(expeditionIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid expedition id"})
		return
	}

	expedition, err := h.service.repo.GetExpedition(r.Context(), expeditionID, userID)
	if err != nil {
		slog.Error("get expedition failed", "error", err)
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "expedition not found"})
		return
	}

	writeJSON(w, http.StatusOK, toExpeditionResponse(expedition))
}

func (h *Handler) ListExpeditions(w http.ResponseWriter, r *http.Request) {
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

	expeditions, err := h.service.repo.ListPlayerExpeditions(r.Context(), userID)
	if err != nil {
		slog.Error("list expeditions failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	resp := make([]ExpeditionResponse, len(expeditions))
	for i, e := range expeditions {
		resp[i] = toExpeditionResponse(e)
	}
	writeJSON(w, http.StatusOK, resp)
}

func toExpeditionResponse(e Expedition) ExpeditionResponse {
	return ExpeditionResponse{
		ID:              e.ID,
		PlayerID:        e.PlayerID,
		Status:          e.Status,
		ShipsSent:       e.ShipsSent,
		ShipsLost:       e.ShipsLost,
		ShipsFound:      e.ShipsFound,
		ResourcesFound:  e.ResourcesFound,
		DarkMatterFound: e.DarkMatterFound,
		Outcome:         e.Outcome,
		StartedAt:       e.StartedAt,
		TravelDuration:  e.TravelDuration,
		ExploreDuration: e.ExploreDuration,
		CompletedAt:     e.CompletedAt,
	}
}

func (h *Handler) DMBalance(w http.ResponseWriter, r *http.Request) {
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

	balance, totalEarned, err := h.service.repo.GetDarkMatterBalance(r.Context(), userID)
	if err != nil {
		slog.Error("get dm balance failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, PlayerDarkMatter{
		PlayerID:    userID,
		Balance:     balance,
		TotalEarned: totalEarned,
	})
}

func (h *Handler) DMSpend(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}

	var req struct {
		Amount int    `json:"amount"`
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if req.Amount <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid amount"})
		return
	}

	if req.Reason == "" {
		req.Reason = "manual"
	}

	newBalance, err := h.service.SpendDarkMatter(r.Context(), userID, req.Amount, req.Reason)
	if err != nil {
		if err.Error() == "insufficient dark matter" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "insufficient dark matter"})
			return
		}
		slog.Error("spend dm failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]int{"new_balance": newBalance})
}

func (h *Handler) DMSpeedUp(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}
	var req struct {
		TargetType string `json:"target_type"`
		TargetID   int    `json:"target_id"`
		Seconds    int    `json:"seconds_remaining"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	if req.Seconds <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid seconds"})
		return
	}
	validTargets := map[string]bool{"research": true, "building": true, "shipyard": true}
	if !validTargets[req.TargetType] {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid target type"})
		return
	}
	dmCost, secondsSaved, err := h.service.SpeedUp(r.Context(), userID, req.Seconds)
	if err != nil {
		if err.Error() == "insufficient dark matter" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "insufficient dark matter"})
			return
		}
		slog.Error("speed up failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"dm_cost": dmCost, "seconds_saved": secondsSaved})
}

func (h *Handler) DMEstimateCost(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}
	_ = userID
	var req struct {
		Seconds int `json:"seconds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	if req.Seconds <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid seconds"})
		return
	}
	cost := h.service.CalculateSpeedUpCost(req.Seconds)
	writeJSON(w, http.StatusOK, map[string]int{"dm_cost": cost})
}

func (h *Handler) DMTransactions(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}
	txs, err := h.service.repo.ListDMTransactions(r.Context(), userID, 50)
	if err != nil {
		slog.Error("list dm transactions failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	if txs == nil {
		txs = []DMTransaction{}
	}
	writeJSON(w, http.StatusOK, txs)
}

func (h *Handler) HireCommander(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}
	var req struct {
		CommanderType string `json:"commander_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	if req.CommanderType == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing commander_type"})
		return
	}
	entry, err := h.service.HireCommander(r.Context(), userID, req.CommanderType)
	if err != nil {
		if err.Error() == "insufficient dark matter" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "insufficient dark matter"})
			return
		}
		slog.Error("hire commander failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, entry)
}

func (h *Handler) ListCommanders(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}
	entries, err := h.service.GetPlayerCommanders(r.Context(), userID)
	if err != nil {
		slog.Error("list commanders failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	if entries == nil {
		entries = []CommanderEntry{}
	}
	writeJSON(w, http.StatusOK, entries)
}

func (h *Handler) AvailableCommanders(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.service.GetAvailableCommanders())
}

func (h *Handler) InternalActiveCommanders(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PlayerID int `json:"player_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	entries, err := h.service.GetActiveCommandersRaw(r.Context(), req.PlayerID)
	if err != nil {
		slog.Error("internal active commanders failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	type activeCommander struct {
		CommanderType string `json:"commander_type"`
		Level         int    `json:"level"`
	}
	result := make([]activeCommander, len(entries))
	for i, e := range entries {
		result[i] = activeCommander{
			CommanderType: e.CommanderType,
			Level:         e.Level,
		}
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) CreditsBalance(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}

	credits, err := h.service.GetCreditsBalance(r.Context(), userID)
	if err != nil {
		slog.Error("get credits balance failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, credits)
}

func (h *Handler) CreditsSpend(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}

	var req struct {
		Amount int    `json:"amount"`
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if req.Amount <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid amount"})
		return
	}

	if req.Reason == "" {
		req.Reason = "manual"
	}

	newBalance, err := h.service.SpendCredits(r.Context(), userID, req.Amount, req.Reason)
	if err != nil {
		if err.Error() == "insufficient credits" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "insufficient credits"})
			return
		}
		slog.Error("spend credits failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]int{"new_balance": newBalance})
}

func (h *Handler) CreditsTransactions(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}

	txs, err := h.service.ListCreditsTransactions(r.Context(), userID)
	if err != nil {
		slog.Error("list credits transactions failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	if txs == nil {
		txs = []CreditsTransaction{}
	}
	writeJSON(w, http.StatusOK, txs)
}

func (h *Handler) InternalAddCredits(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PlayerID int    `json:"player_id"`
		Amount   int    `json:"amount"`
		Reason   string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if req.PlayerID == 0 || req.Amount <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid player_id or amount"})
		return
	}

	if req.Reason == "" {
		req.Reason = "internal"
	}

	newBalance, err := h.service.AddCredits(r.Context(), req.PlayerID, req.Amount, req.Reason)
	if err != nil {
		slog.Error("internal add credits failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]int{"new_balance": newBalance})
}

// Daily Gift handlers

func (h *Handler) ClaimDailyGift(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}

	result, err := h.service.ClaimDailyGift(r.Context(), userID)
	if err != nil {
		slog.Error("claim daily gift failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) GetDailyGiftStatus(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}

	status, err := h.service.GetDailyGiftStatus(r.Context(), userID)
	if err != nil {
		slog.Error("get daily gift status failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, status)
}

// Daily Task handlers

func (h *Handler) GetDailyTasks(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}

	tasks, err := h.service.GetDailyTasks(r.Context(), userID)
	if err != nil {
		slog.Error("get daily tasks failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	if tasks == nil {
		tasks = []DailyTask{}
	}

	writeJSON(w, http.StatusOK, tasks)
}

func (h *Handler) UpdateTaskProgress(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}

	taskIDStr := chi.URLParam(r, "id")
	taskID, err := strconv.Atoi(taskIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid task id"})
		return
	}

	var req struct {
		TaskType string `json:"task_type"`
		Amount   int    `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	_ = taskID
	if err := h.service.UpdateTaskProgress(r.Context(), userID, req.TaskType, req.Amount); err != nil {
		slog.Error("update task progress failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) ClaimTaskReward(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}

	taskIDStr := chi.URLParam(r, "id")
	taskID, err := strconv.Atoi(taskIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid task id"})
		return
	}

	task, err := h.service.ClaimTask(r.Context(), userID, taskID)
	if err != nil {
		slog.Error("claim task reward failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, task)
}

func (h *Handler) RerollTask(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}

	var req struct {
		TaskID int `json:"task_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	task, err := h.service.RerollTask(r.Context(), userID, req.TaskID)
	if err != nil {
		slog.Error("reroll task failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, task)
}

func (h *Handler) ClaimAllTasks(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}

	tasks, err := h.service.ClaimAllTasks(r.Context(), userID)
	if err != nil {
		slog.Error("claim all tasks failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	if tasks == nil {
		tasks = []DailyTask{}
	}

	writeJSON(w, http.StatusOK, tasks)
}

func (h *Handler) ListStoreItems(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}
	_ = userID

	items := h.service.ListStoreItems()
	if items == nil {
		items = []StoreItem{}
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) BuyItem(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}

	itemID := chi.URLParam(r, "itemId")
	if itemID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing item id"})
		return
	}

	result, err := h.service.BuyItem(r.Context(), userID, itemID)
	if err != nil {
		slog.Error("buy item failed", "item_id", itemID, "error", err)
		code := http.StatusBadRequest
		msg := err.Error()
		if msg == "insufficient dark matter" || msg == "insufficient credits" {
			code = http.StatusBadRequest
		}
		writeJSON(w, code, map[string]string{"error": msg})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) GetGalactoniteDiscoverer(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}

	level, err := h.service.GetGalactoniteDiscovererLevel(r.Context(), userID)
	if err != nil {
		slog.Error("get discoverer level failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, GalactoniteDiscoverer{
		PlayerID: userID,
		Level:    level,
	})
}

func (h *Handler) UpgradeGalactoniteDiscoverer(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}

	newLevel, err := h.service.UpgradeGalactoniteDiscoverer(r.Context(), userID)
	if err != nil {
		if err.Error() == "insufficient dark matter" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "insufficient dark matter"})
			return
		}
		slog.Error("upgrade discoverer failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]int{"level": newLevel})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
