package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	GetActiveEvents(ctx context.Context) ([]Event, error)
	GetAllEvents(ctx context.Context) ([]Event, error)
	GetEventByID(ctx context.Context, eventID int) (Event, error)
	CreateEvent(ctx context.Context, e Event) (Event, error)
	UpdateEventStatus(ctx context.Context, eventID int, status string) error
	GetEventsByStatus(ctx context.Context, status string) ([]Event, error)

	GetParticipation(ctx context.Context, playerID, eventID int) (EventParticipation, error)
	GetPlayerParticipations(ctx context.Context, playerID int, eventIDs []int) (map[int]EventParticipation, error)
	JoinEvent(ctx context.Context, playerID, eventID int) (EventParticipation, error)
	ClaimRewards(ctx context.Context, playerID, eventID int) error
}

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) GetActiveEvents(ctx context.Context) ([]Event, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, description, event_type, modifiers, starts_at, ends_at, status, created_at
		FROM event.events
		WHERE status = 'active'
		ORDER BY ends_at LIMIT 100
`)
	if err != nil {
		return nil, fmt.Errorf("get active events: %w", err)
	}
	defer rows.Close()
	return scanEvents(rows)
}

func (r *PostgresRepository) GetAllEvents(ctx context.Context) ([]Event, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, description, event_type, modifiers, starts_at, ends_at, status, created_at
		FROM event.events
		ORDER BY starts_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("get all events: %w", err)
	}
	defer rows.Close()
	return scanEvents(rows)
}

func (r *PostgresRepository) GetEventByID(ctx context.Context, eventID int) (Event, error) {
	var e Event
	var modifiers []byte
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, description, event_type, modifiers, starts_at, ends_at, status, created_at
		FROM event.events WHERE id = $1
	`, eventID).Scan(&e.ID, &e.Name, &e.Description, &e.EventType, &modifiers, &e.StartsAt, &e.EndsAt, &e.Status, &e.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Event{}, fmt.Errorf("event not found")
		}
		return Event{}, fmt.Errorf("get event: %w", err)
	}
	json.Unmarshal(modifiers, &e.Modifiers)
	if e.Modifiers == nil {
		e.Modifiers = map[string]any{}
	}
	return e, nil
}

func (r *PostgresRepository) CreateEvent(ctx context.Context, e Event) (Event, error) {
	modifiersJSON, _ := json.Marshal(e.Modifiers)
	var id int
	var createdAt time.Time
	err := r.pool.QueryRow(ctx, `
		INSERT INTO event.events (name, description, event_type, modifiers, starts_at, ends_at, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`, e.Name, e.Description, e.EventType, modifiersJSON, e.StartsAt, e.EndsAt, e.Status).Scan(&id, &createdAt)
	if err != nil {
		return Event{}, fmt.Errorf("create event: %w", err)
	}
	e.ID = id
	e.CreatedAt = createdAt
	return e, nil
}

func (r *PostgresRepository) UpdateEventStatus(ctx context.Context, eventID int, status string) error {
	_, err := r.pool.Exec(ctx, `UPDATE event.events SET status = $1 WHERE id = $2`, status, eventID)
	return err
}

func (r *PostgresRepository) GetEventsByStatus(ctx context.Context, status string) ([]Event, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, description, event_type, modifiers, starts_at, ends_at, status, created_at
		FROM event.events WHERE status = $1
		ORDER BY starts_at
	`, status)
	if err != nil {
		return nil, fmt.Errorf("get events by status: %w", err)
	}
	defer rows.Close()
	return scanEvents(rows)
}

func (r *PostgresRepository) GetParticipation(ctx context.Context, playerID, eventID int) (EventParticipation, error) {
	var p EventParticipation
	var progress []byte
	err := r.pool.QueryRow(ctx, `
		SELECT id, player_id, event_id, progress, completed, rewards_claimed, joined_at
		FROM event.event_participation
		WHERE player_id = $1 AND event_id = $2
	`, playerID, eventID).Scan(&p.ID, &p.PlayerID, &p.EventID, &progress, &p.Completed, &p.RewardsClaimed, &p.JoinedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return EventParticipation{}, fmt.Errorf("participation not found")
		}
		return EventParticipation{}, fmt.Errorf("get participation: %w", err)
	}
	json.Unmarshal(progress, &p.Progress)
	if p.Progress == nil {
		p.Progress = map[string]any{}
	}
	return p, nil
}

func (r *PostgresRepository) GetPlayerParticipations(ctx context.Context, playerID int, eventIDs []int) (map[int]EventParticipation, error) {
	if len(eventIDs) == 0 {
		return map[int]EventParticipation{}, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, player_id, event_id, progress, completed, rewards_claimed, joined_at
		FROM event.event_participation
		WHERE player_id = $1 AND event_id = ANY($2)
	`, playerID, eventIDs)
	if err != nil {
		return nil, fmt.Errorf("get player participations: %w", err)
	}
	defer rows.Close()
	result := make(map[int]EventParticipation)
	for rows.Next() {
		var p EventParticipation
		var progress []byte
		if err := rows.Scan(&p.ID, &p.PlayerID, &p.EventID, &progress, &p.Completed, &p.RewardsClaimed, &p.JoinedAt); err != nil {
			return nil, fmt.Errorf("scan participation: %w", err)
		}
		json.Unmarshal(progress, &p.Progress)
		if p.Progress == nil {
			p.Progress = map[string]any{}
		}
		result[p.EventID] = p
	}
	return result, rows.Err()
}

func (r *PostgresRepository) JoinEvent(ctx context.Context, playerID, eventID int) (EventParticipation, error) {
	var p EventParticipation
	var progress []byte
	err := r.pool.QueryRow(ctx, `
		INSERT INTO event.event_participation (player_id, event_id)
		VALUES ($1, $2)
		RETURNING id, player_id, event_id, progress, completed, rewards_claimed, joined_at
	`, playerID, eventID).Scan(&p.ID, &p.PlayerID, &p.EventID, &progress, &p.Completed, &p.RewardsClaimed, &p.JoinedAt)
	if err != nil {
		return EventParticipation{}, fmt.Errorf("join event: %w", err)
	}
	json.Unmarshal(progress, &p.Progress)
	if p.Progress == nil {
		p.Progress = map[string]any{}
	}
	return p, nil
}

func (r *PostgresRepository) ClaimRewards(ctx context.Context, playerID, eventID int) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE event.event_participation
		SET rewards_claimed = TRUE
		WHERE player_id = $1 AND event_id = $2 AND completed = TRUE AND rewards_claimed = FALSE
	`, playerID, eventID)
	if err != nil {
		return fmt.Errorf("claim rewards: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("cannot claim rewards: not completed or already claimed")
	}
	return nil
}

func scanEvents(rows pgx.Rows) ([]Event, error) {
	var events []Event
	for rows.Next() {
		var e Event
		var modifiers []byte
		if err := rows.Scan(&e.ID, &e.Name, &e.Description, &e.EventType, &modifiers, &e.StartsAt, &e.EndsAt, &e.Status, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		json.Unmarshal(modifiers, &e.Modifiers)
		if e.Modifiers == nil {
			e.Modifiers = map[string]any{}
		}
		events = append(events, e)
	}
	return events, rows.Err()
}
