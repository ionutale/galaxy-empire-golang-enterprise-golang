# Fusion Reactor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add Fusion Reactor building — produces energy, consumes gas, gated by Gas Mine Lv5 + Energy Tech Lv3

**Architecture:** New `player_technologies` table in planet schema seeded with Energy Tech Lv3. Tech level passed to production/energy calculations. Gas consumption deducted from net gas output.

**Tech Stack:** Go + chi + pgx, Svelte

---

### Task 1: Schema + Repo + Mock + Seed

**Files:**
- Modify: `cmd/planet/main.go`
- Modify: `cmd/planet/repository.go`
- Modify: `cmd/planet/types.go`

- [ ] **Step 1: Add Technology type**

In `cmd/planet/types.go`, add after `QueueEntry`:

```go
type Technology struct {
	ID   int
	Type string
	Level int
}
```

- [ ] **Step 2: Write failing test for GetTechLevel**

In `cmd/planet/planet_test.go`, add in the mock section:

```go
func (m *mockRepo) GetTechLevel(_ context.Context, userID int, techType string) (int, error) {
	return 3, nil
}
```

And add a test (will fail first since interface has no GetTechLevel):

```go
func TestGetTechLevel_Default(t *testing.T) {
	mock := newMockRepo()
	level, err := mock.GetTechLevel(context.Background(), 1, "energy_tech")
	if err != nil {
		t.Fatal("expected no error, got:", err)
	}
	if level != 3 {
		t.Errorf("expected energy_tech level 3, got %d", level)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./cmd/planet/... -run "TestGetTechLevel" -v -count=1`

- [ ] **Step 4: Add GetTechLevel to Repository interface**

In `cmd/planet/types.go`, find the `Repository` interface and add:

```go
GetTechLevel(ctx context.Context, userID int, techType string) (int, error)
```

- [ ] **Step 5: Run test to verify it passes with mock**

Run: `go test ./cmd/planet/... -run "TestGetTechLevel" -v -count=1`
Expected: PASS

- [ ] **Step 6: Add migration in main.go**

In `cmd/planet/main.go` `runMigrations` function, before the buildings seed block:

```go
if _, err := pool.Exec(ctx, `
	CREATE TABLE IF NOT EXISTS planet.player_technologies (
		id SERIAL PRIMARY KEY,
		user_id INTEGER NOT NULL REFERENCES planet.planets(user_id) ON DELETE CASCADE,
		type VARCHAR(50) NOT NULL,
		level INTEGER NOT NULL DEFAULT 0,
		UNIQUE(user_id, type)
	);
`); err != nil {
	return err
}

if _, err := pool.Exec(ctx, `
	INSERT INTO planet.player_technologies (user_id, type, level)
	SELECT p.user_id, 'energy_tech', 3
	FROM planet.planets p
	WHERE NOT EXISTS (
		SELECT 1 FROM planet.player_technologies t
		WHERE t.user_id = p.user_id AND t.type = 'energy_tech'
	);
`); err != nil {
	return err
}
```

Also add `fusion_reactor` to the buildings seed INSERT:

```go
if _, err := pool.Exec(ctx, `
	INSERT INTO planet.buildings (planet_id, type, level)
	SELECT p.id, btype, 1
	FROM planet.planets p
	CROSS JOIN (VALUES ('robotics_factory'), ('nanite_factory'), ('terraformer'), ('fusion_reactor')) AS t(btype)
	WHERE NOT EXISTS (
		SELECT 1 FROM planet.buildings b
		WHERE b.planet_id = p.id AND b.type = t.btype
	);
`); err != nil {
	return err
}
```

- [ ] **Step 7: Add GetTechLevel in PostgresRepository**

In `cmd/planet/repository.go`, add method:

```go
func (r *PostgresRepository) GetTechLevel(ctx context.Context, userID int, techType string) (int, error) {
	var level int
	err := r.pool.QueryRow(ctx,
		`SELECT level FROM planet.player_technologies WHERE user_id = $1 AND type = $2`,
		userID, techType,
	).Scan(&level)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	return level, nil
}
```

Add pgx import if not already present: check imports.

- [ ] **Step 8: Run all tests**

Run: `go test ./cmd/planet/... -v -count=1`
Expected: all PASS

- [ ] **Step 9: Commit**

```bash
git add cmd/planet/types.go cmd/planet/main.go cmd/planet/repository.go cmd/planet/planet_test.go
git commit -m "feat: add player_technologies schema and GetTechLevel repo"
```

---

### Task 2: Fusion Reactor Gating

**Files:**
- Modify: `cmd/planet/planet.go`
- Modify: `cmd/planet/planet_test.go`

