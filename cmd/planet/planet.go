package main

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"
)

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

const baseStorage = 10000
const baseMaxFields = 40

const (
	PlanetTypeTerran   = "terran"
	PlanetTypeDesert   = "desert"
	PlanetTypeIce      = "ice"
	PlanetTypeVolcanic = "volcanic"
	PlanetTypeGasGiant = "gas_giant"
)

var ErrInsufficientResources = errors.New("insufficient resources")
var ErrTechLevelAddFailed = errors.New("tech level add failed")
var ErrAlreadyQueued = errors.New("building already in queue")
var ErrInvalidBuilding = errors.New("invalid building type")
var ErrNoFieldsAvailable = errors.New("no fields available for construction")
var ErrNoActiveUpgrade = errors.New("no active upgrade for this building")
var ErrAlreadyDeconstructing = errors.New("building already queued for deconstruction")
var ErrBuildingNotFound = errors.New("building not found")
var ErrPrerequisitesNotMet = errors.New("prerequisites not met")
var ErrInvalidShip = errors.New("invalid ship type")
var ErrInvalidDefense = errors.New("invalid defense type")
var ErrNoShipyard = errors.New("no shipyard")
var ErrMoonNotFound = errors.New("moon not found")
var ErrMoonBaseRequired = errors.New("moon base required")
var ErrWormholeNotFound = errors.New("wormhole generator not found")
var ErrPioneerLabRequired = errors.New("pioneer lab required")
var ErrInvalidMissileCount = errors.New("invalid missile count")
var ErrMissileSiloRequired = errors.New("missile silo required")
var ErrInsufficientSiloCapacity = errors.New("insufficient silo capacity")
var ErrInsufficientIPMs = errors.New("insufficient IPMs")
var ErrNoGemSlotsAvailable = errors.New("no gem slots available")
var ErrGemSlotOccupied = errors.New("gem slot already occupied")
var ErrGemSlotEmpty = errors.New("gem slot is empty")
var ErrInvalidGemType = errors.New("invalid gem type")
var ErrInsufficientShards = errors.New("insufficient shards")
var ErrCombineFailed = errors.New("gem combine failed")
var ErrNPCPlanetNotFound = errors.New("NPC planet not found")

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

	if err := s.processCompletedBuilds(ctx, planet.ID); err != nil {
		return Planet{}, nil, err
	}

	buildings, err := s.repo.GetBuildings(ctx, planet.ID)
	if err != nil {
		return Planet{}, nil, err
	}

	energyTechLevel, err := s.repo.GetTechLevel(ctx, planet.UserID, "energy_tech")
	if err != nil {
		return Planet{}, nil, err
	}
	vipPoints, totalResources, err := s.repo.GetPlayerProgress(ctx, planet.ID)
	if err != nil {
		return Planet{}, nil, err
	}
	vipLevel := vipLevelFromPoints(vipPoints)
	rank := rankFromResources(totalResources)
	vipBonus := vipProductionBonus(vipLevel)
	rankBonus := rankProductionBonus(rank)
	netEnergy, efficiency := calculatePenaltyFactor(buildings, energyTechLevel)
	prod := s.calculateProduction(buildings, efficiency, planet.Type, planet.Temperature, energyTechLevel, vipBonus, rankBonus)
	storage := s.calculateStorage(buildings)
	elapsed := time.Since(planet.ResourcesUpdatedAt).Seconds()

	if elapsed > 0 {
		prevMetal, prevCrystal, prevGas := planet.Metal, planet.Crystal, planet.Gas
		planet.Metal = minInt(prevMetal+int(prod.Metal*elapsed), storage.Metal)
		planet.Crystal = minInt(prevCrystal+int(prod.Crystal*elapsed), storage.Crystal)
		planet.Gas = minInt(prevGas+int(prod.Gas*elapsed), storage.Gas)

		addedMetal := planet.Metal - prevMetal
		addedCrystal := planet.Crystal - prevCrystal
		addedGas := planet.Gas - prevGas

		totalMined := addedMetal + addedCrystal + addedGas
		if totalMined > 0 {
			if err := s.repo.AddResourcesProduced(ctx, planet.ID, totalMined); err != nil {
				return Planet{}, nil, err
			}
		}
	}

	now := time.Now()
	planet.ResourcesUpdatedAt = now
	if err := s.repo.UpdateResources(ctx, planet.ID, planet.Metal, planet.Crystal, planet.Gas, now); err != nil {
		return Planet{}, nil, err
	}

	planet.Energy = netEnergy
	return planet, buildings, nil
}

