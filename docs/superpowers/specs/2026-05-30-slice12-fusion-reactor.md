# Slice 12: Fusion Reactor

## Summary
Fusion Reactor building that produces significant energy but consumes gas. Gated by Gas Mine Lv5 + Energy Tech Lv3.

## Gating
- Requires Gas Mine level ≥ 5 and Energy Tech level ≥ 3
- Energy Tech stored in new `planet.player_technologies` table, seeded at Lv3 for all players in migration
- `ErrPrerequisitesNotMet` sentinel error for failed gating checks
- Research service adopts `player_technologies` management in Slice 50+

## Data
- New table: `planet.player_technologies(user_id, type, level)` — seeded with `energy_tech` Lv3
- New building type: `fusion_reactor` — seeded for new/existing planets via migration

## Costs
- `metal = 200 × 2^(level+1)`, `crystal = 150 × 2^(level+1)`, `gas = 50 × 2^(level+1)`

## Production
- Fusion energy output: `50 × level × 1.1^level × (1 + 0.05 × energyTechLevel)` per minute
- Gas consumption: `10 × level × 1.1^level` per minute (deducted from gas production before storage)

## Energy Balance
- `calculatePenaltyFactor`: fusion added to total energy production
- `calculateProduction`: fusion energy output added to Production.Energy
- Gas production shown is net (after fusion consumption)

## Gating check
- `StartBuildingUpgrade` for `fusion_reactor`: verify `GetBuildingLevel("gas_mine") >= 5` and `GetTechLevel("energy_tech") >= 3`

## Frontend
- Add "Fusion Reactor" to building labels array in App.svelte
- Shows in building grid like other buildings

## Testing
- Seed buildings include fusion_reactor Lv1
- `TestFusionReactor_Gating_GasMineTooLow` — fails with gating error
- `TestFusionReactor_Gating_EnergyTechTooLow` — fails with gating error
- `TestFusionReactor_ProducesEnergy` — fusion reactor contributes +energy
- `TestFusionReactor_ConsumesGas` — gas production reduced by consumption
- `TestFusionReactor_GatePasses` — successful build with both prereqs met
