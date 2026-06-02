package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func planetServiceMock(handler func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/internal/planet/info":
			json.NewEncoder(w).Encode(map[string]any{
				"planet_id": 1,
				"player_id": 1,
				"metal":     10000,
				"crystal":   5000,
				"gas":       3000,
				"ships": map[string]int{
					"light_fighter":  100,
					"heavy_fighter":  50,
					"cruiser":        20,
					"cargo":          200,
					"recycler":       50,
				},
			})
		case "/internal/player/techs":
			json.NewEncoder(w).Encode(map[string]any{
				"technologies": map[string]int{"espionage_tech": 5},
			})
		default:
			handler(w, r)
		}
	}))
}

func TestGenerateOutcome_Resources(t *testing.T) {
	ships := map[string]int{"cargo": 50}
	foundResources := false
	for i := 0; i < 1000; i++ {
		outcome := generateOutcome(ships, 0)
		if outcome.Outcome == "resources" {
			foundResources = true
			if outcome.ResourcesFound["metal"] < 10000 || outcome.ResourcesFound["crystal"] < 10000 || outcome.ResourcesFound["gas"] < 10000 {
				t.Error("resources should be at least 10000 each")
			}
			if outcome.ResourcesFound["metal"] > 2000000 || outcome.ResourcesFound["crystal"] > 2000000 || outcome.ResourcesFound["gas"] > 2000000 {
				t.Error("resources should not exceed 2000000 each")
			}
			if outcome.DarkMatter != 0 {
				t.Error("resources outcome should have 0 dark matter")
			}
			if len(outcome.ShipsFound) != 0 {
				t.Error("resources outcome should have 0 ships found")
			}
			break
		}
	}
	if !foundResources {
		t.Error("expected at least one resources outcome in 1000 rolls")
	}
}

func TestGenerateOutcome_Ships(t *testing.T) {
	ships := map[string]int{"light_fighter": 100}
	foundShips := false
	for i := 0; i < 1000; i++ {
		outcome := generateOutcome(ships, 0)
		if outcome.Outcome == "ships" {
			foundShips = true
			if len(outcome.ShipsFound) == 0 {
				t.Error("ships outcome should have ships found")
			}
			if outcome.DarkMatter != 0 {
				t.Error("ships outcome should have 0 dark matter")
			}
			if len(outcome.ResourcesFound) != 0 {
				t.Error("ships outcome should have 0 resources found")
			}
			break
		}
	}
	if !foundShips {
		t.Error("expected at least one ships outcome in 1000 rolls")
	}
}

func TestGenerateOutcome_DarkMatter(t *testing.T) {
	ships := map[string]int{"battleship": 200}
	foundDM := false
	for i := 0; i < 2000; i++ {
		outcome := generateOutcome(ships, 0)
		if outcome.Outcome == "dark_matter" {
			foundDM = true
			if outcome.DarkMatter < 20 || outcome.DarkMatter > 50 {
				t.Errorf("large fleet dark matter should be 20-50, got %d", outcome.DarkMatter)
			}
			if len(outcome.ResourcesFound) != 0 {
				t.Error("dark matter outcome should have 0 resources found")
			}
			if len(outcome.ShipsFound) != 0 {
				t.Error("dark matter outcome should have 0 ships found")
			}
			break
		}
	}
	if !foundDM {
		t.Error("expected at least one dark_matter outcome in 2000 rolls")
	}
}

func TestGenerateOutcome_Pirates(t *testing.T) {
	ships := map[string]int{"light_fighter": 100, "cargo": 50}
	foundPirates := false
	for i := 0; i < 2000; i++ {
		outcome := generateOutcome(ships, 0)
		if outcome.Outcome == "pirates" {
			foundPirates = true
			if len(outcome.ShipsLost) == 0 {
				t.Error("pirates outcome should have ships lost")
			}
			for shipType, qty := range outcome.ShipsLost {
				original := ships[shipType]
				if qty > original {
					t.Errorf("lost %d but only had %d of %s", qty, original, shipType)
				}
				if qty < 1 && original > 0 {
					t.Errorf("should lose at least 1 ship of %s", shipType)
				}
			}
			if len(outcome.ResourcesFound) != 0 {
				t.Error("pirates outcome should have 0 resources found")
			}
			break
		}
	}
	if !foundPirates {
		t.Error("expected at least one pirates outcome in 2000 rolls")
	}
}

