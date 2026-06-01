# Slice 19: Fleet Movement + Speed + Fuel Implementation Plan

> **For agentic workers:** Use subagent-driven-development to implement this plan task-by-task.

**Goal:** Fleets travel with calculated ETA and fuel cost, fuel deducted from origin planet, a travel worker marks fleets as arrived.

**Architecture:** Extend fleet service with travel time/fuel formulas, add `arrives_at` column, start a travel worker goroutine that polls for arrived fleets.

**Tech Stack:** Go, chi, pgx, fleet service port 8083

---

### Task 1: Types — Add ArrivesAt + FuelCost to Fleet

**Files:**
- Modify: `cmd/fleet/types.go`

- [ ] **Step 1: Add fields to Fleet**

```go
type Fleet struct {
	ID              int
	PlayerID        int
	OriginPlanetID  int
	OriginGalaxy    int
	OriginSystem    int
	OriginPosition  int
	TargetGalaxy    int
	TargetSystem    int
	TargetPosition  int
	Mission         string
	Status          string
	SpeedPct        int
	Ships           map[string]int
	ArrivesAt       time.Time
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
	ArrivesAt       *time.Time     `json:"arrives_at,omitempty"`
}
```

- [ ] **Step 2: Add `OriginPlanetID`* import and `time` import to types.go**

Add `import "time"` at top.

- [ ] **Step 3: Build check**

Run: `go build ./cmd/fleet/...`
Expected: success

- [ ] **Step 4: Commit**

```bash
git add -f cmd/fleet/types.go
git commit -m "feat: add ArrivesAt to fleet types"
```

---

### Task 2: Repository — Add arrives_at + MarkFleetArrived + GetArrivedFleets

**Files:**
- Modify: `cmd/fleet/repository.go`

- [ ] **Step 1: Add time import to repository.go**

```go
import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
)
```

- [ ] **Step 2: Update Repository interface**

```go
type Repository interface {
	CreateFleet(ctx context.Context, f Fleet) (Fleet, error)
	ListPlayerFleets(ctx context.Context, playerID int) ([]Fleet, error)
	MarkFleetArrived(ctx context.Context, fleetID int) error
	GetArrivedFleets(ctx context.Context) ([]Fleet, error)
}
```

- [ ] **Step 3: Update CreateFleet to insert arrives_at**

Replace the insert query:

