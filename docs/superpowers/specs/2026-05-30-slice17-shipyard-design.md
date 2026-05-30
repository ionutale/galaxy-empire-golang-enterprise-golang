# Slice 17: Shipyard + All Ships — Design

## Goal

Add Shipyard building and all 12 ship types (civil + combat). Players can build ships from their planet's Shipyard.

## Approach

All in existing `cmd/planet` service — no new microservice. Ship types defined as Go struct constants. Ships stored in `player_ships` table per planet.

## Schema

Add `shipyard` to the existing building seed (level 1 for all planets).

New migration adds one table:

```sql
CREATE TABLE IF NOT EXISTS planet.player_ships (
    id SERIAL PRIMARY KEY,
    planet_id INT NOT NULL REFERENCES planet.planets(id) ON DELETE CASCADE,
    ship_type VARCHAR(50) NOT NULL,
    quantity INT NOT NULL DEFAULT 0,
    UNIQUE(planet_id, ship_type)
);
```

Seed: every existing planet gets 0 of each ship type (INSERT with ON CONFLICT DO NOTHING).

## Ship Configs

Defined as Go constants in `cmd/planet/ships.go`:

```go
type ShipConfig struct {
    Type       string
    Name       string
    Metal      int
    Crystal    int
    Gas        int
    Speed      int
    Cargo      int
    Fuel       int
    Strength   int
    Shield     int
    Attack     int
    Requires   string // tech requirement, ignored for now
}

var Ships = []ShipConfig{ ... }
```

12 ships: Cargo, Large Cargo, Recycler, Espionage Probe, Colony Ship, Solar Satellite (civil), Light Fighter, Heavy Fighter, Cruiser, Battleship, Dreadnought, Bomber (combat).

## API

### `GET /api/shipyard` — List buildable ships

Returns the full `Ships` slice with available quantity for the player's planet.

```json
{
  "shipyard_level": 3,
  "ships": [
    {"type": "cargo", "name": "Cargo", "metal": 2000, "crystal": 2000, "gas": 0,
     "speed": 7500, "cargo": 25000, "fuel": 500,
     "strength": 5, "shield": 5, "attack": 3, "quantity": 5}
  ]
}
```

### `GET /api/shipyard/my-ships` — List ship counts for planet

```json
[
  {"type": "cargo", "quantity": 5},
  {"type": "light_fighter", "quantity": 12}
]
```

### `POST /api/shipyard/build` — Build ships

Request:
```json
{"ship_type": "cargo", "quantity": 10}
```

Validates:
1. Shipyard building exists on planet (level ≥ 1)
2. Player has enough resources
3. Deducts resources, increments quantity in `player_ships`

Response:
```json
{"type": "cargo", "quantity": 15}
```

## Frontend

New "Shipyard" tab in dashboard. Tab toggle button next to Galaxy button.

Ship list: grid of cards, each showing:
- Ship icon (emoji: 🚚 🚀 🔍 🛸 etc.)
- Name + type
- Stats row: Speed / Cargo / Fuel
- Combat row: Strength / Shield / Attack
- Cost breakdown: Metal/Crystal/Gas
- Quantity owned
- Quantity input + Build button (1/10/100/Max)

## Files

New:
- `cmd/planet/ships.go` — ShipConfig type + Ships slice + BuildCost/ShipyardRequirement helpers

Modified:
- `cmd/planet/types.go` — ShipResponse, BuildRequest types
- `cmd/planet/repository.go` — GetPlayerShips, AddShips methods
- `cmd/planet/handler.go` — ListShipyardShips, ListMyShips, BuildShips handlers
- `cmd/planet/planet_test.go` — ship tests
- `cmd/planet/main.go` — migration + routes
- `game/src/App.svelte` — Shipyard tab

## Testing

- `TestShipConfigs` — verify all 12 ships have valid stats (>0 where expected)
- `TestBuildShips_Valid` — build ships, verify quantity incremented and resources deducted
- `TestBuildShips_InsufficientResources` — verify error
- `TestBuildShips_NoShipyard` — verify error
- Handler tests for each endpoint
