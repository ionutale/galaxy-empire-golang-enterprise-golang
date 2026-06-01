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
	IsAdmin(ctx context.Context, playerID int) (bool, error)

	SearchUsers(ctx context.Context, q string, page, limit int) ([]UserSearchResult, int, error)

	GetPlanet(ctx context.Context, planetID int) (PlanetView, error)
	GetPlanetsByUserID(ctx context.Context, userID int) ([]int, error)
	GetPlanetBuildings(ctx context.Context, planetIDs []int) (map[int][]BuildingInfo, error)
	GetPlanetShips(ctx context.Context, planetID int) (map[string]int, error)
	GetPlanetDefenses(ctx context.Context, planetID int) (map[string]int, error)

	UpdatePlanetResources(ctx context.Context, planetID, metal, crystal, gas int) error

	AddDM(ctx context.Context, playerID, amount int) error
	AddCredits(ctx context.Context, playerID, amount int) error

	SetBanned(ctx context.Context, playerID int, banned bool) error
	IsBanned(ctx context.Context, playerID int) (bool, error)

	GetPlayerPlanetID(ctx context.Context, playerID int) (int, error)
	CreateNotification(ctx context.Context, playerID int, category, title, message string) error

	CreateEvent(ctx context.Context, name, description, eventType string, modifiers map[string]any, startsAt, endsAt time.Time) error
}

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) IsAdmin(ctx context.Context, playerID int) (bool, error) {
	if playerID == 1 {
		return true, nil
	}
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM admin.admins WHERE player_id = $1`, playerID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check admin: %w", err)
	}
	return count > 0, nil
}

func (r *PostgresRepository) SearchUsers(ctx context.Context, q string, page, limit int) ([]UserSearchResult, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM auth.users WHERE email ILIKE '%' || $1 || '%'`, q,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	offset := (page - 1) * limit
	rows, err := r.pool.Query(ctx,
		`SELECT id, email, created_at FROM auth.users
		 WHERE email ILIKE '%' || $1 || '%'
		 ORDER BY created_at DESC
		 LIMIT $2 OFFSET $3`, q, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("search users: %w", err)
	}
	defer rows.Close()

	var users []UserSearchResult
	for rows.Next() {
		var u UserSearchResult
		if err := rows.Scan(&u.ID, &u.Email, &u.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}
	return users, total, rows.Err()
}

func (r *PostgresRepository) GetPlanet(ctx context.Context, planetID int) (PlanetView, error) {
	var p PlanetView
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, name, galaxy, system, position, metal, crystal, gas, energy, max_fields, type
		 FROM planet.planets WHERE id = $1`, planetID,
	).Scan(&p.ID, &p.UserID, &p.Name, &p.Galaxy, &p.System, &p.Position, &p.Metal, &p.Crystal, &p.Gas, &p.Energy, &p.MaxFields, &p.Type)
	if err != nil {
		if err == pgx.ErrNoRows {
			return PlanetView{}, fmt.Errorf("planet not found")
		}
		return PlanetView{}, fmt.Errorf("get planet: %w", err)
	}
	return p, nil
}

func (r *PostgresRepository) GetPlanetsByUserID(ctx context.Context, userID int) ([]int, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id FROM planet.planets WHERE user_id = $1 ORDER BY id`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("get planets by user: %w", err)
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan planet id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *PostgresRepository) GetPlanetBuildings(ctx context.Context, planetIDs []int) (map[int][]BuildingInfo, error) {
	result := make(map[int][]BuildingInfo)
	if len(planetIDs) == 0 {
		return result, nil
	}

	rows, err := r.pool.Query(ctx,
		`SELECT planet_id, type, level FROM planet.buildings WHERE planet_id = ANY($1) ORDER BY type`,
		planetIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("get buildings: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var planetID int
		var b BuildingInfo
		if err := rows.Scan(&planetID, &b.Type, &b.Level); err != nil {
			return nil, fmt.Errorf("scan building: %w", err)
		}
		result[planetID] = append(result[planetID], b)
	}
	return result, rows.Err()
}

func (r *PostgresRepository) GetPlanetShips(ctx context.Context, planetID int) (map[string]int, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT ship_type, quantity FROM planet.player_ships WHERE planet_id = $1`, planetID,
	)
	if err != nil {
		return nil, fmt.Errorf("get ships: %w", err)
	}
	defer rows.Close()

	ships := make(map[string]int)
	for rows.Next() {
		var shipType string
		var qty int
		if err := rows.Scan(&shipType, &qty); err != nil {
			return nil, fmt.Errorf("scan ship: %w", err)
		}
		ships[shipType] = qty
	}
	return ships, rows.Err()
}

func (r *PostgresRepository) GetPlanetDefenses(ctx context.Context, planetID int) (map[string]int, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT defense_type, quantity FROM planet.player_defenses WHERE planet_id = $1`, planetID,
	)
	if err != nil {
		return nil, fmt.Errorf("get defenses: %w", err)
	}
	defer rows.Close()

	defenses := make(map[string]int)
	for rows.Next() {
		var defType string
		var qty int
		if err := rows.Scan(&defType, &qty); err != nil {
			return nil, fmt.Errorf("scan defense: %w", err)
		}
		defenses[defType] = qty
	}
	return defenses, rows.Err()
}

func (r *PostgresRepository) UpdatePlanetResources(ctx context.Context, planetID, metal, crystal, gas int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE planet.planets SET metal = $1, crystal = $2, gas = $3, resources_updated_at = NOW() WHERE id = $4`,
		metal, crystal, gas, planetID,
	)
	return err
}

func (r *PostgresRepository) AddDM(ctx context.Context, playerID, amount int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE planet.player_progress SET vip_points = GREATEST(0, vip_points + $1) WHERE user_id = $2`,
		amount, playerID,
	)
	return err
}

