package main

import "math/rand"

func ResolveCombat(attackerShips, defenderShips map[string]int, shipStats map[string]ShipCombatConfig) CombatResult {
	attacker := copyShips(attackerShips)
	defender := copyShips(defenderShips)

	if isEmpty(defender) {
		return CombatResult{
			AttackerWon:        true,
			AttackerShipsAfter: attacker,
			DefenderShipsAfter: defender,
		}
	}

	var rounds []RoundResult
	maxRounds := 6

	for round := 1; round <= maxRounds; round++ {
		attackerLosses := make(map[string]int)
		defenderLosses := make(map[string]int)

		attackerTotalAttack := totalAttack(attacker, shipStats)
		defenderTotalAttack := totalAttack(defender, shipStats)
		attackerEffectiveDefense := effectiveDefense(attacker, shipStats)
		defenderEffectiveDefense := effectiveDefense(defender, shipStats)

		wipe := false

		if attackerTotalAttack >= defenderEffectiveDefense*3/2 && !isEmpty(defender) {
			for shipType := range defender {
				defenderLosses[shipType] = defender[shipType]
				delete(defender, shipType)
			}
			wipe = true
		} else if defenderTotalAttack >= attackerEffectiveDefense*3/2 && !isEmpty(attacker) {
			for shipType := range attacker {
				attackerLosses[shipType] = attacker[shipType]
				delete(attacker, shipType)
			}
			wipe = true
		}

		var totalDamageDealt, totalDamageTaken int

		if !wipe {
			attackerCarMult := checkCar(attacker, shipStats)
			defenderCarMult := checkCar(defender, shipStats)

			effectiveDefenderDamage := defenderTotalAttack / 2
			totalDamageTaken = effectiveDefenderDamage
			defenderFires(effectiveDefenderDamage, attacker, shipStats, attackerLosses, defenderCarMult)
			cleanShips(attacker)

			effectiveAttackerDamage := attackerTotalAttack / 2
			totalDamageDealt = effectiveAttackerDamage
			attackerFires(effectiveAttackerDamage, defender, shipStats, defenderLosses, attackerCarMult)
			cleanShips(defender)
		}

		rounds = append(rounds, RoundResult{
			Round:            round,
			Wipe:             wipe,
			AttackerShips:    copyShips(attacker),
			DefenderShips:    copyShips(defender),
			AttackerLosses:   attackerLosses,
			DefenderLosses:   defenderLosses,
			TotalDamageDealt: totalDamageDealt,
			TotalDamageTaken: totalDamageTaken,
		})

		if isEmpty(attacker) || isEmpty(defender) {
			break
		}
	}

	attackerWon := false
	if isEmpty(defender) && !isEmpty(attacker) {
		attackerWon = true
	}

	debrisMetal, debrisCrystal := calculateDebris(attackerShips, defenderShips, attacker, defender)

	attackerLoot := make(map[string]int)
	defenderLostRes := make(map[string]int)

	return CombatResult{
		AttackerWon:        attackerWon,
		Rounds:             rounds,
		AttackerShipsAfter: attacker,
		DefenderShipsAfter: defender,
		DebrisMetal:        debrisMetal,
		DebrisCrystal:      debrisCrystal,
		AttackerLoot:       attackerLoot,
		DefenderLostRes:    defenderLostRes,
	}
}

func copyShips(ships map[string]int) map[string]int {
	c := make(map[string]int, len(ships))
	for k, v := range ships {
		if v > 0 {
			c[k] = v
		}
	}
	return c
}

func isEmpty(ships map[string]int) bool {
	for _, qty := range ships {
		if qty > 0 {
			return false
		}
	}
	return true
}

func totalAttack(ships map[string]int, stats map[string]ShipCombatConfig) int {
	total := 0
	for shipType, qty := range ships {
		if s, ok := stats[shipType]; ok {
			total += qty * s.Attack
		}
	}
	return total
}

func effectiveDefense(ships map[string]int, stats map[string]ShipCombatConfig) int {
	total := 0
	for shipType, qty := range ships {
		if s, ok := stats[shipType]; ok {
			total += qty * (s.Strength + s.Shield)
		}
	}
	return total
}

func cleanShips(ships map[string]int) {
	for k, v := range ships {
		if v <= 0 {
			delete(ships, k)
		}
	}
}

func defenderFires(damage int, targetShips map[string]int, stats map[string]ShipCombatConfig, losses map[string]int, carMultipliers map[string]float64) {
	fire(damage, targetShips, stats, losses, carMultipliers)
}

func attackerFires(damage int, targetShips map[string]int, stats map[string]ShipCombatConfig, losses map[string]int, carMultipliers map[string]float64) {
	fire(damage, targetShips, stats, losses, carMultipliers)
}

