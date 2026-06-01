package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

var QuestDefinitions = []QuestDefinition{
	{
		ID: "first_mine", Name: "First Mine", Description: "Build a Metal Mine to level 1",
		Category: CategoryBuilding,
		Requirements: []QuestRequirement{{Type: "building_level", Key: "metal_mine", Value: 1}},
		RewardDM: 5, RewardMetal: 500,
	},
	{
		ID: "power_up", Name: "Power Up", Description: "Build a Solar Plant to level 1",
		Category: CategoryBuilding,
		Requirements: []QuestRequirement{{Type: "building_level", Key: "solar_plant", Value: 1}},
		RewardDM: 3,
	},
	{
		ID: "crystal_clear", Name: "Crystal Clear", Description: "Build a Crystal Mine to level 1",
		Category: CategoryBuilding,
		Requirements: []QuestRequirement{{Type: "building_level", Key: "crystal_mine", Value: 1}},
		RewardDM: 5,
	},
	{
		ID: "gas_up", Name: "Gas Up", Description: "Build a Gas Mine to level 1",
		Category: CategoryBuilding,
		Requirements: []QuestRequirement{{Type: "building_level", Key: "gas_mine", Value: 1}},
		RewardDM: 5,
	},
	{
		ID: "storage_space", Name: "Storage Space", Description: "Build a Metal Storage to level 1",
		Category: CategoryBuilding,
		Requirements: []QuestRequirement{{Type: "building_level", Key: "metal_storage", Value: 1}},
		RewardDM: 3,
	},
	{
		ID: "factory_worker", Name: "Factory Worker", Description: "Build a Robotics Factory to level 1",
		Category: CategoryBuilding,
		Requirements: []QuestRequirement{{Type: "building_level", Key: "robotics_factory", Value: 1}},
		RewardDM: 5,
	},
	{
		ID: "ship_builder", Name: "Ship Builder", Description: "Build a Shipyard to level 1",
		Category: CategoryBuilding,
		Requirements: []QuestRequirement{{Type: "building_level", Key: "shipyard", Value: 1}},
		RewardDM: 8,
	},
	{
		ID: "first_ship", Name: "First Ship", Description: "Build 1 Cargo ship",
		Category: CategoryFleet,
		Requirements: []QuestRequirement{{Type: "ship_count", Key: "cargo", Value: 1}},
		RewardDM: 10,
	},
	{
		ID: "researcher", Name: "Researcher", Description: "Build a Research Lab to level 1",
		Category: CategoryBuilding,
		Requirements: []QuestRequirement{{Type: "building_level", Key: "research_lab", Value: 1}},
		RewardDM: 8,
	},
	{
		ID: "research_energy", Name: "Research Energy", Description: "Research Energy Tech to level 1",
		Category: CategoryResearch,
		Requirements: []QuestRequirement{{Type: "tech_level", Key: "energy_tech", Value: 1}},
		RewardDM: 10,
	},
	{
		ID: "explorer", Name: "Explorer", Description: "Send an Expedition into the nebula",
		Category: CategoryNebula,
		Requirements: []QuestRequirement{{Type: "expedition_count", Key: "expedition", Value: 1}},
		RewardDM: 15,
	},
	{
		ID: "fleet_power", Name: "Fleet Power", Description: "Build 5 Light Fighters",
		Category: CategoryFleet,
		Requirements: []QuestRequirement{{Type: "ship_count", Key: "light_fighter", Value: 5}},
		RewardDM: 12,
	},
	{
		ID: "collector", Name: "Collector", Description: "Have 10,000 total resources on your planet",
		Category: CategoryEconomy,
		Requirements: []QuestRequirement{{Type: "total_resources", Key: "resources", Value: 10000}},
		RewardDM: 5,
	},
	{
		ID: "attack", Name: "Attack!", Description: "Launch an attack on another player",
		Category: CategoryCombat,
		Requirements: []QuestRequirement{{Type: "attack_count", Key: "attack", Value: 1}},
		RewardDM: 15,
	},
	{
		ID: "defender", Name: "Defender", Description: "Build 5 Rocket Launchers",
		Category: CategoryCombat,
		Requirements: []QuestRequirement{{Type: "defense_count", Key: "rocket_launcher", Value: 5}},
		RewardDM: 10,
	},
	{
		ID: "social", Name: "Social", Description: "Join an alliance",
		Category: CategorySocial,
		Requirements: []QuestRequirement{{Type: "alliance_join", Key: "alliance", Value: 1}},
		RewardDM: 10,
	},
	{
		ID: "alliance_bank", Name: "Alliance Bank", Description: "Donate resources to your alliance bank",
		Category: CategorySocial,
		Requirements: []QuestRequirement{{Type: "alliance_donate", Key: "donation", Value: 1}},
		RewardDM: 8,
	},
	{
		ID: "nebula_explorer", Name: "Nebula Explorer", Description: "Complete 3 expeditions",
		Category: CategoryNebula,
		Requirements: []QuestRequirement{{Type: "expedition_count", Key: "expedition", Value: 3}},
		RewardDM: 20,
	},
	{
		ID: "commander", Name: "Commander", Description: "Hire a commander",
		Category: CategoryNebula,
		Requirements: []QuestRequirement{{Type: "commander_hired", Key: "commander", Value: 1}},
		RewardDM: 10,
	},
	{
		ID: "recycler", Name: "Recycler", Description: "Build a Recycler",
		Category: CategoryFleet,
		Requirements: []QuestRequirement{{Type: "ship_count", Key: "recycler", Value: 1}},
		RewardDM: 10,
	},
}

