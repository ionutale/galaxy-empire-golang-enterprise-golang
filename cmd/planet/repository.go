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
	CreateAtCoords(ctx context.Context, userID int, galaxy, system, position int) (Planet, []Building, error)
	SeedBuildingsForPlanet(ctx context.Context, planetID int) error
	SeedShipsForPlanet(ctx context.Context, planetID int) error
	SeedDefensesForPlanet(ctx context.Context, planetID int) error
	SeedTechnologiesForPlanet(ctx context.Context, planetID int) error
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
	AddTechLevel(ctx context.Context, userID int, techType string, level int) error
	GetHighestLabLevel(ctx context.Context, playerID int) (int, error)
	GetPlayerProgress(ctx context.Context, planetID int) (vipPoints int, totalResources int, err error)
	AddVIPPoints(ctx context.Context, planetID int, points int) error
	AddResourcesProduced(ctx context.Context, planetID int, amount int) error
	FindByCoords(ctx context.Context, galaxy, system, position int) (Planet, error)
	ListGalaxies(ctx context.Context) ([]Galaxy, error)
	ListSystems(ctx context.Context, galaxyID int, page, pageSize int) ([]System, int, error)
	GetSystemPositions(ctx context.Context, systemID int) ([]Position, error)
	GetPlayerShips(ctx context.Context, planetID int) (map[string]int, error)
	AddPlayerShips(ctx context.Context, planetID, planetUserID int, shipType string, quantity int) error
	DeductPlayerShips(ctx context.Context, planetID int, ships map[string]int) error
	GetPlayerDefenses(ctx context.Context, planetID int) (map[string]int, error)
	AddPlayerDefenses(ctx context.Context, planetID, planetUserID int, defenseType string, quantity int) error
	GetPlayerDefense(ctx context.Context, planetID int, defenseType string) (int, error)
	SetPlayerDefense(ctx context.Context, planetID int, defenseType string, quantity int) error

	// Moon buildings
	MoonExists(ctx context.Context, galaxy, system, position int) (bool, error)
	GetMoonBuildings(ctx context.Context, galaxy, system, position int) ([]MoonBuilding, error)
	GetMoonBuildingLevel(ctx context.Context, galaxy, system, position int, buildingType string) (int, error)
	UpdateMoonBuildingLevel(ctx context.Context, galaxy, system, position int, buildingType string, level int) error

	// Wormholes
	GetWormhole(ctx context.Context, galaxy, system, position int) (*WormholeEntry, error)
	CreateOrUpdateWormhole(ctx context.Context, galaxy, system, position int, level int) error
	LinkWormholes(ctx context.Context, srcGalaxy, srcSystem, srcPos, dstGalaxy, dstSystem, dstPos int) error
	GetLinkedWormhole(ctx context.Context, galaxy, system, position int) (*WormholeEntry, error)
	UpdatePlanetName(ctx context.Context, planetID int, name string) error

	// Missiles
	GetMissileCounts(ctx context.Context, planetID int) (ipms, abms int, err error)
	AddIPMs(ctx context.Context, planetID, count int) error
	AddABMs(ctx context.Context, planetID, count int) error
	DeductIPMs(ctx context.Context, planetID, count int) error
	DeductABMs(ctx context.Context, planetID, count int) error

	// Star Gate
	StarGateLink(ctx context.Context, planetID, targetPlanetID int) error
	StarGateUnlink(ctx context.Context, planetID int) error
	GetStarGateLink(ctx context.Context, planetID int) (*StarGateLink, error)

	// Gem slots
	GetGemSlots(ctx context.Context, planetID int) ([]GemSlot, error)
	SetGemSlot(ctx context.Context, planetID, slotIndex int, gemType string, starLevel int) error

	// Galactonite shards
	GetShardCount(ctx context.Context, playerID int) (map[string]int, error)
	AddShards(ctx context.Context, playerID int, gemType string, count int) error
	RemoveShards(ctx context.Context, playerID int, gemType string, count int) error
	IncrementCombineAttempts(ctx context.Context, playerID int, gemType string) error

	// NPC planets
	CreateNPCPlanet(ctx context.Context, galaxy, system, position int, planetType string, temperature int) (int, error)
	SeedNPCResources(ctx context.Context, planetID int) error
	SeedNPCFleet(ctx context.Context, planetID int) error
	RegisterNPCPlanet(ctx context.Context, planetID, galaxy, system, position int) error
	GetNPCPlanetByPlanetID(ctx context.Context, planetID int) (*NPCPlanet, error)
	GetRespawnedNPCPlanets(ctx context.Context) ([]NPCPlanet, error)
	ClearNPCPlanet(ctx context.Context, planetID int) error
	RespawnNPCPlanet(ctx context.Context, npcPlanetID int) error
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

