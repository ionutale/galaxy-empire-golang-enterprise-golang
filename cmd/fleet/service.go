package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

type FleetService struct {
	repo          Repository
	planetBaseURL string
	httpClient    *http.Client
}

func NewFleetService(repo Repository, planetBaseURL string) *FleetService {
	return &FleetService{
		repo:          repo,
		planetBaseURL: planetBaseURL,
		httpClient:    &http.Client{Timeout: 10 * time.Second},
	}
}

var validMissions = map[string]bool{
	"attack": true, "acs_attack": true, "acs_defend": true,
	"transport": true, "deploy": true, "espionage": true,
	"colonize": true, "expedition": true, "recycle": true,
}

func (s *FleetService) DispatchFleet(ctx context.Context, playerID int, req DispatchRequest) (Fleet, error) {
	if !validMissions[req.Mission] {
		return Fleet{}, fmt.Errorf("invalid mission: %s", req.Mission)
	}
	if len(req.Ships) == 0 {
		return Fleet{}, fmt.Errorf("no ships selected")
	}
	if req.SpeedPct < 10 || req.SpeedPct > 100 {
		return Fleet{}, fmt.Errorf("speed must be 10-100")
	}

	for shipType := range req.Ships {
		if _, ok := shipConfig(shipType); !ok {
			return Fleet{}, fmt.Errorf("unknown ship: %s", shipType)
		}
	}

	minSpd, onlyBomber := minShipSpeed(req.Ships)
	if onlyBomber {
		return Fleet{}, fmt.Errorf("bombers cannot fly alone (fuel exceeds cargo)")
	}
	if minSpd == 0 {
		return Fleet{}, fmt.Errorf("no ships with positive speed")
	}

	originCoords, err := s.getPlanetCoords(ctx, req.OriginPlanetID)
	if err != nil {
		return Fleet{}, err
	}

	if err := s.deductShips(ctx, req.OriginPlanetID, req.Ships); err != nil {
		return Fleet{}, err
	}

	driveTechs, err := s.getPlayerTechLevels(ctx, playerID)
	if err != nil {
		slog.Warn("failed to fetch drive techs", "error", err)
		driveTechs = map[string]int{}
	}

	if eff := effectiveMinSpeed(req.Ships, driveTechs); eff > 0 {
		minSpd = eff
	}

	dist := distance(originCoords.galaxy, originCoords.system, originCoords.position, req.TargetGalaxy, req.TargetSystem, req.TargetPosition)

	travelSeconds := float64(dist) / float64(minSpd) * float64(100) / float64(req.SpeedPct) * 3600
	if travelSeconds < 1 {
		travelSeconds = 1
	}
	travelDuration := time.Duration(travelSeconds) * time.Second

	var totalFuel int
	for shipType, qty := range req.Ships {
		if cfg, ok := shipConfig(shipType); ok {
			totalFuel += qty * cfg.Fuel
		}
	}
	distanceFactor := float64(dist) / 35000
	if distanceFactor < 1 {
		distanceFactor = 1
	}
	speedFactor := 1.0 + float64(100-req.SpeedPct)/100.0
	fuelCost := int(float64(totalFuel) * distanceFactor * speedFactor)

	if err := s.deductFuel(ctx, req.OriginPlanetID, fuelCost); err != nil {
		return Fleet{}, fmt.Errorf("fuel deduction: %w", err)
	}

	if err := s.CheckFleetSlotLimit(ctx, playerID); err != nil {
		return Fleet{}, err
	}

	now := time.Now()
	fleet := Fleet{
		PlayerID:       playerID,
		OriginPlanetID: req.OriginPlanetID,
		TargetGalaxy:   req.TargetGalaxy,
		TargetSystem:   req.TargetSystem,
		TargetPosition: req.TargetPosition,
		Mission:        req.Mission,
		Status:         "in_transit",
		SpeedPct:       req.SpeedPct,
		Ships:          req.Ships,
		ArrivesAt:      now.Add(travelDuration),
	}

	return s.repo.CreateFleet(ctx, fleet)
}

func (s *FleetService) RecallFleet(ctx context.Context, playerID, fleetID int) (Fleet, error) {
	fleet, err := s.repo.GetFleetByID(ctx, fleetID)
	if err != nil {
		return Fleet{}, fmt.Errorf("fleet not found")
	}
	if fleet.PlayerID != playerID {
		return Fleet{}, fmt.Errorf("not your fleet")
	}
	if fleet.Status != "in_transit" && fleet.Status != "stationed" {
		return Fleet{}, fmt.Errorf("fleet cannot be recalled")
	}

	now := time.Now()
	remaining := fleet.ArrivesAt.Sub(now)
	if remaining < 0 {
		remaining = 0
	}
	returnTime := remaining * 2
	if returnTime < 10*time.Second {
		returnTime = 10 * time.Second
	}
	fleet.ArrivesAt = now.Add(returnTime)

	if err := s.repo.SetFleetReturning(ctx, fleetID, fleet.ArrivesAt); err != nil {
		return Fleet{}, err
	}
	fleet.Status = "returning"
	return fleet, nil
}

