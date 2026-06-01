package main

type ShipConfig struct {
	Type     string
	Name     string
	Metal    int
	Crystal  int
	Gas      int
	Speed    int
	Cargo    int
	Fuel     int
	Strength int
	Shield   int
	Attack   int
}

var Ships = []ShipConfig{
	{Type: "cargo", Name: "Cargo", Metal: 2000, Crystal: 2000, Speed: 7500, Cargo: 25000, Fuel: 500, Strength: 5, Shield: 5, Attack: 3},
	{Type: "large_cargo", Name: "Large Cargo", Metal: 6000, Crystal: 6000, Speed: 7500, Cargo: 100000, Fuel: 1500, Strength: 10, Shield: 10, Attack: 5},
	{Type: "recycler", Name: "Recycler", Metal: 10000, Crystal: 6000, Gas: 2000, Speed: 2000, Cargo: 20000, Fuel: 800, Strength: 15, Shield: 10, Attack: 1},
	{Type: "espionage_probe", Name: "Espionage Probe", Crystal: 1000, Speed: 100000000, Cargo: 5, Fuel: 1, Strength: 1, Shield: 1, Attack: 1},
	{Type: "colony_ship", Name: "Colony Ship", Metal: 10000, Crystal: 20000, Gas: 10000, Speed: 2500, Cargo: 7500, Fuel: 2000, Strength: 30, Shield: 30, Attack: 15},
	{Type: "solar_satellite", Name: "Solar Satellite", Crystal: 2000, Gas: 500, Cargo: 0, Fuel: 0, Strength: 1, Shield: 1, Attack: 1},
	{Type: "light_fighter", Name: "Light Fighter", Metal: 3000, Crystal: 1000, Speed: 12500, Cargo: 50, Fuel: 20, Strength: 5, Shield: 10, Attack: 50},
	{Type: "heavy_fighter", Name: "Heavy Fighter", Metal: 6000, Crystal: 4000, Speed: 10000, Cargo: 100, Fuel: 75, Strength: 15, Shield: 25, Attack: 150},
	{Type: "cruiser", Name: "Cruiser", Metal: 20000, Crystal: 7000, Gas: 2000, Speed: 15000, Cargo: 800, Fuel: 300, Strength: 50, Shield: 50, Attack: 400},
	{Type: "battleship", Name: "Battleship", Metal: 45000, Crystal: 15000, Speed: 10000, Cargo: 1500, Fuel: 1000, Strength: 200, Shield: 200, Attack: 1000},
	{Type: "dreadnought", Name: "Dreadnought", Metal: 90000, Crystal: 45000, Gas: 15000, Speed: 5000, Cargo: 2500, Fuel: 2000, Strength: 700, Shield: 500, Attack: 4000},
	{Type: "bomber", Name: "Bomber", Metal: 50000, Crystal: 25000, Gas: 15000, Speed: 4000, Cargo: 500, Fuel: 1000, Strength: 500, Shield: 500, Attack: 1000},
	{Type: "iron_behemoth", Name: "Iron Behemoth", Metal: 350000, Crystal: 4000, Gas: 5500, Speed: 7000, Cargo: 5000, Fuel: 3000, Strength: 3000, Shield: 3000, Attack: 5000},
}

func shipConfig(shipType string) (ShipConfig, bool) {
	for _, s := range Ships {
		if s.Type == shipType {
			return s, true
		}
	}
	return ShipConfig{}, false
}