func (r *PostgresRepository) AddCredits(ctx context.Context, playerID, amount int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE planet.player_progress SET total_resources_produced = GREATEST(0, total_resources_produced::bigint + $1) WHERE user_id = $2`,
		amount, playerID,
	)
	return err
}

func (r *PostgresRepository) SetBanned(ctx context.Context, playerID int, banned bool) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE auth.users SET banned = $1, updated_at = NOW() WHERE id = $2`,
		banned, playerID,
	)
	return err
}

func (r *PostgresRepository) IsBanned(ctx context.Context, playerID int) (bool, error) {
	var banned bool
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(banned, FALSE) FROM auth.users WHERE id = $1`, playerID,
	).Scan(&banned)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, fmt.Errorf("user not found")
		}
		return false, fmt.Errorf("check banned: %w", err)
	}
	return banned, nil
}

func (r *PostgresRepository) GetPlayerPlanetID(ctx context.Context, playerID int) (int, error) {
	var planetID int
	err := r.pool.QueryRow(ctx,
		`SELECT id FROM planet.planets WHERE user_id = $1 ORDER BY id LIMIT 1`, playerID,
	).Scan(&planetID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, fmt.Errorf("player has no planets")
		}
		return 0, fmt.Errorf("get player planet: %w", err)
	}
	return planetID, nil
}

func (r *PostgresRepository) CreateNotification(ctx context.Context, playerID int, category, title, message string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO notification.notifications (player_id, category, title, message)
		 VALUES ($1, $2, $3, $4)`,
		playerID, category, title, message,
	)
	return err
}

func (r *PostgresRepository) CreateEvent(ctx context.Context, name, description, eventType string, modifiers map[string]any, startsAt, endsAt time.Time) error {
	modBytes, _ := json.Marshal(modifiers)
	_, err := r.pool.Exec(ctx,
		`INSERT INTO event.events (name, description, event_type, modifiers, starts_at, ends_at, status)
		 VALUES ($1, $2, $3, $4, $5, $6, 'upcoming')`,
		name, description, eventType, modBytes, startsAt, endsAt,
	)
	return err
}
