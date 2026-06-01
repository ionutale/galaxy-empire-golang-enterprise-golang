# PRD: Security Hardening & Reliability — Galaxy Empire

**Date:** 2026-06-01  
**Status:** Ready for implementation  
**Source:** findings.md audit (513 issues, 87 CRITICAL, 167 HIGH)

---

## Problem Statement

A full-codebase audit of Galaxy Empire's 18 Go microservices, Svelte frontend, and infrastructure configuration identified 513 issues across five severity tiers. The most urgent cluster is a set of 87 CRITICAL findings that, individually, allow players to:

- Claim quest and tutorial rewards without performing any game action
- Bypass admin authorisation and manipulate any player's resources
- Read any other player's combat history, radar events, and fleet positions
- Execute fleet missions with arbitrarily large (overflowing) ship counts
- Drain alliance banks concurrently faster than the balance allows
- Predict combat outcomes exactly (unseeded PRNG in combat service)

Beyond cheating, the session token is stored in `localStorage` and embedded in SSE URL query strings, where it appears in server access logs — a perimeter breach anywhere in the stack exposes every logged-in user's session.

At the reliability level, the fleet travel worker has no panic recovery (one bad fleet record kills all fleet movement), resource-deduction operations across services are non-atomic (partial failures leave the game in permanently inconsistent state), and several long-running background workers use `context.Background()` with no deadlines.

---

## Solution

Remediate all CRITICAL and HIGH findings in priority order, then address the MEDIUM and LOW tiers. Work is split into five phases so each phase is independently shippable and reviewable without blocking other development.

**Phase 1 — Authentication & Authorisation Foundation**  
Replace client-supplied trust headers with a proper auth model, fix admin middleware, and move the JWT token to an HttpOnly cookie.

**Phase 2 — Data Integrity & Race Conditions**  
Wrap all multi-step resource-modification flows in database transactions or atomic SQL. Eliminate the TOCTOU races across planet, fleet, research, alliance, and quest services.

**Phase 3 — Service-Level Security Hardening**  
Fix per-service vulnerabilities: combat PRNG seeding, quest self-reporting, tutorial skip bypass, combat report visibility, espionage data leakage, radar auth, and admin audit logging.

**Phase 4 — Infrastructure & Frontend Hardening**  
Add HttpOnly cookie auth flow, input size limits, rate limiting, SSE backoff, nginx security headers, Docker hardening, and health check coverage.

**Phase 5 — Code Quality & Observability**  
Address MEDIUM/LOW items: pagination everywhere, graceful shutdown, OpenTelemetry tracing, structured audit logging, TypeScript migration for the frontend, and component decomposition.

---

## User Stories

### Authentication & Session

1. As a player, I want my session token stored in an HttpOnly cookie so that a chat-based XSS cannot steal my credentials.
2. As a player, I want the SSE stream to authenticate via a short-lived signed ticket rather than a URL query-parameter token so that my session never appears in server access logs.
3. As a player, I want automatic logout when my token expires so that I am not silently operating on an expired session.
4. As a player, I want the login endpoint to respond in constant time whether or not my email exists in the system so that my account cannot be enumerated by timing.
5. As a player, I want my password to require at least 12 characters with mixed complexity so that my account cannot be brute-forced offline if the hash database is leaked.
6. As a platform operator, I want all authentication events (login, failed login, logout, token refresh) recorded in a structured audit log so that I can detect credential-stuffing attacks.

### Admin Security

7. As a platform operator, I want admin-only endpoints to validate admin status from the verified JWT claim, not from a client-supplied header, so that non-admin players cannot forge admin access.
8. As a platform operator, I want all admin actions (grant DM, grant credits, ban, resource override) to be written to an immutable audit log so that I can reconstruct any admin session after the fact.
9. As a platform operator, I want admin resource grants to be bounded (max DM, max credits per call) and validated server-side so that a compromised admin token cannot instantly win-the-game for a player.
10. As a platform operator, I want admin search queries to be protected against ReDoS by applying a query-length cap and using a full-text index rather than bare ILIKE wildcards.

### Quest & Tutorial Integrity