func (r *PostgresRepository) FindByCoords(ctx context.Context, galaxy, system, position int) (Planet, error) {
	var p Planet
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, name, metal, crystal, gas, energy,
		        galaxy, system, position, max_fields, type, temperature, resources_updated_at
		 FROM planet.planets WHERE galaxy = $1 AND system = $2 AND position = $3`,
		galaxy, system, position,
	).Scan(&p.ID, &p.UserID, &p.Name, &p.Metal, &p.Crystal, &p.Gas, &p.Energy,
		&p.Galaxy, &p.System, &p.Position, &p.MaxFields, &p.Type, &p.Temperature, &p.ResourcesUpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Planet{}, ErrPlanetNotFound
		}
		return Planet{}, fmt.Errorf("find planet by coords: %w", err)
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
		INSERT INTO planet.buildings (planet_id, type, level)
		VALUES ($1, 'missile_silo', 1)
		ON CONFLICT (planet_id, type) DO NOTHING
	`, p.ID); err != nil {
		return Planet{}, nil, fmt.Errorf("seed missile silo: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO planet.player_technologies (user_id, type, level)
		VALUES ($1, 'energy_tech', 3)
		ON CONFLICT (user_id, type) DO NOTHING
	`, userID); err != nil {
		return Planet{}, nil, fmt.Errorf("insert default tech: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO planet.player_progress (user_id, vip_points, total_resources_produced)
		VALUES ($1, 0, 0)
		ON CONFLICT (user_id) DO NOTHING
	`, userID); err != nil {
		return Planet{}, nil, fmt.Errorf("create progress: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO planet.buildings (planet_id, type, level)
		VALUES ($1, 'small_shield_dome', 0), ($1, 'large_shield_dome', 0)
		ON CONFLICT (planet_id, type) DO NOTHING
	`, p.ID); err != nil {
		return Planet{}, nil, fmt.Errorf("seed shield domes: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Planet{}, nil, fmt.Errorf("commit: %w", err)
	}
	return p, buildings, nil
}

func (r *PostgresRepository) CreateAtCoords(ctx context.Context, userID int, galaxy, system, position int) (Planet, []Building, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Planet{}, nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	typ, temp := planetTypeAndTemp(position)
	name := fmt.Sprintf("Colony [%d:%d:%d]", galaxy, system, position)

	var p Planet
	err = tx.QueryRow(ctx,
		`INSERT INTO planet.planets (user_id, name, galaxy, system, position, max_fields, type, temperature, resources_updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		 RETURNING id, user_id, name, metal, crystal, gas, energy,
		           galaxy, system, position, max_fields, type, temperature, resources_updated_at`,
		userID, name, galaxy, system, position, baseMaxFields, typ, temp,
	).Scan(&p.ID, &p.UserID, &p.Name, &p.Metal, &p.Crystal, &p.Gas, &p.Energy,
		&p.Galaxy, &p.System, &p.Position, &p.MaxFields, &p.Type, &p.Temperature, &p.ResourcesUpdatedAt)
	if err != nil {
		return Planet{}, nil, fmt.Errorf("insert planet: %w", err)
	}

	if err := r.SeedBuildingsForPlanet(ctx, p.ID); err != nil {
		return Planet{}, nil, err
	}

	buildings, err := r.GetBuildings(ctx, p.ID)
	if err != nil {
		return Planet{}, nil, fmt.Errorf("get buildings after seed: %w", err)
	}

	if err := r.SeedTechnologiesForPlanet(ctx, p.ID); err != nil {
		return Planet{}, nil, err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO planet.player_progress (user_id, vip_points, total_resources_produced)
		VALUES ($1, 0, 0)
		ON CONFLICT (user_id) DO NOTHING
	`, userID); err != nil {
		return Planet{}, nil, fmt.Errorf("create progress: %w", err)
	}

	if err := r.SeedShipsForPlanet(ctx, p.ID); err != nil {
		return Planet{}, nil, err
	}

	if err := r.SeedDefensesForPlanet(ctx, p.ID); err != nil {
		return Planet{}, nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Planet{}, nil, fmt.Errorf("commit: %w", err)
	}
	return p, buildings, nil
}

func (r *PostgresRepository) SeedBuildingsForPlanet(ctx context.Context, planetID int) error {
	seedTypes := []string{
		"metal_mine", "crystal_mine", "gas_mine", "solar_plant",
		"metal_storage", "crystal_storage", "gas_storage",
		"robotics_factory", "nanite_factory", "terraformer",
	}
	for _, bType := range seedTypes {
		_, err := r.pool.Exec(ctx,
			`INSERT INTO planet.buildings (planet_id, type, level)
			 VALUES ($1, $2, 1)
			 ON CONFLICT (planet_id, type) DO NOTHING`,
			planetID, bType,
		)
		if err != nil {
			return fmt.Errorf("seed building %s: %w", bType, err)
		}
	}

	if _, err := r.pool.Exec(ctx, `
		INSERT INTO planet.buildings (planet_id, type, level)
		VALUES ($1, 'missile_silo', 1)
		ON CONFLICT (planet_id, type) DO NOTHING
	`, planetID); err != nil {
		return fmt.Errorf("seed missile silo: %w", err)
	}

	return nil
}

func (r *PostgresRepository) SeedShipsForPlanet(ctx context.Context, planetID int) error {
	shipTypes := []string{
		"cargo", "large_cargo", "recycler", "espionage_probe", "colony_ship",
		"solar_satellite", "light_fighter", "heavy_fighter", "cruiser",
		"battleship", "dreadnought", "bomber",
	}
	for _, sType := range shipTypes {
		_, err := r.pool.Exec(ctx, `
			INSERT INTO planet.player_ships (planet_id, ship_type, quantity)
			VALUES ($1, $2, 0)
			ON CONFLICT (planet_id, ship_type) DO NOTHING
		`, planetID, sType)
		if err != nil {
			return fmt.Errorf("seed ship %s: %w", sType, err)
		}
	}
	return nil
}

func (r *PostgresRepository) SeedDefensesForPlanet(ctx context.Context, planetID int) error {
	defenseTypes := []string{
		"rocket_launcher", "light_laser", "heavy_laser", "mk2_cannon",
		"ion_cannon", "plasma_cannon", "proton_cannon",
	}
	for _, dType := range defenseTypes {
		_, err := r.pool.Exec(ctx, `
			INSERT INTO planet.player_defenses (planet_id, defense_type, quantity)
			VALUES ($1, $2, 0)
			ON CONFLICT (planet_id, defense_type) DO NOTHING
		`, planetID, dType)
		if err != nil {
			return fmt.Errorf("seed defense %s: %w", dType, err)
		}
	}
	return nil
}

func (r *PostgresRepository) SeedTechnologiesForPlanet(ctx context.Context, planetID int) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO planet.player_technologies (user_id, type, level)
		SELECT p.user_id, 'energy_tech', 3
		FROM planet.planets p
		WHERE p.id = $1
		ON CONFLICT (user_id, type) DO NOTHING
	`, planetID)
	return err
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

func (r *PostgresRepository) UpdatePlanetName(ctx context.Context, planetID int, name string) error {
	_, err := r.pool.Exec(ctx, `UPDATE planet.planets SET name = $1, updated_at = NOW() WHERE id = $2`, name, planetID)
	if err != nil {
		return fmt.Errorf("update planet name: %w", err)
	}
	return nil
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

func (r *PostgresRepository) AddTechLevel(ctx context.Context, userID int, techType string, level int) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO planet.player_technologies (user_id, type, level)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user_id, type) DO UPDATE SET level = $3`,
		userID, techType, level,
	)
	if err != nil {
		return fmt.Errorf("add tech level: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetHighestLabLevel(ctx context.Context, playerID int) (int, error) {
	var maxLevel int
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(MAX(b.level), 0)
		 FROM planet.buildings b
		 JOIN planet.planets p ON p.id = b.planet_id
		 WHERE p.user_id = $1 AND b.type = 'research_lab'`,
		playerID,
	).Scan(&maxLevel)
	if err != nil {
		return 0, fmt.Errorf("get highest lab level: %w", err)
	}
	return maxLevel, nil
}

func (r *PostgresRepository) GetPlayerProgress(ctx context.Context, planetID int) (int, int, error) {
	var vipPoints int
	var totalResources int64
	err := r.pool.QueryRow(ctx,
		`SELECT pp.vip_points, pp.total_resources_produced
		 FROM planet.player_progress pp
		 JOIN planet.planets p ON p.user_id = pp.user_id
		 WHERE p.id = $1`,
		planetID,
	).Scan(&vipPoints, &totalResources)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, 0, nil
		}
		return 0, 0, fmt.Errorf("get player progress: %w", err)
	}
	return vipPoints, int(totalResources), nil
}

func (r *PostgresRepository) AddVIPPoints(ctx context.Context, planetID int, points int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE planet.player_progress pp
		 SET vip_points = pp.vip_points + $1
		 FROM planet.planets p
		 WHERE p.id = $2 AND pp.user_id = p.user_id`,
		points, planetID,
	)
	if err != nil {
		return fmt.Errorf("add vip points: %w", err)
	}
	return nil
}

func (r *PostgresRepository) AddResourcesProduced(ctx context.Context, planetID int, amount int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE planet.player_progress pp
		 SET total_resources_produced = pp.total_resources_produced + $1
		 FROM planet.planets p
		 WHERE p.id = $2 AND pp.user_id = p.user_id`,
		amount, planetID,
	)
	if err != nil {
		return fmt.Errorf("add resources produced: %w", err)
	}
	return nil
}

func (r *PostgresRepository) ListGalaxies(ctx context.Context) ([]Galaxy, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, name FROM galaxy.galaxies ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list galaxies: %w", err)
	}
	defer rows.Close()

	var galaxies []Galaxy
	for rows.Next() {
		var g Galaxy
		if err := rows.Scan(&g.ID, &g.Name); err != nil {
			return nil, fmt.Errorf("scan galaxy: %w", err)
		}
		galaxies = append(galaxies, g)
	}
	return galaxies, rows.Err()
}

func (r *PostgresRepository) ListSystems(ctx context.Context, galaxyID int, page, pageSize int) ([]System, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM galaxy.systems WHERE galaxy_id = $1`, galaxyID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count systems: %w", err)
	}

	offset := (page - 1) * pageSize
	rows, err := r.pool.Query(ctx, `
		SELECT s.id, s.system_num,
			(SELECT COUNT(*) FROM planet.planets pl WHERE pl.galaxy = s.galaxy_id AND pl.system = s.system_num) AS occupied_count
		FROM galaxy.systems s
		WHERE s.galaxy_id = $1
		ORDER BY s.system_num
		LIMIT $2 OFFSET $3
	`, galaxyID, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list systems: %w", err)
	}
	defer rows.Close()

	var systems []System
	for rows.Next() {
		var s System
		if err := rows.Scan(&s.ID, &s.SystemNum, &s.OccupiedCount); err != nil {
			return nil, 0, fmt.Errorf("scan system: %w", err)
		}
		systems = append(systems, s)
	}
	return systems, total, rows.Err()
}

