package main

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

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

func (h *Handler) RenamePlanet(w http.ResponseWriter, r *http.Request) {
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

	planetIDStr := chi.URLParam(r, "id")
	planetID, err := strconv.Atoi(planetIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid planet id"})
		return
	}

	planet, err := h.service.repo.FindByID(r.Context(), planetID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "planet not found"})
		return
	}
	if planet.UserID != userID {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "not your planet"})
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if err := h.service.RenamePlanet(r.Context(), planetID, req.Name); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
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

	quantity, buildTime, err := h.service.BuildShips(r.Context(), planet.ID, req.ShipType, req.Quantity)
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

	maxQty, _ := h.service.MaxShipQuantity(r.Context(), planet.ID, req.ShipType)

	writeJSON(w, http.StatusOK, map[string]any{
		"type":              req.ShipType,
		"quantity":          quantity,
		"build_time_seconds": buildTime,
		"max_quantity":      maxQty,
	})
}

func (h *Handler) ListDefenses(w http.ResponseWriter, r *http.Request) {
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
	playerDefenses, _ := h.service.repo.GetPlayerDefenses(r.Context(), planet.ID)

	defenses := make([]DefenseResponse, len(Defenses))
	for i, cfg := range Defenses {
		defenses[i] = DefenseResponse{
			Type: cfg.Type, Name: cfg.Name,
			Metal: cfg.Metal, Crystal: cfg.Crystal, Gas: cfg.Gas,
			Strength: cfg.Strength, Shield: cfg.Shield, Attack: cfg.Attack,
			Fields:   cfg.Fields,
			Quantity: playerDefenses[cfg.Type],
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"shipyard_level": shipyardLevel,
		"defenses":       defenses,
	})
}

func (h *Handler) BuildDefenses(w http.ResponseWriter, r *http.Request) {
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

	var req DefenseBuildRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	quantity, buildTime, err := h.service.BuildDefenses(r.Context(), planet.ID, req.DefenseType, req.Quantity)
	if err != nil {
		slog.Error("build defenses failed", "defense_type", req.DefenseType, "error", err)
		msg := "internal error"
		code := http.StatusInternalServerError
		switch {
		case errors.Is(err, ErrInvalidDefense):
			msg = "invalid defense type"
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

	maxQty, _ := h.service.MaxDefenseQuantity(r.Context(), planet.ID, req.DefenseType)

	writeJSON(w, http.StatusOK, map[string]any{
		"type":              req.DefenseType,
		"quantity":          quantity,
		"build_time_seconds": buildTime,
		"max_quantity":      maxQty,
	})
}

func (h *Handler) BuildIPM(w http.ResponseWriter, r *http.Request) {
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

	planetIDStr := chi.URLParam(r, "id")
	planetID, err := strconv.Atoi(planetIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid planet id"})
		return
	}

	planet, err := h.service.repo.FindByID(r.Context(), planetID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "planet not found"})
		return
	}
	if planet.UserID != userID {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "not your planet"})
		return
	}

	var req BuildMissileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if err := h.service.BuildIPM(r.Context(), userID, planetID, req.Count); err != nil {
		slog.Error("build IPM failed", "error", err)
		msg := "internal error"
		code := http.StatusInternalServerError
		switch {
		case errors.Is(err, ErrInsufficientResources):
			msg = "insufficient resources"
			code = http.StatusBadRequest
		case errors.Is(err, ErrMissileSiloRequired) || strings.Contains(err.Error(), "missile silo"):
			msg = err.Error()
			code = http.StatusBadRequest
		case strings.Contains(err.Error(), "silo capacity"):
			msg = err.Error()
			code = http.StatusBadRequest
		}
		writeJSON(w, code, map[string]string{"error": msg})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"count": req.Count, "type": "ipm"})
}

func (h *Handler) BuildABM(w http.ResponseWriter, r *http.Request) {
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

	planetIDStr := chi.URLParam(r, "id")
	planetID, err := strconv.Atoi(planetIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid planet id"})
		return
	}

	planet, err := h.service.repo.FindByID(r.Context(), planetID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "planet not found"})
		return
	}
	if planet.UserID != userID {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "not your planet"})
		return
	}

	var req BuildMissileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if err := h.service.BuildABM(r.Context(), userID, planetID, req.Count); err != nil {
		slog.Error("build ABM failed", "error", err)
		msg := "internal error"
		code := http.StatusInternalServerError
		switch {
		case errors.Is(err, ErrInsufficientResources):
			msg = "insufficient resources"
			code = http.StatusBadRequest
		case strings.Contains(err.Error(), "missile silo"):
			msg = err.Error()
			code = http.StatusBadRequest
		case strings.Contains(err.Error(), "silo capacity"):
			msg = err.Error()
			code = http.StatusBadRequest
		}
		writeJSON(w, code, map[string]string{"error": msg})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"count": req.Count, "type": "abm"})
}

