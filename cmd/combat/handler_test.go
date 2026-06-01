package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func setupTestRouter(svc *CombatService) http.Handler {
	h := NewCombatHandler(svc)
	r := chi.NewRouter()
	r.Post("/combat/resolve", h.Resolve)
	r.Post("/combat/moon-info", h.MoonInfo)
	r.Get("/combat/reports/{id}", h.GetReport)
	r.Get("/combat/reports/by-player", h.ListPlayerReports)
	return r
}

func mockPlanetHandler(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/internal/planet/info":
		json.NewEncoder(w).Encode(planetInfoResponse{
			PlanetID: 10,
			PlayerID: 2,
			Metal:    10000,
			Crystal:  5000,
			Gas:      2000,
			Ships:    map[string]int{"light_fighter": 5},
		})
	case "/internal/ships/deduct":
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	case "/internal/resources/add":
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	case "/internal/resources/deduct":
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func TestHandler_Resolve_Success(t *testing.T) {
	planetTS := httptest.NewServer(http.HandlerFunc(mockPlanetHandler))
	defer planetTS.Close()

	svc := NewCombatService(newMockRepo(), planetTS.URL)
	router := setupTestRouter(svc)

	body, _ := json.Marshal(resolveRequest{
		FleetID:       1,
		AttackerID:    1,
		OriginPlanet:  5,
		AttackerShips: map[string]int{"light_fighter": 50},
		TargetGalaxy:  1,
		TargetSystem:  1,
		TargetPos:     3,
	})

	req := httptest.NewRequest("POST", "/combat/resolve", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp resolveResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if !resp.AttackerWon {
		t.Error("attacker should win")
	}
	if resp.ReportID == 0 {
		t.Error("expected report ID")
	}
}

func TestHandler_Resolve_MissingAttackerID(t *testing.T) {
	planetTS := httptest.NewServer(http.HandlerFunc(mockPlanetHandler))
	defer planetTS.Close()

	svc := NewCombatService(newMockRepo(), planetTS.URL)
	router := setupTestRouter(svc)

	body, _ := json.Marshal(map[string]any{
		"attacker_ships":  map[string]int{"light_fighter": 10},
		"target_galaxy":   1,
		"target_system":   1,
		"target_position": 3,
	})

	req := httptest.NewRequest("POST", "/combat/resolve", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandler_Resolve_MissingShips(t *testing.T) {
	planetTS := httptest.NewServer(http.HandlerFunc(mockPlanetHandler))
	defer planetTS.Close()

	svc := NewCombatService(newMockRepo(), planetTS.URL)
	router := setupTestRouter(svc)

	body, _ := json.Marshal(map[string]any{
		"attacker_player_id": 1,
		"attacker_ships":     map[string]int{},
		"target_galaxy":      1,
		"target_system":      1,
		"target_position":    3,
	})

	req := httptest.NewRequest("POST", "/combat/resolve", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandler_GetReport_Success(t *testing.T) {
	planetTS := httptest.NewServer(http.HandlerFunc(mockPlanetHandler))
	defer planetTS.Close()

	svc := NewCombatService(newMockRepo(), planetTS.URL)
	router := setupTestRouter(svc)

	body, _ := json.Marshal(resolveRequest{
		FleetID:       1,
		AttackerID:    1,
		OriginPlanet:  5,
		AttackerShips: map[string]int{"light_fighter": 50},
		TargetGalaxy:  1,
		TargetSystem:  1,
		TargetPos:     3,
	})

	req := httptest.NewRequest("POST", "/combat/resolve", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var resolveResp resolveResponse
	json.NewDecoder(rec.Body).Decode(&resolveResp)

	reqGet := httptest.NewRequest("GET", fmt.Sprintf("/combat/reports/%d", resolveResp.ReportID), nil)
	recGet := httptest.NewRecorder()
	router.ServeHTTP(recGet, reqGet)

	if recGet.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", recGet.Code)
	}

	var report CombatReport
	if err := json.NewDecoder(recGet.Body).Decode(&report); err != nil {
		t.Fatal(err)
	}
	if report.ID != resolveResp.ReportID {
		t.Errorf("report ID: got %d, want %d", report.ID, resolveResp.ReportID)
	}
	if report.ExpiresAt.IsZero() {
		t.Error("expires_at should be set")
	}
}

func TestHandler_GetReport_NotFound(t *testing.T) {
	planetTS := httptest.NewServer(http.HandlerFunc(mockPlanetHandler))
	defer planetTS.Close()

	svc := NewCombatService(newMockRepo(), planetTS.URL)
	router := setupTestRouter(svc)

	req := httptest.NewRequest("GET", "/combat/reports/999", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestHandler_ListPlayerReports(t *testing.T) {
	planetTS := httptest.NewServer(http.HandlerFunc(mockPlanetHandler))
	defer planetTS.Close()

	svc := NewCombatService(newMockRepo(), planetTS.URL)
	router := setupTestRouter(svc)

	body, _ := json.Marshal(resolveRequest{
		FleetID:       1,
		AttackerID:    1,
		OriginPlanet:  5,
		AttackerShips: map[string]int{"light_fighter": 50},
		TargetGalaxy:  1,
		TargetSystem:  1,
		TargetPos:     3,
	})

	req := httptest.NewRequest("POST", "/combat/resolve", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	reqList := httptest.NewRequest("GET", "/combat/reports/by-player?player_id=1", nil)
	recList := httptest.NewRecorder()
	router.ServeHTTP(recList, reqList)

	if recList.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", recList.Code)
	}

	var summaries []map[string]any
	if err := json.NewDecoder(recList.Body).Decode(&summaries); err != nil {
		t.Fatal(err)
	}
	if len(summaries) == 0 {
		t.Error("expected at least 1 report summary")
	}
}

func TestHandler_MoonInfo_NotFound(t *testing.T) {
	planetTS := httptest.NewServer(http.HandlerFunc(mockPlanetHandler))
	defer planetTS.Close()

	svc := NewCombatService(newMockRepo(), planetTS.URL)
	router := setupTestRouter(svc)

	body, _ := json.Marshal(moonInfoRequest{
		Galaxy: 1, System: 1, Position: 1,
	})
	req := httptest.NewRequest("POST", "/combat/moon-info", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}
