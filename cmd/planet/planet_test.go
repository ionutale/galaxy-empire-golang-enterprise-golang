package main

import (
	"context"
	"testing"
)

type mockRepo struct {
	planets map[int]Planet
	nextID  int
}

func newMockRepo() *mockRepo {
	return &mockRepo{planets: make(map[int]Planet), nextID: 1}
}

func (m *mockRepo) FindByUserID(_ context.Context, userID int) (Planet, error) {
	for _, p := range m.planets {
		if p.UserID == userID {
			return p, nil
		}
	}
	return Planet{}, ErrPlanetNotFound
}

func (m *mockRepo) Create(_ context.Context, userID int) (Planet, error) {
	p := Planet{
		ID: m.nextID, UserID: userID, Name: "Homeworld",
		Metal: 500, Crystal: 300, Gas: 200, Energy: 50,
		Galaxy: 1, System: 1, Position: 7,
	}
	m.nextID++
	m.planets[p.ID] = p
	return p, nil
}

func TestGetOrCreate_FirstCallCreates(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	p, err := svc.GetOrCreatePlanet(context.Background(), 42)
	if err != nil {
		t.Fatal("expected no error, got:", err)
	}
	if p.UserID != 42 {
		t.Errorf("expected user_id 42, got %d", p.UserID)
	}
	if p.Name != "Homeworld" {
		t.Errorf("expected Homeworld, got %s", p.Name)
	}
	if p.Metal != 500 || p.Crystal != 300 || p.Gas != 200 {
		t.Error("unexpected default resource values")
	}
}

func TestGetOrCreate_SecondCallReturnsExisting(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	p1, err := svc.GetOrCreatePlanet(context.Background(), 42)
	if err != nil {
		t.Fatal("first call failed:", err)
	}

	p2, err := svc.GetOrCreatePlanet(context.Background(), 42)
	if err != nil {
		t.Fatal("second call failed:", err)
	}

	if p1.ID != p2.ID {
		t.Errorf("expected same planet ID, got %d vs %d", p1.ID, p2.ID)
	}
	if p2.UserID != 42 {
		t.Errorf("expected user_id 42, got %d", p2.UserID)
	}
}

func TestGetOrCreate_DifferentUsersGetDifferentPlanets(t *testing.T) {
	svc := NewPlanetService(newMockRepo())
	p1, _ := svc.GetOrCreatePlanet(context.Background(), 1)
	p2, _ := svc.GetOrCreatePlanet(context.Background(), 2)

	if p1.ID == p2.ID {
		t.Error("expected different planet IDs for different users")
	}
}
