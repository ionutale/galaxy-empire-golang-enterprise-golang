package main

var CombatShips = []ShipCombatConfig{
	{Type: "light_fighter", Strength: 5, Shield: 10, Attack: 50, Metal: 3000, Crystal: 1000, Cargo: 50,
		CarTargets: map[string]int{"espionage_probe": 5, "solar_satellite": 5}},
	{Type: "heavy_fighter", Strength: 15, Shield: 25, Attack: 150, Metal: 6000, Crystal: 4000, Cargo: 100,
		CarTargets: map[string]int{"espionage_probe": 5, "solar_satellite": 5, "light_fighter": 3}},
	{Type: "cruiser", Strength: 50, Shield: 50, Attack: 400, Metal: 20000, Crystal: 7000, Gas: 2000, Cargo: 800,
		CarTargets: map[string]int{"espionage_probe": 5, "solar_satellite": 5, "light_fighter": 6}},
	{Type: "battleship", Strength: 200, Shield: 200, Attack: 1000, Metal: 45000, Crystal: 15000, Cargo: 1500,
		CarTargets: map[string]int{"espionage_probe": 5, "solar_satellite": 5}},
	{Type: "dreadnought", Strength: 700, Shield: 500, Attack: 4000, Metal: 90000, Crystal: 45000, Gas: 15000, Cargo: 2500,
		CarTargets: map[string]int{"heavy_fighter": 4, "cruiser": 4, "battleship": 7}},
	{Type: "bomber", Strength: 500, Shield: 500, Attack: 1000, Metal: 50000, Crystal: 25000, Gas: 15000, Cargo: 500,
		CarTargets: map[string]int{"espionage_probe": 5, "solar_satellite": 5},
		DefenseDamageMultiplier: 2.0},
	{Type: "cargo", Strength: 5, Shield: 5, Attack: 3, Metal: 2000, Crystal: 2000, Cargo: 25000},
	{Type: "large_cargo", Strength: 10, Shield: 10, Attack: 5, Metal: 6000, Crystal: 6000, Cargo: 100000},
	{Type: "recycler", Strength: 15, Shield: 10, Attack: 1, Metal: 10000, Crystal: 6000, Gas: 2000, Cargo: 20000},
	{Type: "espionage_probe", Strength: 1, Shield: 1, Attack: 1, Crystal: 1000, Cargo: 5},
	{Type: "colony_ship", Strength: 30, Shield: 30, Attack: 15, Metal: 10000, Crystal: 20000, Gas: 10000, Cargo: 7500},
	{Type: "solar_satellite", Strength: 1, Shield: 1, Attack: 1, Crystal: 2000, Gas: 500, Cargo: 0},
}

var deathPriority = []string{
	"light_fighter",
	"heavy_fighter",
	"cruiser",
	"battleship",
	"dreadnought",
	"bomber",
	"cargo",
	"large_cargo",
	"recycler",
	"espionage_probe",
	"colony_ship",
	"solar_satellite",
}

var shipStatsMap map[string]ShipCombatConfig

func init() {
	shipStatsMap = make(map[string]ShipCombatConfig, len(CombatShips))
	for _, s := range CombatShips {
		if s.CarTargets == nil {
			s.CarTargets = make(map[string]int)
		}
		if s.DefenseDamageMultiplier == 0 {
			s.DefenseDamageMultiplier = 1.0
		}
		shipStatsMap[s.Type] = s
	}
}

func shipCombatConfig(shipType string) (ShipCombatConfig, bool) {
	s, ok := shipStatsMap[shipType]
	return s, ok
}

func totalCargoCapacity(ships map[string]int) int {
	total := 0
	for shipType, qty := range ships {
		if cfg, ok := shipCombatConfig(shipType); ok {
			total += qty * cfg.Cargo
		}
	}
	return total
}
