# Fleet Dispatch Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** New `cmd/fleet` microservice for fleet dispatch with ship selection, target coordinates, and mission type.

**Architecture:** Fleet service (port 8083) calls planet's internal endpoint `/internal/ships/deduct` via HTTP to validate and deduct ships. Same PG instance, separate `fleet` schema.

**Tech Stack:** Go + chi/pgx + pgxpool

---

### Task 1: Planet — DeductShips Repo Method + Internal Handler

**Files:**
- Modify: `cmd/planet/repository.go`
- Modify: `cmd/planet/handler.go`
- Modify: `cmd/planet/main.go`

- [ ] **Step 1: Add DeductPlayerShips to Repository interface + Postgres + mock**

In `cmd/planet/repository.go`, add to interface after `AddPlayerShips`:

```go
	DeductPlayerShips(ctx context.Context, planetID int, ships map[string]int) error
```

Add to PostgresRepo:

```go
func (r *PostgresRepository) DeductPlayerShips(ctx context.Context, planetID int, ships map[string]int) error {
	for shipType, qty := range ships {
		tag, err := r.pool.Exec(ctx, `
			UPDATE planet.player_ships
			SET quantity = quantity - $1
			WHERE planet_id = $2 AND ship_type = $3 AND quantity >= $1
		`, qty, planetID, shipType)
		if err != nil {
			return fmt.Errorf("deduct ship %s: %w", shipType, err)
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("insufficient %s ships", shipType)
		}
	}
	return nil
}
```

Add to mockRepo:

```go
func (m *mockRepo) DeductPlayerShips(ctx context.Context, planetID int, ships map[string]int) error {
	return nil
}
```

- [ ] **Step 2: Add InternalDeductShips handler in handler.go**

