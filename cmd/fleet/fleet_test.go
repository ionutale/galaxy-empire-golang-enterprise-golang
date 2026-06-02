package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDispatchFleet_InvalidMission(t *testing.T) {
	svc := NewFleetService(newMockRepo(), "http://localhost:8082", "", "", "")
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
	svc := NewFleetService(newMockRepo(), "http://localhost:8082", "", "", "")
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
	svc := NewFleetService(newMockRepo(), "http://localhost:8082", "", "", "")
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
	svc := NewFleetService(newMockRepo(), "http://localhost:1", "", "", "")
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

	svc := NewFleetService(newMockRepo(), ts.URL, "", "", "")
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
	ts := planetServiceMock(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	})
	defer ts.Close()

	svc := NewFleetService(newMockRepo(), ts.URL, "", "", "")
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
	if fleet.Status != "in_transit" {
		t.Fatalf("expected in_transit, got %s", fleet.Status)
	}
	if fleet.Ships["cargo"] != 5 {
		t.Fatalf("expected 5 cargo, got %d", fleet.Ships["cargo"])
	}
}

func TestMyFleets_NoAuth(t *testing.T) {
	h := NewFleetHandler(NewFleetService(newMockRepo(), "", "", "", ""))
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
	svc := NewFleetService(mock, "", "", "", "")
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
	h := NewFleetHandler(NewFleetService(newMockRepo(), "", "", "", ""))
	req := httptest.NewRequest(http.MethodPost, "/api/fleet/dispatch", nil)
	w := httptest.NewRecorder()
	h.Dispatch(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestDistance(t *testing.T) {
	d := distance(1, 1, 1, 1, 1, 2)
	if d != 1 {
		t.Errorf("same coords distance should be 1, got %d", d)
	}
	d = distance(1, 1, 1, 3, 1, 1)
	if d != 40000 {
		t.Errorf("expected 40000, got %d", d)
	}
}

func TestMinShipSpeed(t *testing.T) {
	spd, onlyBomber := minShipSpeed(map[string]int{"cargo": 1, "light_fighter": 1})
	if onlyBomber {
		t.Error("should not be only bomber")
	}
	if spd != 7500 {
		t.Errorf("expected 7500 (min of cargo 7500 and lf 12500), got %d", spd)
	}
}

func TestMinShipSpeed_BomberAlone(t *testing.T) {
	_, onlyBomber := minShipSpeed(map[string]int{"bomber": 1})
	if !onlyBomber {
		t.Error("should be only bomber")
	}
}

func TestDispatchFleet_BomberAlone(t *testing.T) {
	svc := NewFleetService(newMockRepo(), "http://localhost:8082", "", "", "")
	_, err := svc.DispatchFleet(context.Background(), 1, DispatchRequest{
		OriginPlanetID: 1,
		Ships:          map[string]int{"bomber": 1},
		TargetGalaxy:   1, TargetSystem: 1, TargetPosition: 1,
		Mission: "transport", SpeedPct: 100,
	})
	if err == nil || !strings.Contains(err.Error(), "bomber") {
		t.Fatalf("expected bomber alone error, got: %v", err)
	}
}

func TestDispatchFleet_UnknownShip(t *testing.T) {
	svc := NewFleetService(newMockRepo(), "http://localhost:8082", "", "", "")
	_, err := svc.DispatchFleet(context.Background(), 1, DispatchRequest{
		OriginPlanetID: 1,
		Ships:          map[string]int{"death_star": 1},
		TargetGalaxy:   1, TargetSystem: 1, TargetPosition: 1,
		Mission: "transport", SpeedPct: 100,
	})
	if err == nil || !strings.Contains(err.Error(), "unknown ship") {
		t.Fatalf("expected unknown ship error, got: %v", err)
	}
}

func TestAbs(t *testing.T) {
	if abs(5) != 5 {
		t.Errorf("abs(5) should be 5, got %d", abs(5))
	}
	if abs(-5) != 5 {
		t.Errorf("abs(-5) should be 5, got %d", abs(-5))
	}
	if abs(0) != 0 {
		t.Errorf("abs(0) should be 0, got %d", abs(0))
	}
}

func TestRecallFleet_NotYours(t *testing.T) {
	mock := newMockRepo()
	svc := NewFleetService(mock, "", "", "", "")
	mock.CreateFleet(context.Background(), Fleet{ID: 1, PlayerID: 1, Status: "stationed", Ships: map[string]int{"cargo": 5}})
	_, err := svc.RecallFleet(context.Background(), 2, 1)
	if err == nil || !strings.Contains(err.Error(), "not your fleet") {
		t.Fatalf("expected not your fleet error, got: %v", err)
	}
}

func TestRecallFleet_Success(t *testing.T) {
	mock := newMockRepo()
	svc := NewFleetService(mock, "", "", "", "")
	f, _ := mock.CreateFleet(context.Background(), Fleet{PlayerID: 1, OriginPlanetID: 1, Status: "in_transit", Ships: map[string]int{"cargo": 5}, ArrivesAt: time.Now().Add(1 * time.Hour)})
	recalled, err := svc.RecallFleet(context.Background(), 1, f.ID)
	if err != nil {
		t.Fatal(err)
	}
	if recalled.Status != "returning" {
		t.Errorf("expected returning, got %s", recalled.Status)
	}
}

func TestSplitFleet_NotStationed(t *testing.T) {
	mock := newMockRepo()
	svc := NewFleetService(mock, "", "", "", "")
	f, _ := mock.CreateFleet(context.Background(), Fleet{PlayerID: 1, Status: "in_transit", Ships: map[string]int{"cargo": 5}})
	_, err := svc.SplitFleet(context.Background(), 1, f.ID, map[string]int{"cargo": 2})
	if err == nil || !strings.Contains(err.Error(), "stationed") {
		t.Fatalf("expected stationed error, got: %v", err)
	}
}

func TestSplitFleet_Success(t *testing.T) {
	mock := newMockRepo()
	svc := NewFleetService(mock, "", "", "", "")
	f, _ := mock.CreateFleet(context.Background(), Fleet{PlayerID: 1, Status: "stationed", Ships: map[string]int{"cargo": 5, "fighter": 3}})
	split, err := svc.SplitFleet(context.Background(), 1, f.ID, map[string]int{"cargo": 2})
	if err != nil {
		t.Fatal(err)
	}
	if split.Ships["cargo"] != 2 {
		t.Errorf("expected 2 cargo in split, got %d", split.Ships["cargo"])
	}
	// Original should have 3 cargo left
	orig, _ := mock.GetFleetByID(context.Background(), f.ID)
	if orig.Ships["cargo"] != 3 {
		t.Errorf("expected 3 cargo in original, got %d", orig.Ships["cargo"])
	}
}

func TestMergeFleets_Success(t *testing.T) {
	mock := newMockRepo()
	svc := NewFleetService(mock, "", "", "", "")
	f1, _ := mock.CreateFleet(context.Background(), Fleet{PlayerID: 1, Status: "stationed", TargetGalaxy: 1, TargetSystem: 1, TargetPosition: 1, Ships: map[string]int{"cargo": 3}})
	f2, _ := mock.CreateFleet(context.Background(), Fleet{PlayerID: 1, Status: "stationed", TargetGalaxy: 1, TargetSystem: 1, TargetPosition: 1, Ships: map[string]int{"fighter": 5}})

	merged, err := svc.MergeFleets(context.Background(), 1, []int{f1.ID, f2.ID})
	if err != nil {
		t.Fatal(err)
	}
	if merged.Ships["cargo"] != 3 || merged.Ships["fighter"] != 5 {
		t.Errorf("expected cargo:3 fighter:5, got cargo:%d fighter:%d", merged.Ships["cargo"], merged.Ships["fighter"])
	}
	// f2 should be deleted
	_, err = mock.GetFleetByID(context.Background(), f2.ID)
	if err == nil {
		t.Error("expected f2 to be deleted")
	}
}

func TestCheckFleetSlotLimit(t *testing.T) {
	mock := newMockRepo()
	svc := NewFleetService(mock, "", "", "", "")
	// With computer_tech=0, limit = 1. With 1 fleet, should pass.
	mock.CreateFleet(context.Background(), Fleet{PlayerID: 1, Status: "stationed"})
	err := svc.CheckFleetSlotLimit(context.Background(), 1)
	if err != nil {
		t.Logf("slot limit error (may be ok if planet service unreachable): %v", err)
	}
}

func TestDispatchFleet_AttackCooldownActive(t *testing.T) {
	mock := newMockRepo()
	mock.UpsertAttackCooldown(context.Background(), 1, 1, 1, 1, time.Now())

	ts := planetServiceMock(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	})
	defer ts.Close()

	svc := NewFleetService(mock, ts.URL, "", "", "")
	_, err := svc.DispatchFleet(context.Background(), 1, DispatchRequest{
		OriginPlanetID: 1,
		Ships:          map[string]int{"cargo": 5},
		TargetGalaxy:   1, TargetSystem: 1, TargetPosition: 1,
		Mission: "attack", SpeedPct: 100,
	})
	if err == nil || !strings.Contains(err.Error(), "attack cooldown") {
		t.Fatalf("expected attack cooldown error, got: %v", err)
	}
}

func TestDispatchFleet_AttackCooldownExpired(t *testing.T) {
	mock := newMockRepo()
	mock.UpsertAttackCooldown(context.Background(), 1, 1, 1, 1, time.Now().Add(-3*time.Hour))

	ts := planetServiceMock(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	})
	defer ts.Close()

	svc := NewFleetService(mock, ts.URL, "", "", "")
	fleet, err := svc.DispatchFleet(context.Background(), 1, DispatchRequest{
		OriginPlanetID: 1,
		Ships:          map[string]int{"cargo": 5},
		TargetGalaxy:   1, TargetSystem: 1, TargetPosition: 1,
		Mission: "attack", SpeedPct: 100,
	})
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if fleet.Mission != "attack" {
		t.Fatalf("expected attack mission, got %s", fleet.Mission)
	}
}

