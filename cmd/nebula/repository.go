package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	CreateExpedition(ctx context.Context, e Expedition) (Expedition, error)
	GetExpedition(ctx context.Context, expeditionID, playerID int) (Expedition, error)
	ListPlayerExpeditions(ctx context.Context, playerID int) ([]Expedition, error)
	UpdateExpeditionOutcome(ctx context.Context, expeditionID int, outcome string, resourcesFound, shipsFound, shipsLost map[string]int, darkMatter int) error
	GetDarkMatterBalance(ctx context.Context, playerID int) (balance, totalEarned int, err error)
	AddDarkMatter(ctx context.Context, playerID, amount int) error
	SpendDarkMatter(ctx context.Context, playerID, amount int) error
	AddDMTransaction(ctx context.Context, playerID, amount, balanceAfter int, reason string) error
	ListDMTransactions(ctx context.Context, playerID, limit int) ([]DMTransaction, error)
	HireCommander(ctx context.Context, playerID int, commanderType string, level int, expiresAt time.Time) (CommanderEntry, error)
	GetActiveCommanders(ctx context.Context, playerID int) ([]CommanderEntry, error)
	GetCommander(ctx context.Context, playerID int, commanderType string) (*CommanderEntry, error)
	ListAllCommanders(ctx context.Context, playerID int) ([]CommanderEntry, error)
	GetCreditsBalance(ctx context.Context, playerID int) (balance, totalEarned int, err error)
	AddCredits(ctx context.Context, playerID, amount int) error
	SpendCredits(ctx context.Context, playerID, amount int) error
	AddCreditsTransaction(ctx context.Context, playerID, amount, balanceAfter int, reason string) error
	ListCreditsTransactions(ctx context.Context, playerID, limit int) ([]CreditsTransaction, error)
	GetDailyGiftStatus(ctx context.Context, playerID int) (streakDay, consecutiveDays int, lastClaimDate string, err error)
	ClaimDailyGift(ctx context.Context, playerID int) (newStreakDay, consecutiveDays int, err error)
	ResetDailyGiftStreak(ctx context.Context, playerID int) error
	GetDailyTasks(ctx context.Context, playerID int) ([]dailyTaskRow, error)
	AssignDailyTasks(ctx context.Context, playerID int, tasks []dailyTaskRow) error
	UpdateTaskProgress(ctx context.Context, taskID, playerID, progress int) error
	MarkTaskCompleted(ctx context.Context, taskID, playerID int) error
	ClaimTaskReward(ctx context.Context, taskID, playerID int) (dailyTaskRow, error)
	ClaimAllTasksReward(ctx context.Context, playerID int) ([]dailyTaskRow, error)
	IncrementTaskRerolls(ctx context.Context, playerID int) error
	RerollTask(ctx context.Context, taskID, playerID int) (dailyTaskRow, error)
	CreateStorePurchase(ctx context.Context, playerID int, itemID string, cost int, currency string) error
	GetPlayerShifterUses(ctx context.Context, playerID int) (int, error)
	SetPlayerShifterUses(ctx context.Context, playerID int, uses int) error
	ExtendCommanderDuration(ctx context.Context, playerID int, commanderType string, days int) error
	GetPlayerPlanetID(ctx context.Context, playerID int) (int, error)
	AddGalactoniteShards(ctx context.Context, playerID int, shardType string, count int) error
	GetGalactoniteDiscovererLevel(ctx context.Context, playerID int) (int, error)
	UpgradeGalactoniteDiscoverer(ctx context.Context, playerID int) (int, error)
}

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) CreateExpedition(ctx context.Context, e Expedition) (Expedition, error) {
	shipsSentJSON, _ := json.Marshal(e.ShipsSent)
	shipsLostJSON, _ := json.Marshal(e.ShipsLost)
	shipsFoundJSON, _ := json.Marshal(e.ShipsFound)
	resourcesFoundJSON, _ := json.Marshal(e.ResourcesFound)

	var id int
	var startedAt time.Time
	err := r.pool.QueryRow(ctx, `
		INSERT INTO nebula.expeditions (player_id, fleet_id, galaxy, system, position, status, ships_sent, ships_lost, ships_found, resources_found, dark_matter_found, outcome, travel_duration, explore_duration, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id, started_at
	`, e.PlayerID, e.FleetID, e.Galaxy, e.System, e.Position, e.Status,
		shipsSentJSON, shipsLostJSON, shipsFoundJSON, resourcesFoundJSON,
		e.DarkMatterFound, e.Outcome, e.TravelDuration, e.ExploreDuration, e.CompletedAt,
	).Scan(&id, &startedAt)
	if err != nil {
		return Expedition{}, fmt.Errorf("create expedition: %w", err)
	}
	e.ID = id
	e.StartedAt = startedAt
	return e, nil
}