func (h *Handler) GetMissileCounts(w http.ResponseWriter, r *http.Request) {
	planetIDStr := chi.URLParam(r, "id")
	planetID, err := strconv.Atoi(planetIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid planet id"})
		return
	}

	counts, err := h.service.GetMissileCounts(r.Context(), planetID)
	if err != nil {
		slog.Error("get missile counts failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, counts)
}

func (h *Handler) LaunchIPMs(w http.ResponseWriter, r *http.Request) {
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

	planetIDStr := chi.URLParam(r, "id")
	planetID, err := strconv.Atoi(planetIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid planet id"})
		return
	}

	planet, err := h.service.repo.FindByID(r.Context(), planetID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "planet not found"})
		return
	}
	if planet.UserID != userID {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "not your planet"})
		return
	}

	var req LaunchIPMRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if err := h.service.LaunchIPMs(r.Context(), userID, planetID, req.TargetGalaxy, req.TargetSystem, req.TargetPosition, req.Count); err != nil {
		slog.Error("launch IPMs failed", "error", err)
		msg := "internal error"
		code := http.StatusInternalServerError
		switch {
		case strings.Contains(err.Error(), "insufficient IPMs"):
			msg = err.Error()
			code = http.StatusBadRequest
		case strings.Contains(err.Error(), "silo level"):
			msg = err.Error()
			code = http.StatusBadRequest
		}
		writeJSON(w, code, map[string]string{"error": msg})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"launched": req.Count})
}

func (h *Handler) InternalDeductABMs(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PlanetID int `json:"planet_id"`
		Count    int `json:"count"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if err := h.service.repo.DeductABMs(r.Context(), req.PlanetID, req.Count); err != nil {
		slog.Error("deduct ABMs failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) InternalDeductShips(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PlanetID int            `json:"planet_id"`
		Ships    map[string]int `json:"ships"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if err := h.service.repo.DeductPlayerShips(r.Context(), req.PlanetID, req.Ships); err != nil {
		slog.Error("deduct ships failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) InternalGetPlanetCoords(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PlanetID int `json:"planet_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	planet, err := h.service.repo.FindByID(r.Context(), req.PlanetID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "planet not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{
		"galaxy":   planet.Galaxy,
		"system":   planet.System,
		"position": planet.Position,
	})
}

func (h *Handler) InternalDeductResource(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PlanetID int    `json:"planet_id"`
		Resource string `json:"resource"`
		Amount   int    `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	planet, err := h.service.repo.FindByID(r.Context(), req.PlanetID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "planet not found"})
		return
	}

	switch req.Resource {
	case "metal":
		if planet.Metal < req.Amount {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "insufficient metal"})
			return
		}
		planet.Metal -= req.Amount
	case "crystal":
		if planet.Crystal < req.Amount {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "insufficient crystal"})
			return
		}
		planet.Crystal -= req.Amount
	case "gas":
		if planet.Gas < req.Amount {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "insufficient gas"})
			return
		}
		planet.Gas -= req.Amount
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid resource"})
		return
	}

	if err := h.service.repo.UpdateResources(r.Context(), req.PlanetID, planet.Metal, planet.Crystal, planet.Gas, time.Now()); err != nil {
		slog.Error("update resources failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) InternalGetPlanetInfo(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Galaxy   int `json:"galaxy"`
		System   int `json:"system"`
		Position int `json:"position"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	planet, err := h.service.repo.FindByCoords(r.Context(), req.Galaxy, req.System, req.Position)
	if err != nil {
		if errors.Is(err, ErrPlanetNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "planet not found"})
			return
		}
		slog.Error("find planet by coords failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	ships, err := h.service.repo.GetPlayerShips(r.Context(), planet.ID)
	if err != nil {
		slog.Error("get player ships failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"planet_id": planet.ID,
		"player_id": planet.UserID,
		"metal":     planet.Metal,
		"crystal":   planet.Crystal,
		"gas":       planet.Gas,
		"ships":     ships,
	})
}

func (h *Handler) InternalGetPlayerTechs(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PlayerID int `json:"player_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	techTypes := []string{"combustion_drive", "impulse_drive", "hyperspace_drive"}
	techs := make(map[string]int)
	for _, t := range techTypes {
		level, err := h.service.repo.GetTechLevel(r.Context(), req.PlayerID, t)
		if err != nil {
			slog.Error("get tech level", "tech", t, "error", err)
			continue
		}
		techs[t] = level
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"technologies": techs,
	})
}

func (h *Handler) InternalAddResource(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PlanetID int    `json:"planet_id"`
		Resource string `json:"resource"`
		Amount   int    `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	planet, err := h.service.repo.FindByID(r.Context(), req.PlanetID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "planet not found"})
		return
	}

	switch req.Resource {
	case "metal":
		planet.Metal += req.Amount
	case "crystal":
		planet.Crystal += req.Amount
	case "gas":
		planet.Gas += req.Amount
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid resource"})
		return
	}

	if err := h.service.repo.UpdateResources(r.Context(), req.PlanetID, planet.Metal, planet.Crystal, planet.Gas, time.Now()); err != nil {
		slog.Error("update resources failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) InternalAddShips(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PlanetID int            `json:"planet_id"`
		Ships    map[string]int `json:"ships"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	planet, err := h.service.repo.FindByID(r.Context(), req.PlanetID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "planet not found"})
		return
	}

	for shipType, qty := range req.Ships {
		if err := h.service.repo.AddPlayerShips(r.Context(), req.PlanetID, planet.UserID, shipType, qty); err != nil {
			slog.Error("add ships failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) InternalFindPlanetByCoords(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Galaxy   int `json:"galaxy"`
		System   int `json:"system"`
		Position int `json:"position"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	planet, err := h.service.repo.FindByCoords(r.Context(), req.Galaxy, req.System, req.Position)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "planet not found"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]int{"planet_id": planet.ID})
}