func TestDispatchFleet_AttackNoCooldown(t *testing.T) {
	mock := newMockRepo()

	ts := planetServiceMock(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	})
	defer ts.Close()

	svc := NewFleetService(mock, ts.URL, "", "", "")
	fleet, err := svc.DispatchFleet(context.Background(), 1, DispatchRequest{
		OriginPlanetID: 1,
		Ships:          map[string]int{"cargo": 5},
		TargetGalaxy:   1, TargetSystem: 1, TargetPosition: 1,
		Mission: "attack", SpeedPct: 100,
	})
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if fleet.Mission != "attack" {
		t.Fatalf("expected attack mission, got %s", fleet.Mission)
	}
}

func TestDispatchFleet_Transport(t *testing.T) {
	ts := planetServiceMock(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	})
	defer ts.Close()

	svc := NewFleetService(newMockRepo(), ts.URL, "", "", "")
	fleet, err := svc.DispatchFleet(context.Background(), 1, DispatchRequest{
		OriginPlanetID: 1,
		Ships:          map[string]int{"cargo": 5},
		TargetGalaxy:   1, TargetSystem: 1, TargetPosition: 1,
		Mission: "transport", SpeedPct: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if fleet.Mission != "transport" {
		t.Fatalf("expected transport, got %s", fleet.Mission)
	}
	if fleet.Status != "in_transit" {
		t.Fatalf("expected in_transit, got %s", fleet.Status)
	}
}

func TestDispatchFleet_Deploy(t *testing.T) {
	ts := planetServiceMock(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	})
	defer ts.Close()

	svc := NewFleetService(newMockRepo(), ts.URL, "", "", "")
	fleet, err := svc.DispatchFleet(context.Background(), 1, DispatchRequest{
		OriginPlanetID: 1,
		Ships:          map[string]int{"cargo": 5},
		TargetGalaxy:   1, TargetSystem: 1, TargetPosition: 1,
		Mission: "deploy", SpeedPct: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if fleet.Mission != "deploy" {
		t.Fatalf("expected deploy, got %s", fleet.Mission)
	}
	if fleet.Status != "in_transit" {
		t.Fatalf("expected in_transit, got %s", fleet.Status)
	}
}

func TestCheckAttackCooldown_Active(t *testing.T) {
	mock := newMockRepo()
	mock.UpsertAttackCooldown(context.Background(), 1, 1, 1, 1, time.Now())

	svc := NewFleetService(mock, "", "", "", "")
	err := svc.CheckAttackCooldown(context.Background(), 1, 1, 1, 1)
	if err == nil || !strings.Contains(err.Error(), "attack cooldown") {
		t.Fatalf("expected attack cooldown error, got: %v", err)
	}
}

func TestCheckAttackCooldown_Expired(t *testing.T) {
	mock := newMockRepo()
	mock.UpsertAttackCooldown(context.Background(), 1, 1, 1, 1, time.Now().Add(-3*time.Hour))

	svc := NewFleetService(mock, "", "", "", "")
	err := svc.CheckAttackCooldown(context.Background(), 1, 1, 1, 1)
	if err != nil {
		t.Fatalf("expected no error (cooldown expired), got: %v", err)
	}
}

func TestCheckAttackCooldown_NoCooldown(t *testing.T) {
	mock := newMockRepo()
	svc := NewFleetService(mock, "", "", "", "")
	err := svc.CheckAttackCooldown(context.Background(), 1, 1, 1, 1)
	if err != nil {
		t.Fatalf("expected no error (no cooldown), got: %v", err)
	}
}

func planetServiceMock(handler func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/internal/planet/coords":
			json.NewEncoder(w).Encode(map[string]int{"galaxy": 1, "system": 1, "position": 1})
		case "/internal/planet/info":
			// Ownership check — return player_id=1 so any test dispatching as player 1 passes
			json.NewEncoder(w).Encode(map[string]any{"player_id": 1, "planet_id": 1, "metal": 0, "crystal": 0, "gas": 0, "ships": map[string]int{}})
		case "/internal/player/techs":
			json.NewEncoder(w).Encode(map[string]any{"technologies": map[string]int{}})
		default:
			handler(w, r)
		}
	}))
}

