package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type jwtClaims struct {
	UserID int    `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

type SSEHub struct {
	sync.Mutex
	channels map[int][]chan Notification
}

func NewSSEHub() *SSEHub {
	return &SSEHub{channels: make(map[int][]chan Notification)}
}

func (h *SSEHub) Subscribe(playerID int) chan Notification {
	h.Lock()
	defer h.Unlock()
	ch := make(chan Notification, 100)
	h.channels[playerID] = append(h.channels[playerID], ch)
	return ch
}

func (h *SSEHub) Unsubscribe(playerID int, ch chan Notification) {
	h.Lock()
	defer h.Unlock()
	listeners := h.channels[playerID]
	for i, l := range listeners {
		if l == ch {
			h.channels[playerID] = append(listeners[:i], listeners[i+1:]...)
			break
		}
	}
	if len(h.channels[playerID]) == 0 {
		delete(h.channels, playerID)
	}
}

func (h *SSEHub) Publish(n Notification) {
	h.Lock()
	defer h.Unlock()
	listeners := h.channels[n.PlayerID]
	for _, ch := range listeners {
		select {
		case ch <- n:
		default:
		}
	}
}

type NotificationService struct {
	repo        Repository
	hub         *SSEHub
	jwtKey      []byte
	internalKey string
}

func NewNotificationService(repo Repository, jwtSecret, internalSecret string) *NotificationService {
	return &NotificationService{
		repo:        repo,
		hub:         NewSSEHub(),
		jwtKey:      []byte(jwtSecret),
		internalKey: internalSecret,
	}
}

func (s *NotificationService) CreateNotification(ctx context.Context, playerID int, category, title, message string) (Notification, error) {
	n, err := s.repo.CreateNotification(ctx, playerID, category, title, message)
	if err != nil {
		return Notification{}, err
	}
	s.hub.Publish(n)
	return n, nil
}

func (s *NotificationService) ListNotifications(ctx context.Context, playerID int, unreadOnly bool, limit, offset int) ([]Notification, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListNotifications(ctx, playerID, unreadOnly, limit, offset)
}

func (s *NotificationService) UnreadCount(ctx context.Context, playerID int) (int, error) {
	return s.repo.UnreadCount(ctx, playerID)
}

func (s *NotificationService) MarkRead(ctx context.Context, id, playerID int) error {
	return s.repo.MarkRead(ctx, id, playerID)
}

func (s *NotificationService) MarkAllRead(ctx context.Context, playerID int) error {
	return s.repo.MarkAllRead(ctx, playerID)
}

func (s *NotificationService) validateToken(r *http.Request) (int, error) {
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		return 0, fmt.Errorf("missing token query parameter")
	}

	claims := &jwtClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (any, error) {
		return s.jwtKey, nil
	})
	if err != nil || !token.Valid {
		return 0, fmt.Errorf("invalid or expired token")
	}

	return claims.UserID, nil
}

func (s *NotificationService) verifyInternalSecret(r *http.Request) bool {
	return r.Header.Get("X-Internal-Secret") == s.internalKey
}

type NotificationEvent struct {
	Type string       `json:"type"`
	Data Notification `json:"data"`
}

func (s *NotificationService) Stream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	playerID, err := s.validateToken(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	listener := s.hub.Subscribe(playerID)
	defer s.hub.Unsubscribe(playerID, listener)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case n := <-listener:
			event := NotificationEvent{Type: "notification", Data: n}
			data, _ := json.Marshal(event)
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

func (s *NotificationService) GetUserID(r *http.Request) (int, bool) {
	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		return 0, false
	}
	var userID int
	if _, err := fmt.Sscanf(userIDStr, "%d", &userID); err != nil {
		return 0, false
	}
	return userID, true
}