- [ ] **Step 1: Write failing gating tests**

In `cmd/planet/planet_test.go`, add:

```go
func TestFusionReactor_Gating_GasMineTooLow(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	planet, buildings, err := svc.GetOrCreatePlanet(context.Background(), 50)
	if err != nil {
		t.Fatal(err)
	}
	// Set gas_mine to level 4 (below requirement of 5)
	for i, b := range buildings {
		if b.Type == "gas_mine" {
			planet, _, _ := svc.GetOrCreatePlanet(context.Background(), 50)
			mock := svc.repo.(*mockRepo)
			mock.buildings[planet.ID][i].Level = 4
			break
		}
	}
	_, err = svc.StartBuildingUpgrade(context.Background(), planet.ID, "fusion_reactor")
	if err != ErrPrerequisitesNotMet {
		t.Errorf("expected ErrPrerequisitesNotMet, got %v", err)
	}
}

func TestFusionReactor_Gating_EnergyTechTooLow(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	planet, buildings, err := svc.GetOrCreatePlanet(context.Background(), 51)
	if err != nil {
		t.Fatal(err)
	}
	// Ensure gas_mine is high enough, then set energy_tech to 2
	for i, b := range buildings {
		if b.Type == "gas_mine" {
			mock := svc.repo.(*mockRepo)
			mock.buildings[planet.ID][i].Level = 5
			break
		}
	}
	// Need to override mock GetTechLevel for user 51
	mock := svc.repo.(*mockRepo)
	mock.techLevels[51] = map[string]int{"energy_tech": 2}
	
	_, err = svc.StartBuildingUpgrade(context.Background(), planet.ID, "fusion_reactor")
	if err != ErrPrerequisitesNotMet {
		t.Errorf("expected ErrPrerequisitesNotMet, got %v", err)
	}
}

func TestFusionReactor_GatePasses(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	planet, buildings, err := svc.GetOrCreatePlanet(context.Background(), 52)
	if err != nil {
		t.Fatal(err)
	}
	// Ensure gas_mine is level 5
	mock := svc.repo.(*mockRepo)
	for i, b := range buildings {
		if b.Type == "gas_mine" {
			mock.buildings[planet.ID][i].Level = 5
			break
		}
	}
	// energy_tech defaults to 3 from mock

	entry, err := svc.StartBuildingUpgrade(context.Background(), planet.ID, "fusion_reactor")
	if err != nil {
		t.Fatal("expected success, got:", err)
	}
	if entry.BuildingType != "fusion_reactor" {
		t.Errorf("expected fusion_reactor, got %s", entry.BuildingType)
	}
}
```

- [ ] **Step 2: Update mock to support per-user tech levels**

In the `mockRepo` struct, add:

```go
techLevels map[int]map[string]int // userID -> techType -> level
```

In `newMockRepo`, initialize it:

```go
techLevels: make(map[int]map[string]int),
```

Update the `GetTechLevel` mock to return from techLevels or default to 3 for energy_tech:

```go
func (m *mockRepo) GetTechLevel(_ context.Context, userID int, techType string) (int, error) {
	if m.techLevels[userID] != nil {
		if l, ok := m.techLevels[userID][techType]; ok {
			return l, ok
		}
	}
	if techType == "energy_tech" {
		return 3, nil
	}
	return 0, nil
}
```

Wait, but `m.techLevels[userID][techType]` returns int, not (int, bool). Let me fix:

```go
func (m *mockRepo) GetTechLevel(_ context.Context, userID int, techType string) (int, error) {
	if m.techLevels[userID] != nil {
		if l, ok := m.techLevels[userID][techType]; ok {
			return l, nil
		}
	}
	if techType == "energy_tech" {
		return 3, nil
	}
	return 0, nil
}
```

- [ ] **Step 3: Add ErrPrerequisitesNotMet and gating logic**

In `cmd/planet/planet.go`, add sentinel error after existing errors:

```go
var ErrPrerequisitesNotMet = errors.New("prerequisites not met")
```

Add building cost for fusion_reactor (in `buildingCostResources`):

```go
case "fusion_reactor":
	return int(200 * math.Pow(2, next)), int(150 * math.Pow(2, next)), int(50 * math.Pow(2, next))
```

In `StartBuildingUpgrade`, after the `ErrNoFieldsAvailable` check, add:

```go
if buildingType == "fusion_reactor" {
	gasLevel, err := s.repo.GetBuildingLevel(ctx, planetID, "gas_mine")
	if err != nil {
		return QueueEntry{}, err
	}
	if gasLevel < 5 {
		return QueueEntry{}, ErrPrerequisitesNotMet
	}
	techLevel, err := s.repo.GetTechLevel(ctx, planet.UserID, "energy_tech")
	if err != nil {
		return QueueEntry{}, err
	}
	if techLevel < 3 {
		return QueueEntry{}, ErrPrerequisitesNotMet
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/planet/... -run "TestFusionReactor" -v -count=1`
Expected: all PASS

Run: `go test ./cmd/planet/... -count=1`
Expected: all tests PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/planet/planet.go cmd/planet/planet_test.go
git commit -m "feat: fusion reactor gating (Gas Mine Lv5 + Energy Tech Lv3)"
```

---

### Task 3: Fusion Production and Consumption

**Files:**
- Modify: `cmd/planet/planet.go`
- Modify: `cmd/planet/handler.go`
- Modify: `cmd/planet/planet_test.go`
- Modify: `cmd/planet/handler_test.go`
- Modify: `cmd/planet/repository.go`

- [ ] **Step 1: Write failing production tests**

```go
func TestFusionReactor_ProducesEnergy(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	buildings := []Building{
		{Type: "fusion_reactor", Level: 1},
		{Type: "solar_plant", Level: 1},
		{Type: "gas_mine", Level: 1},
	}
	prod := svc.calculateProduction(buildings, 1.0, PlanetTypeTerran, 15, 3)
	energyPerMin := prod.Energy * 60
	if energyPerMin <= 44 {
		t.Errorf("fusion L1 + solar L1 should produce > 44/min, got %.2f/min", energyPerMin)
	}
}

func TestFusionReactor_ConsumesGas(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	buildings := []Building{
		{Type: "gas_mine", Level: 1},
		{Type: "fusion_reactor", Level: 1},
	}
	// Gas L1 = ~11/min. Fusion L1 consumes ~11/min. Net should be ~0 or slightly positive.
	prod := svc.calculateProduction(buildings, 1.0, PlanetTypeTerran, 15, 3)
	if prod.Gas <= 0 {
		t.Errorf("net gas should be positive, got %.4f/s", prod.Gas)
	}
}

