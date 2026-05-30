package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrPlanetNotFound = errors.New("planet not found")

type Repository interface {
	FindByUserID(ctx context.Context, userID int) (Planet, error)
	FindByID(ctx context.Context, planetID int) (Planet, error)
	Create(ctx context.Context, userID int) (Planet, []Building, error)
	UpdateResources(ctx context.Context, planetID, metal, crystal, gas int, updatedAt time.Time) error
	UpdateMaxFields(ctx context.Context, planetID, maxFields int) error
	GetBuildings(ctx context.Context, planetID int) ([]Building, error)
	GetBuildingLevel(ctx context.Context, planetID int, buildingType string) (int, error)
	GetActiveQueue(ctx context.Context, planetID int) ([]QueueEntry, error)
	CreateQueueEntry(ctx context.Context, planetID int, buildingType string, targetLevel int, completesAt time.Time) (QueueEntry, error)
	CreateQueueEntryDeconstruct(ctx context.Context, planetID int, buildingType string, targetLevel int, completesAt time.Time) (QueueEntry, error)
	CompleteBuild(ctx context.Context, queueID int, buildingType string, targetLevel int) error
	CancelQueueEntry(ctx context.Context, queueID int) error
	CancelUpgradeWithRefund(ctx context.Context, planetID, queueID, refundMetal, refundCrystal, refundGas int) error
	DeleteBuilding(ctx context.Context, planetID int, buildingType string) error
	UpdateBuildingLevel(ctx context.Context, planetID int, buildingType string, level int) error
	DeconstructComplete(ctx context.Context, planetID, queueID int, buildingType string, targetLevel int, refundMetal, refundCrystal, refundGas, maxFields int) error
	GetTechLevel(ctx context.Context, userID int, techType string) (int, error)
}

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) FindByUserID(ctx context.Context, userID int) (Planet, error) {
	var p Planet
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, name, metal, crystal, gas, energy,
		        galaxy, system, position, max_fields, type, temperature, resources_updated_at
		 FROM planet.planets WHERE user_id = $1`,
		userID,
	).Scan(&p.ID, &p.UserID, &p.Name, &p.Metal, &p.Crystal, &p.Gas, &p.Energy,
		&p.Galaxy, &p.System, &p.Position, &p.MaxFields, &p.Type, &p.Temperature, &p.ResourcesUpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Planet{}, ErrPlanetNotFound
		}
		return Planet{}, fmt.Errorf("find planet by user id: %w", err)
	}
	return p, nil
}

func (r *PostgresRepository) FindByID(ctx context.Context, planetID int) (Planet, error) {
	var p Planet
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, name, metal, crystal, gas, energy,
		        galaxy, system, position, max_fields, type, temperature, resources_updated_at
		 FROM planet.planets WHERE id = $1`,
		planetID,
	).Scan(&p.ID, &p.UserID, &p.Name, &p.Metal, &p.Crystal, &p.Gas, &p.Energy,
		&p.Galaxy, &p.System, &p.Position, &p.MaxFields, &p.Type, &p.Temperature, &p.ResourcesUpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Planet{}, ErrPlanetNotFound
		}
		return Planet{}, fmt.Errorf("find planet by id: %w", err)
	}
	return p, nil
}

func (r *PostgresRepository) Create(ctx context.Context, userID int) (Planet, []Building, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Planet{}, nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	typ, temp := planetTypeAndTemp(7)

	var p Planet
	err = tx.QueryRow(ctx,
		`INSERT INTO planet.planets (user_id, max_fields, type, temperature, resources_updated_at)
		 VALUES ($1, $2, $3, $4, NOW())
		 RETURNING id, user_id, name, metal, crystal, gas, energy,
		           galaxy, system, position, max_fields, type, temperature, resources_updated_at`,
		userID, baseMaxFields, typ, temp,
	).Scan(&p.ID, &p.UserID, &p.Name, &p.Metal, &p.Crystal, &p.Gas, &p.Energy,
		&p.Galaxy, &p.System, &p.Position, &p.MaxFields, &p.Type, &p.Temperature, &p.ResourcesUpdatedAt)
	if err != nil {
		return Planet{}, nil, fmt.Errorf("insert planet: %w", err)
	}

	seedTypes := []string{
		"metal_mine", "crystal_mine", "gas_mine", "solar_plant",
		"metal_storage", "crystal_storage", "gas_storage",
		"robotics_factory", "nanite_factory", "terraformer",
	}
	buildings := make([]Building, 0, len(seedTypes))
	for _, bType := range seedTypes {
		var b Building
		err := tx.QueryRow(ctx,
			`INSERT INTO planet.buildings (planet_id, type, level)
			 VALUES ($1, $2, 1)
			 RETURNING id, planet_id, type, level`,
			p.ID, bType,
		).Scan(&b.ID, &b.PlanetID, &b.Type, &b.Level)
		if err != nil {
			return Planet{}, nil, fmt.Errorf("seed building %s: %w", bType, err)
		}
		buildings = append(buildings, b)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO planet.player_technologies (user_id, type, level)
		VALUES ($1, 'energy_tech', 3)
		ON CONFLICT (user_id, type) DO NOTHING
	`, userID); err != nil {
		return Planet{}, nil, fmt.Errorf("insert default tech: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Planet{}, nil, fmt.Errorf("commit: %w", err)
	}
	return p, buildings, nil
}

func (r *PostgresRepository) UpdateResources(ctx context.Context, planetID, metal, crystal, gas int, updatedAt time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE planet.planets
		 SET metal = $1, crystal = $2, gas = $3, resources_updated_at = $4
		 WHERE id = $5`,
		metal, crystal, gas, updatedAt, planetID,
	)
	return err
}

