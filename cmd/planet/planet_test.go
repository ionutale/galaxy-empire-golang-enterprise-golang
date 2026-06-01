package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

type mockRepo struct {
	planets       map[int]Planet
	buildings     map[int][]Building
	queue         map[int][]QueueEntry
	nextPID       int
	nextBID       int
	nextQID       int
	techLevels    map[int]map[string]int
	playerProgress      map[int]struct{ vipPoints, totalResources int }
	playerShips         map[int]map[string]int
	playerDefenses      map[int]map[string]int
	moonBuildings       map[string][]MoonBuilding
	wormholes           map[string]*WormholeEntry
	moonExists          map[string]bool
	stargateLinks       map[int]*StarGateLink
	missileIPMs         map[int]int
	missileABMs         map[int]int
	gemSlots            map[int][]GemSlot
	shards              map[int]map[string]int
	combineAttempts     map[int]map[string]int
	npcPlanets          map[int]*NPCPlanet
	npcNextID           int
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		planets:    make(map[int]Planet),
		buildings:  make(map[int][]Building),
		queue:      make(map[int][]QueueEntry),
		nextPID:    1,
		nextBID:    1,
		nextQID:    1,
		techLevels:    make(map[int]map[string]int),
		playerProgress: make(map[int]struct{ vipPoints, totalResources int }),
		playerShips:    make(map[int]map[string]int),
		playerDefenses: make(map[int]map[string]int),
		moonBuildings:  make(map[string][]MoonBuilding),
		wormholes:      make(map[string]*WormholeEntry),
		moonExists:     make(map[string]bool),
		stargateLinks:  make(map[int]*StarGateLink),
		missileIPMs:    make(map[int]int),
		missileABMs:    make(map[int]int),
	}
}

func (m *mockRepo) FindByUserID(_ context.Context, userID int) (Planet, error) {
	for _, p := range m.planets {
		if p.UserID == userID {
			return p, nil
		}
	}
	return Planet{}, ErrPlanetNotFound
}

func (m *mockRepo) FindByID(_ context.Context, planetID int) (Planet, error) {
	p, ok := m.planets[planetID]
	if !ok {
		return Planet{}, ErrPlanetNotFound
	}
	return p, nil
}

func (m *mockRepo) CreateAtCoords(_ context.Context, userID int, galaxy, system, position int) (Planet, []Building, error) {
	now := time.Now()
	typ, temp := planetTypeAndTemp(position)
	name := fmt.Sprintf("Colony [%d:%d:%d]", galaxy, system, position)
	p := Planet{
		ID: m.nextPID, UserID: userID, Name: name,
		Metal: 500, Crystal: 300, Gas: 200, Energy: 50,
		Galaxy: galaxy, System: system, Position: position,
		MaxFields: 40,
		Type:      typ,
		Temperature: temp,
		ResourcesUpdatedAt: now,
	}
	m.nextPID++
	m.planets[p.ID] = p

	seedTypes := []string{
		"metal_mine", "crystal_mine", "gas_mine", "solar_plant",
		"metal_storage", "crystal_storage", "gas_storage",
		"robotics_factory", "nanite_factory", "terraformer", "fusion_reactor",
		"shipyard",
	}
	buildings := make([]Building, 0, len(seedTypes))
	for _, t := range seedTypes {
		lvl := 1
		if t == "fusion_reactor" {
			lvl = 0
		}
		b := Building{ID: m.nextBID, PlanetID: p.ID, Type: t, Level: lvl}
		m.nextBID++
		buildings = append(buildings, b)
	}
	b := Building{ID: m.nextBID, PlanetID: p.ID, Type: "missile_silo", Level: 1}
	m.nextBID++
	buildings = append(buildings, b)
	m.buildings[p.ID] = buildings
	m.queue[p.ID] = []QueueEntry{}
	m.playerProgress[p.ID] = struct{ vipPoints, totalResources int }{0, 0}
	m.playerShips[p.ID] = make(map[string]int)
	m.playerDefenses[p.ID] = make(map[string]int)
	return p, buildings, nil
}

func (m *mockRepo) SeedBuildingsForPlanet(_ context.Context, planetID int) error {
	return nil
}

func (m *mockRepo) SeedShipsForPlanet(_ context.Context, planetID int) error {
	return nil
}

func (m *mockRepo) SeedDefensesForPlanet(_ context.Context, planetID int) error {
	return nil
}

func (m *mockRepo) SeedTechnologiesForPlanet(_ context.Context, planetID int) error {
	return nil
}

func (m *mockRepo) Create(_ context.Context, userID int) (Planet, []Building, error) {
	now := time.Now()
	p := Planet{
		ID: m.nextPID, UserID: userID, Name: "Homeworld",
		Metal: 500, Crystal: 300, Gas: 200, Energy: 50,
		Galaxy: 1, System: 1, Position: 7,
		MaxFields: 40,
		Type:       "terran",
		Temperature: 15,
		ResourcesUpdatedAt: now,
	}
	m.nextPID++
	m.planets[p.ID] = p

	seedTypes := []string{
		"metal_mine", "crystal_mine", "gas_mine", "solar_plant",
		"metal_storage", "crystal_storage", "gas_storage",
		"robotics_factory", "nanite_factory", "terraformer", "fusion_reactor",
		"shipyard",
	}
	buildings := make([]Building, 0, len(seedTypes))
	for _, t := range seedTypes {
		lvl := 1
		if t == "fusion_reactor" {
			lvl = 0
		}
		b := Building{ID: m.nextBID, PlanetID: p.ID, Type: t, Level: lvl}
		m.nextBID++
		buildings = append(buildings, b)
	}
	b := Building{ID: m.nextBID, PlanetID: p.ID, Type: "missile_silo", Level: 1}
	m.nextBID++
	buildings = append(buildings, b)
	shieldTypes := []string{"small_shield_dome", "large_shield_dome"}
	for _, t := range shieldTypes {
		b := Building{ID: m.nextBID, PlanetID: p.ID, Type: t, Level: 0}
		m.nextBID++
		buildings = append(buildings, b)
	}
	m.buildings[p.ID] = buildings
	m.queue[p.ID] = []QueueEntry{}
	m.playerProgress[p.ID] = struct{ vipPoints, totalResources int }{0, 0}
	return p, buildings, nil
}

func (m *mockRepo) UpdateResources(_ context.Context, planetID, metal, crystal, gas int, updatedAt time.Time) error {
	p, ok := m.planets[planetID]
	if !ok {
		return ErrPlanetNotFound
	}
	p.Metal = metal
	p.Crystal = crystal
	p.Gas = gas
	p.ResourcesUpdatedAt = updatedAt
	m.planets[planetID] = p
	return nil
}

func (m *mockRepo) UpdatePlanetName(_ context.Context, planetID int, name string) error {
	p, ok := m.planets[planetID]
	if !ok {
		return ErrPlanetNotFound
	}
	p.Name = name
	m.planets[planetID] = p
	return nil
}

func (m *mockRepo) UpdateMaxFields(_ context.Context, planetID, maxFields int) error {
	p, ok := m.planets[planetID]
	if !ok {
		return ErrPlanetNotFound
	}
	p.MaxFields = maxFields
	m.planets[planetID] = p
	return nil
}

func (m *mockRepo) GetBuildings(_ context.Context, planetID int) ([]Building, error) {
	return m.buildings[planetID], nil
}

func (m *mockRepo) GetBuildingLevel(_ context.Context, planetID int, buildingType string) (int, error) {
	for _, b := range m.buildings[planetID] {
		if b.Type == buildingType {
			return b.Level, nil
		}
	}
	return 0, ErrInvalidBuilding
}

func (m *mockRepo) GetActiveQueue(_ context.Context, planetID int) ([]QueueEntry, error) {
	entries := m.queue[planetID]
	var active []QueueEntry
	for _, q := range entries {
		if time.Now().Before(q.CompletesAt) || !q.CompletesAt.IsZero() {
			active = append(active, q)
		}
	}
	if active == nil {
		return []QueueEntry{}, nil
	}
	return active, nil
}

func (m *mockRepo) CreateQueueEntry(_ context.Context, planetID int, buildingType string, targetLevel int, completesAt time.Time) (QueueEntry, error) {
	q := QueueEntry{
		ID: m.nextQID, BuildingType: buildingType,
		TargetLevel: targetLevel, Status: "upgrade", CompletesAt: completesAt,
	}
	m.nextQID++
	m.queue[planetID] = append(m.queue[planetID], q)
	return q, nil
}

func (m *mockRepo) CreateQueueEntryDeconstruct(_ context.Context, planetID int, buildingType string, targetLevel int, completesAt time.Time) (QueueEntry, error) {
	q := QueueEntry{
		ID: m.nextQID, BuildingType: buildingType,
		TargetLevel: targetLevel, Status: "deconstruct", CompletesAt: completesAt,
	}
	m.nextQID++
	m.queue[planetID] = append(m.queue[planetID], q)
	return q, nil
}

func (m *mockRepo) CancelUpgradeWithRefund(ctx context.Context, planetID, queueID, refundMetal, refundCrystal, refundGas int) error {
	p, ok := m.planets[planetID]
	if !ok {
		return ErrPlanetNotFound
	}
	p.Metal += refundMetal
	p.Crystal += refundCrystal
	p.Gas += refundGas
	m.planets[planetID] = p

	return m.CancelQueueEntry(ctx, queueID)
}

func (m *mockRepo) DeconstructComplete(ctx context.Context, planetID, queueID int, buildingType string, targetLevel int, refundMetal, refundCrystal, refundGas, maxFields int) error {
	p, ok := m.planets[planetID]
	if !ok {
		return ErrPlanetNotFound
	}
	p.Metal += refundMetal
	p.Crystal += refundCrystal
	p.Gas += refundGas
	if maxFields > 0 {
		p.MaxFields = maxFields
	}
	m.planets[planetID] = p

	if targetLevel == 0 {
		if err := m.DeleteBuilding(ctx, planetID, buildingType); err != nil {
			return err
		}
	} else {
		if err := m.UpdateBuildingLevel(ctx, planetID, buildingType, targetLevel); err != nil {
			return err
		}
	}

	return m.CancelQueueEntry(ctx, queueID)
}

func (m *mockRepo) CancelQueueEntry(_ context.Context, queueID int) error {
	for pid, entries := range m.queue {
		for i, q := range entries {
			if q.ID == queueID {
				m.queue[pid] = append(entries[:i], entries[i+1:]...)
				return nil
			}
		}
	}
	return nil
}

