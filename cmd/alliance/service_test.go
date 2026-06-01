package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func planetServiceMock(handler func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/internal/resources/deduct", "/internal/resources/add":
			handler(w, r)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestCreateAlliance_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewAllianceService(repo, "http://localhost:8082")

	alliance, err := svc.CreateAlliance(context.Background(), 1, "Galaxy Empire", "GE")
	if err != nil {
		t.Fatal(err)
	}
	if alliance.Name != "Galaxy Empire" {
		t.Errorf("expected name Galaxy Empire, got %s", alliance.Name)
	}
	if alliance.Tag != "GE" {
		t.Errorf("expected tag GE, got %s", alliance.Tag)
	}

	member, err := repo.GetMember(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if member == nil {
		t.Fatal("expected member to exist")
	}
	if member.Role != "founder" {
		t.Errorf("expected role founder, got %s", member.Role)
	}

	bank, err := repo.GetBank(context.Background(), alliance.ID)
	if err != nil {
		t.Fatal(err)
	}
	if bank == nil {
		t.Fatal("expected bank to exist")
	}
}

func TestCreateAlliance_InvalidName(t *testing.T) {
	svc := NewAllianceService(newMockRepo(), "http://localhost:8082")
	_, err := svc.CreateAlliance(context.Background(), 1, "AB", "TAG")
	if err == nil || !strings.Contains(err.Error(), "3-50") {
		t.Fatalf("expected name length error, got: %v", err)
	}
}

func TestCreateAlliance_InvalidNameChars(t *testing.T) {
	svc := NewAllianceService(newMockRepo(), "http://localhost:8082")
	_, err := svc.CreateAlliance(context.Background(), 1, "Hello!!!", "TAG")
	if err == nil || !strings.Contains(err.Error(), "letters, numbers, and spaces") {
		t.Fatalf("expected name chars error, got: %v", err)
	}
}

func TestCreateAlliance_InvalidTag(t *testing.T) {
	svc := NewAllianceService(newMockRepo(), "http://localhost:8082")
	_, err := svc.CreateAlliance(context.Background(), 1, "Test Alliance", "X")
	if err == nil || !strings.Contains(err.Error(), "2-10") {
		t.Fatalf("expected tag length error, got: %v", err)
	}
}

func TestCreateAlliance_InvalidTagChars(t *testing.T) {
	svc := NewAllianceService(newMockRepo(), "http://localhost:8082")
	_, err := svc.CreateAlliance(context.Background(), 1, "Test Alliance", "t@G")
	if err == nil || !strings.Contains(err.Error(), "uppercase") {
		t.Fatalf("expected tag chars error, got: %v", err)
	}
}

func TestCreateAlliance_AlreadyInAlliance(t *testing.T) {
	repo := newMockRepo()
	svc := NewAllianceService(repo, "http://localhost:8082")

	svc.CreateAlliance(context.Background(), 1, "First", "FRST")
	_, err := svc.CreateAlliance(context.Background(), 1, "Second", "SCND")
	if err == nil || !strings.Contains(err.Error(), "already in an alliance") {
		t.Fatalf("expected already in alliance error, got: %v", err)
	}
}

func TestCreateAlliance_DuplicateName(t *testing.T) {
	repo := newMockRepo()
	svc := NewAllianceService(repo, "http://localhost:8082")

	svc.CreateAlliance(context.Background(), 1, "Same Name", "SN1")
	_, err := svc.CreateAlliance(context.Background(), 2, "Same Name", "SN2")
	if err == nil || !strings.Contains(err.Error(), "name already taken") {
		t.Fatalf("expected name taken error, got: %v", err)
	}
}

func TestCreateAlliance_DuplicateTag(t *testing.T) {
	repo := newMockRepo()
	svc := NewAllianceService(repo, "http://localhost:8082")

	svc.CreateAlliance(context.Background(), 1, "First", "TAG")
	_, err := svc.CreateAlliance(context.Background(), 2, "Second", "TAG")
	if err == nil || !strings.Contains(err.Error(), "tag already taken") {
		t.Fatalf("expected tag taken error, got: %v", err)
	}
}

func TestApplyToAlliance_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewAllianceService(repo, "http://localhost:8082")

	alliance, _ := svc.CreateAlliance(context.Background(), 1, "Test", "TST")
	member, err := svc.ApplyToAlliance(context.Background(), 2, alliance.ID)
	if err != nil {
		t.Fatal(err)
	}
	if member.Role != "member" {
		t.Errorf("expected role member, got %s", member.Role)
	}
	if member.AllianceID != alliance.ID {
		t.Errorf("expected alliance %d, got %d", alliance.ID, member.AllianceID)
	}
}

