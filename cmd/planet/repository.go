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
	Create(ctx context.Context, userID int) (Planet, []Building, error)
	UpdateResources(ctx context.Context, planetID, metal, crystal, gas int, updatedAt time.Time) error
	GetBuildings(ctx context.Context, planetID int) ([]Building, error)
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
		        galaxy, system, position, resources_updated_at
		 FROM planet.planets WHERE user_id = $1`,
		userID,
	).Scan(&p.ID, &p.UserID, &p.Name, &p.Metal, &p.Crystal, &p.Gas, &p.Energy,
		&p.Galaxy, &p.System, &p.Position, &p.ResourcesUpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Planet{}, ErrPlanetNotFound
		}
		return Planet{}, fmt.Errorf("find planet by user id: %w", err)
	}
	return p, nil
}

func (r *PostgresRepository) Create(ctx context.Context, userID int) (Planet, []Building, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Planet{}, nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var p Planet
	err = tx.QueryRow(ctx,
		`INSERT INTO planet.planets (user_id, resources_updated_at)
		 VALUES ($1, NOW())
		 RETURNING id, user_id, name, metal, crystal, gas, energy,
		           galaxy, system, position, resources_updated_at`,
		userID,
	).Scan(&p.ID, &p.UserID, &p.Name, &p.Metal, &p.Crystal, &p.Gas, &p.Energy,
		&p.Galaxy, &p.System, &p.Position, &p.ResourcesUpdatedAt)
	if err != nil {
		return Planet{}, nil, fmt.Errorf("insert planet: %w", err)
	}

	seedTypes := []string{
		"metal_mine", "crystal_mine", "gas_mine", "solar_plant",
		"metal_storage", "crystal_storage", "gas_storage",
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
