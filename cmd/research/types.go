package main

import "time"

type TechConfig struct {
	Type             string   `json:"type"`
	Name             string   `json:"name"`
	Category         string   `json:"category"`
	CostMetal        int      `json:"cost_metal"`
	CostCrystal      int      `json:"cost_crystal"`
	CostGas          int      `json:"cost_gas"`
	CostFactor       float64  `json:"cost_factor"`
	Prerequisites    []Prereq `json:"prerequisites"`
	Description      string   `json:"description"`
	Effect           string   `json:"effect"`
	ResearchLocation string   `json:"research_location"`
}

type Prereq struct {
	Type  string `json:"type"`
	Level int    `json:"level"`
}

type ResearchQueue struct {
	ID          int       `json:"id"`
	PlayerID    int       `json:"player_id"`
	TechType    string    `json:"tech_type"`
	TargetLevel int       `json:"target_level"`
	StartedAt   time.Time `json:"started_at"`
	CompletesAt time.Time `json:"completes_at"`
	Completed   bool      `json:"completed"`
	Cancelled   bool      `json:"cancelled"`
}

type TechWithStatus struct {
	Type             string   `json:"type"`
	Name             string   `json:"name"`
	Category         string   `json:"category"`
	Level            int      `json:"level"`
	CostMetal        int      `json:"cost_metal"`
	CostCrystal      int      `json:"cost_crystal"`
	CostGas          int      `json:"cost_gas"`
	Researching      bool     `json:"researching"`
	Prerequisites    []Prereq `json:"prerequisites"`
	Description      string   `json:"description"`
	Effect           string   `json:"effect"`
	ResearchLocation string   `json:"research_location"`
}

type StartResearchRequest struct {
	PlanetID int `json:"planet_id"`
}

type StartResearchResponse struct {
	TechType    string    `json:"tech_type"`
	TargetLevel int       `json:"target_level"`
	CompletesAt time.Time `json:"completes_at"`
}

type CancelResearchResponse struct {
	RefundMetal   int `json:"refund_metal"`
	RefundCrystal int `json:"refund_crystal"`
	RefundGas     int `json:"refund_gas"`
}
