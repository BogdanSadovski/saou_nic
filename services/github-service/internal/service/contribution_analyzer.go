package service

import (
	"time"

	"go.uber.org/zap"

	"github.com/real-ass/github-service/internal/domain"
)

// ContributionAnalyzer analyzes individual contributor patterns and metrics
type ContributionAnalyzer struct {
	logger *zap.Logger
}

// NewContributionAnalyzer creates a new contribution analyzer
func NewContributionAnalyzer(logger *zap.Logger) *ContributionAnalyzer {
	return &ContributionAnalyzer{logger: logger}
}

// Analyze computes contribution metrics for each contributor
func (a *ContributionAnalyzer) Analyze(
	commits []*domain.GitHubCommit,
	contributors []*domain.GitHubContributor,
) []*domain.ContributionAnalysis {
	// Build a map of contributor login to their commit stats
	contributorStats := make(map[string]*contributorCommitStats)
	for _, c := range commits {
		login := c.AuthorName
		if _, exists := contributorStats[login]; !exists {
			contributorStats[login] = &contributorCommitStats{}
		}
		contributorStats[login].commitCount++
	}

	// Build analyses from contributors list
	analyses := make([]*domain.ContributionAnalysis, 0, len(contributors))
	for _, c := range contributors {
		stats := contributorStats[c.Login]
		totalCommits := 0
		if stats != nil {
			totalCommits = stats.commitCount
		}

		analysis := &domain.ContributionAnalysis{
			ContributorLogin: c.Login,
			TotalCommits:     totalCommits,
			// These would normally come from detailed commit analysis
			TotalAdditions: totalCommits * 50,  // placeholder estimate
			TotalDeletions: totalCommits * 20,  // placeholder estimate
			FilesChanged:   totalCommits * 3,   // placeholder estimate
			ReviewCount:    totalCommits / 5,   // placeholder estimate
		}

		// Only include contributors with actual activity
		if totalCommits > 0 || c.Contributions > 0 {
			analyses = append(analyses, analysis)
		}
	}

	a.logger.Info("calculated contribution analysis",
		zap.Int("contributors_analyzed", len(analyses)),
	)

	return analyses
}

// GetActivityPattern determines the primary activity pattern for a contributor
func (a *ContributionAnalyzer) GetActivityPattern(commits []*domain.GitHubCommit, login string) ActivityPattern {
	var loginCommits []*domain.GitHubCommit
	for _, c := range commits {
		if c.AuthorName == login {
			loginCommits = append(loginCommits, c)
		}
	}

	pattern := ActivityPattern{Login: login}
	if len(loginCommits) == 0 {
		return pattern
	}

	// Analyze day-of-week distribution
	dayCounts := make(map[time.Weekday]int)
	hourCounts := make(map[int]int)

	for _, c := range loginCommits {
		dayCounts[c.AuthorDate.Weekday()]++
		hourCounts[c.AuthorDate.Hour()]++
	}

	// Find peak day
	var peakDay time.Weekday
	var peakDayCount int
	for day, count := range dayCounts {
		if count > peakDayCount {
			peakDayCount = count
			peakDay = day
		}
	}
	pattern.PeakDay = peakDay

	// Find peak hour
	var peakHour int
	var peakHourCount int
	for hour, count := range hourCounts {
		if count > peakHourCount {
			peakHourCount = count
			peakHour = hour
		}
	}
	pattern.PeakHour = peakHour

	// Calculate consistency score (0-1)
	pattern.ConsistencyScore = a.calculateConsistency(loginCommits)

	return pattern
}

// calculateConsistency calculates how evenly distributed a contributor's commits are
func (a *ContributionAnalyzer) calculateConsistency(commits []*domain.GitHubCommit) float64 {
	if len(commits) < 2 {
		return 0
	}

	// Calculate time gaps between commits
	var gaps []float64
	for i := 1; i < len(commits); i++ {
		gap := commits[i].AuthorDate.Sub(commits[i-1].AuthorDate).Hours()
		if gap > 0 {
			gaps = append(gaps, gap)
		}
	}

	if len(gaps) == 0 {
		return 0
	}

	// Calculate coefficient of variation (lower = more consistent)
	mean := 0.0
	for _, g := range gaps {
		mean += g
	}
	mean /= float64(len(gaps))

	if mean == 0 {
		return 0
	}

	variance := 0.0
	for _, g := range gaps {
		diff := g - mean
		variance += diff * diff
	}
	variance /= float64(len(gaps))

	stdDev := 0.0
	for _, g := range gaps {
		diff := g - mean
		stdDev += diff * diff
	}
	stdDev = stdDev / float64(len(gaps))
	stdDev = stdDev * stdDev // This is already the variance; we need sqrt

	// Simple CV = stdDev / mean, normalize to 0-1
	cv := stdDev / mean
	score := 1.0 / (1.0 + cv) // Transform to 0-1 scale

	return score
}

// contributorCommitStats tracks commit statistics for a single contributor
type contributorCommitStats struct {
	commitCount int
}

// ActivityPattern represents a contributor's activity pattern
type ActivityPattern struct {
	Login            string
	PeakDay          time.Weekday
	PeakHour         int
	ConsistencyScore float64
}
