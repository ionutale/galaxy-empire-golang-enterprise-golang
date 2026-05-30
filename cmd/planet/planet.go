package main

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"
)

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
var ErrAlreadyQueued = errors.New("building already in queue")
var ErrInvalidBuilding = errors.New("invalid building type")
var ErrNoFieldsAvailable = errors.New("no fields available for construction")
var ErrNoActiveUpgrade = errors.New("no active upgrade for this building")
var ErrAlreadyDeconstructing = errors.New("building already queued for deconstruction")
var ErrBuildingNotFound = errors.New("building not found")
var ErrPrerequisitesNotMet = errors.New("prerequisites not met")
var ErrInvalidShip = errors.New("invalid ship type")
var ErrNoShipyard = errors.New("no shipyard")

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

func (s *PlanetService) BuildShips(ctx context.Context, planetID int, shipType string, quantity int) (int, error) {
	cfg, ok := shipConfig(shipType)
	if !ok {
		return 0, ErrInvalidShip
	}

	if quantity < 1 {
		return 0, fmt.Errorf("quantity must be positive")
	}

	shipyardLevel, err := s.repo.GetBuildingLevel(ctx, planetID, "shipyard")
	if err != nil {
		return 0, err
	}
	if shipyardLevel < 1 {
		return 0, ErrNoShipyard
	}

	planet, err := s.repo.FindByID(ctx, planetID)
	if err != nil {
		return 0, err
	}

	totalMetal := cfg.Metal * quantity
	totalCrystal := cfg.Crystal * quantity
	totalGas := cfg.Gas * quantity

	if planet.Metal < totalMetal || planet.Crystal < totalCrystal || planet.Gas < totalGas {
		return 0, ErrInsufficientResources
	}

	if err := s.repo.UpdateResources(ctx, planetID, planet.Metal-totalMetal, planet.Crystal-totalCrystal, planet.Gas-totalGas, time.Now()); err != nil {
		return 0, err
	}

	if err := s.repo.AddPlayerShips(ctx, planetID, planet.UserID, shipType, quantity); err != nil {
		return 0, err
	}

	return quantity, nil
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
	}
	return 0, 0, 0
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
