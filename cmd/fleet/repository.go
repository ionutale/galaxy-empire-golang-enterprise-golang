package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	CreateFleet(ctx context.Context, f Fleet) (Fleet, error)
	ListPlayerFleets(ctx context.Context, playerID int) ([]Fleet, error)
	MarkFleetArrived(ctx context.Context, fleetID int) error
	GetArrivedFleets(ctx context.Context) ([]Fleet, error)
	CountPlayerFleets(ctx context.Context, playerID int) (int, error)
	GetFleetByID(ctx context.Context, fleetID int) (Fleet, error)
	DeleteFleet(ctx context.Context, fleetID int) error
	UpdateFleetShips(ctx context.Context, fleetID int, ships map[string]int) error
	SetFleetReturning(ctx context.Context, fleetID int, arrivesAt time.Time) error
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
		INSERT INTO fleet.fleets (player_id, origin_planet_id, target_galaxy, target_system, target_position, mission, status, speed_pct, ships, arrives_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`, f.PlayerID, f.OriginPlanetID, f.TargetGalaxy, f.TargetSystem, f.TargetPosition, f.Mission, f.Status, f.SpeedPct, shipsJSON, f.ArrivesAt).Scan(&id)
	if err != nil {
		return Fleet{}, fmt.Errorf("create fleet: %w", err)
	}
	f.ID = id
	return f, nil
}

func (r *PostgresRepository) ListPlayerFleets(ctx context.Context, playerID int) ([]Fleet, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, player_id, origin_planet_id, target_galaxy, target_system, target_position, mission, status, speed_pct, ships, arrives_at
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
		var arrivesAt *time.Time
		if err := rows.Scan(&f.ID, &f.PlayerID, &f.OriginPlanetID, &f.TargetGalaxy, &f.TargetSystem, &f.TargetPosition, &f.Mission, &f.Status, &f.SpeedPct, &shipsJSON, &arrivesAt); err != nil {
			return nil, fmt.Errorf("scan fleet: %w", err)
		}
		json.Unmarshal(shipsJSON, &f.Ships)
		if arrivesAt != nil {
			f.ArrivesAt = *arrivesAt
		}
		fleets = append(fleets, f)
	}
	return fleets, rows.Err()
}

func (r *PostgresRepository) MarkFleetArrived(ctx context.Context, fleetID int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE fleet.fleets SET status = 'arrived' WHERE id = $1 AND status IN ('in_transit', 'returning')`,
		fleetID,
	)
	if err != nil {
		return fmt.Errorf("mark fleet arrived: %w", err)
	}
	return nil
}

func (r *PostgresRepository) CountPlayerFleets(ctx context.Context, playerID int) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM fleet.fleets WHERE player_id = $1`, playerID).Scan(&count)
	return count, err
}

func (r *PostgresRepository) GetFleetByID(ctx context.Context, fleetID int) (Fleet, error) {
	var f Fleet
	var shipsJSON []byte
	var arrivesAt *time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT id, player_id, origin_planet_id, target_galaxy, target_system, target_position, mission, status, speed_pct, ships, arrives_at
		FROM fleet.fleets WHERE id = $1
	`, fleetID).Scan(&f.ID, &f.PlayerID, &f.OriginPlanetID, &f.TargetGalaxy, &f.TargetSystem, &f.TargetPosition, &f.Mission, &f.Status, &f.SpeedPct, &shipsJSON, &arrivesAt)
	if err != nil {
		return Fleet{}, err
	}
	json.Unmarshal(shipsJSON, &f.Ships)
	if arrivesAt != nil {
		f.ArrivesAt = *arrivesAt
	}
	return f, nil
}

func (r *PostgresRepository) DeleteFleet(ctx context.Context, fleetID int) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM fleet.fleets WHERE id = $1`, fleetID)
	return err
}

func (r *PostgresRepository) UpdateFleetShips(ctx context.Context, fleetID int, ships map[string]int) error {
	shipsJSON, _ := json.Marshal(ships)
	_, err := r.pool.Exec(ctx, `UPDATE fleet.fleets SET ships = $1 WHERE id = $2`, shipsJSON, fleetID)
	return err
}

func (r *PostgresRepository) SetFleetReturning(ctx context.Context, fleetID int, arrivesAt time.Time) error {
	_, err := r.pool.Exec(ctx, `UPDATE fleet.fleets SET status = 'returning', arrives_at = $1 WHERE id = $2`, arrivesAt, fleetID)
	return err
}

func (r *PostgresRepository) GetArrivedFleets(ctx context.Context) ([]Fleet, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, player_id, origin_planet_id, target_galaxy, target_system, target_position, mission, status, speed_pct, ships, arrives_at
		FROM fleet.fleets
		WHERE (status = 'in_transit' OR status = 'returning') AND arrives_at <= NOW()
	`)
	if err != nil {
		return nil, fmt.Errorf("get arrived fleets: %w", err)
	}
	defer rows.Close()

	var fleets []Fleet
	for rows.Next() {
		var f Fleet
		var shipsJSON []byte
		var arrivesAt *time.Time
		if err := rows.Scan(&f.ID, &f.PlayerID, &f.OriginPlanetID, &f.TargetGalaxy, &f.TargetSystem, &f.TargetPosition, &f.Mission, &f.Status, &f.SpeedPct, &shipsJSON, &arrivesAt); err != nil {
			return nil, fmt.Errorf("scan fleet: %w", err)
		}
		json.Unmarshal(shipsJSON, &f.Ships)
		if arrivesAt != nil {
			f.ArrivesAt = *arrivesAt
		}
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

func (m *mockRepo) MarkFleetArrived(ctx context.Context, fleetID int) error {
	for i, f := range m.fleets {
		if f.ID == fleetID {
			m.fleets[i].Status = "arrived"
			return nil
		}
	}
	return nil
}

func (m *mockRepo) CountPlayerFleets(ctx context.Context, playerID int) (int, error) {
	count := 0
	for _, f := range m.fleets {
		if f.PlayerID == playerID {
			count++
		}
	}
	return count, nil
}

func (m *mockRepo) GetFleetByID(ctx context.Context, fleetID int) (Fleet, error) {
	for _, f := range m.fleets {
		if f.ID == fleetID {
			return f, nil
		}
	}
	return Fleet{}, fmt.Errorf("fleet not found")
}

func (m *mockRepo) DeleteFleet(ctx context.Context, fleetID int) error {
	for i, f := range m.fleets {
		if f.ID == fleetID {
			m.fleets = append(m.fleets[:i], m.fleets[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *mockRepo) UpdateFleetShips(ctx context.Context, fleetID int, ships map[string]int) error {
	for i, f := range m.fleets {
		if f.ID == fleetID {
			m.fleets[i].Ships = ships
			return nil
		}
	}
	return nil
}

func (m *mockRepo) SetFleetReturning(ctx context.Context, fleetID int, arrivesAt time.Time) error {
	for i, f := range m.fleets {
		if f.ID == fleetID {
			m.fleets[i].Status = "returning"
			m.fleets[i].ArrivesAt = arrivesAt
			return nil
		}
	}
	return nil
}

func (m *mockRepo) GetArrivedFleets(ctx context.Context) ([]Fleet, error) {
	var result []Fleet
	for _, f := range m.fleets {
		if (f.Status == "in_transit" || f.Status == "returning") && !f.ArrivesAt.IsZero() && time.Now().After(f.ArrivesAt) {
			result = append(result, f)
		}
	}
	return result, nil
}
