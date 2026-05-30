# Slice 11: Planet Types & Temperature Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add planet type and temperature with production effects.

**Architecture:** DB migration adds `type` and `temperature` columns. Planet creation assigns type/temp based on position. `calculateProduction` accepts temp for effective level bonuses.

**Tech Stack:** Go, chi, pgx, Svelte

---

### Task 1: Planet Type + Temperature Schema and Creation

**Files:**
- Modify: `cmd/planet/types.go`
- Modify: `cmd/planet/main.go`
- Modify: `cmd/planet/repository.go`
- Modify: `cmd/planet/planet.go`
- Test: `cmd/planet/planet_test.go`

- [ ] **Step 1: Add Type and Temperature to Planet and PlanetResponse**

In `cmd/planet/types.go`, add fields:

```go
type Planet struct {
	// ... existing fields
	MaxFields          int
	Type               string
	Temperature        int
	ResourcesUpdatedAt time.Time
}

type PlanetResponse struct {
	// ... existing fields
	MaxFields   int          `json:"max_fields"`
	FieldsUsed  int          `json:"fields_used"`
	Type        string       `json:"type"`
	Temperature int          `json:"temperature"`
	Buildings   []Building   `json:"buildings"`
	// ...
}
```

- [ ] **Step 2: Write failing test for planet type assignment**

In `cmd/planet/planet_test.go`, add:

```go
func TestPlanetTypeAndTemp_HomePlanet(t *testing.T) {
	typ, temp := planetTypeAndTemp(7)
	if typ != "terran" {
		t.Errorf("expected terran, got %s", typ)
	}
	if temp < 0 || temp > 20 {
		t.Errorf("expected temp 0-20 for home, got %d", temp)
	}
}

func TestPlanetTypeAndTemp_Position1(t *testing.T) {
	typ, _ := planetTypeAndTemp(1)
	if typ != "desert" && typ != "volcanic" {
		t.Errorf("expected desert or volcanic for pos 1, got %s", typ)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./cmd/planet/... -run "TestPlanetTypeAndTemp" -v -count=1`
Expected: FAIL — `planetTypeAndTemp` not defined

- [ ] **Step 4: Add planetTypeAndTemp function**

In `cmd/planet/planet.go`, add constants and function:

```go
const (
	PlanetTypeTerran   = "terran"
	PlanetTypeDesert   = "desert"
	PlanetTypeIce      = "ice"
	PlanetTypeVolcanic = "volcanic"
	PlanetTypeGasGiant = "gas_giant"
)

func planetTypeAndTemp(position int) (typ string, temperature int) {
    switch {
    case position >= 1 && position <= 3:
        if rand.Intn(100) < 80 {
            typ = PlanetTypeDesert
        } else {
            typ = PlanetTypeVolcanic
        }
        temperature = 60 + rand.Intn(41) // 60-100
    case position >= 4 && position <= 6:
        typ = PlanetTypeTerran
        temperature = 10 + rand.Intn(31) // 10-40
    case position == 7:
        typ = PlanetTypeTerran
        temperature = rand.Intn(21) // 0-20
    case position >= 8 && position <= 9:
        if rand.Intn(100) < 60 {
            typ = PlanetTypeTerran
        } else {
            typ = PlanetTypeIce
        }
        temperature = -10 + rand.Intn(41) // -10-30
    case position >= 10 && position <= 12:
        typ = PlanetTypeIce
        temperature = -50 + rand.Intn(51) // -50-0
    case position >= 13 && position <= 15:
        typ = PlanetTypeGasGiant
        temperature = -80 + rand.Intn(51) // -80--30
    default:
        typ = PlanetTypeTerran
        temperature = 20
    }
    return
}
```

Add import for `"math/rand"`.

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./cmd/planet/... -run "TestPlanetTypeAndTemp" -v -count=1`
Expected: PASS

- [ ] **Step 6: Add migration for type and temperature**

In `cmd/planet/main.go`, add after existing migrations:

```go
if _, err := pool.Exec(ctx, `
    ALTER TABLE planet.planets
    ADD COLUMN IF NOT EXISTS type VARCHAR(20) NOT NULL DEFAULT 'terran';
`); err != nil {
    return err
}

