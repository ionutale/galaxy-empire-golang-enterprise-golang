package main

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"
	"time"
)

func newTestService() *ResearchService {
	repo := newMockRepo()
	svc := NewResearchService(repo, "http://fake-planet:8082")
	svc.httpPost = func(_ context.Context, url, body string) (string, error) {
		switch {
		case strings.Contains(url, "/internal/planet/coords"):
			return `{"galaxy":1,"system":1,"position":1}`, nil
		case strings.Contains(url, "/internal/planet/info"):
			return `{"player_id":1,"planet_id":1,"metal":0,"crystal":0,"gas":0,"ships":{}}`, nil
		case strings.Contains(url, "/internal/player/tech-level"):
			return `{"level":0}`, nil
		case strings.Contains(url, "/internal/planet/building-level"):
			return `{"level":1}`, nil
		case strings.Contains(url, "/internal/resources/deduct"):
			return `{"ok":true}`, nil
		case strings.Contains(url, "/internal/player/tech/add"):
			return `{"ok":true}`, nil
		case strings.Contains(url, "/internal/resources/add"):
			return `{"ok":true}`, nil
		}
		return `{"error":"unknown"}`, nil
	}
	return svc
}

func TestResearchDuration_DefaultLab(t *testing.T) {
	d := researchDuration(800, 400, 0)
	if d <= 0 {
		t.Error("expected positive duration")
	}
	if d < time.Hour {
		t.Errorf("expected at least 1 hour, got %v", d)
	}
}

func TestResearchDuration_WithLab(t *testing.T) {
	d1 := researchDuration(100000, 50000, 1)
	d5 := researchDuration(100000, 50000, 5)
	if d5 >= d1 {
		t.Errorf("higher lab should be faster: lab1=%v lab5=%v", d1, d5)
	}
}

func TestResearchDuration_Formula(t *testing.T) {
	metal, crystal := 800, 400
	lab := 1
	hours := float64(metal+crystal) / (1000.0 * float64(lab+1))
	if hours < 1 {
		hours = 1
	}
	expected := time.Duration(hours * float64(time.Hour))
	got := researchDuration(metal, crystal, lab)
	if got != expected {
		t.Errorf("expected %v, got %v", expected, got)
	}
}

func TestTechConfigs_AllDefined(t *testing.T) {
	if len(Techs) != 19 {
		t.Errorf("expected 19 techs, got %d", len(Techs))
	}
	for _, tc := range Techs {
		if tc.Type == "" || tc.Name == "" {
			t.Errorf("tech missing type or name: %+v", tc)
		}
		if tc.CostFactor <= 0 {
			t.Errorf("%s: cost factor must be positive", tc.Type)
		}
	}
}

func TestTechConfig_Lookup(t *testing.T) {
	cfg, ok := techConfig("energy_tech")
	if !ok {
		t.Fatal("expected energy_tech to be found")
	}
	if cfg.CostMetal != 800 || cfg.CostCrystal != 400 {
		t.Errorf("unexpected costs: metal=%d crystal=%d", cfg.CostMetal, cfg.CostCrystal)
	}
	_, ok = techConfig("nonexistent")
	if ok {
		t.Error("expected nonexistent to not be found")
	}
}

func TestTechConfig_CostAtLevel(t *testing.T) {
	cfg, _ := techConfig("laser_tech")
	level := 1
	expectedMetal := cfg.CostMetal * int(math.Pow(cfg.CostFactor, float64(level)))
	expectedCrystal := cfg.CostCrystal * int(math.Pow(cfg.CostFactor, float64(level)))
	if expectedMetal != 400 {
		t.Errorf("expected metal cost 400 at level 1, got %d", expectedMetal)
	}
	if expectedCrystal != 200 {
		t.Errorf("expected crystal cost 200 at level 1, got %d", expectedCrystal)
	}

	level = 2
	expectedMetal2 := cfg.CostMetal * int(math.Pow(cfg.CostFactor, float64(level)))
	if expectedMetal2 != 800 {
		t.Errorf("expected metal cost 800 at level 2, got %d", expectedMetal2)
	}
}

func TestBasicTechs_Researchable(t *testing.T) {
	basicTechs := []string{"energy_tech", "laser_tech", "ion_tech", "hyperspace_tech", "plasma_tech"}
	for _, name := range basicTechs {
		cfg, ok := techConfig(name)
		if !ok {
			t.Errorf("basic tech %s not found", name)
			continue
		}
		if cfg.Category != "basic" {
			t.Errorf("%s should be basic category", name)
		}
	}
}

func TestStartResearch_InvalidTech(t *testing.T) {
	svc := newTestService()
	_, err := svc.StartResearch(context.Background(), 1, 1, "nonexistent")
	if !errors.Is(err, ErrInvalidTech) {
		t.Errorf("expected ErrInvalidTech, got %v", err)
	}
}

