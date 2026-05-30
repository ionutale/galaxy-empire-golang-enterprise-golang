# Shipyard + All Ships Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add Shipyard building and 12 ship types players can build from their planet.

**Architecture:** All in existing `cmd/planet` service. Ship configs as Go constants in new `ships.go`. Ships stored in `player_ships` table per planet. Shipyard building already exists in seed migration (level 1 seeded for all planets in Slice 08).

**Tech Stack:** Go + chi/pgx, Svelte

---

### Task 1: Ship Config Types + Data

**Files:**
- Create: `cmd/planet/ships.go`

- [ ] **Step 1: Create ships.go**

```go
package main

type ShipConfig struct {
	Type     string
	Name     string
	Metal    int
	Crystal  int
	Gas      int
	Speed    int
	Cargo    int
	Fuel     int
	Strength int
	Shield   int
	Attack   int
}

var Ships = []ShipConfig{
	{Type: "cargo", Name: "Cargo", Metal: 2000, Crystal: 2000, Speed: 7500, Cargo: 25000, Fuel: 500, Strength: 5, Shield: 5, Attack: 3},
	{Type: "large_cargo", Name: "Large Cargo", Metal: 6000, Crystal: 6000, Speed: 7500, Cargo: 100000, Fuel: 1500, Strength: 10, Shield: 10, Attack: 5},
	{Type: "recycler", Name: "Recycler", Metal: 10000, Crystal: 6000, Gas: 2000, Speed: 2000, Cargo: 20000, Fuel: 800, Strength: 15, Shield: 10, Attack: 1},
	{Type: "espionage_probe", Name: "Espionage Probe", Crystal: 1000, Speed: 100000000, Cargo: 5, Fuel: 1, Strength: 1, Shield: 1, Attack: 1},
	{Type: "colony_ship", Name: "Colony Ship", Metal: 10000, Crystal: 20000, Gas: 10000, Speed: 2500, Cargo: 7500, Fuel: 2000, Strength: 30, Shield: 30, Attack: 15},
	{Type: "solar_satellite", Name: "Solar Satellite", Crystal: 2000, Gas: 500, Cargo: 0, Fuel: 0, Strength: 1, Shield: 1, Attack: 1},
	{Type: "light_fighter", Name: "Light Fighter", Metal: 3000, Crystal: 1000, Speed: 12500, Cargo: 50, Fuel: 20, Strength: 5, Shield: 10, Attack: 50},
	{Type: "heavy_fighter", Name: "Heavy Fighter", Metal: 6000, Crystal: 4000, Speed: 10000, Cargo: 100, Fuel: 75, Strength: 15, Shield: 25, Attack: 150},
	{Type: "cruiser", Name: "Cruiser", Metal: 20000, Crystal: 7000, Gas: 2000, Speed: 15000, Cargo: 800, Fuel: 300, Strength: 50, Shield: 50, Attack: 400},
	{Type: "battleship", Name: "Battleship", Metal: 45000, Crystal: 15000, Speed: 10000, Cargo: 1500, Fuel: 1000, Strength: 200, Shield: 200, Attack: 1000},
	{Type: "dreadnought", Name: "Dreadnought", Metal: 90000, Crystal: 45000, Gas: 15000, Speed: 5000, Cargo: 2500, Fuel: 2000, Strength: 700, Shield: 500, Attack: 4000},
	{Type: "bomber", Name: "Bomber", Metal: 50000, Crystal: 25000, Gas: 15000, Speed: 4000, Cargo: 500, Fuel: 1000, Strength: 500, Shield: 500, Attack: 1000},
}

func shipConfig(shipType string) (ShipConfig, bool) {
	for _, s := range Ships {
		if s.Type == shipType {
			return s, true
		}
	}
	return ShipConfig{}, false
}
```

- [ ] **Step 2: Build check**

Run: `go build ./cmd/planet/...`
Expected: success

