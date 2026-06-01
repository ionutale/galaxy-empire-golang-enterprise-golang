package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var (
	validNameRegex = regexp.MustCompile(`^[a-zA-Z0-9 ]+$`)
	validTagRegex  = regexp.MustCompile(`^[A-Z0-9]+$`)
)

type AllianceService struct {
	repo          Repository
	planetBaseURL string
	httpClient    *http.Client
}

func NewAllianceService(repo Repository, planetBaseURL string) *AllianceService {
	return &AllianceService{
		repo:          repo,
		planetBaseURL: planetBaseURL,
		httpClient:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *AllianceService) CreateAlliance(ctx context.Context, playerID int, name, tag string) (Alliance, error) {
	name = strings.TrimSpace(name)
	tag = strings.TrimSpace(tag)

	if len(name) < 3 || len(name) > 50 {
		return Alliance{}, fmt.Errorf("alliance name must be 3-50 characters")
	}
	if !validNameRegex.MatchString(name) {
		return Alliance{}, fmt.Errorf("alliance name can only contain letters, numbers, and spaces")
	}
	if len(tag) < 2 || len(tag) > 10 {
		return Alliance{}, fmt.Errorf("alliance tag must be 2-10 characters")
	}
	if !validTagRegex.MatchString(tag) {
		return Alliance{}, fmt.Errorf("alliance tag can only contain uppercase letters and numbers")
	}

	existing, err := s.repo.GetMember(ctx, playerID)
	if err != nil {
		return Alliance{}, fmt.Errorf("check membership: %w", err)
	}
	if existing != nil {
		return Alliance{}, fmt.Errorf("you are already in an alliance")
	}

	existingByName, err := s.repo.GetAllianceByName(ctx, name)
	if err != nil {
		return Alliance{}, fmt.Errorf("check name: %w", err)
	}
	if existingByName != nil {
		return Alliance{}, fmt.Errorf("alliance name already taken")
	}

	existingByTag, err := s.repo.GetAllianceByTag(ctx, tag)
	if err != nil {
		return Alliance{}, fmt.Errorf("check tag: %w", err)
	}
	if existingByTag != nil {
		return Alliance{}, fmt.Errorf("alliance tag already taken")
	}

	alliance, err := s.repo.CreateAlliance(ctx, name, tag, playerID)
	if err != nil {
		return Alliance{}, fmt.Errorf("create alliance: %w", err)
	}

	if _, err := s.repo.AddMember(ctx, alliance.ID, playerID, "founder"); err != nil {
		return Alliance{}, fmt.Errorf("add founder: %w", err)
	}

	if err := s.repo.UpdateBank(ctx, alliance.ID, 0, 0, 0); err != nil {
		return Alliance{}, fmt.Errorf("create bank: %w", err)
	}

	if err := s.repo.AddAuditLog(ctx, alliance.ID, playerID, "alliance_created", nil); err != nil {
		slog.Error("audit log failed", "error", err)
	}

	return alliance, nil
}

func (s *AllianceService) ApplyToAlliance(ctx context.Context, playerID, allianceID int) (Member, error) {
	alliance, err := s.repo.GetAlliance(ctx, allianceID)
	if err != nil {
		return Member{}, fmt.Errorf("alliance not found")
	}

	existing, err := s.repo.GetMember(ctx, playerID)
	if err != nil {
		return Member{}, fmt.Errorf("check membership: %w", err)
	}
	if existing != nil {
		return Member{}, fmt.Errorf("you are already in an alliance")
	}

	member, err := s.repo.AddMember(ctx, alliance.ID, playerID, "pending")
	if err != nil {
		return Member{}, fmt.Errorf("apply to alliance: %w", err)
	}

	if err := s.repo.AddAuditLog(ctx, alliance.ID, playerID, "member_joined", nil); err != nil {
		slog.Error("audit log failed", "error", err)
	}

	return member, nil
}

func (s *AllianceService) LeaveAlliance(ctx context.Context, playerID int) error {
	member, err := s.repo.GetMember(ctx, playerID)
	if err != nil {
		return fmt.Errorf("check membership: %w", err)
	}
	if member == nil {
		return fmt.Errorf("you are not in an alliance")
	}

	if member.Role == "founder" {
		return fmt.Errorf("founder cannot leave, transfer ownership first")
	}

	allianceID := member.AllianceID

	if err := s.repo.RemoveMember(ctx, playerID); err != nil {
		return fmt.Errorf("leave alliance: %w", err)
	}

	if err := s.repo.AddAuditLog(ctx, allianceID, playerID, "member_left", nil); err != nil {
		slog.Error("audit log failed", "error", err)
	}

	return nil
}

func (s *AllianceService) TransferFounder(ctx context.Context, playerID, targetPlayerID int) error {
	member, err := s.repo.GetMember(ctx, playerID)
	if err != nil {
		return fmt.Errorf("check membership: %w", err)
	}
	if member == nil {
		return fmt.Errorf("you are not in an alliance")
	}
	if member.Role != "founder" {
		return fmt.Errorf("only the founder can transfer ownership")
	}

	target, err := s.repo.GetMember(ctx, targetPlayerID)
	if err != nil {
		return fmt.Errorf("check target: %w", err)
	}
	if target == nil {
		return fmt.Errorf("target player is not in any alliance")
	}
	if target.AllianceID != member.AllianceID {
		return fmt.Errorf("target player is not in your alliance")
	}

	if err := s.repo.UpdateMemberRole(ctx, member.AllianceID, playerID, "officer"); err != nil {
		return fmt.Errorf("demote founder: %w", err)
	}

	if err := s.repo.UpdateMemberRole(ctx, member.AllianceID, targetPlayerID, "founder"); err != nil {
		return fmt.Errorf("promote target: %w", err)
	}

	if err := s.repo.AddAuditLog(ctx, member.AllianceID, playerID, "founder_transferred", map[string]any{
		"from_player_id": playerID,
		"to_player_id":   targetPlayerID,
	}); err != nil {
		slog.Error("audit log failed", "error", err)
	}

	return nil
}

func (s *AllianceService) GetMyAlliance(ctx context.Context, playerID int) (*Alliance, *Member, []Member, error) {
	member, err := s.repo.GetMember(ctx, playerID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("check membership: %w", err)
	}
	if member == nil {
		return nil, nil, nil, fmt.Errorf("you are not in an alliance")
	}

	alliance, err := s.repo.GetAlliance(ctx, member.AllianceID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("get alliance: %w", err)
	}

	members, err := s.repo.GetMembers(ctx, alliance.ID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("get members: %w", err)
	}

	return alliance, member, members, nil
}

func (s *AllianceService) BankDeposit(ctx context.Context, playerID, planetID int, metal, crystal, gas int) (Bank, error) {
	if metal < 0 || crystal < 0 || gas < 0 {
		return Bank{}, fmt.Errorf("deposit amounts cannot be negative")
	}
	if metal == 0 && crystal == 0 && gas == 0 {
		return Bank{}, fmt.Errorf("deposit amounts cannot all be zero")
	}

	member, err := s.repo.GetMember(ctx, playerID)
	if err != nil {
		return Bank{}, fmt.Errorf("check membership: %w", err)
	}
	if member == nil {
		return Bank{}, fmt.Errorf("you are not in an alliance")
	}

	if metal > 0 {
		if err := s.deductResource(ctx, planetID, "metal", metal); err != nil {
			return Bank{}, fmt.Errorf("deduct metal: %w", err)
		}
	}
	if crystal > 0 {
		if err := s.deductResource(ctx, planetID, "crystal", crystal); err != nil {
			return Bank{}, fmt.Errorf("deduct crystal: %w", err)
		}
	}
	if gas > 0 {
		if err := s.deductResource(ctx, planetID, "gas", gas); err != nil {
			return Bank{}, fmt.Errorf("deduct gas: %w", err)
		}
	}

	bank, err := s.repo.GetBank(ctx, member.AllianceID)
	if err != nil {
		return Bank{}, fmt.Errorf("get bank: %w", err)
	}
	if bank == nil {
		return Bank{}, fmt.Errorf("bank not found")
	}

	newMetal := bank.Metal + metal
	newCrystal := bank.Crystal + crystal
	newGas := bank.Gas + gas

	if err := s.repo.UpdateBank(ctx, member.AllianceID, newMetal, newCrystal, newGas); err != nil {
		return Bank{}, fmt.Errorf("update bank: %w", err)
	}

	if err := s.repo.AddAuditLog(ctx, member.AllianceID, playerID, "bank_deposit", map[string]any{
		"metal":   metal,
		"crystal": crystal,
		"gas":     gas,
	}); err != nil {
		slog.Error("audit log failed", "error", err)
	}

	return Bank{AllianceID: member.AllianceID, Metal: newMetal, Crystal: newCrystal, Gas: newGas}, nil
}

func (s *AllianceService) BankWithdraw(ctx context.Context, playerID, planetID int, metal, crystal, gas int) (Bank, error) {
	member, err := s.repo.GetMember(ctx, playerID)
	if err != nil {
		return Bank{}, fmt.Errorf("check membership: %w", err)
	}
	if member == nil {
		return Bank{}, fmt.Errorf("you are not in an alliance")
	}

	if member.Role != "founder" && member.Role != "officer" {
		return Bank{}, fmt.Errorf("only officers and founders can withdraw from the bank")
	}

	bank, err := s.repo.GetBank(ctx, member.AllianceID)
	if err != nil {
		return Bank{}, fmt.Errorf("get bank: %w", err)
	}
	if bank == nil {
		return Bank{}, fmt.Errorf("bank not found")
	}

	if bank.Metal < metal {
		return Bank{}, fmt.Errorf("insufficient metal in bank: have %d, need %d", bank.Metal, metal)
	}
	if bank.Crystal < crystal {
		return Bank{}, fmt.Errorf("insufficient crystal in bank: have %d, need %d", bank.Crystal, crystal)
	}
	if bank.Gas < gas {
		return Bank{}, fmt.Errorf("insufficient gas in bank: have %d, need %d", bank.Gas, gas)
	}

	newMetal := bank.Metal - metal
	newCrystal := bank.Crystal - crystal
	newGas := bank.Gas - gas

	if err := s.repo.UpdateBank(ctx, member.AllianceID, newMetal, newCrystal, newGas); err != nil {
		return Bank{}, fmt.Errorf("update bank: %w", err)
	}

	if metal > 0 {
		if err := s.addResource(ctx, planetID, "metal", metal); err != nil {
			slog.Error("add metal to player failed", "error", err)
		}
	}
	if crystal > 0 {
		if err := s.addResource(ctx, planetID, "crystal", crystal); err != nil {
			slog.Error("add crystal to player failed", "error", err)
		}
	}
	if gas > 0 {
		if err := s.addResource(ctx, planetID, "gas", gas); err != nil {
			slog.Error("add gas to player failed", "error", err)
		}
	}

	if err := s.repo.AddAuditLog(ctx, member.AllianceID, playerID, "bank_withdraw", map[string]any{
		"metal":   metal,
		"crystal": crystal,
		"gas":     gas,
	}); err != nil {
		slog.Error("audit log failed", "error", err)
	}

	return Bank{AllianceID: member.AllianceID, Metal: newMetal, Crystal: newCrystal, Gas: newGas}, nil
}

func (s *AllianceService) GetBank(ctx context.Context, playerID int) (Bank, error) {
	member, err := s.repo.GetMember(ctx, playerID)
	if err != nil {
		return Bank{}, fmt.Errorf("check membership: %w", err)
	}
	if member == nil {
		return Bank{}, fmt.Errorf("you are not in an alliance")
	}

	bank, err := s.repo.GetBank(ctx, member.AllianceID)
	if err != nil {
		return Bank{}, fmt.Errorf("get bank: %w", err)
	}
	if bank == nil {
		return Bank{}, fmt.Errorf("bank not found")
	}

	return *bank, nil
}

func (s *AllianceService) GetPlayerAlliance(ctx context.Context, playerID int) PlayerAllianceResponse {
	member, err := s.repo.GetMember(ctx, playerID)
	if err != nil || member == nil {
		return PlayerAllianceResponse{InAlliance: false}
	}

	alliance, err := s.repo.GetAlliance(ctx, member.AllianceID)
	if err != nil || alliance == nil {
		return PlayerAllianceResponse{InAlliance: false}
	}

	return PlayerAllianceResponse{
		InAlliance:   true,
		AllianceID:   alliance.ID,
		Role:         member.Role,
		AllianceName: alliance.Name,
		AllianceTag:  alliance.Tag,
	}
}

func (s *AllianceService) deductResource(ctx context.Context, planetID int, resource string, amount int) error {
	body, _ := json.Marshal(map[string]any{
		"planet_id": planetID,
		"resource":  resource,
		"amount":    amount,
	})
	resp, err := s.httpClient.Post(s.planetBaseURL+"/internal/resources/deduct", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("planet service unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("planet service: %s", string(respBody))
	}
	return nil
}

func (s *AllianceService) PostBulletin(ctx context.Context, playerID int, title, content string) (Bulletin, error) {
	member, err := s.repo.GetMember(ctx, playerID)
	if err != nil {
		return Bulletin{}, fmt.Errorf("check membership: %w", err)
	}
	if member == nil {
		return Bulletin{}, fmt.Errorf("you are not in an alliance")
	}

	if member.Role != "founder" && member.Role != "officer" {
		return Bulletin{}, fmt.Errorf("only founders and officers can post bulletins")
	}

	bulletin, err := s.repo.PostBulletin(ctx, member.AllianceID, playerID, title, content)
	if err != nil {
		return Bulletin{}, fmt.Errorf("post bulletin: %w", err)
	}

	return bulletin, nil
}

func (s *AllianceService) GetBulletins(ctx context.Context, playerID int) ([]Bulletin, error) {
	member, err := s.repo.GetMember(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if member == nil {
		return nil, fmt.Errorf("you are not in an alliance")
	}

	bulletins, err := s.repo.GetBulletins(ctx, member.AllianceID)
	if err != nil {
		return nil, fmt.Errorf("get bulletins: %w", err)
	}

	return bulletins, nil
}

func (s *AllianceService) DeleteBulletin(ctx context.Context, bulletinID, playerID int) error {
	authorMember, err := s.repo.GetMemberBulletin(ctx, bulletinID)
	if err != nil {
		return fmt.Errorf("get bulletin: %w", err)
	}

	requesterMember, err := s.repo.GetMember(ctx, playerID)
	if err != nil {
		return fmt.Errorf("check membership: %w", err)
	}
	if requesterMember == nil {
		return fmt.Errorf("you are not in an alliance")
	}

	isAuthor := requesterMember.PlayerID == authorMember.PlayerID
	isOfficerOrFounder := requesterMember.Role == "founder" || requesterMember.Role == "officer"

	if !isAuthor && !isOfficerOrFounder {
		return fmt.Errorf("you are not authorized to delete this bulletin")
	}

	if err := s.repo.DeleteBulletin(ctx, bulletinID); err != nil {
		return fmt.Errorf("delete bulletin: %w", err)
	}

	return nil
}

func (s *AllianceService) PingMember(ctx context.Context, playerID int) error {
	return s.repo.UpdatePlayerLastActive(ctx, playerID)
}

func (s *AllianceService) ShareReport(ctx context.Context, playerID, reportID int) error {
	member, err := s.repo.GetMember(ctx, playerID)
	if err != nil {
		return fmt.Errorf("check membership: %w", err)
	}
	if member == nil {
		return fmt.Errorf("you are not in an alliance")
	}

	if err := s.repo.ShareReport(ctx, member.AllianceID, playerID, reportID); err != nil {
		return fmt.Errorf("share report: %w", err)
	}

	return nil
}

func (s *AllianceService) GetSharedReports(ctx context.Context, playerID int) ([]SharedReport, error) {
	member, err := s.repo.GetMember(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if member == nil {
		return nil, fmt.Errorf("you are not in an alliance")
	}

	reports, err := s.repo.GetSharedReports(ctx, member.AllianceID)
	if err != nil {
		return nil, fmt.Errorf("get shared reports: %w", err)
	}

	return reports, nil
}

func (s *AllianceService) UnshareReport(ctx context.Context, reportID, playerID int) error {
	report, err := s.repo.GetSharedReport(ctx, reportID)
	if err != nil {
		return fmt.Errorf("get report: %w", err)
	}

	member, err := s.repo.GetMember(ctx, playerID)
	if err != nil {
		return fmt.Errorf("check membership: %w", err)
	}
	if member == nil {
		return fmt.Errorf("you are not in an alliance")
	}

	isOriginalSharer := report.SharedBy == playerID
	isOfficerOrFounder := member.Role == "founder" || member.Role == "officer"

	if !isOriginalSharer && !isOfficerOrFounder {
		return fmt.Errorf("you are not authorized to unshare this report")
	}

	if err := s.repo.RemoveSharedReport(ctx, reportID); err != nil {
		return fmt.Errorf("unshare report: %w", err)
	}

	return nil
}

func (s *AllianceService) addResource(ctx context.Context, planetID int, resource string, amount int) error {
	body, _ := json.Marshal(map[string]any{
		"planet_id": planetID,
		"resource":  resource,
		"amount":    amount,
	})
	resp, err := s.httpClient.Post(s.planetBaseURL+"/internal/resources/add", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("planet service unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("planet service: %s", string(respBody))
	}
	return nil
}
