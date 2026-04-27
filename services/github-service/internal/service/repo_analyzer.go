package service

import (
	"math"
	"sort"
	"time"

	"go.uber.org/zap"

	"github.com/real-ass/github-service/internal/domain"
)

// RepoAnalyzer calculates repository-level metrics and health indicators
type RepoAnalyzer struct {
	logger *zap.Logger
}

// NewRepoAnalyzer creates a new repository analyzer
func NewRepoAnalyzer(logger *zap.Logger) *RepoAnalyzer {
	return &RepoAnalyzer{logger: logger}
}

// Analyze computes metrics for a repository based on its data
func (a *RepoAnalyzer) Analyze(
	commits []*domain.GitHubCommit,
	prs []*domain.GitHubPullRequest,
	contributors []*domain.GitHubContributor,
) *domain.RepositoryMetrics {
	metrics := &domain.RepositoryMetrics{}

	metrics.CommitFrequency = a.calculateCommitFrequency(commits)
	metrics.AvgPRSize = a.calculateAvgPRSize(commits, prs)
	metrics.MergeRate = a.calculateMergeRate(prs)
	metrics.AvgMergeTime = a.calculateAvgMergeTime(prs)
	metrics.BusFactor = a.calculateBusFactor(contributors)
	// Issue resolution rate would require issue data, placeholder for now
	metrics.IssueResolutionRate = 0.0

	a.logger.Info("calculated repository metrics",
		zap.Float64("commit_frequency", metrics.CommitFrequency),
		zap.Float64("avg_pr_size", metrics.AvgPRSize),
		zap.Float64("merge_rate", metrics.MergeRate),
		zap.Float64("avg_merge_time", metrics.AvgMergeTime),
		zap.Int("bus_factor", metrics.BusFactor),
	)

	return metrics
}

// calculateCommitFrequency calculates commits per week over the last 3 months
func (a *RepoAnalyzer) calculateCommitFrequency(commits []*domain.GitHubCommit) float64 {
	if len(commits) == 0 {
		return 0
	}

	// Find the date range
	var oldest, newest time.Time
	for i, c := range commits {
		if i == 0 || c.AuthorDate.Before(oldest) {
			oldest = c.AuthorDate
		}
		if i == 0 || c.AuthorDate.After(newest) {
			newest = c.AuthorDate
		}
	}

	if oldest.IsZero() || newest.IsZero() {
		return 0
	}

	weeks := newest.Sub(oldest).Hours() / (24 * 7)
	if weeks <= 0 {
		return float64(len(commits))
	}

	return float64(len(commits)) / weeks
}

// calculateAvgPRSize estimates average PR size based commit patterns
func (a *RepoAnalyzer) calculateAvgPRSize(commits []*domain.GitHubCommit, prs []*domain.GitHubPullRequest) float64 {
	if len(prs) == 0 || len(commits) == 0 {
		return 0
	}

	// Estimate based on total commits per PR ratio
	return float64(len(commits)) / float64(len(prs))
}

// calculateMergeRate calculates the ratio of merged PRs to total closed PRs
func (a *RepoAnalyzer) calculateMergeRate(prs []*domain.GitHubPullRequest) float64 {
	if len(prs) == 0 {
		return 0
	}

	var merged, closed int
	for _, pr := range prs {
		if pr.State == "closed" {
			closed++
			if !pr.MergedAt.IsZero() {
				merged++
			}
		}
	}

	if closed == 0 {
		return 0
	}

	return float64(merged) / float64(closed)
}

// calculateAvgMergeTime calculates the average time from PR creation to merge
func (a *RepoAnalyzer) calculateAvgMergeTime(prs []*domain.GitHubPullRequest) float64 {
	var totalTime float64
	var count int

	for _, pr := range prs {
		if !pr.MergedAt.IsZero() && !pr.CreatedAt.IsZero() {
			duration := pr.MergedAt.Sub(pr.CreatedAt).Hours()
			totalTime += duration
			count++
		}
	}

	if count == 0 {
		return 0
	}

	return totalTime / float64(count)
}

// calculateBusFactor estimates the bus factor based on contributor contribution distribution
func (a *RepoAnalyzer) calculateBusFactor(contributors []*domain.GitHubContributor) int {
	if len(contributors) == 0 {
		return 0
	}

	// Sort contributors by contributions descending
	sorted := make([]*domain.GitHubContributor, len(contributors))
	copy(sorted, contributors)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Contributions > sorted[j].Contributions
	})

	// Calculate total contributions
	var totalContributions int
	for _, c := range sorted {
		totalContributions += c.Contributions
	}

	if totalContributions == 0 {
		return len(sorted)
	}

	// Find minimum number of contributors that account for >50% of contributions
	var cumulative int
	busFactor := 0
	threshold := float64(totalContributions) * 0.5

	for _, c := range sorted {
		cumulative += c.Contributions
		busFactor++
		if float64(cumulative) > threshold {
			break
		}
	}

	// Ensure minimum bus factor of 1
	return int(math.Max(1, float64(busFactor)))
}
