package main

import (
	"context"
	"sync"
	"time"
)

type mockRepo struct {
	mu       sync.Mutex
	queue    []ResearchQueue
	nextID   int
	techs    map[int]map[string]int
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		nextID: 1,
		techs:  make(map[int]map[string]int),
	}
}

func (m *mockRepo) CreateResearch(_ context.Context, playerID int, techType string, targetLevel int, completesAt time.Time) (ResearchQueue, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	q := ResearchQueue{
		ID:          m.nextID,
		PlayerID:    playerID,
		TechType:    techType,
		TargetLevel: targetLevel,
		StartedAt:   time.Now(),
		CompletesAt: completesAt,
	}
	m.nextID++
	m.queue = append(m.queue, q)
	return q, nil
}

func (m *mockRepo) GetActiveResearch(_ context.Context, playerID int, techType string) (*ResearchQueue, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, q := range m.queue {
		if q.PlayerID == playerID && q.TechType == techType && !q.Completed && !q.Cancelled {
			return &q, nil
		}
	}
	return nil, nil
}

func (m *mockRepo) ListActiveResearch(_ context.Context, playerID int) ([]ResearchQueue, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []ResearchQueue
	for _, q := range m.queue {
		if q.PlayerID == playerID && !q.Completed && !q.Cancelled {
			result = append(result, q)
		}
	}
	if result == nil {
		return []ResearchQueue{}, nil
	}
	return result, nil
}

func (m *mockRepo) GetCompletedResearch(_ context.Context) ([]ResearchQueue, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []ResearchQueue
	now := time.Now()
	for _, q := range m.queue {
		if !q.Completed && !q.Cancelled && now.After(q.CompletesAt) {
			result = append(result, q)
		}
	}
	if result == nil {
		return []ResearchQueue{}, nil
	}
	return result, nil
}

func (m *mockRepo) CompleteResearch(_ context.Context, id int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, q := range m.queue {
		if q.ID == id {
			m.queue[i].Completed = true
			return nil
		}
	}
	return nil
}

func (m *mockRepo) CancelResearchWithRefund(_ context.Context, id, playerID int, _, _, _ int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, q := range m.queue {
		if q.ID == id {
			m.queue[i].Cancelled = true
			return nil
		}
	}
	return nil
}
