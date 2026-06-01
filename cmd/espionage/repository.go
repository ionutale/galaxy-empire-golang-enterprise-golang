package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	CreateReport(ctx context.Context, r EspionageReport) (EspionageReport, error)
	GetReportByID(ctx context.Context, reportID int) (EspionageReport, error)
	ListReportsForPlayer(ctx context.Context, playerID int) ([]EspionageReport, error)
	DeleteReport(ctx context.Context, reportID int) error
}

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) CreateReport(ctx context.Context, report EspionageReport) (EspionageReport, error) {
	resourcesJSON, _ := json.Marshal(report.Resources)
	fleetJSON, _ := json.Marshal(report.Fleet)
	defenseJSON, _ := json.Marshal(report.Defense)
	techJSON, _ := json.Marshal(report.Tech)
	reportDataJSON, _ := json.Marshal(report.ReportData)

	var id int
	var createdAt, expiresAt time.Time
	err := r.pool.QueryRow(ctx, `
		INSERT INTO espionage.espionage_reports
			(player_id, target_player_id, target_galaxy, target_system, target_position,
			 detail_level, resources, fleet, defense, tech, report_data, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at, expires_at
	`,
		report.PlayerID, report.TargetPlayerID,
		report.TargetGalaxy, report.TargetSystem, report.TargetPosition,
		report.DetailLevel,
		resourcesJSON, fleetJSON, defenseJSON, techJSON, reportDataJSON,
		report.ExpiresAt,
	).Scan(&id, &createdAt, &expiresAt)
	if err != nil {
		return EspionageReport{}, fmt.Errorf("create report: %w", err)
	}
	report.ID = id
	report.CreatedAt = createdAt
	report.ExpiresAt = expiresAt
	return report, nil
}

func (r *PostgresRepository) GetReportByID(ctx context.Context, reportID int) (EspionageReport, error) {
	var rep EspionageReport
	var resourcesJSON, fleetJSON, defenseJSON, techJSON, reportDataJSON []byte

	err := r.pool.QueryRow(ctx, `
		SELECT id, player_id, target_player_id, target_galaxy, target_system, target_position,
		       detail_level, resources, fleet, defense, tech, report_data, created_at, expires_at
		FROM espionage.espionage_reports
		WHERE id = $1
	`, reportID).Scan(
		&rep.ID, &rep.PlayerID, &rep.TargetPlayerID,
		&rep.TargetGalaxy, &rep.TargetSystem, &rep.TargetPosition,
		&rep.DetailLevel, &resourcesJSON, &fleetJSON, &defenseJSON, &techJSON,
		&reportDataJSON, &rep.CreatedAt, &rep.ExpiresAt,
	)
	if err != nil {
		return EspionageReport{}, fmt.Errorf("get report: %w", err)
	}

	json.Unmarshal(resourcesJSON, &rep.Resources)
	json.Unmarshal(fleetJSON, &rep.Fleet)
	json.Unmarshal(defenseJSON, &rep.Defense)
	json.Unmarshal(techJSON, &rep.Tech)
	json.Unmarshal(reportDataJSON, &rep.ReportData)

	return rep, nil
}

func (r *PostgresRepository) ListReportsForPlayer(ctx context.Context, playerID int) ([]EspionageReport, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, player_id, target_player_id, target_galaxy, target_system, target_position,
		       detail_level, resources, fleet, defense, tech, report_data, created_at, expires_at
		FROM espionage.espionage_reports
		WHERE player_id = $1 OR target_player_id = $1
		ORDER BY created_at DESC
	`, playerID)
	if err != nil {
		return nil, fmt.Errorf("list reports: %w", err)
	}
	defer rows.Close()

	var reports []EspionageReport
	for rows.Next() {
		var rep EspionageReport
		var resourcesJSON, fleetJSON, defenseJSON, techJSON, reportDataJSON []byte
		if err := rows.Scan(
			&rep.ID, &rep.PlayerID, &rep.TargetPlayerID,
			&rep.TargetGalaxy, &rep.TargetSystem, &rep.TargetPosition,
			&rep.DetailLevel, &resourcesJSON, &fleetJSON, &defenseJSON, &techJSON,
			&reportDataJSON, &rep.CreatedAt, &rep.ExpiresAt,
		); err != nil {
			return nil, fmt.Errorf("scan report: %w", err)
		}
		json.Unmarshal(resourcesJSON, &rep.Resources)
		json.Unmarshal(fleetJSON, &rep.Fleet)
		json.Unmarshal(defenseJSON, &rep.Defense)
		json.Unmarshal(techJSON, &rep.Tech)
		json.Unmarshal(reportDataJSON, &rep.ReportData)
		reports = append(reports, rep)
	}
	return reports, rows.Err()
}

func (r *PostgresRepository) DeleteReport(ctx context.Context, reportID int) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM espionage.espionage_reports WHERE id = $1`, reportID)
	return err
}

type mockRepo struct {
	reports []EspionageReport
	nextID  int
}

func newMockRepo() *mockRepo {
	return &mockRepo{nextID: 1}
}

func (m *mockRepo) CreateReport(ctx context.Context, report EspionageReport) (EspionageReport, error) {
	report.ID = m.nextID
	m.nextID++
	now := time.Now()
	report.CreatedAt = now
	if report.ExpiresAt.IsZero() {
		report.ExpiresAt = now.Add(24 * time.Hour)
	}
	m.reports = append(m.reports, report)
	return report, nil
}

func (m *mockRepo) GetReportByID(ctx context.Context, reportID int) (EspionageReport, error) {
	for _, r := range m.reports {
		if r.ID == reportID {
			return r, nil
		}
	}
	return EspionageReport{}, fmt.Errorf("report not found")
}

func (m *mockRepo) ListReportsForPlayer(ctx context.Context, playerID int) ([]EspionageReport, error) {
	var result []EspionageReport
	for _, r := range m.reports {
		if r.PlayerID == playerID || r.TargetPlayerID == playerID {
			result = append(result, r)
		}
	}
	return result, nil
}

func (m *mockRepo) DeleteReport(ctx context.Context, reportID int) error {
	for i, r := range m.reports {
		if r.ID == reportID {
			m.reports = append(m.reports[:i], m.reports[i+1:]...)
			return nil
		}
	}
	return nil
}