func (h *Handler) InternalDefenseRepair(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PlanetID int            `json:"planet_id"`
		Losses   map[string]int `json:"defense_losses"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	repaired, err := h.service.RepairDefenses(r.Context(), req.PlanetID, req.Losses)
	if err != nil {
		slog.Error("defense repair failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"repaired": repaired})
}

func (h *Handler) InternalDefenseDeduct(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PlanetID int            `json:"planet_id"`
		Losses   map[string]int `json:"defense_losses"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if err := h.service.DeductDefenses(r.Context(), req.PlanetID, req.Losses); err != nil {
		slog.Error("defense deduct failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) InternalDefenseList(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PlanetID int `json:"planet_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	defenses, err := h.service.repo.GetPlayerDefenses(r.Context(), req.PlanetID)
	if err != nil {
		slog.Error("defense list failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"defenses": defenses})
}

func (h *Handler) InternalAddTechLevel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PlayerID int    `json:"player_id"`
		TechType string `json:"tech_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	newLevel, err := h.service.AddTechLevel(r.Context(), req.PlayerID, req.TechType)
	if err != nil {
		slog.Error("add tech level failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"tech_type": req.TechType,
		"new_level": newLevel,
	})
}

func (h *Handler) InternalGetBuildingLevel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PlanetID     int    `json:"planet_id"`
		BuildingType string `json:"building_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	level, err := h.service.repo.GetBuildingLevel(r.Context(), req.PlanetID, req.BuildingType)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "building not found"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]int{"level": level})
}

func (h *Handler) InternalGetPlayerTechLevel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PlayerID int    `json:"player_id"`
		TechType string `json:"tech_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	level, err := h.service.repo.GetTechLevel(r.Context(), req.PlayerID, req.TechType)
	if err != nil {
		slog.Error("get tech level failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]int{"level": level})
}

func (h *Handler) InternalGetHighestLabLevel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PlayerID int `json:"player_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	labLevel, err := h.service.GetHighestLabLevel(r.Context(), req.PlayerID)
	if err != nil {
		slog.Error("get highest lab level failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]int{"lab_level": labLevel})
}

func (h *Handler) InternalCreatePlanet(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID   int `json:"user_id"`
		Galaxy   int `json:"galaxy"`
		System   int `json:"system"`
		Position int `json:"position"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	planet, _, err := h.service.repo.CreateAtCoords(r.Context(), req.UserID, req.Galaxy, req.System, req.Position)
	if err != nil {
		slog.Error("create planet failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"planet_id": planet.ID,
		"name":      planet.Name,
		"galaxy":    planet.Galaxy,
		"system":    planet.System,
		"position":  planet.Position,
	})
}

func (h *Handler) GetMoonBuildings(w http.ResponseWriter, r *http.Request) {
	galaxyStr := chi.URLParam(r, "galaxy")
	systemStr := chi.URLParam(r, "system")
	posStr := chi.URLParam(r, "position")

	galaxy, err := strconv.Atoi(galaxyStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid galaxy"})
		return
	}
	system, err := strconv.Atoi(systemStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid system"})
		return
	}
	position, err := strconv.Atoi(posStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid position"})
		return
	}

	buildings, maxFields, err := h.service.GetMoonBuildings(r.Context(), galaxy, system, position)
	if err != nil {
		slog.Error("get moon buildings failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, MoonBuildingsResponse{
		Buildings:  buildings,
		MaxFields:  maxFields,
		FieldsUsed: len(buildings),
	})
}

func (h *Handler) UpgradeMoonBuilding(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if _, err := strconv.Atoi(userIDStr); err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid user"})
		return
	}

	galaxyStr := chi.URLParam(r, "galaxy")
	systemStr := chi.URLParam(r, "system")
	posStr := chi.URLParam(r, "position")
	buildingType := chi.URLParam(r, "type")

	galaxy, err := strconv.Atoi(galaxyStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid galaxy"})
		return
	}
	system, err := strconv.Atoi(systemStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid system"})
		return
	}
	position, err := strconv.Atoi(posStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid position"})
		return
	}
	if buildingType == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing building type"})
		return
	}

	if err := h.service.StartMoonBuildingUpgrade(r.Context(), galaxy, system, position, buildingType); err != nil {
		slog.Error("upgrade moon building failed", "building", buildingType, "error", err)
		code := http.StatusInternalServerError
		msg := "internal error"
		switch {
		case errors.Is(err, ErrMoonNotFound):
			code = http.StatusNotFound
			msg = "moon not found"
		case errors.Is(err, ErrInsufficientResources):
			code = http.StatusBadRequest
			msg = "insufficient resources"
		case errors.Is(err, ErrMoonBaseRequired):
			code = http.StatusBadRequest
			msg = "moon base level 1 required"
		case errors.Is(err, ErrNoFieldsAvailable):
			code = http.StatusBadRequest
			msg = "no fields available"
		case errors.Is(err, ErrPrerequisitesNotMet):
			code = http.StatusBadRequest
			msg = "prerequisites not met"
		}
		writeJSON(w, code, map[string]string{"error": msg})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) InternalGetMoonBuildingLevel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Galaxy       int    `json:"galaxy"`
		System       int    `json:"system"`
		Position     int    `json:"position"`
		BuildingType string `json:"building_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	level, err := h.service.repo.GetMoonBuildingLevel(r.Context(), req.Galaxy, req.System, req.Position, req.BuildingType)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "building not found"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]int{"level": level})
}

func (h *Handler) LinkWormholes(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SourceGalaxy   int `json:"source_galaxy"`
		SourceSystem   int `json:"source_system"`
		SourcePosition int `json:"source_position"`
		TargetGalaxy   int `json:"target_galaxy"`
		TargetSystem   int `json:"target_system"`
		TargetPosition int `json:"target_position"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if err := h.service.LinkWormholes(r.Context(), req.SourceGalaxy, req.SourceSystem, req.SourcePosition, req.TargetGalaxy, req.TargetSystem, req.TargetPosition); err != nil {
		slog.Error("link wormholes failed", "error", err)
		code := http.StatusInternalServerError
		msg := "internal error"
		if errors.Is(err, ErrWormholeNotFound) {
			code = http.StatusBadRequest
			msg = "wormhole generator not found on one or both moons"
		} else {
			msg = err.Error()
		}
		writeJSON(w, code, map[string]string{"error": msg})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) InternalWormholeInfo(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Galaxy   int `json:"galaxy"`
		System   int `json:"system"`
		Position int `json:"position"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	wEntry, err := h.service.repo.GetWormhole(r.Context(), req.Galaxy, req.System, req.Position)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"has_generator": false,
		})
		return
	}

	linkedCoords := map[string]any{}
	if wEntry.LinkedGalaxy != nil {
		linkedCoords = map[string]any{
			"galaxy":   *wEntry.LinkedGalaxy,
			"system":   *wEntry.LinkedSystem,
			"position": *wEntry.LinkedPosition,
		}
	}

	var cooldownStr string
	if wEntry.CooldownUntil != nil {
		cooldownStr = wEntry.CooldownUntil.Format(time.RFC3339)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"has_generator":  true,
		"level":          wEntry.Level,
		"linked_coords":  linkedCoords,
		"cooldown_until": cooldownStr,
	})
}