func TestApplyToAlliance_AllianceNotFound(t *testing.T) {
	svc := NewAllianceService(newMockRepo(), "http://localhost:8082")
	_, err := svc.ApplyToAlliance(context.Background(), 1, 999)
	if err == nil || !strings.Contains(err.Error(), "alliance not found") {
		t.Fatalf("expected alliance not found error, got: %v", err)
	}
}

func TestApplyToAlliance_AlreadyInAlliance(t *testing.T) {
	repo := newMockRepo()
	svc := NewAllianceService(repo, "http://localhost:8082")

	a1, _ := svc.CreateAlliance(context.Background(), 1, "First", "FRST")
	svc.ApplyToAlliance(context.Background(), 2, a1.ID)
	_, err := svc.ApplyToAlliance(context.Background(), 2, a1.ID)
	if err == nil || !strings.Contains(err.Error(), "already in an alliance") {
		t.Fatalf("expected already in alliance error, got: %v", err)
	}
}

func TestLeaveAlliance_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewAllianceService(repo, "http://localhost:8082")

	alliance, _ := svc.CreateAlliance(context.Background(), 1, "Test", "TST")
	svc.ApplyToAlliance(context.Background(), 2, alliance.ID)

	err := svc.LeaveAlliance(context.Background(), 2)
	if err != nil {
		t.Fatal(err)
	}

	member, _ := repo.GetMember(context.Background(), 2)
	if member != nil {
		t.Error("expected member to be removed")
	}
}

func TestLeaveAlliance_FounderCannotLeave(t *testing.T) {
	repo := newMockRepo()
	svc := NewAllianceService(repo, "http://localhost:8082")

	svc.CreateAlliance(context.Background(), 1, "Test", "TST")
	err := svc.LeaveAlliance(context.Background(), 1)
	if err == nil || !strings.Contains(err.Error(), "founder cannot leave") {
		t.Fatalf("expected founder cannot leave error, got: %v", err)
	}
}

func TestLeaveAlliance_NotInAlliance(t *testing.T) {
	svc := NewAllianceService(newMockRepo(), "http://localhost:8082")
	err := svc.LeaveAlliance(context.Background(), 1)
	if err == nil || !strings.Contains(err.Error(), "not in an alliance") {
		t.Fatalf("expected not in alliance error, got: %v", err)
	}
}

func TestTransferFounder_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewAllianceService(repo, "http://localhost:8082")

	alliance, _ := svc.CreateAlliance(context.Background(), 1, "Test", "TST")
	svc.ApplyToAlliance(context.Background(), 2, alliance.ID)

	err := svc.TransferFounder(context.Background(), 1, 2)
	if err != nil {
		t.Fatal(err)
	}

	m1, _ := repo.GetMember(context.Background(), 1)
	if m1.Role != "officer" {
		t.Errorf("expected old founder role officer, got %s", m1.Role)
	}

	m2, _ := repo.GetMember(context.Background(), 2)
	if m2.Role != "founder" {
		t.Errorf("expected new founder role founder, got %s", m2.Role)
	}
}

func TestTransferFounder_NotFounder(t *testing.T) {
	repo := newMockRepo()
	svc := NewAllianceService(repo, "http://localhost:8082")

	alliance, _ := svc.CreateAlliance(context.Background(), 1, "Test", "TST")
	svc.ApplyToAlliance(context.Background(), 2, alliance.ID)

	err := svc.TransferFounder(context.Background(), 2, 3)
	if err == nil || !strings.Contains(err.Error(), "only the founder") {
		t.Fatalf("expected only founder error, got: %v", err)
	}
}

func TestTransferFounder_TargetNotInAlliance(t *testing.T) {
	repo := newMockRepo()
	svc := NewAllianceService(repo, "http://localhost:8082")

	svc.CreateAlliance(context.Background(), 1, "Test", "TST")
	err := svc.TransferFounder(context.Background(), 1, 999)
	if err == nil || !strings.Contains(err.Error(), "not in any alliance") {
		t.Fatalf("expected target not in alliance error, got: %v", err)
	}
}

