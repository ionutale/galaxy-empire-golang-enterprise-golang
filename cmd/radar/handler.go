package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
)

type Handler struct {
	service *RadarService
}

func NewHandler(service *RadarService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Scan(w http.ResponseWriter, r *http.Request) {
	playerID, ok := getUserID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	events, err := h.service.Scan(r.Context(), playerID)
	if err != nil {
		slog.Error("scan failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, toRadarEventResponses(events))
}

func (h *Handler) GetEvents(w http.ResponseWriter, r *http.Request) {
	playerID, ok := getUserID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req EventsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	events, err := h.service.GetEvents(r.Context(), playerID, req.Scope)
	if err != nil {
		slog.Error("get events failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, toRadarEventResponses(events))
}

func (h *Handler) ResolveEvent(w http.ResponseWriter, r *http.Request) {
	playerID, ok := getUserID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req ResolveEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if req.EventID == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing event_id"})
		return
	}

	if err := h.service.ResolveEvent(r.Context(), playerID, req.EventID); err != nil {
		slog.Error("resolve event failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "could not resolve event"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) PlanetStatus(w http.ResponseWriter, r *http.Request) {
	playerID, ok := getUserID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	statuses, err := h.service.PlanetStatus(r.Context(), playerID)
	if err != nil {
		slog.Error("planet status failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	if statuses == nil {
		statuses = []PlanetStatusResponse{}
	}

	writeJSON(w, http.StatusOK, statuses)
}

func (h *Handler) EUXScan(w http.ResponseWriter, r *http.Request) {
	playerID, ok := getUserID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req EUXScanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	result, err := h.service.EUXScan(r.Context(), playerID, req.TargetGalaxy, req.TargetSystem, req.TargetPosition)
	if err != nil {
		slog.Error("eu-x scan failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "scan failed"})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) InternalDetect(w http.ResponseWriter, r *http.Request) {
	var req DetectFleetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if req.TargetPlayerID == 0 || req.FleetID == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing required fields"})
		return
	}

	if err := h.service.DetectFleet(r.Context(), req); err != nil {
		slog.Error("detect fleet failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func toRadarEventResponses(events []RadarEvent) []RadarEventResponse {
	if events == nil {
		return []RadarEventResponse{}
	}
	resp := make([]RadarEventResponse, len(events))
	for i, e := range events {
		resp[i] = RadarEventResponse{
			ID:             e.ID,
			EventType:      e.EventType,
			SourcePlayerID: e.SourcePlayerID,
			FleetID:        e.FleetID,
			TargetGalaxy:   e.TargetGalaxy,
			TargetSystem:   e.TargetSystem,
			TargetPosition: e.TargetPosition,
			OriginGalaxy:   e.OriginGalaxy,
			OriginSystem:   e.OriginSystem,
			OriginPosition: e.OriginPosition,
			ArrivalTime:    e.ArrivalTime,
			DetectedAt:     e.DetectedAt,
			Resolved:       e.Resolved,
		}
	}
	return resp
}

func getUserID(r *http.Request) (int, bool) {
	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		return 0, false
	}
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return 0, false
	}
	return userID, true
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