11. As a game designer, I want quest progress to be verified server-side against actual game events (ships built, attacks launched, research completed) so that players cannot self-report fake progress.
12. As a game designer, I want quest rewards to be idempotent — claimable exactly once — enforced by a DB-level unique constraint plus a compare-and-swap, so that concurrent claim requests cannot double-grant rewards.
13. As a game designer, I want tutorial steps to enforce prerequisites in order so that a player cannot skip to step 7 and claim the attack reward without having completed steps 1–6.
14. As a game designer, I want `SkipStep` to require a valid authenticated session so that unauthenticated callers cannot bypass the tutorial.
15. As a new player, I want tutorial rewards to be granted atomically so that completing a step never results in a step advance without the associated reward.

### Fleet & Combat Correctness

16. As a player, I want fleet dispatch to validate that ship quantities do not overflow so that I cannot create an integer-overflow exploit to bypass resource checks.
17. As a player, I want fleet split operations to be atomic so that I never lose ships if a split partially fails.
18. As a player, I want the fleet travel worker to recover from panics on individual fleet records so that one bad record does not stop all fleet movement for every player.
19. As a player, I want cargo fleets to actually deduct the transported resources from my planet at departure and add them at arrival, not deliver a free fixed amount.
20. As a player, I want to be unable to attack my own planet so that I cannot farm myself for loot.
21. As a player, I want fleet slot limits to include stationed fleets so that I cannot bypass the limit by using the stationed status.
22. As a player, I want combat outcomes to be unpredictable so that an attacker cannot replay the exact same fleet composition and guarantee a win through seed knowledge.
23. As a player, I want to see only my own combat reports, not any other player's, so that I cannot spy on other players' battle histories for free.
24. As a player, I want the moon creation chance to scale with debris amount beyond the base threshold rather than being capped at the floor value.
25. As a player, I want fleet missions that fail mid-execution (e.g., planet service unreachable) to roll back cleanly rather than leaving resources deducted with no fleet created.

### Resource & Economy Integrity

26. As a player, I want resource deduction operations to be atomic so that I cannot exploit concurrent requests to spend the same resources twice.
27. As a player, I want ship and defense build costs to be calculated safely for large quantities so that integer overflow cannot bypass resource requirements.
28. As a player, I want upper bounds enforced on build quantities (ships, defenses, missiles, ABMs) so that I cannot queue unreasonably large orders.
29. As a player, I want research resource deduction to roll back completely if any step fails, not partially refund through a series of compensating calls that can themselves fail.
30. As a player, I want alliance bank withdrawals to be atomic so that two players cannot both withdraw the last 1 000 metal simultaneously.
31. As a player, I want expedition DM charges to be idempotent so that a retry cannot double-charge me.
32. As a player, I want nebula SpeedUp to charge DM and apply the speed boost atomically so that a service restart during the operation does not charge me without applying the effect.

### Ownership & Access Control

33. As a player, I want planet internal endpoints to verify I own the planet before deducting resources, so that another service cannot drain my planet on my behalf.
34. As a player, I want to be unable to start research that deducts resources from a planet I do not own.
35. As a player, I want to be unable to view another player's radar events or resolve another player's radar alerts.
36. As a player, I want espionage probe ships to be deducted only after the target planet is confirmed valid, so I do not lose a ship probing a nonexistent target.
37. As a player, I want combat report visibility to be restricted to the attacker and the defender only.
38. As a player, I want moon coordinate queries to return only the existence/size of a moon, with no ownership or resource data, so that coordinates cannot be used to spy freely.
39. As a player, I want EUX radar scans to use the target coordinates from the request body rather than silently scanning a hardcoded target.

### Chat & Communication

40. As a player, I want chat messages to be stored and rendered safely so that a malicious player cannot inject HTML or scripts via chat.
41. As a player, I want to be unable to read messages from alliance channels I am not a member of.
42. As a player, I want chat message history to be paginated so that loading older messages does not transfer the entire chat history in one response.
43. As a player, I want the SSE chat stream to reconnect with exponential backoff so that a brief server restart does not trigger a thundering-herd reconnect storm from all clients.
44. As a player, I want notification `MarkRead` to verify I own the notification before marking it read.

### Infrastructure & Reliability