func (r *PostgresRepository) GetSystemPositions(ctx context.Context, systemID int) ([]Position, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT p.position_num,
			CASE WHEN pl.id IS NOT NULL THEN
				CASE WHEN pl.user_id = 0 THEN 'npc' ELSE 'occupied' END
			ELSE 'empty' END AS state,
			COALESCE(pl.name, '') AS planet_name,
			COALESCE(pl.user_id, 0) AS player_id
		FROM galaxy.positions p
		JOIN galaxy.systems s ON p.system_id = s.id
		LEFT JOIN planet.planets pl
			ON pl.galaxy = s.galaxy_id
			AND pl.system = s.system_num
			AND pl.position = p.position_num
		WHERE p.system_id = $1
		ORDER BY p.position_num
	`, systemID)
	if err != nil {
		return nil, fmt.Errorf("get positions: %w", err)
	}
	defer rows.Close()

	var positions []Position
	for rows.Next() {
		var pos Position
		if err := rows.Scan(&pos.PositionNum, &pos.State, &pos.PlanetName, &pos.PlayerID); err != nil {
			return nil, fmt.Errorf("scan position: %w", err)
		}
		positions = append(positions, pos)
	}
	return positions, rows.Err()
}

func (r *PostgresRepository) GetPlayerShips(ctx context.Context, planetID int) (map[string]int, error) {
	rows, err := r.pool.Query(ctx, `SELECT ship_type, quantity FROM planet.player_ships WHERE planet_id = $1`, planetID)
	if err != nil {
		return nil, fmt.Errorf("get player ships: %w", err)
	}
	defer rows.Close()

	ships := make(map[string]int)
	for rows.Next() {
		var shipType string
		var quantity int
		if err := rows.Scan(&shipType, &quantity); err != nil {
			return nil, fmt.Errorf("scan ship: %w", err)
		}
		ships[shipType] = quantity
	}
	return ships, rows.Err()
}

func (r *PostgresRepository) AddPlayerShips(ctx context.Context, planetID, planetUserID int, shipType string, quantity int) error {
	if _, err := r.pool.Exec(ctx, `
		INSERT INTO planet.player_ships (planet_id, ship_type, quantity)
		VALUES ($1, $2, $3)
		ON CONFLICT (planet_id, ship_type) DO UPDATE SET quantity = planet.player_ships.quantity + $3
	`, planetID, shipType, quantity); err != nil {
		return fmt.Errorf("add player ships: %w", err)
	}
	return nil
}

func (r *PostgresRepository) DeductPlayerShips(ctx context.Context, planetID int, ships map[string]int) error {
	for shipType, qty := range ships {
		tag, err := r.pool.Exec(ctx, `
			UPDATE planet.player_ships
			SET quantity = quantity - $1
			WHERE planet_id = $2 AND ship_type = $3 AND quantity >= $1
		`, qty, planetID, shipType)
		if err != nil {
			return fmt.Errorf("deduct ship %s: %w", shipType, err)
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("insufficient %s ships", shipType)
		}
	}
	return nil
}

func (r *PostgresRepository) GetPlayerDefenses(ctx context.Context, planetID int) (map[string]int, error) {
	rows, err := r.pool.Query(ctx, `SELECT defense_type, quantity FROM planet.player_defenses WHERE planet_id = $1`, planetID)
	if err != nil {
		return nil, fmt.Errorf("get player defenses: %w", err)
	}
	defer rows.Close()

	defenses := make(map[string]int)
	for rows.Next() {
		var defenseType string
		var quantity int
		if err := rows.Scan(&defenseType, &quantity); err != nil {
			return nil, fmt.Errorf("scan defense: %w", err)
		}
		defenses[defenseType] = quantity
	}
	return defenses, rows.Err()
}

func (r *PostgresRepository) AddPlayerDefenses(ctx context.Context, planetID, planetUserID int, defenseType string, quantity int) error {
	if _, err := r.pool.Exec(ctx, `
		INSERT INTO planet.player_defenses (planet_id, defense_type, quantity)
		VALUES ($1, $2, $3)
		ON CONFLICT (planet_id, defense_type) DO UPDATE SET quantity = planet.player_defenses.quantity + $3
	`, planetID, defenseType, quantity); err != nil {
		return fmt.Errorf("add player defenses: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetPlayerDefense(ctx context.Context, planetID int, defenseType string) (int, error) {
	var quantity int
	err := r.pool.QueryRow(ctx,
		`SELECT quantity FROM planet.player_defenses WHERE planet_id = $1 AND defense_type = $2`,
		planetID, defenseType,
	).Scan(&quantity)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}
		return 0, fmt.Errorf("get player defense: %w", err)
	}
	return quantity, nil
}

func (r *PostgresRepository) SetPlayerDefense(ctx context.Context, planetID int, defenseType string, quantity int) error {
	if _, err := r.pool.Exec(ctx, `
		INSERT INTO planet.player_defenses (planet_id, defense_type, quantity)
		VALUES ($1, $2, $3)
		ON CONFLICT (planet_id, defense_type) DO UPDATE SET quantity = $3
	`, planetID, defenseType, quantity); err != nil {
		return fmt.Errorf("set player defense: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetMissileCounts(ctx context.Context, planetID int) (int, int, error) {
	var ipms, abms int
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(missile_ipms, 0), COALESCE(missile_abms, 0) FROM planet.planets WHERE id = $1`,
		planetID,
	).Scan(&ipms, &abms)
	if err != nil {
		return 0, 0, fmt.Errorf("get missile counts: %w", err)
	}
	return ipms, abms, nil
}