func TestFusionReactor_EnergyTechBoostsOutput(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	buildings := []Building{
		{Type: "fusion_reactor", Level: 1},
	}
	prodLow := svc.calculateProduction(buildings, 1.0, PlanetTypeTerran, 15, 3)
	prodHigh := svc.calculateProduction(buildings, 1.0, PlanetTypeTerran, 15, 5)
	if prodHigh.Energy <= prodLow.Energy {
		t.Errorf("higher energy tech should boost fusion output: L3=%.4f L5=%.4f", prodLow.Energy, prodHigh.Energy)
	}
}
```

- [ ] **Step 2: Run test to verify failures**

Run: `go test ./cmd/planet/... -run "TestFusionReactor" -v -count=1`
Expected: FAIL — `calculateProduction` doesn't accept energyTechLevel parameter

- [ ] **Step 3: Update calculateProduction signature and implementation**

Change signature to accept `energyTechLevel int`:

```go
func (s *PlanetService) calculateProduction(buildings []Building, efficiency float64, planetType string, temperature int, energyTechLevel int) Production {
```

Add fusion logic:

```go
func (s *PlanetService) calculateProduction(buildings []Building, efficiency float64, planetType string, temperature int, energyTechLevel int) Production {
	levels := make(map[string]int)
	for _, b := range buildings {
		levels[b.Type] = b.Level
	}

	gasLevel := float64(levels["gas_mine"])
	solarLevel := float64(levels["solar_plant"])

	if planetType == PlanetTypeIce || planetType == PlanetTypeGasGiant {
		gasLevel += 1.5
	}
	if planetType == PlanetTypeDesert || planetType == PlanetTypeVolcanic {
		solarLevel += 1.5
	}

	gasProduction := productionRateForLevel("gas_mine", gasLevel) / 60.0 * efficiency
	fusionConsumption := 0.0
	fusionEnergy := 0.0
	if levels["fusion_reactor"] >= 1 {
		fusionConsumption = fusionGasConsumption(levels["fusion_reactor"]) / 60.0
		fusionEnergy = fusionEnergyOutput(levels["fusion_reactor"], energyTechLevel) / 60.0
	}

	netGas := gasProduction - fusionConsumption
	if netGas < 0 {
		netGas = 0
	}

	return Production{
		Metal:   productionRate("metal_mine", levels["metal_mine"]) / 60.0 * efficiency,
		Crystal: productionRate("crystal_mine", levels["crystal_mine"]) / 60.0 * efficiency,
		Gas:     netGas,
		Energy:  productionRateForLevel("solar_plant", solarLevel)/60.0 + fusionEnergy,
	}
}
```

Add helper functions after `storageCapacity`:

```go
func fusionEnergyOutput(level, energyTechLevel int) float64 {
	if level < 1 {
		return 0
	}
	base := math.Round(50 * float64(level) * math.Pow(1.1, float64(level)) * 100) / 100
	return base * (1 + 0.05*float64(energyTechLevel))
}

func fusionGasConsumption(level int) float64 {
	if level < 1 {
		return 0
	}
	return math.Round(10 * float64(level) * math.Pow(1.1, float64(level)) * 100) / 100
}
```

- [ ] **Step 4: Update calculatePenaltyFactor to include fusion**

Change signature to accept `energyTechLevel int`:

```go
func calculatePenaltyFactor(buildings []Building, energyTechLevel int) (netEnergyPerMin int, efficiency float64) {
```

In the loop, add fusion_reactor as energy producer:

```go
if b.Type == "solar_plant" {
	totalProd += productionRate("solar_plant", b.Level)
} else if b.Type == "fusion_reactor" {
	totalProd += fusionEnergyOutput(b.Level, energyTechLevel)
} else {
	totalCons += energyConsumptionPerMinute(b.Type, b.Level)
}
```

- [ ] **Step 5: Update all callers**

In `planet.go` `GetOrCreatePlanet`:

```go
energyTechLevel, err := s.repo.GetTechLevel(ctx, planet.UserID, "energy_tech")
if err != nil {
	return Planet{}, nil, err
}
netEnergy, efficiency := calculatePenaltyFactor(buildings, energyTechLevel)
prod := s.calculateProduction(buildings, efficiency, planet.Type, planet.Temperature, energyTechLevel)
```

In `handler.go`, `GetMyPlanet`:

```go
energyTechLevel, err := h.service.repo.GetTechLevel(r.Context(), planet.UserID, "energy_tech")
if err != nil {
	slog.Error("get tech level failed", "error", err)
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
	return
}
netEnergy, efficiency := calculatePenaltyFactor(buildings, energyTechLevel)
prod := h.service.calculateProduction(buildings, efficiency, planet.Type, planet.Temperature, energyTechLevel)
```

- [ ] **Step 6: Update existing test callers**

In `planet_test.go`:
- `TestCalculateProduction` — change to `svc.calculateProduction(buildings, 1.0, PlanetTypeTerran, 15, 3)`
- `TestCalculateProduction_WithPenalty` — change to `svc.calculateProduction(buildings, 0.5, PlanetTypeTerran, 15, 3)`
- `TestPenaltyFactor` — change to `calculatePenaltyFactor(buildings, 0)` (energy tech 0)

In `planet_test.go`, add fusion_reactor to seed types in mock Create:

```go
seedTypes := []string{
	"metal_mine", "crystal_mine", "gas_mine", "solar_plant",
	"metal_storage", "crystal_storage", "gas_storage",
	"robotics_factory", "nanite_factory", "terraformer", "fusion_reactor",
}
```

Update building count checks:
- `TestGetOrCreate_FirstCallCreatesWithBuildings` — change `10` to `11`

In `handler_test.go`:
- `TestGetMyPlanet_WithUserID` — change `10` to `11`

- [ ] **Step 7: Run all tests**

Run: `go test ./cmd/planet/... -v -count=1`
Expected: all PASS (including new fusion reactor tests)

- [ ] **Step 8: Run go vet**

Run: `go vet ./cmd/planet/...`
Expected: clean

- [ ] **Step 9: Commit**

```bash
git add cmd/planet/planet.go cmd/planet/handler.go cmd/planet/planet_test.go cmd/planet/handler_test.go
git commit -m "feat: fusion reactor production, consumption, and energy balance"
```

---

### Task 4: Frontend Fusion Reactor Label

**Files:**
- Modify: `game/src/App.svelte`

- [ ] **Step 1: Add fusion_reactor to buildingLabel**

In the `buildingLabel` function in App.svelte, add:

```js
fusion_reactor: 'Fusion Reactor',
```

- [ ] **Step 2: Add fusion_reactor to buildingCost**

In the `buildingCost` function in App.svelte, add:

```js
case 'fusion_reactor': return { metal: Math.floor(200 * Math.pow(2, next)), crystal: Math.floor(150 * Math.pow(2, next)), gas: Math.floor(50 * Math.pow(2, next)) }
```

- [ ] **Step 3: Commit**

```bash
git add game/src/App.svelte
git commit -m "feat: add fusion reactor to frontend building list"
```