func TestStartResearch_AlreadyResearching(t *testing.T) {
	svc := newTestService()
	_, err := svc.StartResearch(context.Background(), 1, 1, "energy_tech")
	if err != nil {
		t.Fatalf("first research failed: %v", err)
	}
	_, err = svc.StartResearch(context.Background(), 1, 1, "energy_tech")
	if !errors.Is(err, ErrAlreadyResearching) && !errors.Is(err, ErrResearchInProgress) {
		t.Errorf("expected ErrAlreadyResearching or ErrResearchInProgress, got %v", err)
	}
}

func TestCancelResearch_NoActive(t *testing.T) {
	svc := newTestService()
	_, err := svc.CancelResearch(context.Background(), 1, "energy_tech")
	if !errors.Is(err, ErrNoActiveResearch) {
		t.Errorf("expected ErrNoActiveResearch, got %v", err)
	}
}

func TestCancelResearch_Success(t *testing.T) {
	svc := newTestService()
	_, err := svc.StartResearch(context.Background(), 1, 1, "energy_tech")
	if err != nil {
		t.Fatalf("start research failed: %v", err)
	}

	resp, err := svc.CancelResearch(context.Background(), 1, "energy_tech")
	if err != nil {
		t.Fatalf("cancel research failed: %v", err)
	}
	if resp.RefundMetal <= 0 {
		t.Error("expected positive metal refund")
	}
	if resp.RefundCrystal <= 0 {
		t.Error("expected positive crystal refund")
	}
}

func TestProcessCompletedResearch(t *testing.T) {
	svc := newTestService()
	_, err := svc.StartResearch(context.Background(), 1, 1, "energy_tech")
	if err != nil {
		t.Fatalf("start research failed: %v", err)
	}

	repo := svc.repo.(*mockRepo)
	repo.mu.Lock()
	for i := range repo.queue {
		repo.queue[i].CompletesAt = time.Now().Add(-1 * time.Second)
	}
	repo.mu.Unlock()

	err = svc.ProcessCompleted(context.Background())
	if err != nil {
		t.Fatalf("process completed failed: %v", err)
	}
}

func TestListTechs_Basic(t *testing.T) {
	svc := newTestService()
	techs, labLevel, err := svc.ListTechs(context.Background(), 1)
	if err != nil {
		t.Fatalf("list techs failed: %v", err)
	}
	if len(techs) != 19 {
		t.Errorf("expected 19 techs, got %d", len(techs))
	}
	if labLevel < 1 {
		t.Errorf("expected lab level at least 1, got %d", labLevel)
	}
}

func TestStartResearch_ReturnsTargetLevelAndTime(t *testing.T) {
	svc := newTestService()
	resp, err := svc.StartResearch(context.Background(), 1, 1, "energy_tech")
	if err != nil {
		t.Fatalf("start research failed: %v", err)
	}
	if resp.TechType != "energy_tech" {
		t.Errorf("expected energy_tech, got %s", resp.TechType)
	}
	if resp.TargetLevel != 1 {
		t.Errorf("expected target level 1, got %d", resp.TargetLevel)
	}
	if resp.CompletesAt.Before(time.Now()) {
		t.Error("completes_at should be in the future")
	}
}

func TestAdvancedTechs_Researchable(t *testing.T) {
	advancedTechs := []string{"astrophysics", "computer_tech", "espionage_tech", "ultra_temperature", "anti_gravity"}
	for _, name := range advancedTechs {
		cfg, ok := techConfig(name)
		if !ok {
			t.Errorf("advanced tech %s not found", name)
			continue
		}
		if cfg.Category != "advanced" {
			t.Errorf("%s should be advanced category", name)
		}
		if cfg.Effect == "" {
			t.Errorf("%s missing effect description", name)
		}
	}
}

func TestCombatTechs_Researchable(t *testing.T) {
	combatTechs := []string{"combustion_drive", "impulse_drive", "hyperspace_drive", "weapons_tech", "shielding_tech", "strength_tech"}
	for _, name := range combatTechs {
		cfg, ok := techConfig(name)
		if !ok {
			t.Errorf("combat tech %s not found", name)
			continue
		}
		if cfg.Category != "combat" {
			t.Errorf("%s should be combat category", name)
		}
		if cfg.Effect == "" {
			t.Errorf("%s missing effect description", name)
		}
	}
}

func TestAllTechs_PrerequisitesDoNotIncludeSelf(t *testing.T) {
	for _, tc := range Techs {
		for _, p := range tc.Prerequisites {
			if p.Type == tc.Type {
				t.Errorf("%s: prerequisite cannot be itself", tc.Type)
			}
		}
	}
}