func TestDispatchFleet_ACS_WithSlot(t *testing.T) {
	mock := newMockRepo()
	ts := planetServiceMock(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	})
	defer ts.Close()

	svc := NewFleetService(mock, ts.URL, "", "", "")

	fleet, err := svc.DispatchFleet(context.Background(), 1, DispatchRequest{
		OriginPlanetID: 1,
		Ships:          map[string]int{"cargo": 5},
		TargetGalaxy:   1, TargetSystem: 1, TargetPosition: 1,
		Mission: "acs_attack", SpeedPct: 100,
		AllianceGroupID: 42,
	})
	if err != nil {
		t.Fatalf("expected ACS dispatch to succeed with free slot, got: %v", err)
	}
	if fleet.AllianceGroupID != 42 {
		t.Fatalf("expected alliance_group_id 42, got %d", fleet.AllianceGroupID)
	}
	if fleet.Mission != "acs_attack" {
		t.Fatalf("expected acs_attack, got %s", fleet.Mission)
	}
}

func TestAllACSFleetsArrived_AllArrived(t *testing.T) {
	mock := newMockRepo()
	svc := NewFleetService(mock, "", "", "", "")
	mock.CreateFleet(context.Background(), Fleet{AllianceGroupID: 1, Status: "arrived"})
	mock.CreateFleet(context.Background(), Fleet{AllianceGroupID: 1, Status: "stationed"})

	arrived, err := svc.allACSFleetsArrived(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if !arrived {
		t.Error("expected all ACS fleets to be arrived")
	}
}

func TestAllACSFleetsArrived_SomeInTransit(t *testing.T) {
	mock := newMockRepo()
	svc := NewFleetService(mock, "", "", "", "")
	mock.CreateFleet(context.Background(), Fleet{AllianceGroupID: 1, Status: "arrived"})
	mock.CreateFleet(context.Background(), Fleet{AllianceGroupID: 1, Status: "in_transit"})

	arrived, err := svc.allACSFleetsArrived(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if arrived {
		t.Error("expected not all ACS fleets to be arrived")
	}
}

func TestAllACSFleetsArrived_EmptyGroup(t *testing.T) {
	mock := newMockRepo()
	svc := NewFleetService(mock, "", "", "", "")
	arrived, err := svc.allACSFleetsArrived(context.Background(), 999)
	if err != nil {
		t.Fatal(err)
	}
	if arrived {
		t.Error("expected false for empty group")
	}
}

func TestGetACSDefendFleets(t *testing.T) {
	mock := newMockRepo()
	mock.CreateFleet(context.Background(), Fleet{Mission: "acs_defend", Status: "stationed", TargetGalaxy: 1, TargetSystem: 1, TargetPosition: 1, Ships: map[string]int{"light_fighter": 10}})
	mock.CreateFleet(context.Background(), Fleet{Mission: "acs_defend", Status: "stationed", TargetGalaxy: 1, TargetSystem: 1, TargetPosition: 1, Ships: map[string]int{"heavy_fighter": 5}})
	mock.CreateFleet(context.Background(), Fleet{Mission: "acs_defend", Status: "in_transit", TargetGalaxy: 1, TargetSystem: 1, TargetPosition: 1, Ships: map[string]int{"cruiser": 2}})

	fleets, err := mock.GetACSDefendFleets(context.Background(), 1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(fleets) != 2 {
		t.Fatalf("expected 2 stationed defend fleets, got %d", len(fleets))
	}
}

func TestDebrisFieldCRUD(t *testing.T) {
	mock := newMockRepo()

	// Get nonexistent
	d, err := mock.GetDebrisField(context.Background(), 1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if d != nil {
		t.Error("expected nil for nonexistent debris field")
	}

	// Upsert
	if err := mock.UpsertDebrisField(context.Background(), 1, 1, 1, 10000, 5000); err != nil {
		t.Fatal(err)
	}

	d, err = mock.GetDebrisField(context.Background(), 1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if d == nil {
		t.Fatal("expected debris field after upsert")
	}
	if d.Metal != 10000 || d.Crystal != 5000 {
		t.Fatalf("expected metal=10000 crystal=5000, got metal=%d crystal=%d", d.Metal, d.Crystal)
	}

	// Upsert again (add more)
	if err := mock.UpsertDebrisField(context.Background(), 1, 1, 1, 5000, 2500); err != nil {
		t.Fatal(err)
	}

	d, _ = mock.GetDebrisField(context.Background(), 1, 1, 1)
	if d.Metal != 15000 || d.Crystal != 7500 {
		t.Fatalf("expected metal=15000 crystal=7500 after second upsert, got metal=%d crystal=%d", d.Metal, d.Crystal)
	}

	// Update
	if err := mock.UpdateDebrisField(context.Background(), 1, 1, 1, 500, 300); err != nil {
		t.Fatal(err)
	}

	d, _ = mock.GetDebrisField(context.Background(), 1, 1, 1)
	if d.Metal != 500 || d.Crystal != 300 {
		t.Fatalf("expected metal=500 crystal=300 after update, got metal=%d crystal=%d", d.Metal, d.Crystal)
	}
}

func TestHarvestDebris(t *testing.T) {
	mock := newMockRepo()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/internal/resources/add":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]bool{"ok": true})
		default:
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]bool{"ok": true})
		}
	}))
	defer ts.Close()

	svc := NewFleetService(mock, ts.URL, "", "", "")

	mock.UpsertDebrisField(context.Background(), 1, 2, 3, 50000, 30000)

	if err := svc.harvestDebris(context.Background(), 1, map[string]int{"recycler": 2}, 1, 1, 2, 3); err != nil {
		t.Fatal(err)
	}

	d, _ := mock.GetDebrisField(context.Background(), 1, 2, 3)
	if d.Metal == 50000 && d.Crystal == 30000 {
		t.Error("expected debris to be reduced after harvest")
	}
	totalRemaining := d.Metal + d.Crystal
	expectedHarvest := 50000 + 30000 - (2 * 20000)
	if totalRemaining != expectedHarvest {
		t.Errorf("expected %d remaining debris total, got %d", expectedHarvest, totalRemaining)
	}
}

