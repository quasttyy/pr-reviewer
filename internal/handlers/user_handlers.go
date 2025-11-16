package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/quasttyy/pr-reviewer/internal/service"
)

type UserHandlers struct {
	svc *service.UserService
}

func NewUserHandlers(svc *service.UserService) *UserHandlers {
	return &UserHandlers{svc: svc}
}

// POST /users/setIsActive (Admin)
func (h *UserHandlers) SetIsActive(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID   string `json:"user_id"`
		IsActive bool   `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.UserID == "" {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "user_id and is_active are required")
		return
	}
	row, err := h.svc.SetIsActiveAdmin(r.Context(), req.UserID, req.IsActive)
	if err != nil {
		if err == pgx.ErrNoRows {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "resource not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	resp := struct {
		User struct {
			UserID   string `json:"user_id"`
			Username string `json:"username"`
			TeamName string `json:"team_name"`
			IsActive bool   `json:"is_active"`
		} `json:"user"`
	}{}
	resp.User.UserID = row.UserID
	resp.User.Username = row.Username
	resp.User.TeamName = row.TeamName
	resp.User.IsActive = row.IsActive
	writeJSON(w, http.StatusOK, resp)
}

// GET /users/getReview?user_id=...
func (h *UserHandlers) GetReview(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "user_id is required")
		return
	}
	rows, err := h.svc.GetUserReviews(r.Context(), userID)
	if err != nil {
		if err == pgx.ErrNoRows {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "resource not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	type prShort struct {
		PullRequestID   string `json:"pull_request_id"`
		PullRequestName string `json:"pull_request_name"`
		AuthorID        string `json:"author_id"`
		Status          string `json:"status"`
	}
	resp := struct {
		UserID       string    `json:"user_id"`
		PullRequests []prShort `json:"pull_requests"`
	}{
		UserID: userID,
	}
	for _, p := range rows {
		resp.PullRequests = append(resp.PullRequests, prShort{
			PullRequestID:   p.ID,
			PullRequestName: p.Name,
			AuthorID:        p.AuthorID,
			Status:          p.Status,
		})
	}
	writeJSON(w, http.StatusOK, resp)
}