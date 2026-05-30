# Slice 18: Batch Building

## Changes

### Backend (planet service)

1. **`BuildShips` response** — add `build_time_seconds` and `max_quantity` fields
2. **Add `CalculateMaxShipQuantity` helper** — min over `floor(metal/cost.metal)`, `floor(crystal/cost.crystal)`, `floor(gas/cost.gas)`
3. **Build time formula**: `T(h) = ((M+C)/(11132×(ShipyardLevel+1))) / 2^NaniteLevel`, converted to seconds
4. **New endpoint** `GET /api/shipyard/build-info?type=cargo&quantity=5` — returns cost + build time + max quantity without actually building

### Frontend

1. **Shipyard tab** — quantity selector per ship with buttons: 1, 10, 100, Max
2. **Show build time** next to the quantity selector (maybe on hover or as text)
