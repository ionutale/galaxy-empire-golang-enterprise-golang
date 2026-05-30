package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type FleetService struct {
	repo          Repository
	planetBaseURL string
	httpClient    *http.Client
}

func NewFleetService(repo Repository, planetBaseURL string) *FleetService {
	return &FleetService{
		repo:          repo,
		planetBaseURL: planetBaseURL,
		httpClient:    &http.Client{Timeout: 10 * time.Second},
	}
}

var validMissions = map[string]bool{
	"attack": true, "acs_attack": true, "acs_defend": true,
	"transport": true, "deploy": true, "espionage": true,
	"colonize": true, "expedition": true, "recycle": true,
}

func (s *FleetService) DispatchFleet(ctx context.Context, playerID int, req DispatchRequest) (Fleet, error) {
	if !validMissions[req.Mission] {
		return Fleet{}, fmt.Errorf("invalid mission: %s", req.Mission)
	}
	if len(req.Ships) == 0 {
		return Fleet{}, fmt.Errorf("no ships selected")
	}
	if req.SpeedPct < 10 || req.SpeedPct > 100 {
		return Fleet{}, fmt.Errorf("speed must be 10-100")
	}

	body, _ := json.Marshal(map[string]any{
		"planet_id": req.OriginPlanetID,
		"ships":     req.Ships,
	})
	resp, err := s.httpClient.Post(s.planetBaseURL+"/internal/ships/deduct", "application/json", bytes.NewReader(body))
	if err != nil {
		return Fleet{}, fmt.Errorf("planet service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return Fleet{}, fmt.Errorf("planet service: %s", string(respBody))
	}

	fleet := Fleet{
		PlayerID:       playerID,
		OriginPlanetID: req.OriginPlanetID,
		TargetGalaxy:   req.TargetGalaxy,
		TargetSystem:   req.TargetSystem,
		TargetPosition: req.TargetPosition,
		Mission:        req.Mission,
		Status:         "stationed",
		SpeedPct:       req.SpeedPct,
		Ships:          req.Ships,
	}

	return s.repo.CreateFleet(ctx, fleet)
}
