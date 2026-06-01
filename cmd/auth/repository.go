package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	Create(ctx context.Context, email, passwordHash string) (User, error)
	FindByEmail(ctx context.Context, email string) (User, error)
	FindByID(ctx context.Context, id int) (User, error)
	SetVacationStart(ctx context.Context, id int) error
	SetVacationEnabled(ctx context.Context, id int, enabled bool) error
	ClearVacationMode(ctx context.Context, id int) error
	GetVacationStatus(ctx context.Context, id int) (bool, *time.Time, error)
	GetVacationEnabled(ctx context.Context, id int) (bool, error)
}

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, email, passwordHash string) (User, error) {
	var user User
	err := r.pool.QueryRow(ctx,
		`INSERT INTO auth.users (email, password_hash)
		 VALUES ($1, $2)
		 RETURNING id, email, password_hash, created_at, updated_at, vacation_mode_enabled, vacation_mode_started_at`,
		email, passwordHash,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt, &user.VacationModeEnabled, &user.VacationModeStartedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return User{}, ErrDuplicateEmail
		}
		return User{}, fmt.Errorf("create user: %w", err)
	}
	return user, nil
}

func (r *PostgresRepository) FindByEmail(ctx context.Context, email string) (User, error) {
	var user User
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, created_at, updated_at, vacation_mode_enabled, vacation_mode_started_at
		 FROM auth.users WHERE email = $1`,
		email,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt, &user.VacationModeEnabled, &user.VacationModeStartedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, fmt.Errorf("find user by email: %w", err)
	}
	return user, nil
}

func (r *PostgresRepository) FindByID(ctx context.Context, id int) (User, error) {
	var user User
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, created_at, updated_at, vacation_mode_enabled, vacation_mode_started_at
		 FROM auth.users WHERE id = $1`,
		id,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt, &user.VacationModeEnabled, &user.VacationModeStartedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, fmt.Errorf("find user by id: %w", err)
	}
	return user, nil
}

func (r *PostgresRepository) SetVacationStart(ctx context.Context, id int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE auth.users SET vacation_mode_started_at = NOW(), updated_at = NOW() WHERE id = $1`,
		id,
	)
	return err
}

func (r *PostgresRepository) SetVacationEnabled(ctx context.Context, id int, enabled bool) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE auth.users SET vacation_mode_enabled = $1, updated_at = NOW() WHERE id = $2`,
		enabled, id,
	)
	return err
}

func (r *PostgresRepository) ClearVacationMode(ctx context.Context, id int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE auth.users SET vacation_mode_enabled = FALSE, vacation_mode_started_at = NULL, updated_at = NOW() WHERE id = $1`,
		id,
	)
	return err
}

func (r *PostgresRepository) GetVacationStatus(ctx context.Context, id int) (bool, *time.Time, error) {
	var enabled bool
	var startedAt *time.Time
	err := r.pool.QueryRow(ctx,
		`SELECT vacation_mode_enabled, vacation_mode_started_at FROM auth.users WHERE id = $1`,
		id,
	).Scan(&enabled, &startedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil, ErrUserNotFound
		}
		return false, nil, fmt.Errorf("get vacation status: %w", err)
	}
	return enabled, startedAt, nil
}

func (r *PostgresRepository) GetVacationEnabled(ctx context.Context, id int) (bool, error) {
	var enabled bool
	err := r.pool.QueryRow(ctx,
		`SELECT vacation_mode_enabled FROM auth.users WHERE id = $1`,
		id,
	).Scan(&enabled)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, ErrUserNotFound
		}
		return false, fmt.Errorf("get vacation enabled: %w", err)
	}
	return enabled, nil
}