func (h *Handler) StarGateLink(w http.ResponseWriter, r *http.Request) {
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

	planetIDStr := chi.URLParam(r, "id")
	planetID, err := strconv.Atoi(planetIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid planet id"})
		return
	}

	planet, err := h.service.repo.FindByID(r.Context(), planetID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "planet not found"})
		return
	}
	if planet.UserID != userID {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "not your planet"})
		return
	}

	var req struct {
		TargetPlanetID int `json:"target_planet_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if err := h.service.StarGateLink(r.Context(), planetID, req.TargetPlanetID); err != nil {
		slog.Error("stargate link failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) StarGateUnlink(w http.ResponseWriter, r *http.Request) {
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

	planetIDStr := chi.URLParam(r, "id")
	planetID, err := strconv.Atoi(planetIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid planet id"})
		return
	}

	planet, err := h.service.repo.FindByID(r.Context(), planetID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "planet not found"})
		return
	}
	if planet.UserID != userID {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "not your planet"})
		return
	}

	if err := h.service.StarGateUnlink(r.Context(), planetID); err != nil {
		slog.Error("stargate unlink failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) StarGateLinks(w http.ResponseWriter, r *http.Request) {
	planetIDStr := chi.URLParam(r, "id")
	planetID, err := strconv.Atoi(planetIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid planet id"})
		return
	}

	link, err := h.service.GetStarGateLink(r.Context(), planetID)
	if err != nil {
		slog.Error("get stargate link failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	if link == nil {
		writeJSON(w, http.StatusOK, map[string]any{"has_link": false})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"has_link":        true,
		"target_planet_id": link.TargetPlanetID,
	})
}

func (h *Handler) InternalCheckStarGateLink(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OriginPlanetID int `json:"origin_planet_id"`
		TargetPlanetID int `json:"target_planet_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	has, err := h.service.HasStarGateLink(r.Context(), req.OriginPlanetID, req.TargetPlanetID)
	if err != nil {
		slog.Error("check stargate link failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"has_link": has})
}

func (h *Handler) BuildIronBehemoth(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if _, err := strconv.Atoi(userIDStr); err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid user"})
		return
	}

	galaxyStr := chi.URLParam(r, "galaxy")
	systemStr := chi.URLParam(r, "system")
	posStr := chi.URLParam(r, "position")

	galaxy, err := strconv.Atoi(galaxyStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid galaxy"})
		return
	}
	system, err := strconv.Atoi(systemStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid system"})
		return
	}
	position, err := strconv.Atoi(posStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid position"})
		return
	}

	var req struct {
		Quantity int `json:"quantity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	quantity, buildTime, err := h.service.BuildIronBehemoth(r.Context(), galaxy, system, position, req.Quantity)
	if err != nil {
		slog.Error("build iron behemoth failed", "error", err)
		code := http.StatusInternalServerError
		msg := "internal error"
		switch {
		case errors.Is(err, ErrPioneerLabRequired):
			code = http.StatusBadRequest
			msg = "pioneer lab level 1 required"
		case errors.Is(err, ErrInsufficientResources):
			code = http.StatusBadRequest
			msg = "insufficient resources"
		}
		writeJSON(w, code, map[string]string{"error": msg})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"quantity":          quantity,
		"build_time_seconds": buildTime,
	})
}

func (h *Handler) GetGems(w http.ResponseWriter, r *http.Request) {
	planetIDStr := chi.URLParam(r, "id")
	planetID, err := strconv.Atoi(planetIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid planet id"})
		return
	}

	if err := h.service.EnsureGemSlots(r.Context(), planetID); err != nil {
		slog.Error("ensure gem slots failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	slots, err := h.service.GetGemSlots(r.Context(), planetID)
	if err != nil {
		slog.Error("get gems failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	bonuses, err := h.service.GetGemBonuses(r.Context(), planetID)
	if err != nil {
		slog.Error("get gem bonuses failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"slots":   slots,
		"bonuses": bonuses,
	})
}

func (h *Handler) EquipGem(w http.ResponseWriter, r *http.Request) {
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

	planetIDStr := chi.URLParam(r, "id")
	planetID, err := strconv.Atoi(planetIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid planet id"})
		return
	}

	planet, err := h.service.repo.FindByID(r.Context(), planetID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "planet not found"})
		return
	}
	if planet.UserID != userID {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "not your planet"})
		return
	}

	var req EquipGemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if err := h.service.EquipGem(r.Context(), planetID, req.SlotIndex, req.GemType); err != nil {
		slog.Error("equip gem failed", "error", err)
		code := http.StatusInternalServerError
		msg := "internal error"
		switch {
		case errors.Is(err, ErrInvalidGemType):
			code = http.StatusBadRequest
			msg = "invalid gem type"
		case errors.Is(err, ErrGemSlotOccupied):
			code = http.StatusBadRequest
			msg = "slot already occupied"
		case errors.Is(err, ErrNoGemSlotsAvailable):
			code = http.StatusBadRequest
			msg = "invalid slot index"
		}
		writeJSON(w, code, map[string]string{"error": msg})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) UnequipGem(w http.ResponseWriter, r *http.Request) {
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

	planetIDStr := chi.URLParam(r, "id")
	planetID, err := strconv.Atoi(planetIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid planet id"})
		return
	}

	planet, err := h.service.repo.FindByID(r.Context(), planetID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "planet not found"})
		return
	}
	if planet.UserID != userID {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "not your planet"})
		return
	}

	var req struct {
		SlotIndex int `json:"slot_index"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if err := h.service.UnequipGem(r.Context(), planetID, req.SlotIndex); err != nil {
		slog.Error("unequip gem failed", "error", err)
		code := http.StatusInternalServerError
		msg := "internal error"
		if errors.Is(err, ErrGemSlotEmpty) {
			code = http.StatusBadRequest
			msg = "slot is empty"
		}
		writeJSON(w, code, map[string]string{"error": msg})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) CombineGem(w http.ResponseWriter, r *http.Request) {
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

	planetIDStr := chi.URLParam(r, "id")
	planetID, err := strconv.Atoi(planetIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid planet id"})
		return
	}

	planet, err := h.service.repo.FindByID(r.Context(), planetID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "planet not found"})
		return
	}
	if planet.UserID != userID {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "not your planet"})
		return
	}

	var req CombineGemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if err := h.service.CombineGem(r.Context(), planetID, req.SlotIndex, req.GemType); err != nil {
		slog.Error("combine gem failed", "error", err)
		code := http.StatusInternalServerError
		msg := "internal error"
		switch {
		case errors.Is(err, ErrInvalidGemType):
			code = http.StatusBadRequest
			msg = "invalid gem type"
		case errors.Is(err, ErrInsufficientShards):
			code = http.StatusBadRequest
			msg = "insufficient shards"
		case errors.Is(err, ErrCombineFailed):
			code = http.StatusBadRequest
			msg = "combine failed"
		case errors.Is(err, ErrNoGemSlotsAvailable):
			code = http.StatusBadRequest
			msg = "invalid slot index"
		}
		writeJSON(w, code, map[string]string{"error": msg})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) InternalGemBonuses(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PlanetID int `json:"planet_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	bonuses, err := h.service.GetGemBonuses(r.Context(), req.PlanetID)
	if err != nil {
		slog.Error("get gem bonuses failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, bonuses)
}

func (h *Handler) InternalSeedNPC(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Galaxy   int `json:"galaxy"`
		System   int `json:"system"`
		Position int `json:"position"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if err := h.service.SeedNPCPlanet(r.Context(), req.Galaxy, req.System, req.Position); err != nil {
		slog.Error("seed NPC failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) InternalSeedAllNPC(w http.ResponseWriter, r *http.Request) {
	if err := h.service.SeedAllNPCPlanets(r.Context()); err != nil {
		slog.Error("seed all NPC failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) InternalClearNPC(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PlanetID int `json:"planet_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if err := h.service.ClearNPCPlanet(r.Context(), req.PlanetID); err != nil {
		slog.Error("clear NPC failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) InternalCheckNPC(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PlanetID int `json:"planet_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	npc, err := h.service.GetNPCPlanetByPlanetID(r.Context(), req.PlanetID)
	if err != nil {
		slog.Error("check NPC failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	if npc == nil {
		writeJSON(w, http.StatusOK, map[string]any{"is_npc": false})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"is_npc":      true,
		"status":      npc.Status,
		"respawns_at": npc.RespawnsAt,
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
