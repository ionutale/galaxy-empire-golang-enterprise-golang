package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	CreateNotification(ctx context.Context, playerID int, category, title, message string) (Notification, error)
	CreateBulkNotifications(ctx context.Context, notifications []Notification) error
	ListNotifications(ctx context.Context, playerID int, unreadOnly bool, limit, offset int) ([]Notification, int, error)
	UnreadCount(ctx context.Context, playerID int) (int, error)
	MarkRead(ctx context.Context, id, playerID int) error
	MarkAllRead(ctx context.Context, playerID int) error
}

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) CreateNotification(ctx context.Context, playerID int, category, title, message string) (Notification, error) {
	var n Notification
	err := r.pool.QueryRow(ctx, `
		INSERT INTO notification.notifications (player_id, category, title, message)
		VALUES ($1, $2, $3, $4)
		RETURNING id, player_id, category, title, message, is_read, created_at
	`, playerID, category, title, message).Scan(
		&n.ID, &n.PlayerID, &n.Category, &n.Title, &n.Message, &n.IsRead, &n.CreatedAt,
	)
	if err != nil {
		return Notification{}, fmt.Errorf("create notification: %w", err)
	}
	return n, nil
}

// CreateBulkNotifications inserts all notifications in a single statement for efficiency.
// Callers (e.g. event triggers) that need to fan-out a notification to many players
// should use this instead of looping over CreateNotification.
func (r *PostgresRepository) CreateBulkNotifications(ctx context.Context, notifications []Notification) error {
	if len(notifications) == 0 {
		return nil
	}

	// Build a single multi-row INSERT: VALUES ($1,$2,$3,$4), ($5,$6,$7,$8), ...
	placeholders := make([]string, len(notifications))
	args := make([]any, 0, len(notifications)*4)
	for i, n := range notifications {
		base := i * 4
		placeholders[i] = fmt.Sprintf("($%d, $%d, $%d, $%d)", base+1, base+2, base+3, base+4)
		args = append(args, n.PlayerID, n.Category, n.Title, n.Message)
	}

	query := "INSERT INTO notification.notifications (player_id, category, title, message) VALUES " +
		strings.Join(placeholders, ", ")

	_, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("bulk create notifications: %w", err)
	}
	return nil
}

func (r *PostgresRepository) ListNotifications(ctx context.Context, playerID int, unreadOnly bool, limit, offset int) ([]Notification, int, error) {
	if limit > 100 {
		limit = 100
	}

	query := `SELECT id, player_id, category, title, message, is_read, created_at, COUNT(*) OVER() AS total
		FROM notification.notifications
		WHERE player_id = $1`
	args := []any{playerID}
	if unreadOnly {
		query += " AND is_read = FALSE"
	}
	query += " ORDER BY created_at DESC LIMIT $2 OFFSET $3"
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()

	var notifications []Notification
	var total int
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.PlayerID, &n.Category, &n.Title, &n.Message, &n.IsRead, &n.CreatedAt, &total); err != nil {
			return nil, 0, fmt.Errorf("scan notification: %w", err)
		}
		notifications = append(notifications, n)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return notifications, total, nil
}

func (r *PostgresRepository) UnreadCount(ctx context.Context, playerID int) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM notification.notifications
		WHERE player_id = $1 AND is_read = FALSE
	`, playerID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("unread count: %w", err)
	}
	return count, nil
}

func (r *PostgresRepository) MarkRead(ctx context.Context, id, playerID int) error {
	result, err := r.pool.Exec(ctx, `
		UPDATE notification.notifications
		SET is_read = TRUE
		WHERE id = $1 AND player_id = $2
	`, id, playerID)
	if err != nil {
		return fmt.Errorf("mark read: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("notification not found")
	}
	return nil
}

func (r *PostgresRepository) MarkAllRead(ctx context.Context, playerID int) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE notification.notifications
		SET is_read = TRUE
		WHERE player_id = $1 AND is_read = FALSE
	`, playerID)
	if err != nil {
		return fmt.Errorf("mark all read: %w", err)
	}
	return nil
}
