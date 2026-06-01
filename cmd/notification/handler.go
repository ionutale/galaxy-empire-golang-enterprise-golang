package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type NotificationHandler struct {
	service *NotificationService
}

func NewNotificationHandler(service *NotificationService) *NotificationHandler {
	return &NotificationHandler{service: service}
}

func (h *NotificationHandler) List(w http.ResponseWriter, r *http.Request) {
	playerID, ok := h.service.GetUserID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	unreadOnly := r.URL.Query().Get("unread_only") == "true"
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	notifications, total, err := h.service.ListNotifications(r.Context(), playerID, unreadOnly, limit, offset)
	if err != nil {
		slog.Error("list notifications failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list notifications"})
		return
	}

	writeJSON(w, http.StatusOK, NotificationListResponse{
		Notifications: notifications,
		Total:         total,
	})
}

func (h *NotificationHandler) UnreadCount(w http.ResponseWriter, r *http.Request) {
	playerID, ok := h.service.GetUserID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	count, err := h.service.UnreadCount(r.Context(), playerID)
	if err != nil {
		slog.Error("unread count failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get unread count"})
		return
	}

	writeJSON(w, http.StatusOK, UnreadCountResponse{Count: count})
}

func (h *NotificationHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	playerID, ok := h.service.GetUserID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid notification id"})
		return
	}

	if err := h.service.MarkRead(r.Context(), id, playerID); err != nil {
		slog.Error("mark read failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *NotificationHandler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	playerID, ok := h.service.GetUserID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	if err := h.service.MarkAllRead(r.Context(), playerID); err != nil {
		slog.Error("mark all read failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to mark all as read"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *NotificationHandler) CreateInternal(w http.ResponseWriter, r *http.Request) {
	if !h.service.verifyInternalSecret(r) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req CreateNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if req.PlayerID <= 0 || req.Category == "" || req.Title == "" || req.Message == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "player_id, category, title, and message are required"})
		return
	}

	n, err := h.service.CreateNotification(r.Context(), req.PlayerID, req.Category, req.Title, req.Message)
	if err != nil {
		slog.Error("create notification failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create notification"})
		return
	}

	writeJSON(w, http.StatusCreated, n)
}

func (h *NotificationHandler) Stream(w http.ResponseWriter, r *http.Request) {
	h.service.Stream(w, r)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