- [ ] **Step 3: Commit**

```bash
git add -f cmd/planet/ships.go
git commit -m "feat: ship configs"
```

---

### Task 2: Schema + Migration

**Files:**
- Modify: `cmd/planet/main.go`

- [ ] **Step 1: Add player_ships table migration**

In `runMigrations` in `cmd/planet/main.go`, after the galaxy position migration, add:

```go
if _, err := pool.Exec(ctx, `
    CREATE TABLE IF NOT EXISTS planet.player_ships (
        id SERIAL PRIMARY KEY,
        planet_id INT NOT NULL REFERENCES planet.planets(id) ON DELETE CASCADE,
        ship_type VARCHAR(50) NOT NULL,
        quantity INT NOT NULL DEFAULT 0,
        UNIQUE(planet_id, ship_type)
    );
`); err != nil {
    return err
}

if _, err := pool.Exec(ctx, `
    INSERT INTO planet.player_ships (planet_id, ship_type, quantity)
    SELECT p.id, s.ship_type, 0
    FROM planet.planets p
    CROSS JOIN (VALUES ('cargo'), ('large_cargo'), ('recycler'), ('espionage_probe'), ('colony_ship'), ('solar_satellite'), ('light_fighter'), ('heavy_fighter'), ('cruiser'), ('battleship'), ('dreadnought'), ('bomber')) AS s(ship_type)
    WHERE NOT EXISTS (
        SELECT 1 FROM planet.player_ships ps
        WHERE ps.planet_id = p.id AND ps.ship_type = s.ship_type
    );
`); err != nil {
    return err
}
```

- [ ] **Step 2: Add shipyard to building seed**

Find the existing building seed INSERT (seeds robotics_factory, nanite_factory, terraformer, fusion_reactor at level 1). Add 'shipyard' to the VALUES list.

Change:
```sql
CROSS JOIN (VALUES ('robotics_factory'), ('nanite_factory'), ('terraformer'), ('fusion_reactor')) AS t(btype)
```
To:
```sql
CROSS JOIN (VALUES ('robotics_factory'), ('nanite_factory'), ('terraformer'), ('fusion_reactor'), ('shipyard')) AS t(btype)
```

- [ ] **Step 3: Build check**

Run: `go build ./cmd/planet/...`
Expected: success

- [ ] **Step 4: Commit**

```bash
git add -f cmd/planet/main.go
git commit -m "feat: player_ships table and shipyard building seed"
```

---

### Task 3: Types

**Files:**
- Modify: `cmd/planet/types.go`

- [ ] **Step 1: Add ShipResponse and BuildRequest types**

Add after Position struct:

```go
type ShipResponse struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Metal    int    `json:"metal"`
	Crystal  int    `json:"crystal"`
	Gas      int    `json:"gas"`
	Speed    int    `json:"speed"`
	Cargo    int    `json:"cargo"`
	Fuel     int    `json:"fuel"`
	Strength int    `json:"strength"`
	Shield   int    `json:"shield"`
	Attack   int    `json:"attack"`
	Quantity int    `json:"quantity"`
}

type BuildRequest struct {
	ShipType string `json:"ship_type"`
	Quantity int    `json:"quantity"`
}
```

- [ ] **Step 2: Build check**

Run: `go build ./cmd/planet/...`
Expected: success

- [ ] **Step 3: Commit**

```bash
git add -f cmd/planet/types.go
git commit -m "feat: ship types"
```

---

### Task 4: Repository Methods

**Files:**
- Modify: `cmd/planet/repository.go`

- [ ] **Step 1: Add to Repository interface**

After `GetSystemPositions`:

```go
	GetPlayerShips(ctx context.Context, planetID int) (map[string]int, error)
	AddPlayerShips(ctx context.Context, planetID, planetUserID int, shipType string, quantity int) error
```

- [ ] **Step 2: Add to mockRepo**

