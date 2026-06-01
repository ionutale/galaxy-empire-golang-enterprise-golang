package main

import "time"

type Message struct {
	ID         int       `json:"id"`
	Channel    string    `json:"channel"`
	ChannelID  int       `json:"channel_id"`
	SenderID   int       `json:"sender_id"`
	SenderName string    `json:"sender_name"`
	Content    string    `json:"content"`
	CreatedAt  time.Time `json:"created_at"`
}

type SendMessageRequest struct {
	Channel string `json:"channel"`
	Content string `json:"content"`
}

type SendMessageResponse struct {
	ID        int       `json:"id"`
	Channel   string    `json:"channel"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type GetMessagesResponse struct {
	Messages []Message `json:"messages"`
	HasMore  bool      `json:"has_more"`
}

type PlayerAllianceResponse struct {
	InAlliance   bool   `json:"in_alliance"`
	AllianceID   int    `json:"alliance_id,omitempty"`
	Role         string `json:"role,omitempty"`
	AllianceName string `json:"alliance_name,omitempty"`
	AllianceTag  string `json:"alliance_tag,omitempty"`
}

type PrivateMessage struct {
	ID         int       `json:"id"`
	SenderID   int       `json:"sender_id"`
	ReceiverID int       `json:"receiver_id"`
	Content    string    `json:"content"`
	IsRead     bool      `json:"is_read"`
	IsSystem   bool      `json:"is_system,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type SendPrivateMessageRequest struct {
	ReceiverID int    `json:"receiver_id"`
	Content    string `json:"content"`
}

type PrivateMessagesResponse struct {
	Messages []PrivateMessage `json:"messages"`
	HasMore  bool             `json:"has_more"`
}

type InboxSummary struct {
	UnreadCount int `json:"unread_count"`
	TotalCount  int `json:"total_count"`
}