45. As a platform operator, I want all Docker services to run as a non-root user so that a container escape does not yield root access to the host.
46. As a platform operator, I want all secrets (JWT, internal secret, DB password) to be injected from a secrets manager or environment-specific `.env` file rather than committed to VCS.
47. As a platform operator, I want all services to have health checks in Docker Compose so that the gateway only receives traffic after the upstream service is ready.
48. As a platform operator, I want all HTTP servers to configure `ReadTimeout`, `WriteTimeout`, and `IdleTimeout` so that slow-loris and idle connections cannot exhaust goroutine pools.
49. As a platform operator, I want all services to handle `SIGTERM` gracefully so that in-flight requests complete before the process exits during rolling restarts.
50. As a platform operator, I want all background worker goroutines (fleet travel, research processor, event ticker) to accept a cancellable context with a deadline so that they shut down cleanly and do not run past their deadline.
51. As a platform operator, I want all request body sizes capped at a reasonable maximum (e.g., 1 MB) across all HTTP endpoints so that large-payload DoS is mitigated.
52. As a platform operator, I want nginx HTTPS redirect enforced and HTTP Strict Transport Security (HSTS) header set so that traffic cannot be downgraded.
53. As a platform operator, I want a structured audit log service or log sink that captures all security-sensitive events across all services in a queryable format.
54. As a platform operator, I want the rate limiter in the gateway to use route-pattern as the bucket key (not full path) and to garbage-collect expired buckets so that memory does not grow unbounded.
55. As a platform operator, I want all services to expose Prometheus-compatible metrics (request latency, error rate, DB pool stats) so that I can alert on degradation.

### Developer Experience

56. As a developer, I want `X-User-ID` extracted and validated in gateway middleware and injected into the request context so that each handler receives a typed, verified user ID without repeating boilerplate.
57. As a developer, I want all Go services to have a `recover()` middleware installed so that a panicking handler returns HTTP 500 instead of crashing the process.
58. As a developer, I want the research `fetchTechLevels` function to fetch all tech levels in parallel rather than 18 sequential HTTP calls, so that the list-techs endpoint is fast enough to use.
59. As a developer, I want all SQL migrations to use a proper migration runner (golang-migrate) with numbered up/down files rather than inline `CREATE IF NOT EXISTS` at service boot.
60. As a developer, I want the Svelte frontend split into typed sub-components with TypeScript so that API response shapes are statically checked and the file is maintainable.

---

## Implementation Decisions

### Phase 1 — Auth/Authz Foundation

- **Trust header replacement:** Gateway validates the JWT and sets the user ID into a signed request context value (or a HMAC-signed internal header), not a raw `X-User-ID` header. All downstream services read from this signed value, not from the raw header. Existing `X-User-ID` handlers become consumers of a centrally-validated context key.
- **Admin middleware:** `AdminOnly` reads the `is_admin` claim from the validated JWT payload stored in context, not from a header. The JWT must contain the admin flag at issue time; auth service sets it on login.
- **HttpOnly cookie auth:** Auth service sets `Set-Cookie: token=...; HttpOnly; SameSite=Strict; Secure` on login. Frontend uses `credentials: 'include'` on all fetch calls. Gateway reads the cookie if `Authorization` header is absent.
- **SSE auth ticket:** A new `/api/auth/sse-ticket` endpoint issues a short-lived (30 s) signed JWT containing only the player ID. Frontend passes this one-time ticket as a query parameter. Gateway validates it on SSE upgrade, then drops it from the URL before forwarding.

### Phase 2 — Atomic Operations

- **Atomic resource deduction:** All `UpdateResources` calls where a balance check precedes a deduction are replaced with `UPDATE … SET metal = metal - $1 WHERE id = $2 AND metal >= $1 RETURNING metal`. A `0 rows affected` result means insufficient balance. No application-level check needed.
- **Build-queue transactions:** `StartBuildingUpgrade`, `BuildShips`, `BuildDefenses`, `BuildIPM`, `BuildABM`, `BuildIronBehemoth` all wrapped in a single pgx transaction that creates the queue entry and deducts resources atomically.
- **Fleet split transaction:** `UpdateFleetShips` and `CreateFleet` execute inside a single transaction. The transaction is opened in the repository layer and passed via context.
- **Research deduction:** `deductResources` replaced with a single SQL call that atomically deducts all three resources in one `UPDATE … WHERE metal >= $1 AND crystal >= $2 AND gas >= $3`. Eliminates the compensating-refund chain entirely.
- **Quest claim CAS:** `ClaimPlayerQuest` uses `UPDATE … WHERE status = 'active' RETURNING id`. Zero rows = already claimed. The reward grant runs inside the same transaction.
- **VIP points deduplication:** `AddVIPPoints` checks `IF NOT EXISTS (SELECT 1 FROM vip_log WHERE queue_id = $1)` before inserting, preventing double credit on retried ticks.

