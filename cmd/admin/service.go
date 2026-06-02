package main

import (
	"context"
	"errors"
	"fmt"
)

var ErrNotAdmin = errors.New("not an admin")
var ErrPlanetNotFound = errors.New("planet not found")
var ErrUserNotFound = errors.New("user not found")
var ErrInvalidEventType = errors.New("invalid event type")

// maxAdminGrant is the upper bound for a single admin DM or credits grant (#168).
const maxAdminGrant = 1_000_000

type AdminService struct {
	repo Repository
}

func NewAdminService(repo Repository) *AdminService {
	return &AdminService{repo: repo}
}

func (s *AdminService) RequireAdmin(ctx context.Context, playerID int) error {
	ok, err := s.repo.IsAdmin(ctx, playerID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotAdmin
	}
	return nil
}

func (s *AdminService) SearchUsers(ctx context.Context, q string, page, limit int) ([]UserSearchResult, int, error) {
	return s.repo.SearchUsers(ctx, q, page, limit)
}

func (s *AdminService) GetPlanet(ctx context.Context, planetID int) (PlanetView, error) {
	p, err := s.repo.GetPlanet(ctx, planetID)
	if err != nil {
		return PlanetView{}, err
	}

	buildings, err := s.repo.GetPlanetBuildings(ctx, []int{planetID})
	if err != nil {
		return PlanetView{}, err
	}
	p.Buildings = buildings[planetID]

	ships, err := s.repo.GetPlanetShips(ctx, planetID)
	if err != nil {
		return PlanetView{}, err
	}
	p.Ships = ships

	defenses, err := s.repo.GetPlanetDefenses(ctx, planetID)
	if err != nil {
		return PlanetView{}, err
	}
	p.Defenses = defenses

	return p, nil
}

func (s *AdminService) GetPlanetsByUser(ctx context.Context, userID int) ([]PlanetView, error) {
	planetIDs, err := s.repo.GetPlanetsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	allBuildings, err := s.repo.GetPlanetBuildings(ctx, planetIDs)
	if err != nil {
		return nil, err
	}

	var planets []PlanetView
	for _, pid := range planetIDs {
		p, err := s.repo.GetPlanet(ctx, pid)
		if err != nil {
			return nil, err
		}
		p.Buildings = allBuildings[pid]

		ships, err := s.repo.GetPlanetShips(ctx, pid)
		if err != nil {
			return nil, err
		}
		p.Ships = ships

		defenses, err := s.repo.GetPlanetDefenses(ctx, pid)
		if err != nil {
			return nil, err
		}
		p.Defenses = defenses

		planets = append(planets, p)
	}
	return planets, nil
}

func (s *AdminService) OverrideResources(ctx context.Context, planetID, metal, crystal, gas int) error {
	return s.repo.UpdatePlanetResources(ctx, planetID, metal, crystal, gas)
}

func (s *AdminService) GrantDM(ctx context.Context, playerID, amount int, reason string) error {
	if reason == "" {
		return fmt.Errorf("reason is required")
	}
	// Fix #168: enforce upper bound to prevent runaway grants.
	if amount <= 0 || amount > maxAdminGrant {
		return fmt.Errorf("amount must be between 1 and %d", maxAdminGrant)
	}
	return s.repo.AddDM(ctx, playerID, amount)
}

func (s *AdminService) GrantCredits(ctx context.Context, playerID, amount int, reason string) error {
	if reason == "" {
		return fmt.Errorf("reason is required")
	}
	// Fix #168: enforce upper bound to prevent runaway grants.
	if amount <= 0 || amount > maxAdminGrant {
		return fmt.Errorf("amount must be between 1 and %d", maxAdminGrant)
	}
	return s.repo.AddCredits(ctx, playerID, amount)
}

func (s *AdminService) BanPlayer(ctx context.Context, playerID int, banned bool, reason string) error {
	if reason == "" {
		return fmt.Errorf("reason is required")
	}
	return s.repo.SetBanned(ctx, playerID, banned)
}

func (s *AdminService) SendGMMessage(ctx context.Context, playerID int, subject, message string) error {
	if subject == "" || message == "" {
		return fmt.Errorf("subject and message are required")
	}
	return s.repo.CreateNotification(ctx, playerID, "system", subject, message)
}

// validEventTypes is the authoritative list of allowed event types (#170).
var validEventTypes = map[string]bool{
	"bonus_resources":   true,
	"double_production": true,
	"tournament":        true,
	"special":           true,
}

func (s *AdminService) CreateEvent(ctx context.Context, req EventCreateRequest) error {
	if req.Name == "" || req.EventType == "" {
		return fmt.Errorf("name and event_type are required")
	}
	// Fix #170: validate event type against the known set.
	if !validEventTypes[req.EventType] {
		return fmt.Errorf("%w: %s", ErrInvalidEventType, req.EventType)
	}
	if req.StartsAt.IsZero() || req.EndsAt.IsZero() {
		return fmt.Errorf("starts_at and ends_at are required")
	}
	if req.EndsAt.Before(req.StartsAt) {
		return fmt.Errorf("ends_at must be after starts_at")
	}
	if req.Modifiers == nil {
		req.Modifiers = map[string]any{}
	}
	return s.repo.CreateEvent(ctx, req.Name, req.Description, req.EventType, req.Modifiers, req.StartsAt, req.EndsAt)
}
