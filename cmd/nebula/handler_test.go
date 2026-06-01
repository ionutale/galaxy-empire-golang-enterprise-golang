package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func setupTestRouter(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Post("/api/nebula/start", h.StartExpedition)
	r.Get("/api/nebula/expeditions", h.ListExpeditions)
	r.Get("/api/nebula/expeditions/{id}", h.GetExpedition)
	r.Post("/api/nebula/dm-balance", h.DMBalance)
	r.Post("/api/nebula/dm-spend", h.DMSpend)
	r.Post("/api/nebula/dm/speed-up", h.DMSpeedUp)
	r.Post("/api/nebula/dm/estimate-cost", h.DMEstimateCost)
	r.Get("/api/nebula/dm/transactions", h.DMTransactions)
	r.Post("/api/nebula/commanders/hire", h.HireCommander)
	r.Get("/api/nebula/commanders", h.ListCommanders)
	r.Get("/api/nebula/commanders/available", h.AvailableCommanders)
	r.Post("/internal/nebula/commanders/active", h.InternalActiveCommanders)
	r.Post("/api/nebula/daily-gift/claim", h.ClaimDailyGift)
	r.Get("/api/nebula/daily-gift/status", h.GetDailyGiftStatus)
	r.Get("/api/nebula/daily-tasks", h.GetDailyTasks)
	r.Post("/api/nebula/daily-tasks/{id}/progress", h.UpdateTaskProgress)
	r.Post("/api/nebula/daily-tasks/{id}/claim", h.ClaimTaskReward)
	r.Post("/api/nebula/daily-tasks/reroll", h.RerollTask)
	r.Post("/api/nebula/daily-tasks/claim-all", h.ClaimAllTasks)
	return r
}

