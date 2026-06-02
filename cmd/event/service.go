package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

type EventService struct {
	repo Repository
}

func NewEventService(repo Repository) *EventService {
	return &EventService{repo: repo}
}

func (s *EventService) GetActiveEvents(ctx context.Context) ([]Event, error) {
	return s.repo.GetActiveEvents(ctx)
}

func (s *EventService) GetAllEvents(ctx context.Context) ([]Event, error) {
	return s.repo.GetAllEvents(ctx)
}

func (s *EventService) JoinEvent(ctx context.Context, playerID, eventID int) error {
	event, err := s.repo.GetEventByID(ctx, eventID)
	if err != nil {
		return err
	}
	if event.Status != "active" {
		return errEventNotActive
	}
	// Hard real-time check: the background ticker only fires every 60 s, so the
	// cached status field may still read "active" for up to a minute after the
	// event has truly ended.  Reject late joins immediately (#160).
	if time.Now().After(event.EndsAt) {
		return fmt.Errorf("event has already ended")
	}
	_, err = s.repo.GetParticipation(ctx, playerID, eventID)
	if err == nil {
		return errAlreadyJoined
	}
	_, err = s.repo.JoinEvent(ctx, playerID, eventID)
	return err
}

func (s *EventService) ClaimRewards(ctx context.Context, playerID, eventID int) error {
	event, err := s.repo.GetEventByID(ctx, eventID)
	if err != nil {
		return err
	}
	if event.Status != "active" && event.Status != "ended" {
		return errEventNotActive
	}
	return s.repo.ClaimRewards(ctx, playerID, eventID)
}

func (s *EventService) CreateEvent(ctx context.Context, e Event) (Event, error) {
	if e.Status == "" {
		e.Status = "upcoming"
	}
	if e.Modifiers == nil {
		if mods, ok := eventTypeModifiers[e.EventType]; ok {
			e.Modifiers = mods
		} else {
			e.Modifiers = map[string]any{}
		}
	}
	return s.repo.CreateEvent(ctx, e)
}

func (s *EventService) StartBackgroundTicker(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	s.checkAndUpdateEvents(ctx)

	for {
		select {
		case <-ticker.C:
			s.checkAndUpdateEvents(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (s *EventService) checkAndUpdateEvents(ctx context.Context) {
	now := time.Now()

	upcoming, err := s.repo.GetEventsByStatus(ctx, "upcoming")
	if err != nil {
		slog.Error("check upcoming events", "error", err)
	} else {
		for _, e := range upcoming {
			if now.After(e.StartsAt) || now.Equal(e.StartsAt) {
				if err := s.repo.UpdateEventStatus(ctx, e.ID, "active"); err != nil {
					slog.Error("activate event", "id", e.ID, "error", err)
				} else {
					slog.Info("event activated", "id", e.ID, "name", e.Name)
				}
			}
		}
	}

	active, err := s.repo.GetEventsByStatus(ctx, "active")
	if err != nil {
		slog.Error("check active events", "error", err)
	} else {
		for _, e := range active {
			if now.After(e.EndsAt) {
				if err := s.repo.UpdateEventStatus(ctx, e.ID, "ended"); err != nil {
					slog.Error("end event", "id", e.ID, "error", err)
				} else {
					slog.Info("event ended", "id", e.ID, "name", e.Name)
				}
			}
		}
	}
}
