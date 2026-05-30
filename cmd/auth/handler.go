package main

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
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

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
