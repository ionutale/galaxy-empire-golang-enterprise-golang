package main

import "time"

type PlayerScore struct {
	ID             int       `json:"id"`
	PlayerID       int       `json:"player_id"`
	PlayerName     string    `json:"player_name"`
	TotalScore     int       `json:"total_score"`
	FleetScore     int       `json:"fleet_score"`
	BuildingsScore int       `json:"buildings_score"`
	ResearchScore  int       `json:"research_score"`
	DefenseScore   int       `json:"defense_score"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type RankEntry struct {
	Rank         int       `json:"rank"`
	PlayerID     int       `json:"player_id"`
	PlayerName   string    `json:"player_name"`
	TotalScore   int       `json:"total_score"`
	FleetScore   int       `json:"fleet_score"`
	BuildingsScore int     `json:"buildings_score"`
	ResearchScore int      `json:"research_score"`
	DefenseScore int       `json:"defense_score"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type TopResponse struct {
	Page    int         `json:"page"`
	PerPage int         `json:"per_page"`
	Total   int         `json:"total"`
	Ranking []RankEntry `json:"ranking"`
}

type PlayerRankResponse struct {
	Rank       int    `json:"rank"`
	PlayerID   int    `json:"player_id"`
	PlayerName string `json:"player_name"`
	PlayerScore
}

type UpdateScoreRequest struct {
	PlayerID       int    `json:"player_id"`
	PlayerName     string `json:"player_name"`
	BuildingsScore *int   `json:"buildings_score,omitempty"`
	ResearchScore  *int   `json:"research_score,omitempty"`
	FleetScore     *int   `json:"fleet_score,omitempty"`
	DefenseScore   *int   `json:"defense_score,omitempty"`
}

type RecalcRequest struct {
	PlayerID *int `json:"player_id,omitempty"`
}