```go
func (h *Handler) InternalDeductShips(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PlanetID int            `json:"planet_id"`
		Ships    map[string]int `json:"ships"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if err := h.service.repo.DeductPlayerShips(r.Context(), req.PlanetID, req.Ships); err != nil {
		slog.Error("deduct ships failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
```

- [ ] **Step 3: Add internal route in main.go**

Add before the main routes (no auth middleware):

```go
r.Post("/internal/ships/deduct", h.InternalDeductShips)
```

- [ ] **Step 4: Build check**

Run: `go build ./cmd/planet/...`
Expected: success

- [ ] **Step 5: Commit**

```bash
git add -f cmd/planet/repository.go cmd/planet/handler.go cmd/planet/main.go
git commit -m "feat: internal ship deduction endpoint"
```

---

### Task 2: Fleet Service — Types

**Files:**
- Create: `cmd/fleet/types.go`

- [ ] **Step 1: Create types.go**

```go
package main

type Fleet struct {
	ID              int
	PlayerID        int
	OriginPlanetID  int
	TargetGalaxy    int
	TargetSystem    int
	TargetPosition  int
	Mission         string
	Status          string
	SpeedPct        int
	Ships           map[string]int
}

type FleetResponse struct {
	ID              int            `json:"id"`
	PlayerID        int            `json:"player_id"`
	OriginPlanetID  int            `json:"origin_planet_id"`
	TargetGalaxy    int            `json:"target_galaxy"`
	TargetSystem    int            `json:"target_system"`
	TargetPosition  int            `json:"target_position"`
	Mission         string         `json:"mission"`
	Status          string         `json:"status"`
	SpeedPct        int            `json:"speed_pct"`
	Ships           map[string]int `json:"ships"`
}

type DispatchRequest struct {
	OriginPlanetID int            `json:"origin_planet_id"`
	Ships          map[string]int `json:"ships"`
	TargetGalaxy   int            `json:"target_galaxy"`
	TargetSystem   int            `json:"target_system"`
	TargetPosition int            `json:"target_position"`
	Mission        string         `json:"mission"`
	SpeedPct       int            `json:"speed_pct"`
}
```

- [ ] **Step 2: Build check**

Run: `go build ./cmd/fleet/...`
Expected: success (might need a main.go first — you can skip this step if there's no main.go yet)

Actually, create a minimal main.go first:

```go
package main

func main() {}
```

Then build: `go build ./cmd/fleet/...` → success. Remove this placeholder after Task 3.

- [ ] **Step 3: Commit**

```bash
git add -f cmd/fleet/types.go
git commit -m "feat: fleet types"
```

---

### Task 3: Fleet Service — Main + Migration + Routes

**Files:**
- Create: `cmd/fleet/main.go`

- [ ] **Step 1: Create main.go**

```go
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	databaseURL := getEnv("DATABASE_URL", "postgres://galaxy:galaxy_dev@localhost:5432/galaxy_empire?sslmode=disable")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		slog.Error("connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := runMigrations(ctx, pool); err != nil {
		slog.Error("run migrations", "error", err)
		os.Exit(1)
	}

	planetBaseURL := getEnv("PLANET_SERVICE_URL", "http://localhost:8082")

	repo := NewPostgresRepository(pool)
	svc := NewFleetService(repo, planetBaseURL)
	h := NewFleetHandler(svc)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"fleet"}`))
	})

	r.Get("/api/fleet/my-fleets", h.MyFleets)
	r.Post("/api/fleet/dispatch", h.Dispatch)

	srv := &http.Server{Addr: ":8083", Handler: r}
	go func() {
		slog.Info("fleet service starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("fleet service fatal", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("fleet service shutting down")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	srv.Shutdown(shutdownCtx)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, `
		CREATE SCHEMA IF NOT EXISTS fleet;
		CREATE TABLE IF NOT EXISTS fleet.fleets (
			id SERIAL PRIMARY KEY,
			player_id INT NOT NULL,
			origin_planet_id INT NOT NULL,
			target_galaxy INT NOT NULL,
			target_system INT NOT NULL,
			target_position INT NOT NULL,
			mission VARCHAR(20) NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'stationed',
			speed_pct INT NOT NULL DEFAULT 100,
			ships JSONB NOT NULL DEFAULT '{}',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`); err != nil {
		return err
	}
	return nil
}
```

- [ ] **Step 2: Remove placeholder main.go if you created one in Task 2**

- [ ] **Step 3: Build check**

Run: `go build ./cmd/fleet/...`
Expected: success

- [ ] **Step 4: Commit**

```bash
git add -f cmd/fleet/main.go
git commit -m "feat: fleet service main, migration, routes"
```

---

### Task 4: Fleet Service — Repository

**Files:**
- Create: `cmd/fleet/repository.go`

- [ ] **Step 1: Create repository.go**

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	CreateFleet(ctx context.Context, f Fleet) (Fleet, error)
	ListPlayerFleets(ctx context.Context, playerID int) ([]Fleet, error)
}

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) CreateFleet(ctx context.Context, f Fleet) (Fleet, error) {
	shipsJSON, err := json.Marshal(f.Ships)
	if err != nil {
		return Fleet{}, fmt.Errorf("marshal ships: %w", err)
	}

	var id int
	err = r.pool.QueryRow(ctx, `
		INSERT INTO fleet.fleets (player_id, origin_planet_id, target_galaxy, target_system, target_position, mission, status, speed_pct, ships)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`, f.PlayerID, f.OriginPlanetID, f.TargetGalaxy, f.TargetSystem, f.TargetPosition, f.Mission, f.Status, f.SpeedPct, shipsJSON).Scan(&id)
	if err != nil {
		return Fleet{}, fmt.Errorf("create fleet: %w", err)
	}
	f.ID = id
	return f, nil
}

