package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	CreateAlliance(ctx context.Context, name, tag string, founderID int) (Alliance, error)
	GetAlliance(ctx context.Context, id int) (*Alliance, error)
	GetAllianceByName(ctx context.Context, name string) (*Alliance, error)
	GetAllianceByTag(ctx context.Context, tag string) (*Alliance, error)
	AddMember(ctx context.Context, allianceID, playerID int, role string) (Member, error)
	RemoveMember(ctx context.Context, playerID int) error
	GetMember(ctx context.Context, playerID int) (*Member, error)
	GetMembers(ctx context.Context, allianceID int) ([]Member, error)
	CountMembers(ctx context.Context, allianceID int) (int, error)
	UpdateMemberRole(ctx context.Context, allianceID, playerID int, role string) error
	GetBank(ctx context.Context, allianceID int) (*Bank, error)
	UpdateBank(ctx context.Context, allianceID int, metal, crystal, gas int) error
	WithdrawBank(ctx context.Context, allianceID, metal, crystal, gas int) (*Bank, error)
	AddAuditLog(ctx context.Context, allianceID, playerID int, action string, details map[string]any) error
	PostBulletin(ctx context.Context, allianceID, authorPlayerID int, title, content string) (Bulletin, error)
	GetBulletins(ctx context.Context, allianceID int) ([]Bulletin, error)
	DeleteBulletin(ctx context.Context, bulletinID int) error
	GetMemberBulletin(ctx context.Context, bulletinID int) (*Member, error)
	UpdatePlayerLastActive(ctx context.Context, playerID int) error
	ShareReport(ctx context.Context, allianceID, playerID, reportID int) error
	GetSharedReports(ctx context.Context, allianceID int) ([]SharedReport, error)
	RemoveSharedReport(ctx context.Context, reportID int) error
	GetSharedReport(ctx context.Context, reportID int) (*SharedReport, error)
}

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) CreateAlliance(ctx context.Context, name, tag string, founderID int) (Alliance, error) {
	var a Alliance
	err := r.pool.QueryRow(ctx, `
		INSERT INTO alliance.alliances (name, tag, founder_id)
		VALUES ($1, $2, $3)
		RETURNING id, name, tag, founder_id, created_at
	`, name, tag, founderID).Scan(&a.ID, &a.Name, &a.Tag, &a.FounderID, &a.CreatedAt)
	if err != nil {
		return Alliance{}, fmt.Errorf("create alliance: %w", err)
	}
	return a, nil
}

func (r *PostgresRepository) GetAlliance(ctx context.Context, id int) (*Alliance, error) {
	var a Alliance
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, tag, founder_id, created_at
		FROM alliance.alliances
		WHERE id = $1
	`, id).Scan(&a.ID, &a.Name, &a.Tag, &a.FounderID, &a.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("alliance not found")
		}
		return nil, fmt.Errorf("get alliance: %w", err)
	}
	return &a, nil
}

func (r *PostgresRepository) GetAllianceByName(ctx context.Context, name string) (*Alliance, error) {
	var a Alliance
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, tag, founder_id, created_at
		FROM alliance.alliances
		WHERE name = $1
	`, name).Scan(&a.ID, &a.Name, &a.Tag, &a.FounderID, &a.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get alliance by name: %w", err)
	}
	return &a, nil
}