func (r *PostgresRepository) AddIPMs(ctx context.Context, planetID, count int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE planet.planets SET missile_ipms = COALESCE(missile_ipms, 0) + $1 WHERE id = $2`,
		count, planetID,
	)
	if err != nil {
		return fmt.Errorf("add IPMs: %w", err)
	}
	return nil
}

func (r *PostgresRepository) AddABMs(ctx context.Context, planetID, count int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE planet.planets SET missile_abms = COALESCE(missile_abms, 0) + $1 WHERE id = $2`,
		count, planetID,
	)
	if err != nil {
		return fmt.Errorf("add ABMs: %w", err)
	}
	return nil
}

func (r *PostgresRepository) DeductIPMs(ctx context.Context, planetID, count int) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE planet.planets SET missile_ipms = COALESCE(missile_ipms, 0) - $1 WHERE id = $2 AND COALESCE(missile_ipms, 0) >= $1`,
		count, planetID,
	)
	if err != nil {
		return fmt.Errorf("deduct IPMs: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("insufficient IPMs")
	}
	return nil
}

func (r *PostgresRepository) DeductABMs(ctx context.Context, planetID, count int) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE planet.planets SET missile_abms = COALESCE(missile_abms, 0) - $1 WHERE id = $2 AND COALESCE(missile_abms, 0) >= $1`,
		count, planetID,
	)
	if err != nil {
		return fmt.Errorf("deduct ABMs: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("insufficient ABMs")
	}
	return nil
}

