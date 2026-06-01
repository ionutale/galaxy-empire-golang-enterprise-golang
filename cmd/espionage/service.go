package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

type EspionageService struct {
	repo          Repository
	planetBaseURL string
	httpClient    *http.Client
}

func NewEspionageService(repo Repository, planetBaseURL string) *EspionageService {
	return &EspionageService{
		repo:          repo,
		planetBaseURL: planetBaseURL,
		httpClient:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *EspionageService) SendProbe(ctx context.Context, playerID int, req ProbeRequest) (EspionageReport, error) {
	if err := s.deductProbe(ctx, req.PlanetID); err != nil {
		return EspionageReport{}, fmt.Errorf("deduct probe: %w", err)
	}

	targetInfo, err := s.getPlanetInfo(ctx, req.TargetGalaxy, req.TargetSystem, req.TargetPosition)
	if err != nil {
		return EspionageReport{}, fmt.Errorf("get target info: %w", err)
	}

	techLevel, err := s.getEspionageTechLevel(ctx, playerID)
	if err != nil {
		slog.Error("get espionage tech level", "player", playerID, "error", err)
		techLevel = 0
	}
	detailLevel := techLevel
	if detailLevel > 5 {
		detailLevel = 5
	}

	report := s.buildReport(playerID, targetInfo, req, detailLevel)

	return s.repo.CreateReport(ctx, report)
}

func (s *EspionageService) getEspionageTechLevel(ctx context.Context, playerID int) (int, error) {
	body, _ := json.Marshal(map[string]any{"player_id": playerID, "tech_type": "espionage_tech"})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.planetBaseURL+"/internal/player/tech-level", bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, nil
	}
	var result struct {
		Level int `json:"level"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, nil
	}
	return result.Level, nil
}

func (s *EspionageService) buildReport(playerID int, target PlanetInfo, req ProbeRequest, detailLevel int) EspionageReport {
	report := EspionageReport{
		PlayerID:       playerID,
		TargetPlayerID: target.PlayerID,
		TargetGalaxy:   req.TargetGalaxy,
		TargetSystem:   req.TargetSystem,
		TargetPosition: req.TargetPosition,
		DetailLevel:    detailLevel,
		Resources:      map[string]int{"metal": target.Metal, "crystal": target.Crystal, "gas": target.Gas},
		Fleet:          target.Ships,
		ExpiresAt:      time.Now().Add(24 * time.Hour),
	}

	report.ReportData = map[string]any{
		"resources": report.Resources,
		"fleet":     report.Fleet,
	}

	return report
}

func (s *EspionageService) GetReport(ctx context.Context, playerID, reportID int) (EspionageReport, error) {
	report, err := s.repo.GetReportByID(ctx, reportID)
	if err != nil {
		return EspionageReport{}, fmt.Errorf("report not found")
	}
	if report.PlayerID != playerID && report.TargetPlayerID != playerID {
		return EspionageReport{}, fmt.Errorf("not your report")
	}
	return report, nil
}

func (s *EspionageService) ListReports(ctx context.Context, playerID int) ([]EspionageReport, error) {
	return s.repo.ListReportsForPlayer(ctx, playerID)
}

func (s *EspionageService) DeleteReport(ctx context.Context, playerID, reportID int) error {
	report, err := s.repo.GetReportByID(ctx, reportID)
	if err != nil {
		return fmt.Errorf("report not found")
	}
	if report.PlayerID != playerID && report.TargetPlayerID != playerID {
		return fmt.Errorf("not your report")
	}
	return s.repo.DeleteReport(ctx, reportID)
}

func (s *EspionageService) deductProbe(ctx context.Context, planetID int) error {
	body, _ := json.Marshal(map[string]any{
		"planet_id": planetID,
		"ships":     map[string]int{"espionage_probe": 1},
	})
	resp, err := s.httpClient.Post(s.planetBaseURL+"/internal/ships/deduct", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("planet service unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("planet service: %s", string(respBody))
	}
	return nil
}

func (s *EspionageService) getPlanetInfo(ctx context.Context, galaxy, system, position int) (PlanetInfo, error) {
	body, _ := json.Marshal(map[string]int{
		"galaxy": galaxy, "system": system, "position": position,
	})
	resp, err := s.httpClient.Post(s.planetBaseURL+"/internal/planet/info", "application/json", bytes.NewReader(body))
	if err != nil {
		slog.Warn("planet info unreachable (target may be empty)", "error", err)
		return PlanetInfo{PlayerID: 0, PlanetID: 0, Metal: 0, Crystal: 0, Gas: 0, Ships: map[string]int{}}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return PlanetInfo{PlayerID: 0, PlanetID: 0, Metal: 0, Crystal: 0, Gas: 0, Ships: map[string]int{}}, nil
	}
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return PlanetInfo{}, fmt.Errorf("planet service: %s", string(respBody))
	}

	var info PlanetInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return PlanetInfo{}, fmt.Errorf("parse planet info: %w", err)
	}
	if info.Ships == nil {
		info.Ships = map[string]int{}
	}
	return info, nil
}