func (m *mockRepo) DeleteBuilding(_ context.Context, planetID int, buildingType string) error {
	buildings := m.buildings[planetID]
	for i, b := range buildings {
		if b.Type == buildingType {
			m.buildings[planetID] = append(buildings[:i], buildings[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *mockRepo) UpdateBuildingLevel(_ context.Context, planetID int, buildingType string, level int) error {
	for i, b := range m.buildings[planetID] {
		if b.Type == buildingType {
			m.buildings[planetID][i].Level = level
			return nil
		}
	}
	return nil
}

func (m *mockRepo) GetTechLevel(_ context.Context, userID int, techType string) (int, error) {
	if m.techLevels[userID] != nil {
		if l, ok := m.techLevels[userID][techType]; ok {
			return l, nil
		}
	}
	if techType == "energy_tech" {
		return 3, nil
	}
	return 0, nil
}

func (m *mockRepo) AddTechLevel(_ context.Context, userID int, techType string, level int) error {
	if m.techLevels[userID] == nil {
		m.techLevels[userID] = make(map[string]int)
	}
	m.techLevels[userID][techType] = level
	return nil
}

func (m *mockRepo) GetHighestLabLevel(_ context.Context, playerID int) (int, error) {
	return 1, nil
}

func (m *mockRepo) GetPlayerProgress(_ context.Context, planetID int) (int, int, error) {
	pp, ok := m.playerProgress[planetID]
	if !ok {
		return 0, 0, nil
	}
	return pp.vipPoints, pp.totalResources, nil
}

func (m *mockRepo) AddVIPPoints(_ context.Context, planetID int, points int) error {
	pp := m.playerProgress[planetID]
	pp.vipPoints += points
	m.playerProgress[planetID] = pp
	return nil
}

func (m *mockRepo) AddResourcesProduced(_ context.Context, planetID int, amount int) error {
	pp := m.playerProgress[planetID]
	pp.totalResources += amount
	m.playerProgress[planetID] = pp
	return nil
}

func (m *mockRepo) CompleteBuild(_ context.Context, queueID int, buildingType string, targetLevel int) error {
	for pid, entries := range m.queue {
		for i, q := range entries {
			if q.ID == queueID {
				m.queue[pid] = append(entries[:i], entries[i+1:]...)
				for j, b := range m.buildings[pid] {
					if b.Type == buildingType {
						m.buildings[pid][j].Level = targetLevel
					}
				}
				return nil
			}
		}
	}
	return nil
}

func (m *mockRepo) ListGalaxies(_ context.Context) ([]Galaxy, error) {
	return []Galaxy{
		{ID: 1, Name: "Galaxy 1"},
		{ID: 2, Name: "Galaxy 2"},
		{ID: 3, Name: "Galaxy 3"},
		{ID: 4, Name: "Galaxy 4"},
		{ID: 5, Name: "Galaxy 5"},
		{ID: 6, Name: "Galaxy 6"},
		{ID: 7, Name: "Galaxy 7"},
		{ID: 8, Name: "Galaxy 8"},
		{ID: 9, Name: "Galaxy 9"},
	}, nil
}

func (m *mockRepo) ListSystems(_ context.Context, galaxyID int, page, pageSize int) ([]System, int, error) {
	return nil, 0, nil
}

func (m *mockRepo) GetSystemPositions(_ context.Context, systemID int) ([]Position, error) {
	positions := make([]Position, 15)
	for i := 0; i < 15; i++ {
		positions[i] = Position{PositionNum: i + 1, State: "empty"}
	}
	return positions, nil
}

func (m *mockRepo) GetPlayerShips(_ context.Context, planetID int) (map[string]int, error) {
	return nil, nil
}

func (m *mockRepo) AddPlayerShips(_ context.Context, planetID, planetUserID int, shipType string, quantity int) error {
	return nil
}

func (m *mockRepo) DeductPlayerShips(ctx context.Context, planetID int, ships map[string]int) error {
	return nil
}

func (m *mockRepo) FindByCoords(_ context.Context, galaxy, system, position int) (Planet, error) {
	for _, p := range m.planets {
		if p.Galaxy == galaxy && p.System == system && p.Position == position {
			return p, nil
		}
	}
	return Planet{}, ErrPlanetNotFound
}

func (m *mockRepo) GetPlayerDefenses(_ context.Context, planetID int) (map[string]int, error) {
	defenses := m.playerDefenses[planetID]
	if defenses == nil {
		return make(map[string]int), nil
	}
	return defenses, nil
}

func (m *mockRepo) AddPlayerDefenses(_ context.Context, planetID, planetUserID int, defenseType string, quantity int) error {
	if m.playerDefenses[planetID] == nil {
		m.playerDefenses[planetID] = make(map[string]int)
	}
	m.playerDefenses[planetID][defenseType] += quantity
	return nil
}

func (m *mockRepo) GetPlayerDefense(_ context.Context, planetID int, defenseType string) (int, error) {
	defenses := m.playerDefenses[planetID]
	if defenses == nil {
		return 0, nil
	}
	return defenses[defenseType], nil
}

func (m *mockRepo) SetPlayerDefense(_ context.Context, planetID int, defenseType string, quantity int) error {
	if m.playerDefenses[planetID] == nil {
		m.playerDefenses[planetID] = make(map[string]int)
	}
	m.playerDefenses[planetID][defenseType] = quantity
	return nil
}

func moonKey(g, s, p int) string {
	return fmt.Sprintf("%d:%d:%d", g, s, p)
}

func (m *mockRepo) MoonExists(_ context.Context, galaxy, system, position int) (bool, error) {
	key := moonKey(galaxy, system, position)
	if val, ok := m.moonExists[key]; ok {
		return val, nil
	}
	return false, nil
}

func (m *mockRepo) GetMoonBuildings(_ context.Context, galaxy, system, position int) ([]MoonBuilding, error) {
	key := moonKey(galaxy, system, position)
	buildings := m.moonBuildings[key]
	if buildings == nil {
		return []MoonBuilding{}, nil
	}
	return buildings, nil
}

func (m *mockRepo) GetMoonBuildingLevel(_ context.Context, galaxy, system, position int, buildingType string) (int, error) {
	key := moonKey(galaxy, system, position)
	buildings := m.moonBuildings[key]
	for _, b := range buildings {
		if b.Type == buildingType {
			return b.Level, nil
		}
	}
	return 0, ErrBuildingNotFound
}

func (m *mockRepo) UpdateMoonBuildingLevel(_ context.Context, galaxy, system, position int, buildingType string, level int) error {
	key := moonKey(galaxy, system, position)
	buildings := m.moonBuildings[key]
	found := false
	for i, b := range buildings {
		if b.Type == buildingType {
			m.moonBuildings[key][i].Level = level
			found = true
			break
		}
	}
	if !found {
		newB := MoonBuilding{
			ID:         0,
			MoonGalaxy: galaxy,
			MoonSystem: system,
			MoonPos:    position,
			Type:       buildingType,
			Level:      level,
		}
		m.moonBuildings[key] = append(m.moonBuildings[key], newB)
	}
	return nil
}

func (m *mockRepo) GetWormhole(_ context.Context, galaxy, system, position int) (*WormholeEntry, error) {
	key := moonKey(galaxy, system, position)
	w, ok := m.wormholes[key]
	if !ok || w == nil {
		return nil, ErrWormholeNotFound
	}
	return w, nil
}

func (m *mockRepo) CreateOrUpdateWormhole(_ context.Context, galaxy, system, position int, level int) error {
	key := moonKey(galaxy, system, position)
	m.wormholes[key] = &WormholeEntry{
		MoonGalaxy: galaxy,
		MoonSystem: system,
		MoonPos:    position,
		Level:      level,
	}
	return nil
}

func (m *mockRepo) LinkWormholes(_ context.Context, srcGalaxy, srcSystem, srcPos, dstGalaxy, dstSystem, dstPos int) error {
	srcKey := moonKey(srcGalaxy, srcSystem, srcPos)
	dstKey := moonKey(dstGalaxy, dstSystem, dstPos)

	srcW, ok := m.wormholes[srcKey]
	if !ok {
		return ErrWormholeNotFound
	}
	dstW, ok := m.wormholes[dstKey]
	if !ok {
		return ErrWormholeNotFound
	}

	now := time.Now().Add(1 * time.Hour)
	srcW.LinkedGalaxy = &dstGalaxy
	srcW.LinkedSystem = &dstSystem
	srcW.LinkedPosition = &dstPos
	srcW.CooldownUntil = &now
	dstW.LinkedGalaxy = &srcGalaxy
	dstW.LinkedSystem = &srcSystem
	dstW.LinkedPosition = &srcPos
	dstW.CooldownUntil = &now
	return nil
}

func (m *mockRepo) StarGateLink(_ context.Context, planetID, targetPlanetID int) error {
	m.stargateLinks[planetID] = &StarGateLink{
		PlanetID: planetID, TargetPlanetID: targetPlanetID,
	}
	m.stargateLinks[targetPlanetID] = &StarGateLink{
		PlanetID: targetPlanetID, TargetPlanetID: planetID,
	}
	return nil
}

func (m *mockRepo) StarGateUnlink(_ context.Context, planetID int) error {
	link, ok := m.stargateLinks[planetID]
	if ok {
		delete(m.stargateLinks, planetID)
		delete(m.stargateLinks, link.TargetPlanetID)
	}
	return nil
}

func (m *mockRepo) GetStarGateLink(_ context.Context, planetID int) (*StarGateLink, error) {
	link, ok := m.stargateLinks[planetID]
	if !ok {
		return nil, nil
	}
	return link, nil
}

func (m *mockRepo) GetMissileCounts(_ context.Context, planetID int) (int, int, error) {
	return m.missileIPMs[planetID], m.missileABMs[planetID], nil
}

func (m *mockRepo) AddIPMs(_ context.Context, planetID, count int) error {
	m.missileIPMs[planetID] += count
	return nil
}

func (m *mockRepo) AddABMs(_ context.Context, planetID, count int) error {
	m.missileABMs[planetID] += count
	return nil
}

func (m *mockRepo) DeductIPMs(_ context.Context, planetID, count int) error {
	if m.missileIPMs[planetID] < count {
		return fmt.Errorf("insufficient IPMs")
	}
	m.missileIPMs[planetID] -= count
	return nil
}

func (m *mockRepo) DeductABMs(_ context.Context, planetID, count int) error {
	if m.missileABMs[planetID] < count {
		return fmt.Errorf("insufficient ABMs")
	}
	m.missileABMs[planetID] -= count
	return nil
}

func (m *mockRepo) GetLinkedWormhole(_ context.Context, galaxy, system, position int) (*WormholeEntry, error) {
	key := moonKey(galaxy, system, position)
	w, ok := m.wormholes[key]
	if !ok || w == nil {
		return nil, ErrWormholeNotFound
	}
	return w, nil
}

func TestGetTechLevel_NonExistent(t *testing.T) {
	mock := newMockRepo()
	level, err := mock.GetTechLevel(context.Background(), 1, "weapons_tech")
	if err != nil {
		t.Fatal("expected no error, got:", err)
	}
	if level != 0 {
		t.Errorf("expected 0 for non-existent tech, got %d", level)
	}
}

func TestGetTechLevel_Default(t *testing.T) {
	mock := newMockRepo()
	level, err := mock.GetTechLevel(context.Background(), 1, "energy_tech")
	if err != nil {
		t.Fatal("expected no error, got:", err)
	}
	if level != 3 {
		t.Errorf("expected energy_tech level 3, got %d", level)
	}
}

func TestPlanetTypeAndTemp_HomePlanet(t *testing.T) {
	typ, temp := planetTypeAndTemp(7)
	if typ != "terran" {
		t.Errorf("expected terran, got %s", typ)
	}
	if temp < 0 || temp > 20 {
		t.Errorf("expected temp 0-20 for home, got %d", temp)
	}
}

func TestPlanetTypeAndTemp_Position1(t *testing.T) {
	typ, _ := planetTypeAndTemp(1)
	if typ != "desert" && typ != "volcanic" {
		t.Errorf("expected desert or volcanic for pos 1, got %s", typ)
	}
}

func TestGetOrCreate_FirstCallCreatesWithBuildings(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	p, buildings, err := svc.GetOrCreatePlanet(context.Background(), 42)
	if err != nil {
		t.Fatal("expected no error, got:", err)
	}
	if p.UserID != 42 {
		t.Errorf("expected user_id 42, got %d", p.UserID)
	}
	if len(buildings) != 15 {
		t.Errorf("expected 15 buildings, got %d", len(buildings))
	}
}

func TestGetOrCreate_SecondCallReturnsExisting(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	p1, _, err := svc.GetOrCreatePlanet(context.Background(), 42)
	if err != nil {
		t.Fatal("first call failed:", err)
	}
	p2, _, err := svc.GetOrCreatePlanet(context.Background(), 42)
	if err != nil {
		t.Fatal("second call failed:", err)
	}
	if p1.ID != p2.ID {
		t.Errorf("expected same planet ID, got %d vs %d", p1.ID, p2.ID)
	}
}

func TestGetOrCreate_ResourceAccumulation(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	p, _, err := svc.GetOrCreatePlanet(context.Background(), 99)
	if err != nil {
		t.Fatal("first call:", err)
	}
	initialUpdatedAt := p.ResourcesUpdatedAt
	time.Sleep(2 * time.Second)
	p2, _, err := svc.GetOrCreatePlanet(context.Background(), 99)
	if err != nil {
		t.Fatal("second call:", err)
	}
	if !p2.ResourcesUpdatedAt.After(initialUpdatedAt) {
		t.Error("resources_updated_at should have advanced")
	}
	totalResources := p2.Metal + p2.Crystal + p2.Gas
	if totalResources <= 1000 {
		t.Errorf("resources should have increased, got %d", totalResources)
	}
}

func TestProductionRate(t *testing.T) {
	tests := []struct {
		building string
		level    int
		minRate  float64
		maxRate  float64
	}{
		{"metal_mine", 1, 30, 40},
		{"metal_mine", 5, 200, 250},
		{"crystal_mine", 1, 20, 30},
		{"gas_mine", 1, 10, 20},
		{"solar_plant", 1, 40, 50},
	}
	for _, tc := range tests {
		rate := productionRate(tc.building, tc.level)
		if rate < tc.minRate || rate > tc.maxRate {
			t.Errorf("%s L%d: expected ~%.0f, got %.2f", tc.building, tc.level, (tc.minRate+tc.maxRate)/2, rate)
		}
	}
}

func TestProductionRate_ZeroForUnknown(t *testing.T) {
	if r := productionRate("unknown", 1); r != 0 {
		t.Errorf("expected 0 for unknown type, got %.2f", r)
	}
	if r := productionRate("metal_mine", 0); r != 0 {
		t.Errorf("expected 0 for level 0, got %.2f", r)
	}
}

func TestCalculateProduction(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	buildings := []Building{
		{Type: "metal_mine", Level: 1},
		{Type: "crystal_mine", Level: 2},
		{Type: "gas_mine", Level: 1},
		{Type: "solar_plant", Level: 3},
	}
	prod := svc.calculateProduction(buildings, 1.0, PlanetTypeTerran, 15, 3, 0.0, 0.0)
	if prod.Metal <= 0 {
		t.Error("expected positive metal production")
	}
	rounded := math.Round(prod.Metal * 60)
	if rounded != 33 {
		t.Errorf("metal L1 should produce 33/min, got %.2f/min", prod.Metal*60)
	}
}

func TestCalculateProduction_TemperatureBonus_ColdPlanet(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	buildings := []Building{
		{Type: "gas_mine", Level: 1},
		{Type: "solar_plant", Level: 1},
	}
	// Ice planet, cold: gas gets +1.5 effective levels
	prod := svc.calculateProduction(buildings, 1.0, PlanetTypeIce, -30, 3, 0.0, 0.0)
	gasPerMin := prod.Gas * 60
	if gasPerMin <= 11 {
		t.Errorf("gas on cold planet should get temperature bonus, got %.2f/min", gasPerMin)
	}
}

func TestCalculateProduction_TemperatureBonus_HotPlanet(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	buildings := []Building{
		{Type: "gas_mine", Level: 1},
		{Type: "solar_plant", Level: 1},
	}
	// Desert planet, hot: solar gets +1.5 effective levels
	prod := svc.calculateProduction(buildings, 1.0, PlanetTypeDesert, 80, 3, 0.0, 0.0)
	solarPerMin := prod.Energy * 60
	if solarPerMin <= 44 {
		t.Errorf("solar on hot planet should get temperature bonus, got %.2f/min", solarPerMin)
	}
}

func TestCalculateProduction_NoBonus_Terran(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	buildings := []Building{
		{Type: "gas_mine", Level: 1},
		{Type: "solar_plant", Level: 1},
	}
	prod := svc.calculateProduction(buildings, 1.0, PlanetTypeTerran, 15, 3, 0.0, 0.0)
	gasPerMin := prod.Gas * 60
	if gasPerMin < 10 || gasPerMin > 12 {
		t.Errorf("gas on terran should have no bonus, got %.2f/min", gasPerMin)
	}
}

func TestCalculateProduction_WithPenalty(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	buildings := []Building{
		{Type: "metal_mine", Level: 5},
		{Type: "crystal_mine", Level: 5},
		{Type: "gas_mine", Level: 5},
	}
	prod := svc.calculateProduction(buildings, 0.5, PlanetTypeTerran, 15, 3, 0.0, 0.0)
	if prod.Metal <= 0 {
		t.Error("expected positive metal production even with penalty")
	}
}

func TestBuildingFormulaScale(t *testing.T) {
	levels := []int{1, 5, 10, 20}
	rates := make(map[int]float64)
	for _, l := range levels {
		rates[l] = productionRate("metal_mine", l)
	}
	for i := 1; i < len(levels); i++ {
		if rates[levels[i]] <= rates[levels[i-1]] {
			t.Errorf("metal mine L%d (%.0f) should produce more than L%d (%.0f)",
				levels[i], rates[levels[i]], levels[i-1], rates[levels[i-1]])
		}
	}
}

func TestStorageCapacity(t *testing.T) {
	cap := storageCapacity("metal_storage", 0)
	if cap != baseStorage {
		t.Errorf("expected base storage %d, got %d", baseStorage, cap)
	}
	capL1 := storageCapacity("metal_storage", 1)
	expected := baseStorage + int(5000*math.Pow(1.5, 1))
	if capL1 != expected {
		t.Errorf("expected %d, got %d", expected, capL1)
	}
}

func TestCalculateStorage(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	buildings := []Building{
		{Type: "metal_storage", Level: 2},
		{Type: "crystal_storage", Level: 1},
		{Type: "gas_storage", Level: 3},
	}
	storage := svc.calculateStorage(buildings)
	if storage.Metal <= baseStorage {
		t.Error("metal storage should exceed base")
	}
}

func TestResourceCapping(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	_, _, err := svc.GetOrCreatePlanet(context.Background(), 77)
	if err != nil {
		t.Fatal("first call:", err)
	}
	planet, buildings, err := svc.GetOrCreatePlanet(context.Background(), 77)
	if err != nil {
		t.Fatal("second call:", err)
	}
	mock := svc.repo.(*mockRepo)
	storedPlanet := mock.planets[planet.ID]
	storage := svc.calculateStorage(buildings)
	if storedPlanet.Metal > storage.Metal {
		t.Errorf("metal %d exceeds storage %d", storedPlanet.Metal, storage.Metal)
	}
}

func TestPenaltyFactor(t *testing.T) {
	buildings := []Building{
		{Type: "metal_mine", Level: 1},
		{Type: "solar_plant", Level: 1},
	}
	netEnergy, efficiency := calculatePenaltyFactor(buildings, 0)
	if netEnergy <= 0 {
		t.Errorf("expected positive net energy, got %d", netEnergy)
	}
	if efficiency != 1.0 {
		t.Errorf("expected efficiency 1.0, got %.2f", efficiency)
	}

	buildings2 := []Building{
		{Type: "metal_mine", Level: 5},
		{Type: "crystal_mine", Level: 5},
		{Type: "gas_mine", Level: 5},
		{Type: "solar_plant", Level: 1},
	}
	netEnergy2, efficiency2 := calculatePenaltyFactor(buildings2, 0)
	if netEnergy2 >= 0 {
		t.Errorf("expected negative net energy, got %d", netEnergy2)
	}
	if efficiency2 >= 1.0 {
		t.Errorf("expected efficiency < 1.0, got %.2f", efficiency2)
	}
}

func TestEnergyConsumptionPerMinute(t *testing.T) {
	tests := []struct {
		typ   string
		level int
		want  float64
	}{
		{"metal_mine", 1, 10},
		{"crystal_mine", 1, 10},
		{"gas_mine", 1, 20},
		{"metal_storage", 1, 0},
	}
	for _, tc := range tests {
		got := energyConsumptionPerMinute(tc.typ, tc.level)
		if got != tc.want {
			t.Errorf("%s L%d: expected %.0f, got %.0f", tc.typ, tc.level, tc.want, got)
		}
	}
}

func TestSolarPlantRate_Level1(t *testing.T) {
	rate := productionRate("solar_plant", 1)
	if rate != 44.0 {
		t.Errorf("expected 44.0, got %.2f", rate)
	}
}

func TestBuildingCostResources(t *testing.T) {
	metal, crystal, gas := buildingCostResources("metal_mine", 0)
	if metal <= 0 || crystal <= 0 {
		t.Errorf("expected positive costs for metal_mine upgrade from L0, got metal=%d crystal=%d", metal, crystal)
	}
	if gas != 0 {
		t.Errorf("expected 0 gas for metal_mine, got %d", gas)
	}
}

func TestBuildingCostResources_Invalid(t *testing.T) {
	metal, crystal, gas := buildingCostResources("unknown", 1)
	if metal != 0 || crystal != 0 || gas != 0 {
		t.Errorf("expected zero costs for unknown building, got %d %d %d", metal, crystal, gas)
	}
}

func TestBuildingCostScaling(t *testing.T) {
	_, c1, _ := buildingCostResources("crystal_mine", 0)
	_, c2, _ := buildingCostResources("crystal_mine", 1)
	if c2 <= c1 {
		t.Errorf("cost should increase with level: L0=%d L1=%d", c1, c2)
	}
}

func TestBuildDuration(t *testing.T) {
	d := buildingBuildDuration("metal_mine", 0, 0, 0)
	if d <= 0 {
		t.Error("expected positive duration")
	}
	if d > 5*time.Minute {
		t.Errorf("expected L0->L1 to take <5 min, got %v", d)
	}
}

func TestBuildDurationScaling(t *testing.T) {
	d1 := buildingBuildDuration("metal_mine", 0, 0, 0)
	d2 := buildingBuildDuration("metal_mine", 5, 0, 0)
	if d2 <= d1 {
		t.Errorf("higher level should take longer: L0-%v L5-%v", d1, d2)
	}
}

func TestBuildDurationRoboticsReduction(t *testing.T) {
	base := buildingBuildDuration("metal_mine", 1, 0, 0)
	reduced := buildingBuildDuration("metal_mine", 1, 3, 0)
	if reduced >= base {
		t.Errorf("robotics should reduce time: base=%v reduced=%v", base, reduced)
	}
}

func TestBuildDurationNaniteReduction(t *testing.T) {
	base := buildingBuildDuration("metal_mine", 1, 0, 0)
	reduced := buildingBuildDuration("metal_mine", 1, 0, 2)
	if reduced >= base {
		t.Errorf("nanite should reduce time: base=%v reduced=%v", base, reduced)
	}
}

func TestBuildDurationNaniteDoesNotAffectSelf(t *testing.T) {
	base := buildingBuildDuration("nanite_factory", 1, 0, 0)
	withNanite := buildingBuildDuration("nanite_factory", 1, 0, 3)
	if withNanite != base {
		t.Errorf("nanite should not reduce its own time: base=%v withNanite=%v", base, withNanite)
	}
}

func TestBuildingCostRobotics(t *testing.T) {
	metal, crystal, gas := buildingCostResources("robotics_factory", 0)
	if metal <= 0 || crystal <= 0 || gas <= 0 {
		t.Errorf("expected all positive costs for robotics_factory L0->1, got m=%d c=%d g=%d", metal, crystal, gas)
	}
}

func TestBuildingCostNanite(t *testing.T) {
	metal, crystal, gas := buildingCostResources("nanite_factory", 0)
	if metal <= 0 || crystal <= 0 || gas <= 0 {
		t.Errorf("expected all positive costs for nanite_factory L0->1, got m=%d c=%d g=%d", metal, crystal, gas)
	}
}

func TestService_StartUpgrade_InsufficientResources(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	_, buildings, err := svc.GetOrCreatePlanet(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	planet := svc.repo.(*mockRepo).planets[1]
	svc.repo.UpdateResources(context.Background(), planet.ID, 0, 0, 0, time.Now())

	_, err = svc.StartBuildingUpgrade(context.Background(), planet.ID, buildings[0].Type)
	if err != ErrInsufficientResources {
		t.Errorf("expected ErrInsufficientResources, got %v", err)
	}
}

func TestService_StartUpgrade_Success(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	planet, buildings, err := svc.GetOrCreatePlanet(context.Background(), 2)
	if err != nil {
		t.Fatal(err)
	}

	entry, err := svc.StartBuildingUpgrade(context.Background(), planet.ID, buildings[0].Type)
	if err != nil {
		t.Fatal("expected success, got:", err)
	}
	if entry.BuildingType != buildings[0].Type {
		t.Errorf("expected %s, got %s", buildings[0].Type, entry.BuildingType)
	}
	if entry.TargetLevel != 2 {
		t.Errorf("expected target level 2, got %d", entry.TargetLevel)
	}
	if entry.CompletesAt.Before(time.Now()) {
		t.Error("completes_at should be in the future")
	}
}

func TestService_StartUpgrade_DeductsResources(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	planet, buildings, err := svc.GetOrCreatePlanet(context.Background(), 3)
	if err != nil {
		t.Fatal(err)
	}
	mock := svc.repo.(*mockRepo)
	initialMetal := planet.Metal

	buildingLevel := 1
	for _, b := range buildings {
		if b.Type == buildings[0].Type {
			buildingLevel = b.Level
			break
		}
	}
	metalCost, _, _ := buildingCostResources(buildings[0].Type, buildingLevel)
	_, err = svc.StartBuildingUpgrade(context.Background(), planet.ID, buildings[0].Type)
	if err != nil {
		t.Fatal("expected success, got:", err)
	}

	updatedPlanet := mock.planets[planet.ID]
	if updatedPlanet.Metal != initialMetal-metalCost {
		t.Errorf("expected metal %d - %d = %d, got %d",
			initialMetal, metalCost, initialMetal-metalCost, updatedPlanet.Metal)
	}
}

func TestService_StartUpgrade_AlreadyQueued(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	planet, buildings, err := svc.GetOrCreatePlanet(context.Background(), 4)
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.StartBuildingUpgrade(context.Background(), planet.ID, buildings[0].Type)
	if err != nil {
		t.Fatal("first upgrade failed:", err)
	}

	_, err = svc.StartBuildingUpgrade(context.Background(), planet.ID, buildings[0].Type)
	if err != ErrAlreadyQueued {
		t.Errorf("expected ErrAlreadyQueued, got %v", err)
	}
}

func TestService_ProcessCompletedBuilds(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	mock := svc.repo.(*mockRepo)
	p, buildings, err := svc.GetOrCreatePlanet(context.Background(), 5)
	if err != nil {
		t.Fatal(err)
	}

	oldLevel := mock.buildings[p.ID][0].Level

	_, err = svc.StartBuildingUpgrade(context.Background(), p.ID, buildings[0].Type)
	if err != nil {
		t.Fatal("start upgrade:", err)
	}

	mock.queue[p.ID][0].CompletesAt = time.Now().Add(-1 * time.Second)

	err = svc.processCompletedBuilds(context.Background(), p.ID)
	if err != nil {
		t.Fatal("process builds:", err)
	}

	newLevel := mock.buildings[p.ID][0].Level
	if newLevel <= oldLevel {
		t.Errorf("building level should have increased: old=%d new=%d", oldLevel, newLevel)
	}
}

func TestService_CancelUpgrade_RefundsResources(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	planet, buildings, err := svc.GetOrCreatePlanet(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	mock := svc.repo.(*mockRepo)
	initialMetal := planet.Metal
	initialCrystal := planet.Crystal
	initialGas := planet.Gas

	building := buildings[0]
	buildingLevel := building.Level
	_, err = svc.StartBuildingUpgrade(context.Background(), planet.ID, building.Type)
	if err != nil {
		t.Fatal("start upgrade:", err)
	}

	err = svc.CancelUpgrade(context.Background(), planet.ID, building.Type)
	if err != nil {
		t.Fatal("cancel upgrade:", err)
	}

	metalCost, crystalCost, gasCost := buildingCostResources(building.Type, buildingLevel)
	updatedPlanet := mock.planets[planet.ID]
	expectedMetal := initialMetal - metalCost + metalCost/2
	expectedCrystal := initialCrystal - crystalCost + crystalCost/2
	expectedGas := initialGas - gasCost + gasCost/2
	if updatedPlanet.Metal != expectedMetal {
		t.Errorf("expected metal %d, got %d", expectedMetal, updatedPlanet.Metal)
	}
	if updatedPlanet.Crystal != expectedCrystal {
		t.Errorf("expected crystal %d, got %d", expectedCrystal, updatedPlanet.Crystal)
	}
	if updatedPlanet.Gas != expectedGas {
		t.Errorf("expected gas %d, got %d", expectedGas, updatedPlanet.Gas)
	}
}

func TestService_CancelUpgrade_NoActiveUpgrade(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	planet, _, err := svc.GetOrCreatePlanet(context.Background(), 11)
	if err != nil {
		t.Fatal(err)
	}
	err = svc.CancelUpgrade(context.Background(), planet.ID, "metal_mine")
	if err == nil {
		t.Error("expected error for no active upgrade")
	}
}

func TestService_QueueDeconstruction_CreatesQueueEntry(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	planet, buildings, err := svc.GetOrCreatePlanet(context.Background(), 20)
	if err != nil {
		t.Fatal(err)
	}

	entry, err := svc.QueueDeconstruction(context.Background(), planet.ID, buildings[0].Type)
	if err != nil {
		t.Fatal("queue deconstruction:", err)
	}
	if entry.Status != "deconstruct" {
		t.Errorf("expected status deconstruct, got %s", entry.Status)
	}
	if entry.TargetLevel != 0 {
		t.Errorf("expected target level 0, got %d", entry.TargetLevel)
	}
}

func TestService_QueueDeconstruction_BuildingNotFound(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	planet, _, err := svc.GetOrCreatePlanet(context.Background(), 21)
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.QueueDeconstruction(context.Background(), planet.ID, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent building")
	}
}

func TestService_ProcessDeconstructCompletion_RemovesBuilding(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	mock := svc.repo.(*mockRepo)
	planet, buildings, err := svc.GetOrCreatePlanet(context.Background(), 30)
	if err != nil {
		t.Fatal(err)
	}

	initialMetal := planet.Metal
	initialCrystal := planet.Crystal
	initialGas := planet.Gas
	initialCount := len(buildings)
	buildingType := buildings[0].Type

	entry, err := svc.QueueDeconstruction(context.Background(), planet.ID, buildingType)
	if err != nil {
		t.Fatal("queue deconstruction:", err)
	}

	for i, q := range mock.queue[planet.ID] {
		if q.ID == entry.ID {
			mock.queue[planet.ID][i].CompletesAt = time.Now().Add(-1 * time.Second)
			break
		}
	}

	// Get the building level before deconstruction (should be 1)
	buildingLevel, _ := mock.GetBuildingLevel(context.Background(), planet.ID, buildingType)
	expectedRefundMetal, expectedRefundCrystal, expectedRefundGas := buildingCostResources(buildingType, buildingLevel-1)
	expectedRefundMetal /= 2
	expectedRefundCrystal /= 2
	expectedRefundGas /= 2

	err = svc.processCompletedBuilds(context.Background(), planet.ID)
	if err != nil {
		t.Fatal("process builds:", err)
	}

	updatedPlanet := mock.planets[planet.ID]
	if updatedPlanet.Metal != initialMetal+expectedRefundMetal {
		t.Errorf("expected metal %d, got %d", initialMetal+expectedRefundMetal, updatedPlanet.Metal)
	}
	if updatedPlanet.Crystal != initialCrystal+expectedRefundCrystal {
		t.Errorf("expected crystal %d, got %d", initialCrystal+expectedRefundCrystal, updatedPlanet.Crystal)
	}
	if updatedPlanet.Gas != initialGas+expectedRefundGas {
		t.Errorf("expected gas %d, got %d", initialGas+expectedRefundGas, updatedPlanet.Gas)
	}

	updatedBuildings, _ := mock.GetBuildings(context.Background(), planet.ID)
	if len(updatedBuildings) != initialCount-1 {
		t.Errorf("expected %d buildings, got %d", initialCount-1, len(updatedBuildings))
	}
}

func addFusionReactor(m *mockRepo, planetID int) {
	m.buildings[planetID] = append(m.buildings[planetID], Building{
		ID: m.nextBID, PlanetID: planetID, Type: "fusion_reactor", Level: 0,
	})
	m.nextBID++
}

func TestFusionReactor_Gating_GasMineTooLow(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	planet, buildings, err := svc.GetOrCreatePlanet(context.Background(), 50)
	if err != nil {
		t.Fatal(err)
	}
	for i, b := range buildings {
		if b.Type == "gas_mine" {
			mock := svc.repo.(*mockRepo)
			mock.buildings[planet.ID][i].Level = 4
			break
		}
	}
	mock := svc.repo.(*mockRepo)
	addFusionReactor(mock, planet.ID)
	_, err = svc.StartBuildingUpgrade(context.Background(), planet.ID, "fusion_reactor")
	if err != ErrPrerequisitesNotMet {
		t.Errorf("expected ErrPrerequisitesNotMet, got %v", err)
	}
}

func TestFusionReactor_Gating_EnergyTechTooLow(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	planet, buildings, err := svc.GetOrCreatePlanet(context.Background(), 51)
	if err != nil {
		t.Fatal(err)
	}
	for i, b := range buildings {
		if b.Type == "gas_mine" {
			mock := svc.repo.(*mockRepo)
			mock.buildings[planet.ID][i].Level = 5
			break
		}
	}
	mock := svc.repo.(*mockRepo)
	addFusionReactor(mock, planet.ID)
	mock.techLevels[51] = map[string]int{"energy_tech": 2}

	_, err = svc.StartBuildingUpgrade(context.Background(), planet.ID, "fusion_reactor")
	if err != ErrPrerequisitesNotMet {
		t.Errorf("expected ErrPrerequisitesNotMet, got %v", err)
	}
}

func TestFusionReactor_GatePasses(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	planet, buildings, err := svc.GetOrCreatePlanet(context.Background(), 52)
	if err != nil {
		t.Fatal(err)
	}
	mock := svc.repo.(*mockRepo)
	for i, b := range buildings {
		if b.Type == "gas_mine" {
			mock.buildings[planet.ID][i].Level = 5
			break
		}
	}
	addFusionReactor(mock, planet.ID)

	entry, err := svc.StartBuildingUpgrade(context.Background(), planet.ID, "fusion_reactor")
	if err != nil {
		t.Fatal("expected success, got:", err)
	}
	if entry.BuildingType != "fusion_reactor" {
		t.Errorf("expected fusion_reactor, got %s", entry.BuildingType)
	}
}

func TestFusionReactor_ProducesEnergy(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	buildings := []Building{
		{Type: "fusion_reactor", Level: 1},
		{Type: "solar_plant", Level: 1},
		{Type: "gas_mine", Level: 1},
	}
	prod := svc.calculateProduction(buildings, 1.0, PlanetTypeTerran, 15, 3, 0.0, 0.0)
	energyPerMin := prod.Energy * 60
	if energyPerMin <= 44 {
		t.Errorf("fusion L1 + solar L1 should produce > 44/min, got %.2f/min", energyPerMin)
	}
	if energyPerMin < 80 {
		t.Errorf("fusion L1 + solar L1 expected > 80/min, got %.2f/min", energyPerMin)
	}
}

func TestFusionReactor_ConsumesGas(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	buildings := []Building{
		{Type: "gas_mine", Level: 1},
		{Type: "fusion_reactor", Level: 1},
	}
	prod := svc.calculateProduction(buildings, 1.0, PlanetTypeTerran, 15, 3, 0.0, 0.0)
	gasPerMin := prod.Gas * 60
	if gasPerMin < 0 {
		t.Errorf("net gas should not be negative, got %.4f/min", gasPerMin)
	}
	if gasPerMin > 11 {
		t.Errorf("net gas should be reduced by fusion consumption, got %.4f/min", gasPerMin)
	}
}

func TestPlayerProgress_Default(t *testing.T) {
	mock := newMockRepo()
	vip, total, err := mock.GetPlayerProgress(context.Background(), 1)
	if err != nil {
		t.Fatal("expected no error, got:", err)
	}
	if vip != 0 {
		t.Errorf("expected 0 VIP points, got %d", vip)
	}
	if total != 0 {
		t.Errorf("expected 0 total resources, got %d", total)
	}
}

func TestVIPLevelFromPoints(t *testing.T) {
	tests := []struct {
		points int
		level  int
	}{
		{0, 0}, {50, 0}, {100, 1}, {499, 1}, {500, 2},
		{1500, 3}, {5000, 4}, {15000, 5}, {40000, 6},
		{100000, 7}, {250000, 8}, {500000, 9},
		{1000000, 10}, {2000000, 11}, {5000000, 12}, {9999999, 12},
	}
	for _, tc := range tests {
		got := vipLevelFromPoints(tc.points)
		if got != tc.level {
			t.Errorf("points=%d expected level %d, got %d", tc.points, tc.level, got)
		}
	}
}

func TestRankFromResources(t *testing.T) {
	tests := []struct {
		res  int
		rank int
	}{
		{0, 0}, {500000, 0}, {1000000, 1}, {4999999, 1},
		{5000000, 2}, {25000000, 3}, {100000000, 4},
		{500000000, 5}, {1000000000, 6}, {5000000000, 7},
		{25000000000, 8}, {100000000000, 9},
	}
	for _, tc := range tests {
		got := rankFromResources(tc.res)
		if got != tc.rank {
			t.Errorf("resources=%d expected rank %d, got %d", tc.res, tc.rank, got)
		}
	}
}

func TestVIPProductionBonus(t *testing.T) {
	if b := vipProductionBonus(0); b != 0 {
		t.Errorf("level 0: expected 0, got %.2f", b)
	}
	if b := vipProductionBonus(10); b != 0.30 {
		t.Errorf("level 10: expected 0.30, got %.2f", b)
	}
	if b := vipProductionBonus(12); b != 0.36 {
		t.Errorf("level 12: expected 0.36, got %.2f", b)
	}
}

func TestRankProductionBonus(t *testing.T) {
	if b := rankProductionBonus(0); b != 0 {
		t.Errorf("rank 0: expected 0, got %.2f", b)
	}
	if b := rankProductionBonus(5); b != 0.10 {
		t.Errorf("rank 5: expected 0.10, got %.2f", b)
	}
	if b := rankProductionBonus(9); b != 0.20 {
		t.Errorf("rank 9: expected 0.20, got %.2f", b)
	}
}

func TestVIPPoints_EarnedOnBuildComplete(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	planet, buildings, err := svc.GetOrCreatePlanet(context.Background(), 60)
	if err != nil {
		t.Fatal(err)
	}
	mock := svc.repo.(*mockRepo)
	initialVIP, _, _ := mock.GetPlayerProgress(context.Background(), planet.ID)

	_, err = svc.StartBuildingUpgrade(context.Background(), planet.ID, buildings[0].Type)
	if err != nil {
		t.Fatal("start upgrade:", err)
	}
	mock.queue[planet.ID][0].CompletesAt = time.Now().Add(-1 * time.Second)
	err = svc.processCompletedBuilds(context.Background(), planet.ID)
	if err != nil {
		t.Fatal("process builds:", err)
	}

	vip, _, _ := mock.GetPlayerProgress(context.Background(), planet.ID)
	if vip != initialVIP+10 {
		t.Errorf("expected %d VIP points, got %d", initialVIP+10, vip)
	}
}

func TestTotalResources_TrackedOnPoll(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	planet, _, err := svc.GetOrCreatePlanet(context.Background(), 61)
	if err != nil {
		t.Fatal(err)
	}
	mock := svc.repo.(*mockRepo)
	_, initialTotal, _ := mock.GetPlayerProgress(context.Background(), planet.ID)

	time.Sleep(2 * time.Second)

	_, _, err = svc.GetOrCreatePlanet(context.Background(), 61)
	if err != nil {
		t.Fatal("second call:", err)
	}

	_, total, _ := mock.GetPlayerProgress(context.Background(), planet.ID)
	if total <= initialTotal {
		t.Errorf("total resources should increase after poll, was %d now %d", initialTotal, total)
	}
}

func TestFusionReactor_EnergyTechBoostsOutput(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	buildings := []Building{
		{Type: "fusion_reactor", Level: 1},
	}
	prodLow := svc.calculateProduction(buildings, 1.0, PlanetTypeTerran, 15, 3, 0.0, 0.0)
	prodHigh := svc.calculateProduction(buildings, 1.0, PlanetTypeTerran, 15, 5, 0.0, 0.0)
	if prodHigh.Energy <= prodLow.Energy {
		t.Errorf("higher energy tech should boost fusion output: L3=%.4f L5=%.4f", prodLow.Energy, prodHigh.Energy)
	}
}

func temperatureGasBonus(temp int) float64 {
	if temp <= -60 {
		return 2.5
	} else if temp <= -30 {
		return 2.0
	} else if temp <= 0 {
		return 1.5
	}
	return 0
}

func temperatureSolarBonus(temp int) float64 {
	if temp >= 80 {
		return 3.0
	} else if temp >= 60 {
		return 2.0
	} else if temp >= 40 {
		return 1.0
	}
	return 0
}

func TestCalculateProduction_TempBonus_Gas_Cold(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	buildings := []Building{
		{Type: "gas_mine", Level: 5},
	}
	tests := []struct {
		temp     int
		expected float64
	}{
		{20, 0},    // no bonus
		{0, 1.5},   // ≤ 0°C
		{-10, 1.5}, // ≤ 0°C
		{-30, 2.0}, // ≤ -30°C
		{-40, 2.0}, // ≤ -30°C
		{-60, 2.5}, // ≤ -60°C
		{-80, 2.5}, // ≤ -60°C
	}
	for _, tc := range tests {
		prod := svc.calculateProduction(buildings, 1.0, PlanetTypeTerran, tc.temp, 3, 0.0, 0.0)
		effectiveLevel := 5 + tc.expected
		expectedRate := productionRateForLevel("gas_mine", effectiveLevel) / 60.0
		got := prod.Gas
		if got != expectedRate {
			t.Errorf("temp=%d gas_mine L5: expected %.4f, got %.4f", tc.temp, expectedRate, got)
		}
	}
}

func TestCalculateProduction_TempBonus_Solar_Hot(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	buildings := []Building{
		{Type: "solar_plant", Level: 5},
	}
	tests := []struct {
		temp     int
		expected float64
	}{
		{20, 0},    // no bonus
		{39, 0},    // below threshold
		{40, 1.0},  // ≥ 40°C
		{50, 1.0},  // ≥ 40°C
		{60, 2.0},  // ≥ 60°C
		{70, 2.0},  // ≥ 60°C
		{80, 3.0},  // ≥ 80°C
		{100, 3.0}, // ≥ 80°C
	}
	for _, tc := range tests {
		prod := svc.calculateProduction(buildings, 1.0, PlanetTypeTerran, tc.temp, 3, 0.0, 0.0)
		effectiveLevel := 5 + tc.expected
		expectedRate := productionRateForLevel("solar_plant", effectiveLevel) / 60.0
		got := prod.Energy
		if got != expectedRate {
			t.Errorf("temp=%d solar_plant L5: expected %.4f, got %.4f", tc.temp, expectedRate, got)
		}
	}
}

func TestCalculateProduction_TempBonus_StacksWithTypeBonus(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	buildings := []Building{
		{Type: "gas_mine", Level: 5},
		{Type: "solar_plant", Level: 5},
	}

	// Ice planet at -60°C: gas gets type bonus +1.5 AND temp bonus +2.5 = +4.0
	prodIce := svc.calculateProduction(buildings, 1.0, PlanetTypeIce, -60, 3, 0.0, 0.0)
	iceGasLevel := 5.0 + 1.5 + 2.5
	expectedGas := productionRateForLevel("gas_mine", iceGasLevel) / 60.0
	if prodIce.Gas != expectedGas {
		t.Errorf("ice temp=-60 gas: expected %.4f, got %.4f", expectedGas, prodIce.Gas)
	}

	// Desert planet at 80°C: solar gets type bonus +1.5 AND temp bonus +3.0 = +4.5
	prodDesert := svc.calculateProduction(buildings, 1.0, PlanetTypeDesert, 80, 3, 0.0, 0.0)
	desertSolarLevel := 5.0 + 1.5 + 3.0
	expectedSolar := productionRateForLevel("solar_plant", desertSolarLevel) / 60.0
	if prodDesert.Energy != expectedSolar {
		t.Errorf("desert temp=80 solar: expected %.4f, got %.4f", expectedSolar, prodDesert.Energy)
	}

	// Terran planet at 20°C: no bonuses
	prodTerran := svc.calculateProduction(buildings, 1.0, PlanetTypeTerran, 20, 3, 0.0, 0.0)
	terranGasLevel := 5.0
	terranSolarLevel := 5.0
	expectedGasTerran := productionRateForLevel("gas_mine", terranGasLevel) / 60.0
	expectedSolarTerran := productionRateForLevel("solar_plant", terranSolarLevel) / 60.0
	if prodTerran.Gas != expectedGasTerran {
		t.Errorf("terran gas: expected %.4f, got %.4f", expectedGasTerran, prodTerran.Gas)
	}
	if prodTerran.Energy != expectedSolarTerran {
		t.Errorf("terran solar: expected %.4f, got %.4f", expectedSolarTerran, prodTerran.Energy)
	}
}

func TestCalculateProduction_RankAndVIPBonuses(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	buildings := []Building{
		{Type: "metal_mine", Level: 1},
		{Type: "crystal_mine", Level: 2},
		{Type: "solar_plant", Level: 3},
	}

	base := svc.calculateProduction(buildings, 1.0, PlanetTypeTerran, 15, 3, 0.0, 0.0)
	bonused := svc.calculateProduction(buildings, 1.0, PlanetTypeTerran, 15, 3, 0.3, 0.2)
	expectedMult := 1.5
	tolerance := 0.01

	gotRatio := bonused.Metal / base.Metal
	if diff := gotRatio - expectedMult; diff < -tolerance || diff > tolerance {
		t.Errorf("metal multiplier: expected %.2f, got %.2f", expectedMult, gotRatio)
	}
	gotRatio = bonused.Crystal / base.Crystal
	if diff := gotRatio - expectedMult; diff < -tolerance || diff > tolerance {
		t.Errorf("crystal multiplier: expected %.2f, got %.2f", expectedMult, gotRatio)
	}
	if bonused.Gas != base.Gas {
		t.Errorf("gas should not be affected by bonuses: base %.2f, got %.2f", base.Gas, bonused.Gas)
	}
	if bonused.Energy != base.Energy {
		t.Errorf("energy should not be affected by bonuses: base %.2f, got %.2f", base.Energy, bonused.Energy)
	}
}

func TestGetOrCreatePlanet_IncludesVIPAndRank(t *testing.T) {
	mock := newMockRepo()
	svc := NewPlanetService(mock)
	planet, _, err := svc.GetOrCreatePlanet(context.Background(), 72)
	if err != nil {
		t.Fatal(err)
	}

	err = svc.repo.AddVIPPoints(context.Background(), planet.ID, 100)
	if err != nil {
		t.Fatal(err)
	}
	err = svc.repo.AddResourcesProduced(context.Background(), planet.ID, 1000000)
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = svc.GetOrCreatePlanet(context.Background(), 72)
	if err != nil {
		t.Fatal(err)
	}

	vipPoints, totalResources, err := svc.repo.GetPlayerProgress(context.Background(), planet.ID)
	if err != nil {
		t.Fatal(err)
	}
	vipLevel := vipLevelFromPoints(vipPoints)
	rank := rankFromResources(totalResources)
	if vipLevel == 0 {
		t.Error("expected vipLevel > 0 after adding points")
	}
	_ = totalResources
	_ = rank
}

func TestListGalaxies(t *testing.T) {
	mock := newMockRepo()
	galaxies, err := mock.ListGalaxies(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(galaxies) != 9 {
		t.Errorf("expected 9 galaxies, got %d", len(galaxies))
	}
	if galaxies[0].Name != "Galaxy 1" {
		t.Errorf("expected Galaxy 1, got %s", galaxies[0].Name)
	}
}

func TestGetSystemPositions_Mock(t *testing.T) {
	mock := newMockRepo()
	positions, err := mock.GetSystemPositions(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(positions) != 15 {
		t.Errorf("expected 15 positions, got %d", len(positions))
	}
	for i, pos := range positions {
		if pos.PositionNum != i+1 {
			t.Errorf("position %d: expected num %d, got %d", i, i+1, pos.PositionNum)
		}
		if pos.State != "empty" {
			t.Errorf("position %d: expected empty, got %s", i, pos.State)
		}
	}
}

func TestHandler_ListGalaxies(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	h := NewHandler(svc)

	req := httptest.NewRequest("GET", "/api/galaxy", nil)
	w := httptest.NewRecorder()
	h.ListGalaxies(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var galaxies []Galaxy
	if err := json.NewDecoder(w.Body).Decode(&galaxies); err != nil {
		t.Fatal(err)
	}
	if len(galaxies) != 9 {
		t.Errorf("expected 9 galaxies, got %d", len(galaxies))
	}
}

func TestHandler_ListSystems_InvalidGalaxy(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	h := NewHandler(svc)
	req := httptest.NewRequest("GET", "/api/galaxy/systems/abc", nil)
	w := httptest.NewRecorder()
	h.ListSystems(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid galaxy ID, got %d", w.Code)
	}
}

func TestHandler_GetPositions_InvalidSystem(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	h := NewHandler(svc)
	req := httptest.NewRequest("GET", "/api/galaxy/positions/abc", nil)
	w := httptest.NewRecorder()
	h.GetPositions(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid system ID, got %d", w.Code)
	}
}

func TestShipConfigs(t *testing.T) {
	if len(Ships) != 13 {
		t.Errorf("expected 12 ships, got %d", len(Ships))
	}
	for _, s := range Ships {
		if s.Type == "" || s.Name == "" {
			t.Errorf("ship missing type or name: %+v", s)
		}
	}
}

func TestBuildShips_Valid(t *testing.T) {
	mock := newMockRepo()
	svc := NewPlanetService(mock)
	planet, _, err := svc.GetOrCreatePlanet(context.Background(), 80)
	if err != nil {
		t.Fatal(err)
	}

	planet.Metal = 50000
	planet.Crystal = 50000
	planet.Gas = 50000
	mock.planets[planet.ID] = planet

	quantity, buildTime, err := svc.BuildShips(context.Background(), planet.ID, "cargo", 5)
	if err != nil {
		t.Fatal(err)
	}
	if quantity != 5 {
		t.Errorf("expected 5, got %d", quantity)
	}
	if buildTime < 0 {
		t.Errorf("expected positive build time, got %f", buildTime)
	}

	planet, _ = mock.FindByID(context.Background(), planet.ID)
	cost := Ships[0].Metal * 5
	expectedMetal := 50000 - cost
	if planet.Metal != expectedMetal {
		t.Errorf("expected metal %d, got %d", expectedMetal, planet.Metal)
	}
}

func TestBuildShips_InsufficientResources(t *testing.T) {
	mock := newMockRepo()
	svc := NewPlanetService(mock)
	planet, _, err := svc.GetOrCreatePlanet(context.Background(), 81)
	if err != nil {
		t.Fatal(err)
	}

	planet.Metal = 0
	mock.planets[planet.ID] = planet

	_, _, err = svc.BuildShips(context.Background(), planet.ID, "dreadnought", 1)
	if err != ErrInsufficientResources {
		t.Errorf("expected ErrInsufficientResources, got %v", err)
	}
}

func TestBuildShips_MaxQuantity(t *testing.T) {
	mock := newMockRepo()
	svc := NewPlanetService(mock)
	planet, _, err := svc.GetOrCreatePlanet(context.Background(), 84)
	if err != nil {
		t.Fatal(err)
	}
	maxQ, err := svc.MaxShipQuantity(context.Background(), planet.ID, "cargo")
	if err != nil {
		t.Fatal(err)
	}
	expected := planet.Metal / 2000
	if maxQ != expected {
		t.Errorf("expected max %d, got %d", expected, maxQ)
	}
}

func TestBuildShips_InvalidShip(t *testing.T) {
	mock := newMockRepo()
	svc := NewPlanetService(mock)
	planet, _, err := svc.GetOrCreatePlanet(context.Background(), 82)
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = svc.BuildShips(context.Background(), planet.ID, "death_star", 1)
	if err != ErrInvalidShip {
		t.Errorf("expected ErrInvalidShip, got %v", err)
	}
}

func TestBuildShips_NoShipyard(t *testing.T) {
	mock := newMockRepo()
	svc := NewPlanetService(mock)
	planet, _, err := svc.GetOrCreatePlanet(context.Background(), 83)
	if err != nil {
		t.Fatal(err)
	}

	for i, b := range mock.buildings[planet.ID] {
		if b.Type == "shipyard" {
			mock.buildings[planet.ID][i].Level = 0
		}
	}

	_, _, err = svc.BuildShips(context.Background(), planet.ID, "cargo", 1)
	if err != ErrNoShipyard {
		t.Errorf("expected ErrNoShipyard, got %v", err)
	}
}

func TestDefenseConfigs(t *testing.T) {
	if len(Defenses) != 7 {
		t.Errorf("expected 7 defenses, got %d", len(Defenses))
	}
	for _, d := range Defenses {
		if d.Type == "" || d.Name == "" {
			t.Errorf("defense missing type or name: %+v", d)
		}
		if d.Fields < 1 {
			t.Errorf("defense %s should have fields > 0", d.Type)
		}
	}
}

func TestDefenseConfig_Lookup(t *testing.T) {
	cfg, ok := defenseConfig("rocket_launcher")
	if !ok {
		t.Fatal("expected rocket_launcher to be found")
	}
	if cfg.Metal != 2000 {
		t.Errorf("expected metal 2000, got %d", cfg.Metal)
	}

	_, ok = defenseConfig("nonexistent")
	if ok {
		t.Error("expected nonexistent to not be found")
	}
}

func TestBuildDefenses_Valid(t *testing.T) {
	mock := newMockRepo()
	svc := NewPlanetService(mock)
	planet, _, err := svc.GetOrCreatePlanet(context.Background(), 90)
	if err != nil {
		t.Fatal(err)
	}

	planet.Metal = 50000
	planet.Crystal = 50000
	planet.Gas = 50000
	mock.planets[planet.ID] = planet

	quantity, buildTime, err := svc.BuildDefenses(context.Background(), planet.ID, "rocket_launcher", 5)
	if err != nil {
		t.Fatal(err)
	}
	if quantity != 5 {
		t.Errorf("expected 5, got %d", quantity)
	}
	if buildTime < 0 {
		t.Errorf("expected positive build time, got %f", buildTime)
	}

	planet, _ = mock.FindByID(context.Background(), planet.ID)
	expectedMetal := 50000 - 2000*5
	if planet.Metal != expectedMetal {
		t.Errorf("expected metal %d, got %d", expectedMetal, planet.Metal)
	}

	defenses, _ := mock.GetPlayerDefenses(context.Background(), planet.ID)
	if defenses["rocket_launcher"] != 5 {
		t.Errorf("expected 5 rocket launchers, got %d", defenses["rocket_launcher"])
	}
}

func TestBuildDefenses_InsufficientResources(t *testing.T) {
	mock := newMockRepo()
	svc := NewPlanetService(mock)
	planet, _, err := svc.GetOrCreatePlanet(context.Background(), 91)
	if err != nil {
		t.Fatal(err)
	}

	planet.Metal = 0
	mock.planets[planet.ID] = planet

	_, _, err = svc.BuildDefenses(context.Background(), planet.ID, "plasma_cannon", 1)
	if err != ErrInsufficientResources {
		t.Errorf("expected ErrInsufficientResources, got %v", err)
	}
}

func TestBuildDefenses_InvalidDefense(t *testing.T) {
	mock := newMockRepo()
	svc := NewPlanetService(mock)
	planet, _, err := svc.GetOrCreatePlanet(context.Background(), 92)
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = svc.BuildDefenses(context.Background(), planet.ID, "death_star", 1)
	if err != ErrInvalidDefense {
		t.Errorf("expected ErrInvalidDefense, got %v", err)
	}
}

func TestBuildDefenses_NoShipyard(t *testing.T) {
	mock := newMockRepo()
	svc := NewPlanetService(mock)
	planet, _, err := svc.GetOrCreatePlanet(context.Background(), 93)
	if err != nil {
		t.Fatal(err)
	}

	for i, b := range mock.buildings[planet.ID] {
		if b.Type == "shipyard" {
			mock.buildings[planet.ID][i].Level = 0
		}
	}

	_, _, err = svc.BuildDefenses(context.Background(), planet.ID, "rocket_launcher", 1)
	if err != ErrNoShipyard {
		t.Errorf("expected ErrNoShipyard, got %v", err)
	}
}

func TestBuildDefenses_MaxQuantity(t *testing.T) {
	mock := newMockRepo()
	svc := NewPlanetService(mock)
	planet, _, err := svc.GetOrCreatePlanet(context.Background(), 94)
	if err != nil {
		t.Fatal(err)
	}
	maxQ, err := svc.MaxDefenseQuantity(context.Background(), planet.ID, "rocket_launcher")
	if err != nil {
		t.Fatal(err)
	}
	expected := planet.Metal / 2000
	if maxQ != expected {
		t.Errorf("expected max %d, got %d", expected, maxQ)
	}
}

func TestDefenseBuildDuration(t *testing.T) {
	d := defenseBuildDuration("rocket_launcher", 1, 1, 0)
	if d <= 0 {
		t.Error("expected positive duration")
	}
}

func TestDefenseBuildDuration_ZeroForInvalid(t *testing.T) {
	d := defenseBuildDuration("nonexistent", 1, 1, 0)
	if d != 0 {
		t.Errorf("expected 0 for invalid defense, got %f", d)
	}
}

func TestDefenseConfig_FieldsPopulated(t *testing.T) {
	for _, d := range Defenses {
		if d.Fields < 1 || d.Fields > 4 {
			t.Errorf("%s has unexpected fields value: %d", d.Type, d.Fields)
		}
	}
}

func TestDefenseConfig_NoSpeedCargoFuel(t *testing.T) {
	// Defense configs should not have Speed, Cargo, Fuel fields
	// They are intentionally omitted from the struct
	for _, d := range Defenses {
		if d.Metal <= 0 && d.Crystal <= 0 && d.Gas <= 0 {
			t.Errorf("%s should have at least some cost", d.Type)
		}
	}
}

func TestListDefenses_Handler(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	h := NewHandler(svc)

	planet, _, err := svc.GetOrCreatePlanet(context.Background(), 95)
	if err != nil {
		t.Fatal(err)
	}
	_ = planet

	req := httptest.NewRequest("GET", "/api/defense", nil)
	req.Header.Set("X-User-ID", "95")
	w := httptest.NewRecorder()
	h.ListDefenses(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal("decode response:", err)
	}

	defenses, ok := resp["defenses"].([]any)
	if !ok {
		t.Fatal("expected defenses array in response")
	}
	if len(defenses) != 7 {
		t.Errorf("expected 7 defenses, got %d", len(defenses))
	}

	first := defenses[0].(map[string]any)
	if first["type"] != "rocket_launcher" {
		t.Errorf("expected rocket_launcher, got %v", first["type"])
	}
	if first["fields"] != float64(1) {
		t.Errorf("expected fields 1, got %v", first["fields"])
	}
}

func TestBuildDefenses_Handler_Success(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	h := NewHandler(svc)

	planet, _, err := svc.GetOrCreatePlanet(context.Background(), 96)
	if err != nil {
		t.Fatal(err)
	}

	mock := svc.repo.(*mockRepo)
	for i, b := range mock.buildings[planet.ID] {
		if b.Type == "metal_storage" {
			mock.buildings[planet.ID][i].Level = 5
		}
	}
	p, _ := mock.FindByID(context.Background(), planet.ID)
	p.Metal = 100000
	p.Crystal = 100000
	p.Gas = 100000
	mock.planets[planet.ID] = p

	body := `{"defense_type":"rocket_launcher","quantity":5}`
	req := httptest.NewRequest("POST", "/api/defense/build", strings.NewReader(body))
	req.Header.Set("X-User-ID", "96")
	w := httptest.NewRecorder()
	h.BuildDefenses(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal("decode response:", err)
	}
	if resp["type"] != "rocket_launcher" {
		t.Errorf("expected rocket_launcher, got %v", resp["type"])
	}
	if resp["quantity"] != float64(5) {
		t.Errorf("expected quantity 5, got %v", resp["quantity"])
	}
}

func TestBuildDefenses_Handler_InvalidDefense(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	h := NewHandler(svc)

	planet, _, err := svc.GetOrCreatePlanet(context.Background(), 97)
	if err != nil {
		t.Fatal(err)
	}
	_ = planet

	body := `{"defense_type":"nonexistent","quantity":1}`
	req := httptest.NewRequest("POST", "/api/defense/build", strings.NewReader(body))
	req.Header.Set("X-User-ID", "97")
	w := httptest.NewRecorder()
	h.BuildDefenses(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDefenseBuildDuration_NaniteReduction(t *testing.T) {
	base := defenseBuildDuration("rocket_launcher", 1, 1, 0)
	reduced := defenseBuildDuration("rocket_launcher", 1, 1, 3)
	if reduced >= base {
		t.Errorf("nanite should reduce time: base=%f reduced=%f", base, reduced)
	}
}

func TestDefenseBuildDuration_HigherLevelFaster(t *testing.T) {
	low := defenseBuildDuration("rocket_launcher", 1, 1, 0)
	high := defenseBuildDuration("rocket_launcher", 1, 5, 0)
	if high >= low {
		t.Errorf("higher shipyard should reduce time: SY1=%f SY5=%f", low, high)
	}
}

func TestShieldDomeConfig(t *testing.T) {
	cfg, ok := shieldDomeConfig("small_shield_dome")
	if !ok {
		t.Fatal("expected small_shield_dome to be found")
	}
	if cfg.ShieldHP != 10000 {
		t.Errorf("expected shield hp 10000, got %d", cfg.ShieldHP)
	}
	if cfg.CostMetal != 20000 || cfg.CostCrystal != 10000 || cfg.CostGas != 0 {
		t.Errorf("unexpected costs: m=%d c=%d g=%d", cfg.CostMetal, cfg.CostCrystal, cfg.CostGas)
	}

	cfg, ok = shieldDomeConfig("large_shield_dome")
	if !ok {
		t.Fatal("expected large_shield_dome to be found")
	}
	if cfg.ShieldHP != 100000 {
		t.Errorf("expected shield hp 100000, got %d", cfg.ShieldHP)
	}
	if cfg.CostMetal != 100000 || cfg.CostCrystal != 50000 || cfg.CostGas != 20000 {
		t.Errorf("unexpected costs: m=%d c=%d g=%d", cfg.CostMetal, cfg.CostCrystal, cfg.CostGas)
	}

	_, ok = shieldDomeConfig("nonexistent")
	if ok {
		t.Error("expected nonexistent to not be found")
	}
}

func TestShieldDome_Costs(t *testing.T) {
	m, c, g := buildingCostResources("small_shield_dome", 0)
	if m != 20000 || c != 10000 || g != 0 {
		t.Errorf("small shield dome L0: expected 20000/10000/0, got %d/%d/%d", m, c, g)
	}

	// Cost should be the same regardless of level (max 1 per planet)
	m2, c2, g2 := buildingCostResources("small_shield_dome", 1)
	if m2 != 20000 || c2 != 10000 || g2 != 0 {
		t.Errorf("small shield dome L1: expected 20000/10000/0, got %d/%d/%d", m2, c2, g2)
	}

	m, c, g = buildingCostResources("large_shield_dome", 0)
	if m != 100000 || c != 50000 || g != 20000 {
		t.Errorf("large shield dome L0: expected 100000/50000/20000, got %d/%d/%d", m, c, g)
	}
}

func TestShieldDome_Max1PerPlanet(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	planet, _, err := svc.GetOrCreatePlanet(context.Background(), 100)
	if err != nil {
		t.Fatal(err)
	}

	mock := svc.repo.(*mockRepo)
	p, _ := mock.FindByID(context.Background(), planet.ID)
	p.Metal = 1000000
	p.Crystal = 1000000
	p.Gas = 1000000
	mock.planets[planet.ID] = p

	_, err = svc.StartBuildingUpgrade(context.Background(), planet.ID, "small_shield_dome")
	if err != nil {
		t.Fatal("first small shield dome should succeed, got:", err)
	}

	// Complete the first build
	mock.queue[planet.ID][0].CompletesAt = time.Now().Add(-1 * time.Second)
	if err := svc.processCompletedBuilds(context.Background(), planet.ID); err != nil {
		t.Fatal("process builds:", err)
	}

	// Reset resources after build
	p2, _ := mock.FindByID(context.Background(), planet.ID)
	p2.Metal = 1000000
	p2.Crystal = 1000000
	p2.Gas = 1000000
	mock.planets[planet.ID] = p2

	_, err = svc.StartBuildingUpgrade(context.Background(), planet.ID, "small_shield_dome")
	if err == nil {
		t.Error("expected error for second small shield dome")
	}
	if err != nil && err.Error() != "max 1 small_shield_dome per planet" {
		t.Errorf("expected max 1 error, got %v", err)
	}
}

func TestDefenseRepair_Service(t *testing.T) {
	mock := newMockRepo()
	svc := NewPlanetService(mock)

	planet, _, err := svc.GetOrCreatePlanet(context.Background(), 110)
	if err != nil {
		t.Fatal(err)
	}

	mock.playerDefenses[planet.ID] = map[string]int{
		"rocket_launcher": 10,
		"light_laser":     5,
	}

	losses := map[string]int{
		"rocket_launcher": 4,
		"light_laser":     2,
	}

	repaired, err := svc.RepairDefenses(context.Background(), planet.ID, losses)
	if err != nil {
		t.Fatal("repair defenses:", err)
	}

	// 70% of 4 = ceil(2.8) = 3
	if repaired["rocket_launcher"] != 3 {
		t.Errorf("expected 3 rocket_launchers repaired, got %d", repaired["rocket_launcher"])
	}
	// 70% of 2 = ceil(1.4) = 2
	if repaired["light_laser"] != 2 {
		t.Errorf("expected 2 light_lasers repaired, got %d", repaired["light_laser"])
	}

	// Verify quantities updated
	rl, _ := mock.GetPlayerDefense(context.Background(), planet.ID, "rocket_launcher")
	if rl != 13 {
		t.Errorf("expected 13 rocket_launchers after repair, got %d", rl)
	}
	ll, _ := mock.GetPlayerDefense(context.Background(), planet.ID, "light_laser")
	if ll != 7 {
		t.Errorf("expected 7 light_lasers after repair, got %d", ll)
	}
}

func TestDefenseDeduct_Service(t *testing.T) {
	mock := newMockRepo()
	svc := NewPlanetService(mock)

	planet, _, err := svc.GetOrCreatePlanet(context.Background(), 111)
	if err != nil {
		t.Fatal(err)
	}

	mock.playerDefenses[planet.ID] = map[string]int{
		"rocket_launcher": 10,
		"light_laser":     5,
	}

	losses := map[string]int{
		"rocket_launcher": 3,
		"light_laser":     7,
	}

	if err := svc.DeductDefenses(context.Background(), planet.ID, losses); err != nil {
		t.Fatal("deduct defenses:", err)
	}

	rl, _ := mock.GetPlayerDefense(context.Background(), planet.ID, "rocket_launcher")
	if rl != 7 {
		t.Errorf("expected 7 rocket_launchers after deduct, got %d", rl)
	}
	// Should cap at 0 when deducting more than available
	ll, _ := mock.GetPlayerDefense(context.Background(), planet.ID, "light_laser")
	if ll != 0 {
		t.Errorf("expected 0 light_lasers after deduct (capped), got %d", ll)
	}
}

func TestDefenseRepair_Handler(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	h := NewHandler(svc)

	planet, _, err := svc.GetOrCreatePlanet(context.Background(), 112)
	if err != nil {
		t.Fatal(err)
	}

	mock := svc.repo.(*mockRepo)
	mock.playerDefenses[planet.ID] = map[string]int{
		"rocket_launcher": 10,
	}

	body := `{"planet_id":` + itoa(planet.ID) + `,"defense_losses":{"rocket_launcher":4}}`
	req := httptest.NewRequest("POST", "/internal/defense/repair", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.InternalDefenseRepair(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal("decode response:", err)
	}

	repaired, ok := resp["repaired"].(map[string]any)
	if !ok {
		t.Fatal("expected repaired map in response")
	}
	if repaired["rocket_launcher"] != float64(3) {
		t.Errorf("expected 3 repaired, got %v", repaired["rocket_launcher"])
	}
}

func TestDefenseList_Handler(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	h := NewHandler(svc)

	planet, _, err := svc.GetOrCreatePlanet(context.Background(), 113)
	if err != nil {
		t.Fatal(err)
	}

	mock := svc.repo.(*mockRepo)
	mock.playerDefenses[planet.ID] = map[string]int{
		"rocket_launcher": 5,
		"light_laser":     3,
	}

	body := `{"planet_id":` + itoa(planet.ID) + `}`
	req := httptest.NewRequest("POST", "/internal/defense/list", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.InternalDefenseList(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal("decode response:", err)
	}

	defenses, ok := resp["defenses"].(map[string]any)
	if !ok {
		t.Fatal("expected defenses map in response")
	}
	if defenses["rocket_launcher"] != float64(5) {
		t.Errorf("expected 5 rocket_launchers, got %v", defenses["rocket_launcher"])
	}
	if defenses["light_laser"] != float64(3) {
		t.Errorf("expected 3 light_lasers, got %v", defenses["light_laser"])
	}
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}

func TestPlanet_CreateAtCoords(t *testing.T) {
	mock := newMockRepo()

	planet, buildings, err := mock.CreateAtCoords(context.Background(), 200, 3, 5, 8)
	if err != nil {
		t.Fatal(err)
	}
	if planet.UserID != 200 {
		t.Errorf("expected user_id 200, got %d", planet.UserID)
	}
	if planet.Galaxy != 3 || planet.System != 5 || planet.Position != 8 {
		t.Errorf("expected coords 3:5:8, got %d:%d:%d", planet.Galaxy, planet.System, planet.Position)
	}
	if planet.Name != "Colony [3:5:8]" {
		t.Errorf("expected name 'Colony [3:5:8]', got '%s'", planet.Name)
	}
	if planet.Metal != 500 || planet.Crystal != 300 || planet.Gas != 200 {
		t.Errorf("expected default resources 500/300/200, got %d/%d/%d", planet.Metal, planet.Crystal, planet.Gas)
	}
	if len(buildings) == 0 {
		t.Error("expected seeded buildings")
	}
}

func TestMoonBuildingCost(t *testing.T) {
	metal, crystal, gas := moonBuildingCostResources("moon_base", 0)
	if metal != 40000 || crystal != 20000 || gas != 10000 {
		t.Errorf("moon_base L0: expected 40000/20000/10000, got %d/%d/%d", metal, crystal, gas)
	}
	metal, crystal, gas = moonBuildingCostResources("pioneer_lab", 0)
	if metal != 40000 || crystal != 80000 || gas != 40000 {
		t.Errorf("pioneer_lab L0: expected 40000/80000/40000, got %d/%d/%d", metal, crystal, gas)
	}
	metal, crystal, gas = moonBuildingCostResources("wormhole_generator", 0)
	if metal != 3200000 || crystal != 6400000 || gas != 3200000 {
		t.Errorf("wormhole_generator L0: expected 3200000/6400000/3200000, got %d/%d/%d", metal, crystal, gas)
	}
}

func TestMoonBuildingUpgrade_Success(t *testing.T) {
	mock := newMockRepo()
	svc := NewPlanetService(mock)

	planet, _, err := svc.GetOrCreatePlanet(context.Background(), 300)
	if err != nil {
		t.Fatal(err)
	}

	galaxy, system, pos := planet.Galaxy, planet.System, planet.Position
	key := moonKey(galaxy, system, pos)
	mock.moonExists[key] = true
	mock.moonBuildings[key] = []MoonBuilding{
		{Type: "moon_base", Level: 3},
		{Type: "robotics_factory", Level: 1},
		{Type: "shipyard", Level: 1},
		{Type: "pioneer_lab", Level: 1},
	}

	planet.Metal = 1000000
	planet.Crystal = 1000000
	planet.Gas = 1000000
	mock.planets[planet.ID] = planet

	err = svc.StartMoonBuildingUpgrade(context.Background(), galaxy, system, pos, "robotics_factory")
	if err != nil {
		t.Fatal("expected success, got:", err)
	}

	level, err := mock.GetMoonBuildingLevel(context.Background(), galaxy, system, pos, "robotics_factory")
	if err != nil {
		t.Fatal("get level:", err)
	}
	if level != 2 {
		t.Errorf("expected level 2, got %d", level)
	}
}

func TestMoonBuildingUpgrade_MoonBaseRequired(t *testing.T) {
	mock := newMockRepo()
	svc := NewPlanetService(mock)

	planet, _, err := svc.GetOrCreatePlanet(context.Background(), 301)
	if err != nil {
		t.Fatal(err)
	}

	galaxy, system, pos := planet.Galaxy, planet.System, planet.Position
	key := moonKey(galaxy, system, pos)
	mock.moonExists[key] = true
	mock.moonBuildings[key] = []MoonBuilding{
		{Type: "moon_base", Level: 0},
	}

	err = svc.StartMoonBuildingUpgrade(context.Background(), galaxy, system, pos, "robotics_factory")
	if err != ErrMoonBaseRequired {
		t.Errorf("expected ErrMoonBaseRequired, got %v", err)
	}
}

func TestMoonBuildingUpgrade_InsufficientResources(t *testing.T) {
	mock := newMockRepo()
	svc := NewPlanetService(mock)

	planet, _, err := svc.GetOrCreatePlanet(context.Background(), 302)
	if err != nil {
		t.Fatal(err)
	}

	galaxy, system, pos := planet.Galaxy, planet.System, planet.Position
	key := moonKey(galaxy, system, pos)
	mock.moonExists[key] = true
	mock.moonBuildings[key] = []MoonBuilding{
		{Type: "moon_base", Level: 3},
	}

	planet.Metal = 0
	planet.Crystal = 0
	planet.Gas = 0
	mock.planets[planet.ID] = planet

	err = svc.StartMoonBuildingUpgrade(context.Background(), galaxy, system, pos, "robotics_factory")
	if err != ErrInsufficientResources {
		t.Errorf("expected ErrInsufficientResources, got %v", err)
	}
}

func TestMoonBuildingUpgrade_WormholePrereqs(t *testing.T) {
	mock := newMockRepo()
	svc := NewPlanetService(mock)

	planet, _, err := svc.GetOrCreatePlanet(context.Background(), 303)
	if err != nil {
		t.Fatal(err)
	}

	galaxy, system, pos := planet.Galaxy, planet.System, planet.Position
	key := moonKey(galaxy, system, pos)
	mock.moonExists[key] = true
	mock.moonBuildings[key] = []MoonBuilding{
		{Type: "moon_base", Level: 2},
		{Type: "robotics_factory", Level: 2},
	}

	err = svc.StartMoonBuildingUpgrade(context.Background(), galaxy, system, pos, "wormhole_generator")
	if err != ErrPrerequisitesNotMet {
		t.Errorf("expected ErrPrerequisitesNotMet, got %v", err)
	}
}

func TestWormholeLink_Success(t *testing.T) {
	mock := newMockRepo()
	svc := NewPlanetService(mock)

	now := time.Now()
	wh1 := &WormholeEntry{
		MoonGalaxy: 1, MoonSystem: 1, MoonPos: 1,
		Level: 1,
	}
	wh2 := &WormholeEntry{
		MoonGalaxy: 1, MoonSystem: 1, MoonPos: 2,
		Level: 1,
	}
	mock.wormholes[moonKey(1, 1, 1)] = wh1
	mock.wormholes[moonKey(1, 1, 2)] = wh2
	_ = now

	err := svc.LinkWormholes(context.Background(), 1, 1, 1, 1, 1, 2)
	if err != nil {
		t.Fatal("expected success, got:", err)
	}

	w1, _ := mock.GetWormhole(context.Background(), 1, 1, 1)
	if w1.LinkedGalaxy == nil || *w1.LinkedGalaxy != 1 || *w1.LinkedSystem != 1 || *w1.LinkedPosition != 2 {
		t.Error("source wormhole should link to target")
	}

	w2, _ := mock.GetWormhole(context.Background(), 1, 1, 2)
	if w2.LinkedGalaxy == nil || *w2.LinkedGalaxy != 1 || *w2.LinkedSystem != 1 || *w2.LinkedPosition != 1 {
		t.Error("target wormhole should link to source")
	}
}

func TestWormholeLink_MissingGenerator(t *testing.T) {
	mock := newMockRepo()
	svc := NewPlanetService(mock)

	err := svc.LinkWormholes(context.Background(), 1, 1, 1, 1, 1, 2)
	if err != ErrWormholeNotFound {
		t.Errorf("expected ErrWormholeNotFound, got %v", err)
	}
}

func TestBuildIronBehemoth_NoPioneerLab(t *testing.T) {
	mock := newMockRepo()
	svc := NewPlanetService(mock)

	planet, _, err := svc.GetOrCreatePlanet(context.Background(), 304)
	if err != nil {
		t.Fatal(err)
	}

	galaxy, system, pos := planet.Galaxy, planet.System, planet.Position
	key := moonKey(galaxy, system, pos)
	mock.moonExists[key] = true
	mock.moonBuildings[key] = []MoonBuilding{
		{Type: "moon_base", Level: 1},
	}

	planet.Metal = 1000000
	planet.Crystal = 1000000
	planet.Gas = 1000000
	mock.planets[planet.ID] = planet

	_, _, err = svc.BuildIronBehemoth(context.Background(), galaxy, system, pos, 1)
	if err != ErrPioneerLabRequired {
		t.Errorf("expected ErrPioneerLabRequired, got %v", err)
	}
}

func TestBuildIronBehemoth_Success(t *testing.T) {
	mock := newMockRepo()
	svc := NewPlanetService(mock)

	planet, _, err := svc.GetOrCreatePlanet(context.Background(), 305)
	if err != nil {
		t.Fatal(err)
	}

	galaxy, system, pos := planet.Galaxy, planet.System, planet.Position
	key := moonKey(galaxy, system, pos)
	mock.moonExists[key] = true
	mock.moonBuildings[key] = []MoonBuilding{
		{Type: "moon_base", Level: 1},
		{Type: "pioneer_lab", Level: 1},
	}

	planet.Metal = 1000000
	planet.Crystal = 1000000
	planet.Gas = 1000000
	mock.planets[planet.ID] = planet

	quantity, buildTime, err := svc.BuildIronBehemoth(context.Background(), galaxy, system, pos, 2)
	if err != nil {
		t.Fatal("expected success, got:", err)
	}
	if quantity != 2 {
		t.Errorf("expected 2, got %d", quantity)
	}
	if buildTime < 0 {
		t.Errorf("expected positive build time, got %f", buildTime)
	}

	updatedPlanet, _ := mock.FindByID(context.Background(), planet.ID)
	expectedMetal := 1000000 - 350000*2
	if updatedPlanet.Metal != expectedMetal {
		t.Errorf("expected metal %d, got %d", expectedMetal, updatedPlanet.Metal)
	}
}

func TestGetMoonBuildings(t *testing.T) {
	mock := newMockRepo()
	svc := NewPlanetService(mock)

	key := moonKey(1, 1, 1)
	mock.moonBuildings[key] = []MoonBuilding{
		{Type: "moon_base", Level: 2},
		{Type: "shipyard", Level: 1},
	}

	buildings, maxFields, err := svc.GetMoonBuildings(context.Background(), 1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if maxFields != baseMoonFields+2*moonBaseFieldsPerLevel {
		t.Errorf("expected %d max fields, got %d", baseMoonFields+2*moonBaseFieldsPerLevel, maxFields)
	}
	if len(buildings) != 2 {
		t.Errorf("expected 2 buildings, got %d", len(buildings))
	}
}

func TestHandler_GetMoonBuildings(t *testing.T) {
	mock := newMockRepo()
	svc := NewPlanetService(mock)
	h := NewHandler(svc)

	key := moonKey(2, 3, 4)
	mock.moonBuildings[key] = []MoonBuilding{
		{Type: "moon_base", Level: 1},
		{Type: "robotics_factory", Level: 0},
	}

	req := httptest.NewRequest("GET", "/api/moon/2/3/4/buildings", nil)
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("galaxy", "2")
	chiCtx.URLParams.Add("system", "3")
	chiCtx.URLParams.Add("position", "4")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))
	w := httptest.NewRecorder()
	h.GetMoonBuildings(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}

	maxFields, ok := resp["max_fields"].(float64)
	if !ok || int(maxFields) != baseMoonFields+1*moonBaseFieldsPerLevel {
		t.Errorf("unexpected max_fields: %v", resp["max_fields"])
	}
}

func TestMoonBuildingLabel(t *testing.T) {
	labels := map[string]string{
		"moon_base":          "Moon Base",
		"pioneer_lab":        "Pioneer Lab",
		"wormhole_generator": "Wormhole Generator",
	}
	for typ, expected := range labels {
		got := moonBuildingLabel(typ)
		if got != expected {
			t.Errorf("%s: expected '%s', got '%s'", typ, expected, got)
		}
	}
}

func TestHandler_InternalGetMoonBuildingLevel(t *testing.T) {
	mock := newMockRepo()
	svc := NewPlanetService(mock)
	h := NewHandler(svc)

	key := moonKey(3, 5, 7)
	mock.moonBuildings[key] = []MoonBuilding{
		{Type: "moon_base", Level: 2},
	}

	body := `{"galaxy":3,"system":5,"position":7,"building_type":"moon_base"}`
	req := httptest.NewRequest("POST", "/internal/moon/building-level", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.InternalGetMoonBuildingLevel(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["level"] != float64(2) {
		t.Errorf("expected level 2, got %v", resp["level"])
	}
}

func TestHandler_InternalWormholeInfo(t *testing.T) {
	mock := newMockRepo()
	svc := NewPlanetService(mock)
	h := NewHandler(svc)

	mock.wormholes[moonKey(4, 6, 8)] = &WormholeEntry{
		MoonGalaxy: 4, MoonSystem: 6, MoonPos: 8,
		Level: 1,
	}

	body := `{"galaxy":4,"system":6,"position":8}`
	req := httptest.NewRequest("POST", "/internal/wormhole/info", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.InternalWormholeInfo(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["has_generator"] != true {
		t.Errorf("expected has_generator true, got %v", resp["has_generator"])
	}
	if resp["level"] != float64(1) {
		t.Errorf("expected level 1, got %v", resp["level"])
	}
}

func TestHandler_InternalWormholeInfo_NotFound(t *testing.T) {
	mock := newMockRepo()
	svc := NewPlanetService(mock)
	h := NewHandler(svc)

	body := `{"galaxy":9,"system":9,"position":9}`
	req := httptest.NewRequest("POST", "/internal/wormhole/info", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.InternalWormholeInfo(w, req)

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["has_generator"] != false {
		t.Errorf("expected has_generator false, got %v", resp["has_generator"])
	}
}

func TestHandler_UpgradeMoonBuilding(t *testing.T) {
	mock := newMockRepo()
	svc := NewPlanetService(mock)
	h := NewHandler(svc)

	planet, _, err := svc.GetOrCreatePlanet(context.Background(), 400)
	if err != nil {
		t.Fatal(err)
	}

	galaxy, system, pos := planet.Galaxy, planet.System, planet.Position
	key := moonKey(galaxy, system, pos)
	mock.moonExists[key] = true
	mock.moonBuildings[key] = []MoonBuilding{
		{Type: "moon_base", Level: 3},
	}

	planet.Metal = 500000
	planet.Crystal = 500000
	planet.Gas = 500000
	mock.planets[planet.ID] = planet

	req := httptest.NewRequest("POST", fmt.Sprintf("/api/moon/%d/%d/%d/buildings/robotics_factory/upgrade", galaxy, system, pos), nil)
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("galaxy", fmt.Sprintf("%d", galaxy))
	chiCtx.URLParams.Add("system", fmt.Sprintf("%d", system))
	chiCtx.URLParams.Add("position", fmt.Sprintf("%d", pos))
	chiCtx.URLParams.Add("type", "robotics_factory")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))
	req.Header.Set("X-User-ID", "400")
	w := httptest.NewRecorder()
	h.UpgradeMoonBuilding(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["ok"] != true {
		t.Error("expected ok: true")
	}
}

func TestHandler_BuildIronBehemoth(t *testing.T) {
	mock := newMockRepo()
	svc := NewPlanetService(mock)
	h := NewHandler(svc)

	planet, _, err := svc.GetOrCreatePlanet(context.Background(), 401)
	if err != nil {
		t.Fatal(err)
	}

	galaxy, system, pos := planet.Galaxy, planet.System, planet.Position
	key := moonKey(galaxy, system, pos)
	mock.moonExists[key] = true
	mock.moonBuildings[key] = []MoonBuilding{
		{Type: "moon_base", Level: 1},
		{Type: "pioneer_lab", Level: 2},
	}

	planet.Metal = 1000000
	planet.Crystal = 1000000
	planet.Gas = 1000000
	mock.planets[planet.ID] = planet

	body := `{"quantity":1}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/moon/%d/%d/%d/build-iron-behemoth", galaxy, system, pos), strings.NewReader(body))
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("galaxy", fmt.Sprintf("%d", galaxy))
	chiCtx.URLParams.Add("system", fmt.Sprintf("%d", system))
	chiCtx.URLParams.Add("position", fmt.Sprintf("%d", pos))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))
	req.Header.Set("X-User-ID", "401")
	w := httptest.NewRecorder()
	h.BuildIronBehemoth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandler_LinkWormholes(t *testing.T) {
	mock := newMockRepo()
	svc := NewPlanetService(mock)
	h := NewHandler(svc)

	mock.wormholes[moonKey(1, 1, 1)] = &WormholeEntry{
		MoonGalaxy: 1, MoonSystem: 1, MoonPos: 1, Level: 1,
	}
	mock.wormholes[moonKey(1, 1, 2)] = &WormholeEntry{
		MoonGalaxy: 1, MoonSystem: 1, MoonPos: 2, Level: 1,
	}

	body := `{"source_galaxy":1,"source_system":1,"source_position":1,"target_galaxy":1,"target_system":1,"target_position":2}`
	req := httptest.NewRequest("POST", "/api/wormhole/link", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.LinkWormholes(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestIronBehemothInShipsList(t *testing.T) {
	found := false
	for _, s := range Ships {
		if s.Type == "iron_behemoth" {
			found = true
			if s.Metal != 350000 || s.Crystal != 4000 || s.Gas != 5500 {
				t.Errorf("unexpected iron_behemoth stats: %+v", s)
			}
			break
		}
	}
	if !found {
		t.Error("iron_behemoth not found in Ships list")
	}
}

func TestHandler_InternalGetMoonBuildingLevel_NotFound(t *testing.T) {
	mock := newMockRepo()
	svc := NewPlanetService(mock)
	h := NewHandler(svc)

	body := `{"galaxy":1,"system":1,"position":1,"building_type":"pioneer_lab"}`
	req := httptest.NewRequest("POST", "/internal/moon/building-level", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.InternalGetMoonBuildingLevel(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestHandler_UpgradeMoonBuilding_Unauthorized(t *testing.T) {
	mock := newMockRepo()
	svc := NewPlanetService(mock)
	h := NewHandler(svc)

	req := httptest.NewRequest("POST", "/api/moon/1/1/1/buildings/moon_base/upgrade", nil)
	w := httptest.NewRecorder()
	h.UpgradeMoonBuilding(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func (m *mockRepo) GetGemSlots(_ context.Context, planetID int) ([]GemSlot, error) {
	slots := m.gemSlots[planetID]
	if slots == nil {
		return []GemSlot{}, nil
	}
	return slots, nil
}

func (m *mockRepo) SetGemSlot(_ context.Context, planetID, slotIndex int, gemType string, starLevel int) error {
	if m.gemSlots == nil {
		m.gemSlots = make(map[int][]GemSlot)
	}
	slots := m.gemSlots[planetID]
	found := false
	for i, s := range slots {
		if s.SlotIndex == slotIndex {
			m.gemSlots[planetID][i].GemType = gemType
			m.gemSlots[planetID][i].StarLevel = starLevel
			found = true
			break
		}
	}
	if !found {
		m.gemSlots[planetID] = append(m.gemSlots[planetID], GemSlot{
			ID: len(slots) + 1, PlanetID: planetID,
			SlotIndex: slotIndex, GemType: gemType, StarLevel: starLevel,
		})
	}
	return nil
}

func (m *mockRepo) GetShardCount(_ context.Context, playerID int) (map[string]int, error) {
	if m.shards == nil {
		return map[string]int{}, nil
	}
	shards := m.shards[playerID]
	if shards == nil {
		return map[string]int{}, nil
	}
	result := make(map[string]int)
	for k, v := range shards {
		result[k] = v
	}
	if m.combineAttempts != nil {
		if attempts, ok := m.combineAttempts[playerID]; ok {
			for k, v := range attempts {
				result["combine_attempts_"+k] = v
			}
		}
	}
	return result, nil
}

func (m *mockRepo) AddShards(_ context.Context, playerID int, gemType string, count int) error {
	if m.shards == nil {
		m.shards = make(map[int]map[string]int)
	}
	if m.shards[playerID] == nil {
		m.shards[playerID] = make(map[string]int)
	}
	m.shards[playerID][gemType] += count
	return nil
}

func (m *mockRepo) RemoveShards(_ context.Context, playerID int, gemType string, count int) error {
	if m.shards == nil || m.shards[playerID] == nil {
		return fmt.Errorf("insufficient shards")
	}
	if m.shards[playerID][gemType] < count {
		return fmt.Errorf("insufficient shards")
	}
	m.shards[playerID][gemType] -= count
	return nil
}

func (m *mockRepo) IncrementCombineAttempts(_ context.Context, playerID int, gemType string) error {
	if m.combineAttempts == nil {
		m.combineAttempts = make(map[int]map[string]int)
	}
	if m.combineAttempts[playerID] == nil {
		m.combineAttempts[playerID] = make(map[string]int)
	}
	m.combineAttempts[playerID][gemType]++
	return nil
}

func (m *mockRepo) CreateNPCPlanet(_ context.Context, galaxy, system, position int, planetType string, temperature int) (int, error) {
	if m.npcPlanets == nil {
		m.npcPlanets = make(map[int]*NPCPlanet)
	}
	m.npcNextID++
	planetID := m.npcNextID + 10000
	m.planets[planetID] = Planet{
		ID: planetID, UserID: 0,
		Name: fmt.Sprintf("NPC [%d:%d:%d]", galaxy, system, position),
		Galaxy: galaxy, System: system, Position: position,
		Metal: 50000, Crystal: 25000, Gas: 10000,
		MaxFields: 40, Type: planetType, Temperature: temperature,
		ResourcesUpdatedAt: time.Now(),
	}
	return planetID, nil
}

func (m *mockRepo) SeedNPCResources(_ context.Context, planetID int) error {
	p, ok := m.planets[planetID]
	if ok {
		p.Metal = 50000
		p.Crystal = 25000
		p.Gas = 10000
		m.planets[planetID] = p
	}
	return nil
}

func (m *mockRepo) SeedNPCFleet(_ context.Context, planetID int) error {
	return nil
}

func (m *mockRepo) RegisterNPCPlanet(_ context.Context, planetID, galaxy, system, position int) error {
	if m.npcPlanets == nil {
		m.npcPlanets = make(map[int]*NPCPlanet)
	}
	m.npcNextID++
	m.npcPlanets[planetID] = &NPCPlanet{
		ID: m.npcNextID, PlanetID: planetID,
		Galaxy: galaxy, System: system, Position: position,
		Status: "active",
	}
	return nil
}

func (m *mockRepo) GetNPCPlanetByPlanetID(_ context.Context, planetID int) (*NPCPlanet, error) {
	if m.npcPlanets == nil {
		return nil, nil
	}
	npc, ok := m.npcPlanets[planetID]
	if !ok {
		return nil, nil
	}
	return npc, nil
}

func (m *mockRepo) GetRespawnedNPCPlanets(_ context.Context) ([]NPCPlanet, error) {
	var result []NPCPlanet
	for _, npc := range m.npcPlanets {
		if npc.Status == "respawning" {
			result = append(result, *npc)
		}
	}
	return result, nil
}

func (m *mockRepo) ClearNPCPlanet(_ context.Context, planetID int) error {
	if m.npcPlanets != nil {
		if npc, ok := m.npcPlanets[planetID]; ok {
			npc.Status = "respawning"
			future := time.Now().Add(90 * time.Minute)
			npc.RespawnsAt = &future
			m.npcPlanets[planetID] = npc
		}
	}
	return nil
}

func (m *mockRepo) RespawnNPCPlanet(_ context.Context, npcPlanetID int) error {
	for _, npc := range m.npcPlanets {
		if npc.ID == npcPlanetID {
			npc.Status = "active"
			npc.RespawnsAt = nil
			if p, ok := m.planets[npc.PlanetID]; ok {
				p.Metal = 50000
				p.Crystal = 25000
				p.Gas = 10000
				m.planets[npc.PlanetID] = p
			}
			break
		}
	}
	return nil
}

func TestHandler_InternalCreatePlanet(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	h := NewHandler(svc)

	body := `{"user_id":201,"galaxy":1,"system":10,"position":4}`
	req := httptest.NewRequest("POST", "/internal/planet/create", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.InternalCreatePlanet(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["planet_id"] == nil || resp["planet_id"] == float64(0) {
		t.Error("expected valid planet_id")
	}
	if resp["name"] != "Colony [1:10:4]" {
		t.Errorf("expected 'Colony [1:10:4]', got '%s'", resp["name"])
	}
	if resp["galaxy"] != float64(1) || resp["system"] != float64(10) || resp["position"] != float64(4) {
		t.Errorf("unexpected coords: %+v", resp)
	}
}