func TestGenerateOutcome_Aliens(t *testing.T) {
	ships := map[string]int{"recycler": 100}
	foundAliens := false
	for i := 0; i < 2000; i++ {
		outcome := generateOutcome(ships, 0)
		if outcome.Outcome == "aliens" {
			foundAliens = true
			if len(outcome.ResourcesFound) == 0 {
				t.Error("aliens outcome should have resources found")
			}
			if outcome.DarkMatter != 0 {
				t.Error("aliens outcome should have 0 dark matter")
			}
			break
		}
	}
	if !foundAliens {
		t.Error("expected at least one aliens outcome in 2000 rolls")
	}
}

func TestEnemyScaling_PiratesFleet(t *testing.T) {
	ships := map[string]int{"light_fighter": 10, "cruiser": 3}
	outcome := generatePiratesOutcome(ships)

	if outcome.EnemyFleet["light_fighter"] != 7 {
		t.Errorf("expected 7 pirate light_fighters (70%% of 10), got %d", outcome.EnemyFleet["light_fighter"])
	}
	if outcome.EnemyFleet["cruiser"] != 2 {
		t.Errorf("expected 2 pirate cruisers (70%% of 3), got %d", outcome.EnemyFleet["cruiser"])
	}
	if outcome.Outcome != "pirates" {
		t.Errorf("expected pirates outcome, got %s", outcome.Outcome)
	}
}

func TestEnemyScaling_AliensFleet(t *testing.T) {
	ships := map[string]int{"light_fighter": 10}
	outcome := generateAliensOutcome(ships, 10)

	if outcome.EnemyFleet["light_fighter"] != 13 {
		t.Errorf("expected 13 alien light_fighters (130%% of 10), got %d", outcome.EnemyFleet["light_fighter"])
	}
	if outcome.Outcome != "aliens" {
		t.Errorf("expected aliens outcome, got %s", outcome.Outcome)
	}
}

func TestEnemyScaling_RecyclerOptimization(t *testing.T) {
	ships := map[string]int{"recycler": 10}
	outcome := generatePiratesOutcome(ships)

	if len(outcome.ShipsLost) == 0 {
		t.Error("pirates should still destroy ships even with only recyclers")
	}
	if outcome.EnemyFleet["recycler"] != 7 {
		t.Errorf("expected 7 pirate recyclers, got %d", outcome.EnemyFleet["recycler"])
	}
}

func TestDarkMatterScaling_SmallFleet(t *testing.T) {
	for i := 0; i < 100; i++ {
		outcome := generateDarkMatterOutcome(10)
		if outcome.DarkMatter < 5 || outcome.DarkMatter > 15 {
			t.Errorf("small fleet DM should be 5-15, got %d", outcome.DarkMatter)
		}
	}
}

func TestDarkMatterScaling_MediumFleet(t *testing.T) {
	for i := 0; i < 100; i++ {
		outcome := generateDarkMatterOutcome(75)
		if outcome.DarkMatter < 10 || outcome.DarkMatter > 30 {
			t.Errorf("medium fleet DM should be 10-30, got %d", outcome.DarkMatter)
		}
	}
}

func TestDarkMatterScaling_LargeFleet(t *testing.T) {
	for i := 0; i < 100; i++ {
		outcome := generateDarkMatterOutcome(200)
		if outcome.DarkMatter < 20 || outcome.DarkMatter > 50 {
			t.Errorf("large fleet DM should be 20-50, got %d", outcome.DarkMatter)
		}
	}
}

func TestGenerateOutcome_Nothing(t *testing.T) {
	ships := map[string]int{"espionage_probe": 1}
	foundNothing := false
	for i := 0; i < 1000; i++ {
		outcome := generateOutcome(ships, 0)
		if outcome.Outcome == "nothing" {
			foundNothing = true
			if len(outcome.ResourcesFound) != 0 {
				t.Error("nothing outcome should have 0 resources found")
			}
			if len(outcome.ShipsFound) != 0 {
				t.Error("nothing outcome should have 0 ships found")
			}
			if len(outcome.ShipsLost) != 0 {
				t.Error("nothing outcome should have 0 ships lost")
			}
			if outcome.DarkMatter != 0 {
				t.Error("nothing outcome should have 0 dark matter")
			}
			break
		}
	}
	if !foundNothing {
		t.Error("expected at least one nothing outcome in 1000 rolls")
	}
}

func TestGenerateOutcome_EspionageTechBoostsProbabilities(t *testing.T) {
	lowTechResources := 0
	highTechResources := 0
	iterations := 10000

	for i := 0; i < iterations; i++ {
		o := generateOutcome(map[string]int{"cargo": 10}, 0)
		if o.Outcome == "resources" {
			lowTechResources++
		}
	}
	for i := 0; i < iterations; i++ {
		o := generateOutcome(map[string]int{"cargo": 10}, 10)
		if o.Outcome == "resources" {
			highTechResources++
		}
	}

	if highTechResources <= lowTechResources {
		t.Error("expected higher tech to produce more resources outcomes")
	}
}