func TestGetMyAlliance_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewAllianceService(repo, "http://localhost:8082")

	alliance, _ := svc.CreateAlliance(context.Background(), 1, "Test", "TST")
	gotAlliance, member, members, err := svc.GetMyAlliance(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if gotAlliance.ID != alliance.ID {
		t.Errorf("expected alliance %d, got %d", alliance.ID, gotAlliance.ID)
	}
	if member.Role != "founder" {
		t.Errorf("expected role founder, got %s", member.Role)
	}
	if len(members) != 1 {
		t.Errorf("expected 1 member, got %d", len(members))
	}
}

func TestGetMyAlliance_NotInAlliance(t *testing.T) {
	svc := NewAllianceService(newMockRepo(), "http://localhost:8082")
	_, _, _, err := svc.GetMyAlliance(context.Background(), 1)
	if err == nil || !strings.Contains(err.Error(), "not in an alliance") {
		t.Fatalf("expected not in alliance error, got: %v", err)
	}
}

func TestBankDeposit_Success(t *testing.T) {
	callCount := 0
	ts := planetServiceMock(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	})
	defer ts.Close()

	repo := newMockRepo()
	svc := NewAllianceService(repo, ts.URL)

	alliance, _ := svc.CreateAlliance(context.Background(), 1, "Test", "TST")
	bank, err := svc.BankDeposit(context.Background(), 1, 1, 100, 200, 300)
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
	if callCount != 3 {
		t.Errorf("expected 3 planet calls, got %d", callCount)
	}
	_ = alliance
}

func TestBankDeposit_NotMember(t *testing.T) {
	svc := NewAllianceService(newMockRepo(), "http://localhost:8082")
	_, err := svc.BankDeposit(context.Background(), 1, 1, 100, 0, 0)
	if err == nil || !strings.Contains(err.Error(), "not in an alliance") {
		t.Fatalf("expected not in alliance error, got: %v", err)
	}
}

func TestBankDeposit_PlanetServiceError(t *testing.T) {
	ts := planetServiceMock(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "insufficient metal"})
	})
	defer ts.Close()

	repo := newMockRepo()
	svc := NewAllianceService(repo, ts.URL)

	svc.CreateAlliance(context.Background(), 1, "Test", "TST")
	_, err := svc.BankDeposit(context.Background(), 1, 1, 100, 0, 0)
	if err == nil || !strings.Contains(err.Error(), "insufficient metal") {
		t.Fatalf("expected insufficient metal error, got: %v", err)
	}
}

func TestBankDeposit_Accumulates(t *testing.T) {
	ts := planetServiceMock(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	})
	defer ts.Close()

	repo := newMockRepo()
	svc := NewAllianceService(repo, ts.URL)

	svc.CreateAlliance(context.Background(), 1, "Test", "TST")
	svc.BankDeposit(context.Background(), 1, 1, 100, 200, 300)
	bank, err := svc.BankDeposit(context.Background(), 1, 1, 50, 60, 70)
	if err != nil {
		t.Fatal(err)
	}
	if bank.Metal != 150 {
		t.Errorf("expected 150 metal, got %d", bank.Metal)
	}
	if bank.Crystal != 260 {
		t.Errorf("expected 260 crystal, got %d", bank.Crystal)
	}
	if bank.Gas != 370 {
		t.Errorf("expected 370 gas, got %d", bank.Gas)
	}
}

func TestBankWithdraw_Success(t *testing.T) {
	callCount := 0
	ts := planetServiceMock(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	})
	defer ts.Close()

	repo := newMockRepo()
	svc := NewAllianceService(repo, ts.URL)

	svc.CreateAlliance(context.Background(), 1, "Test", "TST")
	svc.BankDeposit(context.Background(), 1, 1, 500, 500, 500)

	bank, err := svc.BankWithdraw(context.Background(), 1, 1, 100, 200, 300)
	if err != nil {
		t.Fatal(err)
	}
	if bank.Metal != 400 {
		t.Errorf("expected 400 metal, got %d", bank.Metal)
	}
	if bank.Crystal != 300 {
		t.Errorf("expected 300 crystal, got %d", bank.Crystal)
	}
	if bank.Gas != 200 {
		t.Errorf("expected 200 gas, got %d", bank.Gas)
	}
	if callCount != 6 {
		t.Errorf("expected 6 planet calls (3 deposit + 3 withdraw), got %d", callCount)
	}
}

