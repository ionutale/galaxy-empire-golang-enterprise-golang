package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	CreateFleet(ctx context.Context, f Fleet) (Fleet, error)
	ListPlayerFleets(ctx context.Context, playerID int) ([]Fleet, error)
}

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) CreateFleet(ctx context.Context, f Fleet) (Fleet, error) {
	shipsJSON, err := json.Marshal(f.Ships)
	if err != nil {
		return Fleet{}, fmt.Errorf("marshal ships: %w", err)
	}

	var id int
	err = r.pool.QueryRow(ctx, `
		INSERT INTO fleet.fleets (player_id, origin_planet_id, target_galaxy, target_system, target_position, mission, status, speed_pct, ships)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`, f.PlayerID, f.OriginPlanetID, f.TargetGalaxy, f.TargetSystem, f.TargetPosition, f.Mission, f.Status, f.SpeedPct, shipsJSON).Scan(&id)
	if err != nil {
		return Fleet{}, fmt.Errorf("create fleet: %w", err)
	}
	f.ID = id
	return f, nil
}

func (r *PostgresRepository) ListPlayerFleets(ctx context.Context, playerID int) ([]Fleet, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, player_id, origin_planet_id, target_galaxy, target_system, target_position, mission, status, speed_pct, ships
		FROM fleet.fleets
		WHERE player_id = $1
		ORDER BY created_at DESC
	`, playerID)
	if err != nil {
		return nil, fmt.Errorf("list fleets: %w", err)
	}
	defer rows.Close()

	var fleets []Fleet
	for rows.Next() {
		var f Fleet
		var shipsJSON []byte
		if err := rows.Scan(&f.ID, &f.PlayerID, &f.OriginPlanetID, &f.TargetGalaxy, &f.TargetSystem, &f.TargetPosition, &f.Mission, &f.Status, &f.SpeedPct, &shipsJSON); err != nil {
			return nil, fmt.Errorf("scan fleet: %w", err)
		}
		json.Unmarshal(shipsJSON, &f.Ships)
		fleets = append(fleets, f)
	}
	return fleets, rows.Err()
}

type mockRepo struct {
	fleets []Fleet
	nextID int
}

func newMockRepo() *mockRepo {
	return &mockRepo{nextID: 1}
}

func (m *mockRepo) CreateFleet(ctx context.Context, f Fleet) (Fleet, error) {
	f.ID = m.nextID
	m.nextID++
	m.fleets = append(m.fleets, f)
	return f, nil
}

func (m *mockRepo) ListPlayerFleets(ctx context.Context, playerID int) ([]Fleet, error) {
	var result []Fleet
	for _, f := range m.fleets {
		if f.PlayerID == playerID {
			result = append(result, f)
		}
	}
	return result, nil
}
