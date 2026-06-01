package main

type DefenseConfig struct {
	Type     string
	Name     string
	Metal    int
	Crystal  int
	Gas      int
	Strength int
	Shield   int
	Attack   int
	Fields   int
}

var Defenses = []DefenseConfig{
	{Type: "rocket_launcher", Name: "Rocket Launcher", Metal: 2000, Crystal: 0, Gas: 0, Strength: 10, Shield: 20, Attack: 80, Fields: 1},
	{Type: "light_laser", Name: "Light Laser", Metal: 1500, Crystal: 500, Gas: 0, Strength: 10, Shield: 25, Attack: 100, Fields: 1},
	{Type: "heavy_laser", Name: "Heavy Laser", Metal: 6000, Crystal: 2000, Gas: 0, Strength: 100, Shield: 100, Attack: 250, Fields: 1},
	{Type: "mk2_cannon", Name: "MK2 Cannon", Metal: 45000, Crystal: 25000, Gas: 15000, Strength: 300, Shield: 500, Attack: 700, Fields: 2},
	{Type: "ion_cannon", Name: "Ion Cannon", Metal: 4000, Crystal: 8000, Gas: 1000, Strength: 500, Shield: 5000, Attack: 150, Fields: 2},
	{Type: "plasma_cannon", Name: "Plasma Cannon", Metal: 100000, Crystal: 50000, Gas: 30000, Strength: 1000, Shield: 300, Attack: 3000, Fields: 3},
	{Type: "proton_cannon", Name: "Proton Cannon", Metal: 250000, Crystal: 100000, Gas: 50000, Strength: 2000, Shield: 500, Attack: 5000, Fields: 4},
}

func defenseConfig(defenseType string) (DefenseConfig, bool) {
	for _, d := range Defenses {
		if d.Type == defenseType {
			return d, true
		}
	}
	return DefenseConfig{}, false
}
