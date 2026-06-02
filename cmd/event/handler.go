package main

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

var errEventNotActive = errors.New("event is not active")
var errAlreadyJoined = errors.New("already joined this event")

type Handler struct {
	service *EventService
}

func NewHandler(service *EventService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetActiveEvents(w http.ResponseWriter, r *http.Request) {
	events, err := h.service.GetActiveEvents(r.Context())
	if err != nil {
		slog.Error("get active events", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	resp := make([]EventResponse, len(events))
	for i, e := range events {
		resp[i] = toEventResponse(e, false, false, false)
	}

	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr != "" {
		userID, err := strconv.Atoi(userIDStr)
		if err == nil {
			eventIDs := make([]int, len(events))
			for i, e := range events {
				eventIDs[i] = e.ID
			}
			participations, err := h.service.repo.GetPlayerParticipations(r.Context(), userID, eventIDs)
			if err == nil {
				for i, e := range events {
					if p, ok := participations[e.ID]; ok {
						resp[i].Joined = true
						resp[i].Completed = p.Completed
						resp[i].RewardsClaimed = p.RewardsClaimed
					}
				}
			}
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetAllEvents(w http.ResponseWriter, r *http.Request) {
	events, err := h.service.GetAllEvents(r.Context())
	if err != nil {
		slog.Error("get all events", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	resp := make([]EventResponse, len(events))
	for i, e := range events {
		resp[i] = toEventResponse(e, false, false, false)
	}

	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr != "" {
		userID, err := strconv.Atoi(userIDStr)
		if err == nil {
			eventIDs := make([]int, len(events))
			for i, e := range events {
				eventIDs[i] = e.ID
			}
			participations, err := h.service.repo.GetPlayerParticipations(r.Context(), userID, eventIDs)
			if err == nil {
				for i, e := range events {
					if p, ok := participations[e.ID]; ok {
						resp[i].Joined = true
						resp[i].Completed = p.Completed
						resp[i].RewardsClaimed = p.RewardsClaimed
					}
				}
			}
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) JoinEvent(w http.ResponseWriter, r *http.Request) {
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

	eventIDStr := chi.URLParam(r, "id")
	eventID, err := strconv.Atoi(eventIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid event id"})
		return
	}

	if err := h.service.JoinEvent(r.Context(), userID, eventID); err != nil {
		code := http.StatusInternalServerError
		msg := "internal error"
		switch {
		case errors.Is(err, errEventNotActive):
			code = http.StatusBadRequest
			msg = "event is not active"
		case errors.Is(err, errAlreadyJoined):
			code = http.StatusConflict
			msg = "already joined this event"
		}
		writeJSON(w, code, map[string]string{"error": msg})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) ClaimRewards(w http.ResponseWriter, r *http.Request) {
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

	eventIDStr := chi.URLParam(r, "id")
	eventID, err := strconv.Atoi(eventIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid event id"})
		return
	}

	if err := h.service.ClaimRewards(r.Context(), userID, eventID); err != nil {
		code := http.StatusInternalServerError
		msg := "internal error"
		if err.Error() == "cannot claim rewards: not completed or already claimed" {
			code = http.StatusBadRequest
			msg = "cannot claim rewards"
		}
		writeJSON(w, code, map[string]string{"error": msg})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) InternalCheck(w http.ResponseWriter, r *http.Request) {
	h.service.checkAndUpdateEvents(r.Context())
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) InternalCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string         `json:"name"`
		Description string         `json:"description"`
		EventType   string         `json:"event_type"`
		Modifiers   map[string]any `json:"modifiers,omitempty"`
		StartsAt    string         `json:"starts_at"`
		EndsAt      string         `json:"ends_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	startsAt, err := time.Parse(time.RFC3339, req.StartsAt)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid starts_at"})
		return
	}
	endsAt, err := time.Parse(time.RFC3339, req.EndsAt)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid ends_at"})
		return
	}

	if req.Modifiers == nil {
		if mods, ok := eventTypeModifiers[req.EventType]; ok {
			req.Modifiers = mods
		} else {
			req.Modifiers = map[string]any{}
		}
	}

	event := Event{
		Name:        req.Name,
		Description: req.Description,
		EventType:   req.EventType,
		Modifiers:   req.Modifiers,
		StartsAt:    startsAt,
		EndsAt:      endsAt,
		Status:      "upcoming",
	}

	created, err := h.service.CreateEvent(r.Context(), event)
	if err != nil {
		slog.Error("create event", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, created)
}

func toEventResponse(e Event, joined, completed, rewardsClaimed bool) EventResponse {
	return EventResponse{
		ID:            e.ID,
		Name:          e.Name,
		Description:   e.Description,
		EventType:     e.EventType,
		Modifiers:     e.Modifiers,
		StartsAt:      e.StartsAt,
		EndsAt:        e.EndsAt,
		Status:        e.Status,
		Joined:        joined,
		Completed:     completed,
		RewardsClaimed: rewardsClaimed,
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