func (r *PostgresRepository) GetExpedition(ctx context.Context, expeditionID, playerID int) (Expedition, error) {
	var e Expedition
	var shipsSentJSON, shipsLostJSON, shipsFoundJSON, resourcesFoundJSON []byte
	var completedAt *time.Time

	err := r.pool.QueryRow(ctx, `
		SELECT id, player_id, fleet_id, galaxy, system, position, status, ships_sent, ships_lost, ships_found, resources_found, dark_matter_found, outcome, started_at, travel_duration, explore_duration, completed_at
		FROM nebula.expeditions
		WHERE id = $1 AND player_id = $2
	`, expeditionID, playerID).Scan(
		&e.ID, &e.PlayerID, &e.FleetID, &e.Galaxy, &e.System, &e.Position,
		&e.Status, &shipsSentJSON, &shipsLostJSON, &shipsFoundJSON, &resourcesFoundJSON,
		&e.DarkMatterFound, &e.Outcome, &e.StartedAt, &e.TravelDuration, &e.ExploreDuration,
		&completedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Expedition{}, fmt.Errorf("expedition not found")
		}
		return Expedition{}, fmt.Errorf("get expedition: %w", err)
	}

	json.Unmarshal(shipsSentJSON, &e.ShipsSent)
	json.Unmarshal(shipsLostJSON, &e.ShipsLost)
	json.Unmarshal(shipsFoundJSON, &e.ShipsFound)
	json.Unmarshal(resourcesFoundJSON, &e.ResourcesFound)
	if completedAt != nil {
		e.CompletedAt = completedAt
	}
	return e, nil
}

func (r *PostgresRepository) ListPlayerExpeditions(ctx context.Context, playerID int) ([]Expedition, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, player_id, fleet_id, galaxy, system, position, status, ships_sent, ships_lost, ships_found, resources_found, dark_matter_found, outcome, started_at, travel_duration, explore_duration, completed_at
		FROM nebula.expeditions
		WHERE player_id = $1
		ORDER BY started_at DESC
	`, playerID)
	if err != nil {
		return nil, fmt.Errorf("list expeditions: %w", err)
	}
	defer rows.Close()

	var expeditions []Expedition
	for rows.Next() {
		var e Expedition
		var shipsSentJSON, shipsLostJSON, shipsFoundJSON, resourcesFoundJSON []byte
		var completedAt *time.Time

		if err := rows.Scan(
			&e.ID, &e.PlayerID, &e.FleetID, &e.Galaxy, &e.System, &e.Position,
			&e.Status, &shipsSentJSON, &shipsLostJSON, &shipsFoundJSON, &resourcesFoundJSON,
			&e.DarkMatterFound, &e.Outcome, &e.StartedAt, &e.TravelDuration, &e.ExploreDuration,
			&completedAt,
		); err != nil {
			return nil, fmt.Errorf("scan expedition: %w", err)
		}

		json.Unmarshal(shipsSentJSON, &e.ShipsSent)
		json.Unmarshal(shipsLostJSON, &e.ShipsLost)
		json.Unmarshal(shipsFoundJSON, &e.ShipsFound)
		json.Unmarshal(resourcesFoundJSON, &e.ResourcesFound)
		if completedAt != nil {
			e.CompletedAt = completedAt
		}
		expeditions = append(expeditions, e)
	}
	return expeditions, rows.Err()
}

func (r *PostgresRepository) UpdateExpeditionOutcome(ctx context.Context, expeditionID int, outcome string, resourcesFound, shipsFound, shipsLost map[string]int, darkMatter int) error {
	resourcesFoundJSON, _ := json.Marshal(resourcesFound)
	shipsFoundJSON, _ := json.Marshal(shipsFound)
	shipsLostJSON, _ := json.Marshal(shipsLost)

	now := time.Now()
	_, err := r.pool.Exec(ctx, `
		UPDATE nebula.expeditions
		SET status = 'completed', outcome = $1, resources_found = $2, ships_found = $3, ships_lost = $4, dark_matter_found = $5, completed_at = $6
		WHERE id = $7
	`, outcome, resourcesFoundJSON, shipsFoundJSON, shipsLostJSON, darkMatter, now, expeditionID)
	return err
}

func (r *PostgresRepository) GetDarkMatterBalance(ctx context.Context, playerID int) (int, int, error) {
	var balance, totalEarned int
	err := r.pool.QueryRow(ctx, `
		SELECT balance, total_earned FROM nebula.player_dark_matter WHERE player_id = $1
	`, playerID).Scan(&balance, &totalEarned)
	if err == pgx.ErrNoRows {
		return 0, 0, nil
	}
	if err != nil {
		return 0, 0, fmt.Errorf("get dm balance: %w", err)
	}
	return balance, totalEarned, nil
}

func (r *PostgresRepository) AddDarkMatter(ctx context.Context, playerID, amount int) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO nebula.player_dark_matter (player_id, balance, total_earned)
		VALUES ($1, $2, $2)
		ON CONFLICT (player_id) DO UPDATE
		SET balance = nebula.player_dark_matter.balance + $2,
		    total_earned = nebula.player_dark_matter.total_earned + $2
	`, playerID, amount)
	if err != nil {
		return fmt.Errorf("add dm: %w", err)
	}
	return nil
}

func (r *PostgresRepository) SpendDarkMatter(ctx context.Context, playerID, amount int) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE nebula.player_dark_matter
		SET balance = balance - $2
		WHERE player_id = $1 AND balance >= $2
	`, playerID, amount)
	if err != nil {
		return fmt.Errorf("spend dm: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("insufficient dark matter")
	}
	return nil
}

func (r *PostgresRepository) AddDMTransaction(ctx context.Context, playerID, amount, balanceAfter int, reason string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO nebula.dm_transactions (player_id, amount, balance_after, reason)
		VALUES ($1, $2, $3, $4)
	`, playerID, amount, balanceAfter, reason)
	return err
}

