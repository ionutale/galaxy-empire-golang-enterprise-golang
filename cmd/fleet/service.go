package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
