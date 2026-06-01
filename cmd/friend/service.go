package main

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type FriendService struct {
	repo       Repository
	httpClient *http.Client
}

func NewFriendService(repo Repository) *FriendService {
	return &FriendService{
		repo:       repo,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *FriendService) SendRequest(ctx context.Context, playerID, friendID int) (Friendship, error) {
	if playerID == friendID {
		return Friendship{}, fmt.Errorf("cannot add yourself as a friend")
	}

	existing, err := s.repo.GetFriendship(ctx, playerID, friendID)
	if err != nil {
		return Friendship{}, fmt.Errorf("check friendship: %w", err)
	}
	if existing != nil {
		return Friendship{}, fmt.Errorf("friendship already exists")
	}

	f1, err := s.repo.AddFriend(ctx, playerID, friendID)
	if err != nil {
		return Friendship{}, fmt.Errorf("add friend: %w", err)
	}

	_, err = s.repo.AddFriend(ctx, friendID, playerID)
	if err != nil {
		return Friendship{}, fmt.Errorf("add reciprocal: %w", err)
	}

	return f1, nil
}

func (s *FriendService) AcceptRequest(ctx context.Context, playerID, friendID int) error {
	existing, err := s.repo.GetFriendship(ctx, playerID, friendID)
	if err != nil {
		return fmt.Errorf("check friendship: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("friendship not found")
	}
	if existing.Status != "pending" {
		return fmt.Errorf("friendship is not pending")
	}

	if err := s.repo.AcceptFriend(ctx, playerID, friendID); err != nil {
		return fmt.Errorf("accept friend: %w", err)
	}
	return nil
}

func (s *FriendService) RemoveFriend(ctx context.Context, playerID, friendID int) error {
	existing, err := s.repo.GetFriendship(ctx, playerID, friendID)
	if err != nil {
		return fmt.Errorf("check friendship: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("friendship not found")
	}

	if err := s.repo.RemoveFriend(ctx, playerID, friendID); err != nil {
		return fmt.Errorf("remove friend: %w", err)
	}
	return nil
}

func (s *FriendService) ListFriends(ctx context.Context, playerID int) ([]FriendResponse, error) {
	friendships, err := s.repo.GetFriends(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("get friends: %w", err)
	}

	var result []FriendResponse
	for _, f := range friendships {
		if f.Status != "accepted" {
			continue
		}

		lastActive, err := s.repo.GetLastActive(ctx, f.FriendID)
		if err != nil {
			lastActive = nil
		}

		online := false
		var lastActiveStr string
		if lastActive != nil {
			lastActiveStr = lastActive.Format("2006-01-02T15:04:05Z")
			online = time.Since(*lastActive) < 5*time.Minute
		}

		result = append(result, FriendResponse{
			PlayerID:   f.FriendID,
			Status:     f.Status,
			Online:     online,
			LastActive: lastActiveStr,
			CreatedAt:  f.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	if result == nil {
		result = []FriendResponse{}
	}

	return result, nil
}

func (s *FriendService) Ping(ctx context.Context, playerID int) error {
	if err := s.repo.UpdateLastActive(ctx, playerID); err != nil {
		return fmt.Errorf("update last active: %w", err)
	}
	return nil
}
