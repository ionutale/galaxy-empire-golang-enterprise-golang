package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	AddFriend(ctx context.Context, playerID, friendID int) (Friendship, error)
	AddFriendTx(ctx context.Context, playerID, friendID int) (Friendship, error)
	AcceptFriend(ctx context.Context, playerID, friendID int) error
	RemoveFriend(ctx context.Context, playerID, friendID int) error
	GetFriends(ctx context.Context, playerID int) ([]Friendship, error)
	GetFriendship(ctx context.Context, playerID, friendID int) (*Friendship, error)
	UpdateLastActive(ctx context.Context, playerID int) error
	GetLastActive(ctx context.Context, playerID int) (*time.Time, error)
}

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) AddFriend(ctx context.Context, playerID, friendID int) (Friendship, error) {
	var f Friendship
	err := r.pool.QueryRow(ctx, `
		INSERT INTO friend.friendships (player_id, friend_id, status)
		VALUES ($1, $2, 'pending')
		RETURNING id, player_id, friend_id, status, created_at, updated_at
	`, playerID, friendID).Scan(&f.ID, &f.PlayerID, &f.FriendID, &f.Status, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		return Friendship{}, fmt.Errorf("add friend: %w", err)
	}
	return f, nil
}

func (r *PostgresRepository) AddFriendTx(ctx context.Context, playerID, friendID int) (Friendship, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Friendship{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	const insertSQL = `
		INSERT INTO friend.friendships (player_id, friend_id, status)
		VALUES ($1, $2, 'pending')
		RETURNING id, player_id, friend_id, status, created_at, updated_at
	`

	var f1 Friendship
	err = tx.QueryRow(ctx, insertSQL, playerID, friendID).Scan(
		&f1.ID, &f1.PlayerID, &f1.FriendID, &f1.Status, &f1.CreatedAt, &f1.UpdatedAt,
	)
	if err != nil {
		return Friendship{}, fmt.Errorf("add friend: %w", err)
	}

	var f2 Friendship
	err = tx.QueryRow(ctx, insertSQL, friendID, playerID).Scan(
		&f2.ID, &f2.PlayerID, &f2.FriendID, &f2.Status, &f2.CreatedAt, &f2.UpdatedAt,
	)
	if err != nil {
		return Friendship{}, fmt.Errorf("add reciprocal: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Friendship{}, fmt.Errorf("commit transaction: %w", err)
	}

	return f1, nil
}

func (r *PostgresRepository) AcceptFriend(ctx context.Context, playerID, friendID int) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE friend.friendships
		SET status = 'accepted', updated_at = NOW()
		WHERE (player_id = $1 AND friend_id = $2) OR (player_id = $2 AND friend_id = $1)
	`, playerID, friendID)
	return err
}

func (r *PostgresRepository) RemoveFriend(ctx context.Context, playerID, friendID int) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM friend.friendships
		WHERE (player_id = $1 AND friend_id = $2) OR (player_id = $2 AND friend_id = $1)
	`, playerID, friendID)
	return err
}

func (r *PostgresRepository) GetFriends(ctx context.Context, playerID int) ([]Friendship, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, player_id, friend_id, status, created_at, updated_at
		FROM friend.friendships
		WHERE player_id = $1
		ORDER BY created_at DESC LIMIT 200
`, playerID)
	if err != nil {
		return nil, fmt.Errorf("get friends: %w", err)
	}
	defer rows.Close()

	var friendships []Friendship
	for rows.Next() {
		var f Friendship
		if err := rows.Scan(&f.ID, &f.PlayerID, &f.FriendID, &f.Status, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan friendship: %w", err)
		}
		friendships = append(friendships, f)
	}
	return friendships, rows.Err()
}

func (r *PostgresRepository) GetFriendship(ctx context.Context, playerID, friendID int) (*Friendship, error) {
	var f Friendship
	err := r.pool.QueryRow(ctx, `
		SELECT id, player_id, friend_id, status, created_at, updated_at
		FROM friend.friendships
		WHERE (player_id = $1 AND friend_id = $2) OR (player_id = $2 AND friend_id = $1)
	`, playerID, friendID).Scan(&f.ID, &f.PlayerID, &f.FriendID, &f.Status, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get friendship: %w", err)
	}
	return &f, nil
}

func (r *PostgresRepository) UpdateLastActive(ctx context.Context, playerID int) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE friend.friendships
		SET last_active = NOW()
		WHERE player_id = $1
	`, playerID)
	return err
}

func (r *PostgresRepository) GetLastActive(ctx context.Context, playerID int) (*time.Time, error) {
	var t *time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT MAX(last_active) FROM friend.friendships WHERE player_id = $1
	`, playerID).Scan(&t)
	if err != nil {
		return nil, fmt.Errorf("get last active: %w", err)
	}
	return t, nil
}

type mockRepo struct {
	mu          sync.Mutex
	friendships []Friendship
	lastActives map[int]time.Time
	nextID      int
}

func newMockRepo() *mockRepo {
	return &mockRepo{nextID: 1, lastActives: make(map[int]time.Time)}
}

func (m *mockRepo) AddFriend(ctx context.Context, playerID, friendID int) (Friendship, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	f := Friendship{
		ID:        m.nextID,
		PlayerID:  playerID,
		FriendID:  friendID,
		Status:    "pending",
		CreatedAt: now,
		UpdatedAt: now,
	}
	m.nextID++
	m.friendships = append(m.friendships, f)
	return f, nil
}

func (m *mockRepo) AddFriendTx(ctx context.Context, playerID, friendID int) (Friendship, error) {
	f1, err := m.AddFriend(ctx, playerID, friendID)
	if err != nil {
		return Friendship{}, err
	}
	if _, err := m.AddFriend(ctx, friendID, playerID); err != nil {
		// roll back the first insert
		m.mu.Lock()
		for i, f := range m.friendships {
			if f.ID == f1.ID {
				m.friendships = append(m.friendships[:i], m.friendships[i+1:]...)
				break
			}
		}
		m.mu.Unlock()
		return Friendship{}, err
	}
	return f1, nil
}

func (m *mockRepo) AcceptFriend(ctx context.Context, playerID, friendID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, f := range m.friendships {
		if (f.PlayerID == playerID && f.FriendID == friendID) || (f.PlayerID == friendID && f.FriendID == playerID) {
			m.friendships[i].Status = "accepted"
			m.friendships[i].UpdatedAt = time.Now()
			return nil
		}
	}
	return fmt.Errorf("friendship not found")
}

func (m *mockRepo) RemoveFriend(ctx context.Context, playerID, friendID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, f := range m.friendships {
		if (f.PlayerID == playerID && f.FriendID == friendID) || (f.PlayerID == friendID && f.FriendID == playerID) {
			m.friendships = append(m.friendships[:i], m.friendships[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("friendship not found")
}

func (m *mockRepo) GetFriends(ctx context.Context, playerID int) ([]Friendship, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []Friendship
	for _, f := range m.friendships {
		if f.PlayerID == playerID {
			result = append(result, f)
		}
	}
	return result, nil
}

func (m *mockRepo) GetFriendship(ctx context.Context, playerID, friendID int) (*Friendship, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, f := range m.friendships {
		if (f.PlayerID == playerID && f.FriendID == friendID) || (f.PlayerID == friendID && f.FriendID == playerID) {
			f := f
			return &f, nil
		}
	}
	return nil, nil
}

func (m *mockRepo) UpdateLastActive(ctx context.Context, playerID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.lastActives[playerID] = time.Now()
	return nil
}

func (m *mockRepo) GetLastActive(ctx context.Context, playerID int) (*time.Time, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if t, ok := m.lastActives[playerID]; ok {
		return &t, nil
	}
	return nil, nil
}
