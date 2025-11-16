package repo

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	sqlUpdateUserIsActive = `
		UPDATE users
		SET is_active = $2
		WHERE user_id = $1
		RETURNING user_id, username, team_name, is_active
	`
	sqlSelectUserByID = `
		SELECT user_id, username, team_name, is_active
		FROM users
		WHERE user_id = $1
		LIMIT 1
	`
)

type UserRow struct {
	UserID   string
	Username string
	TeamName string
	IsActive bool
}

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

func (r *UserRepo) UpdateIsActive(ctx context.Context, userID string, isActive bool) (UserRow, error) {
	var row UserRow
	err := r.pool.QueryRow(ctx, sqlUpdateUserIsActive, userID, isActive).Scan(
		&row.UserID, &row.Username, &row.TeamName, &row.IsActive,
	)
	return row, err
}

func (r *UserRepo) GetByID(ctx context.Context, userID string) (UserRow, error) {
	var row UserRow
	err := r.pool.QueryRow(ctx, sqlSelectUserByID, userID).Scan(
		&row.UserID, &row.Username, &row.TeamName, &row.IsActive,
	)
	return row, err
}
