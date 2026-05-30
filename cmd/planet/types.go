package main

import "time"

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
	Buildings   []Building   `json:"buildings"`
	Production Production   `json:"production"`
	Storage    Storage      `json:"storage"`
	Queue      []QueueEntry `json:"queue"`
}
