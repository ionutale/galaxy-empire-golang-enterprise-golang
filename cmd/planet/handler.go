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

	energyTechLevel, err := h.service.repo.GetTechLevel(r.Context(), planet.UserID, "energy_tech")
	if err != nil {
		slog.Error("get tech level failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	netEnergy, efficiency := calculatePenaltyFactor(buildings, energyTechLevel)
	vipPoints, totalResources, err := h.service.repo.GetPlayerProgress(r.Context(), planet.ID)
	if err != nil {
		slog.Error("get player progress failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	vipLevel := vipLevelFromPoints(vipPoints)
	rank := rankFromResources(totalResources)
	vipBonus := vipProductionBonus(vipLevel)
	rankBonus := rankProductionBonus(rank)
	prod := h.service.calculateProduction(buildings, efficiency, planet.Type, planet.Temperature, energyTechLevel, vipBonus, rankBonus)
	storage := h.service.calculateStorage(buildings)
	planet.Energy = netEnergy
	resp := toPlanetResponse(planet, buildings, prod, storage, queue, vipPoints, totalResources)
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

func (h *Handler) CancelUpgrade(w http.ResponseWriter, r *http.Request) {
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

	err = h.service.CancelUpgrade(r.Context(), planet.ID, buildingType)
	if err != nil {
		slog.Error("cancel upgrade failed", "building", buildingType, "error", err)
		code := http.StatusInternalServerError
		msg := "internal error"
		if errors.Is(err, ErrNoActiveUpgrade) {
			code = http.StatusBadRequest
			msg = "no active upgrade for this building"
		}
		writeJSON(w, code, map[string]string{"error": msg})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) DeconstructBuilding(w http.ResponseWriter, r *http.Request) {
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

	entry, err := h.service.QueueDeconstruction(r.Context(), planet.ID, buildingType)
	if err != nil {
		slog.Error("deconstruct failed", "building", buildingType, "error", err)
		code := http.StatusInternalServerError
		msg := "internal error"
		switch {
		case errors.Is(err, ErrBuildingNotFound):
			code = http.StatusBadRequest
			msg = "building not found"
		case errors.Is(err, ErrAlreadyDeconstructing):
			code = http.StatusConflict
			msg = "building already queued for deconstruction"
		case errors.Is(err, ErrAlreadyQueued):
			code = http.StatusConflict
			msg = "building is currently being upgraded"
		}
		writeJSON(w, code, map[string]string{"error": msg})
		return
	}

	writeJSON(w, http.StatusOK, entry)
}

func (h *Handler) ListGalaxies(w http.ResponseWriter, r *http.Request) {
	galaxies, err := h.service.repo.ListGalaxies(r.Context())
	if err != nil {
		slog.Error("list galaxies failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, galaxies)
}

func (h *Handler) ListSystems(w http.ResponseWriter, r *http.Request) {
	galaxyIDStr := chi.URLParam(r, "galaxyID")
	galaxyID, err := strconv.Atoi(galaxyIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid galaxy"})
		return
	}

	pageStr := r.URL.Query().Get("page")
	page := 1
	if pageStr != "" {
		page, err = strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid page"})
			return
		}
	}

	pageSize := 10
	systems, total, err := h.service.repo.ListSystems(r.Context(), galaxyID, page, pageSize)
	if err != nil {
		slog.Error("list systems failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	totalPages := (total + pageSize - 1) / pageSize
	writeJSON(w, http.StatusOK, map[string]any{
		"galaxy_id":   galaxyID,
		"page":        page,
		"total_pages": totalPages,
		"systems":     systems,
	})
}

func (h *Handler) GetPositions(w http.ResponseWriter, r *http.Request) {
	systemIDStr := chi.URLParam(r, "systemID")
	systemID, err := strconv.Atoi(systemIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid system"})
		return
	}

	positions, err := h.service.repo.GetSystemPositions(r.Context(), systemID)
	if err != nil {
		slog.Error("get positions failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"system_id": systemID,
		"positions": positions,
	})
}

func (h *Handler) ListShips(w http.ResponseWriter, r *http.Request) {
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

	planet, _, err := h.service.GetOrCreatePlanet(r.Context(), userID)
	if err != nil {
		slog.Error("get planet failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	shipyardLevel, _ := h.service.repo.GetBuildingLevel(r.Context(), planet.ID, "shipyard")
	playerShips, _ := h.service.repo.GetPlayerShips(r.Context(), planet.ID)

	ships := make([]ShipResponse, len(Ships))
	for i, cfg := range Ships {
		ships[i] = ShipResponse{
			Type: cfg.Type, Name: cfg.Name,
			Metal: cfg.Metal, Crystal: cfg.Crystal, Gas: cfg.Gas,
			Speed: cfg.Speed, Cargo: cfg.Cargo, Fuel: cfg.Fuel,
			Strength: cfg.Strength, Shield: cfg.Shield, Attack: cfg.Attack,
			Quantity: playerShips[cfg.Type],
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"shipyard_level": shipyardLevel,
		"ships":          ships,
	})
}

func (h *Handler) BuildShips(w http.ResponseWriter, r *http.Request) {
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

	planet, _, err := h.service.GetOrCreatePlanet(r.Context(), userID)
	if err != nil {
		slog.Error("get planet failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	var req BuildRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	quantity, err := h.service.BuildShips(r.Context(), planet.ID, req.ShipType, req.Quantity)
	if err != nil {
		slog.Error("build ships failed", "ship_type", req.ShipType, "error", err)
		msg := "internal error"
		code := http.StatusInternalServerError
		switch {
		case errors.Is(err, ErrInvalidShip):
			msg = "invalid ship type"
			code = http.StatusBadRequest
		case errors.Is(err, ErrNoShipyard):
			msg = "no shipyard"
			code = http.StatusBadRequest
		case errors.Is(err, ErrInsufficientResources):
			msg = "insufficient resources"
			code = http.StatusBadRequest
		}
		writeJSON(w, code, map[string]string{"error": msg})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"type":     req.ShipType,
		"quantity": quantity,
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
