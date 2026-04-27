package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v57/github"
	"go.uber.org/zap"

	"github.com/real-ass/github-service/internal/config"
	"github.com/real-ass/github-service/internal/domain"
	ghclient "github.com/real-ass/github-service/internal/github"
)

// GitHubService handles GitHub API operations and data synchronization
type GitHubService struct {
	client        *ghclient.Client
	repo          domain.RepositoryRepository
	cfg           *config.GitHubConfig
	logger        *zap.Logger
	repoAnalyzer  *RepoAnalyzer
	contrAnalyzer *ContributionAnalyzer
}

// NewGitHubService creates a new GitHub service
func NewGitHubService(
	client *ghclient.Client,
	repo domain.RepositoryRepository,
	cfg *config.GitHubConfig,
	logger *zap.Logger,
) *GitHubService {
	return &GitHubService{
		client:        client,
		repo:          repo,
		cfg:           cfg,
		logger:        logger,
		repoAnalyzer:  NewRepoAnalyzer(logger),
		contrAnalyzer: NewContributionAnalyzer(logger),
	}
}

// SyncRepository fetches and synchronizes a repository from GitHub
func (s *GitHubService) SyncRepository(ctx context.Context, owner, name string) (*domain.GitHubRepository, error) {
	ghRepo, _, err := s.client.GetRepository(ctx, owner, name)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repository from GitHub: %w", err)
	}

	repo := s.mapToRepository(ghRepo)

	existing, err := s.repo.GetRepositoryByOwnerName(ctx, owner, name)
	if err != nil {
		// Repository doesn't exist, create it
		if err := s.repo.CreateRepository(ctx, repo); err != nil {
			return nil, fmt.Errorf("failed to create repository: %w", err)
		}
	} else {
		// Repository exists, update it
		repo.ID = existing.ID
		if err := s.repo.UpdateRepository(ctx, repo); err != nil {
			return nil, fmt.Errorf("failed to update repository: %w", err)
		}
	}

	s.updateSyncStatus(ctx, "repository", repo.ID, "success", "")

	return repo, nil
}