```go
func (m *mockRepo) GetPlayerShips(ctx context.Context, planetID int) (map[string]int, error) {
	return nil, nil
}

func (m *mockRepo) AddPlayerShips(ctx context.Context, planetID, planetUserID int, shipType string, quantity int) error {
	return nil
}
```

- [ ] **Step 3: Add PostgresRepo methods**

```go
func (r *PostgresRepository) GetPlayerShips(ctx context.Context, planetID int) (map[string]int, error) {
	rows, err := r.pool.Query(ctx, `SELECT ship_type, quantity FROM planet.player_ships WHERE planet_id = $1`, planetID)
	if err != nil {
		return nil, fmt.Errorf("get player ships: %w", err)
	}
	defer rows.Close()

	ships := make(map[string]int)
	for rows.Next() {
		var shipType string
		var quantity int
		if err := rows.Scan(&shipType, &quantity); err != nil {
			return nil, fmt.Errorf("scan ship: %w", err)
		}
		ships[shipType] = quantity
	}
	return ships, rows.Err()
}

func (r *PostgresRepository) AddPlayerShips(ctx context.Context, planetID, planetUserID int, shipType string, quantity int) error {
	if _, err := r.pool.Exec(ctx, `
		INSERT INTO planet.player_ships (planet_id, ship_type, quantity)
		VALUES ($1, $2, $3)
		ON CONFLICT (planet_id, ship_type) DO UPDATE SET quantity = planet.player_ships.quantity + $3
	`, planetID, shipType, quantity); err != nil {
		return fmt.Errorf("add player ships: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: Build check**

Run: `go build ./cmd/planet/...`
Expected: success

- [ ] **Step 5: Commit**

```bash
git add -f cmd/planet/repository.go
git commit -m "feat: ship repository methods"
```

---

### Task 5: Ship Build Logic

**Files:**
- Modify: `cmd/planet/planet.go`

- [ ] **Step 1: Add BuildShips method to PlanetService**

After `QueueDeconstruction` method, add:

```go
var ErrInsufficientResources = errors.New("insufficient resources")
var ErrInvalidShip = errors.New("invalid ship type")
var ErrNoShipyard = errors.New("no shipyard")