func TestBankWithdraw_MemberCannotWithdraw(t *testing.T) {
	ts := planetServiceMock(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	})
	defer ts.Close()

	repo := newMockRepo()
	svc := NewAllianceService(repo, ts.URL)

	alliance, _ := svc.CreateAlliance(context.Background(), 1, "Test", "TST")
	svc.ApplyToAlliance(context.Background(), 2, alliance.ID)
	svc.BankDeposit(context.Background(), 1, 1, 500, 500, 500)

	_, err := svc.BankWithdraw(context.Background(), 2, 2, 100, 0, 0)
	if err == nil || !strings.Contains(err.Error(), "only officers and founders") {
		t.Fatalf("expected permission error, got: %v", err)
	}
}

func TestBankWithdraw_InsufficientFunds(t *testing.T) {
	ts := planetServiceMock(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	})
	defer ts.Close()

	repo := newMockRepo()
	svc := NewAllianceService(repo, ts.URL)

	svc.CreateAlliance(context.Background(), 1, "Test", "TST")
	_, err := svc.BankWithdraw(context.Background(), 1, 1, 999999, 0, 0)
	if err == nil || !strings.Contains(err.Error(), "insufficient metal") {
		t.Fatalf("expected insufficient funds error, got: %v", err)
	}
}

func TestGetBank_Success(t *testing.T) {
	ts := planetServiceMock(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	})
	defer ts.Close()

	repo := newMockRepo()
	svc := NewAllianceService(repo, ts.URL)

	svc.CreateAlliance(context.Background(), 1, "Test", "TST")
	svc.BankDeposit(context.Background(), 1, 1, 100, 200, 300)

	bank, err := svc.GetBank(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if bank.Metal != 100 {
		t.Errorf("expected 100 metal, got %d", bank.Metal)
	}
}

func TestGetBank_NotMember(t *testing.T) {
	svc := NewAllianceService(newMockRepo(), "http://localhost:8082")
	_, err := svc.GetBank(context.Background(), 1)
	if err == nil || !strings.Contains(err.Error(), "not in an alliance") {
		t.Fatalf("expected not in alliance error, got: %v", err)
	}
}

func TestGetPlayerAlliance_InAlliance(t *testing.T) {
	repo := newMockRepo()
	svc := NewAllianceService(repo, "http://localhost:8082")

	svc.CreateAlliance(context.Background(), 1, "Test", "TST")
	resp := svc.GetPlayerAlliance(context.Background(), 1)
	if !resp.InAlliance {
		t.Error("expected in_alliance to be true")
	}
	if resp.AllianceTag != "TST" {
		t.Errorf("expected tag TST, got %s", resp.AllianceTag)
	}
	if resp.Role != "founder" {
		t.Errorf("expected role founder, got %s", resp.Role)
	}
}

func TestGetPlayerAlliance_NotInAlliance(t *testing.T) {
	svc := NewAllianceService(newMockRepo(), "http://localhost:8082")
	resp := svc.GetPlayerAlliance(context.Background(), 1)
	if resp.InAlliance {
		t.Error("expected in_alliance to be false")
	}
}

func TestCreateAlliance_TrimsWhitespace(t *testing.T) {
	repo := newMockRepo()
	svc := NewAllianceService(repo, "http://localhost:8082")

	alliance, err := svc.CreateAlliance(context.Background(), 1, "  Test Alliance  ", "  TA  ")
	if err != nil {
		t.Fatal(err)
	}
	if alliance.Name != "Test Alliance" {
		t.Errorf("expected trimmed name 'Test Alliance', got '%s'", alliance.Name)
	}
	if alliance.Tag != "TA" {
		t.Errorf("expected trimmed tag 'TA', got '%s'", alliance.Tag)
	}
}

func TestOfficerCanWithdraw(t *testing.T) {
	ts := planetServiceMock(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	})
	defer ts.Close()

	repo := newMockRepo()
	svc := NewAllianceService(repo, ts.URL)

	alliance, _ := svc.CreateAlliance(context.Background(), 1, "Test", "TST")
	svc.ApplyToAlliance(context.Background(), 2, alliance.ID)
	svc.TransferFounder(context.Background(), 1, 2)
	svc.BankDeposit(context.Background(), 1, 1, 100, 100, 100)

	bank, err := svc.BankWithdraw(context.Background(), 2, 1, 50, 50, 50)
	if err != nil {
		t.Fatalf("expected officer to be able to withdraw, got: %v", err)
	}
	if bank.Metal != 50 {
		t.Errorf("expected 50 metal, got %d", bank.Metal)
	}
}
