package main

type ShipConfig struct {
	Type      string
	Speed     int
	Fuel      int
	Cargo     int
	DriveTier string
}

var Ships = []ShipConfig{
	{Type: "cargo", Speed: 7500, Fuel: 500, Cargo: 25000, DriveTier: "combustion"},
	{Type: "large_cargo", Speed: 7500, Fuel: 1500, Cargo: 100000, DriveTier: "combustion"},
	{Type: "recycler", Speed: 2000, Fuel: 800, Cargo: 20000, DriveTier: "combustion"},
	{Type: "espionage_probe", Speed: 100000000, Fuel: 1, Cargo: 5, DriveTier: "combustion"},
	{Type: "colony_ship", Speed: 2500, Fuel: 2000, Cargo: 7500, DriveTier: "combustion"},
	{Type: "solar_satellite", Speed: 0, Fuel: 0, Cargo: 0, DriveTier: "combustion"},
	{Type: "light_fighter", Speed: 12500, Fuel: 20, Cargo: 50, DriveTier: "combustion"},
	{Type: "heavy_fighter", Speed: 10000, Fuel: 75, Cargo: 100, DriveTier: "combustion"},
	{Type: "cruiser", Speed: 15000, Fuel: 300, Cargo: 800, DriveTier: "impulse"},
	{Type: "battleship", Speed: 10000, Fuel: 1000, Cargo: 1500, DriveTier: "hyperspace"},
	{Type: "dreadnought", Speed: 5000, Fuel: 2000, Cargo: 2500, DriveTier: "hyperspace"},
	{Type: "bomber", Speed: 4000, Fuel: 1000, Cargo: 500, DriveTier: "impulse"},
}

func shipConfig(shipType string) (ShipConfig, bool) {
	for _, s := range Ships {
		if s.Type == shipType {
			return s, true
		}
	}
	return ShipConfig{}, false
}

func minShipSpeed(ships map[string]int) (int, bool) {
	min := int(^uint(0) >> 1)
	onlyBomber := true
	for shipType, qty := range ships {
		if qty == 0 {
			continue
		}
		cfg, ok := shipConfig(shipType)
		if !ok {
			continue
		}
		if cfg.Speed > 0 && cfg.Speed < min {
			min = cfg.Speed
		}
		if shipType != "bomber" {
			onlyBomber = false
		}
	}
	if min == int(^uint(0)>>1) {
		return 0, false
	}
	return min, onlyBomber
}

func effectiveMinSpeed(ships map[string]int, techLevels map[string]int) int {
	min := int(^uint(0) >> 1)
	for shipType, qty := range ships {
		if qty == 0 {
			continue
		}
		cfg, ok := shipConfig(shipType)
		if !ok || cfg.Speed <= 0 {
			continue
		}
		speed := cfg.Speed
		boost := 1.0
		if lvl, ok := techLevels[cfg.DriveTier+"_drive"]; ok && lvl > 0 {
			boost = 1.0 + float64(lvl)*0.3
		}
		eff := int(float64(speed) * boost)
		if eff < min {
			min = eff
		}
	}
	if min == int(^uint(0)>>1) {
		return 0
	}
	return min
}

func distance(g1, s1, p1, g2, s2, p2 int) int {
	return abs(g1-g2)*20000 + abs(s1-s2)*95 + abs(p1-p2)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