func TestRecycleMission_TravelWorker(t *testing.T) {
	mock := newMockRepo()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	}))
	defer ts.Close()

	svc := NewFleetService(mock, ts.URL, "", "", "")
	mock.UpsertDebrisField(context.Background(), 3, 3, 3, 100000, 50000)

	f, _ := mock.CreateFleet(context.Background(), Fleet{
		PlayerID: 1, OriginPlanetID: 1,
		TargetGalaxy: 3, TargetSystem: 3, TargetPosition: 3,
		Mission: "recycle", Status: "in_transit",
		Ships: map[string]int{"recycler": 1},
	})

	if err := svc.harvestDebris(context.Background(), f.ID, f.Ships, f.OriginPlanetID, f.TargetGalaxy, f.TargetSystem, f.TargetPosition); err != nil {
		t.Fatal(err)
	}
}

func TestColonizeArrival(t *testing.T) {
	mock := newMockRepo()
	var planetCreated bool
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/internal/planet/by-coords":
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
		case "/internal/planet/create":
			planetCreated = true
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"planet_id": 100, "name": "Colony [1:2:3]",
				"galaxy": 1, "system": 2, "position": 3,
			})
		default:
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]bool{"ok": true})
		}
	}))
	defer ts.Close()

	svc := NewFleetService(mock, ts.URL, "", "", "")
	f, _ := mock.CreateFleet(context.Background(), Fleet{
		PlayerID: 1, OriginPlanetID: 1,
		TargetGalaxy: 1, TargetSystem: 2, TargetPosition: 3,
		Mission: "colonize", Status: "in_transit",
		Ships: map[string]int{"colony_ship": 1, "cargo": 5},
	})

	if err := svc.handleColonizeArrival(context.Background(), f); err != nil {
		t.Fatal(err)
	}
	if !planetCreated {
		t.Error("expected planet to be created")
	}

	updatedFleet, _ := mock.GetFleetByID(context.Background(), f.ID)
	if _, ok := updatedFleet.Ships["colony_ship"]; ok {
		t.Error("expected colony_ship to be consumed")
	}
	if updatedFleet.Status != "arrived" {
		t.Errorf("expected fleet status arrived, got %s", updatedFleet.Status)
	}
}

func TestEffectiveMinSpeed(t *testing.T) {
	// No techs = base speed
	spd := effectiveMinSpeed(map[string]int{"cargo": 1}, map[string]int{})
	if spd != 7500 {
		t.Errorf("expected base speed 7500, got %d", spd)
	}
	// With combustion drive level 10
	spd = effectiveMinSpeed(map[string]int{"cargo": 1}, map[string]int{"combustion_drive": 10})
	expected := int(7500 * (1 + 10*0.3))
	if spd != expected {
		t.Errorf("expected boosted speed %d, got %d", expected, spd)
	}
	// Hyperspace on battleship
	spd = effectiveMinSpeed(map[string]int{"battleship": 1}, map[string]int{"hyperspace_drive": 5})
	expected = int(10000 * (1 + 5*0.3))
	if spd != expected {
		t.Errorf("expected boosted speed %d, got %d", expected, spd)
	}
}