func TestGenerateOutcome_TotalShipsScaling(t *testing.T) {
	// 667 recyclers should give max resources
	outcomeSmall := generateResourcesOutcome(10)
	outcomeLarge := generateResourcesOutcome(667)

	if outcomeSmall.ResourcesFound["metal"] > outcomeLarge.ResourcesFound["metal"] {
		t.Error("expected more ships to produce more resources")
	}
}

func TestGenerateOutcome_EmptyShips(t *testing.T) {
	outcome := generateOutcome(map[string]int{}, 0)
	if outcome.Outcome != "nothing" {
		t.Error("empty ships should always produce nothing")
	}
}

func TestCalculateSpeedUpCost(t *testing.T) {
	tests := []struct {
		seconds  int
		expected int
	}{
		{1, 1},
		{899, 1},
		{900, 1},
		{901, 2},
		{1800, 2},
		{2700, 3},
		{3600, 4},
	}
	svc := NewNebulaService(newMockRepo(), "http://localhost:8082", "http://localhost:8085")
	for _, tt := range tests {
		cost := svc.CalculateSpeedUpCost(tt.seconds)
		if cost != tt.expected {
			t.Errorf("CalculateSpeedUpCost(%d) = %d, want %d", tt.seconds, cost, tt.expected)
		}
	}
}

func TestSpeedUp_DeductsDM(t *testing.T) {
	planetSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer planetSrv.Close()

	repo := newMockRepo()
	repo.AddDarkMatter(context.Background(), 1, 100)
	svc := NewNebulaService(repo, planetSrv.URL, "http://localhost:8085")
	cost, saved, err := svc.SpeedUp(context.Background(), 1, "building", 1, 900)
	if err != nil {
		t.Fatal(err)
	}
	if cost != 1 {
		t.Errorf("expected cost 1, got %d", cost)
	}
	if saved != 900 {
		t.Errorf("expected saved 900, got %d", saved)
	}
	balance, _, _ := repo.GetDarkMatterBalance(context.Background(), 1)
	if balance != 99 {
		t.Errorf("expected balance 99, got %d", balance)
	}
	txs, _ := repo.ListDMTransactions(context.Background(), 1, 10)
	if len(txs) != 1 {
		t.Errorf("expected 1 transaction, got %d", len(txs))
	}
	if txs[0].Reason != "speed_up" {
		t.Errorf("expected reason speed_up, got %s", txs[0].Reason)
	}
}

func TestSpeedUp_InsufficientDM(t *testing.T) {
	svc := NewNebulaService(newMockRepo(), "http://localhost:8082", "http://localhost:8085")
	_, _, err := svc.SpeedUp(context.Background(), 1, "building", 1, 900)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "insufficient dark matter" {
		t.Errorf("expected insufficient dark matter, got %v", err)
	}
}

func TestSpendDarkMatter_LogsTransaction(t *testing.T) {
	repo := newMockRepo()
	repo.AddDarkMatter(context.Background(), 1, 100)
	svc := NewNebulaService(repo, "http://localhost:8082", "http://localhost:8085")
	balance, err := svc.SpendDarkMatter(context.Background(), 1, 30, "test_reason")
	if err != nil {
		t.Fatal(err)
	}
	if balance != 70 {
		t.Errorf("expected balance 70, got %d", balance)
	}
	txs, _ := repo.ListDMTransactions(context.Background(), 1, 10)
	if len(txs) != 1 {
		t.Errorf("expected 1 transaction, got %d", len(txs))
	}
	if txs[0].Amount != -30 {
		t.Errorf("expected amount -30, got %d", txs[0].Amount)
	}
	if txs[0].BalanceAfter != 70 {
		t.Errorf("expected balance_after 70, got %d", txs[0].BalanceAfter)
	}
	if txs[0].Reason != "test_reason" {
		t.Errorf("expected reason test_reason, got %s", txs[0].Reason)
	}
}

