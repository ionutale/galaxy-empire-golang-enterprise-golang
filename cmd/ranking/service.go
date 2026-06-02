package main

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

const buildingScoreWeight = 1
const researchScoreWeight = 1

type RankingService struct {
	repo Repository
}

func NewRankingService(repo Repository) *RankingService {
	return &RankingService{repo: repo}
}

func (s *RankingService) GetTop(ctx context.Context, limit, offset int) ([]PlayerScore, int, error) {
	if limit < 1 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.GetTop(ctx, limit, offset)
}

func (s *RankingService) GetByPlayerID(ctx context.Context, playerID int) (*PlayerScore, int, error) {
	score, err := s.repo.GetByPlayerID(ctx, playerID)
	if err != nil {
		return nil, 0, fmt.Errorf("get by player id: %w", err)
	}
	if score == nil {
		return nil, 0, nil
	}
	rank, err := s.repo.GetPlayerRank(ctx, playerID)
	if err != nil {
		return nil, 0, fmt.Errorf("get player rank: %w", err)
	}
	return score, rank, nil
}

func (s *RankingService) UpdateScore(ctx context.Context, req UpdateScoreRequest) error {
	existing, err := s.repo.GetByPlayerID(ctx, req.PlayerID)
	if err != nil {
		return fmt.Errorf("get existing: %w", err)
	}

	score := PlayerScore{
		PlayerID:   req.PlayerID,
		PlayerName: req.PlayerName,
		UpdatedAt:  time.Now(),
	}

	if existing != nil {
		score.ID = existing.ID
		score.BuildingsScore = existing.BuildingsScore
		score.ResearchScore = existing.ResearchScore
		score.FleetScore = existing.FleetScore
		score.DefenseScore = existing.DefenseScore
	}

	if req.BuildingsScore != nil {
		score.BuildingsScore = *req.BuildingsScore
	}
	if req.ResearchScore != nil {
		score.ResearchScore = *req.ResearchScore
	}
	if req.FleetScore != nil {
		score.FleetScore = *req.FleetScore
	}
	if req.DefenseScore != nil {
		score.DefenseScore = *req.DefenseScore
	}

	score.TotalScore = score.BuildingsScore + score.ResearchScore + score.FleetScore + score.DefenseScore

	return s.repo.UpsertScore(ctx, score)
}

func (s *RankingService) RecalculateForPlayer(ctx context.Context, playerID int) error {
	slog.Info("recalculating scores", "player_id", playerID)

	buildingsScore, err := s.repo.SumBuildingLevels(ctx, playerID)
	if err != nil {
		return fmt.Errorf("calculate buildings score: %w", err)
	}

	researchScore, err := s.repo.SumResearchLevels(ctx, playerID)
	if err != nil {
		return fmt.Errorf("calculate research score: %w", err)
	}

	fleetScore, err := s.repo.SumFleetQuantity(ctx, playerID)
	if err != nil {
		return fmt.Errorf("calculate fleet score: %w", err)
	}

	defenseScore, err := s.repo.SumDefenseQuantity(ctx, playerID)
	if err != nil {
		return fmt.Errorf("calculate defense score: %w", err)
	}

	buildingsScore *= buildingScoreWeight
	researchScore *= researchScoreWeight

	totalScore := buildingsScore + researchScore + fleetScore + defenseScore

	playerName := fmt.Sprintf("Player %d", playerID)

	score := PlayerScore{
		PlayerID:       playerID,
		PlayerName:     playerName,
		TotalScore:     totalScore,
		BuildingsScore: buildingsScore,
		ResearchScore:  researchScore,
		FleetScore:     fleetScore,
		DefenseScore:   defenseScore,
		UpdatedAt:      time.Now(),
	}

	return s.repo.UpsertScore(ctx, score)
}

func (s *RankingService) RecalculateAll(ctx context.Context) error {
	playerIDs, err := s.repo.ListAllPlayerIDs(ctx)
	if err != nil {
		return fmt.Errorf("list all player ids: %w", err)
	}

	sem := make(chan struct{}, 10) // 10 concurrent workers
	var wg sync.WaitGroup
	for _, pid := range playerIDs {
		pid := pid
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			if err := s.RecalculateForPlayer(ctx, pid); err != nil {
				slog.Error("recalculate for player failed", "player_id", pid, "error", err)
			}
		}()
	}
	wg.Wait()

	slog.Info("recalculated scores for all players", "count", len(playerIDs))
	return nil
}
