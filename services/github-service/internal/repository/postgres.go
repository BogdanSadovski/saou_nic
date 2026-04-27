package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/real-ass/github-service/internal/config"
	"github.com/real-ass/github-service/internal/domain"
)

type postgresRepo struct {
	db *sqlx.DB
}

func NewPostgresRepository(cfg *config.PostgresConfig) (domain.RepositoryRepository, error) {
	db, err := sqlx.Connect("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return &postgresRepo{db: db}, nil
}

func (r *postgresRepo) Close() error {
	return r.db.Close()
}

func (r *postgresRepo) CreateRepository(ctx context.Context, repo *domain.GitHubRepository) error {
	query := `
		INSERT INTO repositories (
			owner, name, full_name, description, html_url, default_branch,
			language, stars_count, forks_count, open_issues_count,
			is_private, created_at, updated_at, pushed_at, last_synced_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
		) RETURNING id`

	return r.db.QueryRowxContext(ctx, query,
		repo.Owner, repo.Name, repo.FullName, repo.Description, repo.HTMLURL,
		repo.DefaultBranch, repo.Language, repo.StarsCount, repo.ForksCount,
		repo.OpenIssuesCount, repo.IsPrivate, repo.CreatedAt, repo.UpdatedAt,
		repo.PushedAt, repo.LastSyncedAt,
	).Scan(&repo.ID)
}

func (r *postgresRepo) GetRepositoryByID(ctx context.Context, id int64) (*domain.GitHubRepository, error) {
	var repo domain.GitHubRepository
	query := `SELECT * FROM repositories WHERE id = $1`
	err := r.db.GetContext(ctx, &repo, query, id)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("repository with id %d not found: %w", id, err)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}
	return &repo, nil
}

func (r *postgresRepo) GetRepositoryByOwnerName(ctx context.Context, owner, name string) (*domain.GitHubRepository, error) {
	var repo domain.GitHubRepository
	query := `SELECT * FROM repositories WHERE owner = $1 AND name = $2`
	err := r.db.GetContext(ctx, &repo, query, owner, name)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("repository %s/%s not found: %w", owner, name, err)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}
	return &repo, nil
}

func (r *postgresRepo) ListRepositories(ctx context.Context, limit, offset int) ([]*domain.GitHubRepository, error) {
	var repos []*domain.GitHubRepository
	query := `SELECT * FROM repositories ORDER BY stars_count DESC LIMIT $1 OFFSET $2`
	err := r.db.SelectContext(ctx, &repos, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}
	return repos, nil
}

func (r *postgresRepo) ListRepositoriesByOwner(ctx context.Context, owner string, limit, offset int) ([]*domain.GitHubRepository, error) {
	var repos []*domain.GitHubRepository
	query := `SELECT * FROM repositories WHERE owner = $1 ORDER BY stars_count DESC LIMIT $2 OFFSET $3`
	err := r.db.SelectContext(ctx, &repos, query, owner, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories by owner: %w", err)
	}
	return repos, nil
}

func (r *postgresRepo) UpdateRepository(ctx context.Context, repo *domain.GitHubRepository) error {
	query := `
		UPDATE repositories SET
			description = $1, language = $2, stars_count = $3, forks_count = $4,
			open_issues_count = $5, updated_at = $6, pushed_at = $7, last_synced_at = $8
		WHERE id = $9`

	_, err := r.db.ExecContext(ctx, query,
		repo.Description, repo.Language, repo.StarsCount, repo.ForksCount,
		repo.OpenIssuesCount, repo.UpdatedAt, repo.PushedAt, repo.LastSyncedAt, repo.ID,
	)
	return err
}