func (r *PostgresRepository) ListDMTransactions(ctx context.Context, playerID, limit int) ([]DMTransaction, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, player_id, amount, balance_after, reason, created_at
		FROM nebula.dm_transactions
		WHERE player_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, playerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var txs []DMTransaction
	for rows.Next() {
		var tx DMTransaction
		if err := rows.Scan(&tx.ID, &tx.PlayerID, &tx.Amount, &tx.BalanceAfter, &tx.Reason, &tx.CreatedAt); err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}
	return txs, rows.Err()
}

func (r *PostgresRepository) HireCommander(ctx context.Context, playerID int, commanderType string, level int, expiresAt time.Time) (CommanderEntry, error) {
	var entry CommanderEntry
	err := r.pool.QueryRow(ctx, `
		INSERT INTO nebula.player_commanders (player_id, commander_type, level, expires_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (player_id, commander_type) DO UPDATE
		SET level = $3, hired_at = NOW(), expires_at = $4
		RETURNING id, hired_at
	`, playerID, commanderType, level, expiresAt).Scan(&entry.ID, &entry.HiredAt)
	if err != nil {
		return CommanderEntry{}, err
	}
	entry.PlayerID = playerID
	entry.CommanderType = commanderType
	entry.Level = level
	entry.ExpiresAt = expiresAt
	return entry, nil
}

