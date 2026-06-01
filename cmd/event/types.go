package main

import "time"

type Event struct {
	ID          int               `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	EventType   string            `json:"event_type"`
	Modifiers   map[string]any    `json:"modifiers"`
	StartsAt    time.Time         `json:"starts_at"`
	EndsAt      time.Time         `json:"ends_at"`
	Status      string            `json:"status"`
	CreatedAt   time.Time         `json:"created_at"`
}

type EventParticipation struct {
	ID             int            `json:"id"`
	PlayerID       int            `json:"player_id"`
	EventID        int            `json:"event_id"`
	Progress       map[string]any `json:"progress"`
	Completed      bool           `json:"completed"`
	RewardsClaimed bool           `json:"rewards_claimed"`
	JoinedAt       time.Time      `json:"joined_at"`
}

type EventResponse struct {
	ID          int               `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	EventType   string            `json:"event_type"`
	Modifiers   map[string]any    `json:"modifiers"`
	StartsAt    time.Time         `json:"starts_at"`
	EndsAt      time.Time         `json:"ends_at"`
	Status      string            `json:"status"`
	Joined      bool              `json:"joined,omitempty"`
	Completed   bool              `json:"completed,omitempty"`
	RewardsClaimed bool           `json:"rewards_claimed,omitempty"`
}

var eventTypeModifiers = map[string]map[string]any{
	"double_resources":    {"resource_multiplier": 2.0},
	"special_npc":        {"npc_loot_multiplier": 2.0},
	"reduced_build_time":  {"build_time_multiplier": 0.5},
	"combat_bonus":       {"attack_bonus": 0.25},
	"expedition_bonus":   {"expedition_dm_multiplier": 2.0},
}