func TestStartExpedition_NoAuth(t *testing.T) {
	svc := NewNebulaService(newMockRepo(), "http://localhost:8082")
	h := NewHandler(svc)
	mux := setupTestRouter(h)
	req := httptest.NewRequest("POST", "/api/nebula/start", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestStartExpedition_InvalidAuth(t *testing.T) {
	svc := NewNebulaService(newMockRepo(), "http://localhost:8082")
	h := NewHandler(svc)
	mux := setupTestRouter(h)
	req := httptest.NewRequest("POST", "/api/nebula/start", nil)
	req.Header.Set("X-User-ID", "notanumber")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestStartExpedition_MissingPlanetID(t *testing.T) {
	svc := NewNebulaService(newMockRepo(), "http://localhost:8082")
	h := NewHandler(svc)
	mux := setupTestRouter(h)
	body := bytes.NewReader([]byte(`{"ships":{"light_fighter":10}}`))
	req := httptest.NewRequest("POST", "/api/nebula/start", body)
	req.Header.Set("X-User-ID", "1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestStartExpedition_InvalidJSON(t *testing.T) {
	svc := NewNebulaService(newMockRepo(), "http://localhost:8082")
	h := NewHandler(svc)
	mux := setupTestRouter(h)
	body := bytes.NewReader([]byte(`not json`))
	req := httptest.NewRequest("POST", "/api/nebula/start", body)
	req.Header.Set("X-User-ID", "1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestListExpeditions_NoAuth(t *testing.T) {
	svc := NewNebulaService(newMockRepo(), "http://localhost:8082")
	h := NewHandler(svc)
	mux := setupTestRouter(h)
	req := httptest.NewRequest("GET", "/api/nebula/expeditions", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestListExpeditions_Empty(t *testing.T) {
	svc := NewNebulaService(newMockRepo(), "http://localhost:8082")
	h := NewHandler(svc)
	mux := setupTestRouter(h)
	req := httptest.NewRequest("GET", "/api/nebula/expeditions", nil)
	req.Header.Set("X-User-ID", "1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp []ExpeditionResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal("decode response:", err)
	}
	if len(resp) != 0 {
		t.Errorf("expected empty list, got %d items", len(resp))
	}
}

func TestGetExpedition_NoAuth(t *testing.T) {
	svc := NewNebulaService(newMockRepo(), "http://localhost:8082")
	h := NewHandler(svc)
	mux := setupTestRouter(h)
	req := httptest.NewRequest("GET", "/api/nebula/expeditions/1", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestDMBalance_NoAuth(t *testing.T) {
	svc := NewNebulaService(newMockRepo(), "http://localhost:8082")
	h := NewHandler(svc)
	mux := setupTestRouter(h)
	req := httptest.NewRequest("POST", "/api/nebula/dm-balance", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestDMBalance_Success(t *testing.T) {
	svc := NewNebulaService(newMockRepo(), "http://localhost:8082")
	h := NewHandler(svc)
	mux := setupTestRouter(h)
	req := httptest.NewRequest("POST", "/api/nebula/dm-balance", nil)
	req.Header.Set("X-User-ID", "1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp PlayerDarkMatter
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal("decode response:", err)
	}
	if resp.PlayerID != 1 {
		t.Errorf("expected player_id 1, got %d", resp.PlayerID)
	}
	if resp.Balance != 0 {
		t.Errorf("expected balance 0, got %d", resp.Balance)
	}
}

func TestDMSpend_NoAuth(t *testing.T) {
	svc := NewNebulaService(newMockRepo(), "http://localhost:8082")
	h := NewHandler(svc)
	mux := setupTestRouter(h)
	body := bytes.NewReader([]byte(`{"amount":10,"reason":"test"}`))
	req := httptest.NewRequest("POST", "/api/nebula/dm-spend", body)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestDMSpend_InvalidAmount(t *testing.T) {
	svc := NewNebulaService(newMockRepo(), "http://localhost:8082")
	h := NewHandler(svc)
	mux := setupTestRouter(h)
	body := bytes.NewReader([]byte(`{"amount":-5,"reason":"test"}`))
	req := httptest.NewRequest("POST", "/api/nebula/dm-spend", body)
	req.Header.Set("X-User-ID", "1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestDMSpend_Success(t *testing.T) {
	repo := newMockRepo()
	repo.AddDarkMatter(context.Background(), 1, 100)
	svc := NewNebulaService(repo, "http://localhost:8082")
	h := NewHandler(svc)
	mux := setupTestRouter(h)
	body := bytes.NewReader([]byte(`{"amount":30,"reason":"test spend"}`))
	req := httptest.NewRequest("POST", "/api/nebula/dm-spend", body)
	req.Header.Set("X-User-ID", "1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]int
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal("decode response:", err)
	}
	if resp["new_balance"] != 70 {
		t.Errorf("expected new_balance 70, got %d", resp["new_balance"])
	}
}

func TestDMSpend_Insufficient(t *testing.T) {
	repo := newMockRepo()
	repo.AddDarkMatter(context.Background(), 1, 10)
	svc := NewNebulaService(repo, "http://localhost:8082")
	h := NewHandler(svc)
	mux := setupTestRouter(h)
	body := bytes.NewReader([]byte(`{"amount":20,"reason":"too much"}`))
	req := httptest.NewRequest("POST", "/api/nebula/dm-spend", body)
	req.Header.Set("X-User-ID", "1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDMSpeedUp_NoAuth(t *testing.T) {
	svc := NewNebulaService(newMockRepo(), "http://localhost:8082")
	h := NewHandler(svc)
	mux := setupTestRouter(h)
	body := bytes.NewReader([]byte(`{"target_type":"research","target_id":1,"seconds_remaining":900}`))
	req := httptest.NewRequest("POST", "/api/nebula/dm/speed-up", body)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestDMSpeedUp_Success(t *testing.T) {
	repo := newMockRepo()
	repo.AddDarkMatter(context.Background(), 1, 100)
	svc := NewNebulaService(repo, "http://localhost:8082")
	h := NewHandler(svc)
	mux := setupTestRouter(h)
	body := bytes.NewReader([]byte(`{"target_type":"research","target_id":1,"seconds_remaining":900}`))
	req := httptest.NewRequest("POST", "/api/nebula/dm/speed-up", body)
	req.Header.Set("X-User-ID", "1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]int
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal("decode response:", err)
	}
	if resp["dm_cost"] != 1 {
		t.Errorf("expected dm_cost 1, got %d", resp["dm_cost"])
	}
	if resp["seconds_saved"] != 900 {
		t.Errorf("expected seconds_saved 900, got %d", resp["seconds_saved"])
	}
}

func TestDMSpeedUp_InsufficientDM(t *testing.T) {
	repo := newMockRepo()
	svc := NewNebulaService(repo, "http://localhost:8082")
	h := NewHandler(svc)
	mux := setupTestRouter(h)
	body := bytes.NewReader([]byte(`{"target_type":"research","target_id":1,"seconds_remaining":900}`))
	req := httptest.NewRequest("POST", "/api/nebula/dm/speed-up", body)
	req.Header.Set("X-User-ID", "1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDMSpeedUp_InvalidTargetType(t *testing.T) {
	repo := newMockRepo()
	repo.AddDarkMatter(context.Background(), 1, 100)
	svc := NewNebulaService(repo, "http://localhost:8082")
	h := NewHandler(svc)
	mux := setupTestRouter(h)
	body := bytes.NewReader([]byte(`{"target_type":"invalid","target_id":1,"seconds_remaining":900}`))
	req := httptest.NewRequest("POST", "/api/nebula/dm/speed-up", body)
	req.Header.Set("X-User-ID", "1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDMEstimateCost_Success(t *testing.T) {
	svc := NewNebulaService(newMockRepo(), "http://localhost:8082")
	h := NewHandler(svc)
	mux := setupTestRouter(h)
	body := bytes.NewReader([]byte(`{"seconds":1800}`))
	req := httptest.NewRequest("POST", "/api/nebula/dm/estimate-cost", body)
	req.Header.Set("X-User-ID", "1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]int
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal("decode response:", err)
	}
	if resp["dm_cost"] != 2 {
		t.Errorf("expected dm_cost 2, got %d", resp["dm_cost"])
	}
}

func TestDMTransactions_NoAuth(t *testing.T) {
	svc := NewNebulaService(newMockRepo(), "http://localhost:8082")
	h := NewHandler(svc)
	mux := setupTestRouter(h)
	req := httptest.NewRequest("GET", "/api/nebula/dm/transactions", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestDMTransactions_Success(t *testing.T) {
	repo := newMockRepo()
	repo.AddDarkMatter(context.Background(), 1, 100)
	repo.SpendDarkMatter(context.Background(), 1, 30)
	repo.AddDMTransaction(context.Background(), 1, -30, 70, "test")
	svc := NewNebulaService(repo, "http://localhost:8082")
	h := NewHandler(svc)
	mux := setupTestRouter(h)
	req := httptest.NewRequest("GET", "/api/nebula/dm/transactions", nil)
	req.Header.Set("X-User-ID", "1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp []DMTransaction
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal("decode response:", err)
	}
	if len(resp) != 1 {
		t.Errorf("expected 1 transaction, got %d", len(resp))
	}
	if resp[0].Amount != -30 {
		t.Errorf("expected amount -30, got %d", resp[0].Amount)
	}
}

func TestHireCommander_NoAuth(t *testing.T) {
	svc := NewNebulaService(newMockRepo(), "http://localhost:8082")
	h := NewHandler(svc)
	mux := setupTestRouter(h)
	body := bytes.NewReader([]byte(`{"commander_type":"commander"}`))
	req := httptest.NewRequest("POST", "/api/nebula/commanders/hire", body)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestHireCommander_Success(t *testing.T) {
	repo := newMockRepo()
	repo.AddDarkMatter(context.Background(), 1, 100)
	svc := NewNebulaService(repo, "http://localhost:8082")
	h := NewHandler(svc)
	mux := setupTestRouter(h)
	body := bytes.NewReader([]byte(`{"commander_type":"commander"}`))
	req := httptest.NewRequest("POST", "/api/nebula/commanders/hire", body)
	req.Header.Set("X-User-ID", "1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp CommanderEntry
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal("decode response:", err)
	}
	if resp.CommanderType != "commander" {
		t.Errorf("expected commander_type commander, got %s", resp.CommanderType)
	}
	if resp.Name != "Commander" {
		t.Errorf("expected name Commander, got %s", resp.Name)
	}
	if resp.DaysRemaining <= 0 {
		t.Errorf("expected positive days_remaining, got %d", resp.DaysRemaining)
	}
}

func TestHireCommander_InvalidType(t *testing.T) {
	repo := newMockRepo()
	repo.AddDarkMatter(context.Background(), 1, 100)
	svc := NewNebulaService(repo, "http://localhost:8082")
	h := NewHandler(svc)
	mux := setupTestRouter(h)
	body := bytes.NewReader([]byte(`{"commander_type":"invalid_type"}`))
	req := httptest.NewRequest("POST", "/api/nebula/commanders/hire", body)
	req.Header.Set("X-User-ID", "1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListCommanders_NoAuth(t *testing.T) {
	svc := NewNebulaService(newMockRepo(), "http://localhost:8082")
	h := NewHandler(svc)
	mux := setupTestRouter(h)
	req := httptest.NewRequest("GET", "/api/nebula/commanders", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestListCommanders_Success(t *testing.T) {
	repo := newMockRepo()
	repo.AddDarkMatter(context.Background(), 1, 100)
	svc := NewNebulaService(repo, "http://localhost:8082")
	h := NewHandler(svc)
	mux := setupTestRouter(h)
	// hire a commander first
	hireBody := bytes.NewReader([]byte(`{"commander_type":"admiral"}`))
	hireReq := httptest.NewRequest("POST", "/api/nebula/commanders/hire", hireBody)
	hireReq.Header.Set("X-User-ID", "1")
	hireW := httptest.NewRecorder()
	mux.ServeHTTP(hireW, hireReq)
	if hireW.Code != http.StatusOK {
		t.Fatalf("hire failed: %s", hireW.Body.String())
	}
	// list commanders
	req := httptest.NewRequest("GET", "/api/nebula/commanders", nil)
	req.Header.Set("X-User-ID", "1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp []CommanderEntry
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal("decode response:", err)
	}
	if len(resp) != 1 {
		t.Errorf("expected 1 commander, got %d", len(resp))
	}
	if resp[0].CommanderType != "admiral" {
		t.Errorf("expected admiral, got %s", resp[0].CommanderType)
	}
}

func TestAvailableCommanders(t *testing.T) {
	svc := NewNebulaService(newMockRepo(), "http://localhost:8082")
	h := NewHandler(svc)
	mux := setupTestRouter(h)
	req := httptest.NewRequest("GET", "/api/nebula/commanders/available", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp []CommanderConfig
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal("decode response:", err)
	}
	if len(resp) != 6 {
		t.Errorf("expected 6 commanders, got %d", len(resp))
	}
}

func TestInternalActiveCommanders(t *testing.T) {
	repo := newMockRepo()
	repo.AddDarkMatter(context.Background(), 1, 100)
	svc := NewNebulaService(repo, "http://localhost:8082")
	h := NewHandler(svc)
	mux := setupTestRouter(h)
	// hire a commander
	svc.HireCommander(context.Background(), 1, "commander")
	// internal check
	body := bytes.NewReader([]byte(`{"player_id":1}`))
	req := httptest.NewRequest("POST", "/internal/nebula/commanders/active", body)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp []struct {
		CommanderType string `json:"commander_type"`
		Level         int    `json:"level"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal("decode response:", err)
	}
	if len(resp) != 1 {
		t.Errorf("expected 1 commander, got %d", len(resp))
	}
	if resp[0].CommanderType != "commander" {
		t.Errorf("expected commander, got %s", resp[0].CommanderType)
	}
}

func TestGetExpedition_NotFound(t *testing.T) {
	svc := NewNebulaService(newMockRepo(), "http://localhost:8082")
	h := NewHandler(svc)
	mux := setupTestRouter(h)
	req := httptest.NewRequest("GET", "/api/nebula/expeditions/999", nil)
	req.Header.Set("X-User-ID", "1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}