func (r *PostgresRepository) GetAllianceByTag(ctx context.Context, tag string) (*Alliance, error) {
	var a Alliance
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, tag, founder_id, created_at
		FROM alliance.alliances
		WHERE tag = $1
	`, tag).Scan(&a.ID, &a.Name, &a.Tag, &a.FounderID, &a.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get alliance by tag: %w", err)
	}
	return &a, nil
}

func (r *PostgresRepository) AddMember(ctx context.Context, allianceID, playerID int, role string) (Member, error) {
	var m Member
	err := r.pool.QueryRow(ctx, `
		INSERT INTO alliance.members (alliance_id, player_id, role)
		VALUES ($1, $2, $3)
		RETURNING id, alliance_id, player_id, role, joined_at
	`, allianceID, playerID, role).Scan(&m.ID, &m.AllianceID, &m.PlayerID, &m.Role, &m.JoinedAt)
	if err != nil {
		return Member{}, fmt.Errorf("add member: %w", err)
	}
	return m, nil
}

func (r *PostgresRepository) RemoveMember(ctx context.Context, playerID int) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM alliance.members
		WHERE player_id = $1
	`, playerID)
	if err != nil {
		return fmt.Errorf("remove member: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetMember(ctx context.Context, playerID int) (*Member, error) {
	var m Member
	err := r.pool.QueryRow(ctx, `
		SELECT id, alliance_id, player_id, role, joined_at, last_active_at
		FROM alliance.members
		WHERE player_id = $1
	`, playerID).Scan(&m.ID, &m.AllianceID, &m.PlayerID, &m.Role, &m.JoinedAt, &m.LastActiveAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get member: %w", err)
	}
	return &m, nil
}

func (r *PostgresRepository) GetMembers(ctx context.Context, allianceID int) ([]Member, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, alliance_id, player_id, role, joined_at, last_active_at
		FROM alliance.members
		WHERE alliance_id = $1
		ORDER BY joined_at ASC
	`, allianceID)
	if err != nil {
		return nil, fmt.Errorf("get members: %w", err)
	}
	defer rows.Close()

	var members []Member
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.ID, &m.AllianceID, &m.PlayerID, &m.Role, &m.JoinedAt, &m.LastActiveAt); err != nil {
			return nil, fmt.Errorf("scan member: %w", err)
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

func (r *PostgresRepository) CountMembers(ctx context.Context, allianceID int) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM alliance.members WHERE alliance_id = $1
	`, allianceID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count members: %w", err)
	}
	return count, nil
}

func (r *PostgresRepository) UpdateMemberRole(ctx context.Context, allianceID, playerID int, role string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE alliance.members
		SET role = $1
		WHERE alliance_id = $2 AND player_id = $3
	`, role, allianceID, playerID)
	return err
}

func (r *PostgresRepository) GetBank(ctx context.Context, allianceID int) (*Bank, error) {
	var b Bank
	err := r.pool.QueryRow(ctx, `
		SELECT alliance_id, metal, crystal, gas
		FROM alliance.bank
		WHERE alliance_id = $1
	`, allianceID).Scan(&b.AllianceID, &b.Metal, &b.Crystal, &b.Gas)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get bank: %w", err)
	}
	return &b, nil
}

func (r *PostgresRepository) UpdateBank(ctx context.Context, allianceID int, metal, crystal, gas int) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO alliance.bank (alliance_id, metal, crystal, gas)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (alliance_id)
		DO UPDATE SET metal = $2, crystal = $3, gas = $4
	`, allianceID, metal, crystal, gas)
	return err
}

// WithdrawBank atomically deducts resources from the alliance bank, checking
// sufficiency in SQL to prevent the TOCTOU race of concurrent withdrawals.
func (r *PostgresRepository) WithdrawBank(ctx context.Context, allianceID, metal, crystal, gas int) (*Bank, error) {
	var b Bank
	err := r.pool.QueryRow(ctx, `
		UPDATE alliance.bank
		SET metal   = metal   - $2,
		    crystal = crystal - $3,
		    gas     = gas     - $4
		WHERE alliance_id = $1
		  AND metal   >= $2
		  AND crystal >= $3
		  AND gas     >= $4
		RETURNING alliance_id, metal, crystal, gas
	`, allianceID, metal, crystal, gas).Scan(&b.AllianceID, &b.Metal, &b.Crystal, &b.Gas)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("insufficient resources in bank or bank not found")
		}
		return nil, fmt.Errorf("withdraw bank: %w", err)
	}
	return &b, nil
}

func (r *PostgresRepository) AddAuditLog(ctx context.Context, allianceID, playerID int, action string, details map[string]any) error {
	detailsJSON, _ := json.Marshal(details)
	_, err := r.pool.Exec(ctx, `
		INSERT INTO alliance.audit_log (alliance_id, player_id, action, details)
		VALUES ($1, $2, $3, $4)
	`, allianceID, playerID, action, detailsJSON)
	return err
}

