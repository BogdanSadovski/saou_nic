package domain

import (
	"context"
	"time"
)

type RepositoryRepository interface {
	// GitHubRepository operations
	CreateRepository(ctx context.Context, repo *GitHubRepository) error
	GetRepositoryByID(ctx context.Context, id int64) (*GitHubRepository, error)
	GetRepositoryByOwnerName(ctx context.Context, owner, name string) (*GitHubRepository, error)
	ListRepositories(ctx context.Context, limit, offset int) ([]*GitHubRepository, error)
	ListRepositoriesByOwner(ctx context.Context, owner string, limit, offset int) ([]*GitHubRepository, error)
	UpdateRepository(ctx context.Context, repo *GitHubRepository) error
	DeleteRepository(ctx context.Context, id int64) error

	// GitHubContributor operations
	BulkUpsertContributors(ctx context.Context, contributors []*GitHubContributor) error
	GetContributorsByRepoID(ctx context.Context, repoID int64, limit, offset int) ([]*GitHubContributor, error)
	GetTopContributors(ctx context.Context, repoID int64, limit int) ([]*GitHubContributor, error)

	// GitHubCommit operations
	BulkInsertCommits(ctx context.Context, commits []*GitHubCommit) error
	GetCommitsByRepoID(ctx context.Context, repoID int64, since, until time.Time, limit, offset int) ([]*GitHubCommit, error)
	GetCommitCountByRepoID(ctx context.Context, repoID int64) (int, error)

	// GitHubPullRequest operations
	BulkUpsertPullRequests(ctx context.Context, prs []*GitHubPullRequest) error
	GetPullRequestsByRepoID(ctx context.Context, repoID int64, state string, limit, offset int) ([]*GitHubPullRequest, error)

	// RepositoryMetrics operations
	UpsertMetrics(ctx context.Context, metrics *RepositoryMetrics) error
	GetMetricsByRepoID(ctx context.Context, repoID int64) (*RepositoryMetrics, error)

	// ContributionAnalysis operations
	CreateAnalysis(ctx context.Context, analysis *ContributionAnalysis) error
	GetAnalysisByRepoID(ctx context.Context, repoID int64) ([]*ContributionAnalysis, error)

	// SyncStatus operations
	UpdateSyncStatus(ctx context.Context, status *SyncStatus) error
	GetSyncStatus(ctx context.Context, entityType string, entityID int64) (*SyncStatus, error)
}
