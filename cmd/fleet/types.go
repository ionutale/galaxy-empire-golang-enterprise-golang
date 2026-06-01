package main

import "time"

type Fleet struct {
	ID              int
	PlayerID        int
	OriginPlanetID  int
	TargetGalaxy    int
	TargetSystem    int
	TargetPosition  int
	Mission         string
	Status          string
	SpeedPct        int
	Ships           map[string]int
	ArrivesAt       time.Time
	CreatedAt       time.Time
	AllianceGroupID int
}

type FleetResponse struct {
	ID              int            `json:"id"`
	PlayerID        int            `json:"player_id"`
	OriginPlanetID  int            `json:"origin_planet_id"`
	TargetGalaxy    int            `json:"target_galaxy"`
	TargetSystem    int            `json:"target_system"`
	TargetPosition  int            `json:"target_position"`
	Mission         string         `json:"mission"`
	Status          string         `json:"status"`
	SpeedPct        int            `json:"speed_pct"`
	Ships           map[string]int `json:"ships"`
	ArrivesAt       *time.Time     `json:"arrives_at,omitempty"`
	AllianceGroupID int            `json:"alliance_group_id"`
}

type DispatchRequest struct {
	OriginPlanetID  int            `json:"origin_planet_id"`
	Ships           map[string]int `json:"ships"`
	TargetGalaxy    int            `json:"target_galaxy"`
	TargetSystem    int            `json:"target_system"`
	TargetPosition  int            `json:"target_position"`
	Mission         string         `json:"mission"`
	SpeedPct        int            `json:"speed_pct"`
	AllianceGroupID int            `json:"alliance_group_id"`
}

type DebrisField struct {
	ID       int
	Galaxy   int
	System   int
	Position int
	Metal    int
	Crystal  int
}