func fire(damage int, targetShips map[string]int, stats map[string]ShipCombatConfig, losses map[string]int, carMultipliers map[string]float64) {
	remaining := damage
	for _, shipType := range deathPriority {
		if remaining <= 0 {
			break
		}
		qty, ok := targetShips[shipType]
		if !ok || qty <= 0 {
			continue
		}
		cfg, ok := stats[shipType]
		if !ok {
			continue
		}

		shieldPool := qty * cfg.Shield
		hullPool := qty * cfg.Strength
		totalHP := shieldPool + hullPool

		damageToType := remaining
		if damageToType > totalHP {
			damageToType = totalHP
		}
		remaining -= damageToType

		carMult := 1.0
		if m, ok := carMultipliers[shipType]; ok {
			carMult = m
		}

		effectiveDamage := int(float64(damageToType) * carMult)
		if effectiveDamage > totalHP {
			effectiveDamage = totalHP
		}

		hullDamage := effectiveDamage - shieldPool
		if hullDamage < 0 {
			hullDamage = 0
		}
		shipsDestroyed := hullDamage / cfg.Strength
		if shipsDestroyed > qty {
			shipsDestroyed = qty
		}
		if shipsDestroyed > 0 {
			targetShips[shipType] = qty - shipsDestroyed
			losses[shipType] = shipsDestroyed
		}
	}

	cleanShips(targetShips)
}

func calculateDebris(attackerBefore, defenderBefore, attackerAfter, defenderAfter map[string]int) (int, int) {
	var debrisMetal, debrisCrystal int

	for _, ships := range []struct {
		before map[string]int
		after  map[string]int
	}{{attackerBefore, attackerAfter}, {defenderBefore, defenderAfter}} {
		for shipType, beforeQty := range ships.before {
			afterQty := ships.after[shipType]
			destroyed := beforeQty - afterQty
			if destroyed <= 0 {
				continue
			}
			if cfg, ok := shipCombatConfig(shipType); ok {
				debrisMetal += destroyed * cfg.Metal
				debrisCrystal += destroyed * cfg.Crystal
			}
		}
	}

	debrisMetal = debrisMetal * 30 / 100
	debrisCrystal = debrisCrystal * 30 / 100
	return debrisMetal, debrisCrystal
}

func CalculateLoot(defenderMetal, defenderCrystal, defenderGas, totalCargo int) map[string]int {
	loot := make(map[string]int)
	remaining := totalCargo

	availMetal := defenderMetal / 2
	loot["metal"] = min(availMetal, remaining)
	remaining -= loot["metal"]

	availCrystal := defenderCrystal / 2
	loot["crystal"] = min(availCrystal, remaining)
	remaining -= loot["crystal"]

	availGas := defenderGas / 2
	loot["gas"] = min(availGas, remaining)

	return loot
}

func CalculateDefenderLostResources(defenderMetal, defenderCrystal, defenderGas int, loot map[string]int) map[string]int {
	lost := make(map[string]int)
	lost["metal"] = min(defenderMetal, loot["metal"])
	lost["crystal"] = min(defenderCrystal, loot["crystal"])
	lost["gas"] = min(defenderGas, loot["gas"])
	return lost
}

func checkCar(ships map[string]int, shipStats map[string]ShipCombatConfig) map[string]float64 {
	multipliers := make(map[string]float64)
	for shipType, qty := range ships {
		if qty <= 0 {
			continue
		}
		cfg, ok := shipStats[shipType]
		if !ok {
			continue
		}
		for targetType, rfAmount := range cfg.CarTargets {
			if rfAmount <= 0 {
				continue
			}
			chance := 1.0 - 1.0/float64(rfAmount)
			if rand.Float64() < chance {
				multipliers[targetType] = 2.0
			}
		}
	}
	return multipliers
}

func ResolveMissileStrike(req MissileStrikeRequest) MissileStrikeResult {
	result := MissileStrikeResult{
		IPMsLaunched:      req.IPMs,
		DefensesDestroyed: make(map[string]int),
		DefensesDamaged:   make(map[string]int),
	}

	remainingIPMs := req.IPMs
	abmsAvailable := req.ABMDefense

	for i := 0; i < req.IPMs && abmsAvailable > 0; i++ {
		if rand.Float64() < 0.7 {
			remainingIPMs--
			abmsAvailable--
			result.IPMsIntercepted++
			result.ABMsUsed++
		}
	}

	ipmDamage := 8000 + 1000*req.TechLevel

	defenseTypes := make([]string, 0, len(req.Defenses))
	for dType := range req.Defenses {
		defenseTypes = append(defenseTypes, dType)
	}

	for i := 0; i < remainingIPMs; i++ {
		if len(defenseTypes) == 0 {
			break
		}
		targetIdx := rand.Intn(len(defenseTypes))
		targetType := defenseTypes[targetIdx]

		cfg, ok := defenseConfig(targetType)
		if !ok {
			continue
		}

		targetQty := req.Defenses[targetType]
		if targetQty <= 0 {
			defenseTypes = append(defenseTypes[:targetIdx], defenseTypes[targetIdx+1:]...)
			i--
			continue
		}

		if ipmDamage >= cfg.Strength {
			destroyed := ipmDamage / cfg.Strength
			if destroyed > targetQty {
				destroyed = targetQty
			}
			result.DefensesDestroyed[targetType] += destroyed
			req.Defenses[targetType] = targetQty - destroyed

			if req.Defenses[targetType] <= 0 {
				defenseTypes = append(defenseTypes[:targetIdx], defenseTypes[targetIdx+1:]...)
			}
		} else {
			result.DefensesDamaged[targetType]++
		}
	}

	result.IPMsLaunched = req.IPMs
	return result
}

func defenseDamageModifier(attackerType string, shipStats map[string]ShipCombatConfig) float64 {
	if cfg, ok := shipStats[attackerType]; ok {
		if cfg.DefenseDamageMultiplier > 1.0 {
			return cfg.DefenseDamageMultiplier
		}
	}
	return 1.0
}
