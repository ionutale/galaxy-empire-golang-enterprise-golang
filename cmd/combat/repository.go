package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	CreateCombatReport(ctx context.Context, r CombatReport) (int, error)
	GetCombatReport(ctx context.Context, id int) (CombatReport, error)
	ListPlayerCombatReports(ctx context.Context, playerID int) ([]CombatReport, error)
	CleanupExpiredReports(ctx context.Context) error
	CreateMoon(ctx context.Context, galaxy, system, position int, name string, size int) error
	GetMoon(ctx context.Context, galaxy, system, position int) (*Moon, error)
}

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) CreateCombatReport(ctx context.Context, report CombatReport) (int, error) {
	shipsBeforeJSON, _ := json.Marshal(struct {
		Attacker map[string]int `json:"attacker"`
		Defender map[string]int `json:"defender"`
	}{
		Attacker: report.AttackerShipsBefore,
		Defender: report.DefenderShipsBefore,
	})

	shipsAfterJSON, _ := json.Marshal(struct {
		Attacker map[string]int `json:"attacker"`
		Defender map[string]int `json:"defender"`
	}{
		Attacker: report.AttackerShipsAfter,
		Defender: report.DefenderShipsAfter,
	})

	roundsJSON, _ := json.Marshal(report.Rounds)
	lootJSON, _ := json.Marshal(report.AttackerLoot)
	lostResJSON, _ := json.Marshal(report.DefenderLostRes)
	var missileResultJSON []byte
	if report.MissileResult != nil {
		missileResultJSON, _ = json.Marshal(report.MissileResult)
	}

	var id int
	err := r.pool.QueryRow(ctx, `
		INSERT INTO combat.combat_reports
			(attacker_player_id, defender_player_id, target_galaxy, target_system, target_position,
			 attacker_ships_before, defender_ships_before, attacker_ships_after, defender_ships_after,
			 rounds, attacker_won, attacker_loot, defender_lost_resources,
			 debris_metal, debris_crystal, moon_created, moon_size, missile_result, expires_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)
		RETURNING id
	`,
		report.AttackerPlayerID, report.DefenderPlayerID,
		report.TargetGalaxy, report.TargetSystem, report.TargetPosition,
		shipsBeforeJSON, shipsAfterJSON,
		roundsJSON, report.AttackerWon,
		lootJSON, lostResJSON,
		report.DebrisMetal, report.DebrisCrystal,
		report.MoonCreated, report.MoonSize,
		missileResultJSON,
		time.Now().Add(7*24*time.Hour),
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("create combat report: %w", err)
	}
	return id, nil
}

func (r *PostgresRepository) GetCombatReport(ctx context.Context, id int) (CombatReport, error) {
	var report CombatReport
	var shipsBeforeJSON, shipsAfterJSON, roundsJSON, lootJSON, lostResJSON, missileResultJSON []byte

	err := r.pool.QueryRow(ctx, `
		SELECT id, attacker_player_id, defender_player_id,
		       target_galaxy, target_system, target_position,
		       attacker_ships_before, defender_ships_before,
		       attacker_ships_after, defender_ships_after,
		       rounds, attacker_won, attacker_loot, defender_lost_resources,
		       debris_metal, debris_crystal, moon_created, moon_size,
		       missile_result, created_at, expires_at
		FROM combat.combat_reports
		WHERE id = $1
	`, id).Scan(
		&report.ID, &report.AttackerPlayerID, &report.DefenderPlayerID,
		&report.TargetGalaxy, &report.TargetSystem, &report.TargetPosition,
		&shipsBeforeJSON, &shipsAfterJSON,
		&roundsJSON, &report.AttackerWon,
		&lootJSON, &lostResJSON,
		&report.DebrisMetal, &report.DebrisCrystal,
		&report.MoonCreated, &report.MoonSize,
		&missileResultJSON,
		&report.CreatedAt, &report.ExpiresAt,
	)
	if err != nil {
		return CombatReport{}, fmt.Errorf("get combat report: %w", err)
	}

	var shipsBefore, shipsAfter struct {
		Attacker map[string]int `json:"attacker"`
		Defender map[string]int `json:"defender"`
	}
	json.Unmarshal(shipsBeforeJSON, &shipsBefore)
	json.Unmarshal(shipsAfterJSON, &shipsAfter)
	report.AttackerShipsBefore = shipsBefore.Attacker
	report.DefenderShipsBefore = shipsBefore.Defender
	report.AttackerShipsAfter = shipsAfter.Attacker
	report.DefenderShipsAfter = shipsAfter.Defender

	json.Unmarshal(roundsJSON, &report.Rounds)
	json.Unmarshal(lootJSON, &report.AttackerLoot)
	json.Unmarshal(lostResJSON, &report.DefenderLostRes)
	if len(missileResultJSON) > 0 {
		json.Unmarshal(missileResultJSON, &report.MissileResult)
	}

	return report, nil
}