```go
func (r *PostgresRepository) CreateFleet(ctx context.Context, f Fleet) (Fleet, error) {
	shipsJSON, err := json.Marshal(f.Ships)
	if err != nil {
		return Fleet{}, fmt.Errorf("marshal ships: %w", err)
	}

	var id int
	err = r.pool.QueryRow(ctx, `
		INSERT INTO fleet.fleets (player_id, origin_planet_id, target_galaxy, target_system, target_position, mission, status, speed_pct, ships, arrives_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`, f.PlayerID, f.OriginPlanetID, f.TargetGalaxy, f.TargetSystem, f.TargetPosition, f.Mission, f.Status, f.SpeedPct, shipsJSON, f.ArrivesAt).Scan(&id)
	if err != nil {
		return Fleet{}, fmt.Errorf("create fleet: %w", err)
	}
	f.ID = id
	return f, nil
}
```

- [ ] **Step 4: Update ListPlayerFleets to scan arrives_at**

```go
func (r *PostgresRepository) ListPlayerFleets(ctx context.Context, playerID int) ([]Fleet, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, player_id, origin_planet_id, target_galaxy, target_system, target_position, mission, status, speed_pct, ships, arrives_at
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
		var arrivesAt *time.Time
		if err := rows.Scan(&f.ID, &f.PlayerID, &f.OriginPlanetID, &f.TargetGalaxy, &f.TargetSystem, &f.TargetPosition, &f.Mission, &f.Status, &f.SpeedPct, &shipsJSON, &arrivesAt); err != nil {
			return nil, fmt.Errorf("scan fleet: %w", err)
		}
		json.Unmarshal(shipsJSON, &f.Ships)
		if arrivesAt != nil {
			f.ArrivesAt = *arrivesAt
		}
		fleets = append(fleets, f)
	}
	return fleets, rows.Err()
}
```

- [ ] **Step 5: Add MarkFleetArrived**

```go
func (r *PostgresRepository) MarkFleetArrived(ctx context.Context, fleetID int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE fleet.fleets SET status = 'arrived' WHERE id = $1 AND status = 'in_transit'`,
		fleetID,
	)
	if err != nil {
		return fmt.Errorf("mark fleet arrived: %w", err)
	}
	return nil
}
```

- [ ] **Step 6: Add GetArrivedFleets**

```go
func (r *PostgresRepository) GetArrivedFleets(ctx context.Context) ([]Fleet, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, player_id, origin_planet_id, target_galaxy, target_system, target_position, mission, status, speed_pct, ships, arrives_at
		FROM fleet.fleets
		WHERE status = 'in_transit' AND arrives_at <= NOW()
	`)
	if err != nil {
		return nil, fmt.Errorf("get arrived fleets: %w", err)
	}
	defer rows.Close()

	var fleets []Fleet
	for rows.Next() {
		var f Fleet
		var shipsJSON []byte
		var arrivesAt *time.Time
		if err := rows.Scan(&f.ID, &f.PlayerID, &f.OriginPlanetID, &f.TargetGalaxy, &f.TargetSystem, &f.TargetPosition, &f.Mission, &f.Status, &f.SpeedPct, &shipsJSON, &arrivesAt); err != nil {
			return nil, fmt.Errorf("scan fleet: %w", err)
		}
		json.Unmarshal(shipsJSON, &f.Ships)
		if arrivesAt != nil {
			f.ArrivesAt = *arrivesAt
		}
		fleets = append(fleets, f)
	}
	return fleets, rows.Err()
}
```

- [ ] **Step 7: Update mockRepo**

```go
func (m *mockRepo) MarkFleetArrived(ctx context.Context, fleetID int) error {
	for i, f := range m.fleets {
		if f.ID == fleetID {
			m.fleets[i].Status = "arrived"
			return nil
		}
	}
	return nil
}

func (m *mockRepo) GetArrivedFleets(ctx context.Context) ([]Fleet, error) {
	var result []Fleet
	for _, f := range m.fleets {
		if f.Status == "in_transit" && !f.ArrivesAt.IsZero() && time.Now().After(f.ArrivesAt) {
			result = append(result, f)
		}
	}
	return result, nil
}
```

- [ ] **Step 8: Build check**

Run: `go build ./cmd/fleet/...`
Expected: success

- [ ] **Step 9: Commit**

```bash
git add -f cmd/fleet/repository.go
git commit -m "feat: fleet repo with arrives_at + arrival queries"
```

---

### Task 3: Service — Travel Time + Fuel Calculation + Dispatch Overhaul

**Files:**
- Modify: `cmd/fleet/service.go`

- [ ] **Step 1: Add travel/fuel helpers**

```go
func calculateDistance(fleet Fleet) int {
	// Distance formula from OGame: abs(g1-g2)*20000 + abs(s1-s2)*95 + abs(p1-p2)
	return 0
}

func (s *FleetService) calculateTravelTimeAndFuel(ctx context.Context, fleet Fleet) (time.Duration, int, error) {
	// 1. Get ship stats from planet service to know min speed
	// 2. Calculate distance
	// 3. Travel time = distance / minShipSpeed * (100/speed%)
	// 4. Fuel = sum(ships * fuel) * distanceFactor * speedFactor
	// 5. Check bomber alone: if only bomber, fuel > cargo => reject
	return 0, 0, nil
}
```

Actually, the service needs to know ship configs (speed, fuel, cargo). Currently only the planet service has ship configs. The fleet service needs to read ship stats too. Options:

A) Duplicate ship configs in fleet service (simplest)
B) HTTP call to planet service for each ship type

I'll go with A — DRY across services isn't worth the dependency.

- [ ] **Step 2: Add ship configs to fleet service**

Create `cmd/fleet/ships.go` with the same ship configs as planet service:

