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
	ResourcesUpdatedAt time.Time `json:"-"`
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

type PlanetResponse struct {
	ID         int        `json:"id"`
	UserID     int        `json:"user_id"`
	Name       string     `json:"name"`
	Metal      int        `json:"metal"`
	Crystal    int        `json:"crystal"`
	Gas        int        `json:"gas"`
	Energy     int        `json:"energy"`
	Galaxy     int        `json:"galaxy"`
	System     int        `json:"system"`
	Position   int        `json:"position"`
	Buildings  []Building `json:"buildings"`
	Production Production `json:"production"`
}
