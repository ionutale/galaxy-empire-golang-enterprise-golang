package main

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"
)

var (
	ErrInvalidTech         = errors.New("invalid tech type")
	ErrPrerequisitesNotMet = errors.New("prerequisites not met")
	ErrAlreadyResearching  = errors.New("already researching this tech")
	ErrNoActiveResearch    = errors.New("no active research for this tech")
	ErrTechNotFound        = errors.New("tech not found")
)

type httpPostFunc func(ctx context.Context, url, body string) (string, error)

type ResearchService struct {
	repo       Repository
	planetAddr string
	httpPost   httpPostFunc
}

func NewResearchService(repo Repository, planetAddr string) *ResearchService {
	svc := &ResearchService{repo: repo, planetAddr: planetAddr}
	svc.httpPost = svc.httpPostDefault
	return svc
}

func (s *ResearchService) httpPostDefault(ctx context.Context, url, body string) (string, error) {
	req, err := httpPostRequest(ctx, url, body)
	if err != nil {
		return "", err
	}
	return httpDo(req)
}

func (s *ResearchService) ListTechs(ctx context.Context, playerID int) ([]TechWithStatus, int, error) {
	techLevels, err := s.fetchTechLevels(ctx, playerID)
	if err != nil {
		return nil, 0, fmt.Errorf("fetch tech levels: %w", err)
	}

	activeResearch, err := s.repo.ListActiveResearch(ctx, playerID)
	if err != nil {
		return nil, 0, fmt.Errorf("list active: %w", err)
	}

	researching := make(map[string]bool)
	for _, r := range activeResearch {
		researching[r.TechType] = true
	}

	labLevel := 1

	techs := make([]TechWithStatus, len(Techs))
	for i, cfg := range Techs {
		currentLevel := techLevels[cfg.Type]
		costMetal := cfg.CostMetal * int(math.Pow(cfg.CostFactor, float64(currentLevel+1)))
		costCrystal := cfg.CostCrystal * int(math.Pow(cfg.CostFactor, float64(currentLevel+1)))
		costGas := cfg.CostGas * int(math.Pow(cfg.CostFactor, float64(currentLevel+1)))

		techs[i] = TechWithStatus{
			Type:             cfg.Type,
			Name:             cfg.Name,
			Category:         cfg.Category,
			Level:            currentLevel,
			CostMetal:        costMetal,
			CostCrystal:      costCrystal,
			CostGas:          costGas,
			Researching:      researching[cfg.Type],
			Prerequisites:    cfg.Prerequisites,
			Description:      cfg.Description,
			Effect:           cfg.Effect,
			ResearchLocation: cfg.ResearchLocation,
		}
	}

	return techs, labLevel, nil
}

func (s *ResearchService) StartResearch(ctx context.Context, playerID, planetID int, techType string) (StartResearchResponse, error) {
	cfg, ok := techConfig(techType)
	if !ok {
		return StartResearchResponse{}, ErrInvalidTech
	}

	currentLevel, err := s.fetchTechLevel(ctx, playerID, techType)
	if err != nil {
		return StartResearchResponse{}, fmt.Errorf("get tech level: %w", err)
	}

	if err := s.checkPrerequisites(ctx, playerID, cfg.Prerequisites); err != nil {
		return StartResearchResponse{}, err
	}

	active, err := s.repo.GetActiveResearch(ctx, playerID, techType)
	if err != nil {
		return StartResearchResponse{}, fmt.Errorf("check active: %w", err)
	}
	if active != nil {
		return StartResearchResponse{}, ErrAlreadyResearching
	}

	targetLevel := currentLevel + 1
	costMetal := cfg.CostMetal * int(math.Pow(cfg.CostFactor, float64(targetLevel)))
	costCrystal := cfg.CostCrystal * int(math.Pow(cfg.CostFactor, float64(targetLevel)))
	costGas := cfg.CostGas * int(math.Pow(cfg.CostFactor, float64(targetLevel)))

	if err := s.deductResources(ctx, planetID, costMetal, costCrystal, costGas); err != nil {
		return StartResearchResponse{}, err
	}

	labLevel, err := s.fetchBuildingLevel(ctx, planetID, "research_lab")
	if err != nil {
		labLevel = 0
	}

	duration := researchDuration(costMetal, costCrystal, labLevel)
	completesAt := time.Now().Add(duration)

	_, err = s.repo.CreateResearch(ctx, playerID, techType, targetLevel, completesAt)
	if err != nil {
		return StartResearchResponse{}, fmt.Errorf("create research: %w", err)
	}

	return StartResearchResponse{
		TechType:    techType,
		TargetLevel: targetLevel,
		CompletesAt: completesAt,
	}, nil
}

func (s *ResearchService) CancelResearch(ctx context.Context, playerID int, techType string) (CancelResearchResponse, error) {
	active, err := s.repo.GetActiveResearch(ctx, playerID, techType)
	if err != nil {
		return CancelResearchResponse{}, fmt.Errorf("get active: %w", err)
	}
	if active == nil {
		return CancelResearchResponse{}, ErrNoActiveResearch
	}

	targetLevel := active.TargetLevel
	cfg, ok := techConfig(techType)
	if !ok {
		return CancelResearchResponse{}, ErrInvalidTech
	}

	costMetal := cfg.CostMetal * int(math.Pow(cfg.CostFactor, float64(targetLevel)))
	costCrystal := cfg.CostCrystal * int(math.Pow(cfg.CostFactor, float64(targetLevel)))
	costGas := cfg.CostGas * int(math.Pow(cfg.CostFactor, float64(targetLevel)))

	refundMetal := costMetal / 2
	refundCrystal := costCrystal / 2
	refundGas := costGas / 2

	if err := s.repo.CancelResearchWithRefund(ctx, active.ID, playerID, refundMetal, refundCrystal, refundGas); err != nil {
		return CancelResearchResponse{}, fmt.Errorf("cancel: %w", err)
	}

	return CancelResearchResponse{
		RefundMetal:   refundMetal,
		RefundCrystal: refundCrystal,
		RefundGas:     refundGas,
	}, nil
}