func (r *PostgresRepository) GetActiveCommanders(ctx context.Context, playerID int) ([]CommanderEntry, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, player_id, commander_type, level, hired_at, expires_at
		FROM nebula.player_commanders
		WHERE player_id = $1 AND expires_at > NOW()
	`, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []CommanderEntry
	for rows.Next() {
		var e CommanderEntry
		if err := rows.Scan(&e.ID, &e.PlayerID, &e.CommanderType, &e.Level, &e.HiredAt, &e.ExpiresAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (r *PostgresRepository) GetCommander(ctx context.Context, playerID int, commanderType string) (*CommanderEntry, error) {
	var e CommanderEntry
	err := r.pool.QueryRow(ctx, `
		SELECT id, player_id, commander_type, level, hired_at, expires_at
		FROM nebula.player_commanders
		WHERE player_id = $1 AND commander_type = $2
	`, playerID, commanderType).Scan(&e.ID, &e.PlayerID, &e.CommanderType, &e.Level, &e.HiredAt, &e.ExpiresAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *PostgresRepository) ListAllCommanders(ctx context.Context, playerID int) ([]CommanderEntry, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, player_id, commander_type, level, hired_at, expires_at
		FROM nebula.player_commanders
		WHERE player_id = $1
		ORDER BY commander_type
	`, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []CommanderEntry
	for rows.Next() {
		var e CommanderEntry
		if err := rows.Scan(&e.ID, &e.PlayerID, &e.CommanderType, &e.Level, &e.HiredAt, &e.ExpiresAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (r *PostgresRepository) GetCreditsBalance(ctx context.Context, playerID int) (int, int, error) {
	var balance, totalEarned int
	err := r.pool.QueryRow(ctx, `
		SELECT balance, total_earned FROM nebula.player_credits WHERE player_id = $1
	`, playerID).Scan(&balance, &totalEarned)
	if err == pgx.ErrNoRows {
		return 0, 0, nil
	}
	if err != nil {
		return 0, 0, fmt.Errorf("get credits balance: %w", err)
	}
	return balance, totalEarned, nil
}

func (r *PostgresRepository) AddCredits(ctx context.Context, playerID, amount int) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO nebula.player_credits (player_id, balance, total_earned)
		VALUES ($1, $2, $2)
		ON CONFLICT (player_id) DO UPDATE
		SET balance = nebula.player_credits.balance + $2,
		    total_earned = nebula.player_credits.total_earned + $2
	`, playerID, amount)
	if err != nil {
		return fmt.Errorf("add credits: %w", err)
	}
	return nil
}

func (r *PostgresRepository) SpendCredits(ctx context.Context, playerID, amount int) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE nebula.player_credits
		SET balance = balance - $2
		WHERE player_id = $1 AND balance >= $2
	`, playerID, amount)
	if err != nil {
		return fmt.Errorf("spend credits: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("insufficient credits")
	}
	return nil
}

func (r *PostgresRepository) AddCreditsTransaction(ctx context.Context, playerID, amount, balanceAfter int, reason string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO nebula.credits_transactions (player_id, amount, balance_after, reason)
		VALUES ($1, $2, $3, $4)
	`, playerID, amount, balanceAfter, reason)
	return err
}

func (r *PostgresRepository) ListCreditsTransactions(ctx context.Context, playerID, limit int) ([]CreditsTransaction, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, player_id, amount, balance_after, reason, created_at
		FROM nebula.credits_transactions
		WHERE player_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, playerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var txs []CreditsTransaction
	for rows.Next() {
		var tx CreditsTransaction
		if err := rows.Scan(&tx.ID, &tx.PlayerID, &tx.Amount, &tx.BalanceAfter, &tx.Reason, &tx.CreatedAt); err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}
	return txs, rows.Err()
}

func (r *PostgresRepository) GetDailyGiftStatus(ctx context.Context, playerID int) (int, int, string, error) {
	var streakDay, consecutiveDays int
	var lastClaimDate *time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT streak_day, consecutive_days, last_claim_date
		FROM nebula.daily_gift_streak
		WHERE player_id = $1
	`, playerID).Scan(&streakDay, &consecutiveDays, &lastClaimDate)
	if err == pgx.ErrNoRows {
		return 0, 0, "", nil
	}
	if err != nil {
		return 0, 0, "", fmt.Errorf("get daily gift status: %w", err)
	}
	dateStr := ""
	if lastClaimDate != nil {
		dateStr = lastClaimDate.Format("2006-01-02")
	}
	return streakDay, consecutiveDays, dateStr, nil
}

func (r *PostgresRepository) ClaimDailyGift(ctx context.Context, playerID int) (int, int, error) {
	var newStreakDay, consecutiveDays int
	err := r.pool.QueryRow(ctx, `
		INSERT INTO nebula.daily_gift_streak (player_id, streak_day, consecutive_days, last_claim_date)
		VALUES ($1, 1, 1, CURRENT_DATE)
		ON CONFLICT (player_id) DO UPDATE
		SET streak_day = (CASE
			WHEN nebula.daily_gift_streak.last_claim_date = CURRENT_DATE THEN nebula.daily_gift_streak.streak_day
			WHEN nebula.daily_gift_streak.last_claim_date = CURRENT_DATE - 1 THEN nebula.daily_gift_streak.streak_day + 1
			ELSE 1
		END),
		consecutive_days = (CASE
			WHEN nebula.daily_gift_streak.last_claim_date = CURRENT_DATE THEN nebula.daily_gift_streak.consecutive_days
			WHEN nebula.daily_gift_streak.last_claim_date = CURRENT_DATE - 1 THEN nebula.daily_gift_streak.consecutive_days + 1
			ELSE 1
		END),
		last_claim_date = CURRENT_DATE
		RETURNING streak_day, consecutive_days
	`, playerID).Scan(&newStreakDay, &consecutiveDays)
	if err != nil {
		return 0, 0, fmt.Errorf("claim daily gift: %w", err)
	}
	if newStreakDay > 7 {
		newStreakDay = 1
	}
	return newStreakDay, consecutiveDays, nil
}

func (r *PostgresRepository) ResetDailyGiftStreak(ctx context.Context, playerID int) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO nebula.daily_gift_streak (player_id, streak_day, consecutive_days, last_claim_date)
		VALUES ($1, 0, 0, CURRENT_DATE)
		ON CONFLICT (player_id) DO UPDATE
		SET streak_day = 0, consecutive_days = 0, last_claim_date = CURRENT_DATE
	`, playerID)
	return err
}

func (r *PostgresRepository) GetDailyTasks(ctx context.Context, playerID int) ([]dailyTaskRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, player_id, task_type, description, target_amount, progress, reward_dm, reward_resources, completed, claimed, assigned_date, rerolls_used
		FROM nebula.daily_tasks
		WHERE player_id = $1 AND assigned_date = CURRENT_DATE
		ORDER BY id
	`, playerID)
	if err != nil {
		return nil, fmt.Errorf("get daily tasks: %w", err)
	}
	defer rows.Close()
	var tasks []dailyTaskRow
	for rows.Next() {
		var t dailyTaskRow
		var rewardResources []byte
		var assignedDate time.Time
		if err := rows.Scan(&t.ID, &t.PlayerID, &t.TaskType, &t.Description, &t.TargetAmount, &t.Progress, &t.RewardDM, &rewardResources, &t.Completed, &t.Claimed, &assignedDate, &t.RerollsUsed); err != nil {
			return nil, fmt.Errorf("scan daily task: %w", err)
		}
		t.RewardResources = string(rewardResources)
		t.AssignedDate = assignedDate.Format("2006-01-02")
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func (r *PostgresRepository) AssignDailyTasks(ctx context.Context, playerID int, tasks []dailyTaskRow) error {
	for _, t := range tasks {
		rewardResJSON, _ := json.Marshal(t.RewardResources)
		_, err := r.pool.Exec(ctx, `
			INSERT INTO nebula.daily_tasks (player_id, task_type, description, target_amount, progress, reward_dm, reward_resources, completed, claimed, assigned_date, rerolls_used)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, CURRENT_DATE, 0)
			ON CONFLICT (player_id, task_type, assigned_date) DO NOTHING
		`, t.PlayerID, t.TaskType, t.Description, t.TargetAmount, t.Progress, t.RewardDM, rewardResJSON, t.Completed, t.Claimed)
		if err != nil {
			return fmt.Errorf("assign daily task: %w", err)
		}
	}
	return nil
}

func (r *PostgresRepository) UpdateTaskProgress(ctx context.Context, taskID, playerID, progress int) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE nebula.daily_tasks
		SET progress = LEAST(progress + $3, target_amount)
		WHERE id = $1 AND player_id = $2 AND assigned_date = CURRENT_DATE
	`, taskID, playerID, progress)
	return err
}

func (r *PostgresRepository) MarkTaskCompleted(ctx context.Context, taskID, playerID int) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE nebula.daily_tasks
		SET completed = TRUE
		WHERE id = $1 AND player_id = $2 AND progress >= target_amount AND assigned_date = CURRENT_DATE
	`, taskID, playerID)
	return err
}

