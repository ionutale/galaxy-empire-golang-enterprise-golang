# VIP & Official Rank Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add VIP (12 levels) and Official Rank (10 ranks) systems that apply % production bonuses to mine output.

**Architecture:** `player_progress` table per user, VIP points earned per completed building upgrade (+10), total resources produced tracked per tick. Bonuses applied multiplicatively in `calculateProduction`.

**Tech Stack:** Go + chi + pgx, Svelte

---

### Task 1: Schema + Migration + Repo + Types + Mock

**Files:**
- Modify: `cmd/planet/types.go`
- Modify: `cmd/planet/main.go`
- Modify: `cmd/planet/repository.go`
- Modify: `cmd/planet/planet_test.go`

- [ ] **Step 1: Write failing test for GetPlayerProgress**

In `cmd/planet/planet_test.go`, add mock methods and test. First, the mock methods:

```go
func (m *mockRepo) GetPlayerProgress(_ context.Context, planetID int) (int, int, error) {
	pp, ok := m.playerProgress[planetID]
	if !ok {
		return 0, 0, nil
	}
	return pp.vipPoints, pp.totalResources, nil
}

func (m *mockRepo) AddVIPPoints(_ context.Context, planetID int, points int) error {
	pp := m.playerProgress[planetID]
	pp.vipPoints += points
	m.playerProgress[planetID] = pp
	return nil
}

func (m *mockRepo) AddResourcesProduced(_ context.Context, planetID int, amount int) error {
	pp := m.playerProgress[planetID]
	pp.totalResources += amount
	m.playerProgress[planetID] = pp
	return nil
}
```

Then update mockRepo struct in `planet_test.go` — add after `techLevels`:
```go
playerProgress map[int]struct{ vipPoints, totalResources int }
```

In `newMockRepo`, initialize:
```go
playerProgress: make(map[int]struct{ vipPoints, totalResources int }),
```

And update the `Create` method in mock to seed progress:
```go
m.playerProgress[p.ID] = struct{ vipPoints, totalResources int }{0, 0}
```

Add test:
```go
func TestPlayerProgress_Default(t *testing.T) {
	mock := newMockRepo()
	vip, total, err := mock.GetPlayerProgress(context.Background(), 1)
	if err != nil {
		t.Fatal("expected no error, got:", err)
	}
	if vip != 0 {
		t.Errorf("expected 0 VIP points, got %d", vip)
	}
	if total != 0 {
		t.Errorf("expected 0 total resources, got %d", total)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/planet/... -run "TestPlayerProgress" -v -count=1`
Expected: FAIL — Repository interface doesn't have the methods

- [ ] **Step 3: Add methods to Repository interface**

In `cmd/planet/types.go`, find the `Repository` interface and add:

```go
GetPlayerProgress(ctx context.Context, planetID int) (vipPoints int, totalResources int, err error)
AddVIPPoints(ctx context.Context, planetID int, points int) error
AddResourcesProduced(ctx context.Context, planetID int, amount int) error
```

- [ ] **Step 4: Run test to verify mock passes**

Run: `go test ./cmd/planet/... -run "TestPlayerProgress" -v -count=1`
Expected: PASS

- [ ] **Step 5: Add migration in main.go**

In `cmd/planet/main.go` `runMigrations`, before the buildings seed block, add:

```go
if _, err := pool.Exec(ctx, `
	CREATE TABLE IF NOT EXISTS planet.player_progress (
		id SERIAL PRIMARY KEY,
		user_id INTEGER NOT NULL REFERENCES planet.planets(user_id) ON DELETE CASCADE UNIQUE,
		vip_points INTEGER NOT NULL DEFAULT 0,
		total_resources_produced BIGINT NOT NULL DEFAULT 0
	);
`); err != nil {
	return err
}

if _, err := pool.Exec(ctx, `
	INSERT INTO planet.player_progress (user_id, vip_points, total_resources_produced)
	SELECT p.user_id, 0, 0
	FROM planet.planets p
	WHERE NOT EXISTS (
		SELECT 1 FROM planet.player_progress pp
		WHERE pp.user_id = p.user_id
	);
`); err != nil {
	return err
}
```

Also in `Create` in `repository.go`, inside the existing transaction, after the planet INSERT and before tx.Commit(), add:

```go
if _, err := tx.Exec(ctx, `
	INSERT INTO planet.player_progress (user_id, vip_points, total_resources_produced)
	VALUES ($1, 0, 0)
	ON CONFLICT (user_id) DO NOTHING