func (r *PostgresRepository) UpdateMaxFields(ctx context.Context, planetID, maxFields int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE planet.planets SET max_fields = $1 WHERE id = $2`,
		maxFields, planetID,
	)
	return err
}

func (r *PostgresRepository) GetBuildings(ctx context.Context, planetID int) ([]Building, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, planet_id, type, level
		 FROM planet.buildings WHERE planet_id = $1
		 ORDER BY type`,
		planetID,
	)
	if err != nil {
		return nil, fmt.Errorf("get buildings: %w", err)
	}
	defer rows.Close()

	var buildings []Building
	for rows.Next() {
		var b Building
		if err := rows.Scan(&b.ID, &b.PlanetID, &b.Type, &b.Level); err != nil {
			return nil, fmt.Errorf("scan building: %w", err)
		}
		buildings = append(buildings, b)
	}
	return buildings, nil
}

func (r *PostgresRepository) GetBuildingLevel(ctx context.Context, planetID int, buildingType string) (int, error) {
	var level int
	err := r.pool.QueryRow(ctx,
		`SELECT level FROM planet.buildings WHERE planet_id = $1 AND type = $2`,
		planetID, buildingType,
	).Scan(&level)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrInvalidBuilding
		}
		return 0, fmt.Errorf("get building level: %w", err)
	}
	return level, nil
}

func (r *PostgresRepository) GetActiveQueue(ctx context.Context, planetID int) ([]QueueEntry, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, building_type, target_level, status, completes_at
		 FROM planet.construction_queue
		 WHERE planet_id = $1 AND completed = FALSE
		 ORDER BY started_at`,
		planetID,
	)
	if err != nil {
		return nil, fmt.Errorf("get active queue: %w", err)
	}
	defer rows.Close()

	var queue []QueueEntry
	for rows.Next() {
		var q QueueEntry
		if err := rows.Scan(&q.ID, &q.BuildingType, &q.TargetLevel, &q.Status, &q.CompletesAt); err != nil {
			return nil, fmt.Errorf("scan queue entry: %w", err)
		}
		queue = append(queue, q)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate queue: %w", err)
	}
	return queue, nil
}

func (r *PostgresRepository) CreateQueueEntry(ctx context.Context, planetID int, buildingType string, targetLevel int, completesAt time.Time) (QueueEntry, error) {
	var q QueueEntry
	err := r.pool.QueryRow(ctx,
		`INSERT INTO planet.construction_queue (planet_id, building_type, target_level, status, completes_at)
		 VALUES ($1, $2, $3, 'upgrade', $4)
		 RETURNING id, building_type, target_level, status, completes_at`,
		planetID, buildingType, targetLevel, completesAt,
	).Scan(&q.ID, &q.BuildingType, &q.TargetLevel, &q.Status, &q.CompletesAt)
	if err != nil {
		return QueueEntry{}, fmt.Errorf("create queue entry: %w", err)
	}
	return q, nil
}

func (r *PostgresRepository) CancelUpgradeWithRefund(ctx context.Context, planetID, queueID, refundMetal, refundCrystal, refundGas int) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		`UPDATE planet.planets SET metal = metal + $1, crystal = crystal + $2, gas = gas + $3, resources_updated_at = NOW() WHERE id = $4`,
		refundMetal, refundCrystal, refundGas, planetID,
	); err != nil {
		return fmt.Errorf("refund resources: %w", err)
	}

	if _, err := tx.Exec(ctx,
		`DELETE FROM planet.construction_queue WHERE id = $1 AND completed = FALSE`, queueID,
	); err != nil {
		return fmt.Errorf("cancel queue entry: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *PostgresRepository) CancelQueueEntry(ctx context.Context, queueID int) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM planet.construction_queue WHERE id = $1 AND completed = FALSE`,
		queueID,
	)
	if err != nil {
		return fmt.Errorf("cancel queue entry: %w", err)
	}
	return nil
}

