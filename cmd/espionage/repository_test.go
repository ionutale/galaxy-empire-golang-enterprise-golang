package main

import (
	"context"
	"testing"
)

func TestCreateReport(t *testing.T) {
	mock := newMockRepo()
	report, err := mock.CreateReport(context.Background(), EspionageReport{
		PlayerID: 1, TargetPlayerID: 2,
		TargetGalaxy: 1, TargetSystem: 1, TargetPosition: 1,
		DetailLevel: 5,
		Resources:   map[string]int{"metal": 10000},
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.ID != 1 {
		t.Errorf("expected id 1, got %d", report.ID)
	}
	if report.PlayerID != 1 {
		t.Errorf("expected player 1, got %d", report.PlayerID)
	}
}

func TestGetReportByID(t *testing.T) {
	mock := newMockRepo()
	mock.CreateReport(context.Background(), EspionageReport{
		PlayerID: 1, TargetPlayerID: 2,
		TargetGalaxy: 1, TargetSystem: 1, TargetPosition: 1,
	})
	mock.CreateReport(context.Background(), EspionageReport{
		PlayerID: 3, TargetPlayerID: 4,
		TargetGalaxy: 1, TargetSystem: 2, TargetPosition: 3,
	})

	report, err := mock.GetReportByID(context.Background(), 2)
	if err != nil {
		t.Fatal(err)
	}
	if report.PlayerID != 3 {
		t.Errorf("expected player 3, got %d", report.PlayerID)
	}
}

func TestGetReportByID_NotFound(t *testing.T) {
	mock := newMockRepo()
	_, err := mock.GetReportByID(context.Background(), 999)
	if err == nil {
		t.Error("expected error for not found")
	}
}

func TestListReportsForPlayer_AsSpy(t *testing.T) {
	mock := newMockRepo()
	mock.CreateReport(context.Background(), EspionageReport{
		PlayerID: 1, TargetPlayerID: 2,
		TargetGalaxy: 1, TargetSystem: 1, TargetPosition: 1,
	})
	mock.CreateReport(context.Background(), EspionageReport{
		PlayerID: 2, TargetPlayerID: 1,
		TargetGalaxy: 1, TargetSystem: 2, TargetPosition: 3,
	})
	mock.CreateReport(context.Background(), EspionageReport{
		PlayerID: 3, TargetPlayerID: 4,
		TargetGalaxy: 1, TargetSystem: 3, TargetPosition: 5,
	})

	reports, err := mock.ListReportsForPlayer(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(reports) != 2 {
		t.Errorf("expected 2 reports for player 1, got %d", len(reports))
	}
}

func TestDeleteReport_RemovesReport(t *testing.T) {
	mock := newMockRepo()
	mock.CreateReport(context.Background(), EspionageReport{
		PlayerID: 1, TargetPlayerID: 2,
		TargetGalaxy: 1, TargetSystem: 1, TargetPosition: 1,
	})

	err := mock.DeleteReport(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}

	reports, _ := mock.ListReportsForPlayer(context.Background(), 1)
	if len(reports) != 0 {
		t.Errorf("expected 0 reports after delete, got %d", len(reports))
	}
}

func TestReportDataStored(t *testing.T) {
	mock := newMockRepo()
	report, err := mock.CreateReport(context.Background(), EspionageReport{
		PlayerID: 1, TargetPlayerID: 2,
		TargetGalaxy: 1, TargetSystem: 1, TargetPosition: 1,
		DetailLevel: 5,
		Resources:   map[string]int{"metal": 10000, "crystal": 5000, "gas": 2000},
		Fleet:       map[string]int{"cargo": 10},
	})
	if err != nil {
		t.Fatal(err)
	}

	loaded, err := mock.GetReportByID(context.Background(), report.ID)
	if err != nil {
		t.Fatal(err)
	}

	if loaded.Resources["metal"] != 10000 {
		t.Errorf("expected metal 10000, got %d", loaded.Resources["metal"])
	}
	if loaded.Fleet["cargo"] != 10 {
		t.Errorf("expected cargo 10, got %d", loaded.Fleet["cargo"])
	}
}