### Phase 3 — Service Hardening

- **Combat PRNG:** Replace `math/rand` with `crypto/rand`-seeded `rand.New(rand.NewSource(seed))` for all combat randomness. Seed stored in the combat report for reproducibility/auditing.
- **Quest progress verification:** `ProgressUpdate` endpoint replaced with an internal event receiver. Other services POST to `/internal/quest/event` with a signed event payload (e.g., `{"type":"ship_built","player_id":X,"quantity":100}`). Quest service evaluates these events against requirement definitions.
- **Tutorial prerequisite chain:** `ClaimReward` reads the player's `current_step` from DB and validates `step == current_step + 1`. `SkipStep` requires `X-User-ID` from auth context and is restricted to admin callers.
- **Espionage ordering:** `deductProbe` called only after `getPlanetInfo` returns a valid, owned target.
- **Radar ownership:** `GetEvents` and `ResolveEvent` take `playerID` from auth context, not a query parameter. `ResolveEvent` uses `UPDATE … WHERE id = $1 AND player_id = $2`.
- **Admin audit log:** New `admin_audit_log` table. All `GrantDM`, `GrantCredits`, `BanPlayer`, `OverrideResources` calls write a row before executing. Service returns 500 if the audit write fails (fail-closed).

### Phase 4 — Infrastructure

- **Docker user:** All service Dockerfiles add `RUN adduser -D appuser && USER appuser`. Postgres uses the official image's built-in `postgres` user.
- **Secrets:** `docker-compose.yml` references `${JWT_SECRET}`, `${INTERNAL_SECRET}`, `${POSTGRES_PASSWORD}` from a `.env` file excluded from VCS. A `.env.example` file is committed with placeholder values.
- **HTTP timeouts:** All `http.ListenAndServe` replaced with `http.Server{ReadTimeout: 10s, WriteTimeout: 30s, IdleTimeout: 60s}`.
- **Graceful shutdown:** All services implement `signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGINT)` and call `server.Shutdown(ctx)` on signal.
- **Body size limit:** All routers add `http.MaxBytesReader(w, r.Body, 1<<20)` (1 MB) middleware.
- **Rate limiter fix:** Gateway rate limiter keyed on `(remoteIP, routePattern)` using chi's `RouteContext`. Eviction goroutine cleans up buckets older than the window.

### Phase 5 — Code Quality

- **Centralized user extraction:** Gateway middleware extracts and validates user ID once, stores in context. Helper `userIDFromCtx(ctx)` used by all handlers.
- **`recover()` middleware:** All chi routers wrap `middleware.Recoverer` (already present in some) — ensure all 18 services include it.
- **Parallel tech fetching:** `fetchTechLevels` uses `errgroup` to fire all 18 tech-level requests concurrently and aggregate results.
- **golang-migrate:** Each service gains a `migrations/` directory. Boot-time migration replaced with `migrate.Up()` from golang-migrate, which tracks applied versions in a `schema_migrations` table.
- **Frontend decomposition:** `App.svelte` split into `PlanetView`, `FleetView`, `ResearchView`, `AllianceView`, `ShopView`, `ChatView`, `AdminView` components. Shared API client module centralises fetch calls, adds CSRF token injection and error handling.

### Schema Changes

- `auth.users`: add `is_admin BOOLEAN NOT NULL DEFAULT FALSE`
- `fleet.fleets`: `ships TEXT` → `ships JSONB NOT NULL DEFAULT '{}'`; `cargo_metal/crystal/gas` → `BIGINT`
- `combat.reports`: fix column ordering; add `seed BIGINT` column
- `quest.player_quests`: add `NOT NULL`, `CHECK (status IN ('active','completed','claimed'))`, FK to `auth.users`
- `tutorial.player_tutorial`: move from inline SQL to migration file; add FK to `auth.users`
- All `completes_at` / `created_at` columns: `TIMESTAMP` → `TIMESTAMPTZ`
- New table: `admin.audit_log(id, admin_player_id, action, target_player_id, payload JSONB, created_at TIMESTAMPTZ)`
- New table: `auth.sse_tickets(id, player_id, expires_at TIMESTAMPTZ)`

