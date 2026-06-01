package main

import "time"

type Notification struct {
	ID        int       `json:"id"`
	PlayerID  int       `json:"player_id"`
	Category  string    `json:"category"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	IsRead    bool      `json:"is_read"`
	CreatedAt time.Time `json:"created_at"`
}

type NotificationListRequest struct {
	UnreadOnly bool `json:"unread_only"`
	Limit      int  `json:"limit"`
	Offset     int  `json:"offset"`
}

type NotificationListResponse struct {
	Notifications []Notification `json:"notifications"`
	Total         int            `json:"total"`
}

type UnreadCountResponse struct {
	Count int `json:"count"`
}

type CreateNotificationRequest struct {
	PlayerID int    `json:"player_id"`
	Category string `json:"category"`
	Title    string `json:"title"`
	Message  string `json:"message"`
}