func TestAllTechs_EffectsDefined(t *testing.T) {
	for _, tc := range Techs {
		if tc.Effect == "" {
			t.Errorf("%s: effect is required", tc.Type)
		}
	}
}

func TestStartResearch_AdvancedTech_MissingPrereqs(t *testing.T) {
	svc := newTestService()
	_, err := svc.StartResearch(context.Background(), 1, 1, "astrophysics")
	if !errors.Is(err, ErrPrerequisitesNotMet) {
		t.Errorf("expected ErrPrerequisitesNotMet for astrophysics without energy_tech 4, got %v", err)
	}

	_, err = svc.StartResearch(context.Background(), 1, 1, "anti_gravity")
	if !errors.Is(err, ErrPrerequisitesNotMet) {
		t.Errorf("expected ErrPrerequisitesNotMet for anti_gravity without hyperspace_tech 3, got %v", err)
	}
}

func TestStartResearch_CombatTech_MissingPrereqs(t *testing.T) {
	svc := newTestService()
	_, err := svc.StartResearch(context.Background(), 1, 1, "combustion_drive")
	if !errors.Is(err, ErrPrerequisitesNotMet) {
		t.Errorf("expected ErrPrerequisitesNotMet for combustion_drive without research_lab 3, got %v", err)
	}

	_, err = svc.StartResearch(context.Background(), 1, 1, "shielding_tech")
	if !errors.Is(err, ErrPrerequisitesNotMet) {
		t.Errorf("expected ErrPrerequisitesNotMet for shielding_tech without research_lab 6, got %v", err)
	}
}

func TestStartResearch_AdvancedTech_WithSufficientLevels(t *testing.T) {
	svc := newTestService()
	svc.httpPost = func(_ context.Context, url, body string) (string, error) {
		switch {
		case strings.Contains(url, "/internal/planet/coords"):
			return `{"galaxy":1,"system":1,"position":1}`, nil
		case strings.Contains(url, "/internal/planet/info"):
			return `{"player_id":1,"planet_id":1,"metal":0,"crystal":0,"gas":0,"ships":{}}`, nil
		case strings.Contains(url, "/internal/player/tech-level"):
			if strings.Contains(body, "energy_tech") {
				return `{"level":4}`, nil
			}
			if strings.Contains(body, "hyperspace_tech") {
				return `{"level":3}`, nil
			}
			return `{"level":0}`, nil
		case strings.Contains(url, "/internal/planet/building-level"):
			return `{"level":7}`, nil
		case strings.Contains(url, "/internal/resources/deduct"):
			return `{"ok":true}`, nil
		case strings.Contains(url, "/internal/player/tech/add"):
			return `{"ok":true}`, nil
		}
		return `{"error":"unknown"}`, nil
	}

	resp, err := svc.StartResearch(context.Background(), 1, 1, "astrophysics")
	if err != nil {
		t.Fatalf("start astrophysics failed: %v", err)
	}
	if resp.TechType != "astrophysics" {
		t.Errorf("expected astrophysics, got %s", resp.TechType)
	}

	// Complete astrophysics before testing anti_gravity (global one-at-a-time queue)
	mock := svc.repo.(*mockRepo)
	mock.mu.Lock()
	for i := range mock.queue {
		if mock.queue[i].TechType == "astrophysics" {
			mock.queue[i].Completed = true
		}
	}
	mock.mu.Unlock()

	resp, err = svc.StartResearch(context.Background(), 1, 1, "anti_gravity")
	if err != nil {
		t.Fatalf("start anti_gravity failed: %v", err)
	}
	if resp.TechType != "anti_gravity" {
		t.Errorf("expected anti_gravity, got %s", resp.TechType)
	}
}

func TestStartResearch_CombatTech_WithSufficientLevels(t *testing.T) {
	svc := newTestService()
	svc.httpPost = func(_ context.Context, url, body string) (string, error) {
		switch {
		case strings.Contains(url, "/internal/planet/coords"):
			return `{"galaxy":1,"system":1,"position":1}`, nil
		case strings.Contains(url, "/internal/planet/info"):
			return `{"player_id":1,"planet_id":1,"metal":0,"crystal":0,"gas":0,"ships":{}}`, nil
		case strings.Contains(url, "/internal/player/tech-level"):
			if strings.Contains(body, "energy_tech") {
				return `{"level":4}`, nil
			}
			if strings.Contains(body, "laser_tech") {
				return `{"level":2}`, nil
			}
			if strings.Contains(body, "ion_tech") {
				return `{"level":3}`, nil
			}
			if strings.Contains(body, "combustion_drive") {
				return `{"level":3}`, nil
			}
			if strings.Contains(body, "hyperspace_tech") {
				return `{"level":3}`, nil
			}
			if strings.Contains(body, "shielding_tech") {
				return `{"level":3}`, nil
			}
			return `{"level":0}`, nil
		case strings.Contains(url, "/internal/planet/building-level"):
			return `{"level":6}`, nil
		case strings.Contains(url, "/internal/resources/deduct"):
			return `{"ok":true}`, nil
		case strings.Contains(url, "/internal/player/tech/add"):
			return `{"ok":true}`, nil
		}
		return `{"error":"unknown"}`, nil
	}

	// Test each combat tech for player 1; complete each research before starting the next
	combatTechs := []string{"combustion_drive", "impulse_drive", "hyperspace_drive", "weapons_tech", "shielding_tech", "strength_tech"}
	mock := svc.repo.(*mockRepo)
	for _, name := range combatTechs {
		resp, err := svc.StartResearch(context.Background(), 1, 1, name)
		if err != nil {
			t.Fatalf("start %s failed: %v", name, err)
		}
		if resp.TechType != name {
			t.Errorf("expected %s, got %s", name, resp.TechType)
		}
		// Complete this research so the next tech can be queued (one-at-a-time limit)
		mock.mu.Lock()
		for j := range mock.queue {
			if mock.queue[j].TechType == name {
				mock.queue[j].Completed = true
			}
		}
		mock.mu.Unlock()
	}
}

