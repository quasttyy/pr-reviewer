package service

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/quasttyy/pr-reviewer/internal/repo"
)

// fakePRRepo — простая in-memory реализация, покрывающая методы, которые вызывает PRService.
type fakePRRepo struct {
	usersTeam     map[string]string              // user_id -> team_name
	activeInTeam  map[string]map[string]bool     // team_name -> user_id -> isActive
	prs           map[string]repo.PRFull         // pr_id -> PR
	prReviewers   map[string]map[string]struct{} // pr_id -> set(reviewer_id)
	createdAtTime time.Time
}

func newFakePRRepo() *fakePRRepo {
	return &fakePRRepo{
		usersTeam:    make(map[string]string),
		activeInTeam: make(map[string]map[string]bool),
		prs:          make(map[string]repo.PRFull),
		prReviewers:  make(map[string]map[string]struct{}),
		createdAtTime: time.Now().UTC().Truncate(time.Second),
	}
}

func (f *fakePRRepo) GetShortByReviewer(ctx context.Context, userID string) ([]repo.PRShortRow, error) {
	var out []repo.PRShortRow
	for prID, prs := range f.prs {
		if f.prReviewers[prID] != nil {
			if _, ok := f.prReviewers[prID][userID]; ok {
				out = append(out, repo.PRShortRow{ID: prID, Name: prs.Name, AuthorID: prs.AuthorID, Status: prs.Status})
			}
		}
	}
	return out, nil
}

func (f *fakePRRepo) CreatePROpenWithAssigned(ctx context.Context, id, name, author string, needMore bool, reviewers []string) error {
	if _, exists := f.prs[id]; exists {
		return errors.New("duplicate")
	}
	created := f.createdAtTime
	f.prs[id] = repo.PRFull{
		ID:                id,
		Name:              name,
		AuthorID:          author,
		Status:            "OPEN",
		NeedMoreReviewers: needMore,
		CreatedAt:         &created,
	}
	f.prReviewers[id] = make(map[string]struct{})
	for _, r := range reviewers {
		f.prReviewers[id][r] = struct{}{}
	}
	return nil
}

func (f *fakePRRepo) GetPR(ctx context.Context, id string) (repo.PRFull, error) {
	pr, ok := f.prs[id]
	if !ok {
		return repo.PRFull{}, pgx.ErrNoRows
	}
	var assigned []string
	for rv := range f.prReviewers[id] {
		assigned = append(assigned, rv)
	}
	pr.Assigned = assigned
	return pr, nil
}

func (f *fakePRRepo) MarkMerged(ctx context.Context, id string) error {
	pr, ok := f.prs[id]
	if !ok {
		return pgx.ErrNoRows
	}
	pr.Status = "MERGED"
	if pr.MergedAt == nil {
		now := time.Now().UTC().Truncate(time.Second)
		pr.MergedAt = &now
	}
	f.prs[id] = pr
	return nil
}

func (f *fakePRRepo) GetUserTeam(ctx context.Context, userID string) (string, error) {
	tm, ok := f.usersTeam[userID]
	if !ok {
		return "", pgx.ErrNoRows
	}
	return tm, nil
}

func (f *fakePRRepo) GetActiveCandidatesFromTeamExcluding(ctx context.Context, teamName, excludeUserID string) ([]string, error) {
	m := f.activeInTeam[teamName]
	var out []string
	for id, active := range m {
		if active && id != excludeUserID {
			out = append(out, id)
		}
	}
	return out, nil
}

func (f *fakePRRepo) ReplaceReviewer(ctx context.Context, prID, oldReviewer, newReviewer string) error {
	set := f.prReviewers[prID]
	if set == nil {
		return pgx.ErrNoRows
	}
	if _, ok := set[oldReviewer]; !ok {
		return errors.New("not assigned")
	}
	delete(set, oldReviewer)
	set[newReviewer] = struct{}{}
	return nil
}

// GetReviewerStats реализует StatsStore для тестов, просто считает количество назначений по in-memory структурам.
func (f *fakePRRepo) GetReviewerStats(ctx context.Context) ([]repo.ReviewerStatRow, error) {
	counts := make(map[string]int64)
	for _, reviewers := range f.prReviewers {
		for id := range reviewers {
			counts[id]++
		}
	}
	var out []repo.ReviewerStatRow
	for id, c := range counts {
		out = append(out, repo.ReviewerStatRow{
			ReviewerID:    id,
			TotalAssigned: c,
		})
	}
	return out, nil
}

func TestCreate_AssignsUpToTwo(t *testing.T) {
	r := newFakePRRepo()
	// команда backend: u1 (author), u2, u3 активны
	r.usersTeam["u1"] = "backend"
	r.usersTeam["u2"] = "backend"
	r.usersTeam["u3"] = "backend"
	r.activeInTeam["backend"] = map[string]bool{"u1": true, "u2": true, "u3": true}

	svc := NewPRService(r)
	pr, err := svc.Create(context.Background(), "pr-1", "T", "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pr.Status != "OPEN" {
		t.Fatalf("status want OPEN, got %s", pr.Status)
	}
	if len(pr.Assigned) == 0 || len(pr.Assigned) > 2 {
		t.Fatalf("assigned reviewers count must be 1..2, got %d", len(pr.Assigned))
	}
	// не должен включать автора
	for _, id := range pr.Assigned {
		if id == "u1" {
			t.Fatalf("author must not be assigned as reviewer")
		}
	}
}

func TestMerge_Idempotent(t *testing.T) {
	r := newFakePRRepo()
	r.usersTeam["u1"] = "backend"
	r.activeInTeam["backend"] = map[string]bool{"u1": true}
	svc := NewPRService(r)
	if _, err := svc.Create(context.Background(), "pr-2", "X", "u1"); err != nil {
		t.Fatalf("create: %v", err)
	}
	pr1, err := svc.Merge(context.Background(), "pr-2")
	if err != nil {
		t.Fatalf("merge1: %v", err)
	}
	time1 := pr1.MergedAt
	pr2, err := svc.Merge(context.Background(), "pr-2")
	if err != nil {
		t.Fatalf("merge2: %v", err)
	}
	if pr2.Status != "MERGED" {
		t.Fatalf("want MERGED")
	}
	if !reflect.DeepEqual(time1, pr2.MergedAt) {
		t.Fatalf("mergedAt must be stable (idempotent)")
	}
}

func TestReassign_Basic(t *testing.T) {
	r := newFakePRRepo()
	// team A: u1 author, u2,u3 active
	for _, u := range []string{"u1", "u2", "u3"} {
		r.usersTeam[u] = "A"
	}
	r.activeInTeam["A"] = map[string]bool{"u1": true, "u2": true, "u3": true}
	svc := NewPRService(r)
	pr, err := svc.Create(context.Background(), "pr-3", "Feat", "u1")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	var old string
	if len(pr.Assigned) == 0 {
		t.Fatalf("expected at least one reviewer")
	}
	old = pr.Assigned[0]
	pr2, replacedBy, err := svc.Reassign(context.Background(), "pr-3", old)
	if err != nil {
		t.Fatalf("reassign: %v", err)
	}
	// old больше не должен быть в списке
	for _, id := range pr2.Assigned {
		if id == old {
			t.Fatalf("old reviewer still assigned")
		}
	}
	found := false
	for _, id := range pr2.Assigned {
		if id == replacedBy {
			found = true
		}
	}
	if !found {
		t.Fatalf("replacedBy not in assigned list")
	}
}


