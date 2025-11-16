package service

import (
	"context"
	"errors"

	"github.com/quasttyy/pr-reviewer/internal/repo"
)

var (
	ErrUserNotFound = errors.New("user not found")
)

// UserStore описывает операции над пользователями, необходимые сервису
type UserStore interface {
	UpdateIsActive(ctx context.Context, userID string, isActive bool) (repo.UserRow, error)
	GetByID(ctx context.Context, userID string) (repo.UserRow, error)
}

// PRShortStore описывает операции получения краткой информации о PR пользователя
type PRShortStore interface {
	GetShortByReviewer(ctx context.Context, userID string) ([]repo.PRShortRow, error)
}

type UserService struct {
	users UserStore
	prs   PRShortStore
}

func NewUserService(users UserStore, prs PRShortStore) *UserService {
	return &UserService{users: users, prs: prs}
}

func (s *UserService) SetIsActiveAdmin(ctx context.Context, userID string, isActive bool) (repo.UserRow, error) {
	row, err := s.users.UpdateIsActive(ctx, userID, isActive)
	if err != nil {
		// Если пользователя нет, то вернём корректную 404 наверху
		return repo.UserRow{}, err
	}
	return row, nil
}

func (s *UserService) GetUserReviews(ctx context.Context, userID string) ([]repo.PRShortRow, error) {
	// Проверим, что пользователь существует (даже если неактивный)
	if _, err := s.users.GetByID(ctx, userID); err != nil {
		return nil, err
	}
	return s.prs.GetShortByReviewer(ctx, userID)
}