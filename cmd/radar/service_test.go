package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func fleetServiceMock(handler func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/internal/fleet/at-location":
			handler(w, r)
		default:
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
		}
	}))
}

func TestScan_ReturnsUnresolvedEvents(t *testing.T) {
	repo := newMockRepo()
	svc := NewRadarService(repo, "http://localhost:8082", "http://localhost:8083")

	src := 2
	fid := 100
	og, os, op := 1, 1, 1
	arrival := time.Now().Add(1 * time.Hour)

	repo.CreateRadarEvent(context.Background(), RadarEvent{
		PlayerID: 1, EventType: "incoming_attack", SourcePlayerID: &src, FleetID: &fid,
		TargetGalaxy: 1, TargetSystem: 2, TargetPosition: 3,
		OriginGalaxy: &og, OriginSystem: &os, OriginPosition: &op, ArrivalTime: &arrival,
	})
	repo.CreateRadarEvent(context.Background(), RadarEvent{
		PlayerID: 1, EventType: "espionage", SourcePlayerID: &src, FleetID: &fid,
		TargetGalaxy: 4, TargetSystem: 5, TargetPosition: 6,
		OriginGalaxy: &og, OriginSystem: &os, OriginPosition: &op, ArrivalTime: &arrival,
	})

	events, err := svc.Scan(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Errorf("expected 2 events, got %d", len(events))
	}
}

func TestScan_OnlyReturnsUnresolved(t *testing.T) {
	repo := newMockRepo()
	svc := NewRadarService(repo, "http://localhost:8082", "http://localhost:8083")

	src := 2
	fid := 100
	og, os, op := 1, 1, 1
	arrival := time.Now().Add(1 * time.Hour)

	e, _ := repo.CreateRadarEvent(context.Background(), RadarEvent{
		PlayerID: 1, EventType: "incoming_attack", SourcePlayerID: &src, FleetID: &fid,
		TargetGalaxy: 1, TargetSystem: 2, TargetPosition: 3,
		OriginGalaxy: &og, OriginSystem: &os, OriginPosition: &op, ArrivalTime: &arrival,
	})
	repo.ResolveEvent(context.Background(), e.ID)

	events, err := svc.Scan(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 unresolved events, got %d", len(events))
	}
}

func TestGetEvents_ReturnsAll(t *testing.T) {
	repo := newMockRepo()
	svc := NewRadarService(repo, "http://localhost:8082", "http://localhost:8083")

	src := 2
	fid := 100
	og, os, op := 1, 1, 1
	arrival := time.Now().Add(1 * time.Hour)

	repo.CreateRadarEvent(context.Background(), RadarEvent{
		PlayerID: 1, EventType: "incoming_attack", SourcePlayerID: &src, FleetID: &fid,
		TargetGalaxy: 1, TargetSystem: 2, TargetPosition: 3,
		OriginGalaxy: &og, OriginSystem: &os, OriginPosition: &op, ArrivalTime: &arrival,
	})
	e, _ := repo.CreateRadarEvent(context.Background(), RadarEvent{
		PlayerID: 1, EventType: "espionage", SourcePlayerID: &src, FleetID: &fid,
		TargetGalaxy: 4, TargetSystem: 5, TargetPosition: 6,
		OriginGalaxy: &og, OriginSystem: &os, OriginPosition: &op, ArrivalTime: &arrival,
	})
	repo.ResolveEvent(context.Background(), e.ID)

	events, err := svc.GetEvents(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Errorf("expected 2 events (all), got %d", len(events))
	}
}

func TestResolveEvent_OnlyOwnEvent(t *testing.T) {
	repo := newMockRepo()
	svc := NewRadarService(repo, "http://localhost:8082", "http://localhost:8083")

	src := 2
	fid := 100
	og, os, op := 1, 1, 1
	arrival := time.Now().Add(1 * time.Hour)

	e, _ := repo.CreateRadarEvent(context.Background(), RadarEvent{
		PlayerID: 1, EventType: "incoming_attack", SourcePlayerID: &src, FleetID: &fid,
		TargetGalaxy: 1, TargetSystem: 2, TargetPosition: 3,
		OriginGalaxy: &og, OriginSystem: &os, OriginPosition: &op, ArrivalTime: &arrival,
	})

	if err := svc.ResolveEvent(context.Background(), 2, e.ID); err == nil {
		t.Error("expected error resolving another player's event")
	}
}