func (r *PostgresRepository) MoonExists(ctx context.Context, galaxy, system, position int) (bool, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM fleet.moons WHERE galaxy = $1 AND system = $2 AND position = $3`,
		galaxy, system, position,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check moon exists: %w", err)
	}
	return count > 0, nil
}

func (r *PostgresRepository) GetMoonBuildings(ctx context.Context, galaxy, system, position int) ([]MoonBuilding, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, moon_galaxy, moon_system, moon_position, type, level
		 FROM planet.moon_buildings
		 WHERE moon_galaxy = $1 AND moon_system = $2 AND moon_position = $3
		 ORDER BY type`,
		galaxy, system, position,
	)
	if err != nil {
		return nil, fmt.Errorf("get moon buildings: %w", err)
	}
	defer rows.Close()

	var buildings []MoonBuilding
	for rows.Next() {
		var b MoonBuilding
		if err := rows.Scan(&b.ID, &b.MoonGalaxy, &b.MoonSystem, &b.MoonPos, &b.Type, &b.Level); err != nil {
			return nil, fmt.Errorf("scan moon building: %w", err)
		}
		buildings = append(buildings, b)
	}
	return buildings, rows.Err()
}

func (r *PostgresRepository) GetMoonBuildingLevel(ctx context.Context, galaxy, system, position int, buildingType string) (int, error) {
	var level int
	err := r.pool.QueryRow(ctx,
		`SELECT level FROM planet.moon_buildings
		 WHERE moon_galaxy = $1 AND moon_system = $2 AND moon_position = $3 AND type = $4`,
		galaxy, system, position, buildingType,
	).Scan(&level)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrBuildingNotFound
		}
		return 0, fmt.Errorf("get moon building level: %w", err)
	}
	return level, nil
}