if _, err := pool.Exec(ctx, `
    ALTER TABLE planet.planets
    ADD COLUMN IF NOT EXISTS temperature INTEGER NOT NULL DEFAULT 20;
`); err != nil {
    return err
}
```

- [ ] **Step 7: Update repository queries to include type and temperature**

In `cmd/planet/repository.go`, update all SELECT queries for planets:

`FindByUserID`:
```go
err := r.pool.QueryRow(ctx,
    `SELECT id, user_id, name, metal, crystal, gas, energy,
            galaxy, system, position, max_fields, type, temperature, resources_updated_at
     FROM planet.planets WHERE user_id = $1`,
    userID,
).Scan(&p.ID, &p.UserID, &p.Name, &p.Metal, &p.Crystal, &p.Gas, &p.Energy,
    &p.Galaxy, &p.System, &p.Position, &p.MaxFields, &p.Type, &p.Temperature, &p.ResourcesUpdatedAt)
```

`FindByID`:
```go
err := r.pool.QueryRow(ctx,
    `SELECT id, user_id, name, metal, crystal, gas, energy,
            galaxy, system, position, max_fields, type, temperature, resources_updated_at
     FROM planet.planets WHERE id = $1`,
    planetID,
).Scan(&p.ID, &p.UserID, &p.Name, &p.Metal, &p.Crystal, &p.Gas, &p.Energy,
    &p.Galaxy, &p.System, &p.Position, &p.MaxFields, &p.Type, &p.Temperature, &p.ResourcesUpdatedAt)
```

`Create`:
```go
typ, temp := planetTypeAndTemp(position)

err = tx.QueryRow(ctx,
    `INSERT INTO planet.planets (user_id, max_fields, type, temperature, resources_updated_at)
     VALUES ($1, $2, $3, $4, NOW())
     RETURNING id, user_id, name, metal, crystal, gas, energy,
               galaxy, system, position, max_fields, type, temperature, resources_updated_at`,
    userID, baseMaxFields, typ, temp,
).Scan(&p.ID, &p.UserID, &p.Name, &p.Metal, &p.Crystal, &p.Gas, &p.Energy,
    &p.Galaxy, &p.System, &p.Position, &p.MaxFields, &p.Type, &p.Temperature, &p.ResourcesUpdatedAt)
```

- [ ] **Step 8: Update mock Create to set type and temperature**

In `cmd/planet/planet_test.go`, update the `Create` method:

```go
p := Planet{
    ID: m.nextPID, UserID: userID, Name: "Homeworld",
    Metal: 500, Crystal: 300, Gas: 200, Energy: 50,
    Galaxy: 1, System: 1, Position: 7,
    MaxFields: 40,
    Type: "terran", Temperature: 15,
    ResourcesUpdatedAt: now,
}
```

- [ ] **Step 9: Update toPlanetResponse**

In `cmd/planet/planet.go`, update `toPlanetResponse`:

```go
func toPlanetResponse(p Planet, buildings []Building, prod Production, storage Storage, queue []QueueEntry) PlanetResponse {
	return PlanetResponse{
		ID: p.ID, UserID: p.UserID, Name: p.Name,
		Metal: p.Metal, Crystal: p.Crystal, Gas: p.Gas,
		Energy: p.Energy,
		Galaxy: p.Galaxy, System: p.System, Position: p.Position,
		MaxFields: p.MaxFields, FieldsUsed: len(buildings),
		Type: p.Type, Temperature: p.Temperature,
		Buildings: buildings, Production: prod, Storage: storage, Queue: queue,
	}
}
```

- [ ] **Step 10: Run all tests**

Run: `go test ./cmd/planet/... -v -count=1`
Expected: all tests PASS

- [ ] **Step 11: Commit**

```bash
git add cmd/planet/types.go cmd/planet/main.go cmd/planet/repository.go cmd/planet/planet.go cmd/planet/planet_test.go
git commit -m "feat: add planet type and temperature"
```

---

### Task 2: Temperature Production Effects

**Files:**
- Modify: `cmd/planet/planet.go`
- Test: `cmd/planet/planet_test.go`

- [ ] **Step 1: Write failing test for temperature production effects**

In `cmd/planet/planet_test.go`, add:

```go
func TestCalculateProduction_ColdGasBonus(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	buildings := []Building{
		{Type: "gas_mine", Level: 1},
	}
	// Cold planet (temp < 0) should get effective +1.5 levels on gas
	coldProd := svc.calculateProduction(buildings, 1.0, -10)
	normalProd := svc.calculateProduction(buildings, 1.0, 20)
	if coldProd.Gas <= normalProd.Gas {
		t.Errorf("cold gas production %.4f should exceed normal %.4f", coldProd.Gas, normalProd.Gas)
	}
}

func TestCalculateProduction_HotSolarBonus(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	buildings := []Building{
		{Type: "solar_plant", Level: 1},
	}
	// Hot planet (temp > 40) should get effective +1.5 levels on solar
	hotProd := svc.calculateProduction(buildings, 1.0, 50)
	normalProd := svc.calculateProduction(buildings, 1.0, 20)
	if hotProd.Energy <= normalProd.Energy {
		t.Errorf("hot solar production %.4f should exceed normal %.4f", hotProd.Energy, normalProd.Energy)
	}
}

