package main

import (
	"context"
	"testing"
)

func TestMockRepo_CreateAlliance(t *testing.T) {
	repo := newMockRepo()
	a, err := repo.CreateAlliance(context.Background(), "Test Alliance", "TA", 1)
	if err != nil {
		t.Fatal(err)
	}
	if a.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if a.Name != "Test Alliance" {
		t.Errorf("expected name Test Alliance, got %s", a.Name)
	}
	if a.Tag != "TA" {
		t.Errorf("expected tag TA, got %s", a.Tag)
	}
	if a.FounderID != 1 {
		t.Errorf("expected founder 1, got %d", a.FounderID)
	}
}

func TestMockRepo_CreateAlliance_DuplicateName(t *testing.T) {
	repo := newMockRepo()
	_, err := repo.CreateAlliance(context.Background(), "Test Alliance", "TA", 1)
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.CreateAlliance(context.Background(), "Test Alliance", "TB", 2)
	if err == nil {
		t.Error("expected error for duplicate name")
	}
}

func TestMockRepo_CreateAlliance_DuplicateTag(t *testing.T) {
	repo := newMockRepo()
	_, err := repo.CreateAlliance(context.Background(), "Test Alliance", "TA", 1)
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.CreateAlliance(context.Background(), "Other Alliance", "TA", 2)
	if err == nil {
		t.Error("expected error for duplicate tag")
	}
}

func TestMockRepo_GetAlliance(t *testing.T) {
	repo := newMockRepo()
	created, _ := repo.CreateAlliance(context.Background(), "Test", "TST", 1)
	got, err := repo.GetAlliance(context.Background(), created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Test" {
		t.Errorf("expected name Test, got %s", got.Name)
	}
}

func TestMockRepo_GetAlliance_NotFound(t *testing.T) {
	repo := newMockRepo()
	_, err := repo.GetAlliance(context.Background(), 999)
	if err == nil {
		t.Error("expected error for non-existent alliance")
	}
}

func TestMockRepo_GetAllianceByName(t *testing.T) {
	repo := newMockRepo()
	repo.CreateAlliance(context.Background(), "Test Alliance", "TA", 1)
	got, err := repo.GetAllianceByName(context.Background(), "Test Alliance")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("expected alliance, got nil")
	}
	if got.Tag != "TA" {
		t.Errorf("expected tag TA, got %s", got.Tag)
	}
}

func TestMockRepo_GetAllianceByTag(t *testing.T) {
	repo := newMockRepo()
	repo.CreateAlliance(context.Background(), "Test Alliance", "TA", 1)
	got, err := repo.GetAllianceByTag(context.Background(), "TA")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("expected alliance, got nil")
	}
}

func TestMockRepo_AddMember(t *testing.T) {
	repo := newMockRepo()
	a, _ := repo.CreateAlliance(context.Background(), "Test", "TST", 1)
	m, err := repo.AddMember(context.Background(), a.ID, 2, "member")
	if err != nil {
		t.Fatal(err)
	}
	if m.PlayerID != 2 {
		t.Errorf("expected player 2, got %d", m.PlayerID)
	}
	if m.Role != "member" {
		t.Errorf("expected role member, got %s", m.Role)
	}
}

func TestMockRepo_AddMember_Duplicate(t *testing.T) {
	repo := newMockRepo()
	a, _ := repo.CreateAlliance(context.Background(), "Test", "TST", 1)
	repo.AddMember(context.Background(), a.ID, 2, "member")
	_, err := repo.AddMember(context.Background(), a.ID, 2, "member")
	if err == nil {
		t.Error("expected error for duplicate member")
	}
}

func TestMockRepo_RemoveMember(t *testing.T) {
	repo := newMockRepo()
	a, _ := repo.CreateAlliance(context.Background(), "Test", "TST", 1)
	repo.AddMember(context.Background(), a.ID, 2, "member")
	err := repo.RemoveMember(context.Background(), 2)
	if err != nil {
		t.Fatal(err)
	}
	m, _ := repo.GetMember(context.Background(), 2)
	if m != nil {
		t.Error("expected member to be removed")
	}
}

func TestMockRepo_GetMembers(t *testing.T) {
	repo := newMockRepo()
	a, _ := repo.CreateAlliance(context.Background(), "Test", "TST", 1)
	repo.AddMember(context.Background(), a.ID, 1, "founder")
	repo.AddMember(context.Background(), a.ID, 2, "member")
	repo.AddMember(context.Background(), a.ID, 3, "member")

	members, err := repo.GetMembers(context.Background(), a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 3 {
		t.Errorf("expected 3 members, got %d", len(members))
	}
}

func TestMockRepo_UpdateMemberRole(t *testing.T) {
	repo := newMockRepo()
	a, _ := repo.CreateAlliance(context.Background(), "Test", "TST", 1)
	repo.AddMember(context.Background(), a.ID, 2, "member")

	err := repo.UpdateMemberRole(context.Background(), a.ID, 2, "officer")
	if err != nil {
		t.Fatal(err)
	}

	m, _ := repo.GetMember(context.Background(), 2)
	if m.Role != "officer" {
		t.Errorf("expected role officer, got %s", m.Role)
	}
}

func TestMockRepo_UpdateBank(t *testing.T) {
	repo := newMockRepo()
	err := repo.UpdateBank(context.Background(), 1, 100, 200, 300)
	if err != nil {
		t.Fatal(err)
	}

	bank, err := repo.GetBank(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if bank.Metal != 100 {
		t.Errorf("expected 100 metal, got %d", bank.Metal)
	}
	if bank.Crystal != 200 {
		t.Errorf("expected 200 crystal, got %d", bank.Crystal)
	}
	if bank.Gas != 300 {
		t.Errorf("expected 300 gas, got %d", bank.Gas)
	}
}

func TestMockRepo_UpdateBank_Overwrite(t *testing.T) {
	repo := newMockRepo()
	repo.UpdateBank(context.Background(), 1, 100, 200, 300)
	repo.UpdateBank(context.Background(), 1, 50, 60, 70)

	bank, _ := repo.GetBank(context.Background(), 1)
	if bank.Metal != 50 {
		t.Errorf("expected 50 metal, got %d", bank.Metal)
	}
}

func TestMockRepo_AddAuditLog(t *testing.T) {
	repo := newMockRepo()
	err := repo.AddAuditLog(context.Background(), 1, 1, "alliance_created", map[string]any{"key": "value"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestMockRepo_GetBank_NotFound(t *testing.T) {
	repo := newMockRepo()
	bank, err := repo.GetBank(context.Background(), 999)
	if err != nil {
		t.Fatal(err)
	}
	if bank != nil {
		t.Error("expected nil for non-existent bank")
	}
}
