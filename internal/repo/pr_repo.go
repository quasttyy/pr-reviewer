package repo

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
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
	sqlInsertPR = `
		INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status, need_more_reviewers)
		VALUES ($1, $2, $3, 'OPEN', $4)
	`
	sqlSelectTeamActiveCandidatesExcluding = `
		SELECT u.user_id
		FROM users u
		WHERE u.team_name = $1 AND u.is_active = true AND u.user_id <> $2
		ORDER BY u.user_id
	`
	sqlInsertReviewer = `
		INSERT INTO pr_reviewers (pull_request_id, reviewer_id)
		VALUES ($1, $2)
	`
	sqlSelectPRByID = `
		SELECT pull_request_id, pull_request_name, author_id, status, need_more_reviewers, created_at, merged_at
		FROM pull_requests
		WHERE pull_request_id = $1
	`
	sqlSelectReviewersByPR = `
		SELECT reviewer_id
		FROM pr_reviewers
		WHERE pull_request_id = $1
		ORDER BY reviewer_id
	`
	sqlUpdatePRMerged = `
		UPDATE pull_requests
		SET status = 'MERGED',
		    merged_at = COALESCE(merged_at, NOW())
		WHERE pull_request_id = $1
		RETURNING pull_request_id
	`
	sqlReplaceReviewer = `
		DELETE FROM pr_reviewers WHERE pull_request_id = $1 AND reviewer_id = $2;
	`
	sqlSelectTeamActiveCandidates = `
		SELECT u.user_id
		FROM users u
		WHERE u.team_name = $1 AND u.is_active = true
		ORDER BY u.user_id
	`
	sqlSelectUserTeamByID = `
		SELECT team_name
		FROM users
		WHERE user_id = $1
	`
	sqlSelectReviewerStats = `
		SELECT r.reviewer_id, COUNT(*) AS total_assigned
		FROM pr_reviewers r
		GROUP BY r.reviewer_id
		ORDER BY r.reviewer_id
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

type PRFull struct {
	ID                string
	Name              string
	AuthorID          string
	Status            string
	NeedMoreReviewers bool
	Assigned          []string
	CreatedAt         *time.Time
	MergedAt          *time.Time
}

// CreatePROpenWithAssigned создаёт PR и назначает переданных ревьюеров
func (r *PRRepo) CreatePROpenWithAssigned(ctx context.Context, id, name, author string, needMore bool, reviewers []string) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, sqlInsertPR, id, name, author, needMore); err != nil {
		return err
	}
	for _, rv := range reviewers {
		if _, err := tx.Exec(ctx, sqlInsertReviewer, id, rv); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *PRRepo) GetPR(ctx context.Context, id string) (PRFull, error) {
	var pr PRFull
	if err := r.pool.QueryRow(ctx, sqlSelectPRByID, id).Scan(&pr.ID, &pr.Name, &pr.AuthorID, &pr.Status, &pr.NeedMoreReviewers, &pr.CreatedAt, &pr.MergedAt); err != nil {
		return PRFull{}, err
	}
	rows, err := r.pool.Query(ctx, sqlSelectReviewersByPR, id)
	if err != nil {
		return PRFull{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var rv string
		if err := rows.Scan(&rv); err != nil {
			return PRFull{}, err
		}
		pr.Assigned = append(pr.Assigned, rv)
	}
	return pr, rows.Err()
}

func (r *PRRepo) MarkMerged(ctx context.Context, id string) error {
	var got string
	if err := r.pool.QueryRow(ctx, sqlUpdatePRMerged, id).Scan(&got); err != nil {
		return err
	}
	return nil
}

func (r *PRRepo) GetUserTeam(ctx context.Context, userID string) (string, error) {
	var team string
	if err := r.pool.QueryRow(ctx, sqlSelectUserTeamByID, userID).Scan(&team); err != nil {
		return "", err
	}
	return team, nil
}

func (r *PRRepo) GetActiveCandidatesFromTeamExcluding(ctx context.Context, teamName, excludeUserID string) ([]string, error) {
	rows, err := r.pool.Query(ctx, sqlSelectTeamActiveCandidatesExcluding, teamName, excludeUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *PRRepo) ReplaceReviewer(ctx context.Context, prID, oldReviewer, newReviewer string) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	ct, err := tx.Exec(ctx, sqlReplaceReviewer, prID, oldReviewer)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return errors.New("not assigned")
	}
	if _, err := tx.Exec(ctx, sqlInsertReviewer, prID, newReviewer); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ReviewerStatRow описывает простую статистику назначений ревьюеров.
type ReviewerStatRow struct {
	ReviewerID    string
	TotalAssigned int64
}

// GetReviewerStats возвращает статистику назначений по ревьюверам.
func (r *PRRepo) GetReviewerStats(ctx context.Context) ([]ReviewerStatRow, error) {
	rows, err := r.pool.Query(ctx, sqlSelectReviewerStats)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []ReviewerStatRow
	for rows.Next() {
		var s ReviewerStatRow
		if err := rows.Scan(&s.ReviewerID, &s.TotalAssigned); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}