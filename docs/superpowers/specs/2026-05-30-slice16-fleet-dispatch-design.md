# Slice 16: Fleet Dispatch + Mission Types — Design

## Goal

Add fleet dispatch: select ships from planet, set target coordinates, choose mission type. New `cmd/fleet` microservice.

## Architecture

- New `cmd/fleet` microservice (port 8083), chi + pgxpool + pgx
- `fleet` schema in same PG instance
- Fleet calls planet's internal `/internal/ships/deduct` endpoint via HTTP for ship validation + deduction
- Follows existing patterns from `cmd/planet` (runMigrations, Repository interface, mockRepo, handlers)

## Schema

```sql
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
```

## Internal Planet Endpoint

### `POST /internal/ships/deduct`

**Request:**
```json
{"planet_id": 1, "ships": {"cargo": 5, "light_fighter": 3}}
```

**Validates:**
- Planet exists
- Player has required quantities of each ship type
- Deducts quantities from `planet.player_ships`

**Response:** `200 OK` on success, `400` with error message on failure

## Fleet Endpoints

### `GET /api/fleet/my-fleets`

Returns all fleets for the authenticated player.

### `POST /api/fleet/dispatch`

**Request:**
```json
{
  "origin_planet_id": 1,
  "ships": {"cargo": 5, "light_fighter": 3},
  "target_galaxy": 1,
  "target_system": 50,
  "target_position": 4,
  "mission": "transport",
  "speed_pct": 100
}
```

**Flow:**
1. Validate ship types exist in Ships config
2. Call planet service `/internal/ships/deduct` to deduct ships
3. Create fleet record with status = `stationed`
4. Return fleet

## Files

**New:**
- `cmd/fleet/main.go` — server setup, routes, migrations
- `cmd/fleet/handler.go` — FleetHandlers (MyFleets, Dispatch)
- `cmd/fleet/service.go` — FleetService (DispatchFleet)
- `cmd/fleet/repository.go` — Repository interface + PostgresRepository (CreateFleet, ListPlayerFleets) + mockRepo
- `cmd/fleet/types.go` — Fleet, FleetResponse, DispatchRequest types

**Modified:**
- `cmd/planet/handler.go` — add InternalDeductShips handler
- `cmd/planet/main.go` — add /internal/* routes (no auth)
- `cmd/planet/repository.go` — add DeductShips method
- `game/src/App.svelte` — Fleet tab + dispatch form

## Ship Deduction (Planet)

New Repository method on planet service:
```go
DeductPlayerShips(ctx, planetID int, ships map[string]int) error
```

Implements:
```sql
UPDATE planet.player_ships
SET quantity = quantity - $3
WHERE planet_id = $1 AND ship_type = $2 AND quantity >= $3;
```
Then check rows affected match expected.

## Internal Routes

Planet service gets a separate router or prefix for internal endpoints:
- `r.Post("/internal/ships/deduct", h.InternalDeductShips)`
- No auth middleware on `/internal/*`
- Only accessible from within internal network (Docker/K8s)

## Testing

- Fleet service: mockRepo, test dispatch flow, edge cases (unknown ship type, insufficient ships)
- Planet service: test InternalDeductShips handler
- Full flow: verify ships deducted after dispatch
