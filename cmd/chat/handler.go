package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

type ChatHandler struct {
	service *ChatService
}

func NewHandler(service *ChatService) *ChatHandler {
	return &ChatHandler{service: service}
}

func (h *ChatHandler) Send(w http.ResponseWriter, r *http.Request) {
	playerID, ok := getUserID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	msg, err := h.service.SendMessage(r.Context(), playerID, req.Channel, req.Content)
	if err != nil {
		slog.Error("send message failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, SendMessageResponse{
		ID:        msg.ID,
		Channel:   msg.Channel,
		Content:   msg.Content,
		CreatedAt: msg.CreatedAt,
	})
}

func (h *ChatHandler) Messages(w http.ResponseWriter, r *http.Request) {
	playerID, ok := getUserID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	channel := r.URL.Query().Get("channel")
	if channel == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing channel query parameter"})
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	beforeID, _ := strconv.Atoi(r.URL.Query().Get("before_id"))

	messages, hasMore, err := h.service.GetMessages(r.Context(), playerID, channel, limit, beforeID)
	if err != nil {
		slog.Error("get messages failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, GetMessagesResponse{
		Messages: messages,
		HasMore:  hasMore,
	})
}

func (h *ChatHandler) Stream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	playerID, err := h.service.validateToken(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}

	channels, err := h.service.getUserChannels(r.Context(), playerID)
	if err != nil {
		slog.Error("get user channels failed", "error", err, "player_id", playerID)
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	listener := make(chan Message, 100)
	h.service.hub.Lock()
	h.service.hub.listeners = append(h.service.hub.listeners, listener)
	h.service.hub.Unlock()

	defer func() {
		h.service.hub.Lock()
		for i, l := range h.service.hub.listeners {
			if l == listener {
				h.service.hub.listeners = append(h.service.hub.listeners[:i], h.service.hub.listeners[i+1:]...)
				break
			}
		}
		h.service.hub.Unlock()
	}()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case msg := <-listener:
			if !h.service.isRelevantForPlayer(channels, msg) {
				continue
			}
			data, _ := json.Marshal(msg)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		case <-ticker.C:
			fmt.Fprintf(w, ": heartbeat\n\n")
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (h *ChatHandler) SendPrivate(w http.ResponseWriter, r *http.Request) {
	playerID, ok := getUserID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req SendPrivateMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	msg, err := h.service.SendPrivateMessage(r.Context(), playerID, req.ReceiverID, req.Content)
	if err != nil {
		slog.Error("send private message failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, msg)
}

func (h *ChatHandler) Inbox(w http.ResponseWriter, r *http.Request) {
	playerID, ok := getUserID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	beforeID, _ := strconv.Atoi(r.URL.Query().Get("before_id"))

	messages, hasMore, err := h.service.GetInbox(r.Context(), playerID, limit, beforeID)
	if err != nil {
		slog.Error("get inbox failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, PrivateMessagesResponse{
		Messages: messages,
		HasMore:  hasMore,
	})
}

func (h *ChatHandler) Outbox(w http.ResponseWriter, r *http.Request) {
	playerID, ok := getUserID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	beforeID, _ := strconv.Atoi(r.URL.Query().Get("before_id"))

	messages, hasMore, err := h.service.GetOutbox(r.Context(), playerID, limit, beforeID)
	if err != nil {
		slog.Error("get outbox failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, PrivateMessagesResponse{
		Messages: messages,
		HasMore:  hasMore,
	})
}

func (h *ChatHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	playerID, ok := getUserID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req struct {
		MessageID int `json:"message_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if err := h.service.MarkMessageRead(r.Context(), req.MessageID, playerID); err != nil {
		slog.Error("mark read failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *ChatHandler) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	playerID, ok := getUserID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	messageID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid message id"})
		return
	}

	if err := h.service.DeleteMessage(r.Context(), messageID, playerID); err != nil {
		slog.Error("delete message failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *ChatHandler) UnreadCount(w http.ResponseWriter, r *http.Request) {
	playerID, ok := getUserID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	count, err := h.service.GetUnreadCount(r.Context(), playerID)
	if err != nil {
		slog.Error("get unread count failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, InboxSummary{UnreadCount: count, TotalCount: count})
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
