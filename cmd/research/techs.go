package main

var Techs = []TechConfig{
	// === Basic Tech (5) ===
	{
		Type: "energy_tech", Name: "Energy Technology", Category: "basic",
		CostMetal: 800, CostCrystal: 400, CostGas: 0, CostFactor: 2,
		Prerequisites: []Prereq{{Type: "research_lab", Level: 1}},
		Description: "Reduces building energy consumption, increases Fusion Reactor output by 5% per level",
		Effect:      "+5% Fusion Reactor output per level, reduces building energy consumption",
	},
	{
		Type: "laser_tech", Name: "Laser Technology", Category: "basic",
		CostMetal: 200, CostCrystal: 100, CostGas: 0, CostFactor: 2,
		Prerequisites: []Prereq{{Type: "research_lab", Level: 1}, {Type: "energy_tech", Level: 2}},
		Description:   "Unlocks ships and defenses, provides +10% weapon attack per level to all ships",
		Effect:         "+10% weapon attack per level, unlocks ships and defenses",
	},
	{
		Type: "ion_tech", Name: "Ion Technology", Category: "basic",
		CostMetal: 1000, CostCrystal: 300, CostGas: 100, CostFactor: 2,
		Prerequisites: []Prereq{{Type: "research_lab", Level: 2}, {Type: "laser_tech", Level: 2}},
		Description:   "Unlocks advanced ships and defenses",
		Effect:         "Unlocks advanced ships and defenses",
	},
	{
		Type: "hyperspace_tech", Name: "Hyperspace Technology", Category: "basic",
		CostMetal: 0, CostCrystal: 4000, CostGas: 2000, CostFactor: 2,
		Prerequisites: []Prereq{{Type: "research_lab", Level: 3}, {Type: "energy_tech", Level: 3}},
		Description:   "Unlocks advanced ships and defenses",
		Effect:         "Unlocks advanced ships and defenses",
	},
	{
		Type: "plasma_tech", Name: "Plasma Technology", Category: "basic",
		CostMetal: 2000, CostCrystal: 4000, CostGas: 1000, CostFactor: 2,
		Prerequisites: []Prereq{{Type: "research_lab", Level: 4}, {Type: "laser_tech", Level: 5}, {Type: "ion_tech", Level: 3}},
		Description:   "Unlocks advanced ships and defenses",
		Effect:         "Unlocks advanced ships and defenses",
	},
	// === Advanced Tech (5) ===
	{
		Type: "astrophysics", Name: "Astrophysics", Category: "advanced",
		CostMetal: 4000, CostCrystal: 8000, CostGas: 4000, CostFactor: 2,
		Prerequisites: []Prereq{{Type: "research_lab", Level: 3}, {Type: "energy_tech", Level: 4}},
		Description:   "Increases number of planets, unlocks additional colony slots",
		Effect:         "+1 planet per 2 levels, +1 nebula fleet at levels 4/9/12",
	},
	{
		Type: "computer_tech", Name: "Computer Technology", Category: "advanced",
		CostMetal: 4000, CostCrystal: 2000, CostGas: 2000, CostFactor: 2,
		Prerequisites: []Prereq{{Type: "research_lab", Level: 1}},
		Description:   "Increases fleet command slots, allows more simultaneous attacks",
		Effect:         "+2 fleet slots per level",
	},
	{
		Type: "espionage_tech", Name: "Espionage Technology", Category: "advanced",
		CostMetal: 0, CostCrystal: 2000, CostGas: 1000, CostFactor: 2,
		Prerequisites: []Prereq{{Type: "research_lab", Level: 1}},
		Description:   "Improves espionage probe scan details and reduces detection chance",
		Effect:         "+detail level in spy reports, +nebula rewards",
	},
	{
		Type: "ultra_temperature", Name: "Ultra Temperature Technology", Category: "advanced",
		CostMetal: 6000, CostCrystal: 3000, CostGas: 3000, CostFactor: 2,
		Prerequisites: []Prereq{{Type: "research_lab", Level: 5}, {Type: "energy_tech", Level: 4}},
		Description:   "Improves planet temperature for better resource production",
		Effect:         "+5% to production per level when temperature extreme",
	},
	{
		Type: "anti_gravity", Name: "Anti-Gravity Technology", Category: "advanced",
		CostMetal: 10000, CostCrystal: 5000, CostGas: 5000, CostFactor: 2,
		Prerequisites: []Prereq{{Type: "research_lab", Level: 7}, {Type: "hyperspace_tech", Level: 3}},
		Description:   "Unlocks advanced ships and reduces building costs",
		Effect:         "Enables advanced buildings and technologies",
	},
	// === Combat Tech (6) ===
	{
		Type: "combustion_drive", Name: "Combustion Drive", Category: "combat",
		CostMetal: 400, CostCrystal: 200, CostGas: 0, CostFactor: 2,
		Prerequisites: []Prereq{{Type: "research_lab", Level: 3}, {Type: "energy_tech", Level: 2}},
		Description:   "Increases speed of small cargo ships, light fighters, and recyclers",
		Effect:         "+30% speed per level for combustion ships",
	},
	{
		Type: "impulse_drive", Name: "Impulse Drive", Category: "combat",
		CostMetal: 2000, CostCrystal: 4000, CostGas: 600, CostFactor: 2,
		Prerequisites: []Prereq{{Type: "research_lab", Level: 3}, {Type: "energy_tech", Level: 4}, {Type: "combustion_drive", Level: 3}},
		Description:   "Increases speed of cruisers, heavy fighters, and colony ships",
		Effect:         "+30% speed per level for impulse ships",
	},
	{
		Type: "hyperspace_drive", Name: "Hyperspace Drive", Category: "combat",
		CostMetal: 10000, CostCrystal: 20000, CostGas: 6000, CostFactor: 2,
		Prerequisites: []Prereq{{Type: "research_lab", Level: 4}, {Type: "hyperspace_tech", Level: 3}},
		Description:   "Increases speed of battleships, bombers, and large cargo ships",
		Effect:         "+30% speed per level for hyperspace ships",
	},
	{
		Type: "weapons_tech", Name: "Weapons Technology", Category: "combat",
		CostMetal: 800, CostCrystal: 200, CostGas: 0, CostFactor: 2,
		Prerequisites: []Prereq{{Type: "research_lab", Level: 4}, {Type: "laser_tech", Level: 2}},
		Description:   "Increases attack power of all ships and defenses by 10% per level",
		Effect:         "+10% attack per level to ALL ships",
	},
	{
		Type: "shielding_tech", Name: "Shielding Technology", Category: "combat",
		CostMetal: 200, CostCrystal: 600, CostGas: 0, CostFactor: 2,
		Prerequisites: []Prereq{{Type: "research_lab", Level: 6}, {Type: "ion_tech", Level: 3}},
		Description:   "Increases shield strength of all ships and defenses by 10% per level",
		Effect:         "+10% shield per level to ALL ships",
	},
	{
		Type: "strength_tech", Name: "Strength Technology", Category: "combat",
		CostMetal: 1000, CostCrystal: 300, CostGas: 0, CostFactor: 2,
		Prerequisites: []Prereq{{Type: "research_lab", Level: 5}, {Type: "shielding_tech", Level: 3}},
		Description:   "Increases hull strength of all ships by 10% per level",
		Effect:         "+10% hull per level to ALL ships",
	},
	// === Moon Tech (3) ===
	{
		Type: "alloy_detection_tech", Name: "Alloy Detection Technology", Category: "moon",
		CostMetal: 10000, CostCrystal: 20000, CostGas: 5000, CostFactor: 2,
		Prerequisites: []Prereq{},
		Effect:        "Increases debris yield by 10% per level",
		Description:   "Research at the Pioneer Lab to improve debris recovery from destroyed ships.",
	},
	{
		Type: "dynamic_power_tech", Name: "Dynamic Power Technology", Category: "moon",
		CostMetal: 20000, CostCrystal: 10000, CostGas: 10000, CostFactor: 2,
		Prerequisites: []Prereq{{Type: "alloy_detection_tech", Level: 3}},
		Effect:        "Increases moon energy output per level",
		Description:   "Research at the Pioneer Lab to improve moon energy efficiency.",
	},
	{
		Type: "combined_guidance_tech", Name: "Combined Guidance Technology", Category: "moon",
		CostMetal: 50000, CostCrystal: 30000, CostGas: 20000, CostFactor: 2,
		Prerequisites: []Prereq{{Type: "dynamic_power_tech", Level: 3}},
		Effect:        "Enhances fleet command and control capabilities",
		Description:   "Advanced moon research that enhances fleet coordination.",
	},
}

func init() {
	for i := range Techs {
		if Techs[i].Category == "moon" {
			Techs[i].ResearchLocation = "pioneer_lab"
		} else {
			Techs[i].ResearchLocation = "research_lab"
		}
	}
}

func techConfig(techType string) (TechConfig, bool) {
	for _, t := range Techs {
		if t.Type == techType {
			return t, true
		}
	}
	return TechConfig{}, false
}