func (r *PostgresRepository) ListPlayerFleets(ctx context.Context, playerID int) ([]Fleet, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, player_id, origin_planet_id, target_galaxy, target_system, target_position, mission, status, speed_pct, ships
		FROM fleet.fleets
		WHERE player_id = $1
		ORDER BY created_at DESC
	`, playerID)
	if err != nil {
		return nil, fmt.Errorf("list fleets: %w", err)
	}
	defer rows.Close()

	var fleets []Fleet
	for rows.Next() {
		var f Fleet
		var shipsJSON []byte
		if err := rows.Scan(&f.ID, &f.PlayerID, &f.OriginPlanetID, &f.TargetGalaxy, &f.TargetSystem, &f.TargetPosition, &f.Mission, &f.Status, &f.SpeedPct, &shipsJSON); err != nil {
			return nil, fmt.Errorf("scan fleet: %w", err)
		}
		json.Unmarshal(shipsJSON, &f.Ships)
		fleets = append(fleets, f)
	}
	return fleets, rows.Err()
}

type mockRepo struct {
	fleets []Fleet
	nextID int
}

func newMockRepo() *mockRepo {
	return &mockRepo{nextID: 1}
}

func (m *mockRepo) CreateFleet(ctx context.Context, f Fleet) (Fleet, error) {
	f.ID = m.nextID
	m.nextID++
	m.fleets = append(m.fleets, f)
	return f, nil
}

func (m *mockRepo) ListPlayerFleets(ctx context.Context, playerID int) ([]Fleet, error) {
	var result []Fleet
	for _, f := range m.fleets {
		if f.PlayerID == playerID {
			result = append(result, f)
		}
	}
	return result, nil
}
```

- [ ] **Step 2: Build check**

Run: `go build ./cmd/fleet/...`
Expected: success

- [ ] **Step 3: Commit**

```bash
git add -f cmd/fleet/repository.go
git commit -m "feat: fleet repository"
```

---

### Task 5: Fleet Service — Service + Handler

**Files:**
- Create: `cmd/fleet/service.go`
- Create: `cmd/fleet/handler.go`

- [ ] **Step 1: Create service.go**

```go
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type FleetService struct {
	repo           Repository
	planetBaseURL  string
	httpClient     *http.Client
}

func NewFleetService(repo Repository, planetBaseURL string) *FleetService {
	return &FleetService{
		repo:          repo,
		planetBaseURL: planetBaseURL,
		httpClient:    &http.Client{Timeout: 10 * time.Second},
	}
}

var validMissions = map[string]bool{
	"attack": true, "acs_attack": true, "acs_defend": true,
	"transport": true, "deploy": true, "espionage": true,
	"colonize": true, "expedition": true, "recycle": true,
}

func (s *FleetService) DispatchFleet(ctx context.Context, playerID int, req DispatchRequest) (Fleet, error) {
	if !validMissions[req.Mission] {
		return Fleet{}, fmt.Errorf("invalid mission: %s", req.Mission)
	}
	if len(req.Ships) == 0 {
		return Fleet{}, fmt.Errorf("no ships selected")
	}
	if req.SpeedPct < 10 || req.SpeedPct > 100 {
		return Fleet{}, fmt.Errorf("speed must be 10-100")
	}

	// Call planet service to deduct ships
	body, _ := json.Marshal(map[string]any{
		"planet_id": req.OriginPlanetID,
		"ships":     req.Ships,
	})
	resp, err := s.httpClient.Post(s.planetBaseURL+"/internal/ships/deduct", "application/json", bytes.NewReader(body))
	if err != nil {
		return Fleet{}, fmt.Errorf("planet service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return Fleet{}, fmt.Errorf("planet service: %s", string(respBody))
	}

	fleet := Fleet{
		PlayerID:       playerID,
		OriginPlanetID: req.OriginPlanetID,
		TargetGalaxy:   req.TargetGalaxy,
		TargetSystem:   req.TargetSystem,
		TargetPosition: req.TargetPosition,
		Mission:        req.Mission,
		Status:         "stationed",
		SpeedPct:       req.SpeedPct,
		Ships:          req.Ships,
	}

	return s.repo.CreateFleet(ctx, fleet)
}
```

- [ ] **Step 2: Create handler.go**

```go
package main

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
)

type FleetHandler struct {
	service *FleetService
}

func NewFleetHandler(service *FleetService) *FleetHandler {
	return &FleetHandler{service: service}
}

func (h *FleetHandler) MyFleets(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid user"})
		return
	}

	fleets, err := h.service.repo.ListPlayerFleets(r.Context(), userID)
	if err != nil {
		slog.Error("list fleets failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	resp := make([]FleetResponse, len(fleets))
	for i, f := range fleets {
		resp[i] = toFleetResponse(f)
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *FleetHandler) Dispatch(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid user"})
		return
	}

	var req DispatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	fleet, err := h.service.DispatchFleet(r.Context(), userID, req)
	if err != nil {
		slog.Error("dispatch failed", "error", err)
		code := http.StatusBadRequest
		writeJSON(w, code, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, toFleetResponse(fleet))
}

func toFleetResponse(f Fleet) FleetResponse {
	return FleetResponse{
		ID: f.ID, PlayerID: f.PlayerID, OriginPlanetID: f.OriginPlanetID,
		TargetGalaxy: f.TargetGalaxy, TargetSystem: f.TargetSystem, TargetPosition: f.TargetPosition,
		Mission: f.Mission, Status: f.Status, SpeedPct: f.SpeedPct, Ships: f.Ships,
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
```

- [ ] **Step 3: Remove unused import**

Note: `errors` is imported in handler.go but may not be used in the initial version. Remove it if the build complains:

```go
import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
)
```

- [ ] **Step 4: Build check**

Run: `go build ./cmd/fleet/...`
Expected: success

- [ ] **Step 5: Commit**

```bash
git add -f cmd/fleet/service.go cmd/fleet/handler.go
git commit -m "feat: fleet service and handlers"
```

---

### Task 6: Fleet Tests

**Files:**
- Create: `cmd/fleet/fleet_test.go`

- [ ] **Step 1: Create fleet_test.go**

```go
package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDispatch_InvalidMission(t *testing.T) {
	svc := NewFleetService(newMockRepo(), "http://planet:8082")
	_, err := svc.DispatchFleet(context.Background(), 1, DispatchRequest{
		OriginPlanetID: 1, Ships: map[string]int{"cargo": 5},
		TargetGalaxy: 1, TargetSystem: 10, TargetPosition: 4,
		Mission: "invalid", SpeedPct: 100,
	})
	if err == nil || !strings.Contains(err.Error(), "invalid mission") {
		t.Errorf("expected invalid mission error, got %v", err)
	}
}

func TestDispatch_NoShips(t *testing.T) {
	svc := NewFleetService(newMockRepo(), "http://planet:8082")
	_, err := svc.DispatchFleet(context.Background(), 1, DispatchRequest{
		OriginPlanetID: 1, Ships: map[string]int{},
		TargetGalaxy: 1, TargetSystem: 10, TargetPosition: 4,
		Mission: "transport", SpeedPct: 100,
	})
	if err == nil || !strings.Contains(err.Error(), "no ships selected") {
		t.Errorf("expected no ships error, got %v", err)
	}
}

func TestDispatch_InvalidSpeed(t *testing.T) {
	svc := NewFleetService(newMockRepo(), "http://planet:8082")
	_, err := svc.DispatchFleet(context.Background(), 1, DispatchRequest{
		OriginPlanetID: 1, Ships: map[string]int{"cargo": 5},
		TargetGalaxy: 1, TargetSystem: 10, TargetPosition: 4,
		Mission: "transport", SpeedPct: 200,
	})
	if err == nil || !strings.Contains(err.Error(), "speed") {
		t.Errorf("expected speed error, got %v", err)
	}
}

func TestDispatch_PlanetUnreachable(t *testing.T) {
	svc := NewFleetService(newMockRepo(), "http://localhost:19999")
	_, err := svc.DispatchFleet(context.Background(), 1, DispatchRequest{
		OriginPlanetID: 1, Ships: map[string]int{"cargo": 5},
		TargetGalaxy: 1, TargetSystem: 10, TargetPosition: 4,
		Mission: "transport", SpeedPct: 100,
	})
	if err == nil {
		t.Error("expected error for unreachable planet service")
	}
}

func TestMyFleetsHandler(t *testing.T) {
	svc := NewFleetService(newMockRepo(), "http://planet:8082")
	h := NewFleetHandler(svc)

	req := httptest.NewRequest("GET", "/api/fleet/my-fleets", nil)
	req.Header.Set("X-User-ID", "1")
	w := httptest.NewRecorder()
	h.MyFleets(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestDispatchHandler_NoAuth(t *testing.T) {
	svc := NewFleetService(newMockRepo(), "http://planet:8082")
	h := NewFleetHandler(svc)

	req := httptest.NewRequest("POST", "/api/fleet/dispatch", nil)
	w := httptest.NewRecorder()
	h.Dispatch(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./cmd/fleet/... -v -count=1`
Expected: all PASS (the planet unreachable test expects an error, which will happen)

- [ ] **Step 3: Commit**

```bash
git add -f cmd/fleet/fleet_test.go
git commit -m "feat: fleet tests"
```

---

### Task 7: Frontend Fleet Tab

**Files:**
- Modify: `game/src/App.svelte`

- [ ] **Step 1: Read current App.svelte**

Read `game/src/App.svelte` to understand the existing tab pattern.

- [ ] **Step 2: Add fleet state and fetch functions**

In script section, add:

```js
let fleetView = null
let dispatchForm = null
let dispatchShips = {}
let fleetMission = 'transport'
let fleetSpeed = 100
let targetGalaxy = 1
let targetSystem = 1
let targetPosition = 7

async function loadFleets() {
  try {
    const res = await fetch('/api/fleet/my-fleets', {
      headers: { 'Authorization': `Bearer ${token}` }
    })
    if (!res.ok) throw new Error('Failed to load fleets')
    fleetView = await res.json()
  } catch (e) { error = e.message }
}

async function showFleetTab() {
  dispatchForm = null
  await loadFleets()
}

function showDispatchForm() {
  dispatchForm = true
  dispatchShips = {}
  if (planet && planet.ships_available) {
    Object.entries(planet.ships_available).forEach(([type, qty]) => {
      dispatchShips[type] = 0
    })
  }
}

async function submitDispatch() {
  const ships = {}
  Object.entries(dispatchShips).forEach(([type, qty]) => {
    if (qty > 0) ships[type] = qty
  })
  try {
    const res = await fetch('/api/fleet/dispatch', {
      method: 'POST',
      headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({
        origin_planet_id: planet.id,
        ships,
        target_galaxy: targetGalaxy,
        target_system: targetSystem,
        target_position: targetPosition,
        mission: fleetMission,
        speed_pct: fleetSpeed,
      })
    })
    if (!res.ok) {
      const data = await res.json()
      error = data.error || 'Dispatch failed'
      return
    }
    dispatchForm = null
    await loadFleets()
    await loadPlanet()
  } catch (e) { error = e.message }
}
```

Note: `planet.ships_available` doesn't exist yet. We need to either add it to the planet response or fetch it separately. The simplest approach: add `ships_available` to `PlanetResponse` (populated from repo in the handler).

- [ ] **Step 3: Add ships_available to planet response**

In `cmd/planet/types.go`, add to PlanetResponse:

```go
ShipsAvailable map[string]int `json:"ships_available,omitempty"`
```

In `cmd/planet/planet.go`, update the `GetOrCreatePlanet` method to populate ships. Or simpler: in the handler `GetMyPlanet` in `cmd/planet/handler.go`, after building the response, fetch ships:

After line 57 (`resp := toPlanetResponse(...)`), add:

```go
playerShips, _ := h.service.repo.GetPlayerShips(r.Context(), planet.ID)
resp.ShipsAvailable = playerShips
```

- [ ] **Step 4: Add fleet tab button**

```svelte
<button class="fleet-toggle" on:click={showFleetTab}>Fleet</button>
```

Add it near the other tab buttons (Galaxy, Shipyard).

- [ ] **Step 5: Add fleet and dispatch template sections**

After the shipyard section, add:

```svelte
{#if fleetView !== null && !dispatchForm}
  <div class="fleet-section">
    <h3>My Fleets</h3>
    <button class="btn-new-mission" on:click={showDispatchForm}>New Mission</button>
    {#if fleetView.length === 0}
      <p class="empty">No fleets</p>
    {:else}
      {#each fleetView as fleet}
        <div class="fleet-card">
          <div class="fleet-header">
            <span class="fleet-mission">{fleet.mission}</span>
            <span class="fleet-status">{fleet.status}</span>
          </div>
          <div class="fleet-target">[{fleet.target_galaxy}:{fleet.target_system}:{fleet.target_position}]</div>
          <div class="fleet-ships">
            {#each Object.entries(fleet.ships) as [type, qty]}
              <span class="fleet-ship">{type}: {qty}</span>
            {/each}
          </div>
        </div>
      {/each}
    {/if}
  </div>
{:else if dispatchForm}
  <div class="fleet-section">
    <h3>New Mission</h3>
    <div class="dispatch-form">
      <label>Mission
        <select bind:value={fleetMission}>
          <option value="attack">Attack</option>
          <option value="transport">Transport</option>
          <option value="deploy">Deploy</option>
          <option value="espionage">Espionage</option>
          <option value="colonize">Colonize</option>
          <option value="expedition">Expedition</option>
          <option value="recycle">Recycle</option>
        </select>
      </label>
      <div class="coord-inputs">
        <label>G<input type="number" min="1" max="9" bind:value={targetGalaxy} /></label>
        <label>S<input type="number" min="1" max="499" bind:value={targetSystem} /></label>
        <label>P<input type="number" min="1" max="15" bind:value={targetPosition} /></label>
      </div>
      <div class="speed-slider">
        <label>Speed: {fleetSpeed}%</label>
        <input type="range" min="10" max="100" bind:value={fleetSpeed} />
      </div>
      <div class="ship-selectors">
        <h4>Ships</h4>
        {#each Object.entries(dispatchShips) as [type, qty]}
          <div class="ship-selector">
            <span>{type}</span>
            <input type="number" min="0" bind:value={dispatchShips[type]} />
          </div>
        {/each}
      </div>
      <div class="dispatch-actions">
        <button class="btn-cancel" on:click={() => { dispatchForm = null }}>Cancel</button>
        <button class="btn-confirm" on:click={submitDispatch}>Launch</button>
      </div>
    </div>
  </div>
{/if}
```

- [ ] **Step 6: Add CSS**

```css
.fleet-toggle {
  display: block; margin: 1rem auto; padding: 0.5rem 1rem;
  background: #1a2a4a; border: 1px solid #2a4a6a; border-radius: 6px;
  color: #8ab5d4; font-size: 0.85rem; cursor: pointer;
}
.fleet-toggle:hover { background: #2a3a5a; }
.fleet-section { margin-top: 1.5rem; text-align: left; }
.fleet-section h3 { font-size: 0.9rem; color: #8a9ab5; margin-bottom: 0.75rem; text-align: center; }
.fleet-section h4 { font-size: 0.8rem; color: #5a7a9a; margin-bottom: 0.5rem; }
.empty { text-align: center; color: #5a5a6a; font-size: 0.85rem; }
.btn-new-mission {
  display: block; margin: 0 auto 0.75rem; padding: 0.4rem 0.75rem;
  background: #2a4a3a; border: 1px solid #3a6a4a; border-radius: 4px;
  color: #8ad4a8; font-size: 0.8rem; cursor: pointer;
}
.fleet-card {
  padding: 0.5rem 0.75rem; background: #1a2340; border: 1px solid #243050;
  border-radius: 6px; margin-bottom: 0.5rem;
}
.fleet-header { display: flex; justify-content: space-between; margin-bottom: 0.25rem; }
.fleet-mission { font-weight: 600; text-transform: uppercase; font-size: 0.8rem; color: #8ab5d4; }
.fleet-status { font-size: 0.75rem; color: #5a7a9a; }
.fleet-target { font-family: monospace; font-size: 0.75rem; color: #5a7a9a; margin-bottom: 0.25rem; }
.fleet-ships { display: flex; flex-wrap: wrap; gap: 0.35rem; }
.fleet-ship { font-size: 0.7rem; background: #0a0e1a; padding: 0.15rem 0.35rem; border-radius: 3px; color: #8a9ab5; }
.dispatch-form { display: flex; flex-direction: column; gap: 0.75rem; }
.dispatch-form label { font-size: 0.8rem; color: #8a9ab5; }
.dispatch-form select, .dispatch-form input[type="number"] {
  padding: 0.35rem; background: #1a2340; border: 1px solid #243050;
  border-radius: 4px; color: #c8d6e5; font-size: 0.8rem; width: 100%;
}
.coord-inputs { display: flex; gap: 0.5rem; }
.coord-inputs label { flex: 1; }
.coord-inputs input { width: 100%; }
.speed-slider { text-align: center; }
.speed-slider input[type="range"] { width: 100%; }
.ship-selectors { max-height: 200px; overflow-y: auto; }
.ship-selector { display: flex; justify-content: space-between; align-items: center; padding: 0.25rem 0; }
.ship-selector span { font-size: 0.8rem; }
.ship-selector input { width: 70px; }
.dispatch-actions { display: flex; gap: 0.5rem; margin-top: 0.5rem; }
.dispatch-actions .btn-cancel { flex: 1; }
.dispatch-actions .btn-confirm { flex: 1; }
```

- [ ] **Step 7: Commit**

```bash
git add -f cmd/planet/types.go cmd/planet/handler.go game/src/App.svelte
git commit -m "feat: fleet frontend tab with dispatch form"
```

---
