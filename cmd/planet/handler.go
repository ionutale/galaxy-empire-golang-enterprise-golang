package main

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
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

	queue, err := h.service.repo.GetActiveQueue(r.Context(), planet.ID)
	if err != nil {
		slog.Error("get queue failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	netEnergy, efficiency := calculatePenaltyFactor(buildings)
	prod := h.service.calculateProduction(buildings, efficiency)
	storage := h.service.calculateStorage(buildings)
	planet.Energy = netEnergy
	resp := toPlanetResponse(planet, buildings, prod, storage, queue)
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) StartUpgrade(w http.ResponseWriter, r *http.Request) {
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

	buildingType := chi.URLParam(r, "type")
	if buildingType == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing building type"})
		return
	}

	planet, _, err := h.service.GetOrCreatePlanet(r.Context(), userID)
	if err != nil {
		slog.Error("get planet failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	entry, err := h.service.StartBuildingUpgrade(r.Context(), planet.ID, buildingType)
	if err != nil {
		slog.Error("start upgrade failed", "building", buildingType, "error", err)
		code := http.StatusInternalServerError
		msg := "internal error"
		switch {
		case errors.Is(err, ErrInsufficientResources):
			code = http.StatusBadRequest
			msg = "insufficient resources"
		case errors.Is(err, ErrAlreadyQueued):
			code = http.StatusConflict
			msg = "building already in queue"
		case errors.Is(err, ErrInvalidBuilding):
			code = http.StatusBadRequest
			msg = "invalid building type"
		}
		writeJSON(w, code, map[string]string{"error": msg})
		return
	}

	writeJSON(w, http.StatusOK, entry)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
