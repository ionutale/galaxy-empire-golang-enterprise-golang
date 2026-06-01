package main

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	service *AuthService
}

func NewHandler(service *AuthService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	resp, err := h.service.Register(r.Context(), req)
	if err != nil {
		slog.Error("register failed", "error", err)
		code := http.StatusInternalServerError
		msg := "internal error"
		switch {
		case errors.Is(err, ErrDuplicateEmail):
			code = http.StatusConflict
			msg = "email already registered"
		case errors.Is(err, ErrValidation):
			code = http.StatusBadRequest
			msg = err.Error()
		}
		writeJSON(w, code, map[string]string{"error": msg})
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	resp, err := h.service.Login(r.Context(), req)
	if err != nil {
		slog.Error("login failed", "error", err)
		code := http.StatusInternalServerError
		msg := "internal error"
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			code = http.StatusUnauthorized
			msg = "invalid email or password"
		case errors.Is(err, ErrValidation):
			code = http.StatusBadRequest
			msg = err.Error()
		}
		writeJSON(w, code, map[string]string{"error": msg})
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) EnableVacation(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	if err := h.service.EnableVacation(r.Context(), userID); err != nil {
		slog.Error("enable vacation failed", "user_id", userID, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"message": "vacation mode initiated", "cooldown_hours": 48})
}

func (h *Handler) ConfirmVacation(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	if err := h.service.ConfirmVacation(r.Context(), userID); err != nil {
		slog.Error("confirm vacation failed", "user_id", userID, "error", err)
		code := http.StatusInternalServerError
		msg := "internal error"
		if errors.Is(err, ErrValidation) {
			code = http.StatusBadRequest
			msg = err.Error()
		}
		writeJSON(w, code, map[string]string{"error": msg})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"message": "vacation mode enabled"})
}

func (h *Handler) DisableVacation(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	if err := h.service.DisableVacation(r.Context(), userID); err != nil {
		slog.Error("disable vacation failed", "user_id", userID, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"message": "vacation mode disabled"})
}

func (h *Handler) VacationStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	status, err := h.service.GetVacationStatus(r.Context(), userID)
	if err != nil {
		slog.Error("get vacation status failed", "user_id", userID, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, status)
}

func (h *Handler) UserVacationStatus(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		slog.Error("invalid user id", "id", userIDStr)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user id"})
		return
	}

	status, err := h.service.GetUserVacationStatus(r.Context(), userID)
	if err != nil {
		slog.Error("get user vacation status failed", "user_id", userID, "error", err)
		code := http.StatusInternalServerError
		msg := "internal error"
		if errors.Is(err, ErrUserNotFound) {
			code = http.StatusNotFound
			msg = "user not found"
		}
		writeJSON(w, code, map[string]string{"error": msg})
		return
	}

	writeJSON(w, http.StatusOK, status)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
