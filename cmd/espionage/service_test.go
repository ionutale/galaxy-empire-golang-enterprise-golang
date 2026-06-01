package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendProbe_EmptyTarget(t *testing.T) {
	planetSvc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/internal/ships/deduct":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]bool{"ok": true})
		case "/internal/planet/info":
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer planetSvc.Close()

	svc := NewEspionageService(newMockRepo(), planetSvc.URL)
	report, err := svc.SendProbe(context.Background(), 1, ProbeRequest{
		TargetGalaxy: 1, TargetSystem: 1, TargetPosition: 1, PlanetID: 1,
	})
	if err != nil {
		t.Fatal("expected success for empty target, got:", err)
	}
	if report.TargetPlayerID != 0 {
		t.Errorf("expected target_player_id 0 for empty target, got %d", report.TargetPlayerID)
	}
}

func TestSendProbe_DetailLevel(t *testing.T) {
	planetSvc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/internal/ships/deduct":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]bool{"ok": true})
		case "/internal/planet/info":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(PlanetInfo{
				PlanetID: 10, PlayerID: 2,
				Metal: 50000, Crystal: 30000, Gas: 15000,
				Ships: map[string]int{"cargo": 20, "light_fighter": 10},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer planetSvc.Close()

	svc := NewEspionageService(newMockRepo(), planetSvc.URL)
	report, err := svc.SendProbe(context.Background(), 1, ProbeRequest{
		TargetGalaxy: 1, TargetSystem: 1, TargetPosition: 1, PlanetID: 1,
	})
	if err != nil {
		t.Fatal("expected success, got:", err)
	}

	if report.DetailLevel != 5 {
		t.Errorf("expected detail_level 5, got %d", report.DetailLevel)
	}
	if report.Resources["metal"] != 50000 {
		t.Errorf("expected metal 50000, got %d", report.Resources["metal"])
	}
	if report.Fleet["cargo"] != 20 {
		t.Errorf("expected cargo 20, got %d", report.Fleet["cargo"])
	}
}

func TestSendProbe_DeductsProbe(t *testing.T) {
	deductCalled := false
	planetSvc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/internal/ships/deduct":
			deductCalled = true
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]bool{"ok": true})
		case "/internal/planet/info":
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer planetSvc.Close()

	svc := NewEspionageService(newMockRepo(), planetSvc.URL)
	_, err := svc.SendProbe(context.Background(), 1, ProbeRequest{
		TargetGalaxy: 1, TargetSystem: 1, TargetPosition: 1, PlanetID: 5,
	})
	if err != nil {
		t.Fatal("expected success, got:", err)
	}
	if !deductCalled {
		t.Error("expected deduct probe to be called")
	}
}

func TestGetReport_NotYours(t *testing.T) {
	mock := newMockRepo()
	mock.CreateReport(nil, EspionageReport{
		PlayerID: 1, TargetPlayerID: 2,
		TargetGalaxy: 1, TargetSystem: 1, TargetPosition: 1,
	})
	svc := NewEspionageService(mock, "http://localhost:8082")

	_, err := svc.GetReport(context.Background(), 3, 1)
	if err == nil {
		t.Error("expected error for wrong player")
	}
}

func TestServiceGetReport_Success(t *testing.T) {
	mock := newMockRepo()
	mock.CreateReport(nil, EspionageReport{
		PlayerID: 1, TargetPlayerID: 2,
		TargetGalaxy: 1, TargetSystem: 1, TargetPosition: 1,
	})
	svc := NewEspionageService(mock, "http://localhost:8082")

	report, err := svc.GetReport(context.Background(), 1, 1)
	if err != nil {
		t.Fatal("expected success, got:", err)
	}
	if report.PlayerID != 1 {
		t.Errorf("expected player_id 1, got %d", report.PlayerID)
	}
}

func TestListReports_MultiplePlayers(t *testing.T) {
	mock := newMockRepo()
	mock.CreateReport(nil, EspionageReport{PlayerID: 1, TargetPlayerID: 2, TargetGalaxy: 1, TargetSystem: 1, TargetPosition: 1})
	mock.CreateReport(nil, EspionageReport{PlayerID: 2, TargetPlayerID: 1, TargetGalaxy: 1, TargetSystem: 2, TargetPosition: 3})
	mock.CreateReport(nil, EspionageReport{PlayerID: 3, TargetPlayerID: 4, TargetGalaxy: 1, TargetSystem: 3, TargetPosition: 5})
	svc := NewEspionageService(mock, "http://localhost:8082")

	reports, err := svc.ListReports(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(reports) != 2 {
		t.Errorf("expected 2 reports for player 1, got %d", len(reports))
	}
}

func TestDeleteReport_NotYours(t *testing.T) {
	mock := newMockRepo()
	mock.CreateReport(nil, EspionageReport{
		PlayerID: 1, TargetPlayerID: 2,
		TargetGalaxy: 1, TargetSystem: 1, TargetPosition: 1,
	})
	svc := NewEspionageService(mock, "http://localhost:8082")

	err := svc.DeleteReport(context.Background(), 3, 1)
	if err == nil {
		t.Error("expected error for wrong player")
	}
}

func TestServiceDeleteReport_Success(t *testing.T) {
	mock := newMockRepo()
	mock.CreateReport(nil, EspionageReport{
		PlayerID: 1, TargetPlayerID: 2,
		TargetGalaxy: 1, TargetSystem: 1, TargetPosition: 1,
	})
	svc := NewEspionageService(mock, "http://localhost:8082")

	err := svc.DeleteReport(context.Background(), 1, 1)
	if err != nil {
		t.Fatal("expected success, got:", err)
	}

	_, err = svc.GetReport(context.Background(), 1, 1)
	if err == nil {
		t.Error("expected report to be deleted")
	}
}

func TestBuildReport_WithTarget(t *testing.T) {
	svc := NewEspionageService(newMockRepo(), "http://localhost:8082")
	target := PlanetInfo{
		PlanetID: 10, PlayerID: 2,
		Metal: 10000, Crystal: 5000, Gas: 2000,
		Ships: map[string]int{"cargo": 5},
	}
	req := ProbeRequest{TargetGalaxy: 1, TargetSystem: 2, TargetPosition: 3, PlanetID: 1}

	report := svc.buildReport(1, target, req, 5)

	if report.PlayerID != 1 {
		t.Errorf("expected player 1, got %d", report.PlayerID)
	}
	if report.TargetPlayerID != 2 {
		t.Errorf("expected target player 2, got %d", report.TargetPlayerID)
	}
	if report.TargetGalaxy != 1 || report.TargetSystem != 2 || report.TargetPosition != 3 {
		t.Errorf("wrong target coords: %d/%d/%d", report.TargetGalaxy, report.TargetSystem, report.TargetPosition)
	}
	if report.Resources["metal"] != 10000 {
		t.Errorf("expected metal 10000, got %d", report.Resources["metal"])
	}
	if report.Fleet["cargo"] != 5 {
		t.Errorf("expected cargo 5, got %d", report.Fleet["cargo"])
	}
	if report.DetailLevel != 5 {
		t.Errorf("expected detail 5, got %d", report.DetailLevel)
	}
}
