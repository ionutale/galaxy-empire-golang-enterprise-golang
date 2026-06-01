package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
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
	UpdateFleetOrigin(ctx context.Context, fleetID, newOriginPlanetID int) error
	GetAttackCooldown(ctx context.Context, attackerID, targetGalaxy, targetSystem, targetPosition int) (*time.Time, error)
	UpsertAttackCooldown(ctx context.Context, attackerID, targetGalaxy, targetSystem, targetPosition int, lastAttackAt time.Time) error
	GetACSGroupFleets(ctx context.Context, allianceGroupID int) ([]Fleet, error)
	GetACSDefendFleets(ctx context.Context, targetGalaxy, targetSystem, targetPosition int) ([]Fleet, error)
	GetDebrisField(ctx context.Context, galaxy, system, position int) (*DebrisField, error)
	UpsertDebrisField(ctx context.Context, galaxy, system, position, metal, crystal int) error
	UpdateDebrisField(ctx context.Context, galaxy, system, position, metal, crystal int) error
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
	var createdAt time.Time
	err = r.pool.QueryRow(ctx, `
		INSERT INTO fleet.fleets (player_id, origin_planet_id, target_galaxy, target_system, target_position, mission, status, speed_pct, ships, arrives_at, alliance_group_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at
	`, f.PlayerID, f.OriginPlanetID, f.TargetGalaxy, f.TargetSystem, f.TargetPosition, f.Mission, f.Status, f.SpeedPct, shipsJSON, f.ArrivesAt, f.AllianceGroupID).Scan(&id, &createdAt)
	if err != nil {
		return Fleet{}, fmt.Errorf("create fleet: %w", err)
	}
	f.ID = id
	f.CreatedAt = createdAt
	return f, nil
}

func (r *PostgresRepository) ListPlayerFleets(ctx context.Context, playerID int) ([]Fleet, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, player_id, origin_planet_id, target_galaxy, target_system, target_position, mission, status, speed_pct, ships, arrives_at, created_at, COALESCE(alliance_group_id, 0)
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
		if err := rows.Scan(&f.ID, &f.PlayerID, &f.OriginPlanetID, &f.TargetGalaxy, &f.TargetSystem, &f.TargetPosition, &f.Mission, &f.Status, &f.SpeedPct, &shipsJSON, &arrivesAt, &f.CreatedAt, &f.AllianceGroupID); err != nil {
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
		SELECT id, player_id, origin_planet_id, target_galaxy, target_system, target_position, mission, status, speed_pct, ships, arrives_at, created_at, COALESCE(alliance_group_id, 0)
		FROM fleet.fleets WHERE id = $1
	`, fleetID).Scan(&f.ID, &f.PlayerID, &f.OriginPlanetID, &f.TargetGalaxy, &f.TargetSystem, &f.TargetPosition, &f.Mission, &f.Status, &f.SpeedPct, &shipsJSON, &arrivesAt, &f.CreatedAt, &f.AllianceGroupID)
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

func (r *PostgresRepository) UpdateFleetOrigin(ctx context.Context, fleetID, newOriginPlanetID int) error {
	_, err := r.pool.Exec(ctx, `UPDATE fleet.fleets SET origin_planet_id = $1, status = 'stationed' WHERE id = $2`, newOriginPlanetID, fleetID)
	return err
}

func (r *PostgresRepository) GetAttackCooldown(ctx context.Context, attackerID, targetGalaxy, targetSystem, targetPosition int) (*time.Time, error) {
	var lastAttackAt *time.Time
	err := r.pool.QueryRow(ctx,
		`SELECT last_attack_at FROM fleet.attack_cooldowns WHERE attacker_id = $1 AND target_galaxy = $2 AND target_system = $3 AND target_position = $4`,
		attackerID, targetGalaxy, targetSystem, targetPosition,
	).Scan(&lastAttackAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get attack cooldown: %w", err)
	}
	return lastAttackAt, nil
}

func (r *PostgresRepository) UpsertAttackCooldown(ctx context.Context, attackerID, targetGalaxy, targetSystem, targetPosition int, lastAttackAt time.Time) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO fleet.attack_cooldowns (attacker_id, target_galaxy, target_system, target_position, last_attack_at)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (attacker_id, target_galaxy, target_system, target_position)
		 DO UPDATE SET last_attack_at = $5`,
		attackerID, targetGalaxy, targetSystem, targetPosition, lastAttackAt,
	)
	return err
}

