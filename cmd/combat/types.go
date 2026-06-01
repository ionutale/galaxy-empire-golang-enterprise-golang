package main

import "time"

type ShipCombatConfig struct {
	Type                    string
	Strength                int
	Shield                  int
	Attack                  int
	Metal                   int
	Crystal                 int
	Gas                     int
	Cargo                   int
	CarTargets              map[string]int
	DefenseDamageMultiplier float64
}

type CombatResult struct {
	AttackerWon        bool
	Rounds             []RoundResult
	AttackerShipsAfter map[string]int
	DefenderShipsAfter map[string]int
	DebrisMetal        int
	DebrisCrystal      int
	AttackerLoot       map[string]int
	DefenderLostRes    map[string]int
	MoonCreated        bool
	MoonSize           int
}

type Moon struct {
	ID        int       `json:"id"`
	Galaxy    int       `json:"galaxy"`
	System    int       `json:"system"`
	Position  int       `json:"position"`
	PlayerID  int       `json:"player_id"`
	Name      string    `json:"name"`
	Size      int       `json:"size"`
	CreatedAt time.Time `json:"created_at"`
}

type MoonInfo struct {
	Created bool   `json:"created"`
	Size    int    `json:"size,omitempty"`
	Name    string `json:"name,omitempty"`
}

type RoundResult struct {
	Round            int
	Wipe             bool
	AttackerShips    map[string]int
	DefenderShips    map[string]int
	AttackerLosses   map[string]int
	DefenderLosses   map[string]int
	TotalDamageDealt int
	TotalDamageTaken int
}

type MissileStrikeRequest struct {
	TargetPlanetID int    `json:"target_planet_id"`
	IPMs           int    `json:"ipms"`
	ABMDefense     int    `json:"abm_defense"`
	TechLevel      int    `json:"tech_level"`
	Defenses       map[string]int `json:"defenses,omitempty"`
}

type MissileStrikeResult struct {
	IPMsLaunched      int            `json:"ipms_launched"`
	IPMsIntercepted   int            `json:"ipms_intercepted"`
	ABMsUsed          int            `json:"abms_used"`
	DefensesDestroyed map[string]int `json:"defenses_destroyed"`
	DefensesDamaged   map[string]int `json:"defenses_damaged"`
}

type CombatReport struct {
	ID                  int
	AttackerPlayerID    int
	DefenderPlayerID    int
	TargetGalaxy        int
	TargetSystem        int
	TargetPosition      int
	AttackerShipsBefore map[string]int
	DefenderShipsBefore map[string]int
	AttackerShipsAfter  map[string]int
	DefenderShipsAfter  map[string]int
	Rounds              []RoundResult
	AttackerWon         bool
	AttackerLoot        map[string]int
	DefenderLostRes     map[string]int
	DebrisMetal         int
	DebrisCrystal       int
	MoonCreated         bool
	MoonSize            int
	MissileResult       *MissileStrikeResult `json:"missile_result,omitempty"`
	CreatedAt           time.Time
	ExpiresAt           time.Time
}