func (r *PostgresRepository) PostBulletin(ctx context.Context, allianceID, authorPlayerID int, title, content string) (Bulletin, error) {
	var b Bulletin
	err := r.pool.QueryRow(ctx, `
		INSERT INTO alliance.bulletins (alliance_id, author_player_id, title, content)
		VALUES ($1, $2, $3, $4)
		RETURNING id, alliance_id, author_player_id, title, content, created_at, updated_at
	`, allianceID, authorPlayerID, title, content).Scan(&b.ID, &b.AllianceID, &b.AuthorPlayerID, &b.Title, &b.Content, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return Bulletin{}, fmt.Errorf("post bulletin: %w", err)
	}
	return b, nil
}

func (r *PostgresRepository) GetBulletins(ctx context.Context, allianceID int) ([]Bulletin, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, alliance_id, author_player_id, title, content, created_at, updated_at
		FROM alliance.bulletins
		WHERE alliance_id = $1
		ORDER BY created_at DESC LIMIT 50
`, allianceID)
	if err != nil {
		return nil, fmt.Errorf("get bulletins: %w", err)
	}
	defer rows.Close()

	var bulletins []Bulletin
	for rows.Next() {
		var b Bulletin
		if err := rows.Scan(&b.ID, &b.AllianceID, &b.AuthorPlayerID, &b.Title, &b.Content, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan bulletin: %w", err)
		}
		bulletins = append(bulletins, b)
	}
	return bulletins, rows.Err()
}

func (r *PostgresRepository) DeleteBulletin(ctx context.Context, bulletinID int) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM alliance.bulletins
		WHERE id = $1
	`, bulletinID)
	return err
}

func (r *PostgresRepository) GetMemberBulletin(ctx context.Context, bulletinID int) (*Member, error) {
	var m Member
	err := r.pool.QueryRow(ctx, `
		SELECT m.id, m.alliance_id, m.player_id, m.role, m.joined_at, m.last_active_at
		FROM alliance.bulletins b
		JOIN alliance.members m ON m.player_id = b.author_player_id
		WHERE b.id = $1
	`, bulletinID).Scan(&m.ID, &m.AllianceID, &m.PlayerID, &m.Role, &m.JoinedAt, &m.LastActiveAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("bulletin not found")
		}
		return nil, fmt.Errorf("get member bulletin: %w", err)
	}
	return &m, nil
}

func (r *PostgresRepository) UpdatePlayerLastActive(ctx context.Context, playerID int) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE alliance.members
		SET last_active_at = NOW()
		WHERE player_id = $1
	`, playerID)
	return err
}

func (r *PostgresRepository) ShareReport(ctx context.Context, allianceID, playerID, reportID int) error {
	ct, err := r.pool.Exec(ctx, `
		INSERT INTO alliance.shared_reports (alliance_id, report_id, shared_by)
		VALUES ($1, $2, $3)
		ON CONFLICT (alliance_id, report_id) DO NOTHING
	`, allianceID, reportID, playerID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("report already shared")
	}
	return nil
}

func (r *PostgresRepository) GetSharedReports(ctx context.Context, allianceID int) ([]SharedReport, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, alliance_id, report_id, shared_by, shared_at
		FROM alliance.shared_reports
		WHERE alliance_id = $1
		ORDER BY id DESC LIMIT 50
`, allianceID)
	if err != nil {
		return nil, fmt.Errorf("get shared reports: %w", err)
	}
	defer rows.Close()

	var reports []SharedReport
	for rows.Next() {
		var sr SharedReport
		if err := rows.Scan(&sr.ID, &sr.AllianceID, &sr.ReportID, &sr.SharedBy, &sr.SharedAt); err != nil {
			return nil, fmt.Errorf("scan shared report: %w", err)
		}
		reports = append(reports, sr)
	}
	return reports, rows.Err()
}