func (s *PlanetService) BuildShips(ctx context.Context, planetID int, shipType string, quantity int) (int, error) {
	cfg, ok := shipConfig(shipType)
	if !ok {
		return 0, ErrInvalidShip
	}

	if quantity < 1 {
		return 0, fmt.Errorf("quantity must be positive")
	}

	shipyardLevel, err := s.repo.GetBuildingLevel(ctx, planetID, "shipyard")
	if err != nil {
		return 0, err
	}
	if shipyardLevel < 1 {
		return 0, ErrNoShipyard
	}

	planet, err := s.repo.FindByID(ctx, planetID)
	if err != nil {
		return 0, err
	}

	totalMetal := cfg.Metal * quantity
	totalCrystal := cfg.Crystal * quantity
	totalGas := cfg.Gas * quantity

	if planet.Metal < totalMetal || planet.Crystal < totalCrystal || planet.Gas < totalGas {
		return 0, ErrInsufficientResources
	}

	if err := s.repo.UpdateResources(ctx, planetID, planet.Metal-totalMetal, planet.Crystal-totalCrystal, planet.Gas-totalGas, time.Now()); err != nil {
		return 0, err
	}

	if err := s.repo.AddPlayerShips(ctx, planetID, planet.UserID, shipType, quantity); err != nil {
		return 0, err
	}

	return quantity, nil
}
```

- [ ] **Step 2: Build check**

Run: `go build ./cmd/planet/...`
Expected: success

- [ ] **Step 3: Commit**

```bash
git add -f cmd/planet/planet.go
git commit -m "feat: ship build logic"
```

---

### Task 6: Handler Endpoints

**Files:**
- Modify: `cmd/planet/handler.go`
- Modify: `cmd/planet/main.go`

- [ ] **Step 1: Add ship handlers**

After `GetPositions` handler:

```go
func (h *Handler) ListShips(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)
	planet, _, err := h.service.GetOrCreatePlanet(r.Context(), userID)
	if err != nil {
		slog.Error("get planet failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	shipyardLevel, _ := h.service.repo.GetBuildingLevel(r.Context(), planet.ID, "shipyard")
	playerShips, _ := h.service.repo.GetPlayerShips(r.Context(), planet.ID)

	ships := make([]ShipResponse, len(Ships))
	for i, cfg := range Ships {
		ships[i] = ShipResponse{
			Type: cfg.Type, Name: cfg.Name,
			Metal: cfg.Metal, Crystal: cfg.Crystal, Gas: cfg.Gas,
			Speed: cfg.Speed, Cargo: cfg.Cargo, Fuel: cfg.Fuel,
			Strength: cfg.Strength, Shield: cfg.Shield, Attack: cfg.Attack,
			Quantity: playerShips[cfg.Type],
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"shipyard_level": shipyardLevel,
		"ships":          ships,
	})
}

func (h *Handler) BuildShips(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)
	planet, _, err := h.service.GetOrCreatePlanet(r.Context(), userID)
	if err != nil {
		slog.Error("get planet failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	var req BuildRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	quantity, err := h.service.BuildShips(r.Context(), planet.ID, req.ShipType, req.Quantity)
	if err != nil {
		slog.Error("build ships failed", "ship_type", req.ShipType, "error", err)
		code := http.StatusBadRequest
		msg := err.Error()
		switch {
		case errors.Is(err, ErrInvalidShip):
			msg = "invalid ship type"
		case errors.Is(err, ErrNoShipyard):
			msg = "no shipyard"
		case errors.Is(err, ErrInsufficientResources):
			msg = "insufficient resources"
		}
		if code == http.StatusBadRequest {
			writeJSON(w, code, map[string]string{"error": msg})
		} else {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"type":     req.ShipType,
		"quantity": quantity,
	})
}
```

- [ ] **Step 2: Add helper to extract user ID**

Check if there's already a helper for this. If not, note that the handlers currently parse user ID from header inline. For the ship handlers, follow the same pattern as `GetMyPlanet` — parse `X-User-ID` header inline, rather than creating a helper.

Actually, let me re-check. The existing handlers parse userID from header directly. Use that same pattern in the ship handlers instead of calling `userIDFromContext`.

Replace the `userID := userIDFromContext(r)` lines in the handlers above with:

```go
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
```

- [ ] **Step 3: Add routes in main.go**

After the galaxy routes:

```go
r.Get("/api/shipyard", h.ListShips)
r.Post("/api/shipyard/build", h.BuildShips)
```

- [ ] **Step 4: Build check**

Run: `go build ./cmd/planet/...`
Expected: success

- [ ] **Step 5: Commit**

```bash
git add -f cmd/planet/handler.go cmd/planet/main.go
git commit -m "feat: shipyard handlers and routes"
```

---

### Task 7: Tests

**Files:**
- Modify: `cmd/planet/planet_test.go`

- [ ] **Step 1: Add ship config test**

```go
func TestShipConfigs(t *testing.T) {
	if len(Ships) != 12 {
		t.Errorf("expected 12 ships, got %d", len(Ships))
	}
	for _, s := range Ships {
		if s.Type == "" || s.Name == "" {
			t.Errorf("ship missing type or name: %+v", s)
		}
		if s.Cargo == 0 && s.Type != "solar_satellite" && s.Type != "espionage_probe" {
			t.Errorf("ship %s has zero cargo", s.Type)
		}
	}
}

func TestBuildShips_Valid(t *testing.T) {
	mock := newMockRepo()
	svc := NewPlanetService(mock)
	planet, _, err := svc.GetOrCreatePlanet(context.Background(), 80)
	if err != nil {
		t.Fatal(err)
	}

	quantity, err := svc.BuildShips(context.Background(), planet.ID, "cargo", 5)
	if err != nil {
		t.Fatal(err)
	}
	if quantity != 5 {
		t.Errorf("expected 5, got %d", quantity)
	}

	planet, _ = mock.FindByID(context.Background(), planet.ID)
	cargo := Ships[0]
	expectedMetal := 500 - cargo.Metal*5
	if planet.Metal != expectedMetal {
		t.Errorf("expected metal %d, got %d", expectedMetal, planet.Metal)
	}
}

func TestBuildShips_InsufficientResources(t *testing.T) {
	mock := newMockRepo()
	svc := NewPlanetService(mock)
	planet, _, err := svc.GetOrCreatePlanet(context.Background(), 81)
	if err != nil {
		t.Fatal(err)
	}

	planet.Metal = 0
	mock.planets[planet.ID] = planet

	_, err = svc.BuildShips(context.Background(), planet.ID, "dreadnought", 1)
	if err != ErrInsufficientResources {
		t.Errorf("expected ErrInsufficientResources, got %v", err)
	}
}

func TestBuildShips_InvalidShip(t *testing.T) {
	mock := newMockRepo()
	svc := NewPlanetService(mock)
	planet, _, err := svc.GetOrCreatePlanet(context.Background(), 82)
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.BuildShips(context.Background(), planet.ID, "death_star", 1)
	if err != ErrInvalidShip {
		t.Errorf("expected ErrInvalidShip, got %v", err)
	}
}

func TestBuildShips_NoShipyard(t *testing.T) {
	mock := newMockRepo()
	svc := NewPlanetService(mock)
	planet, _, err := svc.GetOrCreatePlanet(context.Background(), 83)
	if err != nil {
		t.Fatal(err)
	}

	mock.buildings[planet.ID] = nil

	_, err = svc.BuildShips(context.Background(), planet.ID, "cargo", 1)
	if err != ErrNoShipyard {
		t.Errorf("expected ErrNoShipyard, got %v", err)
	}
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./cmd/planet/... -run "TestShipConfigs|TestBuildShips" -v -count=1`
Expected: all PASS

- [ ] **Step 3: Run full suite**

Run: `go test ./cmd/planet/... -count=1`
Expected: all PASS

- [ ] **Step 4: Commit**

```bash
git add -f cmd/planet/planet_test.go
git commit -m "feat: shipyard tests"
```

---

### Task 8: Frontend Shipyard Tab

**Files:**
- Modify: `game/src/App.svelte`

- [ ] **Step 1: Add shipyard state and fetch functions**

In script section, after galaxy functions, add:

```js
let shipyardData = null
let buildQuantities = {}

async function loadShipyard() {
  try {
    const res = await fetch('/api/shipyard', {
      headers: { 'Authorization': `Bearer ${token}` }
    })
    if (!res.ok) throw new Error('Failed to load shipyard')
    shipyardData = await res.json()
    shipyardData.ships.forEach(s => { buildQuantities[s.type] = buildQuantities[s.type] || 1 })
  } catch (e) { error = e.message }
}

async function buildShips(shipType) {
  const qty = buildQuantities[shipType] || 1
  try {
    const res = await fetch('/api/shipyard/build', {
      method: 'POST',
      headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({ ship_type: shipType, quantity: qty })
    })
    if (!res.ok) {
      const data = await res.json()
      error = data.error || 'Build failed'
      return
    }
    await loadShipyard()
    await loadPlanet()
  } catch (e) { error = e.message }
}
```

- [ ] **Step 2: Add Shipyard button + tab UI**

Find where Galaxy button is, add Shipyard button next to it:

```svelte
<button class="galaxy-toggle" on:click={showGalaxyTab}>Galaxy</button>
<button class="shipyard-toggle" on:click={loadShipyard}>Shipyard</button>
```

After the galaxy section closing `{/if}`, before `{:else if error}`, add:

```svelte
{#if shipyardData}
  <div class="shipyard-section">
    <h3>Shipyard {shipyardData.shipyard_level > 0 ? `Lv.${shipyardData.shipyard_level}` : '(not built)'}</h3>
    <div class="ship-grid">
      {#each shipyardData.ships as ship}
        <div class="ship-card">
          <div class="ship-header">
            <span class="ship-name">{ship.name}</span>
            <span class="ship-qty">Owned: {ship.quantity}</span>
          </div>
          <div class="ship-stats">
            <span class="stat">⚡ {ship.speed}</span>
            <span class="stat">📦 {ship.cargo}</span>
            <span class="stat">⛽ {ship.fuel}</span>
          </div>
          <div class="ship-cost">
            <span class="cost metal">M {ship.metal}</span>
            <span class="cost crystal">C {ship.crystal}</span>
            {#if ship.gas > 0}
              <span class="cost gas">G {ship.gas}</span>
            {/if}
          </div>
          <div class="ship-build">
            <input type="number" min="1" bind:value={buildQuantities[ship.type]} class="qty-input" />
            <button class="btn-build" disabled={!canAffordShip(ship)} on:click={() => buildShips(ship.type)}>Build</button>
          </div>
        </div>
      {/each}
    </div>
  </div>
{/if}
```

- [ ] **Step 3: Add canAffordShip helper**

In script section after `canAfford`:

```js
function canAffordShip(ship) {
  if (!planet) return false
  const qty = buildQuantities[ship.type] || 1
  return planet.metal >= ship.metal * qty && planet.crystal >= ship.crystal * qty && planet.gas >= ship.gas * qty
}
```

- [ ] **Step 4: Add CSS**

```css
.shipyard-toggle {
  display: block; margin: 1rem auto; padding: 0.5rem 1rem;
  background: #2a3a1a; border: 1px solid #4a6a2a; border-radius: 6px;
  color: #8ad474; font-size: 0.85rem; cursor: pointer;
}
.shipyard-toggle:hover { background: #3a4a2a; }
.shipyard-section { margin-top: 1.5rem; }
.shipyard-section h3 { font-size: 0.9rem; color: #8a9ab5; margin-bottom: 0.75rem; text-align: center; }
.ship-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 0.5rem; }
.ship-card {
  padding: 0.5rem; background: #1a2340; border: 1px solid #243050;
  border-radius: 6px; font-size: 0.8rem;
}
.ship-header { display: flex; justify-content: space-between; margin-bottom: 0.3rem; }
.ship-name { font-weight: 600; color: #c8d6e5; }
.ship-qty { font-size: 0.7rem; color: #5a7a9a; }
.ship-stats { display: flex; gap: 0.5rem; font-size: 0.7rem; color: #8ab5d4; margin-bottom: 0.3rem; }
.ship-cost { display: flex; gap: 0.5rem; font-size: 0.7rem; margin-bottom: 0.3rem; }
.cost.metal { color: #d4a574; }
.cost.crystal { color: #74a8d4; }
.cost.gas { color: #74d4a8; }
.ship-build { display: flex; gap: 0.25rem; }
.qty-input {
  width: 50px; padding: 0.2rem; background: #0a0e1a; border: 1px solid #243050;
  border-radius: 3px; color: #c8d6e5; font-size: 0.75rem; text-align: center;
}
.btn-build {
  flex: 1; padding: 0.25rem; background: #2a5a3a; border: none;
  border-radius: 3px; color: #a8e8a8; font-size: 0.7rem; cursor: pointer;
}
.btn-build:disabled { opacity: 0.4; cursor: not-allowed; }
.btn-build:hover:not(:disabled) { background: #3a6a4a; }
```

- [ ] **Step 5: Commit**

```bash
git add game/src/App.svelte
git commit -m "feat: shipyard frontend tab"
```

---
