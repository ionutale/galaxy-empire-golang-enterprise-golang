# Slice 10: Construction Management

**Date:** 2026-05-30
**Backlog ref:** Slice 10 — Construction management

## Overview

Two features for managing buildings after they've been queued or built:
1. **Cancel upgrade**: Cancel a queued upgrade, refund 50% of resources
2. **Deconstruct building**: Queue removal of an existing building, free a field, partial refund

## Data Model

### construction_queue table changes

Add a `status` column to distinguish upgrade from deconstruct entries.

```sql
ALTER TABLE planet.construction_queue
  ADD COLUMN status VARCHAR(20) NOT NULL DEFAULT 'upgrade';
```

- `status = 'upgrade'` — normal building upgrade (existing behavior)
- `status = 'deconstruct'` — building removal in progress

Refunds are computed from the cost formula at cancel/deconstruct time (no need to store paid amounts).

### QueueEntry type changes

```go
type QueueEntry struct {
    ID           int       `json:"id"`
    BuildingType string    `json:"building_type"`
    TargetLevel  int       `json:"target_level"`
    Status       string    `json:"status"`
    CompletesAt  time.Time `json:"completes_at"`
}
```

## API Endpoints

### Cancel upgrade

```
POST /api/buildings/{type}/cancel
X-User-ID: <int>

Response 200: {"ok": true}
Response 400: {"error": "no active upgrade for this building"}
```

Logic:
1. Find active queue entry for `building_type` where `status = 'upgrade'`
2. Compute cost paid: `buildingCostResources(type, targetLevel - 1)` 
3. Delete the queue entry
4. Refund 50% of that cost to planet resources
5. Return success

### Deconstruct building

```
POST /api/buildings/{type}/deconstruct
X-User-ID: <int>

Response 200: { <QueueEntry JSON> }
Response 400: {"error": "building not found" / "cannot deconstruct below level 0"}
```

Logic:
1. Validate building exists on planet at level >= 1
2. Do NOT deduct resources upfront (deconstruction has no cost)
3. Compute refund: 50% of `buildingCostResources(type, currentLevel)` (cost of the level being torn down)
4. Compute deconstruction duration: half the upgrade duration for symmetry: `buildingBuildDuration(buildingType, currentLevel-1, roboticsLevel, naniteLevel) / 2`
5. Create queue entry with `status = 'deconstruct'`, `target_level = currentLevel - 1`
6. On completion: decrement building level, refund resources, free a field
7. If building reaches level 0 → DELETE the building row from the DB (frees a field)
8. If building is `terraformer`: also reduce `max_fields` by `terraformerFields(currentLevel) - terraformerFields(currentLevel-1)`

### Process completed builds changes

The existing `processCompletedBuilds` must handle both `upgrade` and `deconstruct` statuses:

- `status = 'upgrade'`: existing behavior (level up, terraformer check)
- `status = 'deconstruct'`:
  - Compute refund: 50% of `buildingCostResources(buildingType, targetLevel + 1)` (cost of the level being removed)
  - Add refund to planet resources
  - Decrement building level
  - If new level == 0: DELETE building row (frees a field)
  - If building is `terraformer`: reduce `max_fields` by `terraformerFields(targetLevel+1) - terraformerFields(targetLevel)`

## Frontend Changes

### Cancel button
- On queue items, add a "Cancel" button (small, red tint) next to the countdown
- On click, call `POST /api/buildings/{type}/cancel`
- Refresh planet data

### Deconstruct button
- On each building row, add a "−" button (or a small deconstruct icon next to the upgrade "+")
- Clicking shows confirmation "Deconstruct this building? 50% resource refund."
- On confirm, call `POST /api/buildings/{type}/deconstruct`
- Refresh planet data

### Queue display
- Queue items show status: "Upgrade to Lv.X" vs "Deconstruct (refund)"
- Deconstruct queue items show a progress bar just like upgrades

## Error Handling

| Error | HTTP Code | Message |
|-------|-----------|---------|
| No active upgrade for this building | 400 | "no active upgrade for this building" |
| Building already in queue for deconstruct | 409 | "building already queued for deconstruction" |
| Cannot deconstruct below level 0 | 400 | "cannot deconstruct this building" |
| Building not found | 400 | "building not found" |

## Testing

- Cancel a queued upgrade → resources refunded at 50%
- Cancel with no active upgrade → error
- Deconstruct a building → queue entry created with deconstruct status
- Process completed deconstruct → building level decremented, refund applied
- Deconstruct to level 0 → building removed, field freed
- Cannot deconstruct a building that doesn't exist
- Frontend: cancel + deconstruct buttons appear/hide correctly