```go
package main

type ShipConfig struct {
	Type     string
	Speed    int
	Fuel     int
	Cargo    int
}

var Ships = []ShipConfig{
	{Type: "cargo", Speed: 7500, Fuel: 500, Cargo: 25000},
	{Type: "large_cargo", Speed: 7500, Fuel: 1500, Cargo: 100000},
	{Type: "recycler", Speed: 2000, Fuel: 800, Cargo: 20000},
	{Type: "espionage_probe", Speed: 100000000, Fuel: 1, Cargo: 5},
	{Type: "colony_ship", Speed: 2500, Fuel: 2000, Cargo: 7500},
	{Type: "solar_satellite", Speed: 0, Fuel: 0, Cargo: 0},
	{Type: "light_fighter", Speed: 12500, Fuel: 20, Cargo: 50},
	{Type: "heavy_fighter", Speed: 10000, Fuel: 75, Cargo: 100},
	{Type: "cruiser", Speed: 15000, Fuel: 300, Cargo: 800},
	{Type: "battleship", Speed: 10000, Fuel: 1000, Cargo: 1500},
	{Type: "dreadnought", Speed: 5000, Fuel: 2000, Cargo: 2500},
	{Type: "bomber", Speed: 4000, Fuel: 1000, Cargo: 500},
}

func shipConfig(shipType string) (ShipConfig, bool) {
	for _, s := range Ships {
		if s.Type == shipType {
			return s, true
		}
	}
	return ShipConfig{}, false
}

func minShipSpeed(ships map[string]int) (int, bool) {
	min := int(^uint(0) >> 1) // max int
	onlyBomber := true
	for shipType, qty := range ships {
		if qty == 0 {
			continue
		}
		cfg, ok := shipConfig(shipType)
		if !ok {
			continue
		}
		if cfg.Speed > 0 && cfg.Speed < min {
			min = cfg.Speed
		}
		if shipType != "bomber" {
			onlyBomber = false
		}
	}
	if min == int(^uint(0)>>1) {
		return 0, false
	}
	return min, onlyBomber
}
```

- [ ] **Step 3: Update DispatchFleet with travel/fuel logic**

```go
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

	// Validate ships exist
	for shipType := range req.Ships {
		if _, ok := shipConfig(shipType); !ok {
			return Fleet{}, fmt.Errorf("unknown ship: %s", shipType)
		}
	}

	// Check bomber alone
	minSpd, onlyBomber := minShipSpeed(req.Ships)
	if onlyBomber {
		return Fleet{}, fmt.Errorf("bombers cannot fly alone (fuel exceeds cargo)")
	}
	if minSpd == 0 {
		return Fleet{}, fmt.Errorf("no ships with positive speed")
	}

	// Get planet coordinates
	type planetCoords struct {
		Galaxy, System, Position int
	}
	var origin planetCoords
	body, _ := json.Marshal(map[string]int{"planet_id": req.OriginPlanetID})
	resp, err := s.httpClient.Post(s.planetBaseURL+"/internal/planet/coords", "application/json", bytes.NewReader(body))
	if err != nil {
		return Fleet{}, fmt.Errorf("planet service unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		if err := json.NewDecoder(resp.Body).Decode(&origin); err != nil {
			return Fleet{}, fmt.Errorf("parse coords: %w", err)
		}
	}

	// Deduct ships from origin planet
	body, _ = json.Marshal(map[string]any{
		"planet_id": req.OriginPlanetID,
		"ships":     req.Ships,
	})
	resp, err = s.httpClient.Post(s.planetBaseURL+"/internal/ships/deduct", "application/json", bytes.NewReader(body))
	if err != nil {
		return Fleet{}, fmt.Errorf("planet service unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return Fleet{}, fmt.Errorf("planet service: %s", string(respBody))
	}

	// Calculate distance
	dist := distance(origin.Galaxy, origin.System, origin.Position, req.TargetGalaxy, req.TargetSystem, req.TargetPosition)

	// Travel time in seconds = distance / minSpeed * (100/speedPct)
	travelTime := time.Duration(float64(dist)/float64(minSpd)*float64(100)/float64(req.SpeedPct)*3600) * time.Second
	if travelTime < 1*time.Second {
		travelTime = 1 * time.Second
	}

	// Fuel cost = sum(ships * fuel) * distance/35000 * (1 + (100-speed%)/100)
	var totalFuel float64
	for shipType, qty := range req.Ships {
		cfg, ok := shipConfig(shipType)
		if !ok {
			continue
		}
		totalFuel += float64(qty) * float64(cfg.Fuel)
	}
	distanceFactor := float64(dist) / 35000
	if distanceFactor < 1 {
		distanceFactor = 1
	}
	speedFactor := 1.0 + float64(100-req.SpeedPct)/100.0
	fuelCost := int(totalFuel * distanceFactor * speedFactor)

	// Deduct fuel (gas) from planet
	fuelBody, _ := json.Marshal(map[string]any{
		"planet_id": req.OriginPlanetID,
		"resource":  "gas",
		"amount":    fuelCost,
	})
	resp2, err := s.httpClient.Post(s.planetBaseURL+"/internal/resources/deduct", "application/json", bytes.NewReader(fuelBody))
	if err != nil {
		return Fleet{}, fmt.Errorf("fuel deduction failed: %w", err)
	}
	resp2.Body.Close()

	now := time.Now()
	fleet := Fleet{
		PlayerID:       playerID,
		OriginPlanetID: req.OriginPlanetID,
		TargetGalaxy:   req.TargetGalaxy,
		TargetSystem:   req.TargetSystem,
		TargetPosition: req.TargetPosition,
		Mission:        req.Mission,
		Status:         "in_transit",
		SpeedPct:       req.SpeedPct,
		Ships:          req.Ships,
		ArrivesAt:      now.Add(travelTime),
	}

	return s.repo.CreateFleet(ctx, fleet)
}

func distance(g1, s1, p1, g2, s2, p2 int) int {
	return abs(g1-g2)*20000 + abs(s1-s2)*95 + abs(p1-p2)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
```