---

## Testing Decisions

### What makes a good test

Tests should verify observable external behaviour, not implementation details. A test should break if and only if the external contract changes. Prefer table-driven tests with named cases. Tests should not mock the database — use a real Postgres instance (via `testcontainers-go` or a test DSN) so that atomic SQL, constraints, and transactions are exercised.

### Modules to test with priority

1. **Atomic resource deduction** (planet/repository): concurrent goroutine test — 10 goroutines each attempt to deduct 100 metal from a planet with 500 metal balance; assert exactly 5 succeed.
2. **Fleet split atomicity** (fleet/service): inject a DB failure after `UpdateFleetShips` and before `CreateFleet`; assert source fleet ships are unchanged.
3. **Quest claim idempotency** (quest/repository): 50 concurrent goroutines all attempt to claim the same quest; assert exactly 1 succeeds.
4. **Combat PRNG independence** (combat/combat): given the same seed, `ResolveCombat` returns the same result (deterministic); given different seeds, results differ.
5. **Admin authorisation** (admin/handler): table-driven test covering non-admin JWT, missing JWT, and admin JWT; assert 403/401/200 respectively.
6. **JWT algorithm pinning** (gateway/main): present a JWT signed with RS256; assert 401.
7. **Tutorial prerequisite chain** (tutorial/service): attempt to claim reward for step 4 with `current_step == 1`; assert error.
8. **Radar ownership** (radar/service): call `GetEvents` with playerID from context != playerID in query; assert 403.
9. **SSE subscriber race** (chat/service): 100 concurrent subscribe + broadcast operations under `-race`; assert no data race.
10. **processCompletedBuilds double-VIP** (planet/planet): process same queue entry twice; assert VIP points incremented exactly once.

### Prior art

Existing test files (`cmd/auth/auth_test.go`, `cmd/fleet/fleet_test.go`, `cmd/planet/planet_test.go`, `cmd/planet/handler_test.go`) establish the pattern: `httptest.NewRecorder`, chi router wired with a real repository backed by a test DB, table-driven cases. New tests should follow this pattern.

---

## Out of Scope

- **Real-time multiplayer tick synchronisation** — the travel worker is single-instance; horizontal scaling of the fleet worker is not addressed in this PRD.
- **Full TypeScript migration of the backend** — the Go services remain idiomatic Go; TypeScript is for the frontend only.
- **Kubernetes / production deployment** — Docker Compose hardening is in scope; Helm charts, K8s RBAC, and ingress TLS termination are not.
- **Payment / in-app purchase flows** — Dark Matter is the only premium currency; real-money purchase integration is out of scope.
- **GDPR / data deletion flows** — account deletion cascade is noted as a missing FK issue but a full right-to-erasure implementation is deferred.
- **New gameplay features** — this PRD is purely remediation. No new ships, buildings, or game mechanics.
- **Redis integration** — rate limiting, sessions, and pub/sub are handled within existing services; a Redis dependency is not introduced.

---

## Further Notes

- **Priority ordering within phases:** Within each phase, CRITICAL items come before HIGH before MEDIUM. Within CRITICAL, items that enable cheating (quest self-reporting, tutorial skip, admin bypass) should be resolved before items that are exploitable only with internal network access.
- **Backward compatibility:** Changing `X-User-ID` from a raw header to a context value is a breaking change for all 18 services simultaneously. The safest rollout is: (1) deploy the new signed header alongside the old raw header for one release, (2) migrate services one by one, (3) drop the raw header. The gateway can emit both during the transition window.
- **findings.md as the work-tracking source:** Each finding in `findings.md` has a sequential number. PR descriptions should reference the finding number(s) being closed (e.g., "Closes findings #15, #17, #19").
- **Regression risk for fleet/combat:** The fleet travel worker and combat resolver are the highest-risk areas for accidental regressions. Both need integration tests with a real DB and real service-to-service HTTP calls before any refactoring begins.
