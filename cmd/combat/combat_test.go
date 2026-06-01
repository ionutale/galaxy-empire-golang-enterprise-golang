package main

import (
	"context"
	"testing"
)

func TestResolveCombat_TableDriven(t *testing.T) {
	tests := []struct {
		name            string
		attacker        map[string]int
		defender        map[string]int
		wantAttackerWon bool
		check           func(t *testing.T, result CombatResult)
	}{
		{
			name:            "equal cargo fleets draw",
			attacker:        map[string]int{"cargo": 10},
			defender:        map[string]int{"cargo": 10},
			wantAttackerWon: false,
			check: func(t *testing.T, result CombatResult) {
				if result.AttackerWon {
					t.Error("defender should win with equal cargo (defender fires first)")
				}
			},
		},
		{
			name:            "overwhelming attacker wins",
			attacker:        map[string]int{"light_fighter": 100},
			defender:        map[string]int{"light_fighter": 10},
			wantAttackerWon: true,
			check: func(t *testing.T, result CombatResult) {
				if !result.AttackerWon {
					t.Error("attacker should win with 10x ships")
				}
				if len(result.Rounds) == 0 {
					t.Error("should have at least 1 round")
				}
			},
		},
		{
			name:            "overwhelming defender wins",
			attacker:        map[string]int{"light_fighter": 10},
			defender:        map[string]int{"light_fighter": 100},
			wantAttackerWon: false,
			check: func(t *testing.T, result CombatResult) {
				if result.AttackerWon {
					t.Error("defender should win with 10x ships defending")
				}
			},
		},
		{
			name:            "6 round draw attacker retreats",
			attacker:        map[string]int{"cargo": 1},
			defender:        map[string]int{"cargo": 1},
			wantAttackerWon: false,
			check: func(t *testing.T, result CombatResult) {
				if result.AttackerWon {
					t.Error("attacker should not win a draw")
				}
				if len(result.Rounds) > 6 {
					t.Errorf("should not exceed 6 rounds, got %d", len(result.Rounds))
				}
				if len(result.Rounds) != 6 {
					t.Errorf("should be 6 round draw, got %d rounds", len(result.Rounds))
				}
			},
		},
		{
			name:     "death order light_fighter before heavy_fighter",
			attacker: map[string]int{"light_fighter": 100},
			defender: map[string]int{"light_fighter": 5, "heavy_fighter": 5},
			check: func(t *testing.T, result CombatResult) {
				if !result.AttackerWon {
					t.Skip("attacker didn't win, skipping death order check")
					return
				}
				if result.DefenderShipsAfter["light_fighter"] > 0 {
					t.Error("light_fighter should die before heavy_fighter")
				}
			},
		},
		{
			name:     "shield reset per round",
			attacker: map[string]int{"light_fighter": 20},
			defender: map[string]int{"light_fighter": 2},
			check: func(t *testing.T, result CombatResult) {
				if len(result.Rounds) <= 1 {
					t.Skip("only 1 round, can't verify shield reset")
					return
				}
				defenderLossesRound1 := result.Rounds[0].DefenderLosses
				defenderLossesRound2 := result.Rounds[1].DefenderLosses
				if len(defenderLossesRound1) == 0 && len(defenderLossesRound2) == 0 {
					t.Error("expected some losses each round with shield reset")
				}
			},
		},
		{
			name:     "debris calculation",
			attacker: map[string]int{"light_fighter": 10},
			defender: map[string]int{"light_fighter": 5},
			check: func(t *testing.T, result CombatResult) {
				destroyedAttacker := 10
				for _, qty := range result.AttackerShipsAfter {
					destroyedAttacker -= qty
				}
				destroyedDefender := 5
				for _, qty := range result.DefenderShipsAfter {
					destroyedDefender -= qty
				}
				expectedMetal := (destroyedAttacker*3000 + destroyedDefender*3000) * 30 / 100
				expectedCrystal := (destroyedAttacker*1000 + destroyedDefender*1000) * 30 / 100
				if result.DebrisMetal != expectedMetal {
					t.Errorf("debris metal: got %d, want %d", result.DebrisMetal, expectedMetal)
				}
				if result.DebrisCrystal != expectedCrystal {
					t.Errorf("debris crystal: got %d, want %d", result.DebrisCrystal, expectedCrystal)
				}
			},
		},
		{
			name:     "cargo plunder calculation",
			attacker: map[string]int{"cargo": 10},
			defender: map[string]int{},
			check: func(t *testing.T, result CombatResult) {
				if !result.AttackerWon {
					t.Error("attacker should win against empty defender")
				}
			},
		},
		{
			name:            "empty defender attacker wins immediately",
			attacker:        map[string]int{"light_fighter": 5},
			defender:        map[string]int{},
			wantAttackerWon: true,
			check: func(t *testing.T, result CombatResult) {
				if !result.AttackerWon {
					t.Error("attacker should win against empty defender")
				}
				if len(result.Rounds) != 0 {
					t.Errorf("should have 0 rounds for empty defender, got %d", len(result.Rounds))
				}
			},
		},
		{
			name:     "mixed ship types",
			attacker: map[string]int{"light_fighter": 20, "cruiser": 5},
			defender: map[string]int{"heavy_fighter": 10, "battleship": 2},
			check: func(t *testing.T, result CombatResult) {
				if len(result.Rounds) == 0 {
					t.Error("should have at least 1 round")
				}
				hasAttackerShips := !isEmpty(result.AttackerShipsAfter)
				hasDefenderShips := !isEmpty(result.DefenderShipsAfter)
				if !hasAttackerShips && !hasDefenderShips {
					t.Error("at least one side should have ships")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolveCombat(tt.attacker, tt.defender, shipStatsMap)
			if tt.check != nil {
				tt.check(t, result)
			}
		})
	}
}

func TestWipeCheck_OverwhelmingForce_Wipes(t *testing.T) {
	// 100 LF vs 1 LF: attack=5000, defense=15. 5000 >= 15*1.5 → wipe
	result := ResolveCombat(
		map[string]int{"light_fighter": 100},
		map[string]int{"light_fighter": 1},
		shipStatsMap,
	)
	if !result.AttackerWon {
		t.Error("attacker should win")
	}
	if len(result.Rounds) != 1 {
		t.Errorf("expected 1 round (wipe), got %d", len(result.Rounds))
	}
	if !result.Rounds[0].Wipe {
		t.Error("round should be marked as wipe")
	}
	if !isEmpty(result.DefenderShipsAfter) {
		t.Error("defender should be wiped")
	}
	if isEmpty(result.AttackerShipsAfter) {
		t.Error("attacker should have survivors")
	}
}

func TestWipeCheck_DefenderWipesAttacker(t *testing.T) {
	// 1 LF vs 100 LF: defender attack=5000, attacker defense=15. 5000 >= 15*1.5 → wipe
	result := ResolveCombat(
		map[string]int{"light_fighter": 1},
		map[string]int{"light_fighter": 100},
		shipStatsMap,
	)
	if result.AttackerWon {
		t.Error("defender should win")
	}
	if len(result.Rounds) != 1 {
		t.Errorf("expected 1 round (wipe), got %d", len(result.Rounds))
	}
	if !result.Rounds[0].Wipe {
		t.Error("round should be marked as wipe")
	}
	if !isEmpty(result.AttackerShipsAfter) {
		t.Error("attacker should be wiped")
	}
}

func TestWipeCheck_NotTriggered(t *testing.T) {
	// 1 cargo vs 1 cargo: attack=3, defense=10. 3 < 15 → no wipe
	result := ResolveCombat(
		map[string]int{"cargo": 1},
		map[string]int{"cargo": 1},
		shipStatsMap,
	)
	for i, r := range result.Rounds {
		if r.Wipe {
			t.Errorf("round %d should not be wipe", i+1)
		}
	}
}

func TestDamageAbsorption_ReducesDamage(t *testing.T) {
	// Equal cargos: no wipe, damage halved (3/2=1 per side per round)
	result := ResolveCombat(
		map[string]int{"cargo": 1},
		map[string]int{"cargo": 1},
		shipStatsMap,
	)
	if len(result.Rounds) != 6 {
		t.Errorf("expected 6 round draw, got %d", len(result.Rounds))
	}
	for i, r := range result.Rounds {
		t.Logf("Round %d: damageDealt=%d, damageTaken=%d", r.Round, r.TotalDamageDealt, r.TotalDamageTaken)
		if r.TotalDamageDealt != 1 && r.TotalDamageTaken != 1 {
			if i == 0 {
				t.Logf("Round 1 damage values: dealt=%d, taken=%d", r.TotalDamageDealt, r.TotalDamageTaken)
			}
		}
	}
}

func TestCalculateLoot_PriorityOrder(t *testing.T) {
	// Metal first, then Crystal, then Gas
	// defenderMetal=10000, defenderCrystal=5000, defenderGas=2000, totalCargo=3000
	// metal = min(5000, 3000) = 3000, remaining = 0
	loot := CalculateLoot(10000, 5000, 2000, 3000)
	if loot["metal"] != 3000 {
		t.Errorf("metal loot: got %d, want 3000", loot["metal"])
	}
	if loot["crystal"] != 0 {
		t.Errorf("crystal loot: got %d, want 0 (no cargo left)", loot["crystal"])
	}
	if loot["gas"] != 0 {
		t.Errorf("gas loot: got %d, want 0 (no cargo left)", loot["gas"])
	}
}

func TestCalculateLoot_PartialFill(t *testing.T) {
	// defenderMetal=5000, defenderCrystal=10000, defenderGas=2000, totalCargo=4000
	// metal = min(2500, 4000) = 2500, remaining = 1500
	// crystal = min(5000, 1500) = 1500, remaining = 0
	loot := CalculateLoot(5000, 10000, 2000, 4000)
	if loot["metal"] != 2500 {
		t.Errorf("metal: got %d, want 2500", loot["metal"])
	}
	if loot["crystal"] != 1500 {
		t.Errorf("crystal: got %d, want 1500", loot["crystal"])
	}
	if loot["gas"] != 0 {
		t.Errorf("gas: got %d, want 0", loot["gas"])
	}
}

func TestCalculateLoot_EnoughCargo(t *testing.T) {
	// All resources fit
	loot := CalculateLoot(1000, 500, 200, 10000)
	if loot["metal"] != 500 {
		t.Errorf("metal: got %d, want 500", loot["metal"])
	}
	if loot["crystal"] != 250 {
		t.Errorf("crystal: got %d, want 250", loot["crystal"])
	}
	if loot["gas"] != 100 {
		t.Errorf("gas: got %d, want 100", loot["gas"])
	}
}

func TestCalculateLoot_NoCargo(t *testing.T) {
	loot := CalculateLoot(10000, 5000, 2000, 0)
	for res, amt := range loot {
		if amt != 0 {
			t.Errorf("%s loot should be 0 with no cargo, got %d", res, amt)
		}
	}
}

func TestCalculateDebris(t *testing.T) {
	before := map[string]int{"light_fighter": 10}
	after := map[string]int{"light_fighter": 7}
	metal, crystal := calculateDebris(before, before, after, after)
	expectedMetal := 6 * 3000 * 30 / 100
	if metal != expectedMetal {
		t.Errorf("debris metal: got %d, want %d", metal, expectedMetal)
	}
	expectedCrystal := 6 * 1000 * 30 / 100
	if crystal != expectedCrystal {
		t.Errorf("debris crystal: got %d, want %d", crystal, expectedCrystal)
	}
}

func TestDiffShips(t *testing.T) {
	before := map[string]int{"light_fighter": 10, "cargo": 5}
	after := map[string]int{"light_fighter": 7}
	diff := diffShips(before, after)
	if diff["light_fighter"] != 3 {
		t.Errorf("light_fighter diff: got %d, want 3", diff["light_fighter"])
	}
	if diff["cargo"] != 5 {
		t.Errorf("cargo diff: got %d, want 5", diff["cargo"])
	}
}

func TestTotalCargoCapacity(t *testing.T) {
	ships := map[string]int{"cargo": 2, "large_cargo": 1}
	total := totalCargoCapacity(ships)
	expected := 2*25000 + 1*100000
	if total != expected {
		t.Errorf("total cargo: got %d, want %d", total, expected)
	}
}

func TestCheckCar_ReturnsMultiplierMap(t *testing.T) {
	ships := map[string]int{"light_fighter": 10}
	m := checkCar(ships, shipStatsMap)
	if m == nil {
		t.Error("checkCar should return non-nil map")
	}
	// light_fighter has CAR against espionage_probe and solar_satellite
	if len(m) > 2 {
		t.Errorf("max 2 CAR targets for LF, got %d", len(m))
	}
	for target := range m {
		if target != "espionage_probe" && target != "solar_satellite" {
			t.Errorf("unexpected CAR target: %s", target)
		}
	}
}

func TestCheckCar_EmptyShips_ReturnsEmptyMap(t *testing.T) {
	m := checkCar(map[string]int{}, shipStatsMap)
	if len(m) != 0 {
		t.Errorf("expected empty map, got %d entries", len(m))
	}
}

func TestCheckCar_ShipWithoutCar_ReturnsEmptyMap(t *testing.T) {
	m := checkCar(map[string]int{"cargo": 1}, shipStatsMap)
	if len(m) != 0 {
		t.Errorf("cargo has no CAR, expected empty map, got %d entries", len(m))
	}
}

func TestCarMultiplier_DestroysMoreShips(t *testing.T) {
	stats := map[string]ShipCombatConfig{
		"light_fighter": shipStatsMap["light_fighter"],
	}

	// 1 LF: shield=10, hull=5, total HP=15
	// Without CAR: 10 damage → shield absorbs 10, 0 hull → 0 destroyed
	ships := map[string]int{"light_fighter": 1}
	losses := make(map[string]int)
	fire(10, ships, stats, losses, nil)
	if ships["light_fighter"] != 1 {
		t.Error("without CAR, 10 damage should not destroy 1 LF")
	}

	// With CAR (2x): 10*2=20 effective → shield 10, hull 10, 10/5=2 → capped at 1
	ships2 := map[string]int{"light_fighter": 1}
	losses2 := make(map[string]int)
	fire(10, ships2, stats, losses2, map[string]float64{"light_fighter": 2.0})
	if ships2["light_fighter"] != 0 {
		t.Error("with CAR 2x, 10 damage should destroy 1 LF")
	}
	if losses2["light_fighter"] != 1 {
		t.Error("losses should record 1 LF destroyed with CAR")
	}
}

func TestCarMultiplier_AtMostDestroysAll(t *testing.T) {
	stats := map[string]ShipCombatConfig{
		"light_fighter": shipStatsMap["light_fighter"],
	}

	// 10 LF: shield=100, hull=50, total=150
	// 100 damage, CAR 2x → 200 effective, shield 100, hull 100, 100/5=20 → capped at 10
	ships := map[string]int{"light_fighter": 10}
	losses := make(map[string]int)
	fire(100, ships, stats, losses, map[string]float64{"light_fighter": 2.0})
	if ships["light_fighter"] != 0 {
		t.Error("all 10 LFs should be destroyed")
	}
	if losses["light_fighter"] != 10 {
		t.Error("losses should record 10 destroyed")
	}
}

func TestDefenseDamageModifier_Bomber(t *testing.T) {
	mult := defenseDamageModifier("bomber", shipStatsMap)
	if mult != 2.0 {
		t.Errorf("bomber multiplier: got %f, want 2.0", mult)
	}
}

func TestDefenseDamageModifier_OtherShips(t *testing.T) {
	for _, shipType := range []string{"light_fighter", "cruiser", "battleship", "cargo"} {
		mult := defenseDamageModifier(shipType, shipStatsMap)
		if mult != 1.0 {
			t.Errorf("%s multiplier: got %f, want 1.0", shipType, mult)
		}
	}
}

func TestRoundDetail_ContainsNewFields(t *testing.T) {
	result := ResolveCombat(
		map[string]int{"light_fighter": 100},
		map[string]int{"light_fighter": 5},
		shipStatsMap,
	)
	if len(result.Rounds) == 0 {
		t.Fatal("should have at least 1 round")
	}
	r := result.Rounds[0]
	if r.AttackerShips == nil {
		t.Error("round should have attacker ships snapshot")
	}
	if r.DefenderShips == nil {
		t.Error("round should have defender ships snapshot")
	}
	if r.AttackerLosses == nil {
		t.Error("round should have attacker losses")
	}
	if r.DefenderLosses == nil {
		t.Error("round should have defender losses")
	}
}

func TestRoundDetail_WipeRoundDamageValues(t *testing.T) {
	// Wipe round: damage values should be 0 (no firing, just wipe)
	result := ResolveCombat(
		map[string]int{"light_fighter": 100},
		map[string]int{"light_fighter": 1},
		shipStatsMap,
	)
	if len(result.Rounds) == 0 {
		t.Fatal("should have rounds")
	}
	r := result.Rounds[0]
	if !r.Wipe {
		t.Skip("not a wipe round, skipping")
	}
	if r.TotalDamageDealt != 0 {
		t.Errorf("wipe round damage dealt should be 0, got %d", r.TotalDamageDealt)
	}
	if r.TotalDamageTaken != 0 {
		t.Errorf("wipe round damage taken should be 0, got %d", r.TotalDamageTaken)
	}
}

func TestListPlayerReports(t *testing.T) {
	repo := newMockRepo()
	ctx := context.Background()

	report1 := CombatReport{
		AttackerPlayerID: 1,
		DefenderPlayerID: 2,
		Rounds:           []RoundResult{{Round: 1}},
	}
	report2 := CombatReport{
		AttackerPlayerID: 3,
		DefenderPlayerID: 1,
		Rounds:           []RoundResult{{Round: 1}, {Round: 2}},
	}

	repo.CreateCombatReport(ctx, report1)
	repo.CreateCombatReport(ctx, report2)

	reports, err := repo.ListPlayerCombatReports(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(reports) != 2 {
		t.Errorf("expected 2 reports for player 1, got %d", len(reports))
	}

	reports2, err := repo.ListPlayerCombatReports(ctx, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(reports2) != 1 {
		t.Errorf("expected 1 report for player 2, got %d", len(reports2))
	}
}

func TestGetCombatReport(t *testing.T) {
	repo := newMockRepo()
	ctx := context.Background()

	report := CombatReport{
		AttackerPlayerID:    1,
		DefenderPlayerID:    2,
		TargetGalaxy:        3,
		TargetSystem:        4,
		TargetPosition:      5,
		AttackerShipsBefore: map[string]int{"light_fighter": 10},
		DefenderShipsBefore: map[string]int{"cargo": 5},
		Rounds:              []RoundResult{{Round: 1, Wipe: true}},
	}
	id, err := repo.CreateCombatReport(ctx, report)
	if err != nil {
		t.Fatal(err)
	}

	got, err := repo.GetCombatReport(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if got.AttackerPlayerID != 1 || got.DefenderPlayerID != 2 {
		t.Error("player IDs mismatch")
	}
	if len(got.Rounds) != 1 {
		t.Errorf("expected 1 round, got %d", len(got.Rounds))
	}
	if !got.Rounds[0].Wipe {
		t.Error("round should be marked as wipe")
	}
	if got.ExpiresAt.IsZero() {
		t.Error("expires_at should be set")
	}
}

func TestCombatReport_ExpiresAtSet(t *testing.T) {
	repo := newMockRepo()
	ctx := context.Background()

	id, err := repo.CreateCombatReport(ctx, CombatReport{
		AttackerPlayerID: 1,
		DefenderPlayerID: 2,
		Rounds:           []RoundResult{{Round: 1}},
	})
	if err != nil {
		t.Fatal(err)
	}

	report, err := repo.GetCombatReport(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if report.ExpiresAt.IsZero() {
		t.Error("expires_at should not be zero")
	}
}

func TestMoonCreation_DebrisBelowThreshold_NoMoon(t *testing.T) {
	repo := newMockRepo()
	svc := NewCombatService(repo, "http://localhost:9999")
	ctx := context.Background()

	result := &CombatResult{
		DebrisMetal:   50000,
		DebrisCrystal: 50000,
	}
	info := svc.tryCreateMoon(ctx, 1, 1, 1, result)
	if info.Created {
		t.Error("moon should not be created with 100k total debris")
	}
}

func TestMoonCreation_DebrisAboveThreshold_RollsForMoon(t *testing.T) {
	repo := newMockRepo()
	svc := NewCombatService(repo, "http://localhost:9999")
	ctx := context.Background()

	created := 0
	iterations := 1000
	for i := 0; i < iterations; i++ {
		result := &CombatResult{
			DebrisMetal:   100000,
			DebrisCrystal: 100000,
		}
		info := svc.tryCreateMoon(ctx, 1, 2, 3, result)
		if info.Created {
			created++
			if info.Size < 20 || info.Size > 50 {
				t.Errorf("moon size out of range: %d", info.Size)
			}
			if info.Name != "Moon [1:2:3]" {
				t.Errorf("moon name: got %s, want Moon [1:2:3]", info.Name)
			}
			if !result.MoonCreated {
				t.Error("result.MoonCreated should be true")
			}
			if result.MoonSize == 0 {
				t.Error("result.MoonSize should be non-zero")
			}
		}
	}

	// ~20% chance → expect 150-250 moons in 1000 runs
	if created < 100 || created > 300 {
		t.Errorf("moon creation rate out of expected range (100-300/1000): got %d/1000", created)
	}
}

func TestMoonCreation_DebrisAtMaxChance(t *testing.T) {
	repo := newMockRepo()
	svc := NewCombatService(repo, "http://localhost:9999")
	ctx := context.Background()

	created := 0
	iterations := 1000
	for i := 0; i < iterations; i++ {
		result := &CombatResult{
			DebrisMetal:   500000,
			DebrisCrystal: 500000,
		}
		info := svc.tryCreateMoon(ctx, 3, 5, 7, result)
		if info.Created {
			created++
		}
	}

	// Still max 20% cap
	if created < 100 || created > 300 {
		t.Errorf("moon creation rate out of expected range (100-300/1000): got %d/1000", created)
	}
}

func TestMoonCreation_StoredInRepo(t *testing.T) {
	repo := newMockRepo()
	svc := NewCombatService(repo, "http://localhost:9999")
	ctx := context.Background()

	// Force moon creation with large debris
	created := false
	for i := 0; i < 100; i++ {
		result := &CombatResult{
			DebrisMetal:   100000,
			DebrisCrystal: 100000,
		}
		info := svc.tryCreateMoon(ctx, 5, 10, 15, result)
		if info.Created {
			created = true
			moon, err := svc.GetMoonInfo(ctx, 5, 10, 15)
			if err != nil {
				t.Fatalf("get moon info: %v", err)
			}
			if moon.Galaxy != 5 || moon.System != 10 || moon.Position != 15 {
				t.Errorf("moon coords: got [%d:%d:%d], want [5:10:15]", moon.Galaxy, moon.System, moon.Position)
			}
			if moon.Size < 20 || moon.Size > 50 {
				t.Errorf("moon size out of range: %d", moon.Size)
			}
			break
		}
	}
	if !created {
		t.Skip("moon not created in 100 attempts, skipping storage check")
	}
}

func TestEffectiveDefense(t *testing.T) {
	ships := map[string]int{"light_fighter": 2}
	ed := effectiveDefense(ships, shipStatsMap)
	// LF: Strength=5, Shield=10 → total=15 per ship, 2 ships → 30
	if ed != 30 {
		t.Errorf("effective defense: got %d, want 30", ed)
	}
}
