package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrDuplicateEmail     = errors.New("email already registered")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrValidation         = errors.New("validation error")
)

type AuthService struct {
	repo   Repository
	jwtKey []byte
}

func NewAuthService(repo Repository, jwtKey []byte) *AuthService {
	return &AuthService{repo: repo, jwtKey: jwtKey}
}

type Claims struct {
	UserID int    `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

func (s *AuthService) Register(ctx context.Context, req RegisterRequest) (AuthResponse, error) {
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	parts := strings.SplitN(req.Email, "@", 2)
	if len(parts) != 2 || parts[0] == "" || !strings.Contains(parts[1], ".") {
		return AuthResponse{}, fmt.Errorf("%w: invalid email", ErrValidation)
	}
	if len(req.Password) < 6 {
		return AuthResponse{}, fmt.Errorf("%w: password must be at least 6 characters", ErrValidation)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return AuthResponse{}, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.repo.Create(ctx, req.Email, string(hash))
	if err != nil {
		return AuthResponse{}, err
	}

	token, err := generateToken(user.ID, user.Email, s.jwtKey)
	if err != nil {
		return AuthResponse{}, fmt.Errorf("generate token: %w", err)
	}

	return AuthResponse{Token: token, User: toUserResponse(user)}, nil
}

func (s *AuthService) Login(ctx context.Context, req LoginRequest) (AuthResponse, error) {
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	user, err := s.repo.FindByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return AuthResponse{}, ErrInvalidCredentials
		}
		return AuthResponse{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return AuthResponse{}, ErrInvalidCredentials
	}

	token, err := generateToken(user.ID, user.Email, s.jwtKey)
	if err != nil {
		return AuthResponse{}, fmt.Errorf("generate token: %w", err)
	}

	return AuthResponse{Token: token, User: toUserResponse(user)}, nil
}

func generateToken(userID int, email string, jwtKey []byte) (string, error) {
	claims := Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

func toUserResponse(u User) UserResponse {
	return UserResponse{ID: u.ID, Email: u.Email, CreatedAt: u.CreatedAt}
}
