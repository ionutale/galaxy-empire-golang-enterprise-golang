package main

import "time"

type Friendship struct {
	ID        int       `json:"id"`
	PlayerID  int       `json:"player_id"`
	FriendID  int       `json:"friend_id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type FriendResponse struct {
	PlayerID   int    `json:"player_id"`
	Status     string `json:"status"`
	Online     bool   `json:"online"`
	LastActive string `json:"last_active_at,omitempty"`
	CreatedAt  string `json:"created_at"`
}

type AddFriendRequest struct {
	FriendID int `json:"friend_id"`
}

type FriendListResponse struct {
	Friends []FriendResponse `json:"friends"`
}
