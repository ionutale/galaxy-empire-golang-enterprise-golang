package main

import "time"

type QuestCategory string

const (
	CategoryBuilding QuestCategory = "building"
	CategoryFleet    QuestCategory = "fleet"
	CategoryCombat   QuestCategory = "combat"
	CategoryResearch QuestCategory = "research"
	CategoryNebula   QuestCategory = "nebula"
	CategorySocial   QuestCategory = "social"
	CategoryEconomy  QuestCategory = "economy"
)

type QuestStatus string

const (
	StatusLocked     QuestStatus = "locked"
	StatusAvailable  QuestStatus = "available"
	StatusInProgress QuestStatus = "in_progress"
	StatusCompleted  QuestStatus = "completed"
	StatusClaimed    QuestStatus = "claimed"
)

type QuestDefinition struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Category    QuestCategory `json:"category"`
	Requirements []QuestRequirement `json:"requirements"`
	RewardDM    int           `json:"reward_dm"`
	RewardMetal int           `json:"reward_metal,omitempty"`
	RewardCrystal int         `json:"reward_crystal,omitempty"`
	RewardGas   int           `json:"reward_gas,omitempty"`
}

type QuestRequirement struct {
	Type  string `json:"type"`
	Key   string `json:"key"`
	Value int    `json:"value"`
}

type PlayerQuest struct {
	ID              int        `json:"id"`
	PlayerID        int        `json:"player_id"`
	QuestID         string     `json:"quest_id"`
	Status          QuestStatus `json:"status"`
	ProgressCurrent int        `json:"progress_current"`
	ProgressTarget  int        `json:"progress_target"`
	StartedAt       *time.Time `json:"started_at,omitempty"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	ClaimedAt       *time.Time `json:"claimed_at,omitempty"`
}

type ListQuestsRequest struct {
	PlayerID int `json:"player_id"`
}

type ListQuestsResponse struct {
	Quests []PlayerQuestResponse `json:"quests"`
}

type PlayerQuestResponse struct {
	Definition      QuestDefinition `json:"definition"`
	Progress        PlayerQuest     `json:"progress"`
}

type ClaimRewardRequest struct {
	PlayerID int    `json:"player_id"`
	QuestID  string `json:"quest_id"`
}

type ClaimRewardResponse struct {
	QuestID     string `json:"quest_id"`
	RewardDM    int    `json:"reward_dm"`
	RewardMetal int    `json:"reward_metal,omitempty"`
	RewardCrystal int  `json:"reward_crystal,omitempty"`
	RewardGas   int    `json:"reward_gas,omitempty"`
}

type ProgressUpdateRequest struct {
	PlayerID  int                    `json:"player_id"`
	EventType string                 `json:"event_type"`
	EventData map[string]interface{} `json:"event_data"`
}

type CompletedQuestsResponse struct {
	QuestIDs []string `json:"quest_ids"`
}
