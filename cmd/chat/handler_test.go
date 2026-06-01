package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func setupTestRouter(h *ChatHandler) http.Handler {
	r := chi.NewRouter()
	r.Post("/api/chat/send", h.Send)
	r.Get("/api/chat/messages", h.Messages)
	r.Get("/api/chat/stream", h.Stream)
	return r
}

func TestSend_NoAuth(t *testing.T) {
	svc := NewChatService(newMockRepo(), "http://localhost:8087", "test-secret")
	h := NewHandler(svc)
	mux := setupTestRouter(h)

	body := bytes.NewReader([]byte(`{"channel":"global","content":"hello"}`))
	req := httptest.NewRequest("POST", "/api/chat/send", body)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestSend_InvalidAuth(t *testing.T) {
	svc := NewChatService(newMockRepo(), "http://localhost:8087", "test-secret")
	h := NewHandler(svc)
	mux := setupTestRouter(h)

	body := bytes.NewReader([]byte(`{"channel":"global","content":"hello"}`))
	req := httptest.NewRequest("POST", "/api/chat/send", body)
	req.Header.Set("X-User-ID", "notanumber")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestSend_InvalidJSON(t *testing.T) {
	svc := NewChatService(newMockRepo(), "http://localhost:8087", "test-secret")
	h := NewHandler(svc)
	mux := setupTestRouter(h)

	body := bytes.NewReader([]byte(`not json`))
	req := httptest.NewRequest("POST", "/api/chat/send", body)
	req.Header.Set("X-User-ID", "1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSend_Global_Success(t *testing.T) {
	svc := NewChatService(newMockRepo(), "http://localhost:8087", "test-secret")
	h := NewHandler(svc)
	mux := setupTestRouter(h)

	body := bytes.NewReader([]byte(`{"channel":"global","content":"hello world"}`))
	req := httptest.NewRequest("POST", "/api/chat/send", body)
	req.Header.Set("X-User-ID", "1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp SendMessageResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal("decode response:", err)
	}
	if resp.Content != "hello world" {
		t.Errorf("expected content 'hello world', got '%s'", resp.Content)
	}
	if resp.Channel != "global" {
		t.Errorf("expected channel global, got %s", resp.Channel)
	}
}

func TestMessages_NoAuth(t *testing.T) {
	svc := NewChatService(newMockRepo(), "http://localhost:8087", "test-secret")
	h := NewHandler(svc)
	mux := setupTestRouter(h)

	req := httptest.NewRequest("GET", "/api/chat/messages?channel=global", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestMessages_MissingChannel(t *testing.T) {
	svc := NewChatService(newMockRepo(), "http://localhost:8087", "test-secret")
	h := NewHandler(svc)
	mux := setupTestRouter(h)

	req := httptest.NewRequest("GET", "/api/chat/messages", nil)
	req.Header.Set("X-User-ID", "1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestMessages_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewChatService(repo, "http://localhost:8087", "test-secret")
	svc.SendMessage(nil, 1, "global", "test message")

	h := NewHandler(svc)
	mux := setupTestRouter(h)

	req := httptest.NewRequest("GET", "/api/chat/messages?channel=global", nil)
	req.Header.Set("X-User-ID", "2")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp GetMessagesResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal("decode response:", err)
	}
	if len(resp.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(resp.Messages))
	}
	if resp.Messages[0].Content != "test message" {
		t.Errorf("expected 'test message', got '%s'", resp.Messages[0].Content)
	}
}

func TestStream_NoToken(t *testing.T) {
	svc := NewChatService(newMockRepo(), "http://localhost:8087", "test-secret")
	h := NewHandler(svc)
	mux := setupTestRouter(h)

	req := httptest.NewRequest("GET", "/api/chat/stream", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestStream_InvalidToken(t *testing.T) {
	svc := NewChatService(newMockRepo(), "http://localhost:8087", "test-secret")
	h := NewHandler(svc)
	mux := setupTestRouter(h)

	req := httptest.NewRequest("GET", "/api/chat/stream?token=invalid", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

type slowResponseWriter struct {
	http.ResponseWriter
	flushed bool
}

func (s *slowResponseWriter) Flush() {
	s.flushed = true
}

func TestGetUserID(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		wantID   int
		wantOK   bool
	}{
		{"valid", "42", 42, true},
		{"empty", "", 0, false},
		{"invalid", "abc", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set("X-User-ID", tt.header)
			id, ok := getUserID(req)
			if id != tt.wantID || ok != tt.wantOK {
				t.Errorf("getUserID() = (%d, %v), want (%d, %v)", id, ok, tt.wantID, tt.wantOK)
			}
		})
	}
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusOK, map[string]string{"key": "value"})

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected json content type, got %s", w.Header().Get("Content-Type"))
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["key"] != "value" {
		t.Errorf("expected 'value', got '%s'", resp["key"])
	}
}

func TestHandlerUsesService(t *testing.T) {
	svc := NewChatService(newMockRepo(), "http://localhost:8087", "test-secret")
	h := NewHandler(svc)

	if h.service != svc {
		t.Error("handler should store reference to service")
	}
}