func TestStartResearch_ComputerTech_ResearchableAtStart(t *testing.T) {
	svc := newTestService()
	resp, err := svc.StartResearch(context.Background(), 1, 1, "computer_tech")
	if err != nil {
		t.Fatalf("start computer_tech failed: %v", err)
	}
	if resp.TechType != "computer_tech" {
		t.Errorf("expected computer_tech, got %s", resp.TechType)
	}
	if resp.TargetLevel != 1 {
		t.Errorf("expected target level 1, got %d", resp.TargetLevel)
	}
}

func TestMoonTechs_Defined(t *testing.T) {
	moonTechs := []string{"alloy_detection_tech", "dynamic_power_tech", "combined_guidance_tech"}
	if len(moonTechs) != 3 {
		t.Fatalf("expected 3 moon techs")
	}
	for _, name := range moonTechs {
		cfg, ok := techConfig(name)
		if !ok {
			t.Errorf("moon tech %s not found", name)
			continue
		}
		if cfg.Category != "moon" {
			t.Errorf("%s should be moon category, got %s", name, cfg.Category)
		}
		if cfg.Effect == "" {
			t.Errorf("%s missing effect description", name)
		}
		if cfg.ResearchLocation != "pioneer_lab" {
			t.Errorf("%s should have research_location pioneer_lab, got %s", name, cfg.ResearchLocation)
		}
	}
}

func TestMoonTechs_PrerequisiteChain(t *testing.T) {
	cfg, ok := techConfig("alloy_detection_tech")
	if !ok {
		t.Fatal("alloy_detection_tech not found")
	}
	if len(cfg.Prerequisites) != 0 {
		t.Errorf("alloy_detection_tech should have no prerequisites, got %v", cfg.Prerequisites)
	}

	cfg, ok = techConfig("dynamic_power_tech")
	if !ok {
		t.Fatal("dynamic_power_tech not found")
	}
	found := false
	for _, p := range cfg.Prerequisites {
		if p.Type == "alloy_detection_tech" && p.Level == 3 {
			found = true
		}
	}
	if !found {
		t.Error("dynamic_power_tech should require alloy_detection_tech Lv.3")
	}

	cfg, ok = techConfig("combined_guidance_tech")
	if !ok {
		t.Fatal("combined_guidance_tech not found")
	}
	found = false
	for _, p := range cfg.Prerequisites {
		if p.Type == "dynamic_power_tech" && p.Level == 3 {
			found = true
		}
	}
	if !found {
		t.Error("combined_guidance_tech should require dynamic_power_tech Lv.3")
	}
}

func TestMoonTechs_ResearchLocation(t *testing.T) {
	for _, tc := range Techs {
		if tc.Category == "moon" {
			if tc.ResearchLocation != "pioneer_lab" {
				t.Errorf("%s: expected pioneer_lab, got %s", tc.Type, tc.ResearchLocation)
			}
		} else {
			if tc.ResearchLocation != "research_lab" {
				t.Errorf("%s: expected research_lab, got %s", tc.Type, tc.ResearchLocation)
			}
		}
	}
}

func TestStartResearch_EspionageTech_ResearchableAtStart(t *testing.T) {
	svc := newTestService()
	resp, err := svc.StartResearch(context.Background(), 1, 1, "espionage_tech")
	if err != nil {
		t.Fatalf("start espionage_tech failed: %v", err)
	}
	if resp.TechType != "espionage_tech" {
		t.Errorf("expected espionage_tech, got %s", resp.TechType)
	}
	if resp.TargetLevel != 1 {
		t.Errorf("expected target level 1, got %d", resp.TargetLevel)
	}
}