func TestPlanetStatus_Secure(t *testing.T) {
	repo := newMockRepo()
	svc := NewRadarService(repo, "http://localhost:8082", "http://localhost:8083")

	statuses, err := svc.PlanetStatus(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(statuses) != 0 {
		t.Errorf("expected 0 statuses for no events, got %d", len(statuses))
	}
}

func TestPlanetStatus_AttackIncoming(t *testing.T) {
	repo := newMockRepo()
	svc := NewRadarService(repo, "http://localhost:8082", "http://localhost:8083")

	src := 2
	fid := 100
	og, os, op := 1, 1, 1
	arrival := time.Now().Add(1 * time.Hour)

	repo.CreateRadarEvent(context.Background(), RadarEvent{
		PlayerID: 1, EventType: "incoming_attack", SourcePlayerID: &src, FleetID: &fid,
		TargetGalaxy: 1, TargetSystem: 2, TargetPosition: 3,
		OriginGalaxy: &og, OriginSystem: &os, OriginPosition: &op, ArrivalTime: &arrival,
	})

	statuses, err := svc.PlanetStatus(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(statuses) != 1 {
		t.Fatalf("expected 1 status, got %d", len(statuses))
	}
	if statuses[0].Status != "attack_incoming" {
		t.Errorf("expected attack_incoming, got %s", statuses[0].Status)
	}
	if statuses[0].FleetCount != 1 {
		t.Errorf("expected fleet_count 1, got %d", statuses[0].FleetCount)
	}
}

func TestPlanetStatus_Espionaged(t *testing.T) {
	repo := newMockRepo()
	svc := NewRadarService(repo, "http://localhost:8082", "http://localhost:8083")

	src := 2
	fid := 100
	og, os, op := 1, 1, 1
	arrival := time.Now().Add(1 * time.Hour)

	repo.CreateRadarEvent(context.Background(), RadarEvent{
		PlayerID: 1, EventType: "espionage", SourcePlayerID: &src, FleetID: &fid,
		TargetGalaxy: 1, TargetSystem: 2, TargetPosition: 3,
		OriginGalaxy: &og, OriginSystem: &os, OriginPosition: &op, ArrivalTime: &arrival,
	})

	statuses, err := svc.PlanetStatus(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(statuses) != 1 {
		t.Fatalf("expected 1 status, got %d", len(statuses))
	}
	if statuses[0].Status != "espionaged" {
		t.Errorf("expected espionaged, got %s", statuses[0].Status)
	}
}

func TestPlanetStatus_AttackTakesPriority(t *testing.T) {
	repo := newMockRepo()
	svc := NewRadarService(repo, "http://localhost:8082", "http://localhost:8083")

	src := 2
	fid := 100
	og, os, op := 1, 1, 1
	arrival := time.Now().Add(1 * time.Hour)

	repo.CreateRadarEvent(context.Background(), RadarEvent{
		PlayerID: 1, EventType: "espionage", SourcePlayerID: &src, FleetID: &fid,
		TargetGalaxy: 1, TargetSystem: 2, TargetPosition: 3,
		OriginGalaxy: &og, OriginSystem: &os, OriginPosition: &op, ArrivalTime: &arrival,
	})
	repo.CreateRadarEvent(context.Background(), RadarEvent{
		PlayerID: 1, EventType: "incoming_attack", SourcePlayerID: &src, FleetID: &fid,
		TargetGalaxy: 1, TargetSystem: 2, TargetPosition: 3,
		OriginGalaxy: &og, OriginSystem: &os, OriginPosition: &op, ArrivalTime: &arrival,
	})

	statuses, err := svc.PlanetStatus(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(statuses) != 1 {
		t.Fatalf("expected 1 status, got %d", len(statuses))
	}
	if statuses[0].Status != "attack_incoming" {
		t.Errorf("expected attack_incoming (priority), got %s", statuses[0].Status)
	}
	if statuses[0].FleetCount != 2 {
		t.Errorf("expected fleet_count 2, got %d", statuses[0].FleetCount)
	}
}

func TestDetectFleet_CreatesEvent(t *testing.T) {
	repo := newMockRepo()
	svc := NewRadarService(repo, "http://localhost:8082", "http://localhost:8083")

	err := svc.DetectFleet(context.Background(), DetectFleetRequest{
		TargetPlayerID: 1,
		SourcePlayerID: 2,
		FleetID:        100,
		TargetGalaxy:   1,
		TargetSystem:   2,
		TargetPosition: 3,
		OriginGalaxy:   4,
		OriginSystem:   5,
		OriginPosition: 6,
		ArrivalTime:    "2026-06-01T12:00:00Z",
		Mission:        "attack",
	})
	if err != nil {
		t.Fatal(err)
	}

	events, _ := svc.GetEvents(context.Background(), 1)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].EventType != "incoming_attack" {
		t.Errorf("expected incoming_attack, got %s", events[0].EventType)
	}
	if *events[0].SourcePlayerID != 2 {
		t.Errorf("expected source 2, got %d", *events[0].SourcePlayerID)
	}
}

func TestDetectFleet_EspionageMission(t *testing.T) {
	repo := newMockRepo()
	svc := NewRadarService(repo, "http://localhost:8082", "http://localhost:8083")

	err := svc.DetectFleet(context.Background(), DetectFleetRequest{
		TargetPlayerID: 1,
		SourcePlayerID: 2,
		FleetID:        100,
		TargetGalaxy:   1,
		TargetSystem:   2,
		TargetPosition: 3,
		OriginGalaxy:   4,
		OriginSystem:   5,
		OriginPosition: 6,
		ArrivalTime:    "2026-06-01T12:00:00Z",
		Mission:        "espionage",
	})
	if err != nil {
		t.Fatal(err)
	}

	events, _ := svc.GetEvents(context.Background(), 1)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].EventType != "espionage" {
		t.Errorf("expected espionage, got %s", events[0].EventType)
	}
}

func TestEUXScan_NoRadar(t *testing.T) {
	repo := newMockRepo()
	svc := NewRadarService(repo, "http://localhost:8082", "http://localhost:8083")

	_, err := svc.EUXScan(context.Background(), 1, 3, 15, 5)
	if err == nil || err.Error() != "no eu-x radar installed" {
		t.Fatalf("expected no eu-x radar error, got: %v", err)
	}
}

func TestEUXScan_RangeCheck(t *testing.T) {
	repo := newMockRepo()
	svc := NewRadarService(repo, "http://localhost:8082", "http://localhost:8083")

	repo.CreateOrUpdateEuxRadar(context.Background(), 1, 1, 10, 1, 1)

	_, err := svc.EUXScan(context.Background(), 1, 1, 20, 1)
	if err == nil || err.Error() != "target out of range: 10 systems away, max 5" {
		t.Fatalf("expected out of range error, got: %v", err)
	}
}

func TestEUXScan_WithinRange(t *testing.T) {
	repo := newMockRepo()
	svc := NewRadarService(repo, "http://localhost:8082", "http://localhost:8083")

	repo.CreateOrUpdateEuxRadar(context.Background(), 1, 1, 10, 1, 2)

	ts := fleetServiceMock(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]map[string]any{
			{"id": 55, "mission": "transport", "ships": map[string]int{"cargo": 5}, "arrives_at": nil},
		})
	})
	defer ts.Close()

	svc.fleetBaseURL = ts.URL

	result, err := svc.EUXScan(context.Background(), 1, 1, 12, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Fleets) != 1 {
		t.Fatalf("expected 1 fleet, got %d", len(result.Fleets))
	}
	if result.Fleets[0].Mission != "transport" {
		t.Errorf("expected transport mission, got %s", result.Fleets[0].Mission)
	}
}

func TestEUXScan_RangeAtLimit(t *testing.T) {
	repo := newMockRepo()
	svc := NewRadarService(repo, "http://localhost:8082", "http://localhost:8083")

	repo.CreateOrUpdateEuxRadar(context.Background(), 1, 1, 10, 1, 2)

	ts := fleetServiceMock(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]map[string]any{})
	})
	defer ts.Close()
	svc.fleetBaseURL = ts.URL

	result, err := svc.EUXScan(context.Background(), 1, 1, 15, 1)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestEUXScan_JustOutOfRange(t *testing.T) {
	repo := newMockRepo()
	svc := NewRadarService(repo, "http://localhost:8082", "http://localhost:8083")

	repo.CreateOrUpdateEuxRadar(context.Background(), 1, 1, 10, 1, 2)

	_, err := svc.EUXScan(context.Background(), 1, 1, 16, 1)
	if err == nil {
		t.Fatal("expected error for out of range")
	}
}
