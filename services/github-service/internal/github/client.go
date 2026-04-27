package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/google/go-github/v57/github"
	"go.uber.org/zap"
	"golang.org/x/oauth2"

	"github.com/real-ass/github-service/internal/config"
)

// Client wraps the official GitHub Go client with additional functionality
type Client struct {
	rest      *github.Client
	baseURL   string
	logger    *zap.Logger
	limiter   *RateLimiter
	userAgent string
}

// NewClient creates a new GitHub API client
func NewClient(cfg *config.GitHubConfig, logger *zap.Logger) (*Client, error) {
	httpClient := &http.Client{
		Timeout: cfg.Timeout,
	}

	if cfg.AccessToken != "" {
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: cfg.AccessToken})
		httpClient = oauth2.NewClient(context.Background(), ts)
	}

	ghClient := github.NewClient(httpClient)
	ghClient.UserAgent = "real-ass-github-service"

	if cfg.BaseURL != "" && cfg.BaseURL != "https://api.github.com" {
		parsedURL, err := url.Parse(cfg.BaseURL)
		if err != nil {
			return nil, fmt.Errorf("invalid base URL: %w", err)
		}
		ghClient.BaseURL = parsedURL
	}

	limiter, err := NewRateLimiter(cfg.RateLimit.RequestsPerMinute, cfg.RateLimit.BurstSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create rate limiter: %w", err)
	}

	return &Client{
		rest:      ghClient,
		baseURL:   cfg.BaseURL,
		logger:    logger,
		limiter:   limiter,
		userAgent: "real-ass-github-service",
	}, nil
}

// GetRepository fetches a repository by owner and name
func (c *Client) GetRepository(ctx context.Context, owner, repo string) (*github.Repository, *github.Response, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, nil, fmt.Errorf("rate limiter wait: %w", err)
	}

	c.logger.Debug("fetching repository", zap.String("owner", owner), zap.String("repo", repo))
	repository, resp, err := c.rest.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return nil, resp, fmt.Errorf("failed to get repository: %w", err)
	}
	return repository, resp, nil
}

// ListRepositories lists repositories for a user or organization
func (c *Client) ListRepositories(ctx context.Context, owner string, opts *github.RepositoryListByUserOptions) ([]*github.Repository, *github.Response, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, nil, fmt.Errorf("rate limiter wait: %w", err)
	}

	c.logger.Debug("listing repositories", zap.String("owner", owner))
	repos, resp, err := c.rest.Repositories.ListByUser(ctx, owner, opts)
	if err != nil {
		return nil, resp, fmt.Errorf("failed to list repositories: %w", err)
	}
	return repos, resp, nil
}

// ListContributors fetches contributors for a repository
func (c *Client) ListContributors(ctx context.Context, owner, repo string, opts *github.ListContributorsOptions) ([]*github.Contributor, *github.Response, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, nil, fmt.Errorf("rate limiter wait: %w", err)
	}

	c.logger.Debug("listing contributors", zap.String("owner", owner), zap.String("repo", repo))
	contributors, resp, err := c.rest.Repositories.ListContributors(ctx, owner, repo, opts)
	if err != nil {
		return nil, resp, fmt.Errorf("failed to list contributors: %w", err)
	}
	return contributors, resp, nil
}

// ListCommits fetches commits for a repository
func (c *Client) ListCommits(ctx context.Context, owner, repo string, opts *github.CommitsListOptions) ([]*github.RepositoryCommit, *github.Response, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, nil, fmt.Errorf("rate limiter wait: %w", err)
	}

	c.logger.Debug("listing commits", zap.String("owner", owner), zap.String("repo", repo))
	commits, resp, err := c.rest.Repositories.ListCommits(ctx, owner, repo, opts)
	if err != nil {
		return nil, resp, fmt.Errorf("failed to list commits: %w", err)
	}
	return commits, resp, nil
}

// ListPullRequests fetches pull requests for a repository
func (c *Client) ListPullRequests(ctx context.Context, owner, repo string, opts *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, nil, fmt.Errorf("rate limiter wait: %w", err)
	}

	c.logger.Debug("listing pull requests", zap.String("owner", owner), zap.String("repo", repo))
	prs, resp, err := c.rest.PullRequests.List(ctx, owner, repo, opts)
	if err != nil {
		return nil, resp, fmt.Errorf("failed to list pull requests: %w", err)
	}
	return prs, resp, nil
}

// GetPullRequest fetches a specific pull request
func (c *Client) GetPullRequest(ctx context.Context, owner, repo string, number int) (*github.PullRequest, *github.Response, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, nil, fmt.Errorf("rate limiter wait: %w", err)
	}

	c.logger.Debug("fetching pull request",
		zap.String("owner", owner),
		zap.String("repo", repo),
		zap.Int("number", number),
	)
	pr, resp, err := c.rest.PullRequests.Get(ctx, owner, repo, number)
	if err != nil {
		return nil, resp, fmt.Errorf("failed to get pull request: %w", err)
	}
	return pr, resp, nil
}

// GetCommit fetches a specific commit
func (c *Client) GetCommit(ctx context.Context, owner, repo, sha string) (*github.RepositoryCommit, *github.Response, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, nil, fmt.Errorf("rate limiter wait: %w", err)
	}

	c.logger.Debug("fetching commit",
		zap.String("owner", owner),
		zap.String("repo", repo),
		zap.String("sha", sha),
	)
	commit, resp, err := c.rest.Repositories.GetCommit(ctx, owner, repo, sha, nil)
	if err != nil {
		return nil, resp, fmt.Errorf("failed to get commit: %w", err)
	}
	return commit, resp, nil
}

// GetRateLimit fetches the current rate limit status
func (c *Client) GetRateLimit(ctx context.Context) (*github.RateLimits, error) {
	limits, _, err := c.rest.RateLimits(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get rate limits: %w", err)
	}
	return limits, nil
}

// RemainingRequests returns the approximate number of remaining requests
func (c *Client) RemainingRequests() int {
	return c.limiter.Remaining()
}

// ResetTime returns when the rate limit will reset
func (c *Client) ResetTime() time.Time {
	return c.limiter.ResetAt()
}
