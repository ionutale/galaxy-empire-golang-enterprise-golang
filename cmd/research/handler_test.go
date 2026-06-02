package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func setupTestHandler() *Handler {
	return NewHandler(newTestService())
}

func setupTestRouter(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Get("/api/research", h.ListTechs)
	r.Post("/api/research/{type}/start", h.StartResearch)
	r.Post("/api/research/{type}/cancel", h.CancelResearch)
	r.Get("/api/research/queue", h.ListQueue)
	return r
}

func TestListTechs_NoAuth(t *testing.T) {
	router := setupTestRouter(setupTestHandler())
	req := httptest.NewRequest("GET", "/api/research", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestListTechs_Success(t *testing.T) {
	router := setupTestRouter(setupTestHandler())
	req := httptest.NewRequest("GET", "/api/research", nil)
	req.Header.Set("X-User-ID", "1")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal("decode:", err)
	}

	techs, ok := resp["techs"].([]any)
	if !ok {
		t.Fatal("expected techs array")
	}
	if len(techs) != 19 {
		t.Errorf("expected 19 techs, got %d", len(techs))
	}

	labLevel, ok := resp["lab_level"].(float64)
	if !ok || labLevel < 1 {
		t.Errorf("expected lab_level >= 1, got %v", labLevel)
	}
}

func TestStartResearch_NoAuth(t *testing.T) {
	router := setupTestRouter(setupTestHandler())
	body := `{"planet_id":1}`
	req := httptest.NewRequest("POST", "/api/research/energy_tech/start", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestStartResearch_Success(t *testing.T) {
	router := setupTestRouter(setupTestHandler())
	body := `{"planet_id":1}`
	req := httptest.NewRequest("POST", "/api/research/energy_tech/start", strings.NewReader(body))
	req.Header.Set("X-User-ID", "1")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp StartResearchResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal("decode:", err)
	}
	if resp.TechType != "energy_tech" {
		t.Errorf("expected energy_tech, got %s", resp.TechType)
	}
	if resp.TargetLevel != 1 {
		t.Errorf("expected target level 1, got %d", resp.TargetLevel)
	}
}

func TestHandler_StartResearch_InvalidTech(t *testing.T) {
	router := setupTestRouter(setupTestHandler())
	body := `{"planet_id":1}`
	req := httptest.NewRequest("POST", "/api/research/invalid/start", strings.NewReader(body))
	req.Header.Set("X-User-ID", "1")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestStartResearch_MissingPlanetID(t *testing.T) {
	router := setupTestRouter(setupTestHandler())
	body := `{}`
	req := httptest.NewRequest("POST", "/api/research/energy_tech/start", strings.NewReader(body))
	req.Header.Set("X-User-ID", "1")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_StartResearch_AlreadyResearching(t *testing.T) {
	router := setupTestRouter(setupTestHandler())

	body := `{"planet_id":1}`
	req1 := httptest.NewRequest("POST", "/api/research/energy_tech/start", strings.NewReader(body))
	req1.Header.Set("X-User-ID", "1")
	req1.Header.Set("Content-Type", "application/json")
	rec1 := httptest.NewRecorder()
	router.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("first: expected 200, got %d", rec1.Code)
	}

	req2 := httptest.NewRequest("POST", "/api/research/energy_tech/start", strings.NewReader(body))
	req2.Header.Set("X-User-ID", "1")
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", rec2.Code, rec2.Body.String())
	}
}

func TestCancelResearch_NoAuth(t *testing.T) {
	router := setupTestRouter(setupTestHandler())
	req := httptest.NewRequest("POST", "/api/research/energy_tech/cancel", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestHandler_CancelResearch_Success(t *testing.T) {
	router := setupTestRouter(setupTestHandler())

	body := `{"planet_id":1}`
	startReq := httptest.NewRequest("POST", "/api/research/energy_tech/start", strings.NewReader(body))
	startReq.Header.Set("X-User-ID", "1")
	startReq.Header.Set("Content-Type", "application/json")
	startRec := httptest.NewRecorder()
	router.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("start expected 200, got %d", startRec.Code)
	}

	cancelReq := httptest.NewRequest("POST", "/api/research/energy_tech/cancel", nil)
	cancelReq.Header.Set("X-User-ID", "1")
	cancelRec := httptest.NewRecorder()
	router.ServeHTTP(cancelRec, cancelReq)
	if cancelRec.Code != http.StatusOK {
		t.Errorf("cancel expected 200, got %d: %s", cancelRec.Code, cancelRec.Body.String())
	}

	var resp CancelResearchResponse
	if err := json.NewDecoder(cancelRec.Body).Decode(&resp); err != nil {
		t.Fatal("decode:", err)
	}
	if resp.RefundMetal <= 0 {
		t.Error("expected positive metal refund")
	}
}

func TestHandler_CancelResearch_NoActive(t *testing.T) {
	router := setupTestRouter(setupTestHandler())
	req := httptest.NewRequest("POST", "/api/research/energy_tech/cancel", nil)
	req.Header.Set("X-User-ID", "1")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestListQueue_Success(t *testing.T) {
	router := setupTestRouter(setupTestHandler())

	body := `{"planet_id":1}`
	startReq := httptest.NewRequest("POST", "/api/research/energy_tech/start", strings.NewReader(body))
	startReq.Header.Set("X-User-ID", "1")
	startReq.Header.Set("Content-Type", "application/json")
	startRec := httptest.NewRecorder()
	router.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("start expected 200, got %d", startRec.Code)
	}

	queueReq := httptest.NewRequest("GET", "/api/research/queue", nil)
	queueReq.Header.Set("X-User-ID", "1")
	queueRec := httptest.NewRecorder()
	router.ServeHTTP(queueRec, queueReq)
	if queueRec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", queueRec.Code, queueRec.Body.String())
	}

	var queue []ResearchQueue
	if err := json.NewDecoder(queueRec.Body).Decode(&queue); err != nil {
		t.Fatal("decode:", err)
	}
	if len(queue) < 1 {
		t.Error("expected at least 1 queue entry")
	}
	if queue[0].TechType != "energy_tech" {
		t.Errorf("expected energy_tech, got %s", queue[0].TechType)
	}
}