func (r *PostgresRepository) DeleteBuilding(ctx context.Context, planetID int, buildingType string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM planet.buildings WHERE planet_id = $1 AND type = $2`,
		planetID, buildingType,
	)
	if err != nil {
		return fmt.Errorf("delete building: %w", err)
	}
	return nil
}

func (r *PostgresRepository) UpdateBuildingLevel(ctx context.Context, planetID int, buildingType string, level int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE planet.buildings SET level = $1 WHERE planet_id = $2 AND type = $3`,
		level, planetID, buildingType,
	)
	return err
}

func (r *PostgresRepository) CreateQueueEntryDeconstruct(ctx context.Context, planetID int, buildingType string, targetLevel int, completesAt time.Time) (QueueEntry, error) {
	var q QueueEntry
	err := r.pool.QueryRow(ctx,
		`INSERT INTO planet.construction_queue (planet_id, building_type, target_level, status, completes_at)
		 VALUES ($1, $2, $3, 'deconstruct', $4)
		 RETURNING id, building_type, target_level, status, completes_at`,
		planetID, buildingType, targetLevel, completesAt,
	).Scan(&q.ID, &q.BuildingType, &q.TargetLevel, &q.Status, &q.CompletesAt)
	if err != nil {
		return QueueEntry{}, fmt.Errorf("create deconstruct entry: %w", err)
	}
	return q, nil
}

func (r *PostgresRepository) DeconstructComplete(ctx context.Context, planetID, queueID int, buildingType string, targetLevel int, refundMetal, refundCrystal, refundGas, maxFields int) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		`UPDATE planet.planets SET metal = metal + $1, crystal = crystal + $2, gas = gas + $3, resources_updated_at = NOW() WHERE id = $4`,
		refundMetal, refundCrystal, refundGas, planetID,
	); err != nil {
		return fmt.Errorf("refund resources: %w", err)
	}

	if targetLevel == 0 {
		if _, err := tx.Exec(ctx,
			`DELETE FROM planet.buildings WHERE planet_id = $1 AND type = $2`,
			planetID, buildingType,
		); err != nil {
			return fmt.Errorf("delete building: %w", err)
		}
	} else {
		if _, err := tx.Exec(ctx,
			`UPDATE planet.buildings SET level = $1 WHERE planet_id = $2 AND type = $3`,
			targetLevel, planetID, buildingType,
		); err != nil {
			return fmt.Errorf("update building level: %w", err)
		}
	}

	if maxFields > 0 {
		if _, err := tx.Exec(ctx,
			`UPDATE planet.planets SET max_fields = $1 WHERE id = $2`,
			maxFields, planetID,
		); err != nil {
			return fmt.Errorf("update max fields: %w", err)
		}
	}

	if _, err := tx.Exec(ctx,
		`DELETE FROM planet.construction_queue WHERE id = $1 AND completed = FALSE`, queueID,
	); err != nil {
		return fmt.Errorf("cancel queue entry: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *PostgresRepository) GetTechLevel(ctx context.Context, userID int, techType string) (int, error) {
	var level int
	err := r.pool.QueryRow(ctx,
		`SELECT level FROM planet.player_technologies WHERE user_id = $1 AND type = $2`,
		userID, techType,
	).Scan(&level)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}
		return 0, fmt.Errorf("get tech level: %w", err)
	}
	return level, nil
}

func (r *PostgresRepository) CompleteBuild(ctx context.Context, queueID int, buildingType string, targetLevel int) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		`UPDATE planet.construction_queue SET completed = TRUE WHERE id = $1`, queueID,
	); err != nil {
		return fmt.Errorf("mark queue complete: %w", err)
	}

	if _, err := tx.Exec(ctx,
		`UPDATE planet.buildings SET level = $1 WHERE type = $2 AND planet_id = (
			SELECT planet_id FROM planet.construction_queue WHERE id = $3
		)`,
		targetLevel, buildingType, queueID,
	); err != nil {
		return fmt.Errorf("update building level: %w", err)
	}

	return tx.Commit(ctx)
}
