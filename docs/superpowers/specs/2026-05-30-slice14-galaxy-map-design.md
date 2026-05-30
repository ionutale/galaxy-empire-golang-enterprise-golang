# Slice 14: Galaxy Map (3D Coords) — Design

## Goal

Add galaxy map browsing: navigate [Galaxy:System:Position] hierarchy, see position state (occupied/owner/empty).

## Approach

In-planet service (no separate `cmd/galaxy` microservice). Galaxy tables added to `planet` schema, endpoints added to existing handler. Extract later if needed.

## Schema

All in `planet` schema:

```sql
CREATE TABLE IF NOT EXISTS galaxy.galaxies (
    id   SERIAL PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS galaxy.systems (
    id         SERIAL PRIMARY KEY,
    galaxy_id  INT NOT NULL REFERENCES galaxy.galaxies(id),
    system_num INT NOT NULL CHECK (system_num BETWEEN 1 AND 499),
    UNIQUE(galaxy_id, system_num)
);

CREATE TABLE IF NOT EXISTS galaxy.positions (
    id          SERIAL PRIMARY KEY,
    system_id   INT NOT NULL REFERENCES galaxy.systems(id),
    position_num INT NOT NULL CHECK (position_num BETWEEN 1 AND 15),
    UNIQUE(system_id, position_num)
);
```

### Seed

Migration generates 9 galaxies → 4,491 systems → 67,365 positions using `generate_series()`.

```
Galaxies: 9 rows (Andromeda, Milky Way, etc.)
Systems:  9 × 499 = 4,491
Positions: 4,491 × 15 = 67,365
```

### Position State

Position state resolved via LEFT JOIN against `planet.planets`:

| State | Condition |
|---|---|
| `empty` | No planet at (galaxy, system, position) |
| `occupied` | Planet exists, includes `planet_name`, `player_id` |

## API

### `GET /api/galaxy` — List all galaxies

```json
[
  {"id": 1, "name": "Andromeda"},
  {"id": 2, "name": "Milky Way"}
]
```

### `GET /api/galaxy/systems?galaxy_id=1&page=1` — Paginated systems for a galaxy

```json
{
  "galaxy_id": 1,
  "page": 1,
  "total_pages": 50,
  "systems": [
    {"id": 1, "system_num": 1, "occupied_count": 3},
    {"id": 2, "system_num": 2, "occupied_count": 0}
  ]
}
```

Page size: 10 systems per page.

### `GET /api/galaxy/positions?system_id=1` — 15 positions for a system

```json
{
  "system_id": 1,
  "galaxy_id": 1,
  "system_num": 1,
  "positions": [
    {"position": 1, "state": "occupied", "planet_name": "Home", "player_id": 1},
    {"position": 2, "state": "empty"},
    ...
  ]
}
```

## Frontend

New "Galaxy" tab next to "Planet" heading. 3-level navigation:

1. **Galaxy selector** — dropdown with 9 galaxy names
2. **System list** — paginated grid (10/page), shows system number + occupied count
3. **Position detail** — 15-position grid, each showing:
   - Occupied → planet name + player name (clickable, future)
   - Empty → "Empty" label

No fancy canvas/map — clean table-style layout.

## Files Changed

- `cmd/planet/types.go` — GalaxyResponse, SystemResponse, PositionResponse structs
- `cmd/planet/repository.go` — GalaxyRepository interface + Postgres methods
- `cmd/planet/handler.go` — 3 new handler endpoints
- `cmd/planet/planet_test.go` — repository + handler tests
- `cmd/planet/main.go` — migration + routes
- `game/src/App.svelte` — galaxy tab + navigation