func TestCalculateProduction_NoBonusModerate(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	buildings := []Building{
		{Type: "gas_mine", Level: 1},
		{Type: "solar_plant", Level: 1},
	}
	// Moderate temp (0-40) should have no bonus
	prod := svc.calculateProduction(buildings, 1.0, 20)
	if prod.Gas == 0 || prod.Energy == 0 {
		t.Error("expected positive production")
	}
}
```

- [ ] **Step 2: Update calculateProduction signature and implementation**

Change the existing `calculateProduction` to accept `temperature int`:

```go
func (s *PlanetService) calculateProduction(buildings []Building, efficiency float64, temperature int) Production {
	levels := make(map[string]int)
	for _, b := range buildings {
		levels[b.Type] = b.Level
	}

	gasLevel := levels["gas_mine"]
	solarLevel := levels["solar_plant"]

	if temperature < 0 {
		gasLevel = gasLevel + 1 // +1 level effective for gas on cold planets
	}
	if temperature > 40 {
		solarLevel = solarLevel + 1 // +1 level effective for solar on hot planets
	}

	return Production{
		Metal:   productionRate("metal_mine", levels["metal_mine"]) / 60.0 * efficiency,
		Crystal: productionRate("crystal_mine", levels["crystal_mine"]) / 60.0 * efficiency,
		Gas:     productionRate("gas_mine", gasLevel) / 60.0 * efficiency,
		Energy:  productionRate("solar_plant", solarLevel) / 60.0,
	}
}
```

- [ ] **Step 3: Update all callers of calculateProduction**

In `GetOrCreatePlanet`:
```go
prod := s.calculateProduction(buildings, efficiency, planet.Temperature)
```

In `GetMyPlanet` handler:
```go
prod := h.service.calculateProduction(buildings, efficiency, planet.Temperature)
```

- [ ] **Step 4: Update existing test callers**

In `planet_test.go`, update the existing tests that call `calculateProduction` to pass a temperature:

```go
prod := svc.calculateProduction(buildings, 1.0, 20)  // moderate temp
prod := svc.calculateProduction(buildings, 0.5, 20)  // moderate temp
```

- [ ] **Step 5: Run all tests**

Run: `go test ./cmd/planet/... -v -count=1`
Expected: all tests PASS

- [ ] **Step 6: Commit**

```bash
git add cmd/planet/planet.go cmd/planet/planet_test.go
git commit -m "feat: temperature affects gas and solar production"
```

---

### Task 3: Frontend — Planet Type Badge and Temperature

**Files:**
- Modify: `game/src/App.svelte`

- [ ] **Step 1: Add type badge and temperature display**

In `App.svelte`, update the planet header section after the coords line:

```svelte
{#if planet}
  <h1 class="name">{planet.name}</h1>
  <p class="coords">[{planet.galaxy}:{planet.system}:{planet.position}] | {planet.temperature}°C</p>
  <p class="type-badge type-{planet.type}">{planet.type}</p>
  <p class="fields">Fields: {planet.fields_used}/{planet.max_fields}</p>
```

- [ ] **Step 2: Add CSS for type badge and temperature**

```css
.type-badge {
  display: inline-block; font-size: 0.75rem; text-transform: uppercase;
  letter-spacing: 0.05em; padding: 0.15rem 0.5rem; border-radius: 4px;
  margin-bottom: 1rem; font-weight: 600;
}
.type-terran { color: #5aaa5a; background: #1a2a1a; border: 1px solid #2a4a2a; }
.type-desert { color: #d4a574; background: #2a1a0a; border: 1px solid #4a3a1a; }
.type-ice { color: #74c8d4; background: #0a1a2a; border: 1px solid #1a3a4a; }
.type-volcanic { color: #d47474; background: #2a0a0a; border: 1px solid #4a1a1a; }
.type-gas_giant { color: #b474d4; background: #1a0a2a; border: 1px solid #3a1a4a; }
```

- [ ] **Step 3: Commit**

```bash
git add game/src/App.svelte
git commit -m "feat: display planet type badge and temperature"
```

---

### Self-Review Checklist

- [ ] Spec coverage: type/temp schema (Task 1), type assignment (Task 1), production effects (Task 2), frontend display (Task 3)
- [ ] No placeholders: all code blocks filled, no TODOs
- [ ] Type consistency: `planetTypeAndTemp` return matches `Planet.Type`/`Temperature`, `calculateProduction` signature matches all callers
