package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	GetQuestDefinitions(ctx context.Context) ([]QuestDefinition, error)
	GetPlayerQuest(ctx context.Context, playerID int, questID string) (*PlayerQuest, error)
	GetPlayerQuests(ctx context.Context, playerID int) ([]PlayerQuest, error)
	UpsertPlayerQuest(ctx context.Context, pq PlayerQuest) error
	ClaimPlayerQuest(ctx context.Context, playerID int, questID string, claimedAt time.Time) error
	HasClaimedQuest(ctx context.Context, playerID int, questID string) (bool, error)
	GetCompletedQuestIDs(ctx context.Context, playerID int) ([]string, error)
	GetPlayerPlanetID(ctx context.Context, playerID int) (int, error)
	GetBuildingLevel(ctx context.Context, playerID int, buildingType string) (int, error)
	GetTechLevel(ctx context.Context, playerID int, techType string) (int, error)
	GetPlayerShipCount(ctx context.Context, playerID int, shipType string) (int, error)
	GetPlayerDefenseCount(ctx context.Context, playerID int, defenseType string) (int, error)
	GetTotalPlayerResources(ctx context.Context, playerID int) (int, error)
	GetExpeditionCount(ctx context.Context, playerID int) (int, error)
	// GetPlayerAllianceMembership returns 1 if the player has an active (non-pending)
	// alliance membership, 0 otherwise.
	GetPlayerAllianceMembership(ctx context.Context, playerID int) (int, error)
	AddPlayerResources(ctx context.Context, playerID int, metal, crystal, gas int) error
	AddDarkMatter(ctx context.Context, playerID int, amount int) error
}

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) GetQuestDefinitions(ctx context.Context) ([]QuestDefinition, error) {
	return QuestDefinitions, nil
}

func (r *PostgresRepository) GetPlayerQuest(ctx context.Context, playerID int, questID string) (*PlayerQuest, error) {
	var pq PlayerQuest
	err := r.pool.QueryRow(ctx,
		`SELECT id, player_id, quest_id, status, progress_current, progress_target, started_at, completed_at, claimed_at
		 FROM quest.player_quests WHERE player_id = $1 AND quest_id = $2`,
		playerID, questID,
	).Scan(&pq.ID, &pq.PlayerID, &pq.QuestID, &pq.Status, &pq.ProgressCurrent, &pq.ProgressTarget, &pq.StartedAt, &pq.CompletedAt, &pq.ClaimedAt)
	if err != nil {
		return nil, fmt.Errorf("get player quest: %w", err)
	}
	return &pq, nil
}

func (r *PostgresRepository) GetPlayerQuests(ctx context.Context, playerID int) ([]PlayerQuest, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, player_id, quest_id, status, progress_current, progress_target, started_at, completed_at, claimed_at
		 FROM quest.player_quests WHERE player_id = $1 ORDER BY quest_id`,
		playerID,
	)
	if err != nil {
		return nil, fmt.Errorf("get player quests: %w", err)
	}
	defer rows.Close()

	var quests []PlayerQuest
	for rows.Next() {
		var pq PlayerQuest
		if err := rows.Scan(&pq.ID, &pq.PlayerID, &pq.QuestID, &pq.Status, &pq.ProgressCurrent, &pq.ProgressTarget, &pq.StartedAt, &pq.CompletedAt, &pq.ClaimedAt); err != nil {
			return nil, fmt.Errorf("scan player quest: %w", err)
		}
		quests = append(quests, pq)
	}
	return quests, rows.Err()
}

func (r *PostgresRepository) UpsertPlayerQuest(ctx context.Context, pq PlayerQuest) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO quest.player_quests (player_id, quest_id, status, progress_current, progress_target, started_at, completed_at, claimed_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT (player_id, quest_id) DO UPDATE SET
		   status = EXCLUDED.status,
		   progress_current = EXCLUDED.progress_current,
		   progress_target = EXCLUDED.progress_target,
		   started_at = EXCLUDED.started_at,
		   completed_at = EXCLUDED.completed_at,
		   claimed_at = EXCLUDED.claimed_at`,
		pq.PlayerID, pq.QuestID, pq.Status, pq.ProgressCurrent, pq.ProgressTarget, pq.StartedAt, pq.CompletedAt, pq.ClaimedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert player quest: %w", err)
	}
	return nil
}

func (r *PostgresRepository) ClaimPlayerQuest(ctx context.Context, playerID int, questID string, claimedAt time.Time) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE quest.player_quests
		 SET status = 'claimed', claimed_at = $3
		 WHERE player_id = $1 AND quest_id = $2 AND status = 'completed'`,
		playerID, questID, claimedAt,
	)
	if err != nil {
		return fmt.Errorf("claim player quest: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("quest not available for claiming")
	}
	return nil
}

func (r *PostgresRepository) HasClaimedQuest(ctx context.Context, playerID int, questID string) (bool, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM quest.player_quests WHERE player_id = $1 AND quest_id = $2 AND status = 'claimed'`,
		playerID, questID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("has claimed quest: %w", err)
	}
	return count > 0, nil
}

