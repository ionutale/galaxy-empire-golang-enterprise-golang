package main

import "time"

type EspionageReport struct {
	ID              int
	PlayerID        int
	TargetPlayerID  int
	TargetGalaxy    int
	TargetSystem    int
	TargetPosition  int
	DetailLevel     int
	Resources       map[string]int
	Fleet           map[string]int
	Defense         map[string]int
	Tech            map[string]int
	ReportData      map[string]any
	CreatedAt       time.Time
	ExpiresAt       time.Time
}

type EspionageReportResponse struct {
	ID              int                    `json:"id"`
	PlayerID        int                    `json:"player_id"`
	TargetPlayerID  int                    `json:"target_player_id"`
	TargetGalaxy    int                    `json:"target_galaxy"`
	TargetSystem    int                    `json:"target_system"`
	TargetPosition  int                    `json:"target_position"`
	DetailLevel     int                    `json:"detail_level"`
	Resources       map[string]int         `json:"resources,omitempty"`
	Fleet           map[string]int         `json:"fleet,omitempty"`
	Defense         map[string]int         `json:"defense,omitempty"`
	Tech            map[string]int         `json:"tech,omitempty"`
	ReportData      map[string]any         `json:"report_data,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	ExpiresAt       time.Time              `json:"expires_at"`
}

type ProbeRequest struct {
	TargetGalaxy   int `json:"target_galaxy"`
	TargetSystem   int `json:"target_system"`
	TargetPosition int `json:"target_position"`
	PlanetID       int `json:"planet_id"`
}

type ProbeResponse struct {
	ReportID  int       `json:"report_id"`
	CreatedAt time.Time `json:"created_at"`
}

type PlanetInfo struct {
	PlanetID int           `json:"planet_id"`
	PlayerID int           `json:"player_id"`
	Metal    int           `json:"metal"`
	Crystal  int           `json:"crystal"`
	Gas      int           `json:"gas"`
	Ships    map[string]int `json:"ships"`
}
