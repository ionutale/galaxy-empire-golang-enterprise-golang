package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDispatchFleet_InvalidMission(t *testing.T) {
	svc := NewFleetService(newMockRepo(), "http://localhost:8082")
	_, err := svc.DispatchFleet(context.Background(), 1, DispatchRequest{
		OriginPlanetID: 1,
		Ships:          map[string]int{"cargo": 5},
		TargetGalaxy:   1, TargetSystem: 1, TargetPosition: 1,
		Mission: "invalid", SpeedPct: 100,
	})
	if err == nil || !strings.Contains(err.Error(), "invalid mission") {
		t.Fatalf("expected invalid mission error, got: %v", err)
	}
}

func TestDispatchFleet_NoShips(t *testing.T) {
	svc := NewFleetService(newMockRepo(), "http://localhost:8082")
	_, err := svc.DispatchFleet(context.Background(), 1, DispatchRequest{
		OriginPlanetID: 1,
		Ships:          map[string]int{},
		TargetGalaxy:   1, TargetSystem: 1, TargetPosition: 1,
		Mission: "transport", SpeedPct: 100,
	})
	if err == nil || !strings.Contains(err.Error(), "no ships") {
		t.Fatalf("expected no ships error, got: %v", err)
	}
}

func TestDispatchFleet_InvalidSpeed(t *testing.T) {
	svc := NewFleetService(newMockRepo(), "http://localhost:8082")
	_, err := svc.DispatchFleet(context.Background(), 1, DispatchRequest{
		OriginPlanetID: 1,
		Ships:          map[string]int{"cargo": 5},
		TargetGalaxy:   1, TargetSystem: 1, TargetPosition: 1,
		Mission: "transport", SpeedPct: 0,
	})
	if err == nil || !strings.Contains(err.Error(), "speed") {
		t.Fatalf("expected speed error, got: %v", err)
	}
}

func TestDispatchFleet_PlanetServiceUnreachable(t *testing.T) {
	svc := NewFleetService(newMockRepo(), "http://localhost:1")
	_, err := svc.DispatchFleet(context.Background(), 1, DispatchRequest{
		OriginPlanetID: 1,
		Ships:          map[string]int{"cargo": 5},
		TargetGalaxy:   1, TargetSystem: 1, TargetPosition: 1,
		Mission: "transport", SpeedPct: 100,
	})
	if err == nil || !strings.Contains(err.Error(), "planet service") {
		t.Fatalf("expected planet service error, got: %v", err)
	}
}

func TestDispatchFleet_PlanetServiceDenies(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "insufficient cargo ships"})
	}))
	defer ts.Close()

	svc := NewFleetService(newMockRepo(), ts.URL)
	_, err := svc.DispatchFleet(context.Background(), 1, DispatchRequest{
		OriginPlanetID: 1,
		Ships:          map[string]int{"cargo": 999},
		TargetGalaxy:   1, TargetSystem: 1, TargetPosition: 1,
		Mission: "transport", SpeedPct: 100,
	})
	if err == nil || !strings.Contains(err.Error(), "insufficient cargo ships") {
		t.Fatalf("expected insufficient ships error, got: %v", err)
	}
}

func TestDispatchFleet_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	}))
	defer ts.Close()

	svc := NewFleetService(newMockRepo(), ts.URL)
	fleet, err := svc.DispatchFleet(context.Background(), 1, DispatchRequest{
		OriginPlanetID: 1,
		Ships:          map[string]int{"cargo": 5},
		TargetGalaxy:   1, TargetSystem: 1, TargetPosition: 1,
		Mission: "transport", SpeedPct: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if fleet.PlayerID != 1 {
		t.Fatalf("expected player 1, got %d", fleet.PlayerID)
	}
	if fleet.Mission != "transport" {
		t.Fatalf("expected transport, got %s", fleet.Mission)
	}
	if fleet.Status != "stationed" {
		t.Fatalf("expected stationed, got %s", fleet.Status)
	}
	if fleet.Ships["cargo"] != 5 {
		t.Fatalf("expected 5 cargo, got %d", fleet.Ships["cargo"])
	}
}

func TestMyFleets_NoAuth(t *testing.T) {
	h := NewFleetHandler(NewFleetService(newMockRepo(), ""))
	req := httptest.NewRequest(http.MethodGet, "/api/fleet/my-fleets", nil)
	w := httptest.NewRecorder()
	h.MyFleets(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestMyFleets_Success(t *testing.T) {
	mock := newMockRepo()
	mock.fleets = []Fleet{
		{ID: 1, PlayerID: 1, OriginPlanetID: 1, TargetGalaxy: 1, TargetSystem: 1, TargetPosition: 1, Mission: "transport", Status: "stationed", SpeedPct: 100, Ships: map[string]int{"cargo": 5}},
	}
	svc := NewFleetService(mock, "")
	h := NewFleetHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/fleet/my-fleets", nil)
	req.Header.Set("X-User-ID", "1")
	w := httptest.NewRecorder()
	h.MyFleets(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp []FleetResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if len(resp) != 1 || resp[0].Mission != "transport" {
		t.Fatalf("expected 1 fleet with transport, got %+v", resp)
	}
}

func TestDispatch_NoAuth(t *testing.T) {
	h := NewFleetHandler(NewFleetService(newMockRepo(), ""))
	req := httptest.NewRequest(http.MethodPost, "/api/fleet/dispatch", nil)
	w := httptest.NewRecorder()
	h.Dispatch(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}
