package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrPlanetNotFound = errors.New("planet not found")

type Repository interface {
	FindByUserID(ctx context.Context, userID int) (Planet, error)
	Create(ctx context.Context, userID int) (Planet, error)
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
		`SELECT id, user_id, name, metal, crystal, gas, energy, galaxy, system, position
		 FROM planet.planets WHERE user_id = $1`,
		userID,
	).Scan(&p.ID, &p.UserID, &p.Name, &p.Metal, &p.Crystal, &p.Gas, &p.Energy, &p.Galaxy, &p.System, &p.Position)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Planet{}, ErrPlanetNotFound
		}
		return Planet{}, fmt.Errorf("find planet by user id: %w", err)
	}
	return p, nil
}

func (r *PostgresRepository) Create(ctx context.Context, userID int) (Planet, error) {
	var p Planet
	err := r.pool.QueryRow(ctx,
		`INSERT INTO planet.planets (user_id)
		 VALUES ($1)
		 RETURNING id, user_id, name, metal, crystal, gas, energy, galaxy, system, position`,
		userID,
	).Scan(&p.ID, &p.UserID, &p.Name, &p.Metal, &p.Crystal, &p.Gas, &p.Energy, &p.Galaxy, &p.System, &p.Position)
	if err != nil {
		return Planet{}, fmt.Errorf("create planet: %w", err)
	}
	return p, nil
}