// SyncContributors fetches and synchronizes contributors for a repository
func (s *GitHubService) SyncContributors(ctx context.Context, repoID int64, owner, name string) error {
	s.logger.Info("syncing contributors",
		zap.Int64("repo_id", repoID),
		zap.String("owner", owner),
		zap.String("name", name),
	)

	opts := &github.ListContributorsOptions{
		ListOptions: github.ListOptions{PerPage: s.cfg.PerPage},
	}

	var allContributors []*github.Contributor
	for {
		contributors, resp, err := s.client.ListContributors(ctx, owner, name, opts)
		if err != nil {
			return fmt.Errorf("failed to list contributors: %w", err)
		}

		allContributors = append(allContributors, contributors...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	domainContributors := make([]*domain.GitHubContributor, 0, len(allContributors))
	for _, c := range allContributors {
		domainContributors = append(domainContributors, &domain.GitHubContributor{
			ID:            c.GetID(),
			Login:         c.GetLogin(),
			AvatarURL:     c.GetAvatarURL(),
			HTMLURL:       c.GetHTMLURL(),
			Contributions: c.GetContributions(),
			RepositoryID:  repoID,
		})
	}

	if err := s.repo.BulkUpsertContributors(ctx, domainContributors); err != nil {
		return fmt.Errorf("failed to upsert contributors: %w", err)
	}

	s.updateSyncStatus(ctx, "contributors", repoID, "success", "")
	return nil
}

// SyncCommits fetches and synchronizes commits for a repository
func (s *GitHubService) SyncCommits(ctx context.Context, repoID int64, owner, name string, since time.Time) error {
	s.logger.Info("syncing commits",
		zap.Int64("repo_id", repoID),
		zap.String("owner", owner),
		zap.String("name", name),
	)

	opts := &github.CommitsListOptions{
		Since:       since,
		ListOptions: github.ListOptions{PerPage: s.cfg.PerPage},
	}

	var allCommits []*github.RepositoryCommit
	for {
		commits, resp, err := s.client.ListCommits(ctx, owner, name, opts)
		if err != nil {
			return fmt.Errorf("failed to list commits: %w", err)
		}

		allCommits = append(allCommits, commits...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	domainCommits := make([]*domain.GitHubCommit, 0, len(allCommits))
	for _, c := range allCommits {
		domainCommits = append(domainCommits, &domain.GitHubCommit{
			SHA:           c.GetSHA(),
			Message:       c.GetCommit().GetMessage(),
			AuthorName:    c.GetCommit().GetAuthor().GetName(),
			AuthorEmail:   c.GetCommit().GetAuthor().GetEmail(),
			AuthorDate:    c.GetCommit().GetAuthor().GetDate().Time,
			CommitterName: c.GetCommit().GetCommitter().GetName(),
			CommitterDate: c.GetCommit().GetCommitter().GetDate().Time,
			RepositoryID:  repoID,
		})
	}

	if err := s.repo.BulkInsertCommits(ctx, domainCommits); err != nil {
		return fmt.Errorf("failed to insert commits: %w", err)
	}

	s.updateSyncStatus(ctx, "commits", repoID, "success", "")
	return nil
}

// SyncPullRequests fetches and synchronizes pull requests for a repository
func (s *GitHubService) SyncPullRequests(ctx context.Context, repoID int64, owner, name string) error {
	s.logger.Info("syncing pull requests",
		zap.Int64("repo_id", repoID),
		zap.String("owner", owner),
		zap.String("name", name),
	)

	states := []string{"open", "closed", "all"}
	var allPRs []*github.PullRequest

	for _, state := range states {
		opts := &github.PullRequestListOptions{
			State:       state,
			ListOptions: github.ListOptions{PerPage: s.cfg.PerPage},
		}

		for {
			prs, resp, err := s.client.ListPullRequests(ctx, owner, name, opts)
			if err != nil {
				return fmt.Errorf("failed to list pull requests: %w", err)
			}

			allPRs = append(allPRs, prs...)
			if resp.NextPage == 0 {
				break
			}
			opts.Page = resp.NextPage
		}
	}

	domainPRs := make([]*domain.GitHubPullRequest, 0, len(allPRs))
	for _, pr := range allPRs {
		domainPRs = append(domainPRs, &domain.GitHubPullRequest{
			Number:       pr.GetNumber(),
			Title:        pr.GetTitle(),
			State:        pr.GetState(),
			UserLogin:    pr.GetUser().GetLogin(),
			CreatedAt:    pr.GetCreatedAt().Time,
			MergedAt:     pr.GetMergedAt().Time,
			ClosedAt:     pr.ClosedAt.Time,
			RepositoryID: repoID,
		})
	}

	if err := s.repo.BulkUpsertPullRequests(ctx, domainPRs); err != nil {
		return fmt.Errorf("failed to upsert pull requests: %w", err)
	}

	s.updateSyncStatus(ctx, "pull_requests", repoID, "success", "")
	return nil
}

// GetRepository retrieves a repository from the database
func (s *GitHubService) GetRepository(ctx context.Context, id int64) (*domain.GitHubRepository, error) {
	return s.repo.GetRepositoryByID(ctx, id)
}

// ListRepositories lists repositories from the database
func (s *GitHubService) ListRepositories(ctx context.Context, limit, offset int) ([]*domain.GitHubRepository, error) {
	return s.repo.ListRepositories(ctx, limit, offset)
}

// GetTopContributors retrieves the top contributors for a repository
func (s *GitHubService) GetTopContributors(ctx context.Context, repoID int64, limit int) ([]*domain.GitHubContributor, error) {
	return s.repo.GetTopContributors(ctx, repoID, limit)
}

// AnalyzeRepository runs a full analysis on a repository
func (s *GitHubService) AnalyzeRepository(ctx context.Context, repoID int64) (*domain.RepositoryMetrics, error) {
	commits, err := s.repo.GetCommitsByRepoID(ctx, repoID, time.Now().AddDate(0, -3, 0), time.Now(), 10000, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get commits: %w", err)
	}

	prs, err := s.repo.GetPullRequestsByRepoID(ctx, repoID, "all", 10000, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get pull requests: %w", err)
	}

	contributors, err := s.repo.GetContributorsByRepoID(ctx, repoID, 1000, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get contributors: %w", err)
	}

	metrics := s.repoAnalyzer.Analyze(commits, prs, contributors)
	metrics.RepositoryID = repoID
	metrics.CalculatedAt = time.Now()

	if err := s.repo.UpsertMetrics(ctx, metrics); err != nil {
		return nil, fmt.Errorf("failed to save metrics: %w", err)
	}

	return metrics, nil
}

// AnalyzeContributions runs a contribution analysis for a repository
func (s *GitHubService) AnalyzeContributions(ctx context.Context, repoID int64) ([]*domain.ContributionAnalysis, error) {
	commits, err := s.repo.GetCommitsByRepoID(ctx, repoID, time.Now().AddDate(0, -6, 0), time.Now(), 10000, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get commits: %w", err)
	}

	contributors, err := s.repo.GetContributorsByRepoID(ctx, repoID, 1000, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get contributors: %w", err)
	}

	analyses := s.contrAnalyzer.Analyze(commits, contributors)

	result := make([]*domain.ContributionAnalysis, 0, len(analyses))
	for _, a := range analyses {
		a.RepositoryID = repoID
		a.AnalysisDate = time.Now()
		if err := s.repo.CreateAnalysis(ctx, a); err != nil {
			return nil, fmt.Errorf("failed to save analysis: %w", err)
		}
		result = append(result, a)
	}

	return result, nil
}

func (s *GitHubService) mapToRepository(ghRepo *github.Repository) *domain.GitHubRepository {
	return &domain.GitHubRepository{
		Owner:           ghRepo.GetOwner().GetLogin(),
		Name:            ghRepo.GetName(),
		FullName:        ghRepo.GetFullName(),
		Description:     ghRepo.GetDescription(),
		HTMLURL:         ghRepo.GetHTMLURL(),
		DefaultBranch:   ghRepo.GetDefaultBranch(),
		Language:        ghRepo.GetLanguage(),
		StarsCount:      ghRepo.GetStargazersCount(),
		ForksCount:      ghRepo.GetForksCount(),
		OpenIssuesCount: ghRepo.GetOpenIssuesCount(),
		IsPrivate:       ghRepo.GetPrivate(),
		CreatedAt:       ghRepo.GetCreatedAt().Time,
		UpdatedAt:       ghRepo.GetUpdatedAt().Time,
		PushedAt:        ghRepo.GetPushedAt().Time,
		LastSyncedAt:    time.Now(),
	}
}

func (s *GitHubService) updateSyncStatus(ctx context.Context, entityType string, entityID int64, status, errMsg string) {
	syncStatus := &domain.SyncStatus{
		EntityType: entityType,
		EntityID:   entityID,
		Status:     status,
		LastSyncAt: time.Now(),
		Error:      errMsg,
	}
	if err := s.repo.UpdateSyncStatus(ctx, syncStatus); err != nil {
		s.logger.Error("failed to update sync status",
			zap.String("entity_type", entityType),
			zap.Int64("entity_id", entityID),
			zap.Error(err),
		)
	}
}
