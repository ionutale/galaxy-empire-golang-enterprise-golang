# Slice 13: VIP & Official Rank

## Summary
Two progression systems that apply % production bonuses to mine output (Metal, Crystal, Gas). Both stack additively.

## VIP System (12 levels)
- VIP points earned from gameplay: each completed building upgrade = +10 VIP points
- VIP level determined by cumulative VIP point thresholds
- Production bonus: +3% per VIP level (capped at +36% at VIP 12)

### VIP Point Thresholds
| VIP Level | Cumulative Points |
|-----------|-----------------|
| 1 | 100 |
| 2 | 500 |
| 3 | 1,500 |
| 4 | 5,000 |
| 5 | 15,000 |
| 6 | 40,000 |
| 7 | 100,000 |
| 8 | 250,000 |
| 9 | 500,000 |
| 10 | 1,000,000 |
| 11 | 2,000,000 |
| 12 | 5,000,000 |

## Official Rank (10 ranks)
- Determined by total resources produced (cumulative lifetime production)
- Each rank gives a flat % production bonus

### Rank Thresholds
| Rank | Title | Total Resources Produced | Bonus |
|------|-------|------------------------|-------|
| 0 | Recruit | 0 | 0% |
| 1 | Private | 1,000,000 | +2% |
| 2 | Corporal | 5,000,000 | +4% |
| 3 | Sergeant | 25,000,000 | +6% |
| 4 | Lieutenant | 100,000,000 | +8% |
| 5 | Captain | 500,000,000 | +10% |
| 6 | Major | 1,000,000,000 | +12% |
| 7 | Colonel | 5,000,000,000 | +15% |
| 8 | General | 25,000,000,000 | +18% |
| 9 | Admiral | 100,000,000,000 | +20% |

## Production Bonus Application
- In `calculateProduction`: `rate * (1 + rankBonus + vipBonus)` applied to metal, crystal, gas mine output
- Energy production unaffected
- Efficiency penalty applied after VIP/rank bonus

## Data
- New table: `planet.player_progress(user_id, vip_points, total_resources_produced, vip_level, official_rank)`
- Created on planet creation, updated on:
  - Building completion (+10 VIP points, +resources to total)
  - Resource tick (total_resources_produced += resources mined this tick)

## Implementation Plan
- Task 1: Schema, repo, migration
- Task 2: Track VIP points + total resources (in processCompletedBuilds + GetOrCreatePlanet)
- Task 3: Apply production bonuses in calculateProduction
- Task 4: Frontend display (badge on dashboard)
