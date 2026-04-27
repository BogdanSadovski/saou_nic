package domain

import "time"

type GitHubRepository struct {
	ID                int64     `json:"id" db:"id"`
	Owner             string    `json:"owner" db:"owner"`
	Name              string    `json:"name" db:"name"`
	FullName          string    `json:"full_name" db:"full_name"`
	Description       string    `json:"description" db:"description"`
	HTMLURL           string    `json:"html_url" db:"html_url"`
	DefaultBranch     string    `json:"default_branch" db:"default_branch"`
	Language          string    `json:"language" db:"language"`
	StarsCount        int       `json:"stars_count" db:"stars_count"`
	ForksCount        int       `json:"forks_count" db:"forks_count"`
	OpenIssuesCount   int       `json:"open_issues_count" db:"open_issues_count"`
	IsPrivate         bool      `json:"is_private" db:"is_private"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
	PushedAt          time.Time `json:"pushed_at" db:"pushed_at"`
	LastSyncedAt      time.Time `json:"last_synced_at" db:"last_synced_at"`
}

type GitHubContributor struct {
	ID             int64  `json:"id" db:"id"`
	Login          string `json:"login" db:"login"`
	AvatarURL      string `json:"avatar_url" db:"avatar_url"`
	HTMLURL        string `json:"html_url" db:"html_url"`
	Contributions  int    `json:"contributions" db:"contributions"`
	RepositoryID   int64  `json:"repository_id" db:"repository_id"`
}

type GitHubCommit struct {
	ID             int64     `json:"id" db:"id"`
	SHA            string    `json:"sha" db:"sha"`
	Message        string    `json:"message" db:"message"`
	AuthorName     string    `json:"author_name" db:"author_name"`
	AuthorEmail    string    `json:"author_email" db:"author_email"`
	AuthorDate     time.Time `json:"author_date" db:"author_date"`
	CommitterName  string    `json:"committer_name" db:"committer_name"`
	CommitterDate  time.Time `json:"committer_date" db:"committer_date"`
	RepositoryID   int64     `json:"repository_id" db:"repository_id"`
}

type GitHubPullRequest struct {
	ID        int64     `json:"id" db:"id"`
	Number    int       `json:"number" db:"number"`
	Title     string    `json:"title" db:"title"`
	State     string    `json:"state" db:"state"`
	UserLogin string    `json:"user_login" db:"user_login"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	MergedAt  time.Time `json:"merged_at" db:"merged_at"`
	ClosedAt  time.Time `json:"closed_at" db:"closed_at"`
	RepositoryID int64  `json:"repository_id" db:"repository_id"`
}

type RepositoryMetrics struct {
	RepositoryID      int64     `json:"repository_id" db:"repository_id"`
	CommitFrequency   float64   `json:"commit_frequency" db:"commit_frequency"`
	AvgPRSize         float64   `json:"avg_pr_size" db:"avg_pr_size"`
	MergeRate         float64   `json:"merge_rate" db:"merge_rate"`
	AvgMergeTime      float64   `json:"avg_merge_time" db:"avg_merge_time"`
	IssueResolutionRate float64 `json:"issue_resolution_rate" db:"issue_resolution_rate"`
	BusFactor         int       `json:"bus_factor" db:"bus_factor"`
	CalculatedAt      time.Time `json:"calculated_at" db:"calculated_at"`
}

type ContributionAnalysis struct {
	ID               int64     `json:"id" db:"id"`
	RepositoryID     int64     `json:"repository_id" db:"repository_id"`
	ContributorLogin string    `json:"contributor_login" db:"contributor_login"`
	TotalCommits     int       `json:"total_commits" db:"total_commits"`
	TotalAdditions   int       `json:"total_additions" db:"total_additions"`
	TotalDeletions   int       `json:"total_deletions" db:"total_deletions"`
	FilesChanged     int       `json:"files_changed" db:"files_changed"`
	ReviewCount      int       `json:"review_count" db:"review_count"`
	AnalysisDate     time.Time `json:"analysis_date" db:"analysis_date"`
}

type SyncStatus struct {
	ID         int64     `json:"id" db:"id"`
	EntityType string    `json:"entity_type" db:"entity_type"`
	EntityID   int64     `json:"entity_id" db:"entity_id"`
	Status     string    `json:"status" db:"status"`
	LastSyncAt time.Time `json:"last_sync_at" db:"last_sync_at"`
	Error      string    `json:"error" db:"error"`
}