func (r *PostgresRepository) ClaimTaskReward(ctx context.Context, taskID, playerID int) (dailyTaskRow, error) {
	var t dailyTaskRow
	var rewardResources []byte
	var assignedDate time.Time
	err := r.pool.QueryRow(ctx, `
		UPDATE nebula.daily_tasks
		SET claimed = TRUE
		WHERE id = $1 AND player_id = $2 AND completed = TRUE AND claimed = FALSE AND assigned_date = CURRENT_DATE
		RETURNING id, player_id, task_type, description, target_amount, progress, reward_dm, reward_resources, completed, claimed, assigned_date
	`, taskID, playerID).Scan(&t.ID, &t.PlayerID, &t.TaskType, &t.Description, &t.TargetAmount, &t.Progress, &t.RewardDM, &rewardResources, &t.Completed, &t.Claimed, &assignedDate)
	if err != nil {
		if err == pgx.ErrNoRows {
			return dailyTaskRow{}, fmt.Errorf("task not found or already claimed")
		}
		return dailyTaskRow{}, fmt.Errorf("claim task reward: %w", err)
	}
	t.RewardResources = string(rewardResources)
	t.AssignedDate = assignedDate.Format("2006-01-02")
	return t, nil
}

func (r *PostgresRepository) ClaimAllTasksReward(ctx context.Context, playerID int) ([]dailyTaskRow, error) {
	rows, err := r.pool.Query(ctx, `
		UPDATE nebula.daily_tasks
		SET claimed = TRUE
		WHERE player_id = $1 AND completed = TRUE AND claimed = FALSE AND assigned_date = CURRENT_DATE
		RETURNING id, player_id, task_type, description, target_amount, progress, reward_dm, reward_resources, completed, claimed, assigned_date
	`, playerID)
	if err != nil {
		return nil, fmt.Errorf("claim all tasks reward: %w", err)
	}
	defer rows.Close()
	var tasks []dailyTaskRow
	for rows.Next() {
		var t dailyTaskRow
		var rewardResources []byte
		var assignedDate time.Time
		if err := rows.Scan(&t.ID, &t.PlayerID, &t.TaskType, &t.Description, &t.TargetAmount, &t.Progress, &t.RewardDM, &rewardResources, &t.Completed, &t.Claimed, &assignedDate); err != nil {
			return nil, fmt.Errorf("scan claimed task: %w", err)
		}
		t.RewardResources = string(rewardResources)
		t.AssignedDate = assignedDate.Format("2006-01-02")
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func (r *PostgresRepository) IncrementTaskRerolls(ctx context.Context, playerID int) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE nebula.daily_tasks
		SET rerolls_used = rerolls_used + 1
		WHERE player_id = $1 AND assigned_date = CURRENT_DATE
	`, playerID)
	return err
}

func (r *PostgresRepository) RerollTask(ctx context.Context, taskID, playerID int) (dailyTaskRow, error) {
	// Delete the old task
	_, err := r.pool.Exec(ctx, `
		DELETE FROM nebula.daily_tasks
		WHERE id = $1 AND player_id = $2 AND assigned_date = CURRENT_DATE AND claimed = FALSE
	`, taskID, playerID)
	if err != nil {
		return dailyTaskRow{}, fmt.Errorf("delete old task for reroll: %w", err)
	}
	return dailyTaskRow{}, nil
}

func (r *PostgresRepository) CreateStorePurchase(ctx context.Context, playerID int, itemID string, cost int, currency string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO nebula.store_purchases (player_id, item_id, cost, currency)
		VALUES ($1, $2, $3, $4)
	`, playerID, itemID, cost, currency)
	return err
}

func (r *PostgresRepository) GetPlayerShifterUses(ctx context.Context, playerID int) (int, error) {
	var uses int
	err := r.pool.QueryRow(ctx, `
		SELECT uses FROM nebula.player_shifter_uses WHERE player_id = $1
	`, playerID).Scan(&uses)
	if err == pgx.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("get shifter uses: %w", err)
	}
	return uses, nil
}

func (r *PostgresRepository) SetPlayerShifterUses(ctx context.Context, playerID int, uses int) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO nebula.player_shifter_uses (player_id, uses)
		VALUES ($1, $2)
		ON CONFLICT (player_id) DO UPDATE SET uses = $2
	`, playerID, uses)
	return err
}

func (r *PostgresRepository) ExtendCommanderDuration(ctx context.Context, playerID int, commanderType string, days int) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE nebula.player_commanders
		SET expires_at = GREATEST(expires_at, NOW()) + make_interval(days => $3)
		WHERE player_id = $1 AND commander_type = $2
	`, playerID, commanderType, days)
	return err
}

func (r *PostgresRepository) GetPlayerPlanetID(ctx context.Context, playerID int) (int, error) {
	var planetID int
	err := r.pool.QueryRow(ctx, `
		SELECT id FROM planet.planets WHERE user_id = $1 LIMIT 1
	`, playerID).Scan(&planetID)
	if err != nil {
		return 0, fmt.Errorf("get player planet id: %w", err)
	}
	return planetID, nil
}

