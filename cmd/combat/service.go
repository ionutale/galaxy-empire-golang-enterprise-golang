package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"math/rand"
	"net/http"
	"time"
)

type CombatService struct {
	repo           Repository
	planetBaseURL  string
	httpClient     *http.Client
}

func NewCombatService(repo Repository, planetBaseURL string) *CombatService {
	return &CombatService{
		repo:          repo,
		planetBaseURL: planetBaseURL,
		httpClient:    &http.Client{Timeout: 15 * time.Second},
	}
}

type resolveRequest struct {
	FleetID       int            `json:"fleet_id"`
	AttackerID    int            `json:"attacker_player_id"`
	OriginPlanet  int            `json:"attacker_origin_planet_id"`
	AttackerShips map[string]int `json:"attacker_ships"`
	TargetGalaxy  int            `json:"target_galaxy"`
	TargetSystem  int            `json:"target_system"`
	TargetPos     int            `json:"target_position"`
}

type resolveResponse struct {
	ReportID      int            `json:"report_id"`
	AttackerWon   bool           `json:"attacker_won"`
	Rounds        int            `json:"rounds"`
	AttackerLoot  map[string]int `json:"attacker_loot,omitempty"`
	DebrisMetal   int            `json:"debris_metal"`
	DebrisCrystal int            `json:"debris_crystal"`
	Moon          MoonInfo       `json:"moon"`
}

type planetInfoResponse struct {
	PlanetID int            `json:"planet_id"`
	PlayerID int            `json:"player_id"`
	Metal    int            `json:"metal"`
	Crystal  int            `json:"crystal"`
	Gas      int            `json:"gas"`
	Ships    map[string]int `json:"ships"`
}

func (s *CombatService) GetReport(ctx context.Context, id int) (CombatReport, error) {
	return s.repo.GetCombatReport(ctx, id)
}

func (s *CombatService) ListPlayerReports(ctx context.Context, playerID int) ([]CombatReport, error) {
	return s.repo.ListPlayerCombatReports(ctx, playerID)
}

func (s *CombatService) Resolve(ctx context.Context, req resolveRequest) (resolveResponse, error) {
	defInfo, err := s.fetchDefenderInfo(ctx, req.TargetGalaxy, req.TargetSystem, req.TargetPos)
	if err != nil {
		return resolveResponse{}, fmt.Errorf("fetch defender info: %w", err)
	}

	attackerShips := req.AttackerShips
	defenderShips := defInfo.Ships

	if attackerShips == nil {
		attackerShips = make(map[string]int)
	}
	if defenderShips == nil {
		defenderShips = make(map[string]int)
	}

	result := ResolveCombat(attackerShips, defenderShips, shipStatsMap)

	if result.AttackerWon && !isEmpty(result.AttackerShipsAfter) {
		totalCargo := totalCargoCapacity(result.AttackerShipsAfter)
		result.AttackerLoot = CalculateLoot(defInfo.Metal, defInfo.Crystal, defInfo.Gas, totalCargo)
		result.DefenderLostRes = CalculateDefenderLostResources(defInfo.Metal, defInfo.Crystal, defInfo.Gas, result.AttackerLoot)
	}

	moonInfo := s.tryCreateMoon(ctx, req.TargetGalaxy, req.TargetSystem, req.TargetPos, &result)

	reportID, err := s.repo.CreateCombatReport(ctx, CombatReport{
		AttackerPlayerID:    req.AttackerID,
		DefenderPlayerID:    defInfo.PlayerID,
		TargetGalaxy:        req.TargetGalaxy,
		TargetSystem:        req.TargetSystem,
		TargetPosition:      req.TargetPos,
		AttackerShipsBefore: attackerShips,
		DefenderShipsBefore: defenderShips,
		AttackerShipsAfter:  result.AttackerShipsAfter,
		DefenderShipsAfter:  result.DefenderShipsAfter,
		Rounds:              result.Rounds,
		AttackerWon:         result.AttackerWon,
		AttackerLoot:        result.AttackerLoot,
		DefenderLostRes:     result.DefenderLostRes,
		DebrisMetal:         result.DebrisMetal,
		DebrisCrystal:       result.DebrisCrystal,
		MoonCreated:         result.MoonCreated,
		MoonSize:            result.MoonSize,
	})
	if err != nil {
		return resolveResponse{}, fmt.Errorf("save report: %w", err)
	}

	if !isEmpty(result.DefenderShipsAfter) {
		if err := s.deductDefenderShips(ctx, defInfo.PlanetID, diffShips(defenderShips, result.DefenderShipsAfter)); err != nil {
			slog.Error("deduct defender ships failed", "error", err)
		}
	}

	if result.AttackerWon && !isEmpty(result.AttackerLoot) {
		for res, amt := range result.AttackerLoot {
			if amt > 0 {
				if err := s.addLootToAttacker(ctx, req.OriginPlanet, res, amt); err != nil {
					slog.Error("add loot failed", "resource", res, "error", err)
				}
			}
		}

		if defInfo.PlanetID > 0 {
			for res, amt := range result.DefenderLostRes {
				if amt > 0 {
					if err := s.deductDefenderResources(ctx, defInfo.PlanetID, res, amt); err != nil {
						slog.Error("deduct defender resources failed", "resource", res, "error", err)
					}
				}
			}
		}
	}

	return resolveResponse{
		ReportID:      reportID,
		AttackerWon:   result.AttackerWon,
		Rounds:        len(result.Rounds),
		AttackerLoot:  result.AttackerLoot,
		DebrisMetal:   result.DebrisMetal,
		DebrisCrystal: result.DebrisCrystal,
		Moon:          moonInfo,
	}, nil
}