func (r *PostgresRepository) GetArrivedFleets(ctx context.Context) ([]Fleet, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, player_id, origin_planet_id, target_galaxy, target_system, target_position, mission, status, speed_pct, ships, arrives_at, created_at, COALESCE(alliance_group_id, 0)
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
		if err := rows.Scan(&f.ID, &f.PlayerID, &f.OriginPlanetID, &f.TargetGalaxy, &f.TargetSystem, &f.TargetPosition, &f.Mission, &f.Status, &f.SpeedPct, &shipsJSON, &arrivesAt, &f.CreatedAt, &f.AllianceGroupID); err != nil {
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

func (r *PostgresRepository) GetACSGroupFleets(ctx context.Context, allianceGroupID int) ([]Fleet, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, player_id, origin_planet_id, target_galaxy, target_system, target_position, mission, status, speed_pct, ships, arrives_at, created_at, COALESCE(alliance_group_id, 0)
		FROM fleet.fleets
		WHERE alliance_group_id = $1
	`, allianceGroupID)
	if err != nil {
		return nil, fmt.Errorf("get ACS group fleets: %w", err)
	}
	defer rows.Close()

	var fleets []Fleet
	for rows.Next() {
		var f Fleet
		var shipsJSON []byte
		var arrivesAt *time.Time
		if err := rows.Scan(&f.ID, &f.PlayerID, &f.OriginPlanetID, &f.TargetGalaxy, &f.TargetSystem, &f.TargetPosition, &f.Mission, &f.Status, &f.SpeedPct, &shipsJSON, &arrivesAt, &f.CreatedAt, &f.AllianceGroupID); err != nil {
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

func (r *PostgresRepository) GetACSDefendFleets(ctx context.Context, targetGalaxy, targetSystem, targetPosition int) ([]Fleet, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, player_id, origin_planet_id, target_galaxy, target_system, target_position, mission, status, speed_pct, ships, arrives_at, created_at, COALESCE(alliance_group_id, 0)
		FROM fleet.fleets
		WHERE mission = 'acs_defend' AND status = 'stationed'
		AND target_galaxy = $1 AND target_system = $2 AND target_position = $3
	`, targetGalaxy, targetSystem, targetPosition)
	if err != nil {
		return nil, fmt.Errorf("get ACS defend fleets: %w", err)
	}
	defer rows.Close()

	var fleets []Fleet
	for rows.Next() {
		var f Fleet
		var shipsJSON []byte
		var arrivesAt *time.Time
		if err := rows.Scan(&f.ID, &f.PlayerID, &f.OriginPlanetID, &f.TargetGalaxy, &f.TargetSystem, &f.TargetPosition, &f.Mission, &f.Status, &f.SpeedPct, &shipsJSON, &arrivesAt, &f.CreatedAt, &f.AllianceGroupID); err != nil {
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

func (r *PostgresRepository) GetDebrisField(ctx context.Context, galaxy, system, position int) (*DebrisField, error) {
	var d DebrisField
	err := r.pool.QueryRow(ctx, `
		SELECT id, galaxy, system, position, metal, crystal
		FROM fleet.debris_fields
		WHERE galaxy = $1 AND system = $2 AND position = $3
	`, galaxy, system, position).Scan(&d.ID, &d.Galaxy, &d.System, &d.Position, &d.Metal, &d.Crystal)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get debris field: %w", err)
	}
	return &d, nil
}

func (r *PostgresRepository) UpsertDebrisField(ctx context.Context, galaxy, system, position, metal, crystal int) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO fleet.debris_fields (galaxy, system, position, metal, crystal)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (galaxy, system, position)
		DO UPDATE SET metal = fleet.debris_fields.metal + $4, crystal = fleet.debris_fields.crystal + $5
	`, galaxy, system, position, metal, crystal)
	return err
}

func (r *PostgresRepository) UpdateDebrisField(ctx context.Context, galaxy, system, position, metal, crystal int) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE fleet.debris_fields SET metal = $1, crystal = $2
		WHERE galaxy = $3 AND system = $4 AND position = $5
	`, metal, crystal, galaxy, system, position)
	return err
}

type mockRepo struct {
	fleets     []Fleet
	nextID     int
	cooldowns  map[string]time.Time
	debrisFields []DebrisField
}

func newMockRepo() *mockRepo {
	return &mockRepo{nextID: 1, cooldowns: make(map[string]time.Time)}
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

func (m *mockRepo) UpdateFleetOrigin(ctx context.Context, fleetID, newOriginPlanetID int) error {
	for i, f := range m.fleets {
		if f.ID == fleetID {
			m.fleets[i].OriginPlanetID = newOriginPlanetID
			m.fleets[i].Status = "stationed"
			return nil
		}
	}
	return nil
}

func (m *mockRepo) GetAttackCooldown(ctx context.Context, attackerID, targetGalaxy, targetSystem, targetPosition int) (*time.Time, error) {
	key := fmt.Sprintf("%d-%d-%d-%d", attackerID, targetGalaxy, targetSystem, targetPosition)
	if t, ok := m.cooldowns[key]; ok {
		return &t, nil
	}
	return nil, nil
}

func (m *mockRepo) UpsertAttackCooldown(ctx context.Context, attackerID, targetGalaxy, targetSystem, targetPosition int, lastAttackAt time.Time) error {
	key := fmt.Sprintf("%d-%d-%d-%d", attackerID, targetGalaxy, targetSystem, targetPosition)
	m.cooldowns[key] = lastAttackAt
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

func (m *mockRepo) GetACSGroupFleets(ctx context.Context, allianceGroupID int) ([]Fleet, error) {
	var result []Fleet
	for _, f := range m.fleets {
		if f.AllianceGroupID == allianceGroupID {
			result = append(result, f)
		}
	}
	return result, nil
}

func (m *mockRepo) GetACSDefendFleets(ctx context.Context, targetGalaxy, targetSystem, targetPosition int) ([]Fleet, error) {
	var result []Fleet
	for _, f := range m.fleets {
		if f.Mission == "acs_defend" && f.Status == "stationed" &&
			f.TargetGalaxy == targetGalaxy && f.TargetSystem == targetSystem && f.TargetPosition == targetPosition {
			result = append(result, f)
		}
	}
	return result, nil
}

func (m *mockRepo) GetDebrisField(ctx context.Context, galaxy, system, position int) (*DebrisField, error) {
	for _, d := range m.debrisFields {
		if d.Galaxy == galaxy && d.System == system && d.Position == position {
			return &d, nil
		}
	}
	return nil, nil
}

func (m *mockRepo) UpsertDebrisField(ctx context.Context, galaxy, system, position, metal, crystal int) error {
	for i, d := range m.debrisFields {
		if d.Galaxy == galaxy && d.System == system && d.Position == position {
			m.debrisFields[i].Metal += metal
			m.debrisFields[i].Crystal += crystal
			return nil
		}
	}
	m.debrisFields = append(m.debrisFields, DebrisField{
		Galaxy: galaxy, System: system, Position: position,
		Metal: metal, Crystal: crystal,
	})
	return nil
}

func (m *mockRepo) UpdateDebrisField(ctx context.Context, galaxy, system, position, metal, crystal int) error {
	for i, d := range m.debrisFields {
		if d.Galaxy == galaxy && d.System == system && d.Position == position {
			m.debrisFields[i].Metal = metal
			m.debrisFields[i].Crystal = crystal
			return nil
		}
	}
	return nil
}
