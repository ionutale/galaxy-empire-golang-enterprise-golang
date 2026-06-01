package main

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	CreateNotification(ctx context.Context, playerID int, category, title, message string) (Notification, error)
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

func (r *PostgresRepository) ListNotifications(ctx context.Context, playerID int, unreadOnly bool, limit, offset int) ([]Notification, int, error) {
	var total int
	countQuery := "SELECT COUNT(*) FROM notification.notifications WHERE player_id = $1"
	countArgs := []any{playerID}
	if unreadOnly {
		countQuery += " AND is_read = FALSE"
	}
	err := r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count notifications: %w", err)
	}

	query := "SELECT id, player_id, category, title, message, is_read, created_at FROM notification.notifications WHERE player_id = $1"
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
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.PlayerID, &n.Category, &n.Title, &n.Message, &n.IsRead, &n.CreatedAt); err != nil {
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
