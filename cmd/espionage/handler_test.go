package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func setupTestHandler() *Handler {
	svc := NewEspionageService(newMockRepo(), "http://localhost:8082")
	return NewHandler(svc)
}

func setupTestRouter(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Post("/api/espionage/probe", h.Probe)
	r.Get("/api/espionage/reports", h.ListReports)
	r.Get("/api/espionage/reports/{id}", h.GetReport)
	r.Delete("/api/espionage/reports/{id}", h.DeleteReport)
	return r
}

func TestProbe_NoUserID(t *testing.T) {
	router := setupTestRouter(setupTestHandler())
	body := `{"target_galaxy":1,"target_system":1,"target_position":1,"planet_id":1}`
	req := httptest.NewRequest("POST", "/api/espionage/probe", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestProbe_InvalidTarget(t *testing.T) {
	router := setupTestRouter(setupTestHandler())
	body := `{"target_galaxy":0,"target_system":1,"target_position":1,"planet_id":1}`
	req := httptest.NewRequest("POST", "/api/espionage/probe", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "1")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestProbe_PlanetServiceUnreachable(t *testing.T) {
	svc := NewEspionageService(newMockRepo(), "http://localhost:1")
	h := NewHandler(svc)
	router := setupTestRouter(h)

	body := `{"target_galaxy":1,"target_system":1,"target_position":1,"planet_id":1}`
	req := httptest.NewRequest("POST", "/api/espionage/probe", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "1")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 from unreachable planet service, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestProbe_Success(t *testing.T) {
	planetSvc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/internal/ships/deduct":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]bool{"ok": true})
		case "/internal/planet/info":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(PlanetInfo{
				PlanetID: 10, PlayerID: 2,
				Metal: 10000, Crystal: 5000, Gas: 2000,
				Ships: map[string]int{"cargo": 10, "light_fighter": 5},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer planetSvc.Close()

	svc := NewEspionageService(newMockRepo(), planetSvc.URL)
	h := NewHandler(svc)
	router := setupTestRouter(h)

	body := `{"target_galaxy":1,"target_system":1,"target_position":1,"planet_id":1}`
	req := httptest.NewRequest("POST", "/api/espionage/probe", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "1")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp ProbeResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal("decode response:", err)
	}
	if resp.ReportID == 0 {
		t.Error("expected non-zero report_id")
	}
}

func TestListReports_NoUserID(t *testing.T) {
	router := setupTestRouter(setupTestHandler())
	req := httptest.NewRequest("GET", "/api/espionage/reports", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestListReports_Success(t *testing.T) {
	mock := newMockRepo()
	mock.reports = []EspionageReport{
		{ID: 1, PlayerID: 1, TargetPlayerID: 2, TargetGalaxy: 1, TargetSystem: 1, TargetPosition: 1, DetailLevel: 5},
	}
	svc := NewEspionageService(mock, "http://localhost:8082")
	h := NewHandler(svc)
	router := setupTestRouter(h)

	req := httptest.NewRequest("GET", "/api/espionage/reports", nil)
	req.Header.Set("X-User-ID", "1")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var reports []EspionageReportResponse
	if err := json.NewDecoder(rec.Body).Decode(&reports); err != nil {
		t.Fatal(err)
	}
	if len(reports) != 1 {
		t.Fatalf("expected 1 report, got %d", len(reports))
	}
	if reports[0].ID != 1 {
		t.Errorf("expected report id 1, got %d", reports[0].ID)
	}
}

func TestGetReport_NoUserID(t *testing.T) {
	router := setupTestRouter(setupTestHandler())
	req := httptest.NewRequest("GET", "/api/espionage/reports/1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestGetReport_NotFound(t *testing.T) {
	router := setupTestRouter(setupTestHandler())
	req := httptest.NewRequest("GET", "/api/espionage/reports/999", nil)
	req.Header.Set("X-User-ID", "1")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestGetReport_Success(t *testing.T) {
	mock := newMockRepo()
	mock.CreateReport(nil, EspionageReport{
		PlayerID: 1, TargetPlayerID: 2,
		TargetGalaxy: 1, TargetSystem: 1, TargetPosition: 1,
		DetailLevel: 5,
		Resources:   map[string]int{"metal": 10000, "crystal": 5000, "gas": 2000},
		Fleet:       map[string]int{"cargo": 10},
	})
	svc := NewEspionageService(mock, "http://localhost:8082")
	h := NewHandler(svc)
	router := setupTestRouter(h)

	req := httptest.NewRequest("GET", "/api/espionage/reports/1", nil)
	req.Header.Set("X-User-ID", "1")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp EspionageReportResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.DetailLevel != 5 {
		t.Errorf("expected detail 5, got %d", resp.DetailLevel)
	}
	if resp.Resources["metal"] != 10000 {
		t.Errorf("expected metal 10000, got %d", resp.Resources["metal"])
	}
	if resp.Fleet["cargo"] != 10 {
		t.Errorf("expected cargo 10, got %d", resp.Fleet["cargo"])
	}
}

func TestDeleteReport_NoUserID(t *testing.T) {
	router := setupTestRouter(setupTestHandler())
	req := httptest.NewRequest("DELETE", "/api/espionage/reports/1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestDeleteReport_Success(t *testing.T) {
	mock := newMockRepo()
	mock.CreateReport(nil, EspionageReport{
		PlayerID: 1, TargetPlayerID: 2,
		TargetGalaxy: 1, TargetSystem: 1, TargetPosition: 1,
	})
	svc := NewEspionageService(mock, "http://localhost:8082")
	h := NewHandler(svc)
	router := setupTestRouter(h)

	req := httptest.NewRequest("DELETE", "/api/espionage/reports/1", nil)
	req.Header.Set("X-User-ID", "1")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]bool
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if !resp["ok"] {
		t.Error("expected ok: true")
	}
}
