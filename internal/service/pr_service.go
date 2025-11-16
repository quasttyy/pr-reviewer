package service

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/quasttyy/pr-reviewer/internal/repo"
)

var (
	ErrPRExists     = errors.New("pr exists")
	ErrPRMerged     = errors.New("pr merged")
	ErrNotAssigned  = errors.New("not assigned")
	ErrNoCandidate  = errors.New("no candidate")
	ErrNotFoundPR   = errors.New("pr not found")
	ErrNotFoundUser = errors.New("user not found")
)

type PRService struct {
	prs PRStore
}

type PRStore interface {
	GetUserTeam(ctx context.Context, userID string) (string, error)
	GetActiveCandidatesFromTeamExcluding(ctx context.Context, teamName, excludeUserID string) ([]string, error)
	CreatePROpenWithAssigned(ctx context.Context, id, name, author string, needMore bool, reviewers []string) error
	GetPR(ctx context.Context, id string) (repo.PRFull, error)
	MarkMerged(ctx context.Context, id string) error
	ReplaceReviewer(ctx context.Context, prID, oldReviewer, newReviewer string) error
	GetReviewerStats(ctx context.Context) ([]repo.ReviewerStatRow, error)
}

func NewPRService(prs PRStore) *PRService {
	return &PRService{prs: prs}
}

// Create назначает до двух активных ревьюеров из команды автора (кроме автора)
func (s *PRService) Create(ctx context.Context, prID, prName, authorID string) (repo.PRFull, error) {
	// найдём команду автора
	team, err := s.prs.GetUserTeam(ctx, authorID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return repo.PRFull{}, ErrNotFoundUser
		}
		return repo.PRFull{}, err
	}
	candidates, err := s.prs.GetActiveCandidatesFromTeamExcluding(ctx, team, authorID)
	if err != nil {
		return repo.PRFull{}, err
	}
	reviewers := chooseUpToTwoRandom(candidates)
	needMore := len(reviewers) < 2
	if err := s.prs.CreatePROpenWithAssigned(ctx, prID, prName, authorID, needMore, reviewers); err != nil {
		return repo.PRFull{}, ErrPRExists
	}
	return s.prs.GetPR(ctx, prID)
}

// Merge идемпотентно помечает PR как MERGED
func (s *PRService) Merge(ctx context.Context, prID string) (repo.PRFull, error) {
	pr, err := s.prs.GetPR(ctx, prID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return repo.PRFull{}, ErrNotFoundPR
		}
		return repo.PRFull{}, err
	}
	if pr.Status == "MERGED" {
		return pr, nil
	}
	if err := s.prs.MarkMerged(ctx, prID); err != nil {
		return repo.PRFull{}, err
	}
	return s.prs.GetPR(ctx, prID)
}

// Reassign заменяет одного ревьювера на случайного активного из его команды
func (s *PRService) Reassign(ctx context.Context, prID, oldReviewerID string) (repo.PRFull, string, error) {
	pr, err := s.prs.GetPR(ctx, prID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return repo.PRFull{}, "", ErrNotFoundPR
		}
		return repo.PRFull{}, "", err
	}
	if pr.Status == "MERGED" {
		return repo.PRFull{}, "", ErrPRMerged
	}
	// Убедимся, что oldReviewer назначен
	found := false
	for _, r := range pr.Assigned {
		if r == oldReviewerID {
			found = true
			break
		}
	}
	if !found {
		return repo.PRFull{}, "", ErrNotAssigned
	}
	// Кандидаты из команды oldReviewer
	team, err := s.prs.GetUserTeam(ctx, oldReviewerID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return repo.PRFull{}, "", ErrNotFoundUser
		}
		return repo.PRFull{}, "", err
	}
	candidates, err := s.prs.GetActiveCandidatesFromTeamExcluding(ctx, team, oldReviewerID)
	if err != nil {
		return repo.PRFull{}, "", err
	}
	// Убираем тех, кто уже назначен
	exclude := map[string]struct{}{}
	for _, r := range pr.Assigned {
		exclude[r] = struct{}{}
	}
	var pool []string
	for _, id := range candidates {
		if _, ok := exclude[id]; !ok {
			pool = append(pool, id)
		}
	}
	if len(pool) == 0 {
		return repo.PRFull{}, "", ErrNoCandidate
	}
	newReviewer := pool[rand.Intn(len(pool))]
	if err := s.prs.ReplaceReviewer(ctx, prID, oldReviewerID, newReviewer); err != nil {
		if err.Error() == "not assigned" {
			return repo.PRFull{}, "", ErrNotAssigned
		}
		return repo.PRFull{}, "", err
	}
	pr2, err := s.prs.GetPR(ctx, prID)
	return pr2, newReviewer, err
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func chooseUpToTwoRandom(ids []string) []string {
	if len(ids) <= 2 {
		return append([]string(nil), ids...)
	}
	// Выбираем два случайных неповторяющихся
	i := rand.Intn(len(ids))
	j := rand.Intn(len(ids)-1)
	if j >= i {
		j++
	}
	if i > j {
		i, j = j, i
	}
	return []string{ids[i], ids[j]}
}

// GetReviewerStats проксирует статистику назначений ревьюверов из репозитория.
func (s *PRService) GetReviewerStats(ctx context.Context) ([]repo.ReviewerStatRow, error) {
	return s.prs.GetReviewerStats(ctx)
}
