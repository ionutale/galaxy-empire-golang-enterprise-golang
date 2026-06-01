package main

import (
	"context"
	"testing"
)

func TestMockRepo_CreateExpedition(t *testing.T) {
	repo := newMockRepo()
	exp := Expedition{
		PlayerID:  1,
		Status:    "exploring",
		ShipsSent: map[string]int{"light_fighter": 10},
	}
	created, err := repo.CreateExpedition(context.Background(), exp)
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if created.PlayerID != 1 {
		t.Errorf("expected player 1, got %d", created.PlayerID)
	}
}

func TestMockRepo_GetExpedition(t *testing.T) {
	repo := newMockRepo()
	exp, _ := repo.CreateExpedition(context.Background(), Expedition{
		PlayerID:  1,
		Status:    "exploring",
		ShipsSent: map[string]int{"light_fighter": 10},
	})

	got, err := repo.GetExpedition(context.Background(), exp.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != exp.ID {
		t.Errorf("expected ID %d, got %d", exp.ID, got.ID)
	}
}

func TestMockRepo_GetExpedition_WrongPlayer(t *testing.T) {
	repo := newMockRepo()
	exp, _ := repo.CreateExpedition(context.Background(), Expedition{
		PlayerID:  1,
		Status:    "exploring",
		ShipsSent: map[string]int{"light_fighter": 10},
	})

	_, err := repo.GetExpedition(context.Background(), exp.ID, 2)
	if err == nil {
		t.Error("expected error for wrong player")
	}
}

func TestMockRepo_GetExpedition_NotFound(t *testing.T) {
	repo := newMockRepo()
	_, err := repo.GetExpedition(context.Background(), 999, 1)
	if err == nil {
		t.Error("expected error for non-existent expedition")
	}
}

func TestMockRepo_ListPlayerExpeditions(t *testing.T) {
	repo := newMockRepo()
	repo.CreateExpedition(context.Background(), Expedition{PlayerID: 1, Status: "exploring"})
	repo.CreateExpedition(context.Background(), Expedition{PlayerID: 1, Status: "completed"})
	repo.CreateExpedition(context.Background(), Expedition{PlayerID: 2, Status: "exploring"})

	list, err := repo.ListPlayerExpeditions(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 expeditions, got %d", len(list))
	}
}

func TestMockRepo_ListPlayerExpeditions_Empty(t *testing.T) {
	repo := newMockRepo()
	list, err := repo.ListPlayerExpeditions(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Errorf("expected 0 expeditions, got %d", len(list))
	}
}

func TestMockRepo_DMBalance_Zero(t *testing.T) {
	repo := newMockRepo()
	balance, totalEarned, err := repo.GetDarkMatterBalance(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if balance != 0 {
		t.Errorf("expected balance 0, got %d", balance)
	}
	if totalEarned != 0 {
		t.Errorf("expected totalEarned 0, got %d", totalEarned)
	}
}

func TestMockRepo_AddDarkMatter(t *testing.T) {
	repo := newMockRepo()
	if err := repo.AddDarkMatter(context.Background(), 1, 50); err != nil {
		t.Fatal(err)
	}
	balance, totalEarned, err := repo.GetDarkMatterBalance(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if balance != 50 {
		t.Errorf("expected balance 50, got %d", balance)
	}
	if totalEarned != 50 {
		t.Errorf("expected totalEarned 50, got %d", totalEarned)
	}
}

func TestMockRepo_AddDarkMatter_Accumulates(t *testing.T) {
	repo := newMockRepo()
	repo.AddDarkMatter(context.Background(), 1, 30)
	repo.AddDarkMatter(context.Background(), 1, 20)
	balance, totalEarned, _ := repo.GetDarkMatterBalance(context.Background(), 1)
	if balance != 50 {
		t.Errorf("expected balance 50, got %d", balance)
	}
	if totalEarned != 50 {
		t.Errorf("expected totalEarned 50, got %d", totalEarned)
	}
}

func TestMockRepo_SpendDarkMatter(t *testing.T) {
	repo := newMockRepo()
	repo.AddDarkMatter(context.Background(), 1, 100)
	if err := repo.SpendDarkMatter(context.Background(), 1, 30); err != nil {
		t.Fatal(err)
	}
	balance, _, _ := repo.GetDarkMatterBalance(context.Background(), 1)
	if balance != 70 {
		t.Errorf("expected balance 70, got %d", balance)
	}
}

func TestMockRepo_SpendDarkMatter_Insufficient(t *testing.T) {
	repo := newMockRepo()
	repo.AddDarkMatter(context.Background(), 1, 10)
	err := repo.SpendDarkMatter(context.Background(), 1, 20)
	if err == nil {
		t.Fatal("expected insufficient DM error")
	}
	if err.Error() != "insufficient dark matter" {
		t.Errorf("expected 'insufficient dark matter', got %v", err)
	}
}

func TestMockRepo_SpendDarkMatter_NoBalance(t *testing.T) {
	repo := newMockRepo()
	err := repo.SpendDarkMatter(context.Background(), 1, 10)
	if err == nil {
		t.Fatal("expected insufficient DM error")
	}
}

func TestMockRepo_UpdateExpeditionOutcome(t *testing.T) {
	repo := newMockRepo()
	exp, _ := repo.CreateExpedition(context.Background(), Expedition{
		PlayerID:  1,
		Status:    "exploring",
		ShipsSent: map[string]int{"light_fighter": 10},
	})

	err := repo.UpdateExpeditionOutcome(context.Background(), exp.ID, "resources",
		map[string]int{"metal": 50000},
		map[string]int{},
		map[string]int{},
		0)
	if err != nil {
		t.Fatal(err)
	}

	got, _ := repo.GetExpedition(context.Background(), exp.ID, 1)
	if got.Status != "completed" {
		t.Errorf("expected status completed, got %s", got.Status)
	}
	if got.Outcome != "resources" {
		t.Errorf("expected outcome resources, got %s", got.Outcome)
	}
	if got.ResourcesFound["metal"] != 50000 {
		t.Errorf("expected 50000 metal, got %d", got.ResourcesFound["metal"])
	}
}
