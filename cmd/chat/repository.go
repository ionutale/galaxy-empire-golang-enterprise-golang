package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	CreateMessage(ctx context.Context, channel string, channelID, senderID int, senderName, content string) (Message, error)
	GetMessages(ctx context.Context, channel string, channelID int, limit int, beforeID int) ([]Message, bool, error)
	CreatePrivateMessage(ctx context.Context, senderID, receiverID int, content string, isSystem bool) (PrivateMessage, error)
	GetInbox(ctx context.Context, playerID int, limit, beforeID int) ([]PrivateMessage, bool, error)
	GetOutbox(ctx context.Context, playerID int, limit, beforeID int) ([]PrivateMessage, bool, error)
	MarkMessageRead(ctx context.Context, messageID, playerID int) error
	DeletePrivateMessage(ctx context.Context, messageID, playerID int) error
	GetUnreadCount(ctx context.Context, playerID int) (int, error)
}

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) CreateMessage(ctx context.Context, channel string, channelID, senderID int, senderName, content string) (Message, error) {
	var m Message
	err := r.pool.QueryRow(ctx, `
		INSERT INTO chat.messages (channel, channel_id, sender_id, sender_name, content)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, channel, channel_id, sender_id, sender_name, content, created_at
	`, channel, channelID, senderID, senderName, content).Scan(
		&m.ID, &m.Channel, &m.ChannelID, &m.SenderID, &m.SenderName, &m.Content, &m.CreatedAt,
	)
	if err != nil {
		return Message{}, fmt.Errorf("create message: %w", err)
	}
	return m, nil
}

func (r *PostgresRepository) GetMessages(ctx context.Context, channel string, channelID int, limit int, beforeID int) ([]Message, bool, error) {
	var query string
	var args []any

	if beforeID > 0 {
		query = `
			SELECT id, channel, channel_id, sender_id, sender_name, content, created_at
			FROM chat.messages
			WHERE channel = $1 AND channel_id = $2 AND id < $3
			ORDER BY created_at DESC
			LIMIT $4
		`
		args = []any{channel, channelID, beforeID, limit + 1}
	} else {
		query = `
			SELECT id, channel, channel_id, sender_id, sender_name, content, created_at
			FROM chat.messages
			WHERE channel = $1 AND channel_id = $2
			ORDER BY created_at DESC
			LIMIT $3
		`
		args = []any{channel, channelID, limit + 1}
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, false, fmt.Errorf("get messages: %w", err)
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.Channel, &m.ChannelID, &m.SenderID, &m.SenderName, &m.Content, &m.CreatedAt); err != nil {
			return nil, false, fmt.Errorf("scan message: %w", err)
		}
		messages = append(messages, m)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}

	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit]
	}

	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, hasMore, nil
}

func (r *PostgresRepository) CreatePrivateMessage(ctx context.Context, senderID, receiverID int, content string, isSystem bool) (PrivateMessage, error) {
	var m PrivateMessage
	err := r.pool.QueryRow(ctx, `
		INSERT INTO chat.private_messages (sender_id, receiver_id, content, is_system)
		VALUES ($1, $2, $3, $4)
		RETURNING id, sender_id, receiver_id, content, is_read, is_system, created_at
	`, senderID, receiverID, content, isSystem).Scan(
		&m.ID, &m.SenderID, &m.ReceiverID, &m.Content, &m.IsRead, &m.IsSystem, &m.CreatedAt,
	)
	if err != nil {
		return PrivateMessage{}, fmt.Errorf("create private message: %w", err)
	}
	return m, nil
}