Actually wait, the fleet service doesn't have origin coordinates yet. The `OriginPlanetID` is known but we need to get the galaxy/system/position from the planet service. We need an internal endpoint for that.

Let me adjust: add an internal endpoint on planet service `/internal/planet/coords` that returns `{galaxy, system, position}` given a planet_id. Or simpler: include origin coordinates in the `DispatchRequest`.

Actually, the simpler approach: include origin coords in DispatchRequest since the frontend already knows the current planet's coordinates. Let me add optional origin galaxy/system/position fields.

Hmm, but that would change the API. Let me add an internal endpoint on planet service instead.

- [ ] **Step 4: Build check**

Run: `go build ./cmd/fleet/...`
Expected: success (may fail on undefined endpoint; that's fine)

- [ ] **Step 5: Commit**

```bash
git add -f cmd/fleet/ships.go cmd/fleet/service.go
git commit -m "feat: travel time + fuel calculation in dispatch"
```

---

### Task 4: Planet — Internal coords + resource deduction endpoints

**Files:**
- Add route in: `cmd/planet/handler.go`
- Add route in: `cmd/planet/main.go`

- [ ] **Step 1: Add InternalGetPlanetCoords handler**

```go
func (h *Handler) InternalGetPlanetCoords(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PlanetID int `json:"planet_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	planet, err := h.service.repo.FindByID(r.Context(), req.PlanetID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "planet not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{
		"galaxy":   planet.Galaxy,
		"system":   planet.System,
		"position": planet.Position,
	})
}
```

- [ ] **Step 2: Add InternalDeductResource handler**

```go
func (h *Handler) InternalDeductResource(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PlanetID int    `json:"planet_id"`
		Resource string `json:"resource"`
		Amount   int    `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	planet, err := h.service.repo.FindByID(r.Context(), req.PlanetID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "planet not found"})
		return
	}

	switch req.Resource {
	case "metal":
		if planet.Metal < req.Amount {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "insufficient metal"})
			return
		}
		planet.Metal -= req.Amount
	case "crystal":
		if planet.Crystal < req.Amount {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "insufficient crystal"})
			return
		}
		planet.Crystal -= req.Amount
	case "gas":
		if planet.Gas < req.Amount {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "insufficient gas"})
			return
		}
		planet.Gas -= req.Amount
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid resource"})
		return
	}

	if err := h.service.repo.UpdateResources(r.Context(), req.PlanetID, planet.Metal, planet.Crystal, planet.Gas, time.Now()); err != nil {
		slog.Error("update resources failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
```

- [ ] **Step 3: Add routes in main.go**

```go
r.Post("/internal/planet/coords", h.InternalGetPlanetCoords)
r.Post("/internal/resources/deduct", h.InternalDeductResource)
```

Add `time` import to handler.go if not present.

- [ ] **Step 4: Build and test**

```bash
go build ./cmd/planet/...
go test ./cmd/planet/... -count=1
```

- [ ] **Step 5: Commit**

```bash
git add -f cmd/planet/handler.go cmd/planet/main.go
git commit -m "feat: internal planet coords + resource deduction endpoints"
```

---

### Task 5: Main — Travel worker goroutine

**Files:**
- Modify: `cmd/fleet/main.go`

- [ ] **Step 1: Add travel worker start in main.go**

```go
go func() {
	slog.Info("travel worker started")
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			fleets, err := repo.GetArrivedFleets(ctx)
			cancel()
			if err != nil {
				slog.Error("travel worker: get arrived fleets", "error", err)
				continue
			}
			for _, f := range fleets {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				if err := repo.MarkFleetArrived(ctx, f.ID); err != nil {
					slog.Error("travel worker: mark arrived", "fleet", f.ID, "error", err)
				}
				slog.Info("fleet arrived", "fleet", f.ID, "mission", f.Mission)
				cancel()
			}
		case <-quit:
			slog.Info("travel worker stopped")
			return
		}
	}
}()
```

- [ ] **Step 2: Build check**

Run: `go build ./cmd/fleet/...`
Expected: success

- [ ] **Step 3: Commit**

```bash
git add -f cmd/fleet/main.go
git commit -m "feat: travel worker goroutine"
```

---

### Task 6: Tests

**Files:**
- Modify: `cmd/fleet/fleet_test.go`

- [ ] **Step 1: Add travel time tests**

```go
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
	svc := NewFleetService(newMockRepo(), "http://localhost:8082")
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
```

Add `"strings"` to imports if not present.

- [ ] **Step 2: Run tests**

Run: `go test ./cmd/fleet/... -v -count=1`
Expected: ALL PASS

- [ ] **Step 3: Commit**

```bash
git add -f cmd/fleet/fleet_test.go
git commit -m "feat: travel time + fuel tests"
```

---

### Task 7: Frontend — ETA countdown + fuel display

**Files:**
- Modify: `game/src/App.svelte`

- [ ] **Step 1: Show ETA countdown on fleet cards**

Replace the fleet arrival section:

```html
{#if fleet.arrives_at}
  <div class="fleet-arrival">
    Arrives: {formatFleetETA(fleet.arrives_at)}
    {#if fleet.status === 'in_transit'}
      <span class="fleet-countdown" data-arrives={fleet.arrives_at}></span>
    {/if}
  </div>
{/if}
```

Add a reactive countdown function:

```js
function formatFleetETA(arrivesAt) {
  const eta = new Date(arrivesAt)
  const now = new Date()
  const diff = eta - now
  if (diff <= 0) return 'Arrived'
  const h = Math.floor(diff / 3600000)
  const m = Math.floor((diff % 3600000) / 60000)
  const s = Math.floor((diff % 60000) / 1000)
  if (h > 0) return `${h}h ${m}m`
  if (m > 0) return `${m}m ${s}s`
  return `${s}s`
}
```

- [ ] **Step 2: Show fuel cost estimate in dispatch form**

After the speed slider, add:

```html
<div class="form-row">
  <label>Fuel Cost</label>
  <span class="fuel-estimate">~{estimateFuelCost()} gas</span>
</div>
```

Add a helper:

```js
function estimateFuelCost() {
  // Simplified: sum of all selected ships * fuel, displayed as estimate
  let total = 0
  Object.entries(dispatchForm.shipQuantities || {}).forEach(([type, qty]) => {
    if (qty > 0 && fleetShips) {
      const ship = fleetShips.ships.find(s => s.type === type)
      if (ship) total += qty * ship.fuel
    }
  })
  return total.toLocaleString()
}
```

- [ ] **Step 3: Commit**

```bash
git add -f game/src/App.svelte
git commit -m "feat: fleet ETA countdown + fuel estimate in UI"
```

---

### Task 8: Migration — Add arrives_at column

**Files:**
- Modify: `cmd/fleet/main.go`

- [ ] **Step 1: Add migration for arrives_at**

In `runMigrations`, add after the CREATE TABLE:

```go
if _, err := pool.Exec(ctx, `
	ALTER TABLE fleet.fleets ADD COLUMN IF NOT EXISTS arrives_at TIMESTAMPTZ;
`); err != nil {
	return err
}
```

- [ ] **Step 2: Build check**

Run: `go build ./cmd/fleet/...`
Expected: success

- [ ] **Step 3: Commit**

```bash
git add -f cmd/fleet/main.go
git commit -m "feat: arrives_at migration"
```
