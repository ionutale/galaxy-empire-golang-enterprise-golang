package main

import (
	"context"
	"math"
	"testing"
	"time"
)

type mockRepo struct {
	planets   map[int]Planet
	buildings map[int][]Building
	nextPID   int
	nextBID   int
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		planets:   make(map[int]Planet),
		buildings: make(map[int][]Building),
		nextPID:   1,
		nextBID:   1,
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

func (m *mockRepo) Create(_ context.Context, userID int) (Planet, []Building, error) {
	now := time.Now()
	p := Planet{
		ID: m.nextPID, UserID: userID, Name: "Homeworld",
		Metal: 500, Crystal: 300, Gas: 200, Energy: 50,
		Galaxy: 1, System: 1, Position: 7,
		ResourcesUpdatedAt: now,
	}
	m.nextPID++
	m.planets[p.ID] = p

	seedTypes := []string{
		"metal_mine", "crystal_mine", "gas_mine", "solar_plant",
		"metal_storage", "crystal_storage", "gas_storage",
	}
	buildings := make([]Building, 0, len(seedTypes))
	for _, t := range seedTypes {
		b := Building{ID: m.nextBID, PlanetID: p.ID, Type: t, Level: 1}
		m.nextBID++
		buildings = append(buildings, b)
	}
	m.buildings[p.ID] = buildings
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

func (m *mockRepo) GetBuildings(_ context.Context, planetID int) ([]Building, error) {
	return m.buildings[planetID], nil
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
	if p.Name != "Homeworld" {
		t.Errorf("expected Homeworld, got %s", p.Name)
	}
	if len(buildings) != 7 {
		t.Errorf("expected 7 buildings, got %d", len(buildings))
	}
	types := make(map[string]int)
	for _, b := range buildings {
		types[b.Type] = b.Level
	}
	expected := []string{"metal_mine", "crystal_mine", "gas_mine", "solar_plant",
		"metal_storage", "crystal_storage", "gas_storage"}
	for _, typ := range expected {
		if types[typ] != 1 {
			t.Errorf("expected %s level 1, got %d", typ, types[typ])
		}
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
		t.Error("resources_updated_at should have advanced after second call")
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
	prod := svc.calculateProduction(buildings, 1.0)

	if prod.Metal <= 0 {
		t.Error("expected positive metal production")
	}
	rounded := math.Round(prod.Metal * 60)
	if rounded != 33 {
		t.Errorf("metal L1 should produce 33/min, got %.2f/min", prod.Metal*60)
	}
}

func TestCalculateProduction_WithPenalty(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	buildings := []Building{
		{Type: "metal_mine", Level: 5},
		{Type: "crystal_mine", Level: 5},
		{Type: "gas_mine", Level: 5},
	}
	prod := svc.calculateProduction(buildings, 0.5)

	if prod.Metal <= 0 {
		t.Error("expected positive metal production even with penalty")
	}
	expectedHalf := productionRate("metal_mine", 5) / 60.0 * 0.5
	if math.Abs(prod.Metal-expectedHalf) > 0.01 {
		t.Errorf("expected ~%.4f metal/s with 0.5 penalty, got %.4f", expectedHalf, prod.Metal)
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

func TestStorageCapacity_Default(t *testing.T) {
	cap := storageCapacity("metal_storage", 0)
	if cap != baseStorage {
		t.Errorf("expected base storage %d, got %d", baseStorage, cap)
	}
}

func TestStorageCapacity_Level1(t *testing.T) {
	cap := storageCapacity("metal_storage", 1)
	if cap <= baseStorage {
		t.Errorf("level 1 should exceed base storage %d, got %d", baseStorage, cap)
	}
	expected := baseStorage + int(5000*math.Pow(1.5, 1))
	if cap != expected {
		t.Errorf("expected %d, got %d", expected, cap)
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
	if storage.Crystal <= baseStorage {
		t.Error("crystal storage should exceed base")
	}
	if storage.Gas <= baseStorage {
		t.Error("gas storage should exceed base")
	}
}

func TestStorage_NoStorageBuildings(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	storage := svc.calculateStorage(nil)

	if storage.Metal != baseStorage {
		t.Errorf("expected %d for missing metal_storage, got %d", baseStorage, storage.Metal)
	}
	if storage.Crystal != baseStorage {
		t.Errorf("expected %d for missing crystal_storage, got %d", baseStorage, storage.Crystal)
	}
	if storage.Gas != baseStorage {
		t.Errorf("expected %d for missing gas_storage, got %d", baseStorage, storage.Gas)
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
	if storedPlanet.Crystal > storage.Crystal {
		t.Errorf("crystal %d exceeds storage %d", storedPlanet.Crystal, storage.Crystal)
	}
	if storedPlanet.Gas > storage.Gas {
		t.Errorf("gas %d exceeds storage %d", storedPlanet.Gas, storage.Gas)
	}
}

func TestCalculatePenaltyFactor_PositiveNet(t *testing.T) {
	buildings := []Building{
		{Type: "metal_mine", Level: 1},   // consumes 10
		{Type: "solar_plant", Level: 1},  // produces 44
	}
	netEnergy, efficiency := calculatePenaltyFactor(buildings)
	if netEnergy <= 0 {
		t.Errorf("expected positive net energy, got %d", netEnergy)
	}
	if efficiency != 1.0 {
		t.Errorf("expected efficiency 1.0 for positive net, got %.2f", efficiency)
	}
}

func TestCalculatePenaltyFactor_NegativeNet(t *testing.T) {
	buildings := []Building{
		{Type: "metal_mine", Level: 5},   // consumes 50
		{Type: "crystal_mine", Level: 5}, // consumes 50
		{Type: "gas_mine", Level: 5},     // consumes 100
		{Type: "solar_plant", Level: 1},  // produces 44
	}
	netEnergy, efficiency := calculatePenaltyFactor(buildings)
	if netEnergy >= 0 {
		t.Errorf("expected negative net energy, got %d", netEnergy)
	}
	if efficiency >= 1.0 {
		t.Errorf("expected efficiency < 1.0 for negative net, got %.2f", efficiency)
	}
	if efficiency <= 0 {
		t.Errorf("expected positive efficiency, got %.2f", efficiency)
	}
}

func TestCalculatePenaltyFactor_SolarOnly(t *testing.T) {
	buildings := []Building{
		{Type: "solar_plant", Level: 3},
	}
	netEnergy, efficiency := calculatePenaltyFactor(buildings)
	if netEnergy <= 0 {
		t.Errorf("expected positive net for solar only, got %d", netEnergy)
	}
	if efficiency != 1.0 {
		t.Errorf("expected 1.0 efficiency for solar only, got %.2f", efficiency)
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
		{"unknown", 1, 0},
		{"metal_mine", 0, 0},
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
	expected := 44.0
	if rate != expected {
		t.Errorf("expected solar plant L1 production %.0f/h, got %.2f", expected, rate)
	}
}