func (r *PostgresRepository) GetCompletedQuestIDs(ctx context.Context, playerID int) ([]string, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT quest_id FROM quest.player_quests
		 WHERE player_id = $1 AND (status = 'completed' OR status = 'claimed')
		 ORDER BY quest_id`,
		playerID,
	)
	if err != nil {
		return nil, fmt.Errorf("get completed quest ids: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan quest id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *PostgresRepository) GetPlayerPlanetID(ctx context.Context, playerID int) (int, error) {
	var planetID int
	err := r.pool.QueryRow(ctx,
		`SELECT id FROM planet.planets WHERE user_id = $1 LIMIT 1`,
		playerID,
	).Scan(&planetID)
	if err != nil {
		return 0, fmt.Errorf("get player planet id: %w", err)
	}
	return planetID, nil
}

func (r *PostgresRepository) GetBuildingLevel(ctx context.Context, playerID int, buildingType string) (int, error) {
	var level int
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(MAX(b.level), 0)
		 FROM planet.buildings b
		 JOIN planet.planets p ON p.id = b.planet_id
		 WHERE p.user_id = $1 AND b.type = $2`,
		playerID, buildingType,
	).Scan(&level)
	if err != nil {
		return 0, fmt.Errorf("get building level: %w", err)
	}
	return level, nil
}

func (r *PostgresRepository) GetTechLevel(ctx context.Context, playerID int, techType string) (int, error) {
	var level int
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(level, 0) FROM planet.player_technologies WHERE user_id = $1 AND type = $2`,
		playerID, techType,
	).Scan(&level)
	if err != nil {
		return 0, nil
	}
	return level, nil
}

func (r *PostgresRepository) GetPlayerShipCount(ctx context.Context, playerID int, shipType string) (int, error) {
	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(ps.quantity), 0)
		 FROM planet.player_ships ps
		 JOIN planet.planets p ON p.id = ps.planet_id
		 WHERE p.user_id = $1 AND ps.ship_type = $2`,
		playerID, shipType,
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("get player ship count: %w", err)
	}
	return total, nil
}

func (r *PostgresRepository) GetPlayerDefenseCount(ctx context.Context, playerID int, defenseType string) (int, error) {
	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(pd.quantity), 0)
		 FROM planet.player_defenses pd
		 JOIN planet.planets p ON p.id = pd.planet_id
		 WHERE p.user_id = $1 AND pd.defense_type = $2`,
		playerID, defenseType,
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("get player defense count: %w", err)
	}
	return total, nil
}

func (r *PostgresRepository) GetTotalPlayerResources(ctx context.Context, playerID int) (int, error) {
	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(metal, 0) + COALESCE(crystal, 0) + COALESCE(gas, 0)
		 FROM planet.planets WHERE user_id = $1 LIMIT 1`,
		playerID,
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("get total player resources: %w", err)
	}
	return total, nil
}

func (r *PostgresRepository) GetExpeditionCount(ctx context.Context, playerID int) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM nebula.expeditions WHERE player_id = $1`,
		playerID,
	).Scan(&count)
	if err != nil {
		return 0, nil
	}
	return count, nil
}

func (r *PostgresRepository) GetPlayerAllianceMembership(ctx context.Context, playerID int) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM alliance.members
		 WHERE player_id = $1 AND role NOT IN ('pending', 'applicant')`,
		playerID,
	).Scan(&count)
	if err != nil {
		return 0, nil
	}
	if count > 0 {
		return 1, nil
	}
	return 0, nil
}

func (r *PostgresRepository) AddPlayerResources(ctx context.Context, playerID int, metal, crystal, gas int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE planet.planets
		 SET metal = metal + $1, crystal = crystal + $2, gas = gas + $3, resources_updated_at = NOW()
		 WHERE user_id = $4`,
		metal, crystal, gas, playerID,
	)
	if err != nil {
		return fmt.Errorf("add player resources: %w", err)
	}
	return nil
}

func (r *PostgresRepository) AddDarkMatter(ctx context.Context, playerID int, amount int) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO nebula.player_dark_matter (player_id, balance, total_earned)
		 VALUES ($1, $2, $2)
		 ON CONFLICT (player_id) DO UPDATE SET
		   balance = nebula.player_dark_matter.balance + $2,
		   total_earned = nebula.player_dark_matter.total_earned + $2`,
		playerID, amount,
	)
	if err != nil {
		return fmt.Errorf("add dark matter: %w", err)
	}

	_, err = r.pool.Exec(ctx,
		`INSERT INTO nebula.dm_transactions (player_id, amount, balance_after, reason)
		 VALUES ($1, $2, (SELECT COALESCE(balance, 0) FROM nebula.player_dark_matter WHERE player_id = $1), 'quest_reward')`,
		playerID, amount,
	)
	return err
}
