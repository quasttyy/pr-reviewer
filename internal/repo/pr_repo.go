package repo

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	sqlSelectPRsByReviewer = `
		SELECT pr.pull_request_id, pr.pull_request_name, pr.author_id, pr.status
		FROM pull_requests pr
		JOIN pr_reviewers r ON r.pull_request_id = pr.pull_request_id
		WHERE r.reviewer_id = $1
		ORDER BY pr.pull_request_id
	`
)

type PRShortRow struct {
	ID       string
	Name     string
	AuthorID string
	Status   string
}

type PRRepo struct {
	pool *pgxpool.Pool
}

func NewPRRepo(pool *pgxpool.Pool) *PRRepo {
	return &PRRepo{pool: pool}
}

func (r *PRRepo) GetShortByReviewer(ctx context.Context, userID string) ([]PRShortRow, error) {
	rows, err := r.pool.Query(ctx, sqlSelectPRsByReviewer, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PRShortRow
	for rows.Next() {
		var row PRShortRow
		if err := rows.Scan(&row.ID, &row.Name, &row.AuthorID, &row.Status); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}