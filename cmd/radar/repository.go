package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	CreateRadarEvent(ctx context.Context, e RadarEvent) (RadarEvent, error)
	GetPlayerEvents(ctx context.Context, playerID int) ([]RadarEvent, error)
	GetUnresolvedEvents(ctx context.Context, playerID int) ([]RadarEvent, error)
	ResolveEvent(ctx context.Context, eventID int) error
	CreateOrUpdateEuxRadar(ctx context.Context, playerID int, galaxy, system, position, level int) error
	GetEuxRadar(ctx context.Context, playerID int) (*EuxRadar, error)
}

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) CreateRadarEvent(ctx context.Context, e RadarEvent) (RadarEvent, error) {
	var created RadarEvent
	err := r.pool.QueryRow(ctx, `
		INSERT INTO radar.radar_events (player_id, event_type, source_player_id, fleet_id,
			target_galaxy, target_system, target_position,
			origin_galaxy, origin_system, origin_position,
			arrival_time, resolved)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, player_id, event_type, source_player_id, fleet_id,
			target_galaxy, target_system, target_position,
			origin_galaxy, origin_system, origin_position,
			arrival_time, detected_at, resolved
	`, e.PlayerID, e.EventType, e.SourcePlayerID, e.FleetID,
		e.TargetGalaxy, e.TargetSystem, e.TargetPosition,
		e.OriginGalaxy, e.OriginSystem, e.OriginPosition,
		e.ArrivalTime, e.Resolved,
	).Scan(
		&created.ID, &created.PlayerID, &created.EventType, &created.SourcePlayerID, &created.FleetID,
		&created.TargetGalaxy, &created.TargetSystem, &created.TargetPosition,
		&created.OriginGalaxy, &created.OriginSystem, &created.OriginPosition,
		&created.ArrivalTime, &created.DetectedAt, &created.Resolved,
	)
	if err != nil {
		return RadarEvent{}, fmt.Errorf("create radar event: %w", err)
	}
	return created, nil
}

func (r *PostgresRepository) GetPlayerEvents(ctx context.Context, playerID int) ([]RadarEvent, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, player_id, event_type, source_player_id, fleet_id,
			target_galaxy, target_system, target_position,
			origin_galaxy, origin_system, origin_position,
			arrival_time, detected_at, resolved
		FROM radar.radar_events
		WHERE player_id = $1
		ORDER BY detected_at DESC
	`, playerID)
	if err != nil {
		return nil, fmt.Errorf("get player events: %w", err)
	}
	defer rows.Close()

	var events []RadarEvent
	for rows.Next() {
		var e RadarEvent
		if err := rows.Scan(
			&e.ID, &e.PlayerID, &e.EventType, &e.SourcePlayerID, &e.FleetID,
			&e.TargetGalaxy, &e.TargetSystem, &e.TargetPosition,
			&e.OriginGalaxy, &e.OriginSystem, &e.OriginPosition,
			&e.ArrivalTime, &e.DetectedAt, &e.Resolved,
		); err != nil {
			return nil, fmt.Errorf("scan radar event: %w", err)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func (r *PostgresRepository) GetUnresolvedEvents(ctx context.Context, playerID int) ([]RadarEvent, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, player_id, event_type, source_player_id, fleet_id,
			target_galaxy, target_system, target_position,
			origin_galaxy, origin_system, origin_position,
			arrival_time, detected_at, resolved
		FROM radar.radar_events
		WHERE player_id = $1 AND resolved = false
		ORDER BY detected_at DESC
	`, playerID)
	if err != nil {
		return nil, fmt.Errorf("get unresolved events: %w", err)
	}
	defer rows.Close()

	var events []RadarEvent
	for rows.Next() {
		var e RadarEvent
		if err := rows.Scan(
			&e.ID, &e.PlayerID, &e.EventType, &e.SourcePlayerID, &e.FleetID,
			&e.TargetGalaxy, &e.TargetSystem, &e.TargetPosition,
			&e.OriginGalaxy, &e.OriginSystem, &e.OriginPosition,
			&e.ArrivalTime, &e.DetectedAt, &e.Resolved,
		); err != nil {
			return nil, fmt.Errorf("scan radar event: %w", err)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func (r *PostgresRepository) ResolveEvent(ctx context.Context, eventID int) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE radar.radar_events
		SET resolved = TRUE
		WHERE id = $1 AND resolved = FALSE
	`, eventID)
	if err != nil {
		return fmt.Errorf("resolve event: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("event already resolved")
	}
	return nil
}

func (r *PostgresRepository) CreateOrUpdateEuxRadar(ctx context.Context, playerID int, galaxy, system, position, level int) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO radar.eu_x_radars (player_id, moon_galaxy, moon_system, moon_position, level)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (player_id)
		DO UPDATE SET moon_galaxy = $2, moon_system = $3, moon_position = $4, level = $5
	`, playerID, galaxy, system, position, level)
	if err != nil {
		return fmt.Errorf("create or update eu-x radar: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetEuxRadar(ctx context.Context, playerID int) (*EuxRadar, error) {
	var e EuxRadar
	err := r.pool.QueryRow(ctx, `
		SELECT id, player_id, moon_galaxy, moon_system, moon_position, level
		FROM radar.eu_x_radars
		WHERE player_id = $1
	`, playerID).Scan(&e.ID, &e.PlayerID, &e.Galaxy, &e.System, &e.Position, &e.Level)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get eu-x radar: %w", err)
	}
	return &e, nil
}

type mockRepo struct {
	mu        sync.Mutex
	events    []RadarEvent
	euxRadars []EuxRadar
	nextID    int
	nextEUXID int
}

func newMockRepo() *mockRepo {
	return &mockRepo{nextID: 1, nextEUXID: 1}
}

func (m *mockRepo) CreateRadarEvent(ctx context.Context, e RadarEvent) (RadarEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e.ID = m.nextID
	m.nextID++
	e.DetectedAt = time.Now()
	m.events = append(m.events, e)
	return e, nil
}

func (m *mockRepo) GetPlayerEvents(ctx context.Context, playerID int) ([]RadarEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []RadarEvent
	for _, e := range m.events {
		if e.PlayerID == playerID {
			result = append(result, e)
		}
	}
	return result, nil
}

func (m *mockRepo) GetUnresolvedEvents(ctx context.Context, playerID int) ([]RadarEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []RadarEvent
	for _, e := range m.events {
		if e.PlayerID == playerID && !e.Resolved {
			result = append(result, e)
		}
	}
	return result, nil
}

func (m *mockRepo) ResolveEvent(ctx context.Context, eventID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, e := range m.events {
		if e.ID == eventID {
			m.events[i].Resolved = true
			return nil
		}
	}
	return fmt.Errorf("event not found")
}

func (m *mockRepo) CreateOrUpdateEuxRadar(ctx context.Context, playerID int, galaxy, system, position, level int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, e := range m.euxRadars {
		if e.PlayerID == playerID {
			m.euxRadars[i].Galaxy = galaxy
			m.euxRadars[i].System = system
			m.euxRadars[i].Position = position
			m.euxRadars[i].Level = level
			return nil
		}
	}
	e := EuxRadar{
		ID:       m.nextEUXID,
		PlayerID: playerID,
		Galaxy:   galaxy,
		System:   system,
		Position: position,
		Level:    level,
	}
	m.nextEUXID++
	m.euxRadars = append(m.euxRadars, e)
	return nil
}

func (m *mockRepo) GetEuxRadar(ctx context.Context, playerID int) (*EuxRadar, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.euxRadars {
		if e.PlayerID == playerID {
			er := e
			return &er, nil
		}
	}
	return nil, nil
}