`, userID); err != nil {
	return Planet{}, nil, fmt.Errorf("create progress: %w", err)
}
```

- [ ] **Step 6: Add PostgresRepository methods**

In `cmd/planet/repository.go`, add:

```go
func (r *PostgresRepository) GetPlayerProgress(ctx context.Context, planetID int) (int, int, error) {
	var vipPoints int
	var totalResources int64
	err := r.pool.QueryRow(ctx,
		`SELECT pp.vip_points, pp.total_resources_produced
		 FROM planet.player_progress pp
		 JOIN planet.planets p ON p.user_id = pp.user_id
		 WHERE p.id = $1`,
		planetID,
	).Scan(&vipPoints, &totalResources)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, 0, nil
		}
		return 0, 0, fmt.Errorf("get player progress: %w", err)
	}
	return vipPoints, int(totalResources), nil
}

func (r *PostgresRepository) AddVIPPoints(ctx context.Context, planetID int, points int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE planet.player_progress pp
		 SET vip_points = pp.vip_points + $1
		 FROM planet.planets p
		 WHERE p.id = $2 AND pp.user_id = p.user_id`,
		points, planetID,
	)
	if err != nil {
		return fmt.Errorf("add vip points: %w", err)
	}
	return nil
}

func (r *PostgresRepository) AddResourcesProduced(ctx context.Context, planetID int, amount int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE planet.player_progress pp
		 SET total_resources_produced = pp.total_resources_produced + $1
		 FROM planet.planets p
		 WHERE p.id = $2 AND pp.user_id = p.user_id`,
		amount, planetID,
	)
	if err != nil {
		return fmt.Errorf("add resources produced: %w", err)
	}
	return nil
}
```

- [ ] **Step 7: Run all tests**

Run: `go test ./cmd/planet/... -v -count=1`
Expected: all PASS

- [ ] **Step 8: Commit**

```bash
git add cmd/planet/types.go cmd/planet/main.go cmd/planet/repository.go cmd/planet/planet_test.go
git commit -m "feat: player_progress schema and repo"
```

---

### Task 2: Track VIP Points and Resources + Helpers

**Files:**
- Modify: `cmd/planet/planet.go`
- Modify: `cmd/planet/planet_test.go`

- [ ] **Step 1: Add vipLevelFromPoints and rankFromResources helpers**

In `cmd/planet/planet.go`, add after `planetTypeAndTemp`:

```go
func vipLevelFromPoints(points int) int {
	thresholds := []int{100, 500, 1500, 5000, 15000, 40000, 100000, 250000, 500000, 1000000, 2000000, 5000000}
	level := 0
	for _, t := range thresholds {
		if points >= t {
			level++
		} else {
			break
		}
	}
	return level
}

func rankFromResources(produced int) int {
	thresholds := []int64{1000000, 5000000, 25000000, 100000000, 500000000, 1000000000, 5000000000, 25000000000, 100000000000}
	rank := 0
	for _, t := range thresholds {
		if int64(produced) >= t {
			rank++
		} else {
			break
		}
	}
	return rank
}

func vipProductionBonus(vipLevel int) float64 {
	return float64(vipLevel) * 0.03
}

func rankProductionBonus(rank int) float64 {
	bonuses := []float64{0, 0.02, 0.04, 0.06, 0.08, 0.10, 0.12, 0.15, 0.18, 0.20}
	if rank < 0 || rank >= len(bonuses) {
		return 0
	}
	return bonuses[rank]
}
```

- [ ] **Step 2: Write failing tests for helpers**

```go
func TestVIPLevelFromPoints(t *testing.T) {
	tests := []struct {
		points int
		level  int
	}{
		{0, 0}, {50, 0}, {100, 1}, {499, 1}, {500, 2},
		{1500, 3}, {5000, 4}, {15000, 5}, {40000, 6},
		{100000, 7}, {250000, 8}, {500000, 9},
		{1000000, 10}, {2000000, 11}, {5000000, 12}, {9999999, 12},
	}
	for _, tc := range tests {
		got := vipLevelFromPoints(tc.points)
		if got != tc.level {
			t.Errorf("points=%d expected level %d, got %d", tc.points, tc.level, got)
		}
	}
}

func TestRankFromResources(t *testing.T) {
	tests := []struct {
		res  int
		rank int
	}{
		{0, 0}, {500000, 0}, {1000000, 1}, {4999999, 1},
		{5000000, 2}, {25000000, 3}, {100000000, 4},
		{500000000, 5}, {1000000000, 6}, {5000000000, 7},
		{25000000000, 8}, {100000000000, 9},
	}
	for _, tc := range tests {
		got := rankFromResources(tc.res)
		if got != tc.rank {
			t.Errorf("resources=%d expected rank %d, got %d", tc.res, tc.rank, got)
		}
	}
}

func TestVIPProductionBonus(t *testing.T) {
	if b := vipProductionBonus(0); b != 0 {
		t.Errorf("level 0: expected 0, got %.2f", b)
	}
	if b := vipProductionBonus(10); b != 0.30 {
		t.Errorf("level 10: expected 0.30, got %.2f", b)
	}
	if b := vipProductionBonus(12); b != 0.36 {
		t.Errorf("level 12: expected 0.36, got %.2f", b)
	}
}

func TestRankProductionBonus(t *testing.T) {
	if b := rankProductionBonus(0); b != 0 {
		t.Errorf("rank 0: expected 0, got %.2f", b)
	}
	if b := rankProductionBonus(5); b != 0.10 {
		t.Errorf("rank 5: expected 0.10, got %.2f", b)
	}
	if b := rankProductionBonus(9); b != 0.20 {
		t.Errorf("rank 9: expected 0.20, got %.2f", b)
	}
}
```

- [ ] **Step 3: Run helper tests**

Run: `go test ./cmd/planet/... -run "TestVIPLevel|TestRankFrom|TestVIPProductionBonus|TestRankProductionBonus" -v -count=1`
Expected: all PASS

- [ ] **Step 4: Add tracking for VIP points in processCompletedBuilds**

In `cmd/planet/planet.go`, in `processCompletedBuilds`, after `s.repo.CompleteBuild(ctx, q.ID, q.BuildingType, q.TargetLevel)` call, add:

```go
if err := s.repo.AddVIPPoints(ctx, planetID, 10); err != nil {
    return err
}
```

- [ ] **Step 5: Add tracking for total resources in GetOrCreatePlanet**

In `GetOrCreatePlanet`, update the resource accumulation block (lines 62-65 currently):

Change from:
```go
if elapsed > 0 {
    planet.Metal = minInt(planet.Metal+int(prod.Metal*elapsed), storage.Metal)
    planet.Crystal = minInt(planet.Crystal+int(prod.Crystal*elapsed), storage.Crystal)
    planet.Gas = minInt(planet.Gas+int(prod.Gas*elapsed), storage.Gas)
}
```

To:
```go
if elapsed > 0 {
    addedMetal := minInt(int(prod.Metal*elapsed), storage.Metal-planet.Metal)
    addedCrystal := minInt(int(prod.Crystal*elapsed), storage.Crystal-planet.Crystal)
    addedGas := minInt(int(prod.Gas*elapsed), storage.Gas-planet.Gas)
    if addedMetal < 0 {
        addedMetal = 0
    }
    if addedCrystal < 0 {
        addedCrystal = 0
    }
    if addedGas < 0 {
        addedGas = 0
    }
    planet.Metal += addedMetal
    planet.Crystal += addedCrystal
    planet.Gas += addedGas

    totalMined := addedMetal + addedCrystal + addedGas
    if totalMined > 0 {
        if err := s.repo.AddResourcesProduced(ctx, planet.ID, totalMined); err != nil {
            return Planet{}, nil, err
        }
    }
}
```

- [ ] **Step 6: Write failing tracking tests**

```go
func TestVIPPoints_EarnedOnBuildComplete(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	planet, buildings, err := svc.GetOrCreatePlanet(context.Background(), 60)
	if err != nil {
		t.Fatal(err)
	}
	mock := svc.repo.(*mockRepo)
	initialVIP, _, _ := mock.GetPlayerProgress(context.Background(), planet.ID)

	_, err = svc.StartBuildingUpgrade(context.Background(), planet.ID, buildings[0].Type)
	if err != nil {
		t.Fatal("start upgrade:", err)
	}
	mock.queue[planet.ID][0].CompletesAt = time.Now().Add(-1 * time.Second)
	err = svc.processCompletedBuilds(context.Background(), planet.ID)
	if err != nil {
		t.Fatal("process builds:", err)
	}

	vip, _, _ := mock.GetPlayerProgress(context.Background(), planet.ID)
	if vip != initialVIP+10 {
		t.Errorf("expected %d VIP points, got %d", initialVIP+10, vip)
	}
}

func TestTotalResources_TrackedOnPoll(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	planet, _, err := svc.GetOrCreatePlanet(context.Background(), 61)
	if err != nil {
		t.Fatal(err)
	}
	mock := svc.repo.(*mockRepo)
	_, initialTotal, _ := mock.GetPlayerProgress(context.Background(), planet.ID)

	time.Sleep(2 * time.Second)

	_, _, err = svc.GetOrCreatePlanet(context.Background(), 61)
	if err != nil {
		t.Fatal("second call:", err)
	}

	_, total, _ := mock.GetPlayerProgress(context.Background(), planet.ID)
	if total <= initialTotal {
		t.Errorf("total resources should increase after poll, was %d now %d", initialTotal, total)
	}
}
```

- [ ] **Step 7: Run tests**

Run: `go test ./cmd/planet/... -run "TestVIPPoints|TestTotalResources" -v -count=1`
Expected: PASS (tasks for VIP and resources work through mock)

Run: `go test ./cmd/planet/... -count=1`
Expected: all PASS

- [ ] **Step 8: Commit**

```bash
git add cmd/planet/planet.go cmd/planet/planet_test.go
git commit -m "feat: track VIP points and total resources, add threshold helpers"
```

---

### Task 3: Apply Production Bonuses

**Files:**
- Modify: `cmd/planet/planet.go`
- Modify: `cmd/planet/handler.go`
- Modify: `cmd/planet/planet_test.go`

- [ ] **Step 1: Write failing production bonus tests**

```go
func TestVIPProductionBonus_Applied(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	buildings := []Building{
		{Type: "metal_mine", Level: 1},
	}
	// VIP level 10 = 30% bonus
	prod := svc.calculateProduction(buildings, 1.0, PlanetTypeTerran, 15, 3, 0.30, 0)
	metalPerMin := prod.Metal * 60
	// Metal L1 base = 33/min, with 30% bonus = ~42.9/min
	if metalPerMin < 40 || metalPerMin > 45 {
		t.Errorf("expected ~42.9/min with 30% VIP bonus, got %.2f/min", metalPerMin)
	}
}

func TestRankProductionBonus_Applied(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	buildings := []Building{
		{Type: "metal_mine", Level: 1},
	}
	// Rank 5 = 10% bonus
	prod := svc.calculateProduction(buildings, 1.0, PlanetTypeTerran, 15, 3, 0, 0.10)
	metalPerMin := prod.Metal * 60
	// Metal L1 base = 33/min, with 10% bonus = ~36.3/min
	if metalPerMin < 34 || metalPerMin > 38 {
		t.Errorf("expected ~36.3/min with 10% rank bonus, got %.2f/min", metalPerMin)
	}
}

func TestProductionBonuses_Stack(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	buildings := []Building{
		{Type: "metal_mine", Level: 1},
	}
	// VIP 10 (+30%) + Rank 5 (+10%) = +40%
	prod := svc.calculateProduction(buildings, 1.0, PlanetTypeTerran, 15, 3, 0.30, 0.10)
	metalPerMin := prod.Metal * 60
	// Metal L1 base = 33/min, with 40% bonus = ~46.2/min
	if metalPerMin < 44 || metalPerMin > 48 {
		t.Errorf("expected ~46.2/min with 40% combined bonus, got %.2f/min", metalPerMin)
	}
}

func TestProductionBonuses_WithEfficiency(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	buildings := []Building{
		{Type: "metal_mine", Level: 1},
	}
	prod := svc.calculateProduction(buildings, 0.5, PlanetTypeTerran, 15, 3, 0.30, 0)
	metalPerMin := prod.Metal * 60
	// Metal L1 base = 33/min, *0.5 efficiency = 16.5, *1.3 VIP = 21.45/min
	if metalPerMin < 20 || metalPerMin > 23 {
		t.Errorf("expected ~21.5/min with penalty + VIP, got %.2f/min", metalPerMin)
	}
}
```

- [ ] **Step 2: Run to verify they fail**

Run: `go test ./cmd/planet/... -run "TestVIPProductionBonus_Applied|TestRankProductionBonus_Applied|TestProductionBonuses" -v -count=1`
Expected: FAIL — `calculateProduction` doesn't accept vipBonus/rankBonus params

- [ ] **Step 3: Update calculateProduction signature**

Change `calculateProduction` to accept `vipBonus float64, rankBonus float64`:

```go
func (s *PlanetService) calculateProduction(buildings []Building, efficiency float64, planetType string, temperature int, energyTechLevel int, vipBonus float64, rankBonus float64) Production {
```

Update the function body — add `mineBonus := 1 + vipBonus + rankBonus` and apply to metal, crystal, and gas rates:

```go
func (s *PlanetService) calculateProduction(buildings []Building, efficiency float64, planetType string, temperature int, energyTechLevel int, vipBonus float64, rankBonus float64) Production {
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

	mineBonus := 1 + vipBonus + rankBonus

	gasProduction := productionRateForLevel("gas_mine", gasLevel) / 60.0 * efficiency * mineBonus
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
		Metal:   productionRate("metal_mine", levels["metal_mine"]) / 60.0 * efficiency * mineBonus,
		Crystal: productionRate("crystal_mine", levels["crystal_mine"]) / 60.0 * efficiency * mineBonus,
		Gas:     netGas,
		Energy:  productionRateForLevel("solar_plant", solarLevel)/60.0 + fusionEnergy,
	}
}
```

- [ ] **Step 4: Thread bonuses through GetOrCreatePlanet**

In `GetOrCreatePlanet`, after fetching `energyTechLevel`, add:

```go
vipPoints, totalResources, err := s.repo.GetPlayerProgress(ctx, planet.ID)
if err != nil {
    return Planet{}, nil, err
}
vipLevel := vipLevelFromPoints(vipPoints)
rank := rankFromResources(totalResources)
vipBonus := vipProductionBonus(vipLevel)
rankBonus := rankProductionBonus(rank)
```

Then change the `calculateProduction` call:

```go
prod := s.calculateProduction(buildings, efficiency, planet.Type, planet.Temperature, energyTechLevel, vipBonus, rankBonus)
```

- [ ] **Step 5: Thread bonuses through handler**

In `cmd/planet/handler.go` `GetMyPlanet`, add after fetching `energyTechLevel`:

```go
vipPoints, totalResources, err := h.service.repo.GetPlayerProgress(r.Context(), planet.ID)
if err != nil {
    slog.Error("get player progress failed", "error", err)
    writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
    return
}
vipLevel := vipLevelFromPoints(vipPoints)
rank := rankFromResources(totalResources)
vipBonus := vipProductionBonus(vipLevel)
rankBonus := rankProductionBonus(rank)

netEnergy, efficiency := calculatePenaltyFactor(buildings, energyTechLevel)
prod := h.service.calculateProduction(buildings, efficiency, planet.Type, planet.Temperature, energyTechLevel, vipBonus, rankBonus)
```

- [ ] **Step 6: Update existing test callers**

Find all `calculateProduction` calls in `planet_test.go` and add `, 0, 0` for vipBonus/rankBonus:

```go
// TestCalculateProduction
prod := svc.calculateProduction(buildings, 1.0, PlanetTypeTerran, 15, 3, 0, 0)

// TestCalculateProduction_WithPenalty
prod := svc.calculateProduction(buildings, 0.5, PlanetTypeTerran, 15, 3, 0, 0)

// TestFusionReactor_ProducesEnergy
prod := svc.calculateProduction(buildings, 1.0, PlanetTypeTerran, 15, 3, 0, 0)

// TestFusionReactor_ConsumesGas
prod := svc.calculateProduction(buildings, 1.0, PlanetTypeTerran, 15, 3, 0, 0)

// TestFusionReactor_EnergyTechBoostsOutput
prodLow := svc.calculateProduction(buildings, 1.0, PlanetTypeTerran, 15, 3, 0, 0)
prodHigh := svc.calculateProduction(buildings, 1.0, PlanetTypeTerran, 15, 5, 0, 0)
```

- [ ] **Step 7: Run all tests**

Run: `go test ./cmd/planet/... -v -count=1`
Expected: all PASS (including 5 new bonus tests)

- [ ] **Step 8: Run go vet**

Run: `go vet ./cmd/planet/...`
Expected: clean

- [ ] **Step 9: Commit**

```bash
git add cmd/planet/planet.go cmd/planet/handler.go cmd/planet/planet_test.go
git commit -m "feat: apply VIP and rank production bonuses to mine output"
```

---

### Task 4: Frontend VIP and Rank Display

**Files:**
- Modify: `game/src/App.svelte`

- [ ] **Step 1: Add VIP and rank info to dashboard header**

In `game/src/App.svelte`, in the `<script>` section, add helper:

```js
function rankTitle(rank) {
  const titles = ['Recruit', 'Private', 'Corporal', 'Sergeant', 'Lieutenant',
    'Captain', 'Major', 'Colonel', 'General', 'Admiral']
  return titles[rank] || 'Recruit'
}
```

After the `.planet-meta` div (which has type badge + temperature), add VIP and rank display.

First, read the current template around the planet-meta area to know where to insert.

The planet API doesn't currently return VIP level or rank, so we need to add them to `PlanetResponse` first. Add fields:

In `cmd/planet/types.go`, add to `PlanetResponse`:
```go
VIPLevel int `json:"vip_level"`
Rank     int `json:"rank"`
```

In `cmd/planet/planet.go`, update `toPlanetResponse` to accept and pass the values. But since VIP and rank are computed from player_progress, we need to thread them.

Actually, for simplicity, since `GetOrCreatePlanet` already computes vipBonus/rankBonus, let it also set planet fields or return them differently. The simplest approach: compute VIP level and rank in the handler by calling `GetPlayerProgress` and the helper functions, then pass to `toPlanetResponse`.

In `handler.go`, after computing vipBonus/rankBonus, compute the level/rank:

```go
vipLevel := vipLevelFromPoints(vipPoints)
rank := rankFromResources(totalResources)
```

Then pass to `toPlanetResponse` — update the function signature:

```go
resp := toPlanetResponse(planet, buildings, prod, storage, queue, vipLevel, rank)
```

Update `toPlanetResponse`:

```go
func toPlanetResponse(p Planet, buildings []Building, prod Production, storage Storage, queue []QueueEntry, vipLevel int, rank int) PlanetResponse {
    return PlanetResponse{
        ID: p.ID, UserID: p.UserID, Name: p.Name,
        Metal: p.Metal, Crystal: p.Crystal, Gas: p.Gas,
        Energy: p.Energy,
        Galaxy: p.Galaxy, System: p.System, Position: p.Position,
        MaxFields: p.MaxFields, FieldsUsed: len(buildings),
        Type: p.Type, Temperature: p.Temperature,
        VIPLevel: vipLevel, Rank: rank,
        Buildings: buildings, Production: prod, Storage: storage, Queue: queue,
    }
}
```

Update the other `toPlanetResponse` call in `GetMyPlanet` (handler.go:51) — add `vipLevel, rank`:

```go
resp := toPlanetResponse(planet, buildings, prod, storage, queue, vipLevel, rank)
```

Make sure `GetOrCreatePlanet`'s call to `toPlanetResponse` also passes values. Let me check — actually `GetOrCreatePlanet` doesn't call `toPlanetResponse`. It returns `(Planet, []Building, error)`. Only the handler calls `toPlanetResponse`. So I only need to update handler.go.

Wait, but there might be other places where `toPlanetResponse` is called. Let me check. The handler_test.go calls functions that eventually call the handler, but doesn't call toPlanetResponse directly. The planet_test.go calls service methods directly, not toPlanetResponse.

Let me search for `toPlanetResponse` calls.

There should be 1 call in handler.go `GetMyPlanet`. Let me just update the handler.

Now in the frontend, display them after the planet-meta div:

```html
<p class="player-progress">
  <span class="vip-badge">VIP {planet.vip_level}</span>
  <span class="rank-badge">{rankTitle(planet.rank)}</span>
</p>
```

Add CSS:

```css
.player-progress {
  display: flex; justify-content: center; align-items: center; gap: 0.75rem;
  margin-bottom: 1rem; font-size: 0.85rem;
}
.vip-badge {
  padding: 0.15rem 0.5rem; border-radius: 4px; font-size: 0.8rem;
  background: #2a1a3a; border: 1px solid #4a2a6a; color: #c874d4;
}
.rank-badge {
  padding: 0.15rem 0.5rem; border-radius: 4px; font-size: 0.8rem;
  background: #1a2a3a; border: 1px solid #2a4a6a; color: #74a8d4;
}
```

- [ ] **Step 2: Add VIP and rank to PlanetResponse and toPlanetResponse**

In `cmd/planet/types.go`, add to `PlanetResponse`:
```go
VIPLevel int `json:"vip_level"`
Rank     int `json:"rank"`
```

In `cmd/planet/planet.go`, update `toPlanetResponse` signature and body to include vipLevel and rank.

In `cmd/planet/handler.go`, compute vipLevel and rank and pass to toPlanetResponse.

- [ ] **Step 3: Write template changes in App.svelte**

Add the player-progress div after the planet-meta div and before the fields line.

- [ ] **Step 4: Run tests**

Run: `go test ./cmd/planet/... -count=1`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/planet/types.go cmd/planet/planet.go cmd/planet/handler.go game/src/App.svelte
git commit -m "feat: display VIP level and rank on dashboard"
```
