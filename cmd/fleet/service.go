package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"time"
)

type FleetService struct {
	repo           Repository
	planetBaseURL  string
	combatBaseURL  string
	authBaseURL    string
	internalSecret string
	httpClient     *http.Client
}

func NewFleetService(repo Repository, planetBaseURL string, combatBaseURL string, authBaseURL string, internalSecret string) *FleetService {
	if combatBaseURL == "" {
		combatBaseURL = "http://localhost:8084"
	}
	if authBaseURL == "" {
		authBaseURL = "http://localhost:8081"
	}
	return &FleetService{
		repo:           repo,
		planetBaseURL:  planetBaseURL,
		combatBaseURL:  combatBaseURL,
		authBaseURL:    authBaseURL,
		internalSecret: internalSecret,
		httpClient:     &http.Client{Timeout: 10 * time.Second},
	}
}

var validMissions = map[string]bool{
	"attack": true, "acs_attack": true, "acs_defend": true,
	"transport": true, "deploy": true, "espionage": true,
	"colonize": true, "expedition": true, "recycle": true,
	"stargate": true,
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

	if req.Mission == "stargate" {
		hasLink, err := s.checkStarGateLink(ctx, req.OriginPlanetID, req.TargetGalaxy, req.TargetSystem, req.TargetPosition)
		if err != nil {
			return Fleet{}, fmt.Errorf("stargate check: %w", err)
		}
		if !hasLink {
			return Fleet{}, fmt.Errorf("no stargate link to target coordinates")
		}
	}

	originCoords, err := s.getPlanetCoords(ctx, req.OriginPlanetID)
	if err != nil {
		return Fleet{}, err
	}

	// Verify the origin planet belongs to the requesting player
	ownerID, err := s.getTargetPlayerID(ctx, originCoords.galaxy, originCoords.system, originCoords.position)
	if err == nil && ownerID != playerID {
		return Fleet{}, fmt.Errorf("origin planet does not belong to you")
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
	const maxTravelDuration = 30 * 24 * time.Hour
	if travelDuration > maxTravelDuration {
		travelDuration = maxTravelDuration
	}

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
	fuelCostF := float64(totalFuel) * distanceFactor * speedFactor
	if fuelCostF > float64(math.MaxInt32) {
		fuelCostF = float64(math.MaxInt32)
	}
	fuelCost := int(fuelCostF)

	if req.Mission != "stargate" {
		if err := s.deductFuel(ctx, req.OriginPlanetID, fuelCost); err != nil {
			return Fleet{}, fmt.Errorf("fuel deduction: %w", err)
		}
	}

	if err := s.CheckFleetSlotLimit(ctx, playerID); err != nil {
		return Fleet{}, err
	}

	if req.Mission == "attack" || req.Mission == "acs_attack" {
		if err := s.CheckAttackCooldown(ctx, playerID, req.TargetGalaxy, req.TargetSystem, req.TargetPosition); err != nil {
			return Fleet{}, err
		}
		targetPlayerID, err := s.getTargetPlayerID(ctx, req.TargetGalaxy, req.TargetSystem, req.TargetPosition)
		if err == nil {
			inVacation, vErr := s.checkTargetVacationMode(ctx, targetPlayerID)
			if vErr == nil && inVacation {
				return Fleet{}, fmt.Errorf("target player is in vacation mode")
			}
		}
	}

	if req.Mission == "transport" {
		if req.CargoMetal < 0 || req.CargoCrystal < 0 || req.CargoGas < 0 {
			return Fleet{}, fmt.Errorf("cargo amounts cannot be negative")
		}
		if req.CargoMetal > 0 {
			if err := s.deductResource(ctx, req.OriginPlanetID, "metal", req.CargoMetal); err != nil {
				return Fleet{}, fmt.Errorf("cargo metal deduction: %w", err)
			}
		}
		if req.CargoCrystal > 0 {
			if err := s.deductResource(ctx, req.OriginPlanetID, "crystal", req.CargoCrystal); err != nil {
				return Fleet{}, fmt.Errorf("cargo crystal deduction: %w", err)
			}
		}
		if req.CargoGas > 0 {
			if err := s.deductResource(ctx, req.OriginPlanetID, "gas", req.CargoGas); err != nil {
				return Fleet{}, fmt.Errorf("cargo gas deduction: %w", err)
			}
		}
	}

	now := time.Now()
	fleet := Fleet{
		PlayerID:        playerID,
		OriginPlanetID:  req.OriginPlanetID,
		TargetGalaxy:    req.TargetGalaxy,
		TargetSystem:    req.TargetSystem,
		TargetPosition:  req.TargetPosition,
		Mission:         req.Mission,
		Status:          "in_transit",
		SpeedPct:        req.SpeedPct,
		Ships:           req.Ships,
		ArrivesAt:       now.Add(travelDuration),
		AllianceGroupID: req.AllianceGroupID,
		CargoMetal:      req.CargoMetal,
		CargoCrystal:    req.CargoCrystal,
		CargoGas:        req.CargoGas,
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
		if qty < 0 {
			return Fleet{}, fmt.Errorf("split quantity cannot be negative")
		}
		if qty == 0 {
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

	totalRemaining := 0
	for _, q := range fleet.Ships {
		totalRemaining += q
	}
	if totalRemaining == 0 {
		return Fleet{}, fmt.Errorf("cannot split: source fleet would be empty")
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

func (s *FleetService) getTargetPlayerID(ctx context.Context, galaxy, system, position int) (int, error) {
	body, _ := json.Marshal(map[string]int{
		"galaxy": galaxy, "system": system, "position": position,
	})
	resp, err := s.httpClient.Post(s.planetBaseURL+"/internal/planet/info", "application/json", bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("planet service unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("planet not found at coordinates")
	}
	var result struct {
		PlayerID int `json:"player_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("parse response: %w", err)
	}
	return result.PlayerID, nil
}

func (s *FleetService) checkTargetVacationMode(ctx context.Context, targetPlayerID int) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s/api/auth/user/%d/vacation-status", s.authBaseURL, targetPlayerID), nil)
	if err != nil {
		return false, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("X-Internal-Secret", s.internalSecret)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("auth service unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("auth service: status %d", resp.StatusCode)
	}
	var result struct {
		VacationModeEnabled bool `json:"vacation_mode_enabled"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("parse response: %w", err)
	}
	return result.VacationModeEnabled, nil
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
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("fuel deduction failed (status %d): %s", resp.StatusCode, string(body))
	}
	return nil
}

func (s *FleetService) deductResource(ctx context.Context, planetID int, resource string, amount int) error {
	if amount <= 0 {
		return nil
	}
	body, _ := json.Marshal(map[string]any{
		"planet_id": planetID,
		"resource":  resource,
		"amount":    amount,
	})
	resp, err := s.httpClient.Post(s.planetBaseURL+"/internal/resources/deduct", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("planet service unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("resource deduction failed (status %d): %s", resp.StatusCode, string(b))
	}
	return nil
}

func (s *FleetService) AddResourcesToPlanet(ctx context.Context, planetID int, resource string, amount int) error {
	body, _ := json.Marshal(map[string]any{
		"planet_id": planetID,
		"resource":  resource,
		"amount":    amount,
	})
	resp, err := s.httpClient.Post(s.planetBaseURL+"/internal/resources/add", "application/json", bytes.NewReader(body))
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

func (s *FleetService) AddShipsToPlanet(ctx context.Context, planetID int, ships map[string]int) error {
	body, _ := json.Marshal(map[string]any{
		"planet_id": planetID,
		"ships":     ships,
	})
	resp, err := s.httpClient.Post(s.planetBaseURL+"/internal/ships/add", "application/json", bytes.NewReader(body))
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

func (s *FleetService) checkStarGateLink(ctx context.Context, originPlanetID, targetGalaxy, targetSystem, targetPosition int) (bool, error) {
	targetPlanetID, err := s.FindTargetPlanet(ctx, targetGalaxy, targetSystem, targetPosition)
	if err != nil {
		return false, fmt.Errorf("target planet not found: %w", err)
	}

	body, _ := json.Marshal(map[string]any{
		"origin_planet_id": originPlanetID,
		"target_planet_id": targetPlanetID,
	})
	resp, err := s.httpClient.Post(s.planetBaseURL+"/internal/stargate/check-link", "application/json", bytes.NewReader(body))
	if err != nil {
		return false, fmt.Errorf("planet service unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("planet service: %s", string(respBody))
	}
	var result struct {
		HasLink bool `json:"has_link"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("parse response: %w", err)
	}
	return result.HasLink, nil
}

func (s *FleetService) FindTargetPlanet(ctx context.Context, galaxy, system, position int) (int, error) {
	body, _ := json.Marshal(map[string]int{
		"galaxy":   galaxy,
		"system":   system,
		"position": position,
	})
	resp, err := s.httpClient.Post(s.planetBaseURL+"/internal/planet/by-coords", "application/json", bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("planet service unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("no planet at those coordinates")
	}
	var result struct {
		PlanetID int `json:"planet_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("parse response: %w", err)
	}
	return result.PlanetID, nil
}

type combatResolveResp struct {
	ReportID     int            `json:"report_id"`
	AttackerWon  bool           `json:"attacker_won"`
	Rounds       int            `json:"rounds"`
	AttackerLoot map[string]int `json:"attacker_loot,omitempty"`
	DebrisMetal  int            `json:"debris_metal"`
	DebrisCrystal int           `json:"debris_crystal"`
}

func (s *FleetService) resolveCombatForArrival(ctx context.Context, f Fleet, extraAttackerShips map[string]int) error {
	defendFleets, err := s.repo.GetACSDefendFleets(ctx, f.TargetGalaxy, f.TargetSystem, f.TargetPosition)
	if err != nil {
		slog.Warn("failed to get ACS defend fleets", "error", err)
	}

	defenderShips := make(map[string]int)
	for _, df := range defendFleets {
		for shipType, qty := range df.Ships {
			defenderShips[shipType] += qty
		}
	}

	attackerShips := make(map[string]int)
	for k, v := range f.Ships {
		attackerShips[k] += v
	}
	for k, v := range extraAttackerShips {
		attackerShips[k] += v
	}

	body, _ := json.Marshal(map[string]any{
		"fleet_id":                f.ID,
		"attacker_player_id":      f.PlayerID,
		"attacker_origin_planet_id": f.OriginPlanetID,
		"attacker_ships":          attackerShips,
		"target_galaxy":           f.TargetGalaxy,
		"target_system":           f.TargetSystem,
		"target_position":         f.TargetPosition,
		"defender_ships":          defenderShips,
	})
	resp, err := s.httpClient.Post(s.combatBaseURL+"/combat/resolve", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("combat service unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("combat service: %s", string(respBody))
	}
	var cr combatResolveResp
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return fmt.Errorf("parse combat response: %w", err)
	}
	slog.Info("combat resolved",
		"fleet", f.ID,
		"report", cr.ReportID,
		"attacker_won", cr.AttackerWon,
		"rounds", cr.Rounds,
		"debris_metal", cr.DebrisMetal,
		"debris_crystal", cr.DebrisCrystal,
	)

	if cr.DebrisMetal > 0 || cr.DebrisCrystal > 0 {
		if err := s.repo.UpsertDebrisField(ctx, f.TargetGalaxy, f.TargetSystem, f.TargetPosition, cr.DebrisMetal, cr.DebrisCrystal); err != nil {
			slog.Error("upsert debris field", "error", err)
		}
	}

	return nil
}

func (s *FleetService) AutoReturnIfVacationMode(ctx context.Context, f Fleet) (bool, error) {
	targetPlayerID, err := s.getTargetPlayerID(ctx, f.TargetGalaxy, f.TargetSystem, f.TargetPosition)
	if err != nil {
		return false, fmt.Errorf("get target player: %w", err)
	}
	inVacation, err := s.checkTargetVacationMode(ctx, targetPlayerID)
	if err != nil {
		return false, fmt.Errorf("check vacation mode: %w", err)
	}
	if !inVacation {
		return false, nil
	}
	forwardDuration := time.Since(f.CreatedAt)
	returnArrivesAt := time.Now().Add(forwardDuration)
	if err := s.repo.SetFleetReturning(ctx, f.ID, returnArrivesAt); err != nil {
		return false, fmt.Errorf("set returning: %w", err)
	}
	slog.Info("fleet auto-returned due to target vacation mode", "fleet", f.ID)
	return true, nil
}

func (s *FleetService) CheckAttackCooldown(ctx context.Context, attackerID, targetGalaxy, targetSystem, targetPosition int) error {
	lastAttack, err := s.repo.GetAttackCooldown(ctx, attackerID, targetGalaxy, targetSystem, targetPosition)
	if err != nil {
		return err
	}
	if lastAttack != nil {
		elapsed := time.Since(*lastAttack)
		cooldown := 2 * time.Hour
		if elapsed < cooldown {
			remaining := cooldown - elapsed
			minutes := int(remaining.Minutes())
			return fmt.Errorf("attack cooldown active, remaining %d minutes", minutes)
		}
	}
	return nil
}

func (s *FleetService) allACSFleetsArrived(ctx context.Context, allianceGroupID int) (bool, error) {
	fleets, err := s.repo.GetACSGroupFleets(ctx, allianceGroupID)
	if err != nil {
		return false, err
	}
	if len(fleets) == 0 {
		return false, nil
	}
	for _, f := range fleets {
		if f.Status == "in_transit" || f.Status == "returning" {
			return false, nil
		}
	}
	return true, nil
}

func (s *FleetService) harvestDebris(ctx context.Context, fleetID int, fleetShips map[string]int, originPlanetID, targetGalaxy, targetSystem, targetPosition int) error {
	recyclerQty := fleetShips["recycler"]
	if recyclerQty == 0 {
		return fmt.Errorf("no recyclers in fleet")
	}

	cargoCapacity := recyclerQty * 20000

	debris, err := s.repo.GetDebrisField(ctx, targetGalaxy, targetSystem, targetPosition)
	if err != nil {
		return fmt.Errorf("get debris field: %w", err)
	}
	if debris == nil {
		slog.Warn("no debris field at target", "galaxy", targetGalaxy, "system", targetSystem, "position", targetPosition)
		return nil
	}

	totalAvailable := debris.Metal + debris.Crystal
	if totalAvailable == 0 {
		return nil
	}

	toHarvest := totalAvailable
	if toHarvest > cargoCapacity {
		toHarvest = cargoCapacity
	}

	// Use int64 intermediate values to prevent integer overflow when Metal and
	// toHarvest are both large numbers.
	harvestMetal := int(int64(debris.Metal) * int64(toHarvest) / int64(totalAvailable))
	harvestCrystal := toHarvest - harvestMetal

	if harvestMetal > debris.Metal {
		harvestMetal = debris.Metal
	}
	if harvestCrystal > debris.Crystal {
		harvestCrystal = debris.Crystal
	}

	newMetal := debris.Metal - harvestMetal
	newCrystal := debris.Crystal - harvestCrystal
	if err := s.repo.UpdateDebrisField(ctx, targetGalaxy, targetSystem, targetPosition, newMetal, newCrystal); err != nil {
		return fmt.Errorf("update debris field: %w", err)
	}

	if harvestMetal > 0 {
		if err := s.AddResourcesToPlanet(ctx, originPlanetID, "metal", harvestMetal); err != nil {
			return fmt.Errorf("add metal to origin: %w", err)
		}
	}
	if harvestCrystal > 0 {
		if err := s.AddResourcesToPlanet(ctx, originPlanetID, "crystal", harvestCrystal); err != nil {
			return fmt.Errorf("add crystal to origin: %w", err)
		}
	}

	slog.Info("debris harvested",
		"fleet", fleetID,
		"metal", harvestMetal,
		"crystal", harvestCrystal,
	)
	return nil
}

func (s *FleetService) handleColonizeArrival(ctx context.Context, f Fleet) error {
	existingID, err := s.FindTargetPlanet(ctx, f.TargetGalaxy, f.TargetSystem, f.TargetPosition)
	if err == nil && existingID > 0 {
		return fmt.Errorf("target position already occupied")
	}

	body, _ := json.Marshal(map[string]any{
		"user_id":  f.PlayerID,
		"galaxy":   f.TargetGalaxy,
		"system":   f.TargetSystem,
		"position": f.TargetPosition,
	})
	resp, err := s.httpClient.Post(s.planetBaseURL+"/internal/planet/create", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("planet service unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("planet service: %s", string(respBody))
	}

	ships := f.Ships
	delete(ships, "colony_ship")
	if err := s.repo.UpdateFleetShips(ctx, f.ID, ships); err != nil {
		return fmt.Errorf("update fleet ships: %w", err)
	}

	return s.repo.MarkFleetArrived(ctx, f.ID)
}