func (s *ResearchService) ProcessCompleted(ctx context.Context) error {
	completed, err := s.repo.GetCompletedResearch(ctx)
	if err != nil {
		return fmt.Errorf("get completed: %w", err)
	}

	for _, q := range completed {
		if err := s.levelUpTech(ctx, q.PlayerID, q.TechType); err != nil {
			return fmt.Errorf("level up tech %s for player %d: %w", q.TechType, q.PlayerID, err)
		}
		if err := s.repo.CompleteResearch(ctx, q.ID); err != nil {
			return fmt.Errorf("complete research %d: %w", q.ID, err)
		}
	}

	return nil
}

func (s *ResearchService) fetchTechLevel(ctx context.Context, playerID int, techType string) (int, error) {
	body := fmt.Sprintf(`{"player_id":%d,"tech_type":"%s"}`, playerID, techType)
	resp, err := s.httpPost(ctx, s.planetAddr+"/internal/player/tech-level", body)
	if err != nil {
		return 0, err
	}
	var result struct {
		Level int `json:"level"`
	}
	if err := parseJSON(resp, &result); err != nil {
		return 0, nil
	}
	return result.Level, nil
}

func (s *ResearchService) fetchTechLevels(ctx context.Context, playerID int) (map[string]int, error) {
	levels := map[string]int{}
	for _, cfg := range Techs {
		level, err := s.fetchTechLevel(ctx, playerID, cfg.Type)
		if err != nil {
			return nil, err
		}
		if level > 0 {
			levels[cfg.Type] = level
		}
	}
	return levels, nil
}

func (s *ResearchService) checkPrerequisites(ctx context.Context, playerID int, prereqs []Prereq) error {
	for _, p := range prereqs {
		if p.Type == "research_lab" {
			continue
		}
		level, err := s.fetchTechLevel(ctx, playerID, p.Type)
		if err != nil {
			return err
		}
		if level < p.Level {
			return ErrPrerequisitesNotMet
		}
	}
	return nil
}

func (s *ResearchService) deductResources(ctx context.Context, planetID, metal, crystal, gas int) error {
	if metal > 0 {
		if err := s.deductSingle(ctx, planetID, "metal", metal); err != nil {
			return err
		}
	}
	if crystal > 0 {
		if err := s.deductSingle(ctx, planetID, "crystal", crystal); err != nil {
			return err
		}
	}
	if gas > 0 {
		if err := s.deductSingle(ctx, planetID, "gas", gas); err != nil {
			return err
		}
	}
	return nil
}

func (s *ResearchService) deductSingle(ctx context.Context, planetID int, resource string, amount int) error {
	body := fmt.Sprintf(`{"planet_id":%d,"resource":"%s","amount":%d}`, planetID, resource, amount)
	resp, err := s.httpPost(ctx, s.planetAddr+"/internal/resources/deduct", body)
	if err != nil {
		return fmt.Errorf("deduct %s: %w", resource, err)
	}
	var result struct {
		Error string `json:"error"`
	}
	if err := parseJSON(resp, &result); err != nil {
		return nil
	}
	if result.Error != "" {
		return fmt.Errorf("deduct %s: %s", resource, result.Error)
	}
	return nil
}

func (s *ResearchService) fetchBuildingLevel(ctx context.Context, planetID int, buildingType string) (int, error) {
	body := fmt.Sprintf(`{"planet_id":%d,"building_type":"%s"}`, planetID, buildingType)
	resp, err := s.httpPost(ctx, s.planetAddr+"/internal/planet/building-level", body)
	if err != nil {
		return 0, err
	}
	var result struct {
		Level int `json:"level"`
	}
	if err := parseJSON(resp, &result); err != nil {
		return 0, nil
	}
	return result.Level, nil
}

func (s *ResearchService) levelUpTech(ctx context.Context, playerID int, techType string) error {
	body := fmt.Sprintf(`{"player_id":%d,"tech_type":"%s"}`, playerID, techType)
	resp, err := s.httpPost(ctx, s.planetAddr+"/internal/player/tech/add", body)
	if err != nil {
		return fmt.Errorf("level up tech: %w", err)
	}
	var result struct {
		Error string `json:"error"`
	}
	if err := parseJSON(resp, &result); err != nil {
		return nil
	}
	if result.Error != "" {
		return fmt.Errorf("level up tech: %s", result.Error)
	}
	return nil
}

func researchDuration(metalCost, crystalCost int, labLevel int) time.Duration {
	if labLevel < 1 {
		labLevel = 1
	}
	hours := float64(metalCost+crystalCost) / (1000.0 * float64(labLevel+1))
	if hours < 1 {
		hours = 1
	}
	return time.Duration(hours * float64(time.Hour))
}
