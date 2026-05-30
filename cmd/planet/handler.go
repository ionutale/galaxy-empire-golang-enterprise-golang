package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
)

type Handler struct {
	service *PlanetService
}

func NewHandler(service *PlanetService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetMyPlanet(w http.ResponseWriter, r *http.Request) {
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

	planet, buildings, err := h.service.GetOrCreatePlanet(r.Context(), userID)
	if err != nil {
		slog.Error("get planet failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	netEnergy, efficiency := calculatePenaltyFactor(buildings)
	prod := h.service.calculateProduction(buildings, efficiency)
	storage := h.service.calculateStorage(buildings)
	planet.Energy = netEnergy
	resp := toPlanetResponse(planet, buildings, prod, storage)
	writeJSON(w, http.StatusOK, resp)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