func (r *PostgresRepository) GetInbox(ctx context.Context, playerID int, limit, beforeID int) ([]PrivateMessage, bool, error) {
	var query string
	var args []any

	if beforeID > 0 {
		query = `
			SELECT id, sender_id, receiver_id, content, is_read, is_system, created_at
			FROM chat.private_messages
			WHERE receiver_id = $1 AND id < $2
			ORDER BY created_at DESC
			LIMIT $3
		`
		args = []any{playerID, beforeID, limit + 1}
	} else {
		query = `
			SELECT id, sender_id, receiver_id, content, is_read, is_system, created_at
			FROM chat.private_messages
			WHERE receiver_id = $1
			ORDER BY created_at DESC
			LIMIT $2
		`
		args = []any{playerID, limit + 1}
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, false, fmt.Errorf("get inbox: %w", err)
	}
	defer rows.Close()

	var messages []PrivateMessage
	for rows.Next() {
		var m PrivateMessage
		if err := rows.Scan(&m.ID, &m.SenderID, &m.ReceiverID, &m.Content, &m.IsRead, &m.IsSystem, &m.CreatedAt); err != nil {
			return nil, false, fmt.Errorf("scan private message: %w", err)
		}
		messages = append(messages, m)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}

	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit]
	}

	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, hasMore, nil
}

func (r *PostgresRepository) GetOutbox(ctx context.Context, playerID int, limit, beforeID int) ([]PrivateMessage, bool, error) {
	var query string
	var args []any

	if beforeID > 0 {
		query = `
			SELECT id, sender_id, receiver_id, content, is_read, is_system, created_at
			FROM chat.private_messages
			WHERE sender_id = $1 AND id < $2
			ORDER BY created_at DESC
			LIMIT $3
		`
		args = []any{playerID, beforeID, limit + 1}
	} else {
		query = `
			SELECT id, sender_id, receiver_id, content, is_read, is_system, created_at
			FROM chat.private_messages
			WHERE sender_id = $1
			ORDER BY created_at DESC
			LIMIT $2
		`
		args = []any{playerID, limit + 1}
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, false, fmt.Errorf("get outbox: %w", err)
	}
	defer rows.Close()

	var messages []PrivateMessage
	for rows.Next() {
		var m PrivateMessage
		if err := rows.Scan(&m.ID, &m.SenderID, &m.ReceiverID, &m.Content, &m.IsRead, &m.IsSystem, &m.CreatedAt); err != nil {
			return nil, false, fmt.Errorf("scan private message: %w", err)
		}
		messages = append(messages, m)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}

	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit]
	}

	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, hasMore, nil
}