type QuestService struct {
	repo Repository
}

func NewQuestService(repo Repository) *QuestService {
	return &QuestService{repo: repo}
}

func (s *QuestService) ListQuests(ctx context.Context, playerID int) ([]PlayerQuestResponse, error) {
	defs := QuestDefinitions

	existingQuests, err := s.repo.GetPlayerQuests(ctx, playerID)
	if err != nil {
		existingQuests = nil
	}

	existingMap := make(map[string]PlayerQuest)
	for _, pq := range existingQuests {
		existingMap[pq.QuestID] = pq
	}

	var responses []PlayerQuestResponse
	for _, def := range defs {
		pq, exists := existingMap[def.ID]
		if !exists {
			pq = PlayerQuest{
				PlayerID: playerID,
				QuestID:  def.ID,
				Status:   StatusAvailable,
			}
		}

		if pq.Status != StatusClaimed {
			s.checkAndUpdateSingleQuest(ctx, playerID, &def, &pq)
		}

		responses = append(responses, PlayerQuestResponse{
			Definition: def,
			Progress:   pq,
		})
	}

	return responses, nil
}

func (s *QuestService) ClaimReward(ctx context.Context, playerID int, questID string) (*QuestDefinition, error) {
	def := findQuestDef(questID)
	if def == nil {
		return nil, fmt.Errorf("quest not found: %s", questID)
	}

	pq, err := s.repo.GetPlayerQuest(ctx, playerID, questID)
	if err != nil {
		return nil, fmt.Errorf("quest not started: %w", err)
	}

	if pq.Status != StatusCompleted {
		return nil, fmt.Errorf("quest %s is not completed (status: %s)", questID, pq.Status)
	}

	claimed, err := s.repo.HasClaimedQuest(ctx, playerID, questID)
	if err != nil {
		return nil, err
	}
	if claimed {
		return nil, fmt.Errorf("quest %s already claimed", questID)
	}

	now := time.Now()
	if err := s.repo.ClaimPlayerQuest(ctx, playerID, questID, now); err != nil {
		return nil, fmt.Errorf("claim quest: %w", err)
	}

	if def.RewardDM > 0 {
		if err := s.repo.AddDarkMatter(ctx, playerID, def.RewardDM); err != nil {
			slog.Error("add dark matter reward", "quest", questID, "error", err)
		}
	}

	if def.RewardMetal > 0 || def.RewardCrystal > 0 || def.RewardGas > 0 {
		if err := s.repo.AddPlayerResources(ctx, playerID, def.RewardMetal, def.RewardCrystal, def.RewardGas); err != nil {
			slog.Error("add resource reward", "quest", questID, "error", err)
		}
	}

	return def, nil
}