func (r *PostgresRepository) RemoveSharedReport(ctx context.Context, reportID int) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM alliance.shared_reports
		WHERE id = $1
	`, reportID)
	return err
}

func (r *PostgresRepository) GetSharedReport(ctx context.Context, reportID int) (*SharedReport, error) {
	var sr SharedReport
	err := r.pool.QueryRow(ctx, `
		SELECT id, alliance_id, report_id, shared_by, shared_at
		FROM alliance.shared_reports
		WHERE id = $1
	`, reportID).Scan(&sr.ID, &sr.AllianceID, &sr.ReportID, &sr.SharedBy, &sr.SharedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("shared report not found")
		}
		return nil, fmt.Errorf("get shared report: %w", err)
	}
	return &sr, nil
}

type mockRepo struct {
	mu              sync.Mutex
	alliances       []Alliance
	members         []Member
	banks           []Bank
	auditLogs       []AuditEntry
	bulletins       []Bulletin
	sharedReports   []SharedReport
	nextID          int
	nextMember      int
	nextBulletin    int
	nextSharedRpt   int
}

func newMockRepo() *mockRepo {
	return &mockRepo{nextID: 1, nextMember: 1, nextBulletin: 1, nextSharedRpt: 1}
}

func (m *mockRepo) CreateAlliance(ctx context.Context, name, tag string, founderID int) (Alliance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, a := range m.alliances {
		if a.Name == name {
			return Alliance{}, fmt.Errorf("alliance name already taken")
		}
		if a.Tag == tag {
			return Alliance{}, fmt.Errorf("alliance tag already taken")
		}
	}

	a := Alliance{
		ID:        m.nextID,
		Name:      name,
		Tag:       tag,
		FounderID: founderID,
		CreatedAt: time.Now(),
	}
	m.nextID++
	m.alliances = append(m.alliances, a)
	return a, nil
}

func (m *mockRepo) GetAlliance(ctx context.Context, id int) (*Alliance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, a := range m.alliances {
		if a.ID == id {
			return &a, nil
		}
	}
	return nil, fmt.Errorf("alliance not found")
}

func (m *mockRepo) GetAllianceByName(ctx context.Context, name string) (*Alliance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, a := range m.alliances {
		if a.Name == name {
			return &a, nil
		}
	}
	return nil, nil
}

func (m *mockRepo) GetAllianceByTag(ctx context.Context, tag string) (*Alliance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, a := range m.alliances {
		if a.Tag == tag {
			return &a, nil
		}
	}
	return nil, nil
}

func (m *mockRepo) AddMember(ctx context.Context, allianceID, playerID int, role string) (Member, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, mb := range m.members {
		if mb.PlayerID == playerID {
			return Member{}, fmt.Errorf("player already in an alliance")
		}
	}

	mb := Member{
		ID:         m.nextMember,
		AllianceID: allianceID,
		PlayerID:   playerID,
		Role:       role,
		JoinedAt:   time.Now(),
	}
	m.nextMember++
	m.members = append(m.members, mb)
	return mb, nil
}

func (m *mockRepo) RemoveMember(ctx context.Context, playerID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, mb := range m.members {
		if mb.PlayerID == playerID {
			m.members = append(m.members[:i], m.members[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("member not found")
}

func (m *mockRepo) GetMember(ctx context.Context, playerID int) (*Member, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, mb := range m.members {
		if mb.PlayerID == playerID {
			m := mb
			return &m, nil
		}
	}
	return nil, nil
}

func (m *mockRepo) GetMembers(ctx context.Context, allianceID int) ([]Member, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []Member
	for _, mb := range m.members {
		if mb.AllianceID == allianceID {
			result = append(result, mb)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].JoinedAt.Before(result[j].JoinedAt)
	})
	return result, nil
}

func (m *mockRepo) CountMembers(ctx context.Context, allianceID int) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := 0
	for _, mb := range m.members {
		if mb.AllianceID == allianceID {
			count++
		}
	}
	return count, nil
}

func (m *mockRepo) UpdateMemberRole(ctx context.Context, allianceID, playerID int, role string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, mb := range m.members {
		if mb.AllianceID == allianceID && mb.PlayerID == playerID {
			m.members[i].Role = role
			return nil
		}
	}
	return fmt.Errorf("member not found")
}

func (m *mockRepo) GetBank(ctx context.Context, allianceID int) (*Bank, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, b := range m.banks {
		if b.AllianceID == allianceID {
			return &b, nil
		}
	}
	return nil, nil
}

func (m *mockRepo) UpdateBank(ctx context.Context, allianceID int, metal, crystal, gas int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, b := range m.banks {
		if b.AllianceID == allianceID {
			m.banks[i].Metal = metal
			m.banks[i].Crystal = crystal
			m.banks[i].Gas = gas
			return nil
		}
	}

	m.banks = append(m.banks, Bank{
		AllianceID: allianceID,
		Metal:      metal,
		Crystal:    crystal,
		Gas:        gas,
	})
	return nil
}

func (m *mockRepo) WithdrawBank(ctx context.Context, allianceID, metal, crystal, gas int) (*Bank, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, b := range m.banks {
		if b.AllianceID == allianceID {
			if b.Metal < metal || b.Crystal < crystal || b.Gas < gas {
				return nil, fmt.Errorf("insufficient resources in bank")
			}
			m.banks[i].Metal -= metal
			m.banks[i].Crystal -= crystal
			m.banks[i].Gas -= gas
			updated := m.banks[i]
			return &updated, nil
		}
	}
	return nil, fmt.Errorf("insufficient resources in bank or bank not found")
}

func (m *mockRepo) AddAuditLog(ctx context.Context, allianceID, playerID int, action string, details map[string]any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.auditLogs = append(m.auditLogs, AuditEntry{
		ID:         len(m.auditLogs) + 1,
		AllianceID: allianceID,
		PlayerID:   playerID,
		Action:     action,
		Details:    details,
		CreatedAt:  time.Now(),
	})
	return nil
}

func (m *mockRepo) PostBulletin(ctx context.Context, allianceID, authorPlayerID int, title, content string) (Bulletin, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	b := Bulletin{
		ID:             m.nextBulletin,
		AllianceID:     allianceID,
		AuthorPlayerID: authorPlayerID,
		Title:          title,
		Content:        content,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	m.nextBulletin++
	m.bulletins = append(m.bulletins, b)
	return b, nil
}

func (m *mockRepo) GetBulletins(ctx context.Context, allianceID int) ([]Bulletin, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []Bulletin
	for _, b := range m.bulletins {
		if b.AllianceID == allianceID {
			result = append(result, b)
		}
	}
	return result, nil
}

func (m *mockRepo) DeleteBulletin(ctx context.Context, bulletinID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, b := range m.bulletins {
		if b.ID == bulletinID {
			m.bulletins = append(m.bulletins[:i], m.bulletins[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("bulletin not found")
}

func (m *mockRepo) GetMemberBulletin(ctx context.Context, bulletinID int) (*Member, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, b := range m.bulletins {
		if b.ID == bulletinID {
			for _, mb := range m.members {
				if mb.PlayerID == b.AuthorPlayerID {
					mem := mb
					return &mem, nil
				}
			}
			return nil, fmt.Errorf("author not found")
		}
	}
	return nil, fmt.Errorf("bulletin not found")
}

func (m *mockRepo) UpdatePlayerLastActive(ctx context.Context, playerID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for i, mb := range m.members {
		if mb.PlayerID == playerID {
			m.members[i].LastActiveAt = &now
			return nil
		}
	}
	return nil
}

func (m *mockRepo) ShareReport(ctx context.Context, allianceID, playerID, reportID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, sr := range m.sharedReports {
		if sr.AllianceID == allianceID && sr.ReportID == reportID {
			return fmt.Errorf("report already shared")
		}
	}

	m.sharedReports = append(m.sharedReports, SharedReport{
		ID:         m.nextSharedRpt,
		AllianceID: allianceID,
		ReportID:   reportID,
		SharedBy:   playerID,
		SharedAt:   time.Now(),
	})
	m.nextSharedRpt++
	return nil
}

func (m *mockRepo) GetSharedReports(ctx context.Context, allianceID int) ([]SharedReport, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []SharedReport
	for _, sr := range m.sharedReports {
		if sr.AllianceID == allianceID {
			result = append(result, sr)
		}
	}
	return result, nil
}

func (m *mockRepo) RemoveSharedReport(ctx context.Context, reportID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, sr := range m.sharedReports {
		if sr.ID == reportID {
			m.sharedReports = append(m.sharedReports[:i], m.sharedReports[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("shared report not found")
}

func (m *mockRepo) GetSharedReport(ctx context.Context, reportID int) (*SharedReport, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, sr := range m.sharedReports {
		if sr.ID == reportID {
			s := sr
			return &s, nil
		}
	}
	return nil, fmt.Errorf("shared report not found")
}
