package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func setupHandler() *Handler {
	svc := NewAuthService(&mockRepo{}, []byte("test-key"))
	return NewHandler(svc)
}

func setupRouter(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/register", h.Register)
		r.Post("/login", h.Login)
	})
	return r
}

func jsonBody(v any) *bytes.Buffer {
	var buf bytes.Buffer
	json.NewEncoder(&buf).Encode(v)
	return &buf
}

func TestHandlerRegister_Success(t *testing.T) {
	h := setupHandler()
	router := setupRouter(h)

	req := httptest.NewRequest("POST", "/api/auth/register", jsonBody(map[string]string{
		"email": "handler@example.com", "password": "password123",
	}))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp AuthResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal("decode response:", err)
	}
	if resp.Token == "" {
		t.Error("expected non-empty token")
	}
}

func TestHandlerRegister_Duplicate(t *testing.T) {
	h := setupHandler()
	router := setupRouter(h)

	body := jsonBody(map[string]string{"email": "dup2@example.com", "password": "password123"})
	req1 := httptest.NewRequest("POST", "/api/auth/register", body)
	req1.Header.Set("Content-Type", "application/json")
	rec1 := httptest.NewRecorder()
	router.ServeHTTP(rec1, req1)

	body2 := jsonBody(map[string]string{"email": "dup2@example.com", "password": "password123"})
	req2 := httptest.NewRequest("POST", "/api/auth/register", body2)
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", rec2.Code, rec2.Body.String())
	}
}

func TestHandlerLogin_Success(t *testing.T) {
	h := setupHandler()
	router := setupRouter(h)

	req := httptest.NewRequest("POST", "/api/auth/register", jsonBody(map[string]string{
		"email": "login-handler@example.com", "password": "password123",
	}))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register failed: %d", rec.Code)
	}

	req2 := httptest.NewRequest("POST", "/api/auth/login", jsonBody(map[string]string{
		"email": "login-handler@example.com", "password": "password123",
	}))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec2.Code, rec2.Body.String())
	}
	var resp AuthResponse
	json.NewDecoder(rec2.Body).Decode(&resp)
	if resp.Token == "" {
		t.Error("expected non-empty token")
	}
}

func TestHandlerLogin_WrongPassword(t *testing.T) {
	h := setupHandler()
	router := setupRouter(h)

	req := httptest.NewRequest("POST", "/api/auth/register", jsonBody(map[string]string{
		"email": "wrongpw-handler@example.com", "password": "password123",
	}))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register failed: %d", rec.Code)
	}

	req2 := httptest.NewRequest("POST", "/api/auth/login", jsonBody(map[string]string{
		"email": "wrongpw-handler@example.com", "password": "wrongpassword",
	}))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d: %s", rec2.Code, rec2.Body.String())
	}
}