func (r *PostgresRepository) ListPlayerCombatReports(ctx context.Context, playerID int) ([]CombatReport, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, attacker_player_id, defender_player_id,
		       target_galaxy, target_system, target_position,
		       attacker_won, created_at, expires_at
		FROM combat.combat_reports
		WHERE attacker_player_id = $1 OR defender_player_id = $1
		ORDER BY created_at DESC
	`, playerID)
	if err != nil {
		return nil, fmt.Errorf("list combat reports: %w", err)
	}
	defer rows.Close()

	var reports []CombatReport
	for rows.Next() {
		var r CombatReport
		if err := rows.Scan(
			&r.ID, &r.AttackerPlayerID, &r.DefenderPlayerID,
			&r.TargetGalaxy, &r.TargetSystem, &r.TargetPosition,
			&r.AttackerWon, &r.CreatedAt, &r.ExpiresAt,
		); err != nil {
			return nil, fmt.Errorf("scan report: %w", err)
		}
		reports = append(reports, r)
	}
	return reports, nil
}

func (r *PostgresRepository) CleanupExpiredReports(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM combat.combat_reports WHERE expires_at < NOW()
	`)
	if err != nil {
		return fmt.Errorf("cleanup expired reports: %w", err)
	}
	return nil
}

func newMockRepo() *mockRepo {
	return &mockRepo{nextID: 1}
}

func (m *mockRepo) CreateCombatReport(ctx context.Context, report CombatReport) (int, error) {
	report.ID = m.nextID
	m.nextID++
	report.CreatedAt = time.Now()
	report.ExpiresAt = time.Now().Add(7 * 24 * time.Hour)
	m.reports = append(m.reports, report)
	return report.ID, nil
}

func (m *mockRepo) GetCombatReport(ctx context.Context, id int) (CombatReport, error) {
	for _, r := range m.reports {
		if r.ID == id {
			return r, nil
		}
	}
	return CombatReport{}, fmt.Errorf("report not found")
}

func (m *mockRepo) ListPlayerCombatReports(ctx context.Context, playerID int) ([]CombatReport, error) {
	var result []CombatReport
	for _, r := range m.reports {
		if r.AttackerPlayerID == playerID || r.DefenderPlayerID == playerID {
			result = append(result, r)
		}
	}
	return result, nil
}

type mockMoon struct {
	Galaxy   int
	System   int
	Position int
	Name     string
	Size     int
}

type mockRepo struct {
	reports []CombatReport
	nextID  int
	moons   []mockMoon
}

func (m *mockRepo) CreateMoon(ctx context.Context, galaxy, system, position int, name string, size int) error {
	m.moons = append(m.moons, mockMoon{Galaxy: galaxy, System: system, Position: position, Name: name, Size: size})
	return nil
}

