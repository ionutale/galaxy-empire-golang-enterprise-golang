# Slice 11: Planet Types & Temperature

**Date:** 2026-05-30
**Backlog ref:** Slice 11 — Multiple planets + types + temperature

## Overview

Add planet type (`terran`/`desert`/`ice`/`volcanic`/`gas_giant`) and temperature to planets. Temperature affects production: cold boosts gas, hot boosts solar. Display type badge and temp on the dashboard.

## Data Model

### planets table additions

```sql
ALTER TABLE planet.planets
  ADD COLUMN IF NOT EXISTS type VARCHAR(20) NOT NULL DEFAULT 'terran',
  ADD COLUMN IF NOT EXISTS temperature INTEGER NOT NULL DEFAULT 20;
```

### Planet struct additions

```go
type Planet struct {
    // ... existing fields
    Type        string `json:"type"`
    Temperature int    `json:"temperature"`
}
```

### PlanetResponse additions

```go
type PlanetResponse struct {
    // ... existing fields
    Type        string `json:"type"`
    Temperature int    `json:"temperature"`
}
```

## Type Assignment at Creation

Planet type and temperature determined by position in the system:

| Position | Type | Temp Range |
|----------|------|------------|
| 1-3 | desert (80%) / volcanic (20%) | 60-100°C |
| 4-6 | terran (100%) | 10-40°C |
| 7 (home) | terran (100%) | 0-20°C |
| 8-9 | terran (60%) / ice (40%) | -10-30°C |
| 10-12 | ice (100%) | -50-0°C |
| 13-15 | gas_giant (100%) | -80--30°C |

Home planet (position 7) always terran at 0-20°C. For colonies, the type/temp are set at creation based on position. 25% variance is applied to temperature per position.

### PlanetType constants

```go
const (
    PlanetTypeTerran   = "terran"
    PlanetTypeDesert   = "desert"
    PlanetTypeIce      = "ice"
    PlanetTypeVolcanic = "volcanic"
    PlanetTypeGasGiant = "gas_giant"
)
```

### Functions

```go
func planetTypeAndTemp(position int) (typ string, temperature int)
```

## Production Effects

Temperature affects resource production by effectively changing the building level in the formula:

- **Cold (temp < 0°C):** Gas mine effective level = `buildingLevel + 1.5`
- **Hot (temp > 40°C):** Solar plant effective level = `buildingLevel + 1.5`
- **Moderate:** No bonus

Implementation: modify `calculateProduction` to accept temperature and apply effective level adjustments.

```go
func (s *PlanetService) calculateProduction(buildings []Building, efficiency float64, temperature int) Production
```

The effective level is used only in the rate calculation, not stored.

## Frontend

### Planet type badge

Display colored planet type next to the planet name:
- terran → green `🟢`
- desert → orange `🟠`
- ice → cyan `🔵`
- volcanic → red `🔴`
- gas_giant → purple `🟣`

Show as a small pill/badge: `[Terran]`

### Temperature display

Show next to coordinates: `[1:7:7] | 18°C`

### Color scheme CSS

```css
.type-terran { color: #5aaa5a; }
.type-desert { color: #d4a574; }
.type-ice { color: #74c8d4; }
.type-volcanic { color: #d47474; }
.type-gas_giant { color: #b474d4; }
```

## Testing

- Create a planet at position 7 → type is terran, temp in range
- Create planets at various positions → correct type per rules
- Temperature: cold planet gas production rate matches effective level+1.5
- Temperature: hot planet solar production rate matches effective level+1.5
- Moderate temp: no production change
- Frontend: planet type badge and temperature display
