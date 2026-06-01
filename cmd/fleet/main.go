package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	databaseURL := getEnv("DATABASE_URL", "postgres://galaxy:galaxy_dev@localhost:5432/galaxy_empire?sslmode=disable")

	bootCtx, bootCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer bootCancel()

	pool, err := pgxpool.New(bootCtx, databaseURL)
	if err != nil {
		slog.Error("connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := runMigrations(bootCtx, pool); err != nil {
		slog.Error("run migrations", "error", err)
		os.Exit(1)
	}

	planetBaseURL := getEnv("PLANET_SERVICE_URL", "http://localhost:8082")
	combatBaseURL := getEnv("COMBAT_SERVICE_URL", "http://localhost:8084")
	authBaseURL := getEnv("AUTH_SERVICE_URL", "http://localhost:8081")
	internalSecret := getEnv("INTERNAL_SECRET", "internal-dev-secret")

	repo := NewPostgresRepository(pool)
	svc := NewFleetService(repo, planetBaseURL, combatBaseURL, authBaseURL, internalSecret)
	h := NewFleetHandler(svc)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"fleet"}`))
	})
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"fleet"}`))
	})
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		pingCtx, pingCancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer pingCancel()
		if err := pool.Ping(pingCtx); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"status":"unavailable","error":err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"fleet"}`))
	})

	r.Get("/api/fleet/my-fleets", h.MyFleets)
	r.Post("/api/fleet/dispatch", h.Dispatch)
	r.Post("/api/fleet/merge", h.MergeFleets)
	r.Post("/api/fleet/{id}/recall", h.RecallFleet)
	r.Post("/api/fleet/{id}/split", h.SplitFleet)

	srv := &http.Server{Addr: ":8083", Handler: r}
	go func() {
		slog.Info("fleet service starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("fleet service fatal", "error", err)
			os.Exit(1)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("travel worker started")
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				func() {
					workerCtx, workerCancel := context.WithTimeout(context.Background(), 30*time.Second)
					defer workerCancel()
					fleets, err := repo.GetArrivedFleets(workerCtx)
					if err != nil {
						slog.Error("travel worker: get arrived fleets", "error", err)
						return
					}
					for _, f := range fleets {
						func() {
							fleetCtx, fleetCancel := context.WithTimeout(workerCtx, 10*time.Second)
							defer fleetCancel()
							switch f.Mission {
							case "transport":
								targetPlanetID, err := svc.FindTargetPlanet(fleetCtx, f.TargetGalaxy, f.TargetSystem, f.TargetPosition)
								if err != nil {
									slog.Error("transport: find planet", "fleet", f.ID, "error", err)
									repo.MarkFleetArrived(fleetCtx, f.ID)
								} else {
									metalAmt := 10000
									crystalAmt := 5000
									if err := svc.AddResourcesToPlanet(fleetCtx, targetPlanetID, "metal", metalAmt); err != nil {
										slog.Error("transport: add metal", "fleet", f.ID, "error", err)
									}
									if err := svc.AddResourcesToPlanet(fleetCtx, targetPlanetID, "crystal", crystalAmt); err != nil {
										slog.Error("transport: add crystal", "fleet", f.ID, "error", err)
									}
									forwardDuration := time.Since(f.CreatedAt)
									returnArrivesAt := time.Now().Add(forwardDuration)
									if err := repo.SetFleetReturning(fleetCtx, f.ID, returnArrivesAt); err != nil {
										slog.Error("transport: set returning", "fleet", f.ID, "error", err)
									} else {
										slog.Info("transport fleet delivered and returning", "fleet", f.ID)
									}
								}
							case "deploy":
								targetPlanetID, err := svc.FindTargetPlanet(fleetCtx, f.TargetGalaxy, f.TargetSystem, f.TargetPosition)
								if err != nil {
									slog.Error("deploy: find planet", "fleet", f.ID, "error", err)
									repo.MarkFleetArrived(fleetCtx, f.ID)
								} else {
									if err := svc.AddShipsToPlanet(fleetCtx, targetPlanetID, f.Ships); err != nil {
										slog.Error("deploy: add ships", "fleet", f.ID, "error", err)
										repo.MarkFleetArrived(fleetCtx, f.ID)
									} else {
										if err := repo.UpdateFleetOrigin(fleetCtx, f.ID, targetPlanetID); err != nil {
											slog.Error("deploy: update origin", "fleet", f.ID, "error", err)
										} else {
											slog.Info("deploy fleet stationed", "fleet", f.ID)
										}
									}
								}
							case "attack":
								if err := repo.UpsertAttackCooldown(fleetCtx, f.PlayerID, f.TargetGalaxy, f.TargetSystem, f.TargetPosition, time.Now()); err != nil {
									slog.Error("travel worker: cooldown", "fleet", f.ID, "error", err)
								}
								autoReturned, vErr := svc.AutoReturnIfVacationMode(fleetCtx, f)
								if vErr != nil {
									slog.Error("travel worker: vacation check failed", "fleet", f.ID, "error", vErr)
								}
								if autoReturned {
									return
								}
								if err := svc.resolveCombatForArrival(fleetCtx, f, nil); err != nil {
									slog.Error("travel worker: combat resolve failed", "fleet", f.ID, "error", err)
								} else {
									slog.Info("combat resolved for fleet", "fleet", f.ID)
								}
								if err := repo.MarkFleetArrived(fleetCtx, f.ID); err != nil {
									slog.Error("mark attack fleet arrived", "fleet", f.ID, "error", err)
								}
							case "acs_attack":
								if f.AllianceGroupID == 0 {
									slog.Error("acs_attack fleet has no alliance group", "fleet", f.ID)
									repo.MarkFleetArrived(fleetCtx, f.ID)
									return
								}
								autoReturned, vErr := svc.AutoReturnIfVacationMode(fleetCtx, f)
								if vErr != nil {
									slog.Error("travel worker: vacation check failed", "fleet", f.ID, "error", vErr)
								}
								if autoReturned {
									return
								}
								allArrived, err := svc.allACSFleetsArrived(fleetCtx, f.AllianceGroupID)
								if err != nil {
									slog.Error("acs_attack: check group arrived", "fleet", f.ID, "error", err)
									return
								}
								if !allArrived {
									slog.Info("acs_attack: waiting for group members", "group", f.AllianceGroupID, "fleet", f.ID)
									return
								}
								groupFleets, err := repo.GetACSGroupFleets(fleetCtx, f.AllianceGroupID)
								if err != nil {
									slog.Error("acs_attack: get group fleets", "error", err)
									return
								}
								combinedShips := make(map[string]int)
								var firstFleet Fleet
								for i, gf := range groupFleets {
									if i == 0 {
										firstFleet = gf
									}
									for shipType, qty := range gf.Ships {
										combinedShips[shipType] += qty
									}
								}
								if err := repo.UpsertAttackCooldown(fleetCtx, f.PlayerID, f.TargetGalaxy, f.TargetSystem, f.TargetPosition, time.Now()); err != nil {
									slog.Error("travel worker: cooldown", "fleet", f.ID, "error", err)
								}
								if err := svc.resolveCombatForArrival(fleetCtx, firstFleet, combinedShips); err != nil {
									slog.Error("travel worker: ACS combat resolve failed", "group", f.AllianceGroupID, "error", err)
								} else {
									slog.Info("ACS combat resolved for group", "group", f.AllianceGroupID)
								}
								for _, gf := range groupFleets {
									if err := repo.MarkFleetArrived(fleetCtx, gf.ID); err != nil {
										slog.Error("travel worker: mark ACS fleet arrived", "fleet", gf.ID, "error", err)
									}
								}
							case "acs_defend":
								targetPlanetID, err := svc.FindTargetPlanet(fleetCtx, f.TargetGalaxy, f.TargetSystem, f.TargetPosition)
								if err != nil {
									slog.Error("acs_defend: find planet", "fleet", f.ID, "error", err)
									repo.MarkFleetArrived(fleetCtx, f.ID)
								} else {
									if err := svc.AddShipsToPlanet(fleetCtx, targetPlanetID, f.Ships); err != nil {
										slog.Error("acs_defend: add ships", "fleet", f.ID, "error", err)
										repo.MarkFleetArrived(fleetCtx, f.ID)
									} else {
										if err := repo.UpdateFleetOrigin(fleetCtx, f.ID, targetPlanetID); err != nil {
											slog.Error("acs_defend: update origin", "fleet", f.ID, "error", err)
										} else {
											slog.Info("acs_defend fleet stationed", "fleet", f.ID)
										}
									}
								}
							case "recycle":
								if err := svc.harvestDebris(fleetCtx, f.ID, f.Ships, f.OriginPlanetID, f.TargetGalaxy, f.TargetSystem, f.TargetPosition); err != nil {
									slog.Error("recycle: harvest failed", "fleet", f.ID, "error", err)
								}
								forwardDuration := time.Since(f.CreatedAt)
								returnArrivesAt := time.Now().Add(forwardDuration)
								if err := repo.SetFleetReturning(fleetCtx, f.ID, returnArrivesAt); err != nil {
									slog.Error("recycle: set returning", "fleet", f.ID, "error", err)
								} else {
									slog.Info("recycle fleet harvested and returning", "fleet", f.ID)
								}
							case "colonize":
								if err := svc.handleColonizeArrival(fleetCtx, f); err != nil {
									slog.Error("colonize: failed", "fleet", f.ID, "error", err)
									repo.MarkFleetArrived(fleetCtx, f.ID)
								} else {
									slog.Info("colony established", "fleet", f.ID)
								}
							case "stargate":
								targetPlanetID, err := svc.FindTargetPlanet(fleetCtx, f.TargetGalaxy, f.TargetSystem, f.TargetPosition)
								if err != nil {
									slog.Error("stargate: find planet", "fleet", f.ID, "error", err)
									repo.MarkFleetArrived(fleetCtx, f.ID)
								} else {
									if err := svc.AddShipsToPlanet(fleetCtx, targetPlanetID, f.Ships); err != nil {
										slog.Error("stargate: add ships", "fleet", f.ID, "error", err)
										repo.MarkFleetArrived(fleetCtx, f.ID)
									} else {
										if err := repo.SetFleetReturning(fleetCtx, f.ID, time.Now()); err != nil {
											slog.Error("stargate: set returning", "fleet", f.ID, "error", err)
										} else {
											slog.Info("stargate fleet delivered and returning", "fleet", f.ID)
										}
									}
								}
							default:
								if err := repo.MarkFleetArrived(fleetCtx, f.ID); err != nil {
									slog.Error("travel worker: mark arrived", "fleet", f.ID, "error", err)
								} else {
									slog.Info("fleet arrived", "fleet", f.ID, "mission", f.Mission)
								}
							}
						}()
					}
				}()
			case <-ctx.Done():
				slog.Info("travel worker stopped")
				return
			}
		}
	}()

	<-ctx.Done()

	slog.Info("fleet service shutting down")
	shutdownTimeout := getEnvDuration("SHUTDOWN_TIMEOUT", 15*time.Second)
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()
	srv.Shutdown(shutdownCtx)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		d, err := time.ParseDuration(v)
		if err == nil {
			return d
		}
	}
	return fallback
}

func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, `
		CREATE SCHEMA IF NOT EXISTS fleet;
		CREATE TABLE IF NOT EXISTS fleet.fleets (
			id SERIAL PRIMARY KEY,
			player_id INT NOT NULL,
			origin_planet_id INT NOT NULL,
			target_galaxy INT NOT NULL,
			target_system INT NOT NULL,
			target_position INT NOT NULL,
			mission VARCHAR(20) NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'stationed',
			speed_pct INT NOT NULL DEFAULT 100,
			ships JSONB NOT NULL DEFAULT '{}',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`	); err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, `
		ALTER TABLE fleet.fleets ADD COLUMN IF NOT EXISTS arrives_at TIMESTAMPTZ;
	`); err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS fleet.attack_cooldowns (
			id SERIAL PRIMARY KEY,
			attacker_id INT NOT NULL,
			target_galaxy INT NOT NULL,
			target_system INT NOT NULL,
			target_position INT NOT NULL,
			last_attack_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE(attacker_id, target_galaxy, target_system, target_position)
		);
	`); err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, `
		ALTER TABLE fleet.fleets ADD COLUMN IF NOT EXISTS alliance_group_id INT NOT NULL DEFAULT 0;
	`); err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS fleet.debris_fields (
			id SERIAL PRIMARY KEY,
			galaxy INT NOT NULL,
			system INT NOT NULL,
			position INT NOT NULL,
			metal INT NOT NULL DEFAULT 0,
			crystal INT NOT NULL DEFAULT 0,
			UNIQUE(galaxy, system, position)
		);
	`); err != nil {
		return err
	}
	return nil
}
