package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	UpsertScore(ctx context.Context, s PlayerScore) error
	GetTop(ctx context.Context, limit, offset int) ([]PlayerScore, int, error)
	GetByPlayerID(ctx context.Context, playerID int) (*PlayerScore, error)
	GetPlayerRank(ctx context.Context, playerID int) (int, error)
	ListAllPlayerIDs(ctx context.Context) ([]int, error)
	SumBuildingLevels(ctx context.Context, playerID int) (int, error)
	SumResearchLevels(ctx context.Context, playerID int) (int, error)
	SumFleetQuantity(ctx context.Context, playerID int) (int, error)
	SumDefenseQuantity(ctx context.Context, playerID int) (int, error)
}

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) UpsertScore(ctx context.Context, s PlayerScore) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO ranking.player_scores (player_id, player_name, total_score, fleet_score, buildings_score, research_score, defense_score, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (player_id)
		DO UPDATE SET
			player_name     = CASE WHEN ranking.player_scores.player_name IS NULL
			                           OR ranking.player_scores.player_name = ''
			                           OR ranking.player_scores.player_name ~ '^Player [0-9]+$'
			                       THEN $2
			                       ELSE ranking.player_scores.player_name END,
			total_score     = $3,
			fleet_score     = $4,
			buildings_score = $5,
			research_score  = $6,
			defense_score   = $7,
			updated_at      = $8
	`,
		s.PlayerID, s.PlayerName, s.TotalScore, s.FleetScore,
		s.BuildingsScore, s.ResearchScore, s.DefenseScore, s.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert score: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetTop(ctx context.Context, limit, offset int) ([]PlayerScore, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM ranking.player_scores`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count scores: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, player_id, player_name, total_score, fleet_score, buildings_score, research_score, defense_score, updated_at
		FROM ranking.player_scores
		ORDER BY total_score DESC, player_id ASC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("get top: %w", err)
	}
	defer rows.Close()

	var scores []PlayerScore
	for rows.Next() {
		var s PlayerScore
		if err := rows.Scan(&s.ID, &s.PlayerID, &s.PlayerName, &s.TotalScore,
			&s.FleetScore, &s.BuildingsScore, &s.ResearchScore, &s.DefenseScore, &s.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan score: %w", err)
		}
		scores = append(scores, s)
	}
	return scores, total, rows.Err()
}

func (r *PostgresRepository) GetByPlayerID(ctx context.Context, playerID int) (*PlayerScore, error) {
	var s PlayerScore
	err := r.pool.QueryRow(ctx, `
		SELECT id, player_id, player_name, total_score, fleet_score, buildings_score, research_score, defense_score, updated_at
		FROM ranking.player_scores
		WHERE player_id = $1
	`, playerID).Scan(&s.ID, &s.PlayerID, &s.PlayerName, &s.TotalScore,
		&s.FleetScore, &s.BuildingsScore, &s.ResearchScore, &s.DefenseScore, &s.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get by player id: %w", err)
	}
	return &s, nil
}

func (r *PostgresRepository) GetPlayerRank(ctx context.Context, playerID int) (int, error) {
	var rank int
	err := r.pool.QueryRow(ctx, `
		SELECT rank FROM (
			SELECT player_id, ROW_NUMBER() OVER (ORDER BY total_score DESC, player_id ASC) AS rank
			FROM ranking.player_scores
		) sub WHERE player_id = $1
	`, playerID).Scan(&rank)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}
		return 0, fmt.Errorf("get player rank: %w", err)
	}
	return rank, nil
}

func (r *PostgresRepository) ListAllPlayerIDs(ctx context.Context) ([]int, error) {
	rows, err := r.pool.Query(ctx, `SELECT DISTINCT user_id FROM planet.planets ORDER BY user_id`)
	if err != nil {
		return nil, fmt.Errorf("list player ids: %w", err)
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan player id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *PostgresRepository) SumBuildingLevels(ctx context.Context, playerID int) (int, error) {
	var total int
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(b.level), 0)
		FROM planet.buildings b
		JOIN planet.planets p ON p.id = b.planet_id
		WHERE p.user_id = $1
	`, playerID).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("sum building levels: %w", err)
	}
	return total, nil
}

func (r *PostgresRepository) SumResearchLevels(ctx context.Context, playerID int) (int, error) {
	var total int
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(level), 0)
		FROM planet.player_technologies
		WHERE user_id = $1
	`, playerID).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("sum research levels: %w", err)
	}
	return total, nil
}

func (r *PostgresRepository) SumFleetQuantity(ctx context.Context, playerID int) (int, error) {
	var total int
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(ps.quantity), 0)
		FROM planet.player_ships ps
		JOIN planet.planets p ON p.id = ps.planet_id
		WHERE p.user_id = $1
	`, playerID).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("sum fleet quantity: %w", err)
	}
	return total, nil
}

func (r *PostgresRepository) SumDefenseQuantity(ctx context.Context, playerID int) (int, error) {
	var total int
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(pd.quantity), 0)
		FROM planet.player_defenses pd
		JOIN planet.planets p ON p.id = pd.planet_id
		WHERE p.user_id = $1
	`, playerID).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("sum defense quantity: %w", err)
	}
	return total, nil
}