func (r *PostgresRepository) UpdateMoonBuildingLevel(ctx context.Context, galaxy, system, position int, buildingType string, level int) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO planet.moon_buildings (moon_galaxy, moon_system, moon_position, type, level)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (moon_galaxy, moon_system, moon_position, type)
		 DO UPDATE SET level = $5`,
		galaxy, system, position, buildingType, level,
	)
	if err != nil {
		return fmt.Errorf("update moon building level: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetWormhole(ctx context.Context, galaxy, system, position int) (*WormholeEntry, error) {
	var w WormholeEntry
	err := r.pool.QueryRow(ctx,
		`SELECT id, moon_galaxy, moon_system, moon_position, level,
		        linked_galaxy, linked_system, linked_position, cooldown_until
		 FROM planet.wormhole_generators
		 WHERE moon_galaxy = $1 AND moon_system = $2 AND moon_position = $3`,
		galaxy, system, position,
	).Scan(&w.ID, &w.MoonGalaxy, &w.MoonSystem, &w.MoonPos, &w.Level,
		&w.LinkedGalaxy, &w.LinkedSystem, &w.LinkedPosition, &w.CooldownUntil)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrWormholeNotFound
		}
		return nil, fmt.Errorf("get wormhole: %w", err)
	}
	return &w, nil
}

func (r *PostgresRepository) CreateOrUpdateWormhole(ctx context.Context, galaxy, system, position int, level int) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO planet.wormhole_generators (moon_galaxy, moon_system, moon_position, level)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (moon_galaxy, moon_system, moon_position)
		 DO UPDATE SET level = $4`,
		galaxy, system, position, level,
	)
	if err != nil {
		return fmt.Errorf("create or update wormhole: %w", err)
	}
	return nil
}

func (r *PostgresRepository) LinkWormholes(ctx context.Context, srcGalaxy, srcSystem, srcPos, dstGalaxy, dstSystem, dstPos int) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	cooldown := time.Now().Add(1 * time.Hour)

	if _, err := tx.Exec(ctx,
		`UPDATE planet.wormhole_generators
		 SET linked_galaxy = $1, linked_system = $2, linked_position = $3, cooldown_until = $4
		 WHERE moon_galaxy = $5 AND moon_system = $6 AND moon_position = $7`,
		dstGalaxy, dstSystem, dstPos, cooldown, srcGalaxy, srcSystem, srcPos,
	); err != nil {
		return fmt.Errorf("link source wormhole: %w", err)
	}

	if _, err := tx.Exec(ctx,
		`UPDATE planet.wormhole_generators
		 SET linked_galaxy = $1, linked_system = $2, linked_position = $3, cooldown_until = $4
		 WHERE moon_galaxy = $5 AND moon_system = $6 AND moon_position = $7`,
		srcGalaxy, srcSystem, srcPos, cooldown, dstGalaxy, dstSystem, dstPos,
	); err != nil {
		return fmt.Errorf("link target wormhole: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *PostgresRepository) GetLinkedWormhole(ctx context.Context, galaxy, system, position int) (*WormholeEntry, error) {
	var w WormholeEntry
	err := r.pool.QueryRow(ctx,
		`SELECT id, moon_galaxy, moon_system, moon_position, level,
		        linked_galaxy, linked_system, linked_position, cooldown_until
		 FROM planet.wormhole_generators
		 WHERE (moon_galaxy = $1 AND moon_system = $2 AND moon_position = $3)
		    OR (linked_galaxy = $1 AND linked_system = $2 AND linked_position = $3)`,
		galaxy, system, position,
	).Scan(&w.ID, &w.MoonGalaxy, &w.MoonSystem, &w.MoonPos, &w.Level,
		&w.LinkedGalaxy, &w.LinkedSystem, &w.LinkedPosition, &w.CooldownUntil)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrWormholeNotFound
		}
		return nil, fmt.Errorf("get linked wormhole: %w", err)
	}
	return &w, nil
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

