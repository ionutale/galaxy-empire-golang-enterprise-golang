package main

import (
	"context"
	"testing"
	"time"
)

func TestMockRepo_CreateRadarEvent(t *testing.T) {
	repo := newMockRepo()
	arrival := time.Now().Add(1 * time.Hour)
	srcID := 2
	fleetID := 100
	og := 1
	os := 1
	op := 1

	e, err := repo.CreateRadarEvent(context.Background(), RadarEvent{
		PlayerID:       1,
		EventType:      "incoming_attack",
		SourcePlayerID: &srcID,
		FleetID:        &fleetID,
		TargetGalaxy:   1,
		TargetSystem:   2,
		TargetPosition: 3,
		OriginGalaxy:   &og,
		OriginSystem:   &os,
		OriginPosition: &op,
		ArrivalTime:    &arrival,
	})
	if err != nil {
		t.Fatal(err)
	}
	if e.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if e.EventType != "incoming_attack" {
		t.Errorf("expected incoming_attack, got %s", e.EventType)
	}
	if e.PlayerID != 1 {
		t.Errorf("expected player 1, got %d", e.PlayerID)
	}
	if e.Resolved {
		t.Error("expected resolved to be false")
	}
}

func TestMockRepo_GetPlayerEvents(t *testing.T) {
	repo := newMockRepo()
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
	repo.CreateRadarEvent(context.Background(), RadarEvent{
		PlayerID: 2, EventType: "incoming_attack", SourcePlayerID: &src, FleetID: &fid,
		TargetGalaxy: 7, TargetSystem: 8, TargetPosition: 9,
		OriginGalaxy: &og, OriginSystem: &os, OriginPosition: &op, ArrivalTime: &arrival,
	})

	events, err := repo.GetPlayerEvents(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Errorf("expected 2 events, got %d", len(events))
	}

	events2, err := repo.GetPlayerEvents(context.Background(), 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(events2) != 1 {
		t.Errorf("expected 1 event, got %d", len(events2))
	}
}

func TestMockRepo_GetUnresolvedEvents(t *testing.T) {
	repo := newMockRepo()
	src := 2
	fid := 100
	og, os, op := 1, 1, 1
	arrival := time.Now().Add(1 * time.Hour)

	e1, _ := repo.CreateRadarEvent(context.Background(), RadarEvent{
		PlayerID: 1, EventType: "incoming_attack", SourcePlayerID: &src, FleetID: &fid,
		TargetGalaxy: 1, TargetSystem: 2, TargetPosition: 3,
		OriginGalaxy: &og, OriginSystem: &os, OriginPosition: &op, ArrivalTime: &arrival,
	})
	repo.CreateRadarEvent(context.Background(), RadarEvent{
		PlayerID: 1, EventType: "espionage", SourcePlayerID: &src, FleetID: &fid,
		TargetGalaxy: 4, TargetSystem: 5, TargetPosition: 6,
		OriginGalaxy: &og, OriginSystem: &os, OriginPosition: &op, ArrivalTime: &arrival,
	})

	repo.ResolveEvent(context.Background(), e1.ID)

	events, err := repo.GetUnresolvedEvents(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Errorf("expected 1 unresolved event, got %d", len(events))
	}
	if events[0].EventType != "espionage" {
		t.Errorf("expected espionage event, got %s", events[0].EventType)
	}
}

func TestMockRepo_ResolveEvent(t *testing.T) {
	repo := newMockRepo()
	src := 2
	fid := 100
	og, os, op := 1, 1, 1
	arrival := time.Now().Add(1 * time.Hour)

	e, _ := repo.CreateRadarEvent(context.Background(), RadarEvent{
		PlayerID: 1, EventType: "incoming_attack", SourcePlayerID: &src, FleetID: &fid,
		TargetGalaxy: 1, TargetSystem: 2, TargetPosition: 3,
		OriginGalaxy: &og, OriginSystem: &os, OriginPosition: &op, ArrivalTime: &arrival,
	})

	if err := repo.ResolveEvent(context.Background(), e.ID); err != nil {
		t.Fatal(err)
	}

	events, _ := repo.GetUnresolvedEvents(context.Background(), 1)
	if len(events) != 0 {
		t.Error("expected no unresolved events")
	}

	if err := repo.ResolveEvent(context.Background(), 999); err == nil {
		t.Error("expected error for non-existent event")
	}
}

func TestMockRepo_EuxRadar(t *testing.T) {
	repo := newMockRepo()

	got, err := repo.GetEuxRadar(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Error("expected nil for non-existent eu-x radar")
	}

	if err := repo.CreateOrUpdateEuxRadar(context.Background(), 1, 3, 10, 5, 2); err != nil {
		t.Fatal(err)
	}

	eux, err := repo.GetEuxRadar(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if eux == nil {
		t.Fatal("expected eu-x radar")
	}
	if eux.Level != 2 {
		t.Errorf("expected level 2, got %d", eux.Level)
	}
	if eux.Galaxy != 3 {
		t.Errorf("expected galaxy 3, got %d", eux.Galaxy)
	}

	if err := repo.CreateOrUpdateEuxRadar(context.Background(), 1, 4, 11, 6, 5); err != nil {
		t.Fatal(err)
	}
	eux, _ = repo.GetEuxRadar(context.Background(), 1)
	if eux.Level != 5 {
		t.Errorf("expected level 5 after update, got %d", eux.Level)
	}
}
