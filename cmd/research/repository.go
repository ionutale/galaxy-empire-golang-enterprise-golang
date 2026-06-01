package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	CreateResearch(ctx context.Context, playerID, planetID int, techType string, targetLevel int, completesAt time.Time) (ResearchQueue, error)
	GetActiveResearch(ctx context.Context, playerID int, techType string) (*ResearchQueue, error)
	ListActiveResearch(ctx context.Context, playerID int) ([]ResearchQueue, error)
	GetCompletedResearch(ctx context.Context) ([]ResearchQueue, error)
	CompleteResearch(ctx context.Context, id int) error
	CancelResearchWithRefund(ctx context.Context, id, playerID int, refundMetal, refundCrystal, refundGas int) error
	CountActiveResearch(ctx context.Context, playerID int) (int, error)
	TryCompleteResearch(ctx context.Context, id int) (bool, error)
	SpeedUpResearch(ctx context.Context, playerID, seconds int) error
}

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) CreateResearch(ctx context.Context, playerID, planetID int, techType string, targetLevel int, completesAt time.Time) (ResearchQueue, error) {
	var q ResearchQueue
	err := r.pool.QueryRow(ctx,
		`INSERT INTO research.research_queue (player_id, planet_id, tech_type, target_level, completes_at)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, player_id, planet_id, tech_type, target_level, started_at, completes_at, completed, cancelled`,
		playerID, planetID, techType, targetLevel, completesAt,
	).Scan(&q.ID, &q.PlayerID, &q.PlanetID, &q.TechType, &q.TargetLevel, &q.StartedAt, &q.CompletesAt, &q.Completed, &q.Cancelled)
	if err != nil {
		return ResearchQueue{}, fmt.Errorf("create research: %w", err)
	}
	return q, nil
}

func (r *PostgresRepository) GetActiveResearch(ctx context.Context, playerID int, techType string) (*ResearchQueue, error) {
	var q ResearchQueue
	err := r.pool.QueryRow(ctx,
		`SELECT id, player_id, planet_id, tech_type, target_level, started_at, completes_at, completed, cancelled
		 FROM research.research_queue
		 WHERE player_id = $1 AND tech_type = $2 AND completed = FALSE AND cancelled = FALSE`,
		playerID, techType,
	).Scan(&q.ID, &q.PlayerID, &q.PlanetID, &q.TechType, &q.TargetLevel, &q.StartedAt, &q.CompletesAt, &q.Completed, &q.Cancelled)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get active research: %w", err)
	}
	return &q, nil
}

func (r *PostgresRepository) ListActiveResearch(ctx context.Context, playerID int) ([]ResearchQueue, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, player_id, planet_id, tech_type, target_level, started_at, completes_at, completed, cancelled
		 FROM research.research_queue
		 WHERE player_id = $1 AND completed = FALSE AND cancelled = FALSE
		 ORDER BY started_at`,
		playerID,
	)
	if err != nil {
		return nil, fmt.Errorf("list active research: %w", err)
	}
	defer rows.Close()

	var queue []ResearchQueue
	for rows.Next() {
		var q ResearchQueue
		if err := rows.Scan(&q.ID, &q.PlayerID, &q.PlanetID, &q.TechType, &q.TargetLevel, &q.StartedAt, &q.CompletesAt, &q.Completed, &q.Cancelled); err != nil {
			return nil, fmt.Errorf("scan research: %w", err)
		}
		queue = append(queue, q)
	}
	if queue == nil {
		queue = []ResearchQueue{}
	}
	return queue, rows.Err()
}

func (r *PostgresRepository) GetCompletedResearch(ctx context.Context) ([]ResearchQueue, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, player_id, planet_id, tech_type, target_level, started_at, completes_at, completed, cancelled
		 FROM research.research_queue
		 WHERE completed = FALSE AND cancelled = FALSE AND completes_at <= NOW()
		 ORDER BY completes_at`,
	)
	if err != nil {
		return nil, fmt.Errorf("get completed research: %w", err)
	}
	defer rows.Close()

	var queue []ResearchQueue
	for rows.Next() {
		var q ResearchQueue
		if err := rows.Scan(&q.ID, &q.PlayerID, &q.PlanetID, &q.TechType, &q.TargetLevel, &q.StartedAt, &q.CompletesAt, &q.Completed, &q.Cancelled); err != nil {
			return nil, fmt.Errorf("scan completed research: %w", err)
		}
		queue = append(queue, q)
	}
	if queue == nil {
		queue = []ResearchQueue{}
	}
	return queue, rows.Err()
}

func (r *PostgresRepository) CompleteResearch(ctx context.Context, id int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE research.research_queue SET completed = TRUE WHERE id = $1`,
		id,
	)
	if err != nil {
		return fmt.Errorf("complete research: %w", err)
	}
	return nil
}

func (r *PostgresRepository) CancelResearchWithRefund(ctx context.Context, id, playerID int, refundMetal, refundCrystal, refundGas int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE research.research_queue SET cancelled = TRUE WHERE id = $1 AND completed = FALSE`,
		id,
	)
	if err != nil {
		return fmt.Errorf("cancel research: %w", err)
	}
	return nil
}

func (r *PostgresRepository) CountActiveResearch(ctx context.Context, playerID int) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM research.research_queue
		 WHERE player_id = $1 AND completed = FALSE AND cancelled = FALSE`,
		playerID,
	).Scan(&count)
	return count, err
}

func (r *PostgresRepository) TryCompleteResearch(ctx context.Context, id int) (bool, error) {
	tag, err := r.pool.Exec(ctx,
		`UPDATE research.research_queue SET completed = TRUE WHERE id = $1 AND completed = FALSE`,
		id,
	)
	if err != nil {
		return false, fmt.Errorf("try complete research: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

func (r *PostgresRepository) SpeedUpResearch(ctx context.Context, playerID, seconds int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE research.research_queue
		 SET completes_at = GREATEST(NOW(), completes_at - ($2 * INTERVAL '1 second'))
		 WHERE player_id = $1 AND completed = FALSE AND cancelled = FALSE`,
		playerID, seconds,
	)
	return err
}
