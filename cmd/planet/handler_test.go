package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func setupTestHandler() *Handler {
	return NewHandler(NewPlanetService(newMockRepo()))
}

func setupTestRouter(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Get("/api/planet/mine", h.GetMyPlanet)
	r.Post("/api/buildings/{type}/upgrade", h.StartUpgrade)
	r.Post("/api/buildings/{type}/cancel", h.CancelUpgrade)
	r.Post("/api/buildings/{type}/deconstruct", h.DeconstructBuilding)
	return r
}

func TestGetMyPlanet_NoUserID(t *testing.T) {
	router := setupTestRouter(setupTestHandler())
	req := httptest.NewRequest("GET", "/api/planet/mine", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestGetMyPlanet_WithUserID(t *testing.T) {
	router := setupTestRouter(setupTestHandler())
	req := httptest.NewRequest("GET", "/api/planet/mine", nil)
	req.Header.Set("X-User-ID", "7")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

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
	if len(resp.Buildings) != 12 {
		t.Errorf("expected 12 buildings, got %d", len(resp.Buildings))
	}
	if resp.Production.Metal <= 0 {
		t.Error("expected positive metal production")
	}
	if resp.Storage.Metal <= 0 {
		t.Error("expected positive metal storage")
	}
}

func TestGetMyPlanet_SameUserReturnsSamePlanet(t *testing.T) {
	h := setupTestHandler()
	router := setupTestRouter(h)

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

func TestStartUpgrade_NoUserID(t *testing.T) {
	router := setupTestRouter(setupTestHandler())
	req := httptest.NewRequest("POST", "/api/buildings/metal_mine/upgrade", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestStartUpgrade_Success(t *testing.T) {
	router := setupTestRouter(setupTestHandler())
	req := httptest.NewRequest("POST", "/api/buildings/metal_mine/upgrade", nil)
	req.Header.Set("X-User-ID", "1")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var entry QueueEntry
	if err := json.NewDecoder(rec.Body).Decode(&entry); err != nil {
		t.Fatal("decode queue entry:", err)
	}
	if entry.BuildingType != "metal_mine" {
		t.Errorf("expected metal_mine, got %s", entry.BuildingType)
	}
	if entry.TargetLevel != 2 {
		t.Errorf("expected target level 2, got %d", entry.TargetLevel)
	}
}

func TestStartUpgrade_InvalidBuilding(t *testing.T) {
	router := setupTestRouter(setupTestHandler())
	req := httptest.NewRequest("POST", "/api/buildings/invalid/upgrade", nil)
	req.Header.Set("X-User-ID", "1")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestStartUpgrade_AlreadyQueued(t *testing.T) {
	router := setupTestRouter(setupTestHandler())
	makeReq := func() *http.Response {
		req := httptest.NewRequest("POST", "/api/buildings/metal_mine/upgrade", nil)
		req.Header.Set("X-User-ID", "2")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec.Result()
	}

	resp1 := makeReq()
	if resp1.StatusCode != http.StatusOK {
		t.Errorf("first upgrade expected 200, got %d", resp1.StatusCode)
	}

	resp2 := makeReq()
	if resp2.StatusCode != http.StatusConflict {
		t.Errorf("second upgrade expected 409, got %d", resp2.StatusCode)
	}
}

func TestCancelUpgrade_Success(t *testing.T) {
	router := setupTestRouter(setupTestHandler())

	req1 := httptest.NewRequest("POST", "/api/buildings/metal_mine/upgrade", nil)
	req1.Header.Set("X-User-ID", "1")
	rec1 := httptest.NewRecorder()
	router.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("upgrade expected 200, got %d", rec1.Code)
	}

	req2 := httptest.NewRequest("POST", "/api/buildings/metal_mine/cancel", nil)
	req2.Header.Set("X-User-ID", "1")
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Errorf("cancel expected 200, got %d: %s", rec2.Code, rec2.Body.String())
	}
}

func TestCancelUpgrade_NoActiveUpgrade(t *testing.T) {
	router := setupTestRouter(setupTestHandler())
	req := httptest.NewRequest("POST", "/api/buildings/metal_mine/cancel", nil)
	req.Header.Set("X-User-ID", "1")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestDeconstructBuilding_Success(t *testing.T) {
	router := setupTestRouter(setupTestHandler())
	req := httptest.NewRequest("POST", "/api/buildings/metal_mine/deconstruct", nil)
	req.Header.Set("X-User-ID", "1")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var entry QueueEntry
	if err := json.NewDecoder(rec.Body).Decode(&entry); err != nil {
		t.Fatal("decode queue entry:", err)
	}
	if entry.Status != "deconstruct" {
		t.Errorf("expected status deconstruct, got %s", entry.Status)
	}
}

func TestDeconstructBuilding_NotFound(t *testing.T) {
	router := setupTestRouter(setupTestHandler())
	req := httptest.NewRequest("POST", "/api/buildings/nonexistent/deconstruct", nil)
	req.Header.Set("X-User-ID", "1")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestStartUpgrade_ProvidesQueueInPlanetResponse(t *testing.T) {
	h := setupTestHandler()
	router := setupTestRouter(h)

	upgradeReq := httptest.NewRequest("POST", "/api/buildings/crystal_mine/upgrade", nil)
	upgradeReq.Header.Set("X-User-ID", "3")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, upgradeReq)

	planetReq := httptest.NewRequest("GET", "/api/planet/mine", nil)
	planetReq.Header.Set("X-User-ID", "3")
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, planetReq)

	var resp PlanetResponse
	if err := json.NewDecoder(rec2.Body).Decode(&resp); err != nil {
		t.Fatal("decode:", err)
	}
	if len(resp.Queue) == 0 {
		t.Error("expected queue entries in planet response")
	}
	found := false
	for _, q := range resp.Queue {
		if q.BuildingType == "crystal_mine" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected crystal_mine in queue")
	}
}
