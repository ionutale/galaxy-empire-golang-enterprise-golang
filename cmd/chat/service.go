package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type jwtClaims struct {
	UserID int    `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

type SubscriberManager struct {
	sync.Mutex
	listeners []chan Message
}

func (sm *SubscriberManager) Broadcast(msg Message) {
	sm.Lock()
	defer sm.Unlock()
	for _, ch := range sm.listeners {
		select {
		case ch <- msg:
		default:
		}
	}
}

type ChatService struct {
	repo        Repository
	hub         *SubscriberManager
	rateLimits  map[int]time.Time
	rateMu      sync.Mutex
	allianceURL string
	httpClient  *http.Client
	jwtKey      []byte
}

func NewChatService(repo Repository, allianceURL, jwtSecret string) *ChatService {
	return &ChatService{
		repo:        repo,
		hub:         &SubscriberManager{},
		rateLimits:  make(map[int]time.Time),
		allianceURL: allianceURL,
		httpClient:  &http.Client{Timeout: 10 * time.Second},
		jwtKey:      []byte(jwtSecret),
	}
}

func (s *ChatService) SendMessage(ctx context.Context, playerID int, channel, content string) (Message, error) {
	if channel != "global" && channel != "alliance" {
		return Message{}, fmt.Errorf("invalid channel: must be 'global' or 'alliance'")
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return Message{}, fmt.Errorf("content cannot be empty")
	}
	if len(content) > 500 {
		return Message{}, fmt.Errorf("content too long (max 500 characters)")
	}

	s.rateMu.Lock()
	lastTime, exists := s.rateLimits[playerID]
	s.rateLimits[playerID] = time.Now()
	s.rateMu.Unlock()

	if exists && time.Since(lastTime) < 2*time.Second {
		return Message{}, fmt.Errorf("rate limited: please wait before sending another message")
	}

	channelID := 0
	if channel == "alliance" {
		resp, err := s.checkAllianceMembership(ctx, playerID)
		if err != nil {
			return Message{}, fmt.Errorf("alliance check failed: %w", err)
		}
		if !resp.InAlliance {
			return Message{}, fmt.Errorf("you are not in an alliance")
		}
		channelID = resp.AllianceID
	}

	senderName := fmt.Sprintf("Player %d", playerID)

	msg, err := s.repo.CreateMessage(ctx, channel, channelID, playerID, senderName, content)
	if err != nil {
		slog.Error("create message failed", "error", err)
		return Message{}, fmt.Errorf("failed to send message: %w", err)
	}

	s.hub.Broadcast(msg)

	return msg, nil
}

func (s *ChatService) GetMessages(ctx context.Context, playerID int, channel string, limit, beforeID int) ([]Message, bool, error) {
	if channel != "global" && channel != "alliance" {
		return nil, false, fmt.Errorf("invalid channel: must be 'global' or 'alliance'")
	}

	if limit <= 0 || limit > 100 {
		limit = 50
	}

	channelID := 0
	if channel == "alliance" {
		resp, err := s.checkAllianceMembership(ctx, playerID)
		if err != nil {
			return nil, false, fmt.Errorf("alliance check failed: %w", err)
		}
		if !resp.InAlliance {
			return nil, false, fmt.Errorf("you are not in an alliance")
		}
		channelID = resp.AllianceID
	}

	return s.repo.GetMessages(ctx, channel, channelID, limit, beforeID)
}

func (s *ChatService) checkAllianceMembership(ctx context.Context, playerID int) (*PlayerAllianceResponse, error) {
	body, _ := json.Marshal(map[string]int{"player_id": playerID})
	resp, err := s.httpClient.Post(s.allianceURL+"/internal/alliance/player", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("alliance service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("alliance service error: %s", string(respBody))
	}

	var result PlayerAllianceResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode alliance response: %w", err)
	}

	return &result, nil
}

func (s *ChatService) validateToken(r *http.Request) (int, error) {
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		return 0, fmt.Errorf("missing token query parameter")
	}

	claims := &jwtClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtKey, nil
	})
	if err != nil || !token.Valid {
		return 0, fmt.Errorf("invalid or expired token")
	}

	return claims.UserID, nil
}

func (s *ChatService) getUserChannels(ctx context.Context, playerID int) ([]channelSub, error) {
	channels := []channelSub{{name: "global", channelID: 0}}

	resp, err := s.checkAllianceMembership(ctx, playerID)
	if err != nil {
		return channels, nil
	}
	if resp.InAlliance {
		channels = append(channels, channelSub{name: "alliance", channelID: resp.AllianceID})
	}

	return channels, nil
}

type channelSub struct {
	name      string
	channelID int
}

func (s *ChatService) isRelevantForPlayer(channels []channelSub, msg Message) bool {
	for _, ch := range channels {
		if msg.Channel == ch.name && msg.ChannelID == ch.channelID {
			return true
		}
	}
	return false
}

func (s *ChatService) SendPrivateMessage(ctx context.Context, senderID, receiverID int, content string) (PrivateMessage, error) {
	if senderID == receiverID {
		return PrivateMessage{}, fmt.Errorf("cannot send message to yourself")
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return PrivateMessage{}, fmt.Errorf("content cannot be empty")
	}
	if len(content) > 500 {
		return PrivateMessage{}, fmt.Errorf("content too long (max 500 characters)")
	}

	s.rateMu.Lock()
	lastTime, exists := s.rateLimits[senderID]
	s.rateLimits[senderID] = time.Now()
	s.rateMu.Unlock()

	if exists && time.Since(lastTime) < 2*time.Second {
		return PrivateMessage{}, fmt.Errorf("rate limited: please wait before sending another message")
	}

	msg, err := s.repo.CreatePrivateMessage(ctx, senderID, receiverID, content, false)
	if err != nil {
		slog.Error("create private message failed", "error", err)
		return PrivateMessage{}, fmt.Errorf("failed to send private message: %w", err)
	}

	return msg, nil
}

func (s *ChatService) SendSystemMessage(ctx context.Context, receiverID int, content string) (PrivateMessage, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return PrivateMessage{}, fmt.Errorf("content cannot be empty")
	}
	if len(content) > 500 {
		return PrivateMessage{}, fmt.Errorf("content too long (max 500 characters)")
	}

	msg, err := s.repo.CreatePrivateMessage(ctx, 0, receiverID, content, true)
	if err != nil {
		slog.Error("create system message failed", "error", err)
		return PrivateMessage{}, fmt.Errorf("failed to send system message: %w", err)
	}

	return msg, nil
}

func (s *ChatService) GetInbox(ctx context.Context, playerID int, limit, beforeID int) ([]PrivateMessage, bool, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.repo.GetInbox(ctx, playerID, limit, beforeID)
}

func (s *ChatService) GetOutbox(ctx context.Context, playerID int, limit, beforeID int) ([]PrivateMessage, bool, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.repo.GetOutbox(ctx, playerID, limit, beforeID)
}

func (s *ChatService) MarkMessageRead(ctx context.Context, messageID, playerID int) error {
	return s.repo.MarkMessageRead(ctx, messageID, playerID)
}

func (s *ChatService) DeleteMessage(ctx context.Context, messageID, playerID int) error {
	return s.repo.DeletePrivateMessage(ctx, messageID, playerID)
}

func (s *ChatService) GetUnreadCount(ctx context.Context, playerID int) (int, error) {
	return s.repo.GetUnreadCount(ctx, playerID)
}
