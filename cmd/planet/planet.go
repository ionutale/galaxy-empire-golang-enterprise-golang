package main

import (
	"context"
	"errors"
)

type PlanetService struct {
	repo Repository
}

func NewPlanetService(repo Repository) *PlanetService {
	return &PlanetService{repo: repo}
}

func (s *PlanetService) GetOrCreatePlanet(ctx context.Context, userID int) (Planet, error) {
	planet, err := s.repo.FindByUserID(ctx, userID)
	if err == nil {
		return planet, nil
	}
	if !errors.Is(err, ErrPlanetNotFound) {
		return Planet{}, err
	}
	return s.repo.Create(ctx, userID)
}