func TestHireCommander_Valid(t *testing.T) {
	repo := newMockRepo()
	repo.AddDarkMatter(context.Background(), 1, 100)
	svc := NewNebulaService(repo, "http://localhost:8082", "http://localhost:8085")
	entry, err := svc.HireCommander(context.Background(), 1, "commander")
	if err != nil {
		t.Fatal(err)
	}
	if entry.CommanderType != "commander" {
		t.Errorf("expected type commander, got %s", entry.CommanderType)
	}
	if entry.Level != 1 {
		t.Errorf("expected level 1, got %d", entry.Level)
	}
	if entry.Name != "Commander" {
		t.Errorf("expected name Commander, got %s", entry.Name)
	}
	if entry.DaysRemaining <= 0 {
		t.Errorf("expected positive days_remaining, got %d", entry.DaysRemaining)
	}
	balance, _, _ := repo.GetDarkMatterBalance(context.Background(), 1)
	if balance != 75 {
		t.Errorf("expected balance 75 after spending 25 DM, got %d", balance)
	}
}

func TestServiceHireCommander_InvalidType(t *testing.T) {
	svc := NewNebulaService(newMockRepo(), "http://localhost:8082", "http://localhost:8085")
	_, err := svc.HireCommander(context.Background(), 1, "nonexistent")
	if err == nil || err.Error() != "unknown commander type: nonexistent" {
		t.Errorf("expected unknown commander type error, got %v", err)
	}
}

func TestHireCommander_InsufficientDM(t *testing.T) {
	repo := newMockRepo()
	repo.AddDarkMatter(context.Background(), 1, 10)
	svc := NewNebulaService(repo, "http://localhost:8082", "http://localhost:8085")
	_, err := svc.HireCommander(context.Background(), 1, "commander")
	if err == nil || err.Error() != "insufficient dark matter" {
		t.Errorf("expected insufficient dark matter error, got %v", err)
	}
}

func TestHireCommander_AlreadyHired(t *testing.T) {
	repo := newMockRepo()
	repo.AddDarkMatter(context.Background(), 1, 100)
	svc := NewNebulaService(repo, "http://localhost:8082", "http://localhost:8085")
	_, err := svc.HireCommander(context.Background(), 1, "commander")
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.HireCommander(context.Background(), 1, "commander")
	if err == nil {
		t.Fatal("expected error for duplicate hire")
	}
}

func TestGetPlayerCommanders_ActiveOnly(t *testing.T) {
	repo := newMockRepo()
	repo.AddDarkMatter(context.Background(), 1, 100)
	svc := NewNebulaService(repo, "http://localhost:8082", "http://localhost:8085")
	svc.HireCommander(context.Background(), 1, "commander")
	entries, err := svc.GetPlayerCommanders(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 active commander, got %d", len(entries))
	}
}

func TestGetPlayerCommanders_NoExpired(t *testing.T) {
	repo := newMockRepo()
	// add an expired commander directly
	repo.commanders = append(repo.commanders, CommanderEntry{
		PlayerID:      1,
		CommanderType: "commander",
		Level:         1,
		HiredAt:       time.Now().AddDate(0, 0, -30),
		ExpiresAt:     time.Now().AddDate(0, 0, -23),
	})
	svc := NewNebulaService(repo, "http://localhost:8082", "http://localhost:8085")
	entries, err := svc.GetPlayerCommanders(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 active commanders (all expired), got %d", len(entries))
	}
}

func TestGetAvailableCommanders(t *testing.T) {
	svc := NewNebulaService(newMockRepo(), "http://localhost:8082", "http://localhost:8085")
	commanders := svc.GetAvailableCommanders()
	if len(commanders) != 6 {
		t.Errorf("expected 6 commanders, got %d", len(commanders))
	}
	commanderMap := make(map[string]CommanderConfig)
	for _, c := range commanders {
		commanderMap[c.Type] = c
	}
	if _, ok := commanderMap["commander"]; !ok {
		t.Error("expected commander type to be available")
	}
	if commanderMap["commander"].DMCost != 25 {
		t.Errorf("expected commander DM cost 25, got %d", commanderMap["commander"].DMCost)
	}
}

func TestServiceInternalActiveCommanders(t *testing.T) {
	repo := newMockRepo()
	repo.AddDarkMatter(context.Background(), 1, 100)
	svc := NewNebulaService(repo, "http://localhost:8082", "http://localhost:8085")
	svc.HireCommander(context.Background(), 1, "engineer")
	entries, err := svc.GetActiveCommandersRaw(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 active commander, got %d", len(entries))
	}
	if entries[0].CommanderType != "engineer" {
		t.Errorf("expected engineer, got %s", entries[0].CommanderType)
	}
}

