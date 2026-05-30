package main

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type mockRepo struct {
	planets    map[int]Planet
	buildings  map[int][]Building
	queue      map[int][]QueueEntry
	nextPID    int
	nextBID    int
	nextQID    int
	techLevels    map[int]map[string]int
	playerProgress map[int]struct{ vipPoints, totalResources int }
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
	if len(buildings) != 12 {
		t.Errorf("expected 12 buildings, got %d", len(buildings))
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
	if len(Ships) != 12 {
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

	quantity, err := svc.BuildShips(context.Background(), planet.ID, "cargo", 5)
	if err != nil {
		t.Fatal(err)
	}
	if quantity != 5 {
		t.Errorf("expected 5, got %d", quantity)
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

	_, err = svc.BuildShips(context.Background(), planet.ID, "dreadnought", 1)
	if err != ErrInsufficientResources {
		t.Errorf("expected ErrInsufficientResources, got %v", err)
	}
}

func TestBuildShips_InvalidShip(t *testing.T) {
	mock := newMockRepo()
	svc := NewPlanetService(mock)
	planet, _, err := svc.GetOrCreatePlanet(context.Background(), 82)
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.BuildShips(context.Background(), planet.ID, "death_star", 1)
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

	_, err = svc.BuildShips(context.Background(), planet.ID, "cargo", 1)
	if err != ErrNoShipyard {
		t.Errorf("expected ErrNoShipyard, got %v", err)
	}
}