func (s *CombatService) tryCreateMoon(ctx context.Context, galaxy, system, position int, result *CombatResult) MoonInfo {
	totalDebris := result.DebrisMetal + result.DebrisCrystal
	if totalDebris < 200000 {
		return MoonInfo{Created: false}
	}

	chance := 20.0 + float64(totalDebris-200000)/400000.0
	chance = math.Min(chance, 20.0)

	if rand.Float64()*100 >= chance {
		return MoonInfo{Created: false}
	}

	size := gaussianMoonSize()
	name := fmt.Sprintf("Moon [%d:%d:%d]", galaxy, system, position)

	if err := s.repo.CreateMoon(ctx, galaxy, system, position, name, size); err != nil {
		slog.Error("create moon failed", "error", err)
		return MoonInfo{Created: false}
	}

	result.MoonCreated = true
	result.MoonSize = size

	return MoonInfo{Created: true, Size: size, Name: name}
}

func gaussianMoonSize() int {
	r := rand.Float64() + rand.Float64() + rand.Float64() + rand.Float64()
	r = r / 4.0
	size := int(20 + r*30)
	if size < 20 {
		return 20
	}
	if size > 50 {
		return 50
	}
	return size
}

func (s *CombatService) MissileStrike(ctx context.Context, req MissileStrikeRequest) (MissileStrikeResult, error) {
	defenses, err := s.fetchDefenderDefenses(ctx, req.TargetPlanetID)
	if err != nil {
		return MissileStrikeResult{}, fmt.Errorf("fetch defender defenses: %w", err)
	}

	req.Defenses = defenses
	result := ResolveMissileStrike(req)

	if result.ABMsUsed > 0 {
		if err := s.deductDefenderABMs(ctx, req.TargetPlanetID, result.ABMsUsed); err != nil {
			slog.Error("deduct defender ABMs failed", "error", err)
		}
	}

	if len(result.DefensesDestroyed) > 0 {
		if err := s.deductDefenderDefenses(ctx, req.TargetPlanetID, result.DefensesDestroyed); err != nil {
			slog.Error("deduct defenses after missile strike failed", "error", err)
		}
	}

	return result, nil
}

func (s *CombatService) GetMoonInfo(ctx context.Context, galaxy, system, position int) (*Moon, error) {
	return s.repo.GetMoon(ctx, galaxy, system, position)
}

func (s *CombatService) fetchDefenderInfo(ctx context.Context, galaxy, system, position int) (planetInfoResponse, error) {
	body, _ := json.Marshal(map[string]int{
		"galaxy": galaxy, "system": system, "position": position,
	})
	resp, err := s.httpClient.Post(s.planetBaseURL+"/internal/planet/info", "application/json", bytes.NewReader(body))
	if err != nil {
		return planetInfoResponse{}, fmt.Errorf("planet service unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return planetInfoResponse{}, fmt.Errorf("planet service: %s", string(respBody))
	}
	var info planetInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return planetInfoResponse{}, fmt.Errorf("parse response: %w", err)
	}
	return info, nil
}

func (s *CombatService) deductDefenderShips(ctx context.Context, planetID int, ships map[string]int) error {
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
		return fmt.Errorf("planet service deduct ships: %d", resp.StatusCode)
	}
	return nil
}

func (s *CombatService) addLootToAttacker(ctx context.Context, planetID int, resource string, amount int) error {
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
		return fmt.Errorf("planet service add resource: %d", resp.StatusCode)
	}
	return nil
}

func (s *CombatService) deductDefenderResources(ctx context.Context, planetID int, resource string, amount int) error {
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
		return fmt.Errorf("planet service deduct resource: %d", resp.StatusCode)
	}
	return nil
}

func (s *CombatService) fetchDefenderDefenses(ctx context.Context, planetID int) (map[string]int, error) {
	body, _ := json.Marshal(map[string]int{
		"planet_id": planetID,
	})
	resp, err := s.httpClient.Post(s.planetBaseURL+"/internal/defense/list", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("planet service unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("planet service defense list: %d", resp.StatusCode)
	}

	var result struct {
		Defenses map[string]int `json:"defenses"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return result.Defenses, nil
}

func (s *CombatService) deductDefenderABMs(ctx context.Context, planetID, count int) error {
	body, _ := json.Marshal(map[string]any{
		"planet_id": planetID,
		"count":     count,
	})
	resp, err := s.httpClient.Post(s.planetBaseURL+"/internal/defense/deduct-abms", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("planet service unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("planet service deduct ABMs: %d", resp.StatusCode)
	}
	return nil
}

func (s *CombatService) deductDefenderDefenses(ctx context.Context, planetID int, losses map[string]int) error {
	body, _ := json.Marshal(map[string]any{
		"planet_id":      planetID,
		"defense_losses": losses,
	})
	resp, err := s.httpClient.Post(s.planetBaseURL+"/internal/defense/deduct", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("planet service unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("planet service deduct defenses: %d", resp.StatusCode)
	}
	return nil
}

func diffShips(before, after map[string]int) map[string]int {
	diff := make(map[string]int)
	for shipType, beforeQty := range before {
		afterQty := after[shipType]
		if lost := beforeQty - afterQty; lost > 0 {
			diff[shipType] = lost
		}
	}
	return diff
}
