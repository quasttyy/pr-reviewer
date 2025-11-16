package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/quasttyy/pr-reviewer/internal/service"
)

type PRHandlers struct {
	svc *service.PRService
}

func NewPRHandlers(svc *service.PRService) *PRHandlers {
	return &PRHandlers{svc: svc}
}

// POST /pullRequest/create
func (h *PRHandlers) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID   string `json:"pull_request_id"`
		Name string `json:"pull_request_name"`
		Auth string `json:"author_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == "" || req.Name == "" || req.Auth == "" {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request")
		return
	}
	pr, err := h.svc.Create(r.Context(), req.ID, req.Name, req.Auth)
	if err != nil {
		switch err {
		case service.ErrPRExists:
			writeError(w, http.StatusConflict, "PR_EXISTS", "PR id already exists")
			return
		case service.ErrNotFoundUser:
			writeError(w, http.StatusNotFound, "NOT_FOUND", "resource not found")
			return
		default:
			writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
			return
		}
	}
	type respPR struct {
		ID       string   `json:"pull_request_id"`
		Name     string   `json:"pull_request_name"`
		AuthorID string   `json:"author_id"`
		Status   string   `json:"status"`
		Reviewers []string `json:"assigned_reviewers"`
		CreatedAt *time.Time `json:"createdAt,omitempty"`
		MergedAt  *time.Time `json:"mergedAt,omitempty"`
	}
	resp := struct {
		PR respPR `json:"pr"`
	}{}
	resp.PR = respPR{
		ID:       pr.ID,
		Name:     pr.Name,
		AuthorID: pr.AuthorID,
		Status:   pr.Status,
		Reviewers: pr.Assigned,
		CreatedAt: pr.CreatedAt,
		MergedAt:  pr.MergedAt,
	}
	writeJSON(w, http.StatusCreated, resp)
}

// POST /pullRequest/merge
func (h *PRHandlers) Merge(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"pull_request_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == "" {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "pull_request_id is required")
		return
	}
	pr, err := h.svc.Merge(r.Context(), req.ID)
	if err != nil {
		if err == pgx.ErrNoRows || err == service.ErrNotFoundPR {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "resource not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	type respPR struct {
		ID       string   `json:"pull_request_id"`
		Name     string   `json:"pull_request_name"`
		AuthorID string   `json:"author_id"`
		Status   string   `json:"status"`
		Reviewers []string `json:"assigned_reviewers"`
		CreatedAt *time.Time `json:"createdAt,omitempty"`
		MergedAt  *time.Time `json:"mergedAt,omitempty"`
	}
	resp := struct {
		PR respPR `json:"pr"`
	}{}
	resp.PR = respPR{
		ID:       pr.ID,
		Name:     pr.Name,
		AuthorID: pr.AuthorID,
		Status:   pr.Status,
		Reviewers: pr.Assigned,
		CreatedAt: pr.CreatedAt,
		MergedAt:  pr.MergedAt,
	}
	writeJSON(w, http.StatusOK, resp)
}

// POST /pullRequest/reassign
func (h *PRHandlers) Reassign(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID  string `json:"pull_request_id"`
		Old string `json:"old_user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == "" || req.Old == "" {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "pull_request_id and old_user_id are required")
		return
	}
	pr, replacedBy, err := h.svc.Reassign(r.Context(), req.ID, req.Old)
	if err != nil {
		switch err {
		case service.ErrPRMerged:
			writeError(w, http.StatusConflict, "PR_MERGED", "cannot reassign on merged PR")
			return
		case service.ErrNotAssigned:
			writeError(w, http.StatusConflict, "NOT_ASSIGNED", "reviewer is not assigned to this PR")
			return
		case service.ErrNoCandidate:
			writeError(w, http.StatusConflict, "NO_CANDIDATE", "no active replacement candidate in team")
			return
		case service.ErrNotFoundPR, pgx.ErrNoRows:
			writeError(w, http.StatusNotFound, "NOT_FOUND", "resource not found")
			return
		default:
			writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
			return
		}
	}
	type respPR struct {
		ID       string   `json:"pull_request_id"`
		Name     string   `json:"pull_request_name"`
		AuthorID string   `json:"author_id"`
		Status   string   `json:"status"`
		Reviewers []string `json:"assigned_reviewers"`
	}
	resp := struct {
		PR         respPR `json:"pr"`
		ReplacedBy string `json:"replaced_by"`
	}{}
	resp.PR = respPR{
		ID:       pr.ID,
		Name:     pr.Name,
		AuthorID: pr.AuthorID,
		Status:   pr.Status,
		Reviewers: pr.Assigned,
	}
	resp.ReplacedBy = replacedBy
	writeJSON(w, http.StatusOK, resp)
}
