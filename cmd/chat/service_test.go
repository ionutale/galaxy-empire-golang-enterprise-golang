package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func allianceServiceMock(handler func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/internal/alliance/player":
			handler(w, r)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestSendMessage_Global_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewChatService(repo, "http://localhost:8087", "test-secret")

	msg, err := svc.SendMessage(context.Background(), 1, "global", "Hello, world!")
	if err != nil {
		t.Fatal(err)
	}
	if msg.Channel != "global" {
		t.Errorf("expected channel global, got %s", msg.Channel)
	}
	if msg.Content != "Hello, world!" {
		t.Errorf("expected content 'Hello, world!', got '%s'", msg.Content)
	}
	if msg.SenderName != "Player 1" {
		t.Errorf("expected sender 'Player 1', got '%s'", msg.SenderName)
	}
	if msg.ID == 0 {
		t.Error("expected non-zero ID")
	}
}

func TestSendMessage_Global_TrimsContent(t *testing.T) {
	svc := NewChatService(newMockRepo(), "http://localhost:8087", "test-secret")

	msg, err := svc.SendMessage(context.Background(), 1, "global", "  hello  ")
	if err != nil {
		t.Fatal(err)
	}
	if msg.Content != "hello" {
		t.Errorf("expected trimmed 'hello', got '%s'", msg.Content)
	}
}

func TestSendMessage_EmptyContent(t *testing.T) {
	svc := NewChatService(newMockRepo(), "http://localhost:8087", "test-secret")

	_, err := svc.SendMessage(context.Background(), 1, "global", "   ")
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("expected empty content error, got: %v", err)
	}
}

func TestSendMessage_ContentTooLong(t *testing.T) {
	svc := NewChatService(newMockRepo(), "http://localhost:8087", "test-secret")

	longContent := strings.Repeat("a", 501)
	_, err := svc.SendMessage(context.Background(), 1, "global", longContent)
	if err == nil || !strings.Contains(err.Error(), "too long") {
		t.Fatalf("expected too long error, got: %v", err)
	}
}

func TestSendMessage_InvalidChannel(t *testing.T) {
	svc := NewChatService(newMockRepo(), "http://localhost:8087", "test-secret")

	_, err := svc.SendMessage(context.Background(), 1, "invalid", "hello")
	if err == nil || !strings.Contains(err.Error(), "invalid channel") {
		t.Fatalf("expected invalid channel error, got: %v", err)
	}
}

func TestSendMessage_Alliance_Success(t *testing.T) {
	ts := allianceServiceMock(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(PlayerAllianceResponse{
			InAlliance:   true,
			AllianceID:   42,
			Role:         "member",
			AllianceName: "Test Alliance",
			AllianceTag:  "TST",
		})
	})
	defer ts.Close()

	svc := NewChatService(newMockRepo(), ts.URL, "test-secret")

	msg, err := svc.SendMessage(context.Background(), 1, "alliance", "Hello allies!")
	if err != nil {
		t.Fatal(err)
	}
	if msg.Channel != "alliance" {
		t.Errorf("expected channel alliance, got %s", msg.Channel)
	}
	if msg.ChannelID != 42 {
		t.Errorf("expected channel_id 42, got %d", msg.ChannelID)
	}
}

func TestSendMessage_Alliance_NotMember(t *testing.T) {
	ts := allianceServiceMock(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(PlayerAllianceResponse{InAlliance: false})
	})
	defer ts.Close()

	svc := NewChatService(newMockRepo(), ts.URL, "test-secret")

	_, err := svc.SendMessage(context.Background(), 1, "alliance", "Hello allies!")
	if err == nil || !strings.Contains(err.Error(), "not in an alliance") {
		t.Fatalf("expected not in alliance error, got: %v", err)
	}
}

func TestSendMessage_RateLimited(t *testing.T) {
	svc := NewChatService(newMockRepo(), "http://localhost:8087", "test-secret")

	_, err := svc.SendMessage(context.Background(), 1, "global", "first message")
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.SendMessage(context.Background(), 1, "global", "second message")
	if err == nil || !strings.Contains(err.Error(), "rate limited") {
		t.Fatalf("expected rate limited error, got: %v", err)
	}
}

func TestSendMessage_DifferentPlayersNotRateLimited(t *testing.T) {
	svc := NewChatService(newMockRepo(), "http://localhost:8087", "test-secret")

	_, err := svc.SendMessage(context.Background(), 1, "global", "message from player 1")
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.SendMessage(context.Background(), 2, "global", "message from player 2")
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetMessages_Global(t *testing.T) {
	repo := newMockRepo()
	svc := NewChatService(repo, "http://localhost:8087", "test-secret")

	repo.CreateMessage(context.Background(), "global", 0, 1, "P1", "first")
	repo.CreateMessage(context.Background(), "global", 0, 2, "P2", "second")
	repo.CreateMessage(context.Background(), "global", 0, 3, "P3", "third")

	messages, hasMore, err := svc.GetMessages(context.Background(), 1, "global", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 3 {
		t.Errorf("expected 3 messages, got %d", len(messages))
	}
	if hasMore {
		t.Error("expected hasMore false")
	}
	if messages[0].Content != "first" {
		t.Errorf("expected first message 'first', got '%s'", messages[0].Content)
	}
	if messages[2].Content != "third" {
		t.Errorf("expected last message 'third', got '%s'", messages[2].Content)
	}
}

func TestGetMessages_WithLimit(t *testing.T) {
	repo := newMockRepo()
	svc := NewChatService(repo, "http://localhost:8087", "test-secret")

	for i := 0; i < 5; i++ {
		repo.CreateMessage(context.Background(), "global", 0, i+1, "Player", "message")
	}

	messages, hasMore, err := svc.GetMessages(context.Background(), 1, "global", 3, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 3 {
		t.Errorf("expected 3 messages, got %d", len(messages))
	}
	if !hasMore {
		t.Error("expected hasMore true")
	}
}

func TestGetMessages_Alliance_NotMember(t *testing.T) {
	ts := allianceServiceMock(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(PlayerAllianceResponse{InAlliance: false})
	})
	defer ts.Close()

	svc := NewChatService(newMockRepo(), ts.URL, "test-secret")

	_, _, err := svc.GetMessages(context.Background(), 1, "alliance", 10, 0)
	if err == nil || !strings.Contains(err.Error(), "not in an alliance") {
		t.Fatalf("expected not in alliance error, got: %v", err)
	}
}

func TestGetMessages_InvalidChannel(t *testing.T) {
	svc := NewChatService(newMockRepo(), "http://localhost:8087", "test-secret")

	_, _, err := svc.GetMessages(context.Background(), 1, "invalid", 10, 0)
	if err == nil || !strings.Contains(err.Error(), "invalid channel") {
		t.Fatalf("expected invalid channel error, got: %v", err)
	}
}

func TestGetMessages_DefaultLimit(t *testing.T) {
	svc := NewChatService(newMockRepo(), "http://localhost:8087", "test-secret")

	messages, hasMore, err := svc.GetMessages(context.Background(), 1, "global", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 0 {
		t.Errorf("expected 0 messages, got %d", len(messages))
	}
	if hasMore {
		t.Error("expected hasMore false for empty")
	}
}

func TestSubscriberManager_Broadcast(t *testing.T) {
	sm := &SubscriberManager{}
	ch := make(chan Message, 10)
	sm.listeners = append(sm.listeners, ch)

	msg := Message{ID: 1, Content: "test", Channel: "global"}
	sm.Broadcast(msg)

	select {
	case received := <-ch:
		if received.ID != 1 {
			t.Errorf("expected ID 1, got %d", received.ID)
		}
	default:
		t.Error("expected message on channel")
	}
}

func TestSubscriberManager_BroadcastMultiple(t *testing.T) {
	sm := &SubscriberManager{}
	ch1 := make(chan Message, 10)
	ch2 := make(chan Message, 10)
	sm.listeners = append(sm.listeners, ch1, ch2)

	msg := Message{ID: 1, Content: "broadcast"}
	sm.Broadcast(msg)

	if len(ch1) != 1 {
		t.Errorf("expected 1 message on ch1, got %d", len(ch1))
	}
	if len(ch2) != 1 {
		t.Errorf("expected 1 message on ch2, got %d", len(ch2))
	}
}

func TestSubscriberManager_DropsOnFullBuffer(t *testing.T) {
	sm := &SubscriberManager{}
	ch := make(chan Message, 1)
	sm.listeners = append(sm.listeners, ch)

	sm.Broadcast(Message{ID: 1})
	sm.Broadcast(Message{ID: 2})

	if len(ch) != 1 {
		t.Errorf("expected 1 message (dropped one), got %d", len(ch))
	}
}

func TestIsRelevantForPlayer(t *testing.T) {
	svc := NewChatService(newMockRepo(), "http://localhost:8087", "test-secret")

	channels := []channelSub{
		{name: "global", channelID: 0},
		{name: "alliance", channelID: 42},
	}

	if !svc.isRelevantForPlayer(channels, Message{Channel: "global", ChannelID: 0}) {
		t.Error("expected global to be relevant")
	}
	if !svc.isRelevantForPlayer(channels, Message{Channel: "alliance", ChannelID: 42}) {
		t.Error("expected alliance 42 to be relevant")
	}
	if svc.isRelevantForPlayer(channels, Message{Channel: "alliance", ChannelID: 99}) {
		t.Error("expected alliance 99 to NOT be relevant")
	}
	if svc.isRelevantForPlayer(channels, Message{Channel: "other", ChannelID: 0}) {
		t.Error("expected other to NOT be relevant")
	}
}

func TestBroadcast_Distribution(t *testing.T) {
	repo := newMockRepo()
	svc := NewChatService(repo, "http://localhost:8087", "test-secret")

	listener := make(chan Message, 10)
	svc.hub.Lock()
	svc.hub.listeners = append(svc.hub.listeners, listener)
	svc.hub.Unlock()

	svc.SendMessage(context.Background(), 1, "global", "broadcast test")

	select {
	case msg := <-listener:
		if msg.Content != "broadcast test" {
			t.Errorf("expected 'broadcast test', got '%s'", msg.Content)
		}
	case <-time.After(time.Second):
		t.Error("timed out waiting for broadcast")
	}
}

func TestConcurrentRateLimit(t *testing.T) {
	svc := NewChatService(newMockRepo(), "http://localhost:8087", "test-secret")

	var wg sync.WaitGroup
	errs := make(chan error, 10)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_, err := svc.SendMessage(context.Background(), id, "global", "test")
			errs <- err
		}(i)
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}
