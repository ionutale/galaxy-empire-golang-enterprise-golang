package main

import "time"

type TutorialStep struct {
	ID              int            `json:"id"`
	Title           string         `json:"title"`
	Description     string         `json:"description"`
	Action          string         `json:"action"`
	Target          string         `json:"target"`
	RewardDM        int            `json:"reward_dm"`
	RewardResources map[string]int `json:"reward_resources"`
}

type PlayerTutorial struct {
	PlayerID    int
	CurrentStep int
	Completed   bool
	StartedAt   time.Time
	CompletedAt *time.Time
}

type TutorialStatusResponse struct {
	CurrentStep int            `json:"current_step"`
	Completed   bool           `json:"completed"`
	Steps       []TutorialStep `json:"steps"`
}

type ClaimRewardResponse struct {
	Step            int            `json:"step"`
	RewardDM        int            `json:"reward_dm"`
	RewardResources map[string]int `json:"reward_resources"`
	NextStep        int            `json:"next_step"`
	Completed       bool           `json:"completed"`
}

var TutorialSteps = []TutorialStep{
	{ID: 1, Title: "First Mine", Description: "Build your first Metal Mine", Action: "upgrade_building", Target: "metal_mine", RewardDM: 3, RewardResources: map[string]int{"metal": 1000}},
	{ID: 2, Title: "Power Up", Description: "Build a Solar Plant", Action: "upgrade_building", Target: "solar_plant", RewardDM: 3, RewardResources: map[string]int{"metal": 500}},
	{ID: 3, Title: "Crystal Clear", Description: "Build a Crystal Mine", Action: "upgrade_building", Target: "crystal_mine", RewardDM: 3, RewardResources: map[string]int{"crystal": 1000}},
	{ID: 4, Title: "Ship Builder", Description: "Build a Shipyard", Action: "upgrade_building", Target: "shipyard", RewardDM: 4, RewardResources: map[string]int{"metal": 2000}},
	{ID: 5, Title: "First Cargo", Description: "Build a Cargo Ship", Action: "build_ship", Target: "cargo", RewardDM: 4, RewardResources: map[string]int{"crystal": 500}},
	{ID: 6, Title: "Nebula Explorer", Description: "Send a Nebula Expedition", Action: "send_expedition", Target: "nebula", RewardDM: 5, RewardResources: map[string]int{"metal": 3000, "crystal": 1000}},
	{ID: 7, Title: "First Attack", Description: "Launch an Attack", Action: "launch_attack", Target: "fleet", RewardDM: 5, RewardResources: map[string]int{"metal": 5000, "crystal": 2000}},
}

type ProgressUpdateRequest struct {
	PlayerID int    `json:"player_id"`
	StepID   int    `json:"step_id"`
}
