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

	types := []string{"metal_mine", "crystal_mine", "gas_mine", "solar_plant"}
	buildings := make([]Building, 0, 4)
	for _, t := range types {
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
	if len(buildings) != 4 {
		t.Errorf("expected 4 buildings, got %d", len(buildings))
	}
	types := make(map[string]int)
	for _, b := range buildings {
		types[b.Type] = b.Level
	}
	for _, typ := range []string{"metal_mine", "crystal_mine", "gas_mine", "solar_plant"} {
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

	initialResources := p.Metal + p.Crystal + p.Gas
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
	if totalResources <= initialResources {
		t.Errorf("resources should have increased: initial=%d, after=%d", initialResources, totalResources)
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
		{"solar_plant", 1, 20, 30},
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
	prod := svc.calculateProduction(buildings)

	if prod.Metal <= 0 {
		t.Error("expected positive metal production")
	}
	if prod.Crystal <= 0 {
		t.Error("expected positive crystal production")
	}
	if prod.Gas <= 0 {
		t.Error("expected positive gas production")
	}
	if prod.Energy <= 0 {
		t.Error("expected positive energy production")
	}

	rounded := math.Round(prod.Metal * 60)
	if rounded != 33 {
		t.Errorf("metal L1 should produce 33/min, got %.2f/min", prod.Metal*60)
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
