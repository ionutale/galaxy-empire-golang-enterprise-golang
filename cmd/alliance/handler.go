package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	service *AllianceService
}

func NewHandler(service *AllianceService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) CreateAlliance(w http.ResponseWriter, r *http.Request) {
	playerID, ok := getUserID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req CreateAllianceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	alliance, err := h.service.CreateAlliance(r.Context(), playerID, req.Name, req.Tag)
	if err != nil {
		slog.Error("create alliance failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, AllianceResponse{
		ID:          alliance.ID,
		Name:        alliance.Name,
		Tag:         alliance.Tag,
		MemberCount: 1,
	})
}

func (h *Handler) ApplyToAlliance(w http.ResponseWriter, r *http.Request) {
	playerID, ok := getUserID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req ApplyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if req.AllianceID == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing alliance_id"})
		return
	}

	member, err := h.service.ApplyToAlliance(r.Context(), playerID, req.AllianceID)
	if err != nil {
		slog.Error("apply to alliance failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"alliance_id": member.AllianceID,
		"player_id":   member.PlayerID,
		"role":        member.Role,
	})
}

func (h *Handler) LeaveAlliance(w http.ResponseWriter, r *http.Request) {
	playerID, ok := getUserID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	if err := h.service.LeaveAlliance(r.Context(), playerID); err != nil {
		slog.Error("leave alliance failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) TransferFounder(w http.ResponseWriter, r *http.Request) {
	playerID, ok := getUserID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req TransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if req.TargetPlayerID == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing target_player_id"})
		return
	}

	if err := h.service.TransferFounder(r.Context(), playerID, req.TargetPlayerID); err != nil {
		slog.Error("transfer founder failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) GetMyAlliance(w http.ResponseWriter, r *http.Request) {
	playerID, ok := getUserID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	alliance, member, members, err := h.service.GetMyAlliance(r.Context(), playerID)
	if err != nil {
		slog.Error("get my alliance failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	memberResp := make([]MemberResponse, len(members))
	for i, m := range members {
		online := false
		var lastActiveStr string
		if m.LastActiveAt != nil {
			lastActiveStr = m.LastActiveAt.Format("2006-01-02T15:04:05Z")
			online = time.Since(*m.LastActiveAt) < 5*time.Minute
		}
		memberResp[i] = MemberResponse{
			PlayerID:     m.PlayerID,
			Role:         m.Role,
			JoinedAt:     m.JoinedAt.Format("2006-01-02T15:04:05Z"),
			Online:       online,
			LastActiveAt: lastActiveStr,
		}
	}

	writeJSON(w, http.StatusOK, AllianceResponse{
		ID:      alliance.ID,
		Name:    alliance.Name,
		Tag:     alliance.Tag,
		Role:    member.Role,
		Members: memberResp,
	})
}

func (h *Handler) BankDeposit(w http.ResponseWriter, r *http.Request) {
	playerID, ok := getUserID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req BankDepositRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if req.PlanetID == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing planet_id"})
		return
	}

	bank, err := h.service.BankDeposit(r.Context(), playerID, req.PlanetID, req.Metal, req.Crystal, req.Gas)
	if err != nil {
		slog.Error("bank deposit failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, BankResponse{
		Metal:   bank.Metal,
		Crystal: bank.Crystal,
		Gas:     bank.Gas,
	})
}

func (h *Handler) BankWithdraw(w http.ResponseWriter, r *http.Request) {
	playerID, ok := getUserID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req BankWithdrawRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if req.PlanetID == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing planet_id"})
		return
	}

	bank, err := h.service.BankWithdraw(r.Context(), playerID, req.PlanetID, req.Metal, req.Crystal, req.Gas)
	if err != nil {
		slog.Error("bank withdraw failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, BankResponse{
		Metal:   bank.Metal,
		Crystal: bank.Crystal,
		Gas:     bank.Gas,
	})
}

func (h *Handler) GetBank(w http.ResponseWriter, r *http.Request) {
	playerID, ok := getUserID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	bank, err := h.service.GetBank(r.Context(), playerID)
	if err != nil {
		slog.Error("get bank failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, BankResponse{
		Metal:   bank.Metal,
		Crystal: bank.Crystal,
		Gas:     bank.Gas,
	})
}

func (h *Handler) InternalGetPlayerAlliance(w http.ResponseWriter, r *http.Request) {
	var req InternalPlayerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	resp := h.service.GetPlayerAlliance(r.Context(), req.PlayerID)
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) PostBulletin(w http.ResponseWriter, r *http.Request) {
	playerID, ok := getUserID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req PostBulletinRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if req.Title == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing title"})
		return
	}

	bulletin, err := h.service.PostBulletin(r.Context(), playerID, req.Title, req.Content)
	if err != nil {
		slog.Error("post bulletin failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, bulletin)
}

func (h *Handler) GetBulletins(w http.ResponseWriter, r *http.Request) {
	playerID, ok := getUserID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	bulletins, err := h.service.GetBulletins(r.Context(), playerID)
	if err != nil {
		slog.Error("get bulletins failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, bulletins)
}

func (h *Handler) DeleteBulletin(w http.ResponseWriter, r *http.Request) {
	playerID, ok := getUserID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	bulletinIDStr := chi.URLParam(r, "id")
	bulletinID, err := strconv.Atoi(bulletinIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid bulletin id"})
		return
	}

	if err := h.service.DeleteBulletin(r.Context(), bulletinID, playerID); err != nil {
		slog.Error("delete bulletin failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) ShareReport(w http.ResponseWriter, r *http.Request) {
	playerID, ok := getUserID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req ShareReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if req.ReportID == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing report_id"})
		return
	}

	if err := h.service.ShareReport(r.Context(), playerID, req.ReportID); err != nil {
		slog.Error("share report failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) GetSharedReports(w http.ResponseWriter, r *http.Request) {
	playerID, ok := getUserID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	reports, err := h.service.GetSharedReports(r.Context(), playerID)
	if err != nil {
		slog.Error("get shared reports failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, reports)
}

func (h *Handler) UnshareReport(w http.ResponseWriter, r *http.Request) {
	playerID, ok := getUserID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req UnshareReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if req.ReportID == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing report_id"})
		return
	}

	if err := h.service.UnshareReport(r.Context(), req.ReportID, playerID); err != nil {
		slog.Error("unshare report failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) InternalPing(w http.ResponseWriter, r *http.Request) {
	var req InternalPlayerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if err := h.service.PingMember(r.Context(), req.PlayerID); err != nil {
		slog.Error("ping member failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func getUserID(r *http.Request) (int, bool) {
	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		return 0, false
	}
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return 0, false
	}
	return userID, true
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
