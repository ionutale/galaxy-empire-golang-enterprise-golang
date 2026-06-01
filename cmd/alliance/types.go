package main

import "time"

type Alliance struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Tag       string    `json:"tag"`
	FounderID int       `json:"founder_id"`
	CreatedAt time.Time `json:"created_at"`
}

type Member struct {
	ID           int        `json:"id"`
	AllianceID   int        `json:"alliance_id"`
	PlayerID     int        `json:"player_id"`
	Role         string     `json:"role"`
	JoinedAt     time.Time  `json:"joined_at"`
	LastActiveAt *time.Time `json:"last_active_at,omitempty"`
}

type Bulletin struct {
	ID             int       `json:"id"`
	AllianceID     int       `json:"alliance_id"`
	AuthorPlayerID int       `json:"author_player_id"`
	Title          string    `json:"title"`
	Content        string    `json:"content"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type SharedReport struct {
	ID         int       `json:"id"`
	AllianceID int       `json:"alliance_id"`
	ReportID   int       `json:"report_id"`
	SharedBy   int       `json:"shared_by"`
	SharedAt   time.Time `json:"shared_at"`
}

type Bank struct {
	AllianceID int `json:"alliance_id"`
	Metal      int `json:"metal"`
	Crystal    int `json:"crystal"`
	Gas        int `json:"gas"`
}

type AuditEntry struct {
	ID         int       `json:"id"`
	AllianceID int       `json:"alliance_id"`
	PlayerID   int       `json:"player_id"`
	Action     string    `json:"action"`
	Details    map[string]any `json:"details,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type CreateAllianceRequest struct {
	Name string `json:"name"`
	Tag  string `json:"tag"`
}

type ApplyRequest struct {
	AllianceID int `json:"alliance_id"`
}

type TransferRequest struct {
	TargetPlayerID int `json:"target_player_id"`
}

type BankDepositRequest struct {
	PlanetID int `json:"planet_id"`
	Metal    int `json:"metal"`
	Crystal  int `json:"crystal"`
	Gas      int `json:"gas"`
}

type BankWithdrawRequest struct {
	PlanetID int `json:"planet_id"`
	Metal    int `json:"metal"`
	Crystal  int `json:"crystal"`
	Gas      int `json:"gas"`
}

type AllianceResponse struct {
	ID          int              `json:"id"`
	Name        string           `json:"name"`
	Tag         string           `json:"tag"`
	Role        string           `json:"role,omitempty"`
	MemberCount int              `json:"member_count,omitempty"`
	Members     []MemberResponse `json:"members,omitempty"`
}

type MemberResponse struct {
	PlayerID     int    `json:"player_id"`
	Role         string `json:"role"`
	JoinedAt     string `json:"joined_at"`
	Online       bool   `json:"online"`
	LastActiveAt string `json:"last_active_at,omitempty"`
}

type BankResponse struct {
	Metal   int `json:"metal"`
	Crystal int `json:"crystal"`
	Gas     int `json:"gas"`
}

type PlayerAllianceResponse struct {
	InAlliance   bool   `json:"in_alliance"`
	AllianceID   int    `json:"alliance_id,omitempty"`
	Role         string `json:"role,omitempty"`
	AllianceName string `json:"alliance_name,omitempty"`
	AllianceTag  string `json:"alliance_tag,omitempty"`
}

type InternalPlayerRequest struct {
	PlayerID int `json:"player_id"`
}

type PostBulletinRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

type ShareReportRequest struct {
	ReportID int `json:"report_id"`
}

type UnshareReportRequest struct {
	ReportID int `json:"report_id"`
}
