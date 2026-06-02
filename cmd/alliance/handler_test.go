package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func setupTestRouter(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Post("/api/alliance/create", h.CreateAlliance)
	r.Post("/api/alliance/apply", h.ApplyToAlliance)
	r.Post("/api/alliance/leave", h.LeaveAlliance)
	r.Post("/api/alliance/transfer", h.TransferFounder)
	r.Get("/api/alliance/my", h.GetMyAlliance)
	r.Post("/api/alliance/bank/deposit", h.BankDeposit)
	r.Post("/api/alliance/bank/withdraw", h.BankWithdraw)
	r.Get("/api/alliance/bank", h.GetBank)
	r.Post("/internal/alliance/player", h.InternalGetPlayerAlliance)
	return r
}

func TestCreateAlliance_NoAuth(t *testing.T) {
	svc := NewAllianceService(newMockRepo(), "http://localhost:8082")
	h := NewHandler(svc)
	mux := setupTestRouter(h)
	req := httptest.NewRequest("POST", "/api/alliance/create", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestCreateAlliance_InvalidAuth(t *testing.T) {
	svc := NewAllianceService(newMockRepo(), "http://localhost:8082")
	h := NewHandler(svc)
	mux := setupTestRouter(h)
	req := httptest.NewRequest("POST", "/api/alliance/create", nil)
	req.Header.Set("X-User-ID", "notanumber")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestCreateAlliance_InvalidJSON(t *testing.T) {
	svc := NewAllianceService(newMockRepo(), "http://localhost:8082")
	h := NewHandler(svc)
	mux := setupTestRouter(h)
	body := bytes.NewReader([]byte(`not json`))
	req := httptest.NewRequest("POST", "/api/alliance/create", body)
	req.Header.Set("X-User-ID", "1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandler_CreateAlliance_Success(t *testing.T) {
	svc := NewAllianceService(newMockRepo(), "http://localhost:8082")
	h := NewHandler(svc)
	mux := setupTestRouter(h)
	body := bytes.NewReader([]byte(`{"name":"Test Alliance","tag":"TA"}`))
	req := httptest.NewRequest("POST", "/api/alliance/create", body)
	req.Header.Set("X-User-ID", "1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp AllianceResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal("decode response:", err)
	}
	if resp.Name != "Test Alliance" {
		t.Errorf("expected name Test Alliance, got %s", resp.Name)
	}
	if resp.Tag != "TA" {
		t.Errorf("expected tag TA, got %s", resp.Tag)
	}
	if resp.MemberCount != 1 {
		t.Errorf("expected member_count 1, got %d", resp.MemberCount)
	}
}

func TestCreateAlliance_Duplicate(t *testing.T) {
	svc := NewAllianceService(newMockRepo(), "http://localhost:8082")
	h := NewHandler(svc)
	mux := setupTestRouter(h)

	body := bytes.NewReader([]byte(`{"name":"Test Alliance","tag":"TA"}`))
	req := httptest.NewRequest("POST", "/api/alliance/create", body)
	req.Header.Set("X-User-ID", "1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	body2 := bytes.NewReader([]byte(`{"name":"Test Alliance","tag":"TB"}`))
	req2 := httptest.NewRequest("POST", "/api/alliance/create", body2)
	req2.Header.Set("X-User-ID", "2")
	w2 := httptest.NewRecorder()
	mux.ServeHTTP(w2, req2)

	if w2.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for duplicate, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestApplyToAlliance_MissingAllianceID(t *testing.T) {
	svc := NewAllianceService(newMockRepo(), "http://localhost:8082")
	h := NewHandler(svc)
	mux := setupTestRouter(h)

	body := bytes.NewReader([]byte(`{}`))
	req := httptest.NewRequest("POST", "/api/alliance/apply", body)
	req.Header.Set("X-User-ID", "1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandler_ApplyToAlliance_Success(t *testing.T) {
	svc := NewAllianceService(newMockRepo(), "http://localhost:8082")
	h := NewHandler(svc)
	mux := setupTestRouter(h)

	body := bytes.NewReader([]byte(`{"name":"Test Alliance","tag":"TA"}`))
	req := httptest.NewRequest("POST", "/api/alliance/create", body)
	req.Header.Set("X-User-ID", "1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var created AllianceResponse
	json.NewDecoder(w.Body).Decode(&created)

	body2 := bytes.NewReader([]byte(`{"alliance_id":1}`))
	req2 := httptest.NewRequest("POST", "/api/alliance/apply", body2)
	req2.Header.Set("X-User-ID", "2")
	w2 := httptest.NewRecorder()
	mux.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(w2.Body).Decode(&resp)
	if resp["role"] != "pending" {
		t.Errorf("expected role pending, got %v", resp["role"])
	}
}

func TestLeaveAlliance_NoAuth(t *testing.T) {
	svc := NewAllianceService(newMockRepo(), "http://localhost:8082")
	h := NewHandler(svc)
	mux := setupTestRouter(h)
	req := httptest.NewRequest("POST", "/api/alliance/leave", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestGetMyAlliance_NoAuth(t *testing.T) {
	svc := NewAllianceService(newMockRepo(), "http://localhost:8082")
	h := NewHandler(svc)
	mux := setupTestRouter(h)
	req := httptest.NewRequest("GET", "/api/alliance/my", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestHandler_GetMyAlliance_Success(t *testing.T) {
	svc := NewAllianceService(newMockRepo(), "http://localhost:8082")
	h := NewHandler(svc)
	mux := setupTestRouter(h)

	body := bytes.NewReader([]byte(`{"name":"Test Alliance","tag":"TA"}`))
	req := httptest.NewRequest("POST", "/api/alliance/create", body)
	req.Header.Set("X-User-ID", "1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	req2 := httptest.NewRequest("GET", "/api/alliance/my", nil)
	req2.Header.Set("X-User-ID", "1")
	w2 := httptest.NewRecorder()
	mux.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	var resp AllianceResponse
	if err := json.NewDecoder(w2.Body).Decode(&resp); err != nil {
		t.Fatal("decode response:", err)
	}
	if resp.Role != "founder" {
		t.Errorf("expected role founder, got %s", resp.Role)
	}
	if len(resp.Members) != 1 {
		t.Errorf("expected 1 member, got %d", len(resp.Members))
	}
}

func TestInternalGetPlayerAlliance(t *testing.T) {
	svc := NewAllianceService(newMockRepo(), "http://localhost:8082")
	h := NewHandler(svc)
	mux := setupTestRouter(h)

	body := bytes.NewReader([]byte(`{"player_id":1}`))
	req := httptest.NewRequest("POST", "/internal/alliance/player", body)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var resp PlayerAllianceResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal("decode response:", err)
	}
	if resp.InAlliance {
		t.Error("expected in_alliance to be false")
	}
}

func TestInternalGetPlayerAlliance_InAlliance(t *testing.T) {
	svc := NewAllianceService(newMockRepo(), "http://localhost:8082")
	h := NewHandler(svc)
	mux := setupTestRouter(h)

	createBody := bytes.NewReader([]byte(`{"name":"Test","tag":"TST"}`))
	creq := httptest.NewRequest("POST", "/api/alliance/create", createBody)
	creq.Header.Set("X-User-ID", "1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, creq)

	body := bytes.NewReader([]byte(`{"player_id":1}`))
	req := httptest.NewRequest("POST", "/internal/alliance/player", body)
	w2 := httptest.NewRecorder()
	mux.ServeHTTP(w2, req)

	var resp PlayerAllianceResponse
	if err := json.NewDecoder(w2.Body).Decode(&resp); err != nil {
		t.Fatal("decode response:", err)
	}
	if !resp.InAlliance {
		t.Error("expected in_alliance to be true")
	}
	if resp.Role != "founder" {
		t.Errorf("expected role founder, got %s", resp.Role)
	}
	if resp.AllianceTag != "TST" {
		t.Errorf("expected tag TST, got %s", resp.AllianceTag)
	}
}


