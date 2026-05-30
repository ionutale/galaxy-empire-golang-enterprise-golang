package main

import (
	"context"
	"errors"
	"math"
	"time"
)

const baseStorage = 10000

type PlanetService struct {
	repo Repository
}

func NewPlanetService(repo Repository) *PlanetService {
	return &PlanetService{repo: repo}
}

func (s *PlanetService) GetOrCreatePlanet(ctx context.Context, userID int) (Planet, []Building, error) {
	planet, err := s.repo.FindByUserID(ctx, userID)
	if err != nil {
		if !errors.Is(err, ErrPlanetNotFound) {
			return Planet{}, nil, err
		}
		return s.repo.Create(ctx, userID)
	}

	buildings, err := s.repo.GetBuildings(ctx, planet.ID)
	if err != nil {
		return Planet{}, nil, err
	}

	prod := s.calculateProduction(buildings)
	storage := s.calculateStorage(buildings)
	elapsed := time.Since(planet.ResourcesUpdatedAt).Seconds()
	if elapsed > 0 {
		planet.Metal = minInt(planet.Metal+int(prod.Metal*elapsed), storage.Metal)
		planet.Crystal = minInt(planet.Crystal+int(prod.Crystal*elapsed), storage.Crystal)
		planet.Gas = minInt(planet.Gas+int(prod.Gas*elapsed), storage.Gas)
	}

	now := time.Now()
	planet.ResourcesUpdatedAt = now
	if err := s.repo.UpdateResources(ctx, planet.ID, planet.Metal, planet.Crystal, planet.Gas, now); err != nil {
		return Planet{}, nil, err
	}

	return planet, buildings, nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (s *PlanetService) calculateProduction(buildings []Building) Production {
	levels := make(map[string]int)
	for _, b := range buildings {
		levels[b.Type] = b.Level
	}

	return Production{
		Metal:   productionRate("metal_mine", levels["metal_mine"]) / 60.0,
		Crystal: productionRate("crystal_mine", levels["crystal_mine"]) / 60.0,
		Gas:     productionRate("gas_mine", levels["gas_mine"]) / 60.0,
		Energy:  productionRate("solar_plant", levels["solar_plant"]) / 60.0,
	}
}

func (s *PlanetService) calculateStorage(buildings []Building) Storage {
	levels := make(map[string]int)
	for _, b := range buildings {
		levels[b.Type] = b.Level
	}

	return Storage{
		Metal:   storageCapacity("metal_storage", levels["metal_storage"]),
		Crystal: storageCapacity("crystal_storage", levels["crystal_storage"]),
		Gas:     storageCapacity("gas_storage", levels["gas_storage"]),
	}
}

func productionRate(buildingType string, level int) float64 {
	if level < 1 {
		return 0
	}
	switch buildingType {
	case "metal_mine":
		return math.Round(30*float64(level)*math.Pow(1.1, float64(level))*100) / 100
	case "crystal_mine":
		return math.Round(20*float64(level)*math.Pow(1.1, float64(level))*100) / 100
	case "gas_mine":
		return math.Round(10*float64(level)*math.Pow(1.1, float64(level))*100) / 100
	case "solar_plant":
		return math.Round(20*float64(level)*math.Pow(1.1, float64(level))*100) / 100
	}
	return 0
}

func storageCapacity(buildingType string, level int) int {
	if level < 1 {
		return baseStorage
	}
	bonus := int(5000 * math.Pow(1.5, float64(level)))
	return baseStorage + bonus
}

func toPlanetResponse(p Planet, buildings []Building, prod Production, storage Storage) PlanetResponse {
	return PlanetResponse{
		ID: p.ID, UserID: p.UserID, Name: p.Name,
		Metal: p.Metal, Crystal: p.Crystal, Gas: p.Gas,
		Energy: int(math.Round(prod.Energy * 60)),
		Galaxy: p.Galaxy, System: p.System, Position: p.Position,
		Buildings: buildings, Production: prod, Storage: storage,
	}
}
