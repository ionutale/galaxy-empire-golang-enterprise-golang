package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	service *EspionageService
}

func NewHandler(service *EspionageService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Probe(w http.ResponseWriter, r *http.Request) {
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

	var req ProbeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if req.TargetGalaxy < 1 || req.TargetSystem < 1 || req.TargetPosition < 1 || req.PlanetID < 1 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid target or planet"})
		return
	}

	report, err := h.service.SendProbe(r.Context(), userID, req)
	if err != nil {
		slog.Error("send probe failed", "error", err)
		code := http.StatusBadRequest
		writeJSON(w, code, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, ProbeResponse{
		ReportID:  report.ID,
		CreatedAt: report.CreatedAt,
	})
}

func (h *Handler) ListReports(w http.ResponseWriter, r *http.Request) {
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

	reports, err := h.service.ListReports(r.Context(), userID)
	if err != nil {
		slog.Error("list reports failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	resp := make([]EspionageReportResponse, len(reports))
	for i, rep := range reports {
		resp[i] = toReportResponse(rep)
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetReport(w http.ResponseWriter, r *http.Request) {
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

	reportIDStr := chi.URLParam(r, "id")
	reportID, err := strconv.Atoi(reportIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid report id"})
		return
	}

	report, err := h.service.GetReport(r.Context(), userID, reportID)
	if err != nil {
		slog.Error("get report failed", "error", err)
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "report not found"})
		return
	}

	writeJSON(w, http.StatusOK, toReportResponse(report))
}

func (h *Handler) DeleteReport(w http.ResponseWriter, r *http.Request) {
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

	reportIDStr := chi.URLParam(r, "id")
	reportID, err := strconv.Atoi(reportIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid report id"})
		return
	}

	if err := h.service.DeleteReport(r.Context(), userID, reportID); err != nil {
		slog.Error("delete report failed", "error", err)
		code := http.StatusNotFound
		writeJSON(w, code, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func toReportResponse(rep EspionageReport) EspionageReportResponse {
	resp := EspionageReportResponse{
		ID:              rep.ID,
		PlayerID:        rep.PlayerID,
		TargetPlayerID:  rep.TargetPlayerID,
		TargetGalaxy:    rep.TargetGalaxy,
		TargetSystem:    rep.TargetSystem,
		TargetPosition:  rep.TargetPosition,
		DetailLevel:     rep.DetailLevel,
		CreatedAt:       rep.CreatedAt,
		ExpiresAt:       rep.ExpiresAt,
	}

	if rep.DetailLevel >= 1 {
		resp.Resources = rep.Resources
	}
	if rep.DetailLevel >= 3 {
		resp.Fleet = rep.Fleet
	}
	resp.ReportData = rep.ReportData

	return resp
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