func (r *PostgresRepository) StarGateLink(ctx context.Context, planetID, targetPlanetID int) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO planet.stargate_links (planet_id, target_planet_id)
		 VALUES ($1, $2)`,
		planetID, targetPlanetID,
	)
	if err != nil {
		return fmt.Errorf("create stargate link: %w", err)
	}
	return nil
}

func (r *PostgresRepository) StarGateUnlink(ctx context.Context, planetID int) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM planet.stargate_links WHERE planet_id = $1`,
		planetID,
	)
	if err != nil {
		return fmt.Errorf("delete stargate link: %w", err)
	}
	_, err = r.pool.Exec(ctx,
		`DELETE FROM planet.stargate_links WHERE target_planet_id = $1`,
		planetID,
	)
	if err != nil {
		return fmt.Errorf("delete reverse stargate link: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetStarGateLink(ctx context.Context, planetID int) (*StarGateLink, error) {
	var l StarGateLink
	err := r.pool.QueryRow(ctx,
		`SELECT id, planet_id, target_planet_id FROM planet.stargate_links WHERE planet_id = $1`,
		planetID,
	).Scan(&l.ID, &l.PlanetID, &l.TargetPlanetID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get stargate link: %w", err)
	}
	return &l, nil
}

func (r *PostgresRepository) GetGemSlots(ctx context.Context, planetID int) ([]GemSlot, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, planet_id, slot_index, COALESCE(gem_type, ''), star_level
		FROM planet.gem_slots WHERE planet_id = $1 ORDER BY slot_index
	`, planetID)
	if err != nil {
		return nil, fmt.Errorf("get gem slots: %w", err)
	}
	defer rows.Close()

	var slots []GemSlot
	for rows.Next() {
		var s GemSlot
		if err := rows.Scan(&s.ID, &s.PlanetID, &s.SlotIndex, &s.GemType, &s.StarLevel); err != nil {
			return nil, fmt.Errorf("scan gem slot: %w", err)
		}
		slots = append(slots, s)
	}
	return slots, rows.Err()
}

func (r *PostgresRepository) SetGemSlot(ctx context.Context, planetID, slotIndex int, gemType string, starLevel int) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO planet.gem_slots (planet_id, slot_index, gem_type, star_level)
		VALUES ($1, $2, NULLIF($3, ''), $4)
		ON CONFLICT (planet_id, slot_index)
		DO UPDATE SET gem_type = NULLIF($3, ''), star_level = $4
	`, planetID, slotIndex, gemType, starLevel)
	if err != nil {
		return fmt.Errorf("set gem slot: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetShardCount(ctx context.Context, playerID int) (map[string]int, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT gem_type, count FROM planet.galactonite_shards WHERE player_id = $1
	`, playerID)
	if err != nil {
		return nil, fmt.Errorf("get shard count: %w", err)
	}
	defer rows.Close()

	shards := make(map[string]int)
	for rows.Next() {
		var gemType string
		var count int
		if err := rows.Scan(&gemType, &count); err != nil {
			return nil, fmt.Errorf("scan shard: %w", err)
		}
		shards[gemType] = count
	}
	return shards, rows.Err()
}

func (r *PostgresRepository) AddShards(ctx context.Context, playerID int, gemType string, count int) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO planet.galactonite_shards (player_id, gem_type, count)
		VALUES ($1, $2, $3)
		ON CONFLICT (player_id, gem_type)
		DO UPDATE SET count = planet.galactonite_shards.count + $3
	`, playerID, gemType, count)
	if err != nil {
		return fmt.Errorf("add shards: %w", err)
	}
	return nil
}

func (r *PostgresRepository) RemoveShards(ctx context.Context, playerID int, gemType string, count int) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE planet.galactonite_shards
		SET count = count - $3
		WHERE player_id = $1 AND gem_type = $2 AND count >= $3
	`, playerID, gemType, count)
	if err != nil {
		return fmt.Errorf("remove shards: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("insufficient shards")
	}
	return nil
}

func (r *PostgresRepository) IncrementCombineAttempts(ctx context.Context, playerID int, gemType string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO planet.galactonite_shards (player_id, gem_type, count, combine_attempts)
		VALUES ($1, $2, 0, 1)
		ON CONFLICT (player_id, gem_type)
		DO UPDATE SET combine_attempts = planet.galactonite_shards.combine_attempts + 1
	`, playerID, gemType)
	if err != nil {
		return fmt.Errorf("increment combine attempts: %w", err)
	}
	return nil
}