func (m *mockRepo) GetMoon(ctx context.Context, galaxy, system, position int) (*Moon, error) {
	for _, mo := range m.moons {
		if mo.Galaxy == galaxy && mo.System == system && mo.Position == position {
			return &Moon{
				Galaxy:   mo.Galaxy,
				System:   mo.System,
				Position: mo.Position,
				Name:     mo.Name,
				Size:     mo.Size,
			}, nil
		}
	}
	return nil, fmt.Errorf("moon not found")
}

func (m *mockRepo) CleanupExpiredReports(ctx context.Context) error {
	var kept []CombatReport
	for _, r := range m.reports {
		if r.ExpiresAt.After(time.Now()) {
			kept = append(kept, r)
		}
	}
	m.reports = kept
	return nil
}

func (r *PostgresRepository) CreateMoon(ctx context.Context, galaxy, system, position int, name string, size int) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO fleet.moons (galaxy, system, position, name, size)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (galaxy, system, position) DO NOTHING
	`, galaxy, system, position, name, size)
	if err != nil {
		return fmt.Errorf("create moon: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetMoon(ctx context.Context, galaxy, system, position int) (*Moon, error) {
	var m Moon
	err := r.pool.QueryRow(ctx, `
		SELECT id, galaxy, system, position, COALESCE(player_id, 0), name, size, created_at
		FROM fleet.moons
		WHERE galaxy = $1 AND system = $2 AND position = $3
	`, galaxy, system, position).Scan(
		&m.ID, &m.Galaxy, &m.System, &m.Position, &m.PlayerID, &m.Name, &m.Size, &m.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get moon: %w", err)
	}
	return &m, nil
}

func runCombatMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, `
		CREATE SCHEMA IF NOT EXISTS combat;
	`); err != nil {
		return fmt.Errorf("create schema: %w", err)
	}

	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS combat.combat_reports (
			id SERIAL PRIMARY KEY,
			attacker_player_id INT NOT NULL,
			defender_player_id INT NOT NULL,
			target_galaxy INT NOT NULL,
			target_system INT NOT NULL,
			target_position INT NOT NULL,
			attacker_ships_before JSONB,
			defender_ships_before JSONB,
			attacker_ships_after JSONB,
			defender_ships_after JSONB,
			rounds JSONB,
			attacker_won BOOLEAN NOT NULL,
			attacker_loot JSONB,
			defender_lost_resources JSONB,
			debris_metal INT NOT NULL DEFAULT 0,
			debris_crystal INT NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			expires_at TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '7 days'
		);
	`); err != nil {
		return fmt.Errorf("create table: %w", err)
	}

	if _, err := pool.Exec(ctx, `
		ALTER TABLE combat.combat_reports
		ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '7 days';
	`); err != nil {
		return fmt.Errorf("add expires_at column: %w", err)
	}

	if _, err := pool.Exec(ctx, `
		ALTER TABLE combat.combat_reports
		ADD COLUMN IF NOT EXISTS moon_created BOOLEAN NOT NULL DEFAULT FALSE;
	`); err != nil {
		return fmt.Errorf("add moon_created column: %w", err)
	}

	if _, err := pool.Exec(ctx, `
		ALTER TABLE combat.combat_reports
		ADD COLUMN IF NOT EXISTS moon_size INT NOT NULL DEFAULT 0;
	`); err != nil {
		return err
	}

	if _, err := pool.Exec(ctx, `
		ALTER TABLE combat.combat_reports
		ADD COLUMN IF NOT EXISTS missile_result JSONB;
	`); err != nil {
		return err
	}

	if _, err := pool.Exec(ctx, `
		CREATE SCHEMA IF NOT EXISTS fleet;
	`); err != nil {
		return fmt.Errorf("create fleet schema: %w", err)
	}

	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS fleet.moons (
			id SERIAL PRIMARY KEY,
			galaxy INT NOT NULL,
			system INT NOT NULL,
			position INT NOT NULL,
			player_id INT,
			name VARCHAR(100) NOT NULL DEFAULT 'Moon',
			size INT NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE(galaxy, system, position)
		);
	`); err != nil {
		return fmt.Errorf("create moons table: %w", err)
	}

	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