func (r *postgresRepo) DeleteRepository(ctx context.Context, id int64) error {
	query := `DELETE FROM repositories WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *postgresRepo) BulkUpsertContributors(ctx context.Context, contributors []*domain.GitHubContributor) error {
	if len(contributors) == 0 {
		return nil
	}

	query := `
		INSERT INTO contributors (id, login, avatar_url, html_url, contributions, repository_id)
		VALUES %s
		ON CONFLICT (id, repository_id) DO UPDATE SET
			login = EXCLUDED.login,
			avatar_url = EXCLUDED.avatar_url,
			html_url = EXCLUDED.html_url,
			contributions = EXCLUDED.contributions`

	// Placeholder for batch insert logic
	_ = query
	for _, c := range contributors {
		upsertQuery := `
			INSERT INTO contributors (id, login, avatar_url, html_url, contributions, repository_id)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (id, repository_id) DO UPDATE SET
				login = EXCLUDED.login,
				contributions = EXCLUDED.contributions`

		_, err := r.db.ExecContext(ctx, upsertQuery,
			c.ID, c.Login, c.AvatarURL, c.HTMLURL, c.Contributions, c.RepositoryID,
		)
		if err != nil {
			return fmt.Errorf("failed to upsert contributor: %w", err)
		}
	}
	return nil
}

func (r *postgresRepo) GetContributorsByRepoID(ctx context.Context, repoID int64, limit, offset int) ([]*domain.GitHubContributor, error) {
	var contributors []*domain.GitHubContributor
	query := `SELECT * FROM contributors WHERE repository_id = $1 ORDER BY contributions DESC LIMIT $2 OFFSET $3`
	err := r.db.SelectContext(ctx, &contributors, query, repoID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get contributors: %w", err)
	}
	return contributors, nil
}

func (r *postgresRepo) GetTopContributors(ctx context.Context, repoID int64, limit int) ([]*domain.GitHubContributor, error) {
	var contributors []*domain.GitHubContributor
	query := `SELECT * FROM contributors WHERE repository_id = $1 ORDER BY contributions DESC LIMIT $2`
	err := r.db.SelectContext(ctx, &contributors, query, repoID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get top contributors: %w", err)
	}
	return contributors, nil
}

func (r *postgresRepo) BulkInsertCommits(ctx context.Context, commits []*domain.GitHubCommit) error {
	if len(commits) == 0 {
		return nil
	}

	query := `
		INSERT INTO commits (sha, message, author_name, author_email, author_date, committer_name, committer_date, repository_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (sha, repository_id) DO NOTHING`

	for _, c := range commits {
		_, err := r.db.ExecContext(ctx, query,
			c.SHA, c.Message, c.AuthorName, c.AuthorEmail, c.AuthorDate,
			c.CommitterName, c.CommitterDate, c.RepositoryID,
		)
		if err != nil {
			return fmt.Errorf("failed to insert commit %s: %w", c.SHA, err)
		}
	}
	return nil
}

func (r *postgresRepo) GetCommitsByRepoID(ctx context.Context, repoID int64, since, until time.Time, limit, offset int) ([]*domain.GitHubCommit, error) {
	var commits []*domain.GitHubCommit
	query := `
		SELECT * FROM commits 
		WHERE repository_id = $1 AND author_date BETWEEN $2 AND $3
		ORDER BY author_date DESC LIMIT $4 OFFSET $5`
	err := r.db.SelectContext(ctx, &commits, query, repoID, since, until, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get commits: %w", err)
	}
	return commits, nil
}

func (r *postgresRepo) GetCommitCountByRepoID(ctx context.Context, repoID int64) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM commits WHERE repository_id = $1`
	err := r.db.GetContext(ctx, &count, query, repoID)
	if err != nil {
		return 0, fmt.Errorf("failed to get commit count: %w", err)
	}
	return count, nil
}

func (r *postgresRepo) BulkUpsertPullRequests(ctx context.Context, prs []*domain.GitHubPullRequest) error {
	if len(prs) == 0 {
		return nil
	}

	query := `
		INSERT INTO pull_requests (number, title, state, user_login, created_at, merged_at, closed_at, repository_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (number, repository_id) DO UPDATE SET
			title = EXCLUDED.title,
			state = EXCLUDED.state,
			merged_at = EXCLUDED.merged_at,
			closed_at = EXCLUDED.closed_at`

	for _, pr := range prs {
		_, err := r.db.ExecContext(ctx, query,
			pr.Number, pr.Title, pr.State, pr.UserLogin,
			pr.CreatedAt, pr.MergedAt, pr.ClosedAt, pr.RepositoryID,
		)
		if err != nil {
			return fmt.Errorf("failed to upsert PR #%d: %w", pr.Number, err)
		}
	}
	return nil
}

func (r *postgresRepo) GetPullRequestsByRepoID(ctx context.Context, repoID int64, state string, limit, offset int) ([]*domain.GitHubPullRequest, error) {
	var prs []*domain.GitHubPullRequest
	query := `SELECT * FROM pull_requests WHERE repository_id = $1 AND state = $2 ORDER BY created_at DESC LIMIT $3 OFFSET $4`
	err := r.db.SelectContext(ctx, &prs, query, repoID, state, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get pull requests: %w", err)
	}
	return prs, nil
}

func (r *postgresRepo) UpsertMetrics(ctx context.Context, metrics *domain.RepositoryMetrics) error {
	query := `
		INSERT INTO repository_metrics (
			repository_id, commit_frequency, avg_pr_size, merge_rate,
			avg_merge_time, issue_resolution_rate, bus_factor, calculated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (repository_id) DO UPDATE SET
			commit_frequency = EXCLUDED.commit_frequency,
			avg_pr_size = EXCLUDED.avg_pr_size,
			merge_rate = EXCLUDED.merge_rate,
			avg_merge_time = EXCLUDED.avg_merge_time,
			issue_resolution_rate = EXCLUDED.issue_resolution_rate,
			bus_factor = EXCLUDED.bus_factor,
			calculated_at = EXCLUDED.calculated_at`

	_, err := r.db.ExecContext(ctx, query,
		metrics.RepositoryID, metrics.CommitFrequency, metrics.AvgPRSize,
		metrics.MergeRate, metrics.AvgMergeTime, metrics.IssueResolutionRate,
		metrics.BusFactor, metrics.CalculatedAt,
	)
	return err
}

func (r *postgresRepo) GetMetricsByRepoID(ctx context.Context, repoID int64) (*domain.RepositoryMetrics, error) {
	var metrics domain.RepositoryMetrics
	query := `SELECT * FROM repository_metrics WHERE repository_id = $1`
	err := r.db.GetContext(ctx, &metrics, query, repoID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("metrics for repository %d not found: %w", repoID, err)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}
	return &metrics, nil
}

func (r *postgresRepo) CreateAnalysis(ctx context.Context, analysis *domain.ContributionAnalysis) error {
	query := `
		INSERT INTO contribution_analyses (
			repository_id, contributor_login, total_commits, total_additions,
			total_deletions, files_changed, review_count, analysis_date
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`

	return r.db.QueryRowxContext(ctx, query,
		analysis.RepositoryID, analysis.ContributorLogin, analysis.TotalCommits,
		analysis.TotalAdditions, analysis.TotalDeletions, analysis.FilesChanged,
		analysis.ReviewCount, analysis.AnalysisDate,
	).Scan(&analysis.ID)
}

func (r *postgresRepo) GetAnalysisByRepoID(ctx context.Context, repoID int64) ([]*domain.ContributionAnalysis, error) {
	var analyses []*domain.ContributionAnalysis
	query := `SELECT * FROM contribution_analyses WHERE repository_id = $1 ORDER BY analysis_date DESC`
	err := r.db.SelectContext(ctx, &analyses, query, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to get analyses: %w", err)
	}
	return analyses, nil
}

func (r *postgresRepo) UpdateSyncStatus(ctx context.Context, status *domain.SyncStatus) error {
	query := `
		INSERT INTO sync_statuses (entity_type, entity_id, status, last_sync_at, error)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (entity_type, entity_id) DO UPDATE SET
			status = EXCLUDED.status,
			last_sync_at = EXCLUDED.last_sync_at,
			error = EXCLUDED.error`

	_, err := r.db.ExecContext(ctx, query,
		status.EntityType, status.EntityID, status.Status, status.LastSyncAt, status.Error,
	)
	return err
}

func (r *postgresRepo) GetSyncStatus(ctx context.Context, entityType string, entityID int64) (*domain.SyncStatus, error) {
	var status domain.SyncStatus
	query := `SELECT * FROM sync_statuses WHERE entity_type = $1 AND entity_id = $2`
	err := r.db.GetContext(ctx, &status, query, entityType, entityID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("sync status for %s %d not found: %w", entityType, entityID, err)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get sync status: %w", err)
	}
	return &status, nil
}
