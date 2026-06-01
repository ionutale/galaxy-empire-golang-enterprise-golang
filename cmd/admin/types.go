package main

import "time"

type UserSearchResult struct {
	ID        int       `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type PlanetView struct {
	ID        int                    `json:"id"`
	UserID    int                    `json:"user_id"`
	Name      string                 `json:"name"`
	Galaxy    int                    `json:"galaxy"`
	System    int                    `json:"system"`
	Position  int                    `json:"position"`
	Metal     int                    `json:"metal"`
	Crystal   int                    `json:"crystal"`
	Gas       int                    `json:"gas"`
	Energy    int                    `json:"energy"`
	MaxFields int                    `json:"max_fields"`
	Type      string                 `json:"type"`
	Buildings []BuildingInfo         `json:"buildings"`
	Ships     map[string]int         `json:"ships"`
	Defenses  map[string]int         `json:"defenses"`
}

type BuildingInfo struct {
	Type  string `json:"type"`
	Level int    `json:"level"`
}

type ResourceOverrideRequest struct {
	Metal   int `json:"metal"`
	Crystal int `json:"crystal"`
	Gas     int `json:"gas"`
}

type DMGrantRequest struct {
	Amount int    `json:"amount"`
	Reason string `json:"reason"`
}

type CreditsGrantRequest struct {
	Amount int    `json:"amount"`
	Reason string `json:"reason"`
}

type BanRequest struct {
	Banned bool   `json:"banned"`
	Reason string `json:"reason"`
}

type GMMessageRequest struct {
	PlayerID int    `json:"player_id"`
	Subject  string `json:"subject"`
	Message  string `json:"message"`
}

type EventCreateRequest struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	EventType   string         `json:"event_type"`
	Modifiers   map[string]any `json:"modifiers"`
	StartsAt    time.Time      `json:"starts_at"`
	EndsAt      time.Time      `json:"ends_at"`
}