func (s *FleetService) SplitFleet(ctx context.Context, playerID, fleetID int, ships map[string]int) (Fleet, error) {
	fleet, err := s.repo.GetFleetByID(ctx, fleetID)
	if err != nil {
		return Fleet{}, fmt.Errorf("fleet not found")
	}
	if fleet.PlayerID != playerID {
		return Fleet{}, fmt.Errorf("not your fleet")
	}
	if fleet.Status != "stationed" {
		return Fleet{}, fmt.Errorf("only stationed fleets can be split")
	}
	if len(ships) == 0 {
		return Fleet{}, fmt.Errorf("no ships selected")
	}

	for shipType, qty := range ships {
		if qty <= 0 {
			continue
		}
		if fleet.Ships[shipType] < qty {
			return Fleet{}, fmt.Errorf("not enough %s ships", shipType)
		}
		fleet.Ships[shipType] -= qty
		if fleet.Ships[shipType] == 0 {
			delete(fleet.Ships, shipType)
		}
	}

	if err := s.repo.UpdateFleetShips(ctx, fleetID, fleet.Ships); err != nil {
		return Fleet{}, err
	}

	newFleet := Fleet{
		PlayerID:       playerID,
		OriginPlanetID: fleet.OriginPlanetID,
		TargetGalaxy:   fleet.TargetGalaxy,
		TargetSystem:   fleet.TargetSystem,
		TargetPosition: fleet.TargetPosition,
		Mission:        fleet.Mission,
		Status:         "stationed",
		SpeedPct:       fleet.SpeedPct,
		Ships:          ships,
	}

	return s.repo.CreateFleet(ctx, newFleet)
}

func (s *FleetService) MergeFleets(ctx context.Context, playerID int, fleetIDs []int) (Fleet, error) {
	if len(fleetIDs) < 2 {
		return Fleet{}, fmt.Errorf("need at least 2 fleets to merge")
	}

	var targetFleet Fleet
	mergedShips := make(map[string]int)

	for i, fid := range fleetIDs {
		fleet, err := s.repo.GetFleetByID(ctx, fid)
		if err != nil {
			return Fleet{}, fmt.Errorf("fleet %d not found", fid)
		}
		if fleet.PlayerID != playerID {
			return Fleet{}, fmt.Errorf("fleet %d is not yours", fid)
		}
		if fleet.Status != "stationed" {
			return Fleet{}, fmt.Errorf("fleet %d must be stationed", fid)
		}

		if i == 0 {
			targetFleet = fleet
		} else {
			if fleet.TargetGalaxy != targetFleet.TargetGalaxy ||
				fleet.TargetSystem != targetFleet.TargetSystem ||
				fleet.TargetPosition != targetFleet.TargetPosition {
				return Fleet{}, fmt.Errorf("fleets must be at the same coordinates")
			}
		}

		for shipType, qty := range fleet.Ships {
			mergedShips[shipType] += qty
		}

		if i > 0 {
			if err := s.repo.DeleteFleet(ctx, fid); err != nil {
				return Fleet{}, err
			}
		}
	}

	if err := s.repo.UpdateFleetShips(ctx, targetFleet.ID, mergedShips); err != nil {
		return Fleet{}, err
	}
	targetFleet.Ships = mergedShips
	return targetFleet, nil
}

func (s *FleetService) CheckFleetSlotLimit(ctx context.Context, playerID int) error {
	count, err := s.repo.CountPlayerFleets(ctx, playerID)
	if err != nil {
		return err
	}
	techs, err := s.getPlayerTechLevels(ctx, playerID)
	if err != nil {
		techs = map[string]int{}
	}
	computerLevel := techs["computer_tech"]
	limit := 1 + computerLevel*2
	if count >= limit {
		return fmt.Errorf("fleet slot limit reached (%d/%d)", count, limit)
	}
	return nil
}

type planetCoords struct {
	galaxy, system, position int
}

func (s *FleetService) getPlanetCoords(ctx context.Context, planetID int) (planetCoords, error) {
	body, _ := json.Marshal(map[string]int{"planet_id": planetID})
	resp, err := s.httpClient.Post(s.planetBaseURL+"/internal/planet/coords", "application/json", bytes.NewReader(body))
	if err != nil {
		return planetCoords{}, fmt.Errorf("planet service unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return planetCoords{}, fmt.Errorf("planet service: %s", string(respBody))
	}
	var coords struct {
		Galaxy   int `json:"galaxy"`
		System   int `json:"system"`
		Position int `json:"position"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&coords); err != nil {
		return planetCoords{}, fmt.Errorf("parse coords: %w", err)
	}
	return planetCoords{coords.Galaxy, coords.System, coords.Position}, nil
}

func (s *FleetService) getPlayerTechLevels(ctx context.Context, playerID int) (map[string]int, error) {
	body, _ := json.Marshal(map[string]int{"player_id": playerID})
	resp, err := s.httpClient.Post(s.planetBaseURL+"/internal/player/techs", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("planet service unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("planet service: %s", string(respBody))
	}
	var result struct {
		Technologies map[string]int `json:"technologies"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("parse techs: %w", err)
	}
	return result.Technologies, nil
}

func (s *FleetService) deductShips(ctx context.Context, planetID int, ships map[string]int) error {
	body, _ := json.Marshal(map[string]any{
		"planet_id": planetID,
		"ships":     ships,
	})
	resp, err := s.httpClient.Post(s.planetBaseURL+"/internal/ships/deduct", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("planet service unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("planet service: %s", string(respBody))
	}
	return nil
}

func (s *FleetService) deductFuel(ctx context.Context, planetID, amount int) error {
	body, _ := json.Marshal(map[string]any{
		"planet_id": planetID,
		"resource":  "gas",
		"amount":    amount,
	})
	resp, err := s.httpClient.Post(s.planetBaseURL+"/internal/resources/deduct", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("planet service unreachable: %w", err)
	}
	resp.Body.Close()
	return nil
}
