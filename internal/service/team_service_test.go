package service

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/quasttyy/pr-reviewer/internal/domain"
	"github.com/quasttyy/pr-reviewer/internal/repo"
)

// fakeTeamStore — простая in-memory реализация TeamStore для юнит-тестов TeamService
type fakeTeamStore struct {
	existingTeams map[string]bool
	membersByTeam map[string][]repo.TeamMemberRow

	// Настройки поведения
	errOnExistsCheck       error
	errOnCreateWithMembers error
	errOnGetWithMembers    error

	// Для проверки входных параметров
	lastCreatedTeamName string
	lastCreatedMembers  []struct {
		UserID   string
		Username string
		IsActive bool
	}
}

func newFakeTeamStore() *fakeTeamStore {
	return &fakeTeamStore{
		existingTeams: make(map[string]bool),
		membersByTeam: make(map[string][]repo.TeamMemberRow),
	}
}

func (f *fakeTeamStore) TeamExists(ctx context.Context, teamName string) (bool, error) {
	if f.errOnExistsCheck != nil {
		return false, f.errOnExistsCheck
	}
	return f.existingTeams[teamName], nil
}

func (f *fakeTeamStore) CreateTeamWithMembers(ctx context.Context, teamName string, members []struct {
	UserID   string
	Username string
	IsActive bool
}) error {
	if f.errOnCreateWithMembers != nil {
		return f.errOnCreateWithMembers
	}
	f.lastCreatedTeamName = teamName
	f.lastCreatedMembers = append([]struct {
		UserID   string
		Username string
		IsActive bool
	}(nil), members...)

	f.existingTeams[teamName] = true
	rows := make([]repo.TeamMemberRow, 0, len(members))
	for _, m := range members {
		rows = append(rows, repo.TeamMemberRow{
			UserID:   m.UserID,
			Username: m.Username,
			IsActive: m.IsActive,
		})
	}
	f.membersByTeam[teamName] = rows
	return nil
}

func (f *fakeTeamStore) GetTeamWithMembers(ctx context.Context, teamName string) ([]repo.TeamMemberRow, error) {
	if f.errOnGetWithMembers != nil {
		return nil, f.errOnGetWithMembers
	}
	return append([]repo.TeamMemberRow(nil), f.membersByTeam[teamName]...), nil
}

func TestTeamService_CreateTeam_Success(t *testing.T) {
	store := newFakeTeamStore()
	svc := NewTeamService(store)

	team := domain.Team{
		Name: "backend",
		Members: []domain.TeamMember{
			{ID: "u1", Username: "alice", IsActive: true},
			{ID: "u2", Username: "bob", IsActive: false},
		},
	}

	if err := svc.CreateTeam(context.Background(), team); err != nil {
		t.Fatalf("CreateTeam() unexpected error: %v", err)
	}

	if !store.existingTeams["backend"] {
		t.Fatalf("team must be marked as existing after creation")
	}
	if store.lastCreatedTeamName != "backend" {
		t.Fatalf("CreateTeamWithMembers called with teamName=%q, want %q", store.lastCreatedTeamName, "backend")
	}
	wantMembers := []struct {
		UserID   string
		Username string
		IsActive bool
	}{
		{UserID: "u1", Username: "alice", IsActive: true},
		{UserID: "u2", Username: "bob", IsActive: false},
	}
	if !reflect.DeepEqual(store.lastCreatedMembers, wantMembers) {
		t.Fatalf("members passed to CreateTeamWithMembers mismatch.\n got: %+v\nwant: %+v", store.lastCreatedMembers, wantMembers)
	}
}

func TestTeamService_CreateTeam_AlreadyExists(t *testing.T) {
	store := newFakeTeamStore()
	store.existingTeams["backend"] = true
	svc := NewTeamService(store)

	err := svc.CreateTeam(context.Background(), domain.Team{Name: "backend"})
	if !errors.Is(err, ErrTeamExists) {
		t.Fatalf("expected ErrTeamExists, got %v", err)
	}
}

func TestTeamService_CreateTeam_ExistsCheckError(t *testing.T) {
	store := newFakeTeamStore()
	store.errOnExistsCheck = errors.New("db error")
	svc := NewTeamService(store)

	err := svc.CreateTeam(context.Background(), domain.Team{Name: "backend"})
	if err == nil || !errors.Is(err, store.errOnExistsCheck) {
		t.Fatalf("expected exists check error, got %v", err)
	}
}

func TestTeamService_CreateTeam_CreateError(t *testing.T) {
	store := newFakeTeamStore()
	store.errOnCreateWithMembers = errors.New("insert error")
	svc := NewTeamService(store)

	err := svc.CreateTeam(context.Background(), domain.Team{
		Name: "backend",
		Members: []domain.TeamMember{
			{ID: "u1", Username: "alice", IsActive: true},
		},
	})
	if err == nil || !errors.Is(err, store.errOnCreateWithMembers) {
		t.Fatalf("expected create error, got %v", err)
	}
}

func TestTeamService_GetTeam_Success(t *testing.T) {
	store := newFakeTeamStore()
	store.existingTeams["backend"] = true
	store.membersByTeam["backend"] = []repo.TeamMemberRow{
		{UserID: "u1", Username: "alice", IsActive: true},
		{UserID: "u2", Username: "bob", IsActive: false},
	}
	svc := NewTeamService(store)

	team, err := svc.GetTeam(context.Background(), "backend")
	if err != nil {
		t.Fatalf("GetTeam() unexpected error: %v", err)
	}

	if team.Name != "backend" {
		t.Fatalf("team.Name = %q, want %q", team.Name, "backend")
	}
	wantMembers := []domain.TeamMember{
		{ID: "u1", Username: "alice", IsActive: true},
		{ID: "u2", Username: "bob", IsActive: false},
	}
	if !reflect.DeepEqual(team.Members, wantMembers) {
		t.Fatalf("team.Members mismatch.\n got: %+v\nwant: %+v", team.Members, wantMembers)
	}
}

func TestTeamService_GetTeam_NotFound(t *testing.T) {
	store := newFakeTeamStore()
	// Команда отсутствует в existingTeams
	svc := NewTeamService(store)

	_, err := svc.GetTeam(context.Background(), "backend")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestTeamService_GetTeam_ExistsCheckError(t *testing.T) {
	store := newFakeTeamStore()
	store.errOnExistsCheck = errors.New("db error")
	svc := NewTeamService(store)

	_, err := svc.GetTeam(context.Background(), "backend")
	if err == nil || !errors.Is(err, store.errOnExistsCheck) {
		t.Fatalf("expected exists check error, got %v", err)
	}
}

func TestTeamService_GetTeam_GetMembersError(t *testing.T) {
	store := newFakeTeamStore()
	store.existingTeams["backend"] = true
	store.errOnGetWithMembers = errors.New("select error")
	svc := NewTeamService(store)

	_, err := svc.GetTeam(context.Background(), "backend")
	if err == nil || !errors.Is(err, store.errOnGetWithMembers) {
		t.Fatalf("expected get members error, got %v", err)
	}
}