func (r *PostgresRepository) MarkMessageRead(ctx context.Context, messageID, playerID int) error {
	result, err := r.pool.Exec(ctx, `
		UPDATE chat.private_messages
		SET is_read = TRUE
		WHERE id = $1 AND receiver_id = $2
	`, messageID, playerID)
	if err != nil {
		return fmt.Errorf("mark message read: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("message not found or not yours")
	}
	return nil
}

func (r *PostgresRepository) DeletePrivateMessage(ctx context.Context, messageID, playerID int) error {
	result, err := r.pool.Exec(ctx, `
		DELETE FROM chat.private_messages
		WHERE id = $1 AND (sender_id = $2 OR receiver_id = $2)
	`, messageID, playerID)
	if err != nil {
		return fmt.Errorf("delete private message: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("message not found")
	}
	return nil
}

func (r *PostgresRepository) GetUnreadCount(ctx context.Context, playerID int) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM chat.private_messages
		WHERE receiver_id = $1 AND is_read = FALSE
	`, playerID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("get unread count: %w", err)
	}
	return count, nil
}

type mockRepo struct {
	mu              sync.Mutex
	messages        []Message
	privateMessages []PrivateMessage
	nextID          int
}

func newMockRepo() *mockRepo {
	return &mockRepo{nextID: 1}
}

func (m *mockRepo) CreateMessage(ctx context.Context, channel string, channelID, senderID int, senderName, content string) (Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	msg := Message{
		ID:         m.nextID,
		Channel:    channel,
		ChannelID:  channelID,
		SenderID:   senderID,
		SenderName: senderName,
		Content:    content,
		CreatedAt:  time.Now(),
	}
	m.nextID++
	m.messages = append(m.messages, msg)
	return msg, nil
}

func (m *mockRepo) GetMessages(ctx context.Context, channel string, channelID int, limit int, beforeID int) ([]Message, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var filtered []Message
	for _, msg := range m.messages {
		if msg.Channel == channel && msg.ChannelID == channelID {
			if beforeID > 0 && msg.ID >= beforeID {
				continue
			}
			filtered = append(filtered, msg)
		}
	}

	// Sort by created_at DESC
	for i := 0; i < len(filtered); i++ {
		for j := i + 1; j < len(filtered); j++ {
			if filtered[j].CreatedAt.After(filtered[i].CreatedAt) {
				filtered[i], filtered[j] = filtered[j], filtered[i]
			}
		}
	}

	hasMore := len(filtered) > limit
	if hasMore {
		filtered = filtered[:limit]
	}

	for i, j := 0, len(filtered)-1; i < j; i, j = i+1, j-1 {
		filtered[i], filtered[j] = filtered[j], filtered[i]
	}

	return filtered, hasMore, nil
}

func (m *mockRepo) CreatePrivateMessage(ctx context.Context, senderID, receiverID int, content string, isSystem bool) (PrivateMessage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	msg := PrivateMessage{
		ID:         m.nextID,
		SenderID:   senderID,
		ReceiverID: receiverID,
		Content:    content,
		IsSystem:   isSystem,
		CreatedAt:  time.Now(),
	}
	m.nextID++
	m.privateMessages = append(m.privateMessages, msg)
	return msg, nil
}

func (m *mockRepo) GetInbox(ctx context.Context, playerID int, limit, beforeID int) ([]PrivateMessage, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var filtered []PrivateMessage
	for _, msg := range m.privateMessages {
		if msg.ReceiverID == playerID {
			if beforeID > 0 && msg.ID >= beforeID {
				continue
			}
			filtered = append(filtered, msg)
		}
	}

	for i := 0; i < len(filtered); i++ {
		for j := i + 1; j < len(filtered); j++ {
			if filtered[j].CreatedAt.After(filtered[i].CreatedAt) {
				filtered[i], filtered[j] = filtered[j], filtered[i]
			}
		}
	}

	hasMore := len(filtered) > limit
	if hasMore {
		filtered = filtered[:limit]
	}

	for i, j := 0, len(filtered)-1; i < j; i, j = i+1, j-1 {
		filtered[i], filtered[j] = filtered[j], filtered[i]
	}

	return filtered, hasMore, nil
}

func (m *mockRepo) GetOutbox(ctx context.Context, playerID int, limit, beforeID int) ([]PrivateMessage, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var filtered []PrivateMessage
	for _, msg := range m.privateMessages {
		if msg.SenderID == playerID {
			if beforeID > 0 && msg.ID >= beforeID {
				continue
			}
			filtered = append(filtered, msg)
		}
	}

	for i := 0; i < len(filtered); i++ {
		for j := i + 1; j < len(filtered); j++ {
			if filtered[j].CreatedAt.After(filtered[i].CreatedAt) {
				filtered[i], filtered[j] = filtered[j], filtered[i]
			}
		}
	}

	hasMore := len(filtered) > limit
	if hasMore {
		filtered = filtered[:limit]
	}

	for i, j := 0, len(filtered)-1; i < j; i, j = i+1, j-1 {
		filtered[i], filtered[j] = filtered[j], filtered[i]
	}

	return filtered, hasMore, nil
}

func (m *mockRepo) MarkMessageRead(ctx context.Context, messageID, playerID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.privateMessages {
		if m.privateMessages[i].ID == messageID && m.privateMessages[i].ReceiverID == playerID {
			m.privateMessages[i].IsRead = true
			return nil
		}
	}
	return fmt.Errorf("message not found or not yours")
}

func (m *mockRepo) DeletePrivateMessage(ctx context.Context, messageID, playerID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.privateMessages {
		if m.privateMessages[i].ID == messageID && (m.privateMessages[i].SenderID == playerID || m.privateMessages[i].ReceiverID == playerID) {
			m.privateMessages = append(m.privateMessages[:i], m.privateMessages[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("message not found")
}

func (m *mockRepo) GetUnreadCount(ctx context.Context, playerID int) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := 0
	for _, msg := range m.privateMessages {
		if msg.ReceiverID == playerID && !msg.IsRead {
			count++
		}
	}
	return count, nil
}
