package main

import (
	"context"
	"errors"
	"math"
	"time"
)

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
		p, b, err := s.repo.Create(ctx, userID)
		return p, b, err
	}

	buildings, err := s.repo.GetBuildings(ctx, planet.ID)
	if err != nil {
		return Planet{}, nil, err
	}

	prod := s.calculateProduction(buildings)
	elapsed := time.Since(planet.ResourcesUpdatedAt).Seconds()
	if elapsed > 0 {
		planet.Metal = planet.Metal + int(prod.Metal*elapsed)
		planet.Crystal = planet.Crystal + int(prod.Crystal*elapsed)
		planet.Gas = planet.Gas + int(prod.Gas*elapsed)
	}

	now := time.Now()
	planet.ResourcesUpdatedAt = now
	if err := s.repo.UpdateResources(ctx, planet.ID, planet.Metal, planet.Crystal, planet.Gas, now); err != nil {
		return Planet{}, nil, err
	}

	return planet, buildings, nil
}

func (s *PlanetService) calculateProduction(buildings []Building) Production {
	levels := make(map[string]int)
	for _, b := range buildings {
		levels[b.Type] = b.Level
	}

	metalRate := productionRate("metal_mine", levels["metal_mine"])
	crystalRate := productionRate("crystal_mine", levels["crystal_mine"])
	gasRate := productionRate("gas_mine", levels["gas_mine"])
	solarRate := productionRate("solar_plant", levels["solar_plant"])

	return Production{
		Metal:   metalRate / 60.0,
		Crystal: crystalRate / 60.0,
		Gas:     gasRate / 60.0,
		Energy:  solarRate / 60.0,
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

func toPlanetResponse(p Planet, buildings []Building, prod Production) PlanetResponse {
	return PlanetResponse{
		ID: p.ID, UserID: p.UserID, Name: p.Name,
		Metal: p.Metal, Crystal: p.Crystal, Gas: p.Gas,
		Energy: int(math.Round(prod.Energy * 60)),
		Galaxy: p.Galaxy, System: p.System, Position: p.Position,
		Buildings: buildings, Production: prod,
	}
}