func (r *PostgresRepository) CreateNPCPlanet(ctx context.Context, galaxy, system, position int, planetType string, temperature int) (int, error) {
	var planetID int
	err := r.pool.QueryRow(ctx, `
		INSERT INTO planet.planets (user_id, name, galaxy, system, position, metal, crystal, gas, max_fields, type, temperature, resources_updated_at)
		VALUES (0, $1, $2, $3, $4, 50000, 25000, 10000, 40, $5, $6, NOW())
		RETURNING id
	`, fmt.Sprintf("NPC [%d:%d:%d]", galaxy, system, position), galaxy, system, position, planetType, temperature).Scan(&planetID)
	if err != nil {
		return 0, fmt.Errorf("create NPC planet: %w", err)
	}
	return planetID, nil
}

func (r *PostgresRepository) SeedNPCResources(ctx context.Context, planetID int) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE planet.planets SET metal = 50000, crystal = 25000, gas = 10000 WHERE id = $1
	`, planetID)
	return err
}

func (r *PostgresRepository) SeedNPCFleet(ctx context.Context, planetID int) error {
	npcShips := map[string]int{
		"light_fighter": 5,
		"cruiser":       3,
	}
	for shipType, qty := range npcShips {
		if err := r.AddPlayerShips(ctx, planetID, 0, shipType, qty); err != nil {
			return fmt.Errorf("seed NPC fleet %s: %w", shipType, err)
		}
	}
	return nil
}

func (r *PostgresRepository) RegisterNPCPlanet(ctx context.Context, planetID, galaxy, system, position int) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO planet.npc_planets (planet_id, galaxy, system, position, status)
		VALUES ($1, $2, $3, $4, 'active')
		ON CONFLICT (planet_id) DO NOTHING
	`, planetID, galaxy, system, position)
	return err
}

func (r *PostgresRepository) GetNPCPlanetByPlanetID(ctx context.Context, planetID int) (*NPCPlanet, error) {
	var npc NPCPlanet
	err := r.pool.QueryRow(ctx, `
		SELECT id, planet_id, galaxy, system, position, status, respawns_at
		FROM planet.npc_planets WHERE planet_id = $1
	`, planetID).Scan(&npc.ID, &npc.PlanetID, &npc.Galaxy, &npc.System, &npc.Position, &npc.Status, &npc.RespawnsAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get NPC planet: %w", err)
	}
	return &npc, nil
}

func (r *PostgresRepository) GetRespawnedNPCPlanets(ctx context.Context) ([]NPCPlanet, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, planet_id, galaxy, system, position, status, respawns_at
		FROM planet.npc_planets
		WHERE status = 'respawning' AND (respawns_at IS NULL OR respawns_at <= NOW())
	`)
	if err != nil {
		return nil, fmt.Errorf("get respawned NPC planets: %w", err)
	}
	defer rows.Close()

	var npcs []NPCPlanet
	for rows.Next() {
		var npc NPCPlanet
		if err := rows.Scan(&npc.ID, &npc.PlanetID, &npc.Galaxy, &npc.System, &npc.Position, &npc.Status, &npc.RespawnsAt); err != nil {
			return nil, fmt.Errorf("scan NPC planet: %w", err)
		}
		npcs = append(npcs, npc)
	}
	return npcs, rows.Err()
}

func (r *PostgresRepository) ClearNPCPlanet(ctx context.Context, planetID int) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE planet.npc_planets
		SET status = 'respawning', respawns_at = NOW() + (INTERVAL '1 minute' * (60 + floor(random() * 61)::int))
		WHERE planet_id = $1 AND status = 'active'
	`, planetID)
	if err != nil {
		return fmt.Errorf("clear NPC planet: %w", err)
	}
	return nil
}

func (r *PostgresRepository) RespawnNPCPlanet(ctx context.Context, npcPlanetID int) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var planetID int
	err = tx.QueryRow(ctx, `
		UPDATE planet.npc_planets SET status = 'active', respawns_at = NULL
		WHERE id = $1 AND status = 'respawning'
		RETURNING planet_id
	`, npcPlanetID).Scan(&planetID)
	if err != nil {
		return fmt.Errorf("update NPC status: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE planet.planets SET metal = 50000, crystal = 25000, gas = 10000 WHERE id = $1
	`, planetID); err != nil {
		return fmt.Errorf("reset NPC resources: %w", err)
	}

	return tx.Commit(ctx)
}
