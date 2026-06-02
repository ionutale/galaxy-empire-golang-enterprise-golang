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
	service *ResearchService
}

func NewHandler(service *ResearchService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListTechs(w http.ResponseWriter, r *http.Request) {
	playerID, err := userIDFromRequest(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}

	techs, labLevel, err := h.service.ListTechs(r.Context(), playerID)
	if err != nil {
		slog.Error("list techs failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"lab_level": labLevel,
		"techs":     techs,
	})
}

func (h *Handler) StartResearch(w http.ResponseWriter, r *http.Request) {
	playerID, err := userIDFromRequest(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}

	techType := chi.URLParam(r, "type")
	if techType == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing tech type"})
		return
	}

	var req StartResearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.PlanetID == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "planet_id is required"})
		return
	}

	resp, err := h.service.StartResearch(r.Context(), playerID, req.PlanetID, techType)
	if err != nil {
		slog.Error("start research failed", "tech", techType, "error", err)
		code := http.StatusInternalServerError
		msg := "internal error"
		switch {
		case errors.Is(err, ErrInvalidTech):
			code = http.StatusBadRequest
			msg = "invalid tech type"
		case errors.Is(err, ErrPrerequisitesNotMet):
			code = http.StatusBadRequest
			msg = "prerequisites not met"
		case errors.Is(err, ErrAlreadyResearching):
			code = http.StatusConflict
			msg = "already researching this tech"
		case errors.Is(err, ErrResearchInProgress):
			code = http.StatusConflict
			msg = "research already in progress"
		}
		writeJSON(w, code, map[string]string{"error": msg})
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) CancelResearch(w http.ResponseWriter, r *http.Request) {
	playerID, err := userIDFromRequest(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}

	techType := chi.URLParam(r, "type")
	if techType == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing tech type"})
		return
	}

	resp, err := h.service.CancelResearch(r.Context(), playerID, techType)
	if err != nil {
		slog.Error("cancel research failed", "tech", techType, "error", err)
		code := http.StatusInternalServerError
		msg := "internal error"
		if errors.Is(err, ErrNoActiveResearch) {
			code = http.StatusBadRequest
			msg = "no active research for this tech"
		}
		writeJSON(w, code, map[string]string{"error": msg})
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) ListQueue(w http.ResponseWriter, r *http.Request) {
	playerID, err := userIDFromRequest(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}

	queue, err := h.service.repo.ListActiveResearch(r.Context(), playerID)
	if err != nil {
		slog.Error("list queue failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, queue)
}

func userIDFromRequest(r *http.Request) (int, error) {
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
