package service

import (
	"context"
	"errors"

	"github.com/quasttyy/pr-reviewer/internal/domain"
	"github.com/quasttyy/pr-reviewer/internal/repo"
)

var (
	ErrTeamExists = errors.New("team already exists")
	ErrNotFound   = errors.New("not found")
)

type TeamService struct {
	teams *repo.TeamRepo
}

func NewTeamService(teams *repo.TeamRepo) *TeamService {
	return &TeamService{teams: teams}
}

func (s *TeamService) CreateTeam(ctx context.Context, team domain.Team) error {
	exists, err := s.teams.TeamExists(ctx, team.Name)
	if err != nil {
		return err
	}
	if exists {
		return ErrTeamExists
	}
	members := make([]struct {
		UserID   string
		Username string
		IsActive bool
	}, 0, len(team.Members))
	for _, m := range team.Members {
		members = append(members, struct {
			UserID   string
			Username string
			IsActive bool
		}{
			UserID:   m.ID,
			Username: m.Username,
			IsActive: m.IsActive,
		})
	}
	return s.teams.CreateTeamWithMembers(ctx, team.Name, members)
}

func (s *TeamService) GetTeam(ctx context.Context, teamName string) (domain.Team, error) {
	exists, err := s.teams.TeamExists(ctx, teamName)
	if err != nil {
		return domain.Team{}, err
	}
	if !exists {
		return domain.Team{}, ErrNotFound
	}
	rows, err := s.teams.GetTeamWithMembers(ctx, teamName)
	if err != nil {
		return domain.Team{}, err
	}
	members := make([]domain.TeamMember, 0, len(rows))
	for _, r := range rows {
		members = append(members, domain.TeamMember{
			ID:       r.UserID,
			Username: r.Username,
			IsActive: r.IsActive,
		})
	}
	return domain.Team{
		Name:    teamName,
		Members: members,
	}, nil
}