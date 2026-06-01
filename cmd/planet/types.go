package main

import "time"

type MoonBuilding struct {
	ID         int    `json:"id"`
	MoonGalaxy int    `json:"moon_galaxy"`
	MoonSystem int    `json:"moon_system"`
	MoonPos    int    `json:"moon_position"`
	Type       string `json:"type"`
	Level      int    `json:"level"`
}

type WormholeEntry struct {
	ID             int        `json:"id"`
	MoonGalaxy     int        `json:"moon_galaxy"`
	MoonSystem     int        `json:"moon_system"`
	MoonPos        int        `json:"moon_position"`
	Level          int        `json:"level"`
	LinkedGalaxy   *int       `json:"linked_galaxy,omitempty"`
	LinkedSystem   *int       `json:"linked_system,omitempty"`
	LinkedPosition *int       `json:"linked_position,omitempty"`
	CooldownUntil  *time.Time `json:"cooldown_until,omitempty"`
}

type MoonBuildingsResponse struct {
	Buildings  []MoonBuilding `json:"buildings"`
	MaxFields  int            `json:"max_fields"`
	FieldsUsed int            `json:"fields_used"`
}

const baseMoonFields = 20
const moonBaseFieldsPerLevel = 3

type Planet struct {
	ID                 int
	UserID             int
	Name               string
	Metal              int
	Crystal            int
	Gas                int
	Energy             int
	Galaxy             int
	System             int
	Position           int
	MaxFields          int
	Type               string
	Temperature        int
	ResourcesUpdatedAt time.Time
}

type Building struct {
	ID       int    `json:"id"`
	PlanetID int    `json:"planet_id"`
	Type     string `json:"type"`
	Level    int    `json:"level"`
}

type Production struct {
	Metal   float64 `json:"metal"`
	Crystal float64 `json:"crystal"`
	Gas     float64 `json:"gas"`
	Energy  float64 `json:"energy"`
}

type Storage struct {
	Metal   int `json:"metal"`
	Crystal int `json:"crystal"`
	Gas     int `json:"gas"`
}

type QueueEntry struct {
	ID           int       `json:"id"`
	BuildingType string    `json:"building_type"`
	TargetLevel  int       `json:"target_level"`
	Status       string    `json:"status"`
	CompletesAt  time.Time `json:"completes_at"`
}

type Technology struct {
	ID    int
	Type  string
	Level int
}

type PlanetResponse struct {
	ID         int          `json:"id"`
	UserID     int          `json:"user_id"`
	Name       string       `json:"name"`
	Metal      int          `json:"metal"`
	Crystal    int          `json:"crystal"`
	Gas        int          `json:"gas"`
	Energy     int          `json:"energy"`
	Galaxy     int          `json:"galaxy"`
	System     int          `json:"system"`
	Position   int          `json:"position"`
	MaxFields   int          `json:"max_fields"`
	FieldsUsed  int          `json:"fields_used"`
	Type        string       `json:"type"`
	Temperature int          `json:"temperature"`
	VIPLevel    int          `json:"vip_level"`
	Rank        int          `json:"rank"`
	Buildings   []Building   `json:"buildings"`
	Production Production   `json:"production"`
	Storage    Storage      `json:"storage"`
	Queue      []QueueEntry `json:"queue"`
}

type Galaxy struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type System struct {
	ID            int `json:"id"`
	SystemNum     int `json:"system_num"`
	OccupiedCount int `json:"occupied_count"`
}

type Position struct {
	PositionNum int    `json:"position"`
	State       string `json:"state"`
	PlanetName  string `json:"planet_name,omitempty"`
	PlayerID    int    `json:"player_id,omitempty"`
}

type ShipResponse struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Metal    int    `json:"metal"`
	Crystal  int    `json:"crystal"`
	Gas      int    `json:"gas"`
	Speed    int    `json:"speed"`
	Cargo    int    `json:"cargo"`
	Fuel     int    `json:"fuel"`
	Strength int    `json:"strength"`
	Shield   int    `json:"shield"`
	Attack   int    `json:"attack"`
	Quantity int    `json:"quantity"`
}

type BuildRequest struct {
	ShipType string `json:"ship_type"`
	Quantity int    `json:"quantity"`
}

type DefenseResponse struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Metal    int    `json:"metal"`
	Crystal  int    `json:"crystal"`
	Gas      int    `json:"gas"`
	Strength int    `json:"strength"`
	Shield   int    `json:"shield"`
	Attack   int    `json:"attack"`
	Fields   int    `json:"fields"`
	Quantity int    `json:"quantity"`
}

type DefenseBuildRequest struct {
	DefenseType string `json:"defense_type"`
	Quantity    int    `json:"quantity"`
}

type StarGateLink struct {
	ID              int  `json:"id"`
	PlanetID        int  `json:"planet_id"`
	TargetPlanetID  int  `json:"target_planet_id"`
}

type ShieldDomeConfig struct {
	Type        string
	Name        string
	ShieldHP    int
	CostMetal   int
	CostCrystal int
	CostGas     int
}

type MissileCounts struct {
	IPMs int `json:"ipms"`
	ABMs int `json:"abms"`
}

type BuildMissileRequest struct {
	Count int `json:"count"`
}

type LaunchIPMRequest struct {
	TargetGalaxy   int `json:"target_galaxy"`
	TargetSystem   int `json:"target_system"`
	TargetPosition int `json:"target_position"`
	Count          int `json:"count"`
}

// Gems
type GemSlot struct {
	ID        int    `json:"id"`
	PlanetID  int    `json:"planet_id"`
	SlotIndex int    `json:"slot_index"`
	GemType   string `json:"gem_type,omitempty"`
	StarLevel int    `json:"star_level"`
}

type GemBonuses struct {
	AttackBonus  float64 `json:"attack_bonus"`
	ArmorBonus   float64 `json:"armor_bonus"`
	StrengthBonus float64 `json:"strength_bonus"`
}

// Galactonite Shards
type GalactoniteShards struct {
	PlayerID        int    `json:"player_id"`
	GemType         string `json:"gem_type"`
	Count           int    `json:"count"`
	CombineAttempts int    `json:"combine_attempts"`
}

// NPC Planets
type NPCPlanet struct {
	ID          int        `json:"id"`
	PlanetID    int        `json:"planet_id"`
	Galaxy      int        `json:"galaxy"`
	System      int        `json:"system"`
	Position    int        `json:"position"`
	Status      string     `json:"status"`
	RespawnsAt  *time.Time `json:"respawns_at,omitempty"`
}

type EquipGemRequest struct {
	SlotIndex int    `json:"slot_index"`
	GemType   string `json:"gem_type"`
}

type CombineGemRequest struct {
	SlotIndex int    `json:"slot_index"`
	GemType   string `json:"gem_type"`
}
