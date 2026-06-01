package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

func setupTestRouter(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Post("/api/radar/scan", h.Scan)
	r.Post("/api/radar/events", h.GetEvents)
	r.Post("/api/radar/events/resolve", h.ResolveEvent)
	r.Post("/api/radar/planet-status", h.PlanetStatus)
	r.Post("/api/radar/eu-scan", h.EUXScan)
	r.Post("/internal/radar/detect", h.InternalDetect)
	return r
}

func TestScan_NoAuth(t *testing.T) {
	svc := NewRadarService(newMockRepo(), "http://localhost:8082", "http://localhost:8083")
	h := NewHandler(svc)
	mux := setupTestRouter(h)
	req := httptest.NewRequest("POST", "/api/radar/scan", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestScan_InvalidAuth(t *testing.T) {
	svc := NewRadarService(newMockRepo(), "http://localhost:8082", "http://localhost:8083")
	h := NewHandler(svc)
	mux := setupTestRouter(h)
	req := httptest.NewRequest("POST", "/api/radar/scan", nil)
	req.Header.Set("X-User-ID", "notanumber")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestScan_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewRadarService(repo, "http://localhost:8082", "http://localhost:8083")
	h := NewHandler(svc)
	mux := setupTestRouter(h)

	src := 2
	fid := 100
	og, os, op := 1, 1, 1
	arrival := mustParseTime("2026-06-01T12:00:00Z")
	repo.CreateRadarEvent(nil, RadarEvent{
		PlayerID: 1, EventType: "incoming_attack", SourcePlayerID: &src, FleetID: &fid,
		TargetGalaxy: 1, TargetSystem: 2, TargetPosition: 3,
		OriginGalaxy: &og, OriginSystem: &os, OriginPosition: &op, ArrivalTime: &arrival,
	})

	req := httptest.NewRequest("POST", "/api/radar/scan", nil)
	req.Header.Set("X-User-ID", "1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []RadarEventResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal("decode response:", err)
	}
	if len(resp) != 1 {
		t.Errorf("expected 1 event, got %d", len(resp))
	}
	if resp[0].EventType != "incoming_attack" {
		t.Errorf("expected incoming_attack, got %s", resp[0].EventType)
	}
}

func TestGetEvents_NoAuth(t *testing.T) {
	svc := NewRadarService(newMockRepo(), "http://localhost:8082", "http://localhost:8083")
	h := NewHandler(svc)
	mux := setupTestRouter(h)
	req := httptest.NewRequest("POST", "/api/radar/events", bytes.NewReader([]byte(`{"scope":"my_planets"}`)))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestGetEvents_InvalidJSON(t *testing.T) {
	svc := NewRadarService(newMockRepo(), "http://localhost:8082", "http://localhost:8083")
	h := NewHandler(svc)
	mux := setupTestRouter(h)
	body := bytes.NewReader([]byte(`not json`))
	req := httptest.NewRequest("POST", "/api/radar/events", body)
	req.Header.Set("X-User-ID", "1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetEvents_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewRadarService(repo, "http://localhost:8082", "http://localhost:8083")
	h := NewHandler(svc)
	mux := setupTestRouter(h)

	src := 2
	fid := 100
	og, os, op := 1, 1, 1
	arrival := mustParseTime("2026-06-01T12:00:00Z")
	repo.CreateRadarEvent(nil, RadarEvent{
		PlayerID: 1, EventType: "espionage", SourcePlayerID: &src, FleetID: &fid,
		TargetGalaxy: 4, TargetSystem: 5, TargetPosition: 6,
		OriginGalaxy: &og, OriginSystem: &os, OriginPosition: &op, ArrivalTime: &arrival,
	})

	body := bytes.NewReader([]byte(`{"scope":"my_planets"}`))
	req := httptest.NewRequest("POST", "/api/radar/events", body)
	req.Header.Set("X-User-ID", "1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []RadarEventResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp) != 1 {
		t.Errorf("expected 1 event, got %d", len(resp))
	}
}

func TestResolveEvent_NoAuth(t *testing.T) {
	svc := NewRadarService(newMockRepo(), "http://localhost:8082", "http://localhost:8083")
	h := NewHandler(svc)
	mux := setupTestRouter(h)
	req := httptest.NewRequest("POST", "/api/radar/events/resolve", bytes.NewReader([]byte(`{"event_id":1}`)))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestResolveEvent_MissingID(t *testing.T) {
	svc := NewRadarService(newMockRepo(), "http://localhost:8082", "http://localhost:8083")
	h := NewHandler(svc)
	mux := setupTestRouter(h)
	body := bytes.NewReader([]byte(`{}`))
	req := httptest.NewRequest("POST", "/api/radar/events/resolve", body)
	req.Header.Set("X-User-ID", "1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestResolveEvent_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewRadarService(repo, "http://localhost:8082", "http://localhost:8083")
	h := NewHandler(svc)
	mux := setupTestRouter(h)

	src := 2
	fid := 100
	og, os, op := 1, 1, 1
	arrival := mustParseTime("2026-06-01T12:00:00Z")
	e, _ := repo.CreateRadarEvent(nil, RadarEvent{
		PlayerID: 1, EventType: "incoming_attack", SourcePlayerID: &src, FleetID: &fid,
		TargetGalaxy: 1, TargetSystem: 2, TargetPosition: 3,
		OriginGalaxy: &og, OriginSystem: &os, OriginPosition: &op, ArrivalTime: &arrival,
	})

	body := bytes.NewReader([]byte(`{"event_id":` + string(rune('0'+e.ID)) + `}`))
	body = bytes.NewReader([]byte(`{"event_id":1}`))
	req := httptest.NewRequest("POST", "/api/radar/events/resolve", body)
	req.Header.Set("X-User-ID", "1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPlanetStatus_NoAuth(t *testing.T) {
	svc := NewRadarService(newMockRepo(), "http://localhost:8082", "http://localhost:8083")
	h := NewHandler(svc)
	mux := setupTestRouter(h)
	req := httptest.NewRequest("POST", "/api/radar/planet-status", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestPlanetStatus_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewRadarService(repo, "http://localhost:8082", "http://localhost:8083")
	h := NewHandler(svc)
	mux := setupTestRouter(h)

	src := 2
	fid := 100
	og, os, op := 1, 1, 1
	arrival := mustParseTime("2026-06-01T12:00:00Z")
	repo.CreateRadarEvent(nil, RadarEvent{
		PlayerID: 1, EventType: "incoming_attack", SourcePlayerID: &src, FleetID: &fid,
		TargetGalaxy: 1, TargetSystem: 2, TargetPosition: 3,
		OriginGalaxy: &og, OriginSystem: &os, OriginPosition: &op, ArrivalTime: &arrival,
	})

	req := httptest.NewRequest("POST", "/api/radar/planet-status", nil)
	req.Header.Set("X-User-ID", "1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []PlanetStatusResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp) != 1 {
		t.Fatalf("expected 1 status, got %d", len(resp))
	}
	if resp[0].Status != "attack_incoming" {
		t.Errorf("expected attack_incoming, got %s", resp[0].Status)
	}
}

func TestEUXScan_NoAuth(t *testing.T) {
	svc := NewRadarService(newMockRepo(), "http://localhost:8082", "http://localhost:8083")
	h := NewHandler(svc)
	mux := setupTestRouter(h)
	body := bytes.NewReader([]byte(`{"target_galaxy":1,"target_system":10,"target_position":5}`))
	req := httptest.NewRequest("POST", "/api/radar/eu-scan", body)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestHandler_EUXScan_NoRadar(t *testing.T) {
	repo := newMockRepo()
	svc := NewRadarService(repo, "http://localhost:8082", "http://localhost:8083")
	h := NewHandler(svc)
	mux := setupTestRouter(h)

	body := bytes.NewReader([]byte(`{"target_galaxy":1,"target_system":10,"target_position":5}`))
	req := httptest.NewRequest("POST", "/api/radar/eu-scan", body)
	req.Header.Set("X-User-ID", "1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestInternalDetect_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewRadarService(repo, "http://localhost:8082", "http://localhost:8083")
	h := NewHandler(svc)
	mux := setupTestRouter(h)

	body := bytes.NewReader([]byte(`{
		"target_player_id":1,"source_player_id":2,"fleet_id":100,
		"target_galaxy":1,"target_system":2,"target_position":3,
		"origin_galaxy":4,"origin_system":5,"origin_position":6,
		"arrival_time":"2026-06-01T12:00:00Z","mission":"attack"
	}`))
	req := httptest.NewRequest("POST", "/internal/radar/detect", body)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestInternalDetect_MissingFields(t *testing.T) {
	svc := NewRadarService(newMockRepo(), "http://localhost:8082", "http://localhost:8083")
	h := NewHandler(svc)
	mux := setupTestRouter(h)

	body := bytes.NewReader([]byte(`{}`))
	req := httptest.NewRequest("POST", "/internal/radar/detect", body)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func mustParseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}
