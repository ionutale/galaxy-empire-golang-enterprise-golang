package main

import "time"

type RadarEvent struct {
	ID             int
	PlayerID       int
	EventType      string
	SourcePlayerID *int
	FleetID        *int
	TargetGalaxy   int
	TargetSystem   int
	TargetPosition int
	OriginGalaxy   *int
	OriginSystem   *int
	OriginPosition *int
	ArrivalTime    *time.Time
	DetectedAt     time.Time
	Resolved       bool
}

type EuxRadar struct {
	ID       int
	PlayerID int
	Galaxy   int
	System   int
	Position int
	Level    int
}

type EventsRequest struct {
	Scope string `json:"scope"`
}

type ResolveEventRequest struct {
	EventID int `json:"event_id"`
}

type EUXScanRequest struct {
	TargetGalaxy   int `json:"target_galaxy"`
	TargetSystem   int `json:"target_system"`
	TargetPosition int `json:"target_position"`
}

type DetectFleetRequest struct {
	TargetPlayerID  int    `json:"target_player_id"`
	SourcePlayerID  int    `json:"source_player_id"`
	FleetID         int    `json:"fleet_id"`
	TargetGalaxy    int    `json:"target_galaxy"`
	TargetSystem    int    `json:"target_system"`
	TargetPosition  int    `json:"target_position"`
	OriginGalaxy    int    `json:"origin_galaxy"`
	OriginSystem    int    `json:"origin_system"`
	OriginPosition  int    `json:"origin_position"`
	ArrivalTime     string `json:"arrival_time"`
	Mission         string `json:"mission"`
}

type RadarEventResponse struct {
	ID             int        `json:"id"`
	EventType      string     `json:"event_type"`
	SourcePlayerID *int       `json:"source_player_id,omitempty"`
	FleetID        *int       `json:"fleet_id,omitempty"`
	TargetGalaxy   int        `json:"target_galaxy"`
	TargetSystem   int        `json:"target_system"`
	TargetPosition int        `json:"target_position"`
	OriginGalaxy   *int       `json:"origin_galaxy,omitempty"`
	OriginSystem   *int       `json:"origin_system,omitempty"`
	OriginPosition *int       `json:"origin_position,omitempty"`
	ArrivalTime    *time.Time `json:"arrival_time,omitempty"`
	DetectedAt     time.Time  `json:"detected_at"`
	Resolved       bool       `json:"resolved"`
}

type PlanetStatusResponse struct {
	PlanetID   int    `json:"planet_id"`
	Status     string `json:"status"`
	FleetCount int    `json:"fleet_count"`
}

type EUXScanResponse struct {
	Fleets []FleetInfo `json:"fleets"`
}

type FleetInfo struct {
	ID       int            `json:"id"`
	Ships    map[string]int `json:"ships"`
	Mission  string         `json:"mission"`
	ArrivesAt *time.Time    `json:"arrives_at,omitempty"`
}

type PlanetCoords struct {
	PlanetID int `json:"planet_id"`
	Galaxy   int `json:"galaxy"`
	System   int `json:"system"`
	Position int `json:"position"`
}
