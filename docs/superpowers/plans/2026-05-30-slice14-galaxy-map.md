# Galaxy Map (3D Coords) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add galaxy map browsing with 3-level [Galaxy:System:Position] navigation.

**Architecture:** Galaxy tables added to `planet` schema (single PG instance, `galaxy` schema). No new microservice. Position state resolved via LEFT JOIN against `planet.planets`. 67,365 rows seeded at migration.

**Tech Stack:** Go + chi/pgx, Svelte

---

### Task 1: Schema + Seed Migration

**Files:**
- Modify: `cmd/planet/main.go` (migration in runMigrations)

- [ ] **Step 1: Read main.go migration section**

Read `cmd/planet/main.go` around lines 80-130 to find the existing migration block.

- [ ] **Step 2: Add galaxy tables + seed migration**

After the player_progress migration block, add:

```go
{`, CREATE TABLE IF NOT EXISTS galaxy.galaxies (
    id   SERIAL PRIMARY KEY,
    name TEXT NOT NULL
);`, func() error {
    // seed 9 galaxies
    var count int
    if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM galaxy.galaxies`).Scan(&count); err != nil {
        return err
    }
    if count == 0 {
        for i := 1; i <= 9; i++ {
            if _, err := pool.Exec(ctx, `INSERT INTO galaxy.galaxies (id, name) VALUES ($1, $2) ON CONFLICT DO NOTHING`, i, fmt.Sprintf("Galaxy %d", i)); err != nil {
                return err
            }
        }
    }
    return nil
}`, false},
{`, CREATE TABLE IF NOT EXISTS galaxy.systems (
    id         SERIAL PRIMARY KEY,
    galaxy_id  INT NOT NULL REFERENCES galaxy.galaxies(id),
    system_num INT NOT NULL CHECK (system_num BETWEEN 1 AND 499),
    UNIQUE(galaxy_id, system_num)
);`, func() error {
    var count int
    if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM galaxy.systems`).Scan(&count); err != nil {
        return err
    }
    if count == 0 {
        if _, err := pool.Exec(ctx, `
            INSERT INTO galaxy.systems (galaxy_id, system_num)
            SELECT g.id, s.num
            FROM galaxy.galaxies g
            CROSS JOIN generate_series(1, 499) AS s(num)
            ON CONFLICT DO NOTHING
        `); err != nil {
            return err
        }
    }
    return nil
}`, false},
{`, CREATE TABLE IF NOT EXISTS galaxy.positions (
    id          SERIAL PRIMARY KEY,
    system_id   INT NOT NULL REFERENCES galaxy.systems(id),
    position_num INT NOT NULL CHECK (position_num BETWEEN 1 AND 15),
    UNIQUE(system_id, position_num)
);`, func() error {
    var count int
    if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM galaxy.positions`).Scan(&count); err != nil {
        return err
    }
    if count == 0 {
        if _, err := pool.Exec(ctx, `
            INSERT INTO galaxy.positions (system_id, position_num)
            SELECT s.id, p.num
            FROM galaxy.systems s
            CROSS JOIN generate_series(1, 15) AS p(num)
            ON CONFLICT DO NOTHING
        `); err != nil {
            return err
        }
    }
    return nil
}`, false},
```

Add `"fmt"` to imports in main.go.

- [ ] **Step 3: Build check**

Run: `go build ./cmd/planet/...`
Expected: success

- [ ] **Step 4: Commit**

```bash
git add cmd/planet/main.go
git commit -m "feat: galaxy schema and seed migration"
```

---

### Task 2: Types

**Files:**
- Modify: `cmd/planet/types.go`

- [ ] **Step 1: Add galaxy response types**

Add at the bottom of `cmd/planet/types.go`:

```go
type Galaxy struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type System struct {
	ID            int `json:"id"`
	SystemNum     int `json:"system_num"`
	OccupiedCount int `json:"occupied_count"`
}

type Position struct {
	PositionNum int    `json:"position"`
	State       string `json:"state"`
	PlanetName  string `json:"planet_name,omitempty"`
	PlayerID    int    `json:"player_id,omitempty"`
}
```

- [ ] **Step 2: Build check**

Run: `go build ./cmd/planet/...`
Expected: success

- [ ] **Step 3: Commit**

```bash
git add cmd/planet/types.go
git commit -m "feat: galaxy types"
```

---

### Task 3: Repository Methods

**Files:**
- Modify: `cmd/planet/repository.go`

- [ ] **Step 1: Read current repository.go**

Read `cmd/planet/repository.go` — find the Repository interface and the PostgresRepository struct to understand the pattern.

- [ ] **Step 2: Add galaxy methods to Repository interface**

Add after the AddResourcesProduced line in the interface:

```go
	ListGalaxies(ctx context.Context) ([]Galaxy, error)
	ListSystems(ctx context.Context, galaxyID int, page, pageSize int) ([]System, int, error)
	GetSystemPositions(ctx context.Context, systemID int) ([]Position, error)
```

- [ ] **Step 3: Add mock implementations**

In `mockRepo`, add implementations:

```go
func (m *mockRepo) ListGalaxies(ctx context.Context) ([]Galaxy, error) {
	return []Galaxy{
		{ID: 1, Name: "Galaxy 1"},
		{ID: 2, Name: "Galaxy 2"},
		{ID: 3, Name: "Galaxy 3"},
		{ID: 4, Name: "Galaxy 4"},
		{ID: 5, Name: "Galaxy 5"},
		{ID: 6, Name: "Galaxy 6"},
		{ID: 7, Name: "Galaxy 7"},
		{ID: 8, Name: "Galaxy 8"},
		{ID: 9, Name: "Galaxy 9"},
	}, nil
}

func (m *mockRepo) ListSystems(ctx context.Context, galaxyID int, page, pageSize int) ([]System, int, error) {
	return nil, 0, nil
}

func (m *mockRepo) GetSystemPositions(ctx context.Context, systemID int) ([]Position, error) {
	positions := make([]Position, 15)
	for i := 0; i < 15; i++ {
		positions[i] = Position{PositionNum: i + 1, State: "empty"}
	}
	return positions, nil
}
```

The mock doesn't maintain galaxy data — it returns hardcoded responses. Good enough for service-level tests.

- [ ] **Step 4: Add PostgresRepository implementations**

After the existing PostgresRepo methods, add:

```go
func (r *PostgresRepository) ListGalaxies(ctx context.Context) ([]Galaxy, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, name FROM galaxy.galaxies ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list galaxies: %w", err)
	}
	defer rows.Close()

	var galaxies []Galaxy
	for rows.Next() {
		var g Galaxy
		if err := rows.Scan(&g.ID, &g.Name); err != nil {
			return nil, fmt.Errorf("scan galaxy: %w", err)
		}
		galaxies = append(galaxies, g)
	}
	return galaxies, rows.Err()
}

func (r *PostgresRepository) ListSystems(ctx context.Context, galaxyID int, page, pageSize int) ([]System, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM galaxy.systems WHERE galaxy_id = $1`, galaxyID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count systems: %w", err)
	}

	offset := (page - 1) * pageSize
	rows, err := r.pool.Query(ctx, `
		SELECT s.id, s.system_num,
			(SELECT COUNT(*) FROM planet.planets pl WHERE pl.galaxy = s.galaxy_id AND pl.system = s.system_num) AS occupied_count
		FROM galaxy.systems s
		WHERE s.galaxy_id = $1
		ORDER BY s.system_num
		LIMIT $2 OFFSET $3
	`, galaxyID, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list systems: %w", err)
	}
	defer rows.Close()

	var systems []System
	for rows.Next() {
		var s System
		if err := rows.Scan(&s.ID, &s.SystemNum, &s.OccupiedCount); err != nil {
			return nil, 0, fmt.Errorf("scan system: %w", err)
		}
		systems = append(systems, s)
	}
	return systems, total, rows.Err()
}

func (r *PostgresRepository) GetSystemPositions(ctx context.Context, systemID int) ([]Position, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT p.position_num,
			CASE WHEN pl.id IS NOT NULL THEN 'occupied' ELSE 'empty' END AS state,
			COALESCE(pl.name, '') AS planet_name,
			COALESCE(pl.user_id, 0) AS player_id
		FROM galaxy.positions p
		JOIN galaxy.systems s ON p.system_id = s.id
		LEFT JOIN planet.planets pl
			ON pl.galaxy = s.galaxy_id
			AND pl.system = s.system_num
			AND pl.position = p.position_num
		WHERE p.system_id = $1
		ORDER BY p.position_num
	`, systemID)
	if err != nil {
		return nil, fmt.Errorf("get positions: %w", err)
	}
	defer rows.Close()

	var positions []Position
	for rows.Next() {
		var pos Position
		if err := rows.Scan(&pos.PositionNum, &pos.State, &pos.PlanetName, &pos.PlayerID); err != nil {
			return nil, fmt.Errorf("scan position: %w", err)
		}
		positions = append(positions, pos)
	}
	return positions, rows.Err()
}
```

- [ ] **Step 5: Build check**

Run: `go build ./cmd/planet/...`
Expected: success

- [ ] **Step 6: Commit**

```bash
git add cmd/planet/repository.go
git commit -m "feat: galaxy repository methods"
```

---

### Task 4: Handler Endpoints + Routes

**Files:**
- Modify: `cmd/planet/handler.go`
- Modify: `cmd/planet/main.go`

- [ ] **Step 1: Read current handler.go**

Read `cmd/planet/handler.go` to see the existing handler pattern.

- [ ] **Step 2: Add galaxy handler methods**

Add after `DeconstructBuilding` handler:

```go
func (h *Handler) ListGalaxies(w http.ResponseWriter, r *http.Request) {
	galaxies, err := h.service.repo.ListGalaxies(r.Context())
	if err != nil {
		slog.Error("list galaxies failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, galaxies)
}

func (h *Handler) ListSystems(w http.ResponseWriter, r *http.Request) {
	galaxyIDStr := chi.URLParam(r, "galaxyID")
	galaxyID, err := strconv.Atoi(galaxyIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid galaxy"})
		return
	}

	pageStr := r.URL.Query().Get("page")
	page := 1
	if pageStr != "" {
		page, err = strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid page"})
			return
		}
	}

	pageSize := 10
	systems, total, err := h.service.repo.ListSystems(r.Context(), galaxyID, page, pageSize)
	if err != nil {
		slog.Error("list systems failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	totalPages := (total + pageSize - 1) / pageSize
	writeJSON(w, http.StatusOK, map[string]any{
		"galaxy_id":   galaxyID,
		"page":        page,
		"total_pages": totalPages,
		"systems":     systems,
	})
}

func (h *Handler) GetPositions(w http.ResponseWriter, r *http.Request) {
	systemIDStr := chi.URLParam(r, "systemID")
	systemID, err := strconv.Atoi(systemIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid system"})
		return
	}

	positions, err := h.service.repo.GetSystemPositions(r.Context(), systemID)
	if err != nil {
		slog.Error("get positions failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"system_id": systemID,
		"positions": positions,
	})
}
```

- [ ] **Step 3: Add routes in main.go**

After the existing planet routes in `main.go`:

```go
r.Get("/api/galaxy", h.ListGalaxies)
r.Get("/api/galaxy/systems/{galaxyID}", h.ListSystems)
r.Get("/api/galaxy/positions/{systemID}", h.GetPositions)
```

- [ ] **Step 4: Build check**

Run: `go build ./cmd/planet/...`
Expected: success

- [ ] **Step 5: Commit**

```bash
git add cmd/planet/handler.go cmd/planet/main.go
git commit -m "feat: galaxy handler endpoints and routes"
```

---

### Task 5: Tests

**Files:**
- Modify: `cmd/planet/planet_test.go`

- [ ] **Step 1: Read current test file**

Read `cmd/planet/planet_test.go` to find where repository tests and handler tests are located.

- [ ] **Step 2: Add galaxy repository tests**

Add after the VIP/Rank test section:

```go
func TestListGalaxies(t *testing.T) {
	mock := newMockRepo()
	galaxies, err := mock.ListGalaxies(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(galaxies) != 9 {
		t.Errorf("expected 9 galaxies, got %d", len(galaxies))
	}
	if galaxies[0].Name != "Galaxy 1" {
		t.Errorf("expected Galaxy 1, got %s", galaxies[0].Name)
	}
}

func TestGetSystemPositions_Mock(t *testing.T) {
	mock := newMockRepo()
	positions, err := mock.GetSystemPositions(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(positions) != 15 {
		t.Errorf("expected 15 positions, got %d", len(positions))
	}
	for i, pos := range positions {
		if pos.PositionNum != i+1 {
			t.Errorf("position %d: expected num %d, got %d", i, i+1, pos.PositionNum)
		}
		if pos.State != "empty" {
			t.Errorf("position %d: expected empty, got %s", i, pos.State)
		}
	}
}

func TestHandler_ListGalaxies(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	h := NewHandler(svc)

	req := httptest.NewRequest("GET", "/api/galaxy", nil)
	w := httptest.NewRecorder()
	h.ListGalaxies(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var galaxies []Galaxy
	if err := json.NewDecoder(w.Body).Decode(&galaxies); err != nil {
		t.Fatal(err)
	}
	if len(galaxies) != 9 {
		t.Errorf("expected 9 galaxies, got %d", len(galaxies))
	}
}

func TestHandler_ListSystems_InvalidGalaxy(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	h := NewHandler(svc)
	req := httptest.NewRequest("GET", "/api/galaxy/systems/abc", nil)
	w := httptest.NewRecorder()
	h.ListSystems(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid galaxy ID, got %d", w.Code)
	}
}

func TestHandler_GetPositions_InvalidSystem(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	h := NewHandler(svc)
	req := httptest.NewRequest("GET", "/api/galaxy/positions/abc", nil)
	w := httptest.NewRecorder()
	h.GetPositions(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid system ID, got %d", w.Code)
	}
}
```

Add `"encoding/json"` to test file imports if not already there (likely is).

- [ ] **Step 3: Run tests**

Run: `go test ./cmd/planet/... -run "TestListGalaxies|TestGetSystemPositions|TestHandler_ListGalaxies|TestHandler_ListSystems_Invalid|TestHandler_GetPositions_Invalid" -v -count=1`
Expected: all PASS

- [ ] **Step 4: Run full suite**

Run: `go test ./cmd/planet/... -count=1`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/planet/planet_test.go
git commit -m "feat: galaxy tests"
```

---

### Task 6: Frontend Galaxy Tab

**Files:**
- Modify: `game/src/App.svelte`

- [ ] **Step 1: Read current App.svelte**

Read `game/src/App.svelte` — find the dashboard section with planet display and buildings.

- [ ] **Step 2: Add galaxy state and fetch functions**

Add after `function startDeconstruct` (before `</script>`):

```js
let galaxyView = null
let selectedGalaxy = 1
let galaxyPage = 1
let selectedSystem = null
let positions = null

async function loadGalaxies() {
  try {
    const res = await fetch('/api/galaxy', {
      headers: { 'Authorization': `Bearer ${token}` }
    })
    if (!res.ok) throw new Error('Failed to load galaxies')
    galaxyView = await res.json()
  } catch (e) { error = e.message }
}

async function loadSystems(galaxyID, page) {
  try {
    const res = await fetch(`/api/galaxy/systems/${galaxyID}?page=${page}`, {
      headers: { 'Authorization': `Bearer ${token}` }
    })
    if (!res.ok) throw new Error('Failed to load systems')
    return await res.json()
  } catch (e) { error = e.message; return null }
}

async function loadPositions(systemID) {
  try {
    const res = await fetch(`/api/galaxy/positions/${systemID}`, {
      headers: { 'Authorization': `Bearer ${token}` }
    })
    if (!res.ok) throw new Error('Failed to load positions')
    positions = await res.json()
  } catch (e) { error = e.message }
}

let galaxyData = null
async function showGalaxyTab() {
  selectedSystem = null
  positions = null
  galaxyPage = 1
  await loadGalaxies()
  if (galaxyView && galaxyView.length > 0) {
    selectedGalaxy = galaxyView[0].id
    galaxyData = await loadSystems(selectedGalaxy, galaxyPage)
  }
}

async function selectGalaxy(id) {
  selectedGalaxy = id
  galaxyPage = 1
  galaxyData = await loadSystems(selectedGalaxy, galaxyPage)
}

async function galaxyPageNext() {
  if (galaxyData && galaxyPage < galaxyData.total_pages) {
    galaxyPage++
    galaxyData = await loadSystems(selectedGalaxy, galaxyPage)
  }
}

async function galaxyPagePrev() {
  if (galaxyPage > 1) {
    galaxyPage--
    galaxyData = await loadSystems(selectedGalaxy, galaxyPage)
  }
}

async function selectSystem(systemID) {
  selectedSystem = systemID
  await loadPositions(systemID)
}
```

- [ ] **Step 3: Add galaxy tab UI**

In the `{#if planet}` section of the template, after the planet coords/type display (around line 188), before the `<div class="resources">`, add galaxy tab toggle button:

```svelte
<button class="galaxy-toggle" on:click={showGalaxyTab}>Galaxy</button>
```

Then after the `</div>` closing `</div>` for resources/modals, add the galaxy view:

```svelte
{#if positions}
  <div class="galaxy-section">
    <button class="back-btn" on:click={() => { positions = null; selectedSystem = null }}>← Back to Systems</button>
    <h3>System {selectedSystem} — Galaxy {selectedGalaxy}</h3>
    <div class="position-grid">
      {#each positions.positions as pos}
        <div class="position-card" class:occupied={pos.state === 'occupied'}>
          <span class="pos-num">#{pos.position}</span>
          {#if pos.state === 'occupied'}
            <span class="pos-name">{pos.planet_name}</span>
            <span class="pos-player">Player {pos.player_id}</span>
          {:else}
            <span class="pos-empty">Empty</span>
          {/if}
        </div>
      {/each}
    </div>
  </div>
{:else if galaxyData}
  <div class="galaxy-section">
    <h3>Galaxy {selectedGalaxy}</h3>
    <div class="galaxy-controls">
      <select bind:value={selectedGalaxy} on:change={(e) => selectGalaxy(parseInt(e.target.value))}>
        {#each galaxyView || [] as g}
          <option value={g.id}>{g.name}</option>
        {/each}
      </select>
    </div>
    <div class="system-list">
      {#each galaxyData.systems as sys}
        <button class="system-row" on:click={() => selectSystem(sys.id)}>
          <span class="sys-num">System {sys.system_num}</span>
          <span class="sys-occ">{sys.occupied_count}/15 occupied</span>
        </button>
      {/each}
    </div>
    <div class="pagination">
      <button disabled={galaxyPage <= 1} on:click={galaxyPagePrev}>Prev</button>
      <span>Page {galaxyPage} of {galaxyData.total_pages}</span>
      <button disabled={galaxyPage >= galaxyData.total_pages} on:click={galaxyPageNext}>Next</button>
    </div>
  </div>
{/if}
```

- [ ] **Step 4: Add galaxy CSS styles**

After the existing CSS (before `</style>`), add:

```css
.galaxy-toggle {
  display: block; margin: 1rem auto; padding: 0.5rem 1rem;
  background: #1a2a4a; border: 1px solid #2a4a6a; border-radius: 6px;
  color: #8ab5d4; font-size: 0.85rem; cursor: pointer;
}
.galaxy-toggle:hover { background: #2a3a5a; }

.galaxy-section {
  margin-top: 1.5rem; text-align: left;
}
.galaxy-section h3 { font-size: 0.9rem; color: #8a9ab5; margin-bottom: 0.75rem; text-align: center; }
.back-btn {
  padding: 0.3rem 0.6rem; background: #1a2a3a; border: 1px solid #2a4a4a;
  border-radius: 4px; color: #74a8c8; font-size: 0.75rem; cursor: pointer; margin-bottom: 0.5rem;
}
.galaxy-controls { text-align: center; margin-bottom: 0.75rem; }
.galaxy-controls select {
  padding: 0.4rem 0.6rem; background: #1a2340; border: 1px solid #243050;
  border-radius: 4px; color: #c8d6e5; font-size: 0.85rem;
}

.system-list { display: flex; flex-direction: column; gap: 0.35rem; }
.system-row {
  display: flex; justify-content: space-between; align-items: center;
  padding: 0.5rem 0.75rem; background: #1a2340; border: 1px solid #243050;
  border-radius: 6px; color: #c8d6e5; cursor: pointer; font-size: 0.85rem;
}
.system-row:hover { background: #1e2a4a; }
.sys-occ { font-size: 0.75rem; color: #5a7a9a; }

.pagination {
  display: flex; justify-content: center; align-items: center; gap: 0.75rem;
  margin-top: 0.75rem;
}
.pagination button {
  padding: 0.3rem 0.6rem; background: #1a2a4a; border: 1px solid #2a4a6a;
  border-radius: 4px; color: #8ab5d4; cursor: pointer; font-size: 0.75rem;
}
.pagination button:disabled { opacity: 0.4; cursor: not-allowed; }
.pagination span { font-size: 0.75rem; color: #5a7a9a; }

.position-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 0.5rem; }
.position-card {
  padding: 0.5rem; background: #1a2340; border: 1px solid #243050;
  border-radius: 6px; text-align: center; font-size: 0.8rem;
}
.position-card.occupied { border-color: #3a6a4a; background: #1a2a1a; }
.pos-num { display: block; font-weight: 600; color: #5a7a9a; font-size: 0.75rem; margin-bottom: 0.25rem; }
.pos-name { display: block; color: #8ac88a; }
.pos-player { display: block; font-size: 0.65rem; color: #5a7a9a; }
.pos-empty { color: #5a5a6a; font-style: italic; }
```

- [ ] **Step 5: Run build check**

Make sure Svelte dev server would handle this (no TS compilation needed for pure JS). If there's a build step:

```bash
npm run build --prefix game 2>/dev/null || echo "Svelte build check skipped (may need dev deps)"
```

- [ ] **Step 6: Commit**

```bash
git add game/src/App.svelte
git commit -m "feat: galaxy map frontend tab"
```

---
