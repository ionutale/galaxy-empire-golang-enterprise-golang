package main

type Planet struct {
	ID       int    `json:"id"`
	UserID   int    `json:"user_id"`
	Name     string `json:"name"`
	Metal    int    `json:"metal"`
	Crystal  int    `json:"crystal"`
	Gas      int    `json:"gas"`
	Energy   int    `json:"energy"`
	Galaxy   int    `json:"galaxy"`
	System   int    `json:"system"`
	Position int    `json:"position"`
}
