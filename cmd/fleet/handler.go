package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
)

type FleetHandler struct {
	service *FleetService
}

func NewFleetHandler(service *FleetService) *FleetHandler {
	return &FleetHandler{service: service}
}

func (h *FleetHandler) MyFleets(w http.ResponseWriter, r *http.Request) {
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

	fleets, err := h.service.repo.ListPlayerFleets(r.Context(), userID)
	if err != nil {
		slog.Error("list fleets failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	resp := make([]FleetResponse, len(fleets))
	for i, f := range fleets {
		resp[i] = toFleetResponse(f)
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *FleetHandler) Dispatch(w http.ResponseWriter, r *http.Request) {
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

	var req DispatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	fleet, err := h.service.DispatchFleet(r.Context(), userID, req)
	if err != nil {
		slog.Error("dispatch failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, toFleetResponse(fleet))
}

func toFleetResponse(f Fleet) FleetResponse {
	return FleetResponse{
		ID: f.ID, PlayerID: f.PlayerID, OriginPlanetID: f.OriginPlanetID,
		TargetGalaxy: f.TargetGalaxy, TargetSystem: f.TargetSystem, TargetPosition: f.TargetPosition,
		Mission: f.Mission, Status: f.Status, SpeedPct: f.SpeedPct, Ships: f.Ships,
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
