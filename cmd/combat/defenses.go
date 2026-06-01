package main

type DefenseCombatConfig struct {
	Type     string
	Strength int
	Shield   int
	Attack   int
	Metal    int
	Crystal  int
	Gas      int
}

var CombatDefenses = []DefenseCombatConfig{
	{Type: "rocket_launcher", Strength: 10, Shield: 20, Attack: 80, Metal: 2000},
	{Type: "light_laser", Strength: 10, Shield: 25, Attack: 100, Metal: 1500, Crystal: 500},
	{Type: "heavy_laser", Strength: 100, Shield: 100, Attack: 250, Metal: 6000, Crystal: 2000},
	{Type: "mk2_cannon", Strength: 300, Shield: 500, Attack: 700, Metal: 45000, Crystal: 25000, Gas: 15000},
	{Type: "ion_cannon", Strength: 500, Shield: 5000, Attack: 150, Metal: 4000, Crystal: 8000, Gas: 1000},
	{Type: "plasma_cannon", Strength: 1000, Shield: 300, Attack: 3000, Metal: 100000, Crystal: 50000, Gas: 30000},
	{Type: "proton_cannon", Strength: 2000, Shield: 500, Attack: 5000, Metal: 250000, Crystal: 100000, Gas: 50000},
}

var defenseStatsMap map[string]DefenseCombatConfig

func init() {
	defenseStatsMap = make(map[string]DefenseCombatConfig, len(CombatDefenses))
	for _, d := range CombatDefenses {
		defenseStatsMap[d.Type] = d
	}
}

func defenseConfig(defenseType string) (DefenseCombatConfig, bool) {
	d, ok := defenseStatsMap[defenseType]
	return d, ok
}
