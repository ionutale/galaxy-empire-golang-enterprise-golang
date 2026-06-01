package main

import (
	"context"
	"fmt"
	"log/slog"
)

type TutorialService struct {
	repo Repository
}

func NewTutorialService(repo Repository) *TutorialService {
	return &TutorialService{repo: repo}
}

func (s *TutorialService) GetStatus(ctx context.Context, playerID int) (*TutorialStatusResponse, error) {
	pt, err := s.repo.GetPlayerTutorial(ctx, playerID)
	if err != nil {
		return nil, err
	}

	if pt == nil {
		if err := s.repo.CreatePlayerTutorial(ctx, playerID); err != nil {
			return nil, err
		}
		pt = &PlayerTutorial{PlayerID: playerID, CurrentStep: 1, Completed: false}
	}

	resp := &TutorialStatusResponse{
		CurrentStep: pt.CurrentStep,
		Completed:   pt.Completed,
		Steps:       TutorialSteps,
	}
	return resp, nil
}

func (s *TutorialService) ClaimReward(ctx context.Context, playerID, stepID int) (*ClaimRewardResponse, error) {
	pt, err := s.repo.GetPlayerTutorial(ctx, playerID)
	if err != nil {
		return nil, err
	}
	if pt == nil {
		return nil, fmt.Errorf("tutorial not started")
	}
	if pt.Completed {
		return nil, fmt.Errorf("tutorial already completed")
	}

	var step *TutorialStep
	for i := range TutorialSteps {
		if TutorialSteps[i].ID == stepID {
			step = &TutorialSteps[i]
			break
		}
	}
	if step == nil {
		return nil, fmt.Errorf("step %d not found", stepID)
	}

	if pt.CurrentStep != stepID {
		return nil, fmt.Errorf("current step is %d, not %d", pt.CurrentStep, stepID)
	}

	ok, err := s.isStepComplete(ctx, playerID, step)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("step %d requirements not met", stepID)
	}

	// Advance step FIRST — this is the idempotency gate.
	// Rewards are only granted after the step is successfully advanced.
	completed := false
	nextStep := stepID + 1
	if nextStep > len(TutorialSteps) {
		if err := s.repo.CompleteTutorial(ctx, playerID); err != nil {
			return nil, err
		}
		completed = true
	} else {
		if err := s.repo.AdvanceStep(ctx, playerID); err != nil {
			return nil, err
		}
	}

	// Grant rewards only after successful step advancement.
	if step.RewardDM > 0 {
		if err := s.repo.AddDarkMatter(ctx, playerID, step.RewardDM); err != nil {
			slog.Error("add dm reward — step advanced but reward lost", "step", stepID, "error", err)
		}
	}

	if len(step.RewardResources) > 0 {
		metal := step.RewardResources["metal"]
		crystal := step.RewardResources["crystal"]
		if metal > 0 || crystal > 0 {
			if err := s.repo.AddPlayerResources(ctx, playerID, metal, crystal); err != nil {
				slog.Error("add resource reward — step advanced but reward lost", "step", stepID, "error", err)
			}
		}
	}

	return &ClaimRewardResponse{
		Step:            stepID,
		RewardDM:        step.RewardDM,
		RewardResources: step.RewardResources,
		NextStep:        nextStep,
		Completed:       completed,
	}, nil
}

func (s *TutorialService) SkipStep(ctx context.Context, playerID int) error {
	pt, err := s.repo.GetPlayerTutorial(ctx, playerID)
	if err != nil {
		return err
	}
	if pt == nil {
		return fmt.Errorf("tutorial not started")
	}
	if pt.Completed {
		return fmt.Errorf("tutorial already completed")
	}

	nextStep := pt.CurrentStep + 1
	if nextStep > len(TutorialSteps) {
		if err := s.repo.CompleteTutorial(ctx, playerID); err != nil {
			return err
		}
		return nil
	}

	return s.repo.AdvanceStep(ctx, playerID)
}

func (s *TutorialService) ProgressUpdate(ctx context.Context, playerID, stepID int) error {
	pt, err := s.repo.GetPlayerTutorial(ctx, playerID)
	if err != nil {
		return err
	}
	if pt == nil {
		return nil
	}
	if pt.Completed {
		return nil
	}

	if stepID > 0 && stepID != pt.CurrentStep {
		return nil
	}

	var step *TutorialStep
	for i := range TutorialSteps {
		if TutorialSteps[i].ID == pt.CurrentStep {
			step = &TutorialSteps[i]
			break
		}
	}
	if step == nil {
		return nil
	}

	ok, err := s.isStepComplete(ctx, playerID, step)
	if err != nil {
		return err
	}
	if ok {
		slog.Info("tutorial step completed", "player", playerID, "step", step.ID)
	}

	return nil
}

func (s *TutorialService) isStepComplete(ctx context.Context, playerID int, step *TutorialStep) (bool, error) {
	switch step.Action {
	case "upgrade_building":
		return s.repo.CheckBuildingLevel(ctx, playerID, step.Target)
	case "build_ship":
		return s.repo.CheckShipCount(ctx, playerID, step.Target)
	case "send_expedition":
		return s.repo.CheckExpeditionExists(ctx, playerID)
	case "launch_attack":
		return s.repo.CheckAttackExists(ctx, playerID)
	default:
		return false, nil
	}
}
