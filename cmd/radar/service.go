package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"
)

type RadarService struct {
	repo           Repository
	planetBaseURL  string
	fleetBaseURL   string
	httpClient     *http.Client
}

func NewRadarService(repo Repository, planetBaseURL, fleetBaseURL string) *RadarService {
	return &RadarService{
		repo:          repo,
		planetBaseURL: planetBaseURL,
		fleetBaseURL:  fleetBaseURL,
		httpClient:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *RadarService) Scan(ctx context.Context, playerID int) ([]RadarEvent, error) {
	events, err := s.repo.GetUnresolvedEvents(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("get unresolved events: %w", err)
	}
	return events, nil
}

func (s *RadarService) GetEvents(ctx context.Context, playerID int, scope string) ([]RadarEvent, error) {
	events, err := s.repo.GetPlayerEvents(ctx, playerID)
	if err != nil {
		return nil, err
	}

	switch scope {
	case "incoming":
		return filterEventsByType(events, "incoming_attack"), nil
	case "espionage":
		return filterEventsByType(events, "espionage"), nil
	case "movement":
		return filterEventsByType(events, "fleet_movement"), nil
	default:
		return events, nil
	}
}

func filterEventsByType(events []RadarEvent, eventType string) []RadarEvent {
	var result []RadarEvent
	for _, e := range events {
		if e.EventType == eventType {
			result = append(result, e)
		}
	}
	return result
}

func (s *RadarService) ResolveEvent(ctx context.Context, playerID int, eventID int) error {
	events, err := s.repo.GetUnresolvedEvents(ctx, playerID)
	if err != nil {
		return fmt.Errorf("get unresolved events: %w", err)
	}
	found := false
	for _, e := range events {
		if e.ID == eventID {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("event not found or already resolved")
	}
	return s.repo.ResolveEvent(ctx, eventID)
}

func (s *RadarService) DetectFleet(ctx context.Context, req DetectFleetRequest) error {
	eventType := "fleet_movement"
	switch req.Mission {
	case "attack":
		eventType = "incoming_attack"
	case "espionage":
		eventType = "espionage"
	}

	arrivalTime, err := time.Parse(time.RFC3339, req.ArrivalTime)
	if err != nil {
		return fmt.Errorf("invalid arrival time: %w", err)
	}

	srcPlayerID := req.SourcePlayerID
	fleetID := req.FleetID
	originGalaxy := req.OriginGalaxy
	originSystem := req.OriginSystem
	originPosition := req.OriginPosition

	_, err = s.repo.CreateRadarEvent(ctx, RadarEvent{
		PlayerID:       req.TargetPlayerID,
		EventType:      eventType,
		SourcePlayerID: &srcPlayerID,
		FleetID:        &fleetID,
		TargetGalaxy:   req.TargetGalaxy,
		TargetSystem:   req.TargetSystem,
		TargetPosition: req.TargetPosition,
		OriginGalaxy:   &originGalaxy,
		OriginSystem:   &originSystem,
		OriginPosition: &originPosition,
		ArrivalTime:    &arrivalTime,
	})
	return err
}

func (s *RadarService) PlanetStatus(ctx context.Context, playerID int) ([]PlanetStatusResponse, error) {
	events, err := s.repo.GetUnresolvedEvents(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("get unresolved events: %w", err)
	}

	planetEvents := make(map[string]struct {
		hasAttack   bool
		hasEspionage bool
		fleetCount  int
		coords      [3]int
	})

	for _, e := range events {
		key := fmt.Sprintf("%d_%d_%d", e.TargetGalaxy, e.TargetSystem, e.TargetPosition)
		entry := planetEvents[key]
		entry.coords = [3]int{e.TargetGalaxy, e.TargetSystem, e.TargetPosition}
		entry.fleetCount++
		if e.EventType == "incoming_attack" {
			entry.hasAttack = true
		}
		if e.EventType == "espionage" {
			entry.hasEspionage = true
		}
		planetEvents[key] = entry
	}

	var result []PlanetStatusResponse
	for _, entry := range planetEvents {
		status := "secure"
		if entry.hasAttack {
			status = "attack_incoming"
		} else if entry.hasEspionage {
			status = "espionaged"
		}
		result = append(result, PlanetStatusResponse{
			PlanetID:   0,
			Status:     status,
			FleetCount: entry.fleetCount,
		})
	}
	return result, nil
}

func (s *RadarService) EUXScan(ctx context.Context, playerID int, targetGalaxy, targetSystem, targetPosition int) (*EUXScanResponse, error) {
	eux, err := s.repo.GetEuxRadar(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("get eu-x radar: %w", err)
	}
	if eux == nil {
		return nil, fmt.Errorf("no eu-x radar installed")
	}

	if targetGalaxy != eux.Galaxy {
		return nil, fmt.Errorf("eu-x radar cannot scan across galaxies")
	}

	rangeLimit := eux.Level * 5
	dist := int(math.Abs(float64(targetSystem - eux.System)))
	if dist > rangeLimit {
		return nil, fmt.Errorf("target out of range: %d systems away, max %d", dist, rangeLimit)
	}

	fleets, err := s.getFleetsAtTarget(ctx, targetGalaxy, targetSystem, targetPosition)
	if err != nil {
		return nil, fmt.Errorf("scan target fleets: %w", err)
	}

	return &EUXScanResponse{Fleets: fleets}, nil
}

type fleetServiceResponse []struct {
	ID              int            `json:"id"`
	PlayerID        int            `json:"player_id"`
	Mission         string         `json:"mission"`
	Ships           map[string]int `json:"ships"`
	TargetGalaxy    int            `json:"target_galaxy"`
	TargetSystem    int            `json:"target_system"`
	TargetPosition  int            `json:"target_position"`
	ArrivesAt       *time.Time     `json:"arrives_at,omitempty"`
	Status          string         `json:"status"`
}

func (s *RadarService) getFleetsAtTarget(ctx context.Context, galaxy, system, position int) ([]FleetInfo, error) {
	body, _ := json.Marshal(map[string]int{
		"galaxy": galaxy, "system": system, "position": position,
	})
	resp, err := s.httpClient.Post(s.fleetBaseURL+"/internal/fleet/at-location", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("fleet service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("fleet service: %s", string(respBody))
	}

	var fleets []struct {
		ID        int            `json:"id"`
		Mission   string         `json:"mission"`
		Ships     map[string]int `json:"ships"`
		ArrivesAt *time.Time     `json:"arrives_at,omitempty"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&fleets); err != nil {
		return nil, fmt.Errorf("parse fleet response: %w", err)
	}

	result := make([]FleetInfo, len(fleets))
	for i, f := range fleets {
		result[i] = FleetInfo{
			ID:        f.ID,
			Ships:     f.Ships,
			Mission:   f.Mission,
			ArrivesAt: f.ArrivesAt,
		}
	}
	return result, nil
}
