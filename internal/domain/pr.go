package domain

import "time"

// Состояние пулл реквеста
type PRStatus string

const (
	PRStatusOpen   PRStatus = "OPEN"
	PRStatusMerged PRStatus = "MERGED"
)

// Структура пулл реквеста
type PullRequest struct {
	ID                string
	Name              string
	AuthorID          string
	Status            PRStatus
	AssignedReviewers []string
	NeedMoreReviewers bool
	CreatedAt         time.Time
	MergedAt          *time.Time
}

