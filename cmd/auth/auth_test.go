package main

import (
	"context"
	"errors"
	"testing"
	"time"
)

type mockRepo struct {
	users  []User
	nextID int
}

func (m *mockRepo) Create(_ context.Context, email, passwordHash string) (User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return User{}, ErrDuplicateEmail
		}
	}
	m.nextID++
	u := User{
		ID: m.nextID, Email: email,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	m.users = append(m.users, u)
	return u, nil
}

func (m *mockRepo) FindByEmail(_ context.Context, email string) (User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return User{}, ErrUserNotFound
}

func (m *mockRepo) FindByID(_ context.Context, id int) (User, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return User{}, ErrUserNotFound
}

func TestRegister_Success(t *testing.T) {
	svc := NewAuthService(&mockRepo{}, []byte("test-key"))
	resp, err := svc.Register(context.Background(), RegisterRequest{
		Email: "test@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatal("expected no error, got:", err)
	}
	if resp.Token == "" {
		t.Error("expected non-empty token")
	}
	if resp.User.Email != "test@example.com" {
		t.Errorf("expected test@example.com, got %s", resp.User.Email)
	}
	if resp.User.ID == 0 {
		t.Error("expected non-zero user ID")
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	svc := NewAuthService(&mockRepo{}, []byte("test-key"))
	_, err := svc.Register(context.Background(), RegisterRequest{
		Email: "dup@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatal("expected no error on first register, got:", err)
	}
	_, err = svc.Register(context.Background(), RegisterRequest{
		Email: "dup@example.com", Password: "password456",
	})
	if err == nil {
		t.Fatal("expected error for duplicate email")
	}
	if err != ErrDuplicateEmail {
		t.Errorf("expected ErrDuplicateEmail, got %v", err)
	}
}

func TestRegister_InvalidEmail(t *testing.T) {
	svc := NewAuthService(&mockRepo{}, []byte("test-key"))
	tests := []string{"", "notanemail", "@example.com", "user@"}
	for _, email := range tests {
		_, err := svc.Register(context.Background(), RegisterRequest{
			Email: email, Password: "password123",
		})
		if err == nil || !isValidationError(err) {
			t.Errorf("expected validation error for email %q, got %v", email, err)
		}
	}
}

func TestRegister_ShortPassword(t *testing.T) {
	svc := NewAuthService(&mockRepo{}, []byte("test-key"))
	_, err := svc.Register(context.Background(), RegisterRequest{
		Email: "test@example.com", Password: "12345",
	})
	if err == nil || !isValidationError(err) {
		t.Error("expected validation error for short password")
	}
}

func TestLogin_Success(t *testing.T) {
	svc := NewAuthService(&mockRepo{}, []byte("test-key"))
	_, err := svc.Register(context.Background(), RegisterRequest{
		Email: "login@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatal("registration failed:", err)
	}

	resp, err := svc.Login(context.Background(), LoginRequest{
		Email: "login@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatal("login failed:", err)
	}
	if resp.Token == "" {
		t.Error("expected non-empty token")
	}
	if resp.User.Email != "login@example.com" {
		t.Errorf("expected login@example.com, got %s", resp.User.Email)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	svc := NewAuthService(&mockRepo{}, []byte("test-key"))
	_, err := svc.Register(context.Background(), RegisterRequest{
		Email: "wrongpw@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatal("registration failed:", err)
	}

	_, err = svc.Login(context.Background(), LoginRequest{
		Email: "wrongpw@example.com", Password: "wrongpassword",
	})
	if err != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_NonExistentEmail(t *testing.T) {
	svc := NewAuthService(&mockRepo{}, []byte("test-key"))
	_, err := svc.Login(context.Background(), LoginRequest{
		Email: "nobody@example.com", Password: "password123",
	})
	if err != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func isValidationError(err error) bool {
	return errors.Is(err, ErrValidation)
}