func (r *PostgresRepository) AddGalactoniteShards(ctx context.Context, playerID int, shardType string, count int) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO planet.galactonite_shards (player_id, gem_type, count)
		VALUES ($1, $2, $3)
		ON CONFLICT (player_id, gem_type)
		DO UPDATE SET count = planet.galactonite_shards.count + $3
	`, playerID, shardType, count)
	return err
}

func (r *PostgresRepository) GetGalactoniteDiscovererLevel(ctx context.Context, playerID int) (int, error) {
	var level int
	err := r.pool.QueryRow(ctx, `
		SELECT level FROM nebula.galactonite_discoverer WHERE player_id = $1
	`, playerID).Scan(&level)
	if err == pgx.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("get discoverer level: %w", err)
	}
	return level, nil
}

func (r *PostgresRepository) UpgradeGalactoniteDiscoverer(ctx context.Context, playerID int) (int, error) {
	var newLevel int
	err := r.pool.QueryRow(ctx, `
		INSERT INTO nebula.galactonite_discoverer (player_id, level)
		VALUES ($1, 1)
		ON CONFLICT (player_id)
		DO UPDATE SET level = nebula.galactonite_discoverer.level + 1
		RETURNING level
	`, playerID).Scan(&newLevel)
	if err != nil {
		return 0, fmt.Errorf("upgrade discoverer: %w", err)
	}
	return newLevel, nil
}

type dailyGiftStreakEntry struct {
	StreakDay       int
	ConsecutiveDays int
	LastClaimDate   string
}

type mockRepo struct {
	expeditions   []Expedition
	nextID        int
	dmBalances    map[int]int
	dmTotalEarned map[int]int
	dmTransactions []DMTransaction
	commanders         []CommanderEntry
	creditBalances     map[int]int
	creditTotalEarned  map[int]int
	creditsTransactions []CreditsTransaction
	dailyGiftStreak     map[int]dailyGiftStreakEntry
	dailyTasks          []dailyTaskRow
	taskNextID          int
	discovererLevels   map[int]int
}

func newMockRepo() *mockRepo {
	return &mockRepo{nextID: 1}
}

func (m *mockRepo) CreateExpedition(ctx context.Context, e Expedition) (Expedition, error) {
	e.ID = m.nextID
	m.nextID++
	e.StartedAt = time.Now()
	m.expeditions = append(m.expeditions, e)
	return e, nil
}

func (m *mockRepo) GetExpedition(ctx context.Context, expeditionID, playerID int) (Expedition, error) {
	for _, e := range m.expeditions {
		if e.ID == expeditionID && e.PlayerID == playerID {
			return e, nil
		}
	}
	return Expedition{}, fmt.Errorf("expedition not found")
}

func (m *mockRepo) ListPlayerExpeditions(ctx context.Context, playerID int) ([]Expedition, error) {
	var result []Expedition
	for _, e := range m.expeditions {
		if e.PlayerID == playerID {
			result = append(result, e)
		}
	}
	return result, nil
}

func (m *mockRepo) UpdateExpeditionOutcome(ctx context.Context, expeditionID int, outcome string, resourcesFound, shipsFound, shipsLost map[string]int, darkMatter int) error {
	for i, e := range m.expeditions {
		if e.ID == expeditionID {
			m.expeditions[i].Status = "completed"
			m.expeditions[i].Outcome = outcome
			m.expeditions[i].ResourcesFound = resourcesFound
			m.expeditions[i].ShipsFound = shipsFound
			m.expeditions[i].ShipsLost = shipsLost
			m.expeditions[i].DarkMatterFound = darkMatter
			now := time.Now()
			m.expeditions[i].CompletedAt = &now
			return nil
		}
	}
	return fmt.Errorf("expedition not found")
}

func (m *mockRepo) GetDarkMatterBalance(ctx context.Context, playerID int) (int, int, error) {
	if m.dmBalances == nil {
		return 0, 0, nil
	}
	return m.dmBalances[playerID], m.dmTotalEarned[playerID], nil
}

func (m *mockRepo) AddDarkMatter(ctx context.Context, playerID, amount int) error {
	if m.dmBalances == nil {
		m.dmBalances = make(map[int]int)
		m.dmTotalEarned = make(map[int]int)
	}
	m.dmBalances[playerID] += amount
	m.dmTotalEarned[playerID] += amount
	return nil
}

func (m *mockRepo) SpendDarkMatter(ctx context.Context, playerID, amount int) error {
	if m.dmBalances == nil {
		return fmt.Errorf("insufficient dark matter")
	}
	if m.dmBalances[playerID] < amount {
		return fmt.Errorf("insufficient dark matter")
	}
	m.dmBalances[playerID] -= amount
	return nil
}

func (m *mockRepo) AddDMTransaction(ctx context.Context, playerID, amount, balanceAfter int, reason string) error {
	m.dmTransactions = append(m.dmTransactions, DMTransaction{
		ID:           len(m.dmTransactions) + 1,
		PlayerID:     playerID,
		Amount:       amount,
		BalanceAfter: balanceAfter,
		Reason:       reason,
		CreatedAt:    time.Now(),
	})
	return nil
}

func (m *mockRepo) ListDMTransactions(ctx context.Context, playerID, limit int) ([]DMTransaction, error) {
	var result []DMTransaction
	for _, tx := range m.dmTransactions {
		if tx.PlayerID == playerID {
			result = append(result, tx)
		}
	}
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (m *mockRepo) GetCreditsBalance(ctx context.Context, playerID int) (int, int, error) {
	if m.creditBalances == nil {
		return 0, 0, nil
	}
	return m.creditBalances[playerID], m.creditTotalEarned[playerID], nil
}

func (m *mockRepo) AddCredits(ctx context.Context, playerID, amount int) error {
	if m.creditBalances == nil {
		m.creditBalances = make(map[int]int)
		m.creditTotalEarned = make(map[int]int)
	}
	m.creditBalances[playerID] += amount
	m.creditTotalEarned[playerID] += amount
	return nil
}

func (m *mockRepo) SpendCredits(ctx context.Context, playerID, amount int) error {
	if m.creditBalances == nil {
		return fmt.Errorf("insufficient credits")
	}
	if m.creditBalances[playerID] < amount {
		return fmt.Errorf("insufficient credits")
	}
	m.creditBalances[playerID] -= amount
	return nil
}

func (m *mockRepo) AddCreditsTransaction(ctx context.Context, playerID, amount, balanceAfter int, reason string) error {
	m.creditsTransactions = append(m.creditsTransactions, CreditsTransaction{
		ID:           len(m.creditsTransactions) + 1,
		PlayerID:     playerID,
		Amount:       amount,
		BalanceAfter: balanceAfter,
		Reason:       reason,
		CreatedAt:    time.Now(),
	})
	return nil
}

func (m *mockRepo) ListCreditsTransactions(ctx context.Context, playerID, limit int) ([]CreditsTransaction, error) {
	var result []CreditsTransaction
	for _, tx := range m.creditsTransactions {
		if tx.PlayerID == playerID {
			result = append(result, tx)
		}
	}
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (m *mockRepo) HireCommander(ctx context.Context, playerID int, commanderType string, level int, expiresAt time.Time) (CommanderEntry, error) {
	for i, c := range m.commanders {
		if c.PlayerID == playerID && c.CommanderType == commanderType {
			m.commanders = append(m.commanders[:i], m.commanders[i+1:]...)
			break
		}
	}
	entry := CommanderEntry{
		ID:            len(m.commanders) + 1,
		PlayerID:      playerID,
		CommanderType: commanderType,
		Level:         level,
		HiredAt:       time.Now(),
		ExpiresAt:     expiresAt,
	}
	m.commanders = append(m.commanders, entry)
	return entry, nil
}

func (m *mockRepo) GetActiveCommanders(ctx context.Context, playerID int) ([]CommanderEntry, error) {
	var result []CommanderEntry
	now := time.Now()
	for _, c := range m.commanders {
		if c.PlayerID == playerID && c.ExpiresAt.After(now) {
			result = append(result, c)
		}
	}
	return result, nil
}

func (m *mockRepo) GetCommander(ctx context.Context, playerID int, commanderType string) (*CommanderEntry, error) {
	for _, c := range m.commanders {
		if c.PlayerID == playerID && c.CommanderType == commanderType {
			return &c, nil
		}
	}
	return nil, nil
}

func (m *mockRepo) ListAllCommanders(ctx context.Context, playerID int) ([]CommanderEntry, error) {
	var result []CommanderEntry
	for _, c := range m.commanders {
		if c.PlayerID == playerID {
			result = append(result, c)
		}
	}
	return result, nil
}

func (m *mockRepo) GetDailyGiftStatus(ctx context.Context, playerID int) (int, int, string, error) {
	if m.dailyGiftStreak == nil {
		return 0, 0, "", nil
	}
	entry, ok := m.dailyGiftStreak[playerID]
	if !ok {
		return 0, 0, "", nil
	}
	return entry.StreakDay, entry.ConsecutiveDays, entry.LastClaimDate, nil
}

func (m *mockRepo) ClaimDailyGift(ctx context.Context, playerID int) (int, int, error) {
	if m.dailyGiftStreak == nil {
		m.dailyGiftStreak = make(map[int]dailyGiftStreakEntry)
	}
	today := time.Now().Format("2006-01-02")
	entry, ok := m.dailyGiftStreak[playerID]
	if !ok {
		m.dailyGiftStreak[playerID] = dailyGiftStreakEntry{StreakDay: 1, ConsecutiveDays: 1, LastClaimDate: today}
		return 1, 1, nil
	}
	if entry.LastClaimDate == today {
		return entry.StreakDay, entry.ConsecutiveDays, nil
	}
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	if entry.LastClaimDate == yesterday {
		newDay := entry.StreakDay + 1
		if newDay > 7 {
			newDay = 1
		}
		m.dailyGiftStreak[playerID] = dailyGiftStreakEntry{
			StreakDay:       newDay,
			ConsecutiveDays: entry.ConsecutiveDays + 1,
			LastClaimDate:   today,
		}
		return newDay, entry.ConsecutiveDays + 1, nil
	}
	m.dailyGiftStreak[playerID] = dailyGiftStreakEntry{StreakDay: 1, ConsecutiveDays: 1, LastClaimDate: today}
	return 1, 1, nil
}

func (m *mockRepo) ResetDailyGiftStreak(ctx context.Context, playerID int) error {
	if m.dailyGiftStreak == nil {
		m.dailyGiftStreak = make(map[int]dailyGiftStreakEntry)
	}
	m.dailyGiftStreak[playerID] = dailyGiftStreakEntry{StreakDay: 0, ConsecutiveDays: 0, LastClaimDate: time.Now().Format("2006-01-02")}
	return nil
}

func (m *mockRepo) GetDailyTasks(ctx context.Context, playerID int) ([]dailyTaskRow, error) {
	today := time.Now().Format("2006-01-02")
	var result []dailyTaskRow
	for _, t := range m.dailyTasks {
		if t.PlayerID == playerID && t.AssignedDate == today {
			result = append(result, t)
		}
	}
	return result, nil
}

func (m *mockRepo) AssignDailyTasks(ctx context.Context, playerID int, tasks []dailyTaskRow) error {
	for _, t := range tasks {
		t.ID = m.taskNextID + 1
		m.taskNextID++
		t.AssignedDate = time.Now().Format("2006-01-02")
		m.dailyTasks = append(m.dailyTasks, t)
	}
	return nil
}

func (m *mockRepo) UpdateTaskProgress(ctx context.Context, taskID, playerID, progress int) error {
	for i, t := range m.dailyTasks {
		if t.ID == taskID && t.PlayerID == playerID {
			newProgress := t.Progress + progress
			if newProgress > t.TargetAmount {
				newProgress = t.TargetAmount
			}
			m.dailyTasks[i].Progress = newProgress
			return nil
		}
	}
	return fmt.Errorf("task not found")
}

func (m *mockRepo) MarkTaskCompleted(ctx context.Context, taskID, playerID int) error {
	for i, t := range m.dailyTasks {
		if t.ID == taskID && t.PlayerID == playerID && t.Progress >= t.TargetAmount {
			m.dailyTasks[i].Completed = true
			return nil
		}
	}
	return fmt.Errorf("task not found or not completable")
}

func (m *mockRepo) ClaimTaskReward(ctx context.Context, taskID, playerID int) (dailyTaskRow, error) {
	for i, t := range m.dailyTasks {
		if t.ID == taskID && t.PlayerID == playerID && t.Completed && !t.Claimed {
			m.dailyTasks[i].Claimed = true
			return m.dailyTasks[i], nil
		}
	}
	return dailyTaskRow{}, fmt.Errorf("task not found or already claimed")
}

func (m *mockRepo) ClaimAllTasksReward(ctx context.Context, playerID int) ([]dailyTaskRow, error) {
	var claimed []dailyTaskRow
	for i, t := range m.dailyTasks {
		if t.PlayerID == playerID && t.Completed && !t.Claimed {
			m.dailyTasks[i].Claimed = true
			claimed = append(claimed, m.dailyTasks[i])
		}
	}
	return claimed, nil
}

func (m *mockRepo) IncrementTaskRerolls(ctx context.Context, playerID int) error {
	for i, t := range m.dailyTasks {
		if t.PlayerID == playerID {
			m.dailyTasks[i].RerollsUsed++
		}
	}
	return nil
}

func (m *mockRepo) CreateStorePurchase(ctx context.Context, playerID int, itemID string, cost int, currency string) error {
	return nil
}

func (m *mockRepo) GetPlayerShifterUses(ctx context.Context, playerID int) (int, error) {
	return 0, nil
}

func (m *mockRepo) SetPlayerShifterUses(ctx context.Context, playerID int, uses int) error {
	return nil
}

func (m *mockRepo) ExtendCommanderDuration(ctx context.Context, playerID int, commanderType string, days int) error {
	return nil
}

func (m *mockRepo) GetPlayerPlanetID(ctx context.Context, playerID int) (int, error) {
	return 1, nil
}

func (m *mockRepo) AddGalactoniteShards(ctx context.Context, playerID int, shardType string, count int) error {
	if m.dmBalances == nil {
		m.dmBalances = make(map[int]int)
	}
	m.dmBalances[playerID] += count
	return nil
}

func (m *mockRepo) GetGalactoniteDiscovererLevel(ctx context.Context, playerID int) (int, error) {
	if m.discovererLevels == nil {
		return 0, nil
	}
	return m.discovererLevels[playerID], nil
}

func (m *mockRepo) UpgradeGalactoniteDiscoverer(ctx context.Context, playerID int) (int, error) {
	if m.discovererLevels == nil {
		m.discovererLevels = make(map[int]int)
	}
	m.discovererLevels[playerID]++
	return m.discovererLevels[playerID], nil
}

func (m *mockRepo) RerollTask(ctx context.Context, taskID, playerID int) (dailyTaskRow, error) {
	for i, t := range m.dailyTasks {
		if t.ID == taskID && t.PlayerID == playerID && !t.Claimed {
			m.dailyTasks = append(m.dailyTasks[:i], m.dailyTasks[i+1:]...)
			return dailyTaskRow{}, nil
		}
	}
	return dailyTaskRow{}, fmt.Errorf("task not found")
}