func (s *QuestService) CheckAndUpdateProgress(ctx context.Context, playerID int, eventType string, eventData map[string]interface{}) error {
	defs := QuestDefinitions

	for _, def := range defs {
		pq, err := s.repo.GetPlayerQuest(ctx, playerID, def.ID)
		if err != nil {
			pq = &PlayerQuest{
				PlayerID: playerID,
				QuestID:  def.ID,
				Status:   StatusAvailable,
			}
		}

		if pq.Status == StatusClaimed || pq.Status == StatusCompleted {
			continue
		}

		s.checkAndUpdateSingleQuest(ctx, playerID, &def, pq)
	}

	return nil
}

func (s *QuestService) checkAndUpdateSingleQuest(ctx context.Context, playerID int, def *QuestDefinition, pq *PlayerQuest) {
	if pq.Status == StatusClaimed {
		return
	}

	allMet := true
	maxProgress := 1

	for _, req := range def.Requirements {
		current := s.evaluateRequirement(ctx, playerID, req)
		if req.Value > 1 {
			maxProgress = req.Value
		}
		if current < req.Value {
			allMet = false
		}
	}

	progressTarget := maxProgress
	for _, req := range def.Requirements {
		if req.Value > progressTarget {
			progressTarget = req.Value
		}
	}

	currentProgress := 0
	for _, req := range def.Requirements {
		val := s.evaluateRequirement(ctx, playerID, req)
		if val > currentProgress {
			currentProgress = val
		}
	}

	if pq.Status == StatusAvailable || pq.Status == StatusLocked {
		now := time.Now()
		pq.Status = StatusInProgress
		pq.StartedAt = &now
		pq.ProgressCurrent = currentProgress
		pq.ProgressTarget = progressTarget
	} else if pq.Status == StatusInProgress {
		pq.ProgressCurrent = currentProgress
		pq.ProgressTarget = progressTarget
	}

	if allMet && pq.Status != StatusCompleted {
		now := time.Now()
		pq.Status = StatusCompleted
		pq.CompletedAt = &now
		pq.ProgressCurrent = progressTarget
	}

	if err := s.repo.UpsertPlayerQuest(ctx, *pq); err != nil {
		slog.Error("upsert player quest", "quest_id", def.ID, "error", err)
	}
}

func (s *QuestService) evaluateRequirement(ctx context.Context, playerID int, req QuestRequirement) int {
	switch req.Type {
	case "building_level":
		level, err := s.repo.GetBuildingLevel(ctx, playerID, req.Key)
		if err != nil {
			return 0
		}
		return level
	case "tech_level":
		level, err := s.repo.GetTechLevel(ctx, playerID, req.Key)
		if err != nil {
			return 0
		}
		return level
	case "ship_count":
		count, err := s.repo.GetPlayerShipCount(ctx, playerID, req.Key)
		if err != nil {
			return 0
		}
		return count
	case "defense_count":
		count, err := s.repo.GetPlayerDefenseCount(ctx, playerID, req.Key)
		if err != nil {
			return 0
		}
		return count
	case "total_resources":
		total, err := s.repo.GetTotalPlayerResources(ctx, playerID)
		if err != nil {
			return 0
		}
		return total
	case "expedition_count":
		count, err := s.repo.GetExpeditionCount(ctx, playerID)
		if err != nil {
			return 0
		}
		return count
	case "alliance_join":
		return 1
	case "alliance_donate":
		return 1
	case "attack_count":
		return 1
	case "commander_hired":
		return 1
	default:
		return 0
	}
}

func (s *QuestService) GetCompletedQuests(ctx context.Context, playerID int) ([]string, error) {
	return s.repo.GetCompletedQuestIDs(ctx, playerID)
}

func findQuestDef(id string) *QuestDefinition {
	for _, d := range QuestDefinitions {
		if d.ID == id {
			return &d
		}
	}
	return nil
}