func (s *PlanetService) processCompletedBuilds(ctx context.Context, planetID int) error {
	queue, err := s.repo.GetActiveQueue(ctx, planetID)
	if err != nil {
		return err
	}
	now := time.Now()
	for _, q := range queue {
		if now.After(q.CompletesAt) {
			if q.Status == "deconstruct" {
				if err := s.handleDeconstructCompletion(ctx, planetID, q); err != nil {
					return err
				}
			} else {
				if err := s.repo.CompleteBuild(ctx, q.ID, q.BuildingType, q.TargetLevel); err != nil {
					return err
				}
				if q.BuildingType == "terraformer" {
					if err := s.repo.UpdateMaxFields(ctx, planetID, baseMaxFields+terraformerFields(q.TargetLevel)); err != nil {
						return err
					}
				}
				if err := s.repo.AddVIPPoints(ctx, planetID, 10); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (s *PlanetService) handleDeconstructCompletion(ctx context.Context, planetID int, q QueueEntry) error {
	metalCost, crystalCost, gasCost := buildingCostResources(q.BuildingType, q.TargetLevel)
	refundMetal := metalCost / 2
	refundCrystal := crystalCost / 2
	refundGas := gasCost / 2

	var maxFields int
	if q.BuildingType == "terraformer" {
		maxFields = baseMaxFields + terraformerFields(q.TargetLevel)
	}

	return s.repo.DeconstructComplete(ctx, planetID, q.ID, q.BuildingType, q.TargetLevel, refundMetal, refundCrystal, refundGas, maxFields)
}

func (s *PlanetService) StartBuildingUpgrade(ctx context.Context, planetID int, buildingType string) (QueueEntry, error) {
	queue, err := s.repo.GetActiveQueue(ctx, planetID)
	if err != nil {
		return QueueEntry{}, err
	}
	for _, q := range queue {
		if q.BuildingType == buildingType && !q.CompletesAt.Before(time.Now()) {
			return QueueEntry{}, ErrAlreadyQueued
		}
	}

	planet, err := s.repo.FindByID(ctx, planetID)
	if err != nil {
		return QueueEntry{}, err
	}

	currentLevel, err := s.repo.GetBuildingLevel(ctx, planetID, buildingType)
	if err != nil {
		return QueueEntry{}, err
	}

	if buildingType == "small_shield_dome" || buildingType == "large_shield_dome" {
		if currentLevel >= 1 {
			return QueueEntry{}, fmt.Errorf("max 1 %s per planet", buildingType)
		}
	}

	if buildingType != "terraformer" {
		buildings, err := s.repo.GetBuildings(ctx, planetID)
		if err != nil {
			return QueueEntry{}, err
		}
		if len(buildings) >= planet.MaxFields {
			return QueueEntry{}, ErrNoFieldsAvailable
		}
	}

	if buildingType == "fusion_reactor" {
		gasLevel, err := s.repo.GetBuildingLevel(ctx, planetID, "gas_mine")
		if err != nil {
			return QueueEntry{}, err
		}
		if gasLevel < 5 {
			return QueueEntry{}, ErrPrerequisitesNotMet
		}
		techLevel, err := s.repo.GetTechLevel(ctx, planet.UserID, "energy_tech")
		if err != nil {
			return QueueEntry{}, err
		}
		if techLevel < 3 {
			return QueueEntry{}, ErrPrerequisitesNotMet
		}
	}

	metalCost, crystalCost, gasCost := buildingCostResources(buildingType, currentLevel)
	if planet.Metal < metalCost || planet.Crystal < crystalCost || planet.Gas < gasCost {
		return QueueEntry{}, ErrInsufficientResources
	}

	roboticsLevel, _ := s.repo.GetBuildingLevel(ctx, planetID, "robotics_factory")
	naniteLevel, _ := s.repo.GetBuildingLevel(ctx, planetID, "nanite_factory")

	completesAt := time.Now().Add(buildingBuildDuration(buildingType, currentLevel, roboticsLevel, naniteLevel))
	entry, err := s.repo.CreateQueueEntry(ctx, planetID, buildingType, currentLevel+1, completesAt)
	if err != nil {
		return QueueEntry{}, err
	}

	if err := s.repo.UpdateResources(ctx, planetID, planet.Metal-metalCost, planet.Crystal-crystalCost, planet.Gas-gasCost, time.Now()); err != nil {
		return QueueEntry{}, err
	}

	return entry, nil
}

func (s *PlanetService) CancelUpgrade(ctx context.Context, planetID int, buildingType string) error {
	queue, err := s.repo.GetActiveQueue(ctx, planetID)
	if err != nil {
		return err
	}

	var targetEntry *QueueEntry
	for _, q := range queue {
		if q.BuildingType == buildingType && q.Status == "upgrade" {
			targetEntry = &q
			break
		}
	}
	if targetEntry == nil {
		return ErrNoActiveUpgrade
	}

	metalCost, crystalCost, gasCost := buildingCostResources(buildingType, targetEntry.TargetLevel-1)
	refundMetal := metalCost / 2
	refundCrystal := crystalCost / 2
	refundGas := gasCost / 2

	return s.repo.CancelUpgradeWithRefund(ctx, planetID, targetEntry.ID, refundMetal, refundCrystal, refundGas)
}

func (s *PlanetService) QueueDeconstruction(ctx context.Context, planetID int, buildingType string) (QueueEntry, error) {
	currentLevel, err := s.repo.GetBuildingLevel(ctx, planetID, buildingType)
	if err != nil {
		return QueueEntry{}, ErrBuildingNotFound
	}
	if currentLevel < 1 {
		return QueueEntry{}, ErrBuildingNotFound
	}

	queue, err := s.repo.GetActiveQueue(ctx, planetID)
	if err != nil {
		return QueueEntry{}, err
	}
	for _, q := range queue {
		if q.BuildingType == buildingType {
			if q.Status == "deconstruct" {
				return QueueEntry{}, ErrAlreadyDeconstructing
			}
			return QueueEntry{}, ErrAlreadyQueued
		}
	}

	roboticsLevel, _ := s.repo.GetBuildingLevel(ctx, planetID, "robotics_factory")
	naniteLevel, _ := s.repo.GetBuildingLevel(ctx, planetID, "nanite_factory")
	duration := buildingBuildDuration(buildingType, currentLevel-1, roboticsLevel, naniteLevel) / 2
	completesAt := time.Now().Add(duration)

	entry, err := s.repo.CreateQueueEntryDeconstruct(ctx, planetID, buildingType, currentLevel-1, completesAt)
	if err != nil {
		return QueueEntry{}, err
	}

	return entry, nil
}

func (s *PlanetService) BuildShips(ctx context.Context, planetID int, shipType string, quantity int) (int, float64, error) {
	cfg, ok := shipConfig(shipType)
	if !ok {
		return 0, 0, ErrInvalidShip
	}

	if quantity < 1 {
		return 0, 0, fmt.Errorf("quantity must be positive")
	}

	shipyardLevel, err := s.repo.GetBuildingLevel(ctx, planetID, "shipyard")
	if err != nil {
		return 0, 0, err
	}
	if shipyardLevel < 1 {
		return 0, 0, ErrNoShipyard
	}

	planet, err := s.repo.FindByID(ctx, planetID)
	if err != nil {
		return 0, 0, err
	}

	totalMetal := cfg.Metal * quantity
	totalCrystal := cfg.Crystal * quantity
	totalGas := cfg.Gas * quantity

	if planet.Metal < totalMetal || planet.Crystal < totalCrystal || planet.Gas < totalGas {
		return 0, 0, ErrInsufficientResources
	}

	if err := s.repo.UpdateResources(ctx, planetID, planet.Metal-totalMetal, planet.Crystal-totalCrystal, planet.Gas-totalGas, time.Now()); err != nil {
		return 0, 0, err
	}

	if err := s.repo.AddPlayerShips(ctx, planetID, planet.UserID, shipType, quantity); err != nil {
		return 0, 0, err
	}

	naniteLevel, _ := s.repo.GetBuildingLevel(ctx, planetID, "nanite_factory")
	buildTime := shipBuildDuration(shipType, quantity, shipyardLevel, naniteLevel)
	return quantity, buildTime, nil
}

func (s *PlanetService) MaxShipQuantity(ctx context.Context, planetID int, shipType string) (int, error) {
	cfg, ok := shipConfig(shipType)
	if !ok {
		return 0, ErrInvalidShip
	}

	planet, err := s.repo.FindByID(ctx, planetID)
	if err != nil {
		return 0, err
	}

	max := math.MaxInt32
	if cfg.Metal > 0 {
		if n := planet.Metal / cfg.Metal; n < max {
			max = n
		}
	}
	if cfg.Crystal > 0 {
		if n := planet.Crystal / cfg.Crystal; n < max {
			max = n
		}
	}
	if cfg.Gas > 0 {
		if n := planet.Gas / cfg.Gas; n < max {
			max = n
		}
	}
	return max, nil
}

func (s *PlanetService) BuildDefenses(ctx context.Context, planetID int, defenseType string, quantity int) (int, float64, error) {
	cfg, ok := defenseConfig(defenseType)
	if !ok {
		return 0, 0, ErrInvalidDefense
	}

	if quantity < 1 {
		return 0, 0, fmt.Errorf("quantity must be positive")
	}

	shipyardLevel, err := s.repo.GetBuildingLevel(ctx, planetID, "shipyard")
	if err != nil {
		return 0, 0, err
	}
	if shipyardLevel < 1 {
		return 0, 0, ErrNoShipyard
	}

	planet, err := s.repo.FindByID(ctx, planetID)
	if err != nil {
		return 0, 0, err
	}

	totalMetal := cfg.Metal * quantity
	totalCrystal := cfg.Crystal * quantity
	totalGas := cfg.Gas * quantity

	if planet.Metal < totalMetal || planet.Crystal < totalCrystal || planet.Gas < totalGas {
		return 0, 0, ErrInsufficientResources
	}

	if err := s.repo.UpdateResources(ctx, planetID, planet.Metal-totalMetal, planet.Crystal-totalCrystal, planet.Gas-totalGas, time.Now()); err != nil {
		return 0, 0, err
	}

	if err := s.repo.AddPlayerDefenses(ctx, planetID, planet.UserID, defenseType, quantity); err != nil {
		return 0, 0, err
	}

	naniteLevel, _ := s.repo.GetBuildingLevel(ctx, planetID, "nanite_factory")
	buildTime := defenseBuildDuration(defenseType, quantity, shipyardLevel, naniteLevel)
	return quantity, buildTime, nil
}

func (s *PlanetService) MaxDefenseQuantity(ctx context.Context, planetID int, defenseType string) (int, error) {
	cfg, ok := defenseConfig(defenseType)
	if !ok {
		return 0, ErrInvalidDefense
	}

	planet, err := s.repo.FindByID(ctx, planetID)
	if err != nil {
		return 0, err
	}

	max := math.MaxInt32
	if cfg.Metal > 0 {
		if n := planet.Metal / cfg.Metal; n < max {
			max = n
		}
	}
	if cfg.Crystal > 0 {
		if n := planet.Crystal / cfg.Crystal; n < max {
			max = n
		}
	}
	if cfg.Gas > 0 {
		if n := planet.Gas / cfg.Gas; n < max {
			max = n
		}
	}
	return max, nil
}

func (s *PlanetService) AddTechLevel(ctx context.Context, playerID int, techType string) (int, error) {
	currentLevel, err := s.repo.GetTechLevel(ctx, playerID, techType)
	if err != nil {
		return 0, fmt.Errorf("get tech level: %w", err)
	}
	newLevel := currentLevel + 1
	if err := s.repo.AddTechLevel(ctx, playerID, techType, newLevel); err != nil {
		return 0, fmt.Errorf("add tech level: %w", err)
	}
	return newLevel, nil
}

func (s *PlanetService) RenamePlanet(ctx context.Context, planetID int, name string) error {
	name = strings.TrimSpace(name)
	if len(name) < 1 || len(name) > 100 {
		return fmt.Errorf("planet name must be 1-100 characters")
	}
	return s.repo.UpdatePlanetName(ctx, planetID, name)
}

func (s *PlanetService) GetHighestLabLevel(ctx context.Context, playerID int) (int, error) {
	return s.repo.GetHighestLabLevel(ctx, playerID)
}

func (s *PlanetService) RepairDefenses(ctx context.Context, planetID int, losses map[string]int) (map[string]int, error) {
	repaired := make(map[string]int)
	for defenseType, lostQty := range losses {
		currentQty, err := s.repo.GetPlayerDefense(ctx, planetID, defenseType)
		if err != nil {
			return nil, err
		}
		repairQty := int(math.Ceil(float64(lostQty) * 0.7))
		newQty := currentQty + repairQty
		if err := s.repo.SetPlayerDefense(ctx, planetID, defenseType, newQty); err != nil {
			return nil, err
		}
		repaired[defenseType] = repairQty
	}
	return repaired, nil
}

func (s *PlanetService) DeductDefenses(ctx context.Context, planetID int, losses map[string]int) error {
	for defenseType, lostQty := range losses {
		currentQty, err := s.repo.GetPlayerDefense(ctx, planetID, defenseType)
		if err != nil {
			return err
		}
		newQty := currentQty - lostQty
		if newQty < 0 {
			newQty = 0
		}
		if err := s.repo.SetPlayerDefense(ctx, planetID, defenseType, newQty); err != nil {
			return err
		}
	}
	return nil
}

func (s *PlanetService) BuildIPM(ctx context.Context, playerID, planetID, count int) error {
	if count < 1 {
		return fmt.Errorf("count must be positive")
	}

	siloLevel, err := s.repo.GetBuildingLevel(ctx, planetID, "missile_silo")
	if err != nil {
		return err
	}
	if siloLevel < 1 {
		return fmt.Errorf("missile silo required")
	}

	capacity := siloLevel * 10
	ipms, abms, err := s.repo.GetMissileCounts(ctx, planetID)
	if err != nil {
		return err
	}
	if ipms+abms+count > capacity {
		return fmt.Errorf("insufficient silo capacity: have %d, need %d more, max %d", ipms+abms, count, capacity)
	}

	totalMetal := 12500 * count
	totalCrystal := 2500 * count
	totalGas := 10000 * count

	planet, err := s.repo.FindByID(ctx, planetID)
	if err != nil {
		return err
	}
	if planet.Metal < totalMetal || planet.Crystal < totalCrystal || planet.Gas < totalGas {
		return ErrInsufficientResources
	}

	if err := s.repo.UpdateResources(ctx, planetID, planet.Metal-totalMetal, planet.Crystal-totalCrystal, planet.Gas-totalGas, time.Now()); err != nil {
		return err
	}

	return s.repo.AddIPMs(ctx, planetID, count)
}

func (s *PlanetService) BuildABM(ctx context.Context, playerID, planetID, count int) error {
	if count < 1 {
		return fmt.Errorf("count must be positive")
	}

	siloLevel, err := s.repo.GetBuildingLevel(ctx, planetID, "missile_silo")
	if err != nil {
		return err
	}
	if siloLevel < 1 {
		return fmt.Errorf("missile silo required")
	}

	capacity := siloLevel * 10
	ipms, abms, err := s.repo.GetMissileCounts(ctx, planetID)
	if err != nil {
		return err
	}
	if ipms+abms+count > capacity {
		return fmt.Errorf("insufficient silo capacity: have %d, need %d more, max %d", ipms+abms, count, capacity)
	}

	totalMetal := 8000 * count
	totalCrystal := 2000 * count
	totalGas := 2000 * count

	planet, err := s.repo.FindByID(ctx, planetID)
	if err != nil {
		return err
	}
	if planet.Metal < totalMetal || planet.Crystal < totalCrystal || planet.Gas < totalGas {
		return ErrInsufficientResources
	}

	if err := s.repo.UpdateResources(ctx, planetID, planet.Metal-totalMetal, planet.Crystal-totalCrystal, planet.Gas-totalGas, time.Now()); err != nil {
		return err
	}

	return s.repo.AddABMs(ctx, planetID, count)
}

func (s *PlanetService) GetMissileCounts(ctx context.Context, planetID int) (MissileCounts, error) {
	ipms, abms, err := s.repo.GetMissileCounts(ctx, planetID)
	if err != nil {
		return MissileCounts{}, err
	}
	return MissileCounts{IPMs: ipms, ABMs: abms}, nil
}

func (s *PlanetService) LaunchIPMs(ctx context.Context, playerID, originPlanetID, targetGalaxy, targetSystem, targetPosition, count int) error {
	if count < 1 {
		return fmt.Errorf("count must be positive")
	}

	siloLevel, err := s.repo.GetBuildingLevel(ctx, originPlanetID, "missile_silo")
	if err != nil {
		return err
	}
	if siloLevel < 2 {
		return fmt.Errorf("missile silo level 2 required to launch IPMs")
	}

	ipms, _, err := s.repo.GetMissileCounts(ctx, originPlanetID)
	if err != nil {
		return err
	}
	if ipms < count {
		return fmt.Errorf("insufficient IPMs: have %d, need %d", ipms, count)
	}

	if err := s.repo.DeductIPMs(ctx, originPlanetID, count); err != nil {
		return err
	}

	return nil
}

func gemStarBonus(starLevel int) float64 {
	return float64(starLevel) * 0.05
}

func gemBonusByType(gemType string, starLevel int) (attackBonus, armorBonus, strengthBonus float64) {
	bonus := gemStarBonus(starLevel)
	switch gemType {
	case "flaming_crystal":
		return bonus, 0, 0
	case "concentrated_galactonite":
		return 0, bonus, 0
	case "galactonite_shield":
		return 0, 0, bonus
	}
	return 0, 0, 0
}

func validGemType(gemType string) bool {
	switch gemType {
	case "flaming_crystal", "concentrated_galactonite", "galactonite_shield":
		return true
	}
	return false
}

func (s *PlanetService) GetGemSlots(ctx context.Context, planetID int) ([]GemSlot, error) {
	return s.repo.GetGemSlots(ctx, planetID)
}

func (s *PlanetService) EquipGem(ctx context.Context, planetID, slotIndex int, gemType string) error {
	if !validGemType(gemType) {
		return ErrInvalidGemType
	}
	planet, err := s.repo.FindByID(ctx, planetID)
	if err != nil {
		return err
	}
	if planet.UserID == 0 {
		return fmt.Errorf("cannot equip gems on NPC planets")
	}
	slots, err := s.repo.GetGemSlots(ctx, planetID)
	if err != nil {
		return err
	}
	if slotIndex < 0 || slotIndex >= 3 {
		return ErrNoGemSlotsAvailable
	}
	for _, slot := range slots {
		if slot.SlotIndex == slotIndex && slot.GemType != "" {
			return ErrGemSlotOccupied
		}
	}
	researchCenterLevel, err := s.repo.GetBuildingLevel(ctx, planetID, "galactonite_research_center")
	if err != nil || researchCenterLevel < 1 {
		return fmt.Errorf("galactonite research center required")
	}
	return s.repo.SetGemSlot(ctx, planetID, slotIndex, gemType, 1)
}

func (s *PlanetService) UnequipGem(ctx context.Context, planetID, slotIndex int) error {
	slots, err := s.repo.GetGemSlots(ctx, planetID)
	if err != nil {
		return err
	}
	found := false
	for _, slot := range slots {
		if slot.SlotIndex == slotIndex && slot.GemType != "" {
			found = true
			break
		}
	}
	if !found {
		return ErrGemSlotEmpty
	}
	return s.repo.SetGemSlot(ctx, planetID, slotIndex, "", 0)
}

func (s *PlanetService) GetGemBonuses(ctx context.Context, planetID int) (GemBonuses, error) {
	slots, err := s.repo.GetGemSlots(ctx, planetID)
	if err != nil {
		return GemBonuses{}, err
	}
	var totalAttack, totalArmor, totalStrength float64
	for _, slot := range slots {
		if slot.GemType != "" && slot.StarLevel > 0 {
			aBonus, arBonus, sBonus := gemBonusByType(slot.GemType, slot.StarLevel)
			totalAttack += aBonus
			totalArmor += arBonus
			totalStrength += sBonus
		}
	}
	return GemBonuses{
		AttackBonus:    totalAttack,
		ArmorBonus:     totalArmor,
		StrengthBonus:  totalStrength,
	}, nil
}

func (s *PlanetService) CombineGem(ctx context.Context, planetID, slotIndex int, gemType string) error {
	if !validGemType(gemType) {
		return ErrInvalidGemType
	}
	planet, err := s.repo.FindByID(ctx, planetID)
	if err != nil {
		return err
	}
	slots, err := s.repo.GetGemSlots(ctx, planetID)
	if err != nil {
		return err
	}
	if slotIndex < 0 || slotIndex >= 3 {
		return ErrNoGemSlotsAvailable
	}
	galactoniteResearchCenter, err := s.repo.GetBuildingLevel(ctx, planetID, "galactonite_research_center")
	if err != nil || galactoniteResearchCenter < 1 {
		return fmt.Errorf("galactonite research center required")
	}
	shards, err := s.repo.GetShardCount(ctx, planet.UserID)
	if err != nil {
		return err
	}
	needed := 10
	currentShards := shards[gemType]
	if currentShards < needed {
		return ErrInsufficientShards
	}
	combineAttempts := 0
	for gemType2, count := range shards {
		_ = count
		if gemType2 == gemType {
			if v, ok := shards["combine_attempts_"+gemType]; ok {
				combineAttempts = v
			}
		}
	}
	success := combineAttempts >= 20
	if !success {
		chance := 0.3 + float64(combineAttempts)*0.035
		if chance > 1.0 {
			chance = 1.0
		}
		success = rand.Float64() < chance
	}
	if err := s.repo.RemoveShards(ctx, planet.UserID, gemType, needed); err != nil {
		return err
	}
	if err := s.repo.IncrementCombineAttempts(ctx, planet.UserID, gemType); err != nil {
		return err
	}
	if !success {
		return ErrCombineFailed
	}
	currentStarLevel := 0
	for _, slot := range slots {
		if slot.SlotIndex == slotIndex {
			if slot.GemType == gemType {
				currentStarLevel = slot.StarLevel
			}
			break
		}
	}
	newStarLevel := currentStarLevel + 1
	if newStarLevel > 5 {
		newStarLevel = 5
	}
	return s.repo.SetGemSlot(ctx, planetID, slotIndex, gemType, newStarLevel)
}

// NPC planet methods
func (s *PlanetService) SeedNPCPlanet(ctx context.Context, galaxy, system, position int) error {
	planet, err := s.repo.FindByCoords(ctx, galaxy, system, position)
	if err == nil {
		return fmt.Errorf("position already occupied by planet %d", planet.ID)
	}
	if err != ErrPlanetNotFound {
		return err
	}
	typ, temp := planetTypeAndTemp(position)
	planetID, err := s.repo.CreateNPCPlanet(ctx, galaxy, system, position, typ, temp)
	if err != nil {
		return err
	}
	if err := s.repo.SeedBuildingsForPlanet(ctx, planetID); err != nil {
		return err
	}
	if err := s.repo.SeedShipsForPlanet(ctx, planetID); err != nil {
		return err
	}
	if err := s.repo.SeedNPCResources(ctx, planetID); err != nil {
		return err
	}
	if err := s.repo.SeedNPCFleet(ctx, planetID); err != nil {
		return err
	}
	return s.repo.RegisterNPCPlanet(ctx, planetID, galaxy, system, position)
}

func (s *PlanetService) SeedAllNPCPlanets(ctx context.Context) error {
	for galaxy := 1; galaxy <= 9; galaxy++ {
		for system := 1; system <= 499; system++ {
			for _, pos := range []int{4, 9, 14} {
				_, err := s.repo.FindByCoords(ctx, galaxy, system, pos)
				if err == ErrPlanetNotFound {
					if err := s.SeedNPCPlanet(ctx, galaxy, system, pos); err != nil {
						continue
					}
				}
			}
		}
	}
	return nil
}

func (s *PlanetService) ClearNPCPlanet(ctx context.Context, planetID int) error {
	return s.repo.ClearNPCPlanet(ctx, planetID)
}

func (s *PlanetService) GetNPCPlanetByPlanetID(ctx context.Context, planetID int) (*NPCPlanet, error) {
	return s.repo.GetNPCPlanetByPlanetID(ctx, planetID)
}

func (s *PlanetService) GetRespawnedNPCPlanets(ctx context.Context) ([]NPCPlanet, error) {
	return s.repo.GetRespawnedNPCPlanets(ctx)
}

func (s *PlanetService) RespawnNPCPlanet(ctx context.Context, npcPlanetID int) error {
	return s.repo.RespawnNPCPlanet(ctx, npcPlanetID)
}

// Get or create gem slots for a planet (3 slots)
func (s *PlanetService) EnsureGemSlots(ctx context.Context, planetID int) error {
	slots, err := s.repo.GetGemSlots(ctx, planetID)
	if err != nil {
		return err
	}
	if len(slots) > 0 {
		return nil
	}
	for i := 0; i < 3; i++ {
		if err := s.repo.SetGemSlot(ctx, planetID, i, "", 0); err != nil {
			return err
		}
	}
	return nil
}

func defenseBuildDuration(defenseType string, quantity, shipyardLevel, naniteLevel int) float64 {
	cfg, ok := defenseConfig(defenseType)
	if !ok {
		return 0
	}
	totalMC := float64(cfg.Metal + cfg.Crystal)
	if totalMC == 0 {
		return 0
	}
	hours := totalMC * float64(quantity) / (11132 * float64(shipyardLevel+1))
	hours /= math.Pow(2, float64(naniteLevel))
	return hours * 3600
}

func shipBuildDuration(shipType string, quantity, shipyardLevel, naniteLevel int) float64 {
	cfg, ok := shipConfig(shipType)
	if !ok {
		return 0
	}
	totalMC := float64(cfg.Metal + cfg.Crystal)
	if totalMC == 0 {
		return 0
	}
	hours := totalMC * float64(quantity) / (11132 * float64(shipyardLevel+1))
	hours /= math.Pow(2, float64(naniteLevel))
	return hours * 3600
}

func buildingCostResources(buildingType string, currentLevel int) (metal, crystal, gas int) {
	next := float64(currentLevel + 1)
	switch buildingType {
	case "metal_mine":
		return int(60 * math.Pow(1.5, next)), int(15 * math.Pow(1.5, next)), 0
	case "crystal_mine":
		return int(48 * math.Pow(1.6, next)), int(24 * math.Pow(1.6, next)), 0
	case "gas_mine":
		return int(225 * math.Pow(1.5, next)), int(75 * math.Pow(1.5, next)), 0
	case "solar_plant":
		return int(75 * math.Pow(1.5, next)), int(30 * math.Pow(1.5, next)), 0
	case "metal_storage":
		return int(1000 * math.Pow(2, next)), 0, 0
	case "crystal_storage":
		return int(1000 * math.Pow(2, next)), 0, 0
	case "gas_storage":
		return int(1000 * math.Pow(2, next)), 0, 0
	case "robotics_factory":
		return int(400 * math.Pow(2, next)), int(120 * math.Pow(2, next)), int(200 * math.Pow(2, next))
	case "nanite_factory":
		return int(1000000 * math.Pow(2, next)), int(500000 * math.Pow(2, next)), int(100000 * math.Pow(2, next))
	case "terraformer":
		return int(50000 * math.Pow(2, next)), int(50000 * math.Pow(2, next)), int(50000 * math.Pow(2, next))
	case "fusion_reactor":
		return int(200 * math.Pow(2, next)), int(150 * math.Pow(2, next)), int(50 * math.Pow(2, next))
	case "small_shield_dome":
		return 20000, 10000, 0
	case "large_shield_dome":
		return 100000, 50000, 20000
	case "star_gate":
		return int(500000 * math.Pow(1.8, next)), int(400000 * math.Pow(1.8, next)), int(200000 * math.Pow(1.8, next))
	case "alliance_depot":
		return int(20000 * math.Pow(1.6, next)), int(20000 * math.Pow(1.6, next)), int(1000 * math.Pow(1.6, next))
	case "missile_silo":
		return int(20000 * math.Pow(2, next)), int(20000 * math.Pow(2, next)), int(1000 * math.Pow(2, next))
	case "galactonite_research_center":
		return int(50000 * math.Pow(2, next)), int(50000 * math.Pow(2, next)), int(25000 * math.Pow(2, next))
	}
	return 0, 0, 0
}

func moonBuildingCostResources(buildingType string, currentLevel int) (metal, crystal, gas int) {
	next := float64(currentLevel + 1)
	switch buildingType {
	case "moon_base":
		return int(20000 * math.Pow(2, next)), int(10000 * math.Pow(2, next)), int(5000 * math.Pow(2, next))
	case "robotics_factory":
		return int(400 * math.Pow(2, next)), int(120 * math.Pow(2, next)), int(200 * math.Pow(2, next))
	case "shipyard":
		return int(400 * math.Pow(2, next)), int(200 * math.Pow(2, next)), int(100 * math.Pow(2, next))
	case "pioneer_lab":
		return int(20000 * math.Pow(2, next)), int(40000 * math.Pow(2, next)), int(20000 * math.Pow(2, next))
	case "wormhole_generator":
		return int(1600000 * math.Pow(2, next)), int(3200000 * math.Pow(2, next)), int(1600000 * math.Pow(2, next))
	}
	return 0, 0, 0
}

func buildingLabel(buildingType string) string {
	switch buildingType {
	case "metal_mine":
		return "Metal Mine"
	case "crystal_mine":
		return "Crystal Mine"
	case "gas_mine":
		return "Gas Mine"
	case "solar_plant":
		return "Solar Plant"
	case "metal_storage":
		return "Metal Storage"
	case "crystal_storage":
		return "Crystal Storage"
	case "gas_storage":
		return "Gas Storage"
	case "robotics_factory":
		return "Robotics Factory"
	case "nanite_factory":
		return "Nanite Factory"
	case "terraformer":
		return "Terraformer"
	case "fusion_reactor":
		return "Fusion Reactor"
	case "small_shield_dome":
		return "Small Shield Dome"
	case "large_shield_dome":
		return "Large Shield Dome"
	case "star_gate":
		return "Star Gate"
	case "alliance_depot":
		return "Alliance Depot"
	case "missile_silo":
		return "Missile Silo"
	case "galactonite_research_center":
		return "Galactonite Research Center"
	}
	return buildingType
}

func moonBuildingLabel(buildingType string) string {
	switch buildingType {
	case "moon_base":
		return "Moon Base"
	case "robotics_factory":
		return "Robotics Factory"
	case "shipyard":
		return "Shipyard"
	case "pioneer_lab":
		return "Pioneer Lab"
	case "wormhole_generator":
		return "Wormhole Generator"
	}
	return buildingType
}

func (s *PlanetService) GetMoonBuildings(ctx context.Context, galaxy, system, position int) ([]MoonBuilding, int, error) {
	buildings, err := s.repo.GetMoonBuildings(ctx, galaxy, system, position)
	if err != nil {
		return nil, 0, err
	}

	moonBaseLevel := 0
	for _, b := range buildings {
		if b.Type == "moon_base" {
			moonBaseLevel = b.Level
			break
		}
	}
	maxFields := baseMoonFields + moonBaseLevel*moonBaseFieldsPerLevel
	return buildings, maxFields, nil
}

func (s *PlanetService) StartMoonBuildingUpgrade(ctx context.Context, galaxy, system, position int, buildingType string) error {
	moonExists, err := s.repo.MoonExists(ctx, galaxy, system, position)
	if err != nil {
		return err
	}
	if !moonExists {
		return ErrMoonNotFound
	}

	currentLevel, err := s.repo.GetMoonBuildingLevel(ctx, galaxy, system, position, buildingType)
	if err != nil && !errors.Is(err, ErrBuildingNotFound) {
		return err
	}

	if buildingType != "moon_base" {
		moonBaseLevel, err := s.repo.GetMoonBuildingLevel(ctx, galaxy, system, position, "moon_base")
		if err != nil || moonBaseLevel < 1 {
			return ErrMoonBaseRequired
		}
	}

	buildings, err := s.repo.GetMoonBuildings(ctx, galaxy, system, position)
	if err != nil {
		return err
	}

	moonBaseLevel := 0
	for _, b := range buildings {
		if b.Type == "moon_base" {
			moonBaseLevel = b.Level
		}
	}
	maxFields := baseMoonFields + moonBaseLevel*moonBaseFieldsPerLevel

	if buildingType != "moon_base" {
		builtCount := len(buildings)
		if buildingType == "wormhole_generator" {
			hasWG := false
			for _, b := range buildings {
				if b.Type == "wormhole_generator" {
					hasWG = true
					break
				}
			}
			if !hasWG {
				builtCount++
			}
		} else {
			hasIt := false
			for _, b := range buildings {
				if b.Type == buildingType {
					hasIt = true
					break
				}
			}
			if !hasIt {
				builtCount++
			}
		}
		if builtCount > maxFields {
			return ErrNoFieldsAvailable
		}
	}

	if buildingType == "wormhole_generator" {
		if moonBaseLevel < 3 {
			return ErrPrerequisitesNotMet
		}
		roboticsLevel, err := s.repo.GetMoonBuildingLevel(ctx, galaxy, system, position, "robotics_factory")
		if err != nil || roboticsLevel < 3 {
			return ErrPrerequisitesNotMet
		}
	}

	planet, err := s.repo.FindByCoords(ctx, galaxy, system, position)
	if err != nil {
		return err
	}

	metalCost, crystalCost, gasCost := moonBuildingCostResources(buildingType, currentLevel)
	if planet.Metal < metalCost || planet.Crystal < crystalCost || planet.Gas < gasCost {
		return ErrInsufficientResources
	}

	if err := s.repo.UpdateResources(ctx, planet.ID, planet.Metal-metalCost, planet.Crystal-crystalCost, planet.Gas-gasCost, time.Now()); err != nil {
		return err
	}

	newLevel := currentLevel + 1
	if err := s.repo.UpdateMoonBuildingLevel(ctx, galaxy, system, position, buildingType, newLevel); err != nil {
		return err
	}

	if err := s.repo.AddVIPPoints(ctx, planet.ID, 10); err != nil {
		return err
	}

	return nil
}

func (s *PlanetService) BuildIronBehemoth(ctx context.Context, galaxy, system, position int, quantity int) (int, float64, error) {
	if quantity < 1 {
		return 0, 0, fmt.Errorf("quantity must be positive")
	}

	pioneerLabLevel, err := s.repo.GetMoonBuildingLevel(ctx, galaxy, system, position, "pioneer_lab")
	if err != nil || pioneerLabLevel < 1 {
		return 0, 0, ErrPioneerLabRequired
	}

	planet, err := s.repo.FindByCoords(ctx, galaxy, system, position)
	if err != nil {
		return 0, 0, err
	}

	cfg, ok := shipConfig("iron_behemoth")
	if !ok {
		return 0, 0, ErrInvalidShip
	}

	totalMetal := cfg.Metal * quantity
	totalCrystal := cfg.Crystal * quantity
	totalGas := cfg.Gas * quantity

	if planet.Metal < totalMetal || planet.Crystal < totalCrystal || planet.Gas < totalGas {
		return 0, 0, ErrInsufficientResources
	}

	if err := s.repo.UpdateResources(ctx, planet.ID, planet.Metal-totalMetal, planet.Crystal-totalCrystal, planet.Gas-totalGas, time.Now()); err != nil {
		return 0, 0, err
	}

	if err := s.repo.AddPlayerShips(ctx, planet.ID, planet.UserID, "iron_behemoth", quantity); err != nil {
		return 0, 0, err
	}

	naniteLevel, _ := s.repo.GetBuildingLevel(ctx, planet.ID, "nanite_factory")
	buildTime := float64(cfg.Metal+cfg.Crystal) * float64(quantity) / (11132 * float64(pioneerLabLevel+1))
	buildTime /= math.Pow(2, float64(naniteLevel))
	buildTime *= 3600

	return quantity, buildTime, nil
}

func (s *PlanetService) LinkWormholes(ctx context.Context, srcGalaxy, srcSystem, srcPos, dstGalaxy, dstSystem, dstPos int) error {
	srcWormhole, err := s.repo.GetWormhole(ctx, srcGalaxy, srcSystem, srcPos)
	if err != nil {
		return ErrWormholeNotFound
	}
	if srcWormhole.Level < 1 {
		return ErrWormholeNotFound
	}

	dstWormhole, err := s.repo.GetWormhole(ctx, dstGalaxy, dstSystem, dstPos)
	if err != nil {
		return ErrWormholeNotFound
	}
	if dstWormhole.Level < 1 {
		return ErrWormholeNotFound
	}

	now := time.Now()
	if srcWormhole.CooldownUntil != nil && now.Before(*srcWormhole.CooldownUntil) {
		return fmt.Errorf("source wormhole on cooldown until %s", srcWormhole.CooldownUntil.Format(time.RFC3339))
	}
	if dstWormhole.CooldownUntil != nil && now.Before(*dstWormhole.CooldownUntil) {
		return fmt.Errorf("target wormhole on cooldown until %s", dstWormhole.CooldownUntil.Format(time.RFC3339))
	}

	return s.repo.LinkWormholes(ctx, srcGalaxy, srcSystem, srcPos, dstGalaxy, dstSystem, dstPos)
}

func (s *PlanetService) StarGateLink(ctx context.Context, planetID, targetPlanetID int) error {
	planet, err := s.repo.FindByID(ctx, planetID)
	if err != nil {
		return err
	}
	target, err := s.repo.FindByID(ctx, targetPlanetID)
	if err != nil {
		return err
	}

	srcLevel, err := s.repo.GetBuildingLevel(ctx, planetID, "star_gate")
	if err != nil || srcLevel < 1 {
		return fmt.Errorf("no star gate on source planet")
	}

	tgtLevel, err := s.repo.GetBuildingLevel(ctx, targetPlanetID, "star_gate")
	if err != nil || tgtLevel < 1 {
		return fmt.Errorf("no star gate on target planet")
	}

	if planet.UserID != target.UserID {
		return fmt.Errorf("can only link your own planets")
	}

	existing, err := s.repo.GetStarGateLink(ctx, planetID)
	if err != nil {
		return err
	}
	if existing != nil {
		return fmt.Errorf("source planet already has a star gate link")
	}

	existingTgt, err := s.repo.GetStarGateLink(ctx, targetPlanetID)
	if err != nil {
		return err
	}
	if existingTgt != nil {
		return fmt.Errorf("target planet already has a star gate link")
	}

	return s.repo.StarGateLink(ctx, planetID, targetPlanetID)
}

func (s *PlanetService) StarGateUnlink(ctx context.Context, planetID int) error {
	return s.repo.StarGateUnlink(ctx, planetID)
}

func (s *PlanetService) GetStarGateLink(ctx context.Context, planetID int) (*StarGateLink, error) {
	return s.repo.GetStarGateLink(ctx, planetID)
}

func (s *PlanetService) HasStarGateLink(ctx context.Context, originPlanetID, targetPlanetID int) (bool, error) {
	link, err := s.repo.GetStarGateLink(ctx, originPlanetID)
	if err != nil {
		return false, err
	}
	if link == nil {
		return false, nil
	}
	return link.TargetPlanetID == targetPlanetID, nil
}

func allianceDepotStorageCapacity(level int) int {
	if level < 1 {
		return 0
	}
	return int(50000 * math.Pow(1.6, float64(level)))
}

func shieldDomeConfig(domeType string) (ShieldDomeConfig, bool) {
	switch domeType {
	case "small_shield_dome":
		return ShieldDomeConfig{Type: "small_shield_dome", Name: "Small Shield Dome", ShieldHP: 10000, CostMetal: 20000, CostCrystal: 10000}, true
	case "large_shield_dome":
		return ShieldDomeConfig{Type: "large_shield_dome", Name: "Large Shield Dome", ShieldHP: 100000, CostMetal: 100000, CostCrystal: 50000, CostGas: 20000}, true
	}
	return ShieldDomeConfig{}, false
}

func terraformerFields(level int) int {
	return 5 * level
}

func buildingBuildDuration(buildingType string, currentLevel int, roboticsLevel int, naniteLevel int) time.Duration {
	next := float64(currentLevel + 1)
	seconds := 71.3 / float64(roboticsLevel+1) * math.Exp(0.406*next)
	if buildingType != "nanite_factory" {
		seconds /= math.Pow(2, float64(naniteLevel))
	}
	return time.Duration(seconds * float64(time.Second))
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func calculatePenaltyFactor(buildings []Building, energyTechLevel int) (netEnergyPerMin int, efficiency float64) {
	var totalProd, totalCons float64
	for _, b := range buildings {
		if b.Type == "solar_plant" {
			totalProd += productionRate("solar_plant", b.Level)
		} else if b.Type == "fusion_reactor" {
			totalProd += fusionEnergyOutput(b.Level, energyTechLevel)
		} else {
			totalCons += energyConsumptionPerMinute(b.Type, b.Level)
		}
	}

	netEnergyPerMin = int(totalProd - totalCons)
	efficiency = 1.0
	if netEnergyPerMin < 0 && totalCons > 0 {
		efficiency = totalProd / totalCons
		if efficiency < 0.1 {
			efficiency = 0.1
		}
	}
	return
}

func energyConsumptionPerMinute(buildingType string, level int) float64 {
	if level < 1 {
		return 0
	}
	switch buildingType {
	case "metal_mine":
		return 10 * float64(level)
	case "crystal_mine":
		return 10 * float64(level)
	case "gas_mine":
		return 20 * float64(level)
	}
	return 0
}

func productionRateForLevel(buildingType string, level float64) float64 {
	if level < 1 {
		return 0
	}
	base := productionRate(buildingType, int(level))
	next := productionRate(buildingType, int(level)+1)
	fract := level - float64(int(level))
	if fract == 0 {
		return base
	}
	return base + fract*(next-base)
}

func (s *PlanetService) calculateProduction(buildings []Building, efficiency float64, planetType string, temperature int, energyTechLevel int, vipBonus float64, rankBonus float64) Production {
	levels := make(map[string]int)
	for _, b := range buildings {
		levels[b.Type] = b.Level
	}

	gasLevel := float64(levels["gas_mine"])
	solarLevel := float64(levels["solar_plant"])

	if planetType == PlanetTypeIce || planetType == PlanetTypeGasGiant {
		gasLevel += 1.5
	}
	if planetType == PlanetTypeDesert || planetType == PlanetTypeVolcanic {
		solarLevel += 1.5
	}

	if temperature <= -60 {
		gasLevel += 2.5
	} else if temperature <= -30 {
		gasLevel += 2.0
	} else if temperature <= 0 {
		gasLevel += 1.5
	}

	if temperature >= 80 {
		solarLevel += 3.0
	} else if temperature >= 60 {
		solarLevel += 2.0
	} else if temperature >= 40 {
		solarLevel += 1.0
	}

	gasProduction := productionRateForLevel("gas_mine", gasLevel) / 60.0 * efficiency

	fusionConsumption := 0.0
	fusionEnergy := 0.0
	if levels["fusion_reactor"] >= 1 {
		fusionConsumption = fusionGasConsumption(levels["fusion_reactor"]) / 60.0
		fusionEnergy = fusionEnergyOutput(levels["fusion_reactor"], energyTechLevel) / 60.0
	}

	netGas := gasProduction - fusionConsumption
	if netGas < 0 {
		netGas = 0
	}

	totalBonus := 1 + vipBonus + rankBonus
	return Production{
		Metal:   productionRate("metal_mine", levels["metal_mine"]) / 60.0 * efficiency * totalBonus,
		Crystal: productionRate("crystal_mine", levels["crystal_mine"]) / 60.0 * efficiency * totalBonus,
		Gas:     netGas,
		Energy:  productionRateForLevel("solar_plant", solarLevel)/60.0 + fusionEnergy,
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
		return math.Round(40*float64(level)*math.Pow(1.1, float64(level))*100) / 100
	}
	return 0
}

func storageCapacity(buildingType string, level int) int {
	if level < 1 {
		return baseStorage
	}
	return baseStorage + int(5000*math.Pow(1.5, float64(level)))
}

func fusionEnergyOutput(level, energyTechLevel int) float64 {
	if level < 1 {
		return 0
	}
	base := math.Round(50 * float64(level) * math.Pow(1.1, float64(level)) * 100) / 100
	return base * (1 + 0.05*float64(energyTechLevel))
}

func fusionGasConsumption(level int) float64 {
	if level < 1 {
		return 0
	}
	return math.Round(10 * float64(level) * math.Pow(1.1, float64(level)) * 100) / 100
}

func planetTypeAndTemp(position int) (typ string, temperature int) {
	switch {
	case position >= 1 && position <= 3:
		if rand.Intn(100) < 80 {
			typ = PlanetTypeDesert
		} else {
			typ = PlanetTypeVolcanic
		}
		temperature = 60 + rand.Intn(41) // 60-100
	case position >= 4 && position <= 6:
		typ = PlanetTypeTerran
		temperature = 10 + rand.Intn(31) // 10-40
	case position == 7:
		typ = PlanetTypeTerran
		temperature = rand.Intn(21) // 0-20
	case position >= 8 && position <= 9:
		if rand.Intn(100) < 60 {
			typ = PlanetTypeTerran
		} else {
			typ = PlanetTypeIce
		}
		temperature = -10 + rand.Intn(41) // -10-30
	case position >= 10 && position <= 12:
		typ = PlanetTypeIce
		temperature = -50 + rand.Intn(51) // -50-0
	case position >= 13 && position <= 15:
		typ = PlanetTypeGasGiant
		temperature = -80 + rand.Intn(51) // -80--30
	default:
		typ = PlanetTypeTerran
		temperature = 20
	}
	return
}

func vipLevelFromPoints(points int) int {
	thresholds := []int{100, 500, 1500, 5000, 15000, 40000, 100000, 250000, 500000, 1000000, 2000000, 5000000}
	level := 0
	for _, t := range thresholds {
		if points >= t {
			level++
		} else {
			break
		}
	}
	return level
}

func rankFromResources(produced int) int {
	thresholds := []int64{1000000, 5000000, 25000000, 100000000, 500000000, 1000000000, 5000000000, 25000000000, 100000000000}
	rank := 0
	for _, t := range thresholds {
		if int64(produced) >= t {
			rank++
		} else {
			break
		}
	}
	return rank
}

func vipProductionBonus(vipLevel int) float64 {
	return float64(vipLevel) * 0.03
}

func rankProductionBonus(rank int) float64 {
	bonuses := []float64{0, 0.02, 0.04, 0.06, 0.08, 0.10, 0.12, 0.15, 0.18, 0.20}
	if rank < 0 || rank >= len(bonuses) {
		return 0
	}
	return bonuses[rank]
}

func toPlanetResponse(p Planet, buildings []Building, prod Production, storage Storage, queue []QueueEntry, vipPoints int, totalResources int) PlanetResponse {
	return PlanetResponse{
		ID: p.ID, UserID: p.UserID, Name: p.Name,
		Metal: p.Metal, Crystal: p.Crystal, Gas: p.Gas,
		Energy: p.Energy,
		Galaxy: p.Galaxy, System: p.System, Position: p.Position,
		MaxFields: p.MaxFields, FieldsUsed: len(buildings),
		Type: p.Type, Temperature: p.Temperature,
		Buildings: buildings, Production: prod, Storage: storage, Queue: queue,
		VIPLevel: vipLevelFromPoints(vipPoints),
		Rank:     rankFromResources(totalResources),
	}
}
