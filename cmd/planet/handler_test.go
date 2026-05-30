package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func setupHandler() *Handler {
	svc := NewPlanetService(newMockRepo())
	return NewHandler(svc)
}

func setupRouter(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Get("/api/planet/mine", h.GetMyPlanet)
	return r
}

func TestGetMyPlanet_NoUserID(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/planet/mine", nil)
	rec := httptest.NewRecorder()
	setupRouter(setupHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestGetMyPlanet_WithUserID(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/planet/mine", nil)
	req.Header.Set("X-User-ID", "7")
	rec := httptest.NewRecorder()
	setupRouter(setupHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp PlanetResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal("decode response:", err)
	}
	if resp.UserID != 7 {
		t.Errorf("expected user_id 7, got %d", resp.UserID)
	}
	if len(resp.Buildings) != 4 {
		t.Errorf("expected 4 buildings, got %d", len(resp.Buildings))
	}
	if resp.Production.Metal <= 0 {
		t.Error("expected positive metal production")
	}
}

func TestGetMyPlanet_SameUserReturnsSamePlanet(t *testing.T) {
	h := setupHandler()
	router := setupRouter(h)

	rec1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("GET", "/api/planet/mine", nil)
	req1.Header.Set("X-User-ID", "10")
	router.ServeHTTP(rec1, req1)

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/api/planet/mine", nil)
	req2.Header.Set("X-User-ID", "10")
	router.ServeHTTP(rec2, req2)

	var p1, p2 PlanetResponse
	json.NewDecoder(rec1.Body).Decode(&p1)
	json.NewDecoder(rec2.Body).Decode(&p2)

	if p1.ID != p2.ID {
		t.Error("expected same planet for same user")
	}
}
