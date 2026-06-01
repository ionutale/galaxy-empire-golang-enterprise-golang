package main

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	GetPlayerTutorial(ctx context.Context, playerID int) (*PlayerTutorial, error)
	CreatePlayerTutorial(ctx context.Context, playerID int) error
	AdvanceStep(ctx context.Context, playerID int) error
	CompleteTutorial(ctx context.Context, playerID int) error
	CheckBuildingLevel(ctx context.Context, playerID int, buildingType string) (bool, error)
	CheckShipCount(ctx context.Context, playerID int, shipType string) (bool, error)
	CheckExpeditionExists(ctx context.Context, playerID int) (bool, error)
	CheckAttackExists(ctx context.Context, playerID int) (bool, error)
	AddDarkMatter(ctx context.Context, playerID int, amount int) error
	AddPlayerResources(ctx context.Context, playerID int, metal, crystal int) error
}

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) GetPlayerTutorial(ctx context.Context, playerID int) (*PlayerTutorial, error) {
	var pt PlayerTutorial
	err := r.pool.QueryRow(ctx, `
		SELECT player_id, current_step, completed, started_at, completed_at
		FROM tutorial.player_tutorial
		WHERE player_id = $1
	`, playerID).Scan(&pt.PlayerID, &pt.CurrentStep, &pt.Completed, &pt.StartedAt, &pt.CompletedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get player tutorial: %w", err)
	}
	return &pt, nil
}

func (r *PostgresRepository) CreatePlayerTutorial(ctx context.Context, playerID int) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO tutorial.player_tutorial (player_id, current_step, completed, started_at)
		VALUES ($1, 1, FALSE, NOW())
		ON CONFLICT (player_id) DO NOTHING
	`, playerID)
	return err
}

func (r *PostgresRepository) AdvanceStep(ctx context.Context, playerID int) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE tutorial.player_tutorial
		SET current_step = current_step + 1
		WHERE player_id = $1 AND completed = FALSE
	`, playerID)
	return err
}

func (r *PostgresRepository) CompleteTutorial(ctx context.Context, playerID int) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE tutorial.player_tutorial
		SET completed = TRUE, completed_at = NOW()
		WHERE player_id = $1
	`, playerID)
	return err
}

func (r *PostgresRepository) CheckBuildingLevel(ctx context.Context, playerID int, buildingType string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM planet.buildings b
			JOIN planet.planets p ON p.id = b.planet_id
			WHERE p.user_id = $1 AND b.type = $2 AND b.level >= 1
		)
	`, playerID, buildingType).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check building level: %w", err)
	}
	return exists, nil
}

func (r *PostgresRepository) CheckShipCount(ctx context.Context, playerID int, shipType string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM planet.player_ships ps
			JOIN planet.planets p ON p.id = ps.planet_id
			WHERE p.user_id = $1 AND ps.ship_type = $2 AND ps.quantity >= 1
		)
	`, playerID, shipType).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check ship count: %w", err)
	}
	return exists, nil
}

func (r *PostgresRepository) CheckExpeditionExists(ctx context.Context, playerID int) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM nebula.expeditions WHERE player_id = $1)
	`, playerID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check expedition: %w", err)
	}
	return exists, nil
}

func (r *PostgresRepository) CheckAttackExists(ctx context.Context, playerID int) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM fleet.fleet_missions WHERE player_id = $1 AND mission_type = 'attack')
	`, playerID).Scan(&exists)
	if err != nil {
		// Try combat table as fallback
		err2 := r.pool.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM combat.combats WHERE attacker_id = $1)
		`, playerID).Scan(&exists)
		if err2 != nil {
			return false, nil
		}
		return exists, nil
	}
	return exists, nil
}

func (r *PostgresRepository) AddDarkMatter(ctx context.Context, playerID int, amount int) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO nebula.player_dark_matter (player_id, balance, total_earned)
		VALUES ($1, $2, $2)
		ON CONFLICT (player_id) DO UPDATE
		SET balance = nebula.player_dark_matter.balance + $2,
		    total_earned = nebula.player_dark_matter.total_earned + $2
	`, playerID, amount)
	if err != nil {
		return fmt.Errorf("add dark matter: %w", err)
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO nebula.dm_transactions (player_id, amount, balance_after, reason)
		VALUES ($1, $2, (SELECT COALESCE(balance, 0) FROM nebula.player_dark_matter WHERE player_id = $1), 'tutorial_reward')
	`, playerID, amount)
	return err
}

func (r *PostgresRepository) AddPlayerResources(ctx context.Context, playerID int, metal, crystal int) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE planet.planets
		SET metal = metal + $1,
		    crystal = crystal + $2,
		    resources_updated_at = NOW()
		WHERE user_id = $3
	`, metal, crystal, playerID)
	return err
}