func TestGetPlayerCommanders_NamesAndDescriptionsSet(t *testing.T) {
	repo := newMockRepo()
	repo.AddDarkMatter(context.Background(), 1, 100)
	svc := NewNebulaService(repo, "http://localhost:8082", "http://localhost:8085")
	svc.HireCommander(context.Background(), 1, "geologist")
	entries, err := svc.GetPlayerCommanders(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1, got %d", len(entries))
	}
	if entries[0].Name != "Geologist" {
		t.Errorf("expected name Geologist, got %s", entries[0].Name)
	}
	if entries[0].Description == "" {
		t.Error("expected non-empty description")
	}
}

func TestStartExpedition_NoShips(t *testing.T) {
	svc := NewNebulaService(newMockRepo(), "http://localhost:8082", "http://localhost:8085")
	_, err := svc.StartExpedition(context.Background(), 1, 1, map[string]int{})
	if err == nil || !strings.Contains(err.Error(), "no ships") {
		t.Fatalf("expected no ships error, got: %v", err)
	}
}

func TestStartExpedition_UnknownShip(t *testing.T) {
	svc := NewNebulaService(newMockRepo(), "http://localhost:8082", "http://localhost:8085")
	_, err := svc.StartExpedition(context.Background(), 1, 1, map[string]int{"death_star": 1})
	if err == nil || !strings.Contains(err.Error(), "unknown ship") {
		t.Fatalf("expected unknown ship error, got: %v", err)
	}
}

func TestStartExpedition_InsufficientShips(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"planet_id": 1,
			"player_id": 1,
			"metal":     10000,
			"crystal":   5000,
			"gas":       3000,
			"ships":     map[string]int{"light_fighter": 5, "cargo": 0},
		})
	}))
	defer ts.Close()

	svc := NewNebulaService(newMockRepo(), ts.URL, "http://localhost:8085")
	_, err := svc.StartExpedition(context.Background(), 1, 1, map[string]int{"light_fighter": 100, "cargo": 50})
	if err == nil || !strings.Contains(err.Error(), "insufficient") {
		t.Fatalf("expected insufficient ships error, got: %v", err)
	}
}

func TestStartExpedition_PlanetServiceUnreachable(t *testing.T) {
	svc := NewNebulaService(newMockRepo(), "http://localhost:1", "http://localhost:8085")
	_, err := svc.StartExpedition(context.Background(), 1, 1, map[string]int{"light_fighter": 10})
	if err == nil || !strings.Contains(err.Error(), "planet service") {
		t.Fatalf("expected planet service error, got: %v", err)
	}
}

func TestStartExpedition_Success(t *testing.T) {
	callCount := 0
	ts := planetServiceMock(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	})
	defer ts.Close()

	svc := NewNebulaService(newMockRepo(), ts.URL, "http://localhost:8085")
	expedition, err := svc.StartExpedition(context.Background(), 1, 1, map[string]int{"light_fighter": 10})
	if err != nil {
		t.Fatal(err)
	}
	if expedition.PlayerID != 1 {
		t.Fatalf("expected player 1, got %d", expedition.PlayerID)
	}
	if expedition.Status != "completed" {
		t.Fatalf("expected status completed, got %s", expedition.Status)
	}
	if expedition.Outcome == "" {
		t.Fatal("expected a non-empty outcome")
	}
	if callCount < 1 {
		t.Error("expected at least one planet service call")
	}
}

func TestStartExpedition_ReturnsShipsAfterExpedition(t *testing.T) {
	ts := planetServiceMock(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	})
	defer ts.Close()

	svc := NewNebulaService(newMockRepo(), ts.URL, "http://localhost:8085")
	ships := map[string]int{"light_fighter": 10, "cargo": 5}
	expedition, err := svc.StartExpedition(context.Background(), 1, 1, ships)
	if err != nil {
		t.Fatal(err)
	}

	if len(expedition.ShipsSent) == 0 {
		t.Error("expected ships sent to be recorded")
	}

	totalLost := 0
	for _, qty := range expedition.ShipsLost {
		totalLost += qty
	}

	if totalLost < 0 {
		t.Error("negative ships lost")
	}
}

func TestStartExpedition_RecordsOutcome(t *testing.T) {
	ts := planetServiceMock(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	})
	defer ts.Close()

	svc := NewNebulaService(newMockRepo(), ts.URL, "http://localhost:8085")
	expedition, err := svc.StartExpedition(context.Background(), 1, 1, map[string]int{"recycler": 50})
	if err != nil {
		t.Fatal(err)
	}

	validOutcomes := map[string]bool{
		"resources": true, "ships": true, "dark_matter": true,
		"pirates": true, "aliens": true, "nothing": true,
	}
	if !validOutcomes[expedition.Outcome] {
		t.Fatalf("invalid outcome: %s", expedition.Outcome)
	}
}
