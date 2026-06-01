package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestService(planetHandler func(w http.ResponseWriter, r *http.Request)) (*CombatService, *httptest.Server) {
	ts := httptest.NewServer(http.HandlerFunc(planetHandler))
	svc := NewCombatService(newMockRepo(), ts.URL)
	return svc, ts
}

func TestResolve_AttackerWins(t *testing.T) {
	svc, ts := newTestService(func(w http.ResponseWriter, r *http.Request) {
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
	})
	defer ts.Close()

	resp, err := svc.Resolve(context.Background(), resolveRequest{
		FleetID:       1,
		AttackerID:    1,
		OriginPlanet:  5,
		AttackerShips: map[string]int{"light_fighter": 50},
		TargetGalaxy:  1,
		TargetSystem:  1,
		TargetPos:     3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.AttackerWon {
		t.Error("attacker should win with 50 vs 5")
	}
	if resp.ReportID == 0 {
		t.Error("expected report ID")
	}
	if resp.Rounds == 0 {
		t.Error("expected at least 1 round")
	}
}

func TestResolve_DefenderWins(t *testing.T) {
	svc, ts := newTestService(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/internal/planet/info":
			json.NewEncoder(w).Encode(planetInfoResponse{
				PlanetID: 10,
				PlayerID: 2,
				Metal:    10000,
				Crystal:  5000,
				Gas:      2000,
				Ships:    map[string]int{"light_fighter": 100},
			})
		case "/internal/ships/deduct":
			json.NewEncoder(w).Encode(map[string]bool{"ok": true})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	defer ts.Close()

	resp, err := svc.Resolve(context.Background(), resolveRequest{
		FleetID:       1,
		AttackerID:    1,
		OriginPlanet:  5,
		AttackerShips: map[string]int{"light_fighter": 5},
		TargetGalaxy:  1,
		TargetSystem:  1,
		TargetPos:     3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.AttackerWon {
		t.Error("defender should win")
	}
}

func TestResolve_PlanetNotFound(t *testing.T) {
	svc, ts := newTestService(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "planet not found"})
	})
	defer ts.Close()

	_, err := svc.Resolve(context.Background(), resolveRequest{
		FleetID:       1,
		AttackerID:    1,
		OriginPlanet:  5,
		AttackerShips: map[string]int{"light_fighter": 10},
		TargetGalaxy:  99,
		TargetSystem:  99,
		TargetPos:     99,
	})
	if err == nil {
		t.Fatal("expected error for planet not found")
	}
}

func TestResolve_EmptyDefender(t *testing.T) {
	svc, ts := newTestService(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/internal/planet/info":
			json.NewEncoder(w).Encode(planetInfoResponse{
				PlanetID: 10,
				PlayerID: 2,
				Metal:    5000,
				Crystal:  3000,
				Gas:      1000,
				Ships:    map[string]int{},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	defer ts.Close()

	resp, err := svc.Resolve(context.Background(), resolveRequest{
		FleetID:       1,
		AttackerID:    1,
		OriginPlanet:  5,
		AttackerShips: map[string]int{"cargo": 5},
		TargetGalaxy:  1,
		TargetSystem:  1,
		TargetPos:     3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.AttackerWon {
		t.Error("attacker should win with empty defender")
	}
	if resp.AttackerLoot["metal"] != 2500 {
		t.Errorf("expected 2500 metal loot (min(5000/2, cargo=125000)), got %d", resp.AttackerLoot["metal"])
	}
}
