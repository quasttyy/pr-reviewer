package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/quasttyy/pr-reviewer/internal/domain"
	"github.com/quasttyy/pr-reviewer/internal/service"
)

type TeamHandlers struct {
	svc *service.TeamService
}

func NewTeamHandlers(svc *service.TeamService) *TeamHandlers {
	return &TeamHandlers{svc: svc}
}

// POST /team/add
func (h *TeamHandlers) AddTeam(w http.ResponseWriter, r *http.Request) {
	type memberDTO struct {
		UserID   string `json:"user_id"`
		Username string `json:"username"`
		IsActive bool   `json:"is_active"`
	}
	var req struct {
		TeamName string      `json:"team_name"`
		Members  []memberDTO `json:"members"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "NOT_FOUND", "invalid json")
		return
	}
	if req.TeamName == "" {
		writeError(w, http.StatusBadRequest, "NOT_FOUND", "team_name is required")
		return
	}
	members := make([]domain.TeamMember, 0, len(req.Members))
	for _, m := range req.Members {
		if m.UserID == "" || m.Username == "" {
			writeError(w, http.StatusBadRequest, "NOT_FOUND", "member.user_id and username are required")
			return
		}
		members = append(members, domain.TeamMember{
			ID:       m.UserID,
			Username: m.Username,
			IsActive: m.IsActive,
		})
	}
	err := h.svc.CreateTeam(r.Context(), domain.Team{
		Name:    req.TeamName,
		Members: members,
	})
	if err != nil {
		if err == service.ErrTeamExists {
			writeError(w, http.StatusBadRequest, "TEAM_EXISTS", "team_name already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	type memberResp struct {
		UserID   string `json:"user_id"`
		Username string `json:"username"`
		IsActive bool   `json:"is_active"`
	}
	resp := struct {
		Team struct {
			TeamName string       `json:"team_name"`
			Members  []memberResp `json:"members"`
		} `json:"team"`
	}{}
	resp.Team.TeamName = req.TeamName
	for _, m := range members {
		resp.Team.Members = append(resp.Team.Members, memberResp{
			UserID:   m.ID,
			Username: m.Username,
			IsActive: m.IsActive,
		})
	}
	writeJSON(w, http.StatusCreated, resp)
}

// GET /team/get?team_name=...
func (h *TeamHandlers) GetTeam(w http.ResponseWriter, r *http.Request) {
	teamName := r.URL.Query().Get("team_name")
	if teamName == "" {
		writeError(w, http.StatusBadRequest, "NOT_FOUND", "team_name is required")
		return
	}
	team, err := h.svc.GetTeam(r.Context(), teamName)
	if err != nil {
		if err == service.ErrNotFound {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "resource not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	type memberResp struct {
		UserID   string `json:"user_id"`
		Username string `json:"username"`
		IsActive bool   `json:"is_active"`
	}
	resp := struct {
		TeamName string       `json:"team_name"`
		Members  []memberResp `json:"members"`
	}{
		TeamName: team.Name,
	}
	for _, m := range team.Members {
		resp.Members = append(resp.Members, memberResp{
			UserID:   m.ID,
			Username: m.Username,
			IsActive: m.IsActive,
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	type errBody struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	var body errBody
	body.Error.Code = code
	body.Error.Message = message
	writeJSON(w, status, body)
}