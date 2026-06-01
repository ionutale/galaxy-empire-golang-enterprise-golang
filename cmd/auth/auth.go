package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
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

const vacationCooldownHours = 48

func (s *AuthService) EnableVacation(ctx context.Context, userID int) error {
	return s.repo.SetVacationStart(ctx, userID)
}

func (s *AuthService) ConfirmVacation(ctx context.Context, userID int) error {
	enabled, startedAt, err := s.repo.GetVacationStatus(ctx, userID)
	if err != nil {
		return err
	}
	if enabled {
		return nil
	}
	if startedAt == nil {
		return fmt.Errorf("%w: vacation mode not initiated", ErrValidation)
	}
	if time.Since(*startedAt) < vacationCooldownHours*time.Hour {
		remaining := vacationCooldownHours*time.Hour - time.Since(*startedAt)
		return fmt.Errorf("%w: %.0f hours remaining before vacation can be confirmed", ErrValidation, remaining.Hours())
	}
	return s.repo.SetVacationEnabled(ctx, userID, true)
}

func (s *AuthService) DisableVacation(ctx context.Context, userID int) error {
	return s.repo.ClearVacationMode(ctx, userID)
}

func (s *AuthService) GetVacationStatus(ctx context.Context, userID int) (VacationStatusResponse, error) {
	enabled, startedAt, err := s.repo.GetVacationStatus(ctx, userID)
	if err != nil {
		return VacationStatusResponse{}, err
	}
	resp := VacationStatusResponse{
		Enabled:   enabled,
		CanConfirm: false,
	}
	if startedAt != nil {
		t := startedAt.Format(time.RFC3339)
		resp.StartedAt = &t
		elapsed := time.Since(*startedAt)
		if elapsed >= vacationCooldownHours*time.Hour {
			resp.CanConfirm = true
			resp.RemainingHours = 0
		} else {
			resp.RemainingHours = (vacationCooldownHours*time.Hour - elapsed).Hours()
		}
	}
	return resp, nil
}

func (s *AuthService) GetUserVacationStatus(ctx context.Context, userID int) (UserVacationStatusResponse, error) {
	enabled, err := s.repo.GetVacationEnabled(ctx, userID)
	if err != nil {
		return UserVacationStatusResponse{}, err
	}
	return UserVacationStatusResponse{VacationModeEnabled: enabled}, nil
}

func (s *AuthService) JWTMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing authorization header"})
			return
		}
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenStr == authHeader {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid authorization format"})
			return
		}
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
			return s.jwtKey, nil
		})
		if err != nil || !token.Valid {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid or expired token"})
			return
		}
		ctx := context.WithValue(r.Context(), ctxKeyUserID, claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type contextKey string

const ctxKeyUserID contextKey = "user_id"

func UserIDFromContext(ctx context.Context) (int, bool) {
	id, ok := ctx.Value(ctxKeyUserID).(int)
	return id, ok
}

func InternalSecretMiddleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-Internal-Secret") != secret {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
