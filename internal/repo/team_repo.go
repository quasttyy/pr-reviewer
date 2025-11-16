package repo

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	sqlInsertTeam = `
		INSERT INTO teams (team_name) VALUES ($1)
	`
	sqlUpdateUserByID = `
		UPDATE users
		SET username = $2,
		    is_active = $3,
		    team_name = $4
		WHERE user_id = $1
	`
	sqlInsertUser = `
		INSERT INTO users (user_id, username, is_active, team_name)
		VALUES ($1, $2, $3, $4)
	`
	sqlSelectTeamByName = `
		SELECT team_name
		FROM teams
		WHERE team_name = $1
		LIMIT 1
	`
	sqlSelectMembersByTeam = `
		SELECT user_id, username, is_active
		FROM users
		WHERE team_name = $1
		ORDER BY user_id
	`
)

type TeamRepo struct {
	pool *pgxpool.Pool
}

func NewTeamRepo(pool *pgxpool.Pool) *TeamRepo {
	return &TeamRepo{pool: pool}
}

func (r *TeamRepo) CreateTeamWithMembers(ctx context.Context, teamName string, members []struct {
	UserID   string
	Username string
	IsActive bool
}) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Создаем команду
	_, err = tx.Exec(ctx, sqlInsertTeam, teamName)
	if err != nil {
		return err
	}
	
	for _, m := range members {
		tag, errExec := tx.Exec(ctx, sqlUpdateUserByID, m.UserID, m.Username, m.IsActive, teamName)
		if errExec != nil {
			return errExec
		}
		if tag.RowsAffected() == 0 {
			if _, errIns := tx.Exec(ctx, sqlInsertUser, m.UserID, m.Username, m.IsActive, teamName); errIns != nil {
				return errIns
			}
		}
	}

	return tx.Commit(ctx)
}

func (r *TeamRepo) TeamExists(ctx context.Context, teamName string) (bool, error) {
	var name string
	err := r.pool.QueryRow(ctx, sqlSelectTeamByName, teamName).Scan(&name)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

type TeamMemberRow struct {
	UserID   string
	Username string
	IsActive bool
}

func (r *TeamRepo) GetTeamWithMembers(ctx context.Context, teamName string) ([]TeamMemberRow, error) {
	rows, err := r.pool.Query(ctx, sqlSelectMembersByTeam, teamName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []TeamMemberRow
	for rows.Next() {
		var row TeamMemberRow
		if err := rows.Scan(&row.UserID, &row.Username, &row.IsActive); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}