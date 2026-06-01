package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
)

type Handler struct {
	service *FriendService
}

func NewHandler(service *FriendService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) AddFriend(w http.ResponseWriter, r *http.Request) {
	playerID, ok := getUserID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req AddFriendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if req.FriendID == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing friend_id"})
		return
	}

	friendship, err := h.service.SendRequest(r.Context(), playerID, req.FriendID)
	if err != nil {
		slog.Error("add friend failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":        friendship.ID,
		"player_id": friendship.PlayerID,
		"friend_id": friendship.FriendID,
		"status":    friendship.Status,
	})
}

func (h *Handler) AcceptFriend(w http.ResponseWriter, r *http.Request) {
	playerID, ok := getUserID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req AddFriendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if req.FriendID == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing friend_id"})
		return
	}

	if err := h.service.AcceptRequest(r.Context(), playerID, req.FriendID); err != nil {
		slog.Error("accept friend failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) RemoveFriend(w http.ResponseWriter, r *http.Request) {
	playerID, ok := getUserID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req AddFriendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if req.FriendID == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing friend_id"})
		return
	}

	if err := h.service.RemoveFriend(r.Context(), playerID, req.FriendID); err != nil {
		slog.Error("remove friend failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) ListFriends(w http.ResponseWriter, r *http.Request) {
	playerID, ok := getUserID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	friends, err := h.service.ListFriends(r.Context(), playerID)
	if err != nil {
		slog.Error("list friends failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, FriendListResponse{Friends: friends})
}

func (h *Handler) Ping(w http.ResponseWriter, r *http.Request) {
	playerID, ok := getUserID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	if err := h.service.Ping(r.Context(), playerID); err != nil {
		slog.Error("ping failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
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
