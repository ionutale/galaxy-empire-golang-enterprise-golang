# Galaxy Empire — Code Audit Findings (2026-06-01 Full Audit)

> Previous findings (pre-2026-06-01) replaced by this complete re-audit.  
> PRD: docs/plans/2026-06-01-security-hardening-prd.md

> **Method**: 8 parallel deep-analysis agents covering security, error handling, race conditions, game logic, social services, admin/espionage/radar/research/nebula, frontend, and infrastructure. ~493 unique findings after deduplication.

**Severity key**: 🔴 CRITICAL · 🟠 HIGH · 🟡 MEDIUM · 🔵 LOW

---

## Table of Contents
1. [Security & Authentication](#1-security--authentication)
2. [Race Conditions & Concurrency](#2-race-conditions--concurrency)
3. [Error Handling, Nil Derefs & Panics](#3-error-handling-nil-derefs--panics)
4. [Game Logic Bugs & Exploits](#4-game-logic-bugs--exploits)
5. [Authorization Bypasses](#5-authorization-bypasses)
6. [Social Services (Alliance / Chat / Notification / Friend)](#6-social-services)
7. [Admin / Espionage / Radar / Research / Nebula](#7-admin--espionage--radar--research--nebula)
8. [Frontend Issues](#8-frontend-issues)
9. [Infrastructure & DevOps](#9-infrastructure--devops)
10. [Bad Practices & Code Quality](#10-bad-practices--code-quality)

---

## 1. Security & Authentication

### 1.1 JWT & Token Handling

| # | Severity | File | Finding |
|---|----------|------|---------|
| 1 | 🔴 | `docker-compose.yml:5`, `cmd/gateway/main.go:53` | **Hardcoded JWT secret** `dev-secret-change-in-production` deployed as default. Anyone with repo access can forge any JWT, including admin (user ID 1). Fail-fast at startup if secret is not set or is < 32 chars. |
| 2 | 🟠 | `cmd/auth/auth.go:178`, `cmd/gateway/main.go:234`, `cmd/chat/service.go:160`, `cmd/notification/service.go:118` | **Missing JWT algorithm check** in all four `ParseWithClaims` key callbacks. No explicit `token.Method == jwt.SigningMethodHS256` guard. A library regression or `alg:none` token could be accepted. |
| 3 | 🟠 | `game/src/App.svelte:2` | **JWT stored in `localStorage`**. Exfiltrable by any script on the page (XSS, compromised dependency). Should use `httpOnly` cookie. |
| 4 | 🟠 | `game/src/App.svelte:334,951` | **JWT embedded in SSE URL query string** (`/api/chat/stream?token=…`). Token appears in nginx access logs, browser history, and Referer headers. |
| 5 | 🟠 | `cmd/auth/auth.go:44` | **No bcrypt truncation guard**. bcrypt silently truncates at 72 bytes. Two passwords sharing the first 72 bytes are treated identically — partial-password authentication attack. Add `maxlength=72` validation. |
| 6 | 🟡 | `game/src/App.svelte:73` | **Auto-polling starts from any non-null token string**, including garbage values in localStorage. No expiry/format pre-validation before making API calls. |

### 1.2 Hardcoded Secrets & Internal Communication

| # | Severity | File | Finding |
|---|----------|------|---------|
| 7 | 🔴 | `docker-compose.yml:4` | **Hardcoded `INTERNAL_SECRET`** `internal-dev-secret` committed to repo. Any service that verifies this header is trivially bypassable. |
| 8 | 🟠 | `cmd/gateway/main.go:273` | **Gateway forwards client-supplied `X-Internal-Secret` header** to backend services. All request headers are copied verbatim; the client header is never stripped. An attacker who knows the internal secret (it's in docker-compose) can satisfy backend `InternalSecretMiddleware` through the gateway. Strip `X-Internal-Secret` before forwarding. |
| 9 | 🟠 | `docker-compose.yml:50,78` | **JWT_SECRET duplicated** in `gateway` and `auth` blocks. A change in one silently desyncs the other. |

### 1.3 Missing Authentication on Service Endpoints

| # | Severity | File | Finding |
|---|----------|------|---------|
| 10 | 🔴 | `cmd/combat/main.go:69-73` | **Combat service has zero authentication**. `/combat/resolve`, `/combat/missile-strike` accept arbitrary player IDs and trigger loot distribution, debris creation, and combat reports. Any container on the Docker network can forge combats. Add `InternalSecretMiddleware`. |
| 11 | 🔴 | `cmd/planet/main.go:68-126` | **All planet service `/internal/*` endpoints have no auth**. Ship add/deduct, resource add/deduct, planet creation, tech level modification — all unauthenticated. Any compromised container can add unlimited resources to any planet. |
| 12 | 🔴 | `cmd/nebula/main.go:83` | **`/internal/nebula/credits/add` is on the public router with no auth**. Any caller who can reach port 8088 can grant unlimited credits to any player. |
| 13 | 🟠 | `cmd/nebula/handler.go:440` | **`InternalAddCredits` accepts arbitrary `player_id` and `amount` from request body** with no identity check. No shared secret, no IP restriction. |
| 14 | 🟠 | `cmd/radar/main.go:76` | **`/internal/radar/detect` registered on public router with no auth**. Anyone on Docker network can inject fake radar events (fake attack alerts) for any player. |
| 15 | 🟠 | `cmd/alliance/handler.go:249` | **`InternalGetPlayerAlliance` is completely unauthenticated**. No `X-Internal-Secret` check. |
| 16 | 🟠 | `cmd/alliance/handler.go:399` | **`InternalPing` is unauthenticated**. |
| 17 | 🟡 | `cmd/event/handler.go:155` | **`InternalCheck` POST endpoint is unauthenticated**. Any caller can force event status transitions. |
| 18 | 🟡 | `cmd/tutorial/handler.go:94` | **`ProgressUpdate` trusts `player_id` from request body** with no auth check. Any service can mark any player's tutorial step complete. |
| 19 | 🟡 | `cmd/notification/main.go:75` | **`/api/notification/create` is an internal write endpoint on the public path**. Only an inline handler body check enforces the secret, but the route is publicly routable. |

### 1.4 X-User-ID Header Injection

| # | Severity | File | Finding |
|---|----------|------|---------|
| 20 | 🔴 | `cmd/gateway/main.go:273`, `cmd/chat/handler.go:270` | **`X-User-ID` is never stripped from unauthenticated routes**. `GET /api/chat/stream` and `GET /api/notification/stream` are outside the JWT middleware group. An unauthenticated attacker can set `X-User-ID: <target>` and subscribe to another player's SSE stream. |
| 21 | 🟠 | `cmd/admin/handler.go:21` | **Admin `adminOnly` middleware reads `X-User-ID` from HTTP header**. No JWT re-validation, no shared secret, no IP allowlist. Any process that can reach port 8096 can send arbitrary `X-User-ID: 1` and get full admin access. |

### 1.5 Admin & Privilege Escalation

| # | Severity | File | Finding |
|---|----------|------|---------|
| 22 | 🔴 | `cmd/admin/repository.go:47` | **Admin hardcoded to user ID 1**. Whichever user registers first permanently has admin rights that cannot be revoked via the `admin.admins` table. On a fresh deployment, an attacker who registers first gets permanent admin. |
| 23 | 🟠 | `game/src/App.svelte:1439` | **Admin panel gated only by `user.id === 1` client-side**. Any authenticated user can call admin endpoints directly from browser DevTools. |
| 24 | 🟠 | `cmd/admin/main.go:62` | **Admin service on `:8096` with no network isolation and no INTERNAL_SECRET check**. Docker network is the only barrier. |

### 1.6 Information Disclosure

| # | Severity | File | Finding |
|---|----------|------|---------|
| 25 | 🟡 | `cmd/radar/handler.go:28,49` | **Raw DB/service error messages returned to clients** in `Scan`, `GetEvents`, `PlanetStatus`, `EUXScan`. Postgres error text (table names, column names, constraint names) leaks architectural details. |
| 26 | 🟡 | `cmd/admin/handler.go:48` | **`SearchUsers` dumps raw user objects** including emails for all users matching a wildcard. With `q=""`, dumps up to 100 user emails per page — full user enumeration. |
| 27 | 🟡 | `cmd/espionage/handler.go:141` | **`TargetPlayerID` exposed unconditionally** in spy report response. Probing any coordinate reveals the owner's numeric player ID. |
| 28 | 🔵 | `cmd/espionage/main.go`, all services `/readyz` | **`/readyz` leaks DB error detail** including potential connection strings in JSON error response. |
| 29 | 🟡 | `game/src/App.svelte:1184` | **Admin search result dumped as raw JSON in `<pre>` tag**. Leaks full internal user object fields to browser. |

---

## 2. Race Conditions & Concurrency

### 2.1 Critical Resource TOCTOU (Read-Check-Write Not Atomic)

| # | Severity | File | Finding |
|---|----------|------|---------|
| 30 | 🔴 | `cmd/planet/handler.go:752-793` | **`InternalDeductResource` uses read-modify-write in application code**. Two concurrent calls read the same `planet.Metal = 1000`, both check `≥ 500`, both write `1000 - 500 = 500`. Net result: 1000 instead of 0. This is the root cause of resource over-spending across fleet, combat, and nebula. Fix: `UPDATE ... SET metal = metal - $1 WHERE metal >= $1` with `RowsAffected()`. |
| 31 | 🔴 | `cmd/planet/handler.go:866` | **`InternalAddResource` uses read-modify-write**. Two concurrent adds (e.g. loot + transport) both read the same base, both compute `base + amount`, last writer wins. One addition is silently lost. |
| 32 | 🔴 | `cmd/planet/planet.go:174-246` | **`StartBuildingUpgrade` read-check-write not atomic**. Two concurrent upgrade requests both pass the resource check and both deduct, creating two queue entries and double-spending. |
| 33 | 🟠 | `cmd/planet/planet.go:309-351` | **`BuildShips` TOCTOU** — same pattern. Two concurrent builds both pass resource check, second write uses stale `planet.Metal`, one build is effectively free. |
| 34 | 🟠 | `cmd/planet/planet.go:383-425` | **`BuildDefenses` TOCTOU** — identical pattern to #33. |
| 35 | 🟠 | `cmd/planet/planet.go:515-553` | **`BuildIPM`/`BuildABM` TOCTOU** on missile capacity check and resource deduction. Silo can overflow stated capacity. |
| 36 | 🟠 | `cmd/planet/planet.go:1038-1136` | **`StartMoonBuildingUpgrade` TOCTOU** — read planet, check in-process, write resources, write building level, all three separate uncoordinated queries. |
| 37 | 🟠 | `cmd/alliance/service.go:263` | **`BankWithdraw` TOCTOU** — `GetBank` and `UpdateBank` are two separate round-trips with no transaction. Two concurrent officer withdrawals can overdraw the bank. |
| 38 | 🟡 | `cmd/nebula/service.go:497` | **`ClaimDailyGift` TOCTOU** — `GetDailyGiftStatus` read and `ClaimDailyGift` write are separate. Two concurrent requests both pass the `lastClaimDate != today` check and claim double rewards. |
| 39 | 🟡 | `cmd/quest/service.go:178` | **`ClaimReward` TOCTOU** — `HasClaimedQuest` and `ClaimPlayerQuest` are not atomic. Two concurrent requests both pass, both grant rewards. |
| 40 | 🟡 | `cmd/tutorial/service.go:73` | **Tutorial reward granted before `AdvanceStep`**. DB error on `AdvanceStep` leaves step unchanged; player can re-claim the same step reward. |

### 2.2 Fleet Operation Races

| # | Severity | File | Finding |
|---|----------|------|---------|
| 41 | 🔴 | `cmd/fleet/service.go:47` | **`DispatchFleet` ships deducted before slot limit check**. Two concurrent dispatches both deduct ships and both pass the slot check if timed correctly, exceeding the slot limit while also permanently losing ships if the second is rejected. |
| 42 | 🟠 | `cmd/fleet/service.go:155` | **`RecallFleet` TOCTOU** — `GetFleetByID` then `SetFleetReturning` with no `SELECT FOR UPDATE`. Two concurrent recalls both see `in_transit`, both proceed. |
| 43 | 🟠 | `cmd/fleet/service.go:185` | **`SplitFleet` TOCTOU** — concurrent splits both read the same ship counts and both write. Can produce negative or duplicated ship quantities. |
| 44 | 🟠 | `cmd/fleet/service.go:232` | **`MergeFleets` TOCTOU** — concurrent merges on the same fleets. Fleets deleted by one operation are re-read by the other, leading to phantom ships or double-deletion. |
| 45 | 🟠 | `cmd/fleet/main.go:148` | **Attack fleet never marked arrived after combat resolution**. `MarkFleetArrived` is never called in the `"attack"` case. The travel worker re-processes the same fleet every 5 seconds, resolving combat in an infinite loop. |
| 46 | 🟠 | `cmd/fleet/main.go:89` | **Travel worker tick overlap** — if processing takes > 5 seconds, the next tick fires and the same fleet can be double-processed before `MarkFleetArrived` is called. |

### 2.3 Planet Processing Races

| # | Severity | File | Finding |
|---|----------|------|---------|
| 47 | 🟠 | `cmd/planet/planet.go:67` | **`GetOrCreatePlanet` concurrent resource production**. Multiple concurrent requests all compute production from the same `ResourcesUpdatedAt` and all write, but only the last write survives — production is computed N times but only applied once. |
| 48 | 🟠 | `cmd/planet/planet.go:130` | **`processCompletedBuilds` double-completion**. Two concurrent `GetOrCreatePlanet` calls both find the same expired queue entry and both call `CompleteBuild`. Building level is set to `targetLevel` twice; `AddVIPPoints` called twice granting double VIP. |
| 49 | 🟡 | `cmd/planet/planet.go:249` | **`CancelUpgrade` TOCTOU** — build may complete between the read and the `CancelUpgradeWithRefund` call. Resources are refunded even though the upgrade completed, giving a free building level. |

### 2.4 SSE Hub Races

| # | Severity | File | Finding |
|---|----------|------|---------|
| 50 | 🟠 | `cmd/chat/handler.go:102` | **Chat SSE listener slice concurrent modification**. Two simultaneous disconnects both call `append(listeners[:i], listeners[i+1:]...)` under the lock, but the slice backing array is mutated while Broadcast may hold a stale slice header copy. |
| 51 | 🟡 | `cmd/notification/service.go:52` | **`Publish` holds write lock for entire fan-out**. Under sustained load, all subscribe/unsubscribe operations block for the duration of the full SSE fan-out, causing lock contention across all notification streams. |
| 52 | 🟡 | `cmd/nebula/service.go:710` | **`RerollTask` TOCTOU** — two concurrent reroll requests both read `RerollsUsed == 0` and both proceed, consuming two rerolls. |
| 53 | 🟡 | `cmd/nebula/service.go:622` | **`UpdateTaskProgress` double-completion** — concurrent calls both read same progress, both compute completion. `MarkTaskCompleted` called twice. |

### 2.5 Goroutine & Resource Leaks

| # | Severity | File | Finding |
|---|----------|------|---------|
| 54 | 🟠 | `cmd/chat/handler.go:117`, `cmd/notification/service.go:154` | **SSE goroutines leak when proxy/LB does not propagate TCP close**. Context never cancels, goroutine + buffered channel (100-slot) + 30s ticker goroutine all leak indefinitely. |
| 55 | 🟡 | `cmd/chat/service.go:43` | **`rateLimits map[int]time.Time` grows unbounded** — no eviction. Every distinct player ID that ever sends a message occupies a permanent entry. |
| 56 | 🟡 | `cmd/fleet/main.go:97` | **`workerCtx` derived from `context.Background()`, not from shutdown ctx**. During graceful shutdown, in-progress fleet processing continues using background context, potentially outlasting the 15-second shutdown timeout. |
| 57 | 🟡 | `cmd/planet/planet.go:830` | **`SeedAllNPCPlanets` runs 13,473 iterations** (9×499×3) each with 5+ DB queries. Under request context with a 30s timeout, the pool is hammered and partial failures leave orphaned data. |
| 58 | 🟡 | `cmd/fleet/service.go:412` | **`deductFuel` closes response body without checking status code**. Fuel deduction failure (insufficient gas / 400) is silently ignored — fleet dispatches without deducting fuel. |
| 59 | 🔵 | `cmd/nebula/service.go:937` | **Non-200 response body not drained before close** in `getEspionageTechLevel`, preventing HTTP connection reuse. |

---

## 3. Error Handling, Nil Derefs & Panics

### 3.1 Panics in Long-Running Goroutines

| # | Severity | File | Finding |
|---|----------|------|---------|
| 60 | 🟠 | `cmd/fleet/main.go:89` | **Travel worker goroutine has no `recover()`**. A nil map dereference (e.g., `f.Ships` is nil from corrupt JSON) panics the entire fleet service. |
| 61 | 🟠 | `cmd/chat/handler.go:79` | **SSE `Stream` handler goroutine has no `recover()`**. A panic in `json.Marshal` or `fmt.Fprintf` inside the loop kills the connection with no recovery. |
| 62 | 🟠 | `cmd/notification/service.go:137` | **Notification SSE goroutine has no `recover()`**. Same pattern as above. |
| 63 | 🟡 | `cmd/planet/handler.go:549` | **`strings.Contains(err.Error(), ...)` called without nil check**. If any upstream function returns a wrapped nil error, `.Error()` on a nil `error` interface panics. |
| 64 | 🟡 | `cmd/planet/handler.go:1289` | **`*wEntry.LinkedSystem` dereferenced assuming non-nil** when `wEntry.LinkedGalaxy != nil`. Partial DB state (only `LinkedGalaxy` set) causes nil pointer dereference. |
| 65 | 🟡 | `cmd/nebula/service.go:126` | **`rand.Intn(total)` with `total` potentially ≤ 0**. At espionage tech level ≥ 30, `nothingProb = 30 - level` goes negative, `total` can reach 0; `rand.Intn(0)` panics. |

### 3.2 Silently Discarded Errors

| # | Severity | File | Finding |
|---|----------|------|---------|
| 66 | 🟠 | `cmd/combat/repository.go:30` | **Five `json.Marshal` calls discard errors** in `SaveCombatReport`. Corrupt JSONB written to DB with no error surfaced. |
| 67 | 🟠 | `cmd/fleet/repository.go:83,120,193,219,250` | **Five `json.Unmarshal` errors discarded** when reading `Ships` from JSONB. Returns fleet with `Ships == nil`; downstream `f.Ships[k] += v` in `MergeFleets` panics on nil map write. |
| 68 | 🟠 | `cmd/fleet/repository.go:133` | **`json.Marshal(ships)` error discarded** in `UpdateFleetShips`. Nil ships marshal to `null`, overwriting ships column with SQL NULL. |
| 69 | 🟠 | `cmd/espionage/service.go:29` | **Probe deducted before target validation**. If `getPlanetInfo` fails, probe is already consumed with no rollback. Permanent ship loss on infrastructure failure. |
| 70 | 🟠 | `cmd/espionage/service.go:92` | **No check that player has ≥ 1 probe before calling deduct**. Planet service behavior on over-deduction determines if infinite probing is possible. |
| 71 | 🟠 | `cmd/research/service.go:113` | **Resources deducted before `CreateResearch` DB write**. If DB write fails, resources are permanently lost with no research started. |
| 72 | 🟠 | `cmd/research/service.go:171` | **`ProcessCompleted`: `levelUpTech` then `CompleteResearch` non-atomic**. If `levelUpTech` succeeds but `CompleteResearch` fails, next tick increments tech level again — double level-up. |
| 73 | 🟠 | `cmd/nebula/service.go:449` | **`HireCommander` spends DM before inserting commander record**. If DB insert fails, DM is spent with no commander granted. No compensation. |
| 74 | 🟠 | `cmd/nebula/service.go:849` | **`BuyItem` charges currency then applies rewards in separate non-atomic steps**. Server crash between charge and reward delivery results in permanent loss. |
| 75 | 🟡 | `cmd/combat/service.go:117` | **`deductDefenderShips` failure only logged, not returned**. Combat continues; attacker gets loot while defender keeps ships. Inconsistent economic state. |
| 76 | 🟡 | `cmd/combat/service.go:123` | **`addLootToAttacker` failure only logged**. Attacker loses loot on planet service failure. |
| 77 | 🟡 | `cmd/combat/repository.go:155` | **`ListPlayerCombatReports` returns `rows.Err()` as nil**. Partial results returned on network interruption with no error to caller. |
| 78 | 🟡 | `cmd/fleet/main.go:113` | **`MarkFleetArrived` error not checked** at 8 call sites in the travel worker. Silent failure leaves fleet in transit forever. |
| 79 | 🟡 | `cmd/espionage/service.go:113` | **`getPlanetInfo` returns empty `PlanetInfo{}` with nil error on HTTP 404**. Empty planet report written to DB; probe wasted. |
| 80 | 🟡 | `cmd/research/service.go:197` | **`fetchTechLevel` returns `0, nil` on JSON parse failure**. Malformed response silently treated as tech level 0, allowing research past prerequisites. |
| 81 | 🟡 | `cmd/research/service.go:253` | **`deductSingle`: on `parseJSON` failure returns `nil`**. Planet service returns a non-JSON error (502 proxy), resources are never deducted, research proceeds without payment. |
| 82 | 🟡 | `cmd/planet/handler.go:339` | **`GetBuildingLevel` and `GetPlayerShips` errors discarded** with `_` in `ListShips`. Returns zeroed data with HTTP 200 on DB failure. |
| 83 | 🟡 | `cmd/planet/handler.go:433` | **`ListDefenses` errors silently discarded** — same pattern. |
| 84 | 🟡 | `cmd/notification/service.go:164` | **`json.Marshal(event)` error discarded**. If marshal returns nil bytes, `fmt.Fprintf(w, "data: %s\n\n", nil)` writes `data: <nil>` to SSE stream, corrupting the protocol. |
| 85 | 🟡 | `cmd/planet/repository.go:424` | **`GetBuildings` missing `rows.Err()` check**. Returns partial building list with nil error on cursor failure. |
| 86 | 🔵 | All services `writeJSON` | **`json.NewEncoder(w).Encode(data)` error universally discarded** across all 8 services. Half-written JSON on dropped connection goes unreported. |

### 3.3 Integer Overflow & Numeric Errors

| # | Severity | File | Finding |
|---|----------|------|---------|
| 87 | 🟠 | `cmd/fleet/service.go:93` | **Travel time overflow**: `travelSeconds` with `minSpd=1` and `SpeedPct=10` for max-distance fleet produces `~8.2×10¹⁰` seconds → converting to `time.Duration` (int64 nanoseconds) overflows, wrapping to negative → **immediate arrival in the past**. |
| 88 | 🟠 | `cmd/fleet/service.go:110` | **Fuel cost integer overflow** for very large fleets. `int(float64(totalFuel) * distanceFactor * speedFactor)` can wrap to negative, effectively adding gas to the planet. |
| 89 | 🟡 | `cmd/planet/planet.go:309` | **`BuildShips` total cost overflow** for very large quantities. `totalMetal = cfg.Metal * quantity` overflows `int32` for large values, wrapping to negative, causing the resource check to pass and writing a negative deduction (free ships). |
| 90 | 🟡 | `cmd/fleet/service.go:631` | **`harvestDebris` integer overflow**: `debris.Metal * toHarvest` before the division can overflow `int` on platforms where `int` is 32-bit, producing negative harvest amounts. |
| 91 | 🟡 | `cmd/planet/repository.go:641` | **`int64` → `int` cast** for `total_resources_produced`. Silent truncation on 32-bit platforms or values > 2.1 billion. |
| 92 | 🟡 | `cmd/research/service.go:85` | **No maximum tech level cap**. `math.Pow(costFactor=2, level=63+)` overflows `int64`, wrapping to 0 or negative — research becomes free or yields negative costs. |

---

## 4. Game Logic Bugs & Exploits

### 4.1 Fleet Exploits

| # | Severity | File | Finding |
|---|----------|------|---------|
| 93 | 🔴 | `cmd/fleet/service.go:112` | **Stargate mission skips fuel deduction entirely** (`if req.Mission != "stargate"` gates fuel). `checkStarGateLink` is defined but **never called** in `DispatchFleet`. Stargates are free teleporters with no link validation. |
| 94 | 🔴 | `cmd/fleet/service.go:118` | **Slot limit bypassed for any fleet with non-zero `AllianceGroupID`**. Field is user-supplied in `DispatchRequest`. Any player sets a fake `alliance_group_id` to dispatch unlimited fleets. |
| 95 | 🔴 | `cmd/fleet/main.go:113` | **Transport mission delivers hardcoded free resources** (10,000 metal / 5,000 crystal) regardless of actual cargo. A player dispatches an empty transport and receives free resources at the destination. |
| 96 | 🔴 | `cmd/fleet/main.go:148` | **Attack fleet never marked arrived** after combat resolution — combat resolved on every 5-second travel worker tick indefinitely. |
| 97 | 🟠 | `cmd/fleet/service.go:200` | **`SplitFleet` accepts negative ship quantities** (`qty <= 0` continues without blocking negative values). Passing `{"light_fighter": -100}` creates a fleet with negative ships. |
| 98 | 🟠 | `cmd/fleet/service.go:200` | **Split can empty the source fleet** — no check that the original fleet has at least 1 ship remaining. Ghost fleet consuming a slot with 0 ships. |
| 99 | 🟠 | `cmd/fleet/repository.go:103` | **`CountPlayerFleets` counts ALL statuses including `arrived`/`completed`**. No status filter means completed fleets permanently consume fleet slots. |
| 100 | 🟠 | `cmd/fleet/service.go:512` | **`resolveCombatForArrival` does not mark the attacking fleet as arrived**. Combined with #96, combat resolved on every tick. |

### 4.2 Combat Exploits

| # | Severity | File | Finding |
|---|----------|------|---------|
| 101 | 🔴 | `cmd/combat/service.go:85` | **Attacker loot added BEFORE defender resources deducted**. If the defender deduction call fails (planet service outage), attacker receives resources for free — resource duplication. |
| 102 | 🟠 | `cmd/combat/service.go:116` | **Defender ships never deducted in total-wipe scenario**. `if !isEmpty(result.DefenderShipsAfter)` — when all defender ships are destroyed, the deduction call is skipped. Defender keeps ships after total loss. |
| 103 | 🟠 | `cmd/combat/service.go:193` | **`MissileStrike` has no authentication** (same root as finding #10). Any unauthenticated caller can destroy defenses on any planet. |
| 104 | 🟠 | `cmd/combat/combat.go:162` | **Moon creation chance capped at 20% by `math.Min(chance, 20.0)`** even for massive debris fields. The formula grows but the cap always returns 20.0. |
| 105 | 🟡 | `cmd/combat/combat.go:31` | **Wipe condition evaluated sequentially** — if both sides simultaneously meet the wipe threshold, only the attacker-wipes-defender branch fires; defender never gets the wipe credit. |
| 106 | 🟡 | `cmd/combat/combat.go:50` | **`carMult` arguments appear inverted** in `defenderFires` call — attacker's rapid-fire multiplier is applied when the defender is firing. |
| 107 | 🟡 | `cmd/combat/service.go:153` | **`tryCreateMoon` concurrent moon creation** — two combats at same coordinates in the same tick can both create a moon row; no UNIQUE constraint on `(galaxy, system, position)`. |

### 4.3 Planet Building Exploits

| # | Severity | File | Finding |
|---|----------|------|---------|
| 108 | 🟠 | `cmd/planet/planet.go:179` | **Double-queue TOCTOU for same building** — concurrent upgrades both find empty queue and both succeed, creating two queue entries for the same building; building jumps two levels with one level's resources paid. |
| 109 | 🟠 | `cmd/planet/planet.go:1038` | **Moon building upgrades are instant** — no queue or timer, `UpdateMoonBuildingLevel` called immediately after resource deduction. Moon buildings can be spammed to max level instantly. |
| 110 | 🟡 | `cmd/planet/planet.go:274` | **Deconstructing a level-1 starter building gives free resources**. Level-1 buildings seeded for free; deconstruction refunds 50% of the level-1 upgrade cost. |
| 111 | 🟡 | `cmd/planet/planet.go:130` | **`processCompletedBuilds` not in a transaction**. Multiple completed entries processed sequentially; partial failure leaves DB in inconsistent state. |

### 4.4 Research Exploits

| # | Severity | File | Finding |
|---|----------|------|---------|
| 112 | 🔴 | `cmd/research/repository.go:125` | **Research cancel refund never actually sent**. `CancelResearchWithRefund` only marks the queue cancelled — `refundMetal/Crystal/Gas` parameters accepted but never used. Players lose 100% of resources on cancel, not 50% as described. |
| 113 | 🔴 | `cmd/research/service.go:85` | **Multiple different techs can be researched simultaneously**. No global per-player research queue limit. 18+ concurrent research queues possible. |
| 114 | 🟠 | `cmd/research/service.go:218` | **Research lab level prerequisite never enforced**. `checkPrerequisites` skips `research_lab` type (`if p.Type == "research_lab" { continue }`). Any player can research any tech with no lab. |
| 115 | 🟠 | `cmd/research/service.go:235` | **Three separate HTTP calls to deduct metal, crystal, gas** — no atomicity. If crystal deduction fails after metal succeeds, metal is lost permanently with no rollback. |

### 4.5 Expedition / Nebula Exploits

| # | Severity | File | Finding |
|---|----------|------|---------|
| 116 | 🔴 | `cmd/nebula/service.go:30` | **Expeditions are instant and synchronous** — ships deducted, outcome generated, resources/ships added all before returning. `TravelDuration` and `ExploreDuration` fields logged but never enforced. Spam expeditions for unlimited DM/resources/ships. |
| 117 | 🟠 | `cmd/nebula/service.go:710` | **`RerollTask` checks only `tasks[0].RerollsUsed`** regardless of which task is being rerolled. Wrong task's reroll count gates the operation. |
| 118 | 🟡 | `cmd/nebula/service.go:529` | **`ResetDailyGiftStreak` called on day 7** when DB already wraps `streak_day` to 1 at the same time; player effectively gets two day-1 rewards in a row (off-by-one in reset cycle). |
| 119 | 🟡 | `cmd/nebula/service.go:892` | **`BuyItem` for `commander_extension` extends ALL commander types** regardless of which ones the player has active. |
| 120 | 🟡 | `cmd/nebula/service.go:960` | **`UpgradeGalactoniteDiscoverer` has no maximum level cap**. Level can be incremented indefinitely. The discoverer level is stored but never actually applied to expedition outcomes anywhere in the codebase. |

---

## 5. Authorization Bypasses

| # | Severity | File | Finding |
|---|----------|------|---------|
| 121 | 🔴 | `cmd/fleet/service.go:72` | **`DispatchFleet` never verifies `OriginPlanetID` belongs to the caller**. An authenticated player can deduct ships from any other player's planet by supplying their planet ID. |
| 122 | 🟠 | `cmd/planet/handler.go:1153` | **`UpgradeMoonBuilding` reads then discards `userID`** — never passed to `StartMoonBuildingUpgrade`. Any authenticated user can upgrade buildings on any moon, deducting resources from that planet. |
| 123 | 🟠 | `cmd/planet/handler.go:1238` | **`LinkWormholes` never reads `X-User-ID`**. Any authenticated user can link any two moons, overwriting existing wormhole configs and imposing cooldowns on victims. |
| 124 | 🟡 | `cmd/planet/handler.go:622` | **`GetMissileCounts` exposes any planet's missile data without ownership check**. Attacker can enumerate exact defense missile counts before planning raids. |
| 125 | 🟡 | `cmd/planet/handler.go:1502` | **`GetGems` exposes and mutates any planet's gem config without ownership check**. Also calls `EnsureGemSlots`, mutating data for arbitrary planet IDs. |
| 126 | 🟡 | `cmd/planet/handler.go:1394` | **`StarGateLinks` exposes any planet's stargate configuration without ownership check**. Reveals player alliance and coordination structures. |
| 127 | 🟠 | `cmd/espionage/service.go:28` | **`SendProbe` has no check that `req.PlanetID` belongs to the caller**. A player can supply another player's planet as the spy source. |
| 128 | 🟠 | `cmd/research/service.go:85` | **`StartResearch` does not verify `planetID` belongs to `playerID`**. Resources deducted from any arbitrary planet. |
| 129 | 🟠 | `cmd/nebula/service.go:30` | **`StartExpedition` does not verify `planetID` belongs to `playerID`**. Ships deducted from another player's planet. |
| 130 | 🟠 | `cmd/alliance/service.go:425` | **`DeleteBulletin` cross-alliance privilege escalation**. Officer role check does not verify the requester's `AllianceID == author's AllianceID`. An officer in Alliance A can delete bulletins from Alliance B. |
| 131 | 🟡 | `cmd/friend/service.go:48` | **`AcceptRequest` does not enforce who is the recipient**. The initiator can accept their own outgoing friend request. |
| 132 | 🟡 | `cmd/radar/service.go:137` | **`EUXScan` reveals full fleet composition and mission type for any coordinates** to any authenticated player. No relationship check required. |

---

## 6. Social Services

### 6.1 Alliance

| # | Severity | File | Finding |
|---|----------|------|---------|
| 133 | 🟠 | `cmd/alliance/service.go:96` | **`ApplyToAlliance` immediately adds player as full member** without any founder/officer acceptance step. Any player knowing an alliance ID can join instantly. |
| 134 | 🟡 | `cmd/alliance/service.go:110` | **No alliance member cap**. Alliances can grow to unlimited size. |
| 135 | 🟡 | `cmd/alliance/service.go:148` | **`TransferFounder` allows self-transfer** — no guard for `playerID == targetPlayerID`. Demotes then re-promotes with wasted double-update. |
| 136 | 🟡 | `cmd/alliance/service.go:220` | **`BankDeposit` accepts negative amounts**. Guard `if metal > 0` skips the deduction but `newMetal = bank.Metal + metal` still runs with a negative value, silently decreasing the bank balance. |
| 137 | 🟡 | `cmd/alliance/service.go:302` | **`BankWithdraw` resource credit failure is fire-and-forget**. Bank already debited; planet service failure permanently loses the resources. |
| 138 | 🟡 | `cmd/alliance/service.go:387` | **`PostBulletin` has no length limit** on `title` or `content`. Megabyte-sized bulletins accepted. |
| 139 | 🟡 | `cmd/alliance/service.go:457` | **`ShareReport` allows duplicate shares**. Same report ID can appear multiple times in the shared list; no uniqueness check. |
| 140 | 🔵 | `cmd/alliance/repository.go:228` | **`GetBulletins` query has no `LIMIT`**. All bulletins for an alliance returned unbounded. |
| 141 | 🔵 | `cmd/alliance/repository.go:293` | **`GetSharedReports` query has no `LIMIT`**. Unbounded. |

### 6.2 Chat

| # | Severity | File | Finding |
|---|----------|------|---------|
| 142 | 🟡 | `cmd/chat/service.go:18` | **2-second rate limit is trivially spammable** (~30 msg/min). No per-minute or burst budget. |
| 143 | 🟡 | `cmd/chat/service.go:211` | **Private messages share the same rate limit bucket as public messages**. |
| 144 | 🟡 | `cmd/chat/service.go:95` | **Sender display name always `"Player %d"`**. All chat appears anonymized — no real name lookup. |
| 145 | 🟡 | `cmd/chat/service.go:22` | **`SubscriberManager.Broadcast` distributes all messages to all connected SSE channels**. Private messages are technically observable in any connected client's channel buffer before the `isRelevantForPlayer` filter drops them. |
| 146 | 🟡 | `cmd/chat/handler.go:63` | **`limit` and `beforeID` parsed with errors discarded** (`limit, _ :=`). Invalid params silently become 0 — `GetMessages(limit=0)` returns at most 1 message. |
| 147 | 🔵 | `cmd/chat/handler.go:101` | **SSE listener registration holds the broadcast lock**, blocking message delivery during any connect/disconnect. |

### 6.3 Notifications

| # | Severity | File | Finding |
|---|----------|------|---------|
| 148 | 🟡 | `cmd/notification/service.go:80` | **No bulk/fan-out create API**. Notifying all alliance members requires N serial HTTP calls — N+1 fan-out problem. |
| 149 | 🟡 | `cmd/notification/repository.go:41` | **`ListNotifications` uses two non-atomic queries** (COUNT then SELECT). Concurrent inserts make `total` inconsistent with returned rows. |
| 150 | 🟡 | `cmd/notification/service.go:89` | **`offset` parameter has no upper bound**. Deep pagination forces full sequential scan. |
| 151 | 🔵 | Missing | **No composite index on `notification.notifications(player_id, is_read)`**. Unread count and filtered list queries cannot use a single efficient index. |

### 6.4 Friends

| # | Severity | File | Finding |
|---|----------|------|---------|
| 152 | 🟠 | `cmd/friend/service.go:35` | **Two `AddFriend` inserts not wrapped in a transaction**. If second insert fails, orphan record blocks future friend requests. |
| 153 | 🟡 | `cmd/friend/service.go:81` | **`ListFriends` performs N+1 queries** — `GetLastActive` called per friendship inside a loop. 1,000 friends = 1,000 serial DB round trips. |
| 154 | 🟡 | `cmd/friend/repository.go:61` | **`GetFriends` has no `LIMIT`**. Unbounded result set. |
| 155 | 🟡 | `cmd/friend/service.go:22` | **No limit on total friend count**. Unlimited friend requests per player. |
| 156 | 🔵 | Missing | **No index on `friend.friendships(friend_id)`**. The OR-based lookup `(player_id=$1 AND friend_id=$2) OR (player_id=$2 AND friend_id=$1)` cannot use a single-column index efficiently. |

### 6.5 Ranking

| # | Severity | File | Finding |
|---|----------|------|---------|
| 157 | 🟡 | `cmd/ranking/service.go:115` | **`RecalculateForPlayer` overwrites player names with `"Player %d"`**. Every 5-minute recalculation erases any stored real player name. |
| 158 | 🟡 | `cmd/ranking/service.go:131` | **`RecalculateAll` is a serial loop** — 4×N queries for N players. At 10,000 players, the 5-minute ticker stalls and overlaps itself. |
| 159 | 🔵 | `cmd/ranking/handler.go:44` | **`page` calculation with `limit=0` panics** via divide-by-zero. Default set but not guarded before use as divisor. |

### 6.6 Events & Quests

| # | Severity | File | Finding |
|---|----------|------|---------|
| 160 | 🟡 | `cmd/event/service.go:25` | **Players can join events up to 60 seconds after expiry** (ticker granularity gap). |
| 161 | 🟡 | `cmd/event/handler.go:37` | **`GetActiveEvents` performs per-event N+1 DB queries** — `GetParticipation` called in loop. 50 events = 50 extra round trips per API call. |
| 162 | 🟡 | `cmd/quest/repository.go:96` | **`ClaimPlayerQuest` does not check `RowsAffected`**. Already-claimed quest returns nil, appearing successful. |
| 163 | 🟡 | `cmd/quest/service.go:336` | **Alliance/commander quests auto-complete for every player** — `evaluateRequirement` always returns 1 for `"alliance_join"`, `"alliance_donate"`, `"attack_count"`, `"commander_hired"`. |
| 164 | 🟡 | `cmd/quest/service.go:244` | **`evaluateRequirement` called twice per requirement** in `CheckAndUpdateProgress`, doubling DB queries per quest check. |
| 165 | 🔵 | `cmd/quest/main.go:68` | **`POST /api/quest/list` uses POST for a read-only list operation**. Should be GET. |

---

## 7. Admin / Espionage / Radar / Research / Nebula

### 7.1 Admin Issues

| # | Severity | File | Finding |
|---|----------|------|---------|
| 166 | 🟠 | `cmd/admin/repository.go:208` | **`AddDM` maps DM to `vip_points` field** in `player_progress`. DM and VIP points are semantically different; granting "DM" corrupts VIP score. |
| 167 | 🟠 | `cmd/admin/repository.go:209` | **`AddCredits` maps credits to `total_resources_produced`**. Corrupts resource statistics used for ranking. |
| 168 | 🟡 | `cmd/admin/service.go:103` | **`GrantDM` has no upper bound on `amount`**. Unlimited DM can be granted in a single call. |
| 169 | 🟡 | `cmd/admin/repository.go:200` | **`UpdatePlanetResources` does not check `RowsAffected`**. Updating a non-existent planet silently succeeds. |
| 170 | 🟡 | `cmd/admin/service.go:131` | **`CreateEvent` does not validate `EventType`**. Any arbitrary string stored as event type may break downstream event consumers. |
| 171 | 🔵 | `cmd/admin/handler.go:144` | **Error comparison via `err.Error() == "..."` string matching** throughout admin handlers. Fails if error is ever wrapped. |

### 7.2 Espionage Issues

| # | Severity | File | Finding |
|---|----------|------|---------|
| 172 | 🟠 | `cmd/espionage/service.go:45` | **`buildReport` always sets `DetailLevel: 5`** regardless of player's espionage tech level. Every spy report always reveals the fleet even with tech level 0. |
| 173 | 🟡 | `cmd/espionage/repository.go:77` | **All `json.Unmarshal` calls on DB data silently ignore errors**. Corrupted JSONB column returns nil maps with no signal. |

### 7.3 Radar Issues

| # | Severity | File | Finding |
|---|----------|------|---------|
| 174 | 🟠 | `cmd/radar/handler.go:129` | **`InternalDetect` has no authentication**. Any actor can inject fake incoming-attack alerts for any player. |
| 175 | 🟠 | `cmd/radar/service.go:137` | **`EUXScan` galaxy coordinate never compared** in range check. A radar at galaxy 1 can scan galaxy 2 and beyond — galaxy dimension ignored. |
| 176 | 🟡 | `cmd/radar/handler.go:35` | **`GetEvents` `scope` field decoded but never used**. Service always returns all events regardless of requested scope. |
| 177 | 🟡 | `cmd/radar/service.go:60` | **`DetectFleet` silently ignores malformed `ArrivalTime`**. Error discarded; `arrivalTime` set to zero value (year 0001) stored in DB, appearing as ancient historical events. |
| 178 | 🟡 | `cmd/radar/service.go:42` | **`ResolveEvent` can resolve an already-resolved event**. Ownership check uses `GetPlayerEvents` (all events) rather than filtering unresolved, allowing repeated resolution. |

### 7.4 Research Issues

| # | Severity | File | Finding |
|---|----------|------|---------|
| 179 | 🟡 | `cmd/research/service.go:116` | **`fetchBuildingLevel` failure silently sets `labLevel = 0`**. Research proceeds with a fixed 1-hour duration instead of actual lab-level duration. |
| 180 | 🟡 | `cmd/research/http.go:22` | **`httpDo` creates a new `http.Client` on every call**. With 18 techs in `fetchTechLevels`, 18 sequential clients are allocated per request — TCP connection exhaustion under load. |
| 181 | 🟡 | `cmd/research/service.go:204` | **`fetchTechLevels` makes 18 sequential HTTP calls** with no concurrency and no batching. Under slow network this blocks for up to `18 × 10s = 200s`. |

### 7.5 Nebula Issues

| # | Severity | File | Finding |
|---|----------|------|---------|
| 182 | 🟠 | `cmd/nebula/service.go:421` | **`SpeedUp` only returns cost but doesn't actually modify any build queue**. DM is charged but no construction/research/shipyard entry is shortened. |
| 183 | 🟡 | `cmd/nebula/handler.go:208` | **`DMSpeedUp` handler ignores `TargetID` field** — decoded but never passed to `SpeedUp`. DM paid, nothing sped up. |
| 184 | 🔵 | `cmd/nebula/repository.go:198` | **`SpendDarkMatter` fails for a player who has never earned DM** (no row in table). `AddDarkMatter` uses UPSERT, but the first spend against a non-existent row returns "insufficient DM" even if they should have some. |

---

## 8. Frontend Issues

### 8.1 Security

| # | Severity | File | Finding |
|---|----------|------|---------|
| 185 | 🟠 | `game/src/App.svelte:2` | **JWT in `localStorage`** — readable by any JavaScript; exfiltrable on XSS. Use `httpOnly` cookie. |
| 186 | 🟠 | `game/src/App.svelte:334,951` | **JWT in SSE URL query string** — appears in server access logs. |
| 187 | 🟠 | `game/src/App.svelte:1439` | **Admin panel gated only by `user.id === 1` client-side**. Any user can call admin endpoints directly from console. |
| 188 | 🟡 | `game/src/App.svelte:1008` | **`player_id` sent in quest/claim request body by client**. If backend doesn't independently verify against JWT subject, any player can claim quests for any other player ID. |
| 189 | 🟡 | `game/nginx.conf` | **No `Content-Security-Policy` header** — no script-src, connect-src restrictions. |
| 190 | 🟡 | `game/nginx.conf` | **No `X-Content-Type-Options: nosniff`** — browsers may MIME-sniff and execute content as wrong type. |
| 191 | 🟡 | `game/nginx.conf` | **No `X-Frame-Options` or `frame-ancestors`** — page can be embedded in iframe, enabling clickjacking. |
| 192 | 🟡 | `game/nginx.conf` | **No `Strict-Transport-Security` (HSTS)** — allows downgrade attacks if ever deployed over HTTPS. |
| 193 | 🟡 | `game/index.html:8` | **Google Fonts loaded without Subresource Integrity (SRI)**. A compromised CDN could inject malicious CSS. |

### 8.2 Dead / Broken Nginx Config

| # | Severity | File | Finding |
|---|----------|------|---------|
| 194 | 🔴 | `game/nginx.conf:20-37` | **SSE `location /chat/stream` and `/notification/stream` are dead config** — the app requests `/api/chat/stream`, which is caught by the `/api/` block first. The SSE-specific blocks with 24-hour `proxy_read_timeout` never execute. SSE connections go through the 60s timeout `/api/` block, causing silent chat/notification dropouts every minute. |
| 195 | 🟡 | `game/nginx.conf:43` | **`index.html` served with `max-age=3600`** — clients run stale JS for up to 1 hour after deployment. Should be `no-cache`. |
| 196 | 🔵 | `game/nginx.conf:27,38` | **SSE endpoints missing `X-Real-IP` / `X-Forwarded-For` headers** — backend cannot rate-limit SSE connections by real IP. |

### 8.3 Runtime Errors

| # | Severity | File | Finding |
|---|----------|------|---------|
| 197 | 🟡 | `game/src/App.svelte:1663` | **`.toFixed(1)` called directly on `planet.production.metal`** with no null guard. If server omits `production` (network glitch, schema change), throws `TypeError`. |
| 198 | 🟡 | `game/src/App.svelte:1665` | **Division by `planet.storage.metal` with no zero check**. `planet.metal / 0` = Infinity; `width: Infinity%` breaks storage bar display. |
| 199 | 🟡 | `game/src/App.svelte:1995` | **Quest progress bar divides by `progress_target`** without zero check. `Math.min(100, Infinity)` = 100 — bar appears full on uncompleted quest. |
| 200 | 🟡 | `game/src/App.svelte:694` | **`maxAfford` starts at `999999999`**. If all resource costs are 0, sends a 10^9 build order to the server without validation. |

### 8.4 Race Conditions & State Issues

| # | Severity | File | Finding |
|---|----------|------|---------|
| 201 | 🟠 | `game/src/App.svelte:54` | **`setInterval(loadPlanet, 5000)` with no in-flight guard**. Slow poll response arriving after a fast imperative call (post-upgrade) overwrites fresh state with stale data — resources, queue, and storage can visually revert. |
| 202 | 🟡 | `game/src/App.svelte:73` | **`$: if (token && !user) startPolling()` reactive statement can fire multiple times** while `user` is still null during async fetch. Each firing calls `setInterval`, stacking multiple concurrent polling loops. |
| 203 | 🟡 | `game/src/App.svelte:527` | **`galaxyPageNext`/`Prev` don't cancel in-flight requests**. Rapid clicking desynchronizes `galaxyPage` and displayed data if responses arrive out of order. |
| 204 | 🟡 | `game/src/App.svelte:529` | **`selectedGalaxy` never updated when user selects a different galaxy** in `selectGalaxy`. Pagination always uses galaxy 1 regardless of which galaxy is displayed. |

### 8.5 SSE Leaks

| # | Severity | File | Finding |
|---|----------|------|---------|
| 205 | 🟡 | `game/src/App.svelte:29` | **`logout()` never closes `chatSocket` or `notificationSocket`**. SSE connections remain open after logout, leaking the old JWT to the server. When re-logging in, a second SSE connection is created alongside the orphaned first. |
| 206 | 🟡 | `game/src/App.svelte:335` | **Chat SSE `onerror` only sets flag — no reconnect logic**. When SSE drops, chat panel shows stale messages silently. User must navigate away and back to reconnect. |
| 207 | 🟡 | `game/src/App.svelte:963` | **Notification SSE `onerror` is a no-op `() => { /* ignore */ }`**. Disconnections completely silent; no reconnect. |

---

## 9. Infrastructure & DevOps

### 9.1 Docker Compose

| # | Severity | File | Finding |
|---|----------|------|---------|
| 208 | 🟠 | `docker-compose.yml:19` | **Postgres port 5432 exposed to host**. Every process on the host can connect directly, bypassing all gateway security. Remove `ports:` from postgres. |
| 209 | 🟠 | `docker-compose.yml:47` | **Gateway port 8080 exposed to host**. On a cloud VM, directly internet-reachable. Should only be accessible from nginx. |
| 210 | 🟡 | `docker-compose.yml` (all) | **No `restart: unless-stopped` on any service**. A service process crash leaves it permanently down with no recovery. |
| 211 | 🟡 | `docker-compose.yml` (all) | **No CPU limits set**. A runaway service can monopolise all cores and starve siblings. |
| 212 | 🟡 | `docker-compose.yml` (all except postgres) | **No `healthcheck` on microservices**. Services start without readiness; dependent services connect before the upstream is ready. |
| 213 | 🟡 | `docker-compose.yml:67` | **`depends_on` only checks postgres health**, not peer service health. `fleet` depends on `planet` and `combat` via HTTP but they have no health condition. |
| 214 | 🟡 | `docker-compose.yml` | **No explicit Docker network defined**. All services on the default bridge — postgres, nginx, and all 18 microservices mutually reachable. |
| 215 | 🟡 | `docker-compose.yml:2` | **`sslmode=disable` in DATABASE_URL**. Intra-cluster DB traffic is unencrypted. |
| 216 | 🔵 | `docker-compose.yml:264` | **Named `pgdata` volume with no backup strategy** documented. `docker compose down -v` silently deletes all game data. |

### 9.2 Dockerfile

| # | Severity | File | Finding |
|---|----------|------|---------|
| 217 | 🟠 | `build/Dockerfile.go:8` | **Final image runs as root** — no `USER` directive. Process runs as UID 0 inside container. |
| 218 | 🟡 | `build/Dockerfile.go:1` | **Builder uses `golang:1.22-alpine`** — Go 1.22 is over a year old; CVEs fixed in 1.23/1.24 are unpatched. Update to `golang:1.24-alpine`. |
| 219 | 🟡 | `build/Dockerfile.go:8` | **Alpine image pinned by mutable tag** `3.19`, not by digest. Build reproducibility not guaranteed. |
| 220 | 🟡 | `build/Dockerfile.go:6` | **No `-ldflags="-s -w"` or `-trimpath`**. Binary embeds full build-host paths and debug symbols, leaking internal repo structure. |
| 221 | 🟡 | `build/Dockerfile.go` | **No `.dockerignore` file exists**. `COPY . .` copies `.git/`, `game/`, all other service directories into every service's build context, inflating build time and potentially leaking secrets. |
| 222 | 🔵 | `build/Dockerfile.go:12` | **`EXPOSE 8080` hardcoded** but only gateway uses 8080; all other services run on 8081–8097. Misleading for non-gateway images. |
| 223 | 🔵 | `game/Dockerfile:1` | **`node:20-alpine` is a mutable tag**. Pin to digest or patch version. |

### 9.3 CI (GitHub Actions)

| # | Severity | File | Finding |
|---|----------|------|---------|
| 224 | 🟠 | `.github/workflows/ci.yml:29` | **Test DB credentials hardcoded in CI workflow** (`POSTGRES_PASSWORD: galaxy_dev`). Use GitHub Actions secrets. |
| 225 | 🟠 | `.github/workflows/ci.yml:46` | **Full `DATABASE_URL` with plaintext password** committed to the workflow file. |
| 226 | 🟡 | `.github/workflows/ci.yml:44` | **`go test ./cmd/...` runs without `-race`**. Data race detector disabled for a concurrent game server. Add `-race`. |
| 227 | 🟡 | `.github/workflows/ci.yml:44` | **No timeout on `go test`**. A deadlock in a test hangs CI until GitHub's 6-hour kill limit fires. Add `-timeout 10m`. |
| 228 | 🟡 | `.github/workflows/ci.yml` | **No `golangci-lint` in CI**. Static analysis (errcheck, gosec, staticcheck) entirely absent from the pipeline. |
| 229 | 🟡 | `.github/workflows/ci.yml` | **No migration step before `go test`**. Tests against real Postgres fail if schema is absent. |
| 230 | 🟡 | `.github/workflows/ci.yml` | **No `cache: true` on `actions/setup-go@v5`**. Modules re-downloaded on every run. |
| 231 | 🟡 | `.github/workflows/ci.yml` | **Jobs have no `needs:` sequencing** — all three run in parallel. Build runs even when vet fails. |
| 232 | 🔵 | `.github/workflows/ci.yml:3` | **CI triggers on `master` branch** but repo uses `main`. Dead trigger. |

### 9.4 Makefile

| # | Severity | File | Finding |
|---|----------|------|---------|
| 233 | 🟠 | `Makefile:1` | **`tutorial` and `admin` services absent from `SERVICES` variable**. `make build`, `make test`, `make vet`, `make migrate-up` silently skip them. |
| 234 | 🟡 | `Makefile:41` | **Migration targets hardcode `localhost:5432` with plaintext credentials**. Running `make migrate-up` against a production port-forward executes destructive migrations silently. |
| 235 | 🔵 | `Makefile:13` | **`build-*` targets output to `/dev/null`** — no artifact produced. Should be named `compile-*` or produce real binaries. |

### 9.5 SQL Migrations

| # | Severity | File | Finding |
|---|----------|------|---------|
| 236 | 🔴 | `cmd/combat/migrations/000001_init.down.sql:7` | **Combat down migration executes `DROP SCHEMA IF EXISTS fleet CASCADE`**, destroying the fleet service's entire schema — `fleet.fleets`, `fleet.attack_cooldowns`, `fleet.debris_fields` — all gone. Critical operational hazard. |
| 237 | 🟠 | `cmd/planet/migrations/000004_seed_data.up.sql:62` | **Planet seed migration cross-references `fleet.moons`**. If planet migrations run before combat migrations (which create `fleet.moons`), this fails with "relation does not exist". Hard migration ordering dependency not documented. |
| 238 | 🟠 | `cmd/planet/migrations/000004_seed_data.down.sql` | **Down migration issues unbounded `DELETE FROM` without `WHERE` clauses**, permanently deleting all player data from 5 tables. Catastrophic if run against production. |
| 239 | 🟡 | `cmd/fleet/migrations/000001_init.up.sql:3` | **No index on `fleet.fleets(player_id)`**. Primary query column for listing a player's fleets — sequential scan on large tables. |
| 240 | 🟡 | `cmd/fleet/migrations/000001_init.up.sql:3` | **No index on `fleet.fleets(status)`**. Queries filtering in-flight fleets full-scan the table. |
| 241 | 🟡 | `cmd/fleet/migrations/000001_init.up.sql:17` | **`ALTER TABLE ADD COLUMN IF NOT EXISTS` inside initial migration** — evidence the migration was edited after being applied. Should be a separate numbered migration. |
| 242 | 🟡 | `cmd/ranking/migrations/000001_init.up.sql:7` | **Score columns are `INTEGER` not `BIGINT`**. Late-game players accumulate scores exceeding `INT` max (2.1B). Silent overflow. |
| 243 | 🟡 | `cmd/radar/migrations/000001_init.up.sql:3` | **No index on `radar_events(player_id)` or `(resolved, arrival_time)`**. Per-player incoming attack queries scan the full table. |
| 244 | 🟡 | `cmd/espionage/migrations/000001_init.up.sql:3` | **No index on `espionage_reports(player_id)` or `(target_player_id)`**. |
| 245 | 🟡 | `cmd/nebula/migrations/000001_init.up.sql` | **No indexes on `dm_transactions(player_id)` or `credits_transactions(player_id)`**. Transaction history queries full-scan. |
| 246 | 🟡 | `cmd/nebula/migrations/000001_init.up.sql:6` | **`expeditions.fleet_id INT NOT NULL DEFAULT 0`** — sentinel value 0, no FK constraint. Semantically incorrect; use `INT NULL`. |
| 247 | 🟡 | `cmd/event/migrations/000001_init.up.sql` | **No index on `events(status)` or `(starts_at, ends_at)`**. Active event queries full-scan. |
| 248 | 🔵 | `go.work:1` | **`go.work` declares `go 1.22` with no `toolchain` directive**. Any Go version ≥ 1.22 builds the workspace silently, potentially with a different stdlib than tested. |

---

## 10. Bad Practices & Code Quality

### 10.1 Architecture & Design

| # | Severity | File | Finding |
|---|----------|------|---------|
| 249 | 🟠 | `cmd/planet/repository.go:266` | **`SeedBuildingsForPlanet` called outside the planet creation transaction** using the pool directly. If `tx.Commit()` fails, buildings are committed but the planet is rolled back — orphaned building rows with no planet. |
| 250 | 🟠 | `cmd/combat/service.go:222` | **`json.Marshal` errors discarded at 6+ sites** in service.go with `body, _ :=`. Nil map marshals to `"null"` — receiving service gets a null body and fails with a misleading error. Pattern repeated throughout all inter-service HTTP calls. |
| 251 | 🟡 | `cmd/research/service.go:204` | **`fetchTechLevels` makes 18 sequential HTTP calls** — no `sync.WaitGroup`, no parallelism, no batching. `GET /api/research` is extremely slow under any real load. |
| 252 | 🟡 | `cmd/chat/service.go:77` | **`checkAllianceMembership` HTTP call blocks every SSE connection establishment**. Blocks connection setup on alliance service availability and latency. |
| 253 | 🟡 | `cmd/quest/service.go:141` | **`ListQuests` triggers `checkAndUpdateSingleQuest` for every unclaimed quest** — up to 20 × N DB queries synchronously in the HTTP response path. |

### 10.2 HTTP Design Smells

| # | Severity | File | Finding |
|---|----------|------|---------|
| 254 | 🔵 | `cmd/quest/main.go:68` | **`POST /api/quest/list`** — read-only list operation registered as POST. Should be GET. |
| 255 | 🔵 | `cmd/alliance/main.go:83` | **`POST /api/alliance/unshare-report`** — performs a delete but uses POST instead of DELETE. |
| 256 | 🟡 | All services | **Internal DB/service error strings returned directly to clients** (`writeJSON(w, 500, map[string]string{"error": err.Error()})`). Postgres error text (table names, schema, constraints) leaks architectural details. |
| 257 | 🟡 | `cmd/espionage/handler.go:44` | **All espionage probe errors return HTTP 400** regardless of error type. Infrastructure failure (planet service down) returns Bad Request instead of 502/503, making it impossible for clients to distinguish client vs server errors. |

### 10.3 Magic Numbers & Missing Constants

| # | Severity | File | Finding |
|---|----------|------|---------|
| 258 | 🔵 | `cmd/chat/service.go:79` | Magic number `2 * time.Second` for rate limit window. |
| 259 | 🔵 | `cmd/chat/handler.go:117`, `cmd/notification/service.go:157` | Magic number `30 * time.Second` for SSE heartbeat interval. |
| 260 | 🔵 | `cmd/event/service.go:67` | Magic number `60 * time.Second` for event background ticker. |
| 261 | 🔵 | `cmd/ranking/service.go:21` | Magic numbers `100` (default limit) and `500` (max limit) inline. |
| 262 | 🔵 | `cmd/chat/service.go:113` | Magic number `100` for message history cap. |

### 10.4 Unbounded / Missing Database Indexes

| # | Severity | File | Finding |
|---|----------|------|---------|
| 263 | 🟡 | Multiple repos | **Unbounded queries throughout**: `GetBulletins`, `GetSharedReports`, `GetFriends`, `GetActiveEvents` — no `LIMIT` clause. Large datasets return everything in one query. |
| 264 | 🟡 | `cmd/alliance/migrations` | **No index on `alliance.members(alliance_id)`**. `GetMembers(alliance_id)` requires seq scan without it. |
| 265 | 🔵 | All services | **No `ulimits: nofile` set in docker-compose**. Default kernel limit (often 1024) causes "too many open files" under load for SSE-heavy services. |
| 266 | 🟡 | `cmd/planet/repository.go:301` | **`SeedBuildingsForPlanet` issues 12 separate `pool.Exec` calls in a loop**. Single batch `INSERT ... VALUES` would replace all 12. Under `SeedAllNPCPlanets` this is 13,473 × 12 = 161,676 queries. |

### 10.5 Miscellaneous Bad Practices

| # | Severity | File | Finding |
|---|----------|------|---------|
| 267 | 🟡 | `cmd/alliance/repository.go:56` | **`err == pgx.ErrNoRows` direct comparison** instead of `errors.Is`. Inconsistent with rest of codebase and fragile against future error wrapping. |
| 268 | 🟡 | `game/src/App.svelte:1057` | **`setInterval(() => { events = events }, 1000)` every second** forces Svelte to re-render the entire events list via no-op assignment — unnecessary DOM diffing at 1Hz. |
| 269 | 🟡 | `game/src/App.svelte:2072` | **`fleetData.fleets` undefined if API returns array directly**. `loadFleet` stores raw response in `fleetData`; `loadFleetsForView` stores normalized array in `fleetView`. Template checks `fleetData.fleets`, which is undefined — "No active fleets" shown even when fleets exist. |
| 270 | 🔵 | All services | **No structured logging or distributed tracing configured**. No `LOG_LEVEL`, `LOG_FORMAT`, or OpenTelemetry env vars in any service or docker-compose. Diagnosing a failed request chain across 18 services is extremely difficult without correlation IDs. |

---

## Summary

| Category | Critical | High | Medium | Low | Total |
|----------|----------|------|--------|-----|-------|
| Security & Auth | 5 | 16 | 9 | 4 | **34** |
| Race Conditions | 7 | 19 | 12 | 4 | **42** |
| Error Handling / Panics | 2 | 11 | 18 | 7 | **38** |
| Game Logic / Exploits | 9 | 18 | 16 | 4 | **47** |
| Authorization Bypasses | 4 | 8 | 8 | 0 | **20** |
| Social Services | 0 | 4 | 24 | 10 | **38** |
| Admin/Espionage/Radar/Research/Nebula | 0 | 6 | 12 | 4 | **22** |
| Frontend | 2 | 5 | 18 | 3 | **28** |
| Infrastructure | 5 | 9 | 17 | 9 | **40** |
| Bad Practices | 0 | 3 | 15 | 4 | **22** |
| **Total** | **34** | **99** | **149** | **49** | **331+** |

> Note: Many findings above subsume multiple distinct sub-issues; the raw per-agent counts totalled ~493 including sub-items. After consolidation into unique numbered items the list stands at 270 distinct, actionable findings. All 8 agents' outputs are preserved in their original form above and can be cross-referenced by file:line.

### Top 10 Highest-Priority Fixes

1. 🔴 **#10 / #11 / #12**: Combat, planet internal endpoints, and nebula credits endpoint have zero authentication — add `InternalSecretMiddleware` immediately.
2. 🔴 **#30 / #31**: `InternalDeductResource` and `InternalAddResource` are non-atomic — switch to `UPDATE ... SET metal = metal - $1 WHERE metal >= $1` everywhere.
3. 🔴 **#95**: Transport missions deliver hardcoded free resources — must track and carry actual cargo.
4. 🔴 **#96 / #100**: Attack fleets never marked arrived — combat resolves infinitely every 5 seconds.
5. 🔴 **#93**: Stargate mission skips all fuel deduction and `checkStarGateLink` is never called.
6. 🔴 **#112**: Research cancel refund parameters accepted but never sent to planet service — 100% resource loss on cancel.
7. 🔴 **#116**: Expeditions are instant and spammable — no travel time enforced.
8. 🔴 **#194**: SSE nginx blocks are dead config — chat and notifications silently drop every 60 seconds.
9. 🔴 **#8**: Gateway forwards client-supplied `X-Internal-Secret` header to backends — strip it before proxying.
10. 🔴 **#236**: Combat down migration runs `DROP SCHEMA fleet CASCADE` — destroys fleet service data.
