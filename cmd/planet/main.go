package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Planet struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Owner    string `json:"owner"`
	Metal    int    `json:"metal"`
	Crystal  int    `json:"crystal"`
	Gas      int    `json:"gas"`
	Energy   int    `json:"energy"`
	Galaxy   int    `json:"galaxy"`
	System   int    `json:"system"`
	Position int    `json:"position"`
}

var planets = map[int]Planet{
	1: {
		ID: 1, Name: "Homeworld", Owner: "Commander",
		Galaxy: 1, System: 1, Position: 7,
		Metal: 500, Crystal: 300, Gas: 200, Energy: 50,
	},
	2: {
		ID: 2, Name: "Colony-1", Owner: "Commander",
		Galaxy: 1, System: 2, Position: 4,
		Metal: 200, Crystal: 150, Gas: 100, Energy: 25,
	},
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"planet"}`))
	})

	r.Get("/api/planet/{id}", func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, `{"error":"invalid planet id"}`, http.StatusBadRequest)
			return
		}
		p, ok := planets[id]
		if !ok {
			http.Error(w, `{"error":"planet not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(p)
	})

	srv := &http.Server{Addr: ":8082", Handler: r}
	go func() {
		slog.Info("planet service starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("planet service fatal", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("planet service shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}
