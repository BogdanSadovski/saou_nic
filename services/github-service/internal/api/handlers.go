package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/real-ass/github-service/internal/service"
)

type Handlers struct {
	githubService *service.GitHubService
	logger        *zap.Logger
}

func NewHandlers(githubService *service.GitHubService, logger *zap.Logger) *Handlers {
	return &Handlers{
		githubService: githubService,
		logger:        logger,
	}
}

// SyncRepository syncs a repository from GitHub
// POST /api/v1/repositories/sync/:owner/:name
func (h *Handlers) SyncRepository(c *gin.Context) {
	owner := c.Param("owner")
	name := c.Param("name")

	if owner == "" || name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "owner and name are required"})
		return
	}

	repo, err := h.githubService.SyncRepository(c.Request.Context(), owner, name)
	if err != nil {
		h.logger.Error("failed to sync repository",
			zap.String("owner", owner),
			zap.String("name", name),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to sync repository"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": repo})
}

// GetRepository retrieves a repository
// GET /api/v1/repositories/:id
func (h *Handlers) GetRepository(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository id"})
		return
	}

	repo, err := h.githubService.GetRepository(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": repo})
}

// ListRepositories lists all repositories
// GET /api/v1/repositories
func (h *Handlers) ListRepositories(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	repos, err := h.githubService.ListRepositories(c.Request.Context(), limit, offset)
	if err != nil {
		h.logger.Error("failed to list repositories", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list repositories"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": repos,
		"pagination": gin.H{
			"limit":  limit,
			"offset": offset,
		},
	})
}

// SyncContributors syncs contributors for a repository
// POST /api/v1/repositories/:id/contributors/sync
func (h *Handlers) SyncContributors(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository id"})
		return
	}

	owner := c.Query("owner")
	name := c.Query("name")
	if owner == "" || name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "owner and name query params are required"})
		return
	}

	if err := h.githubService.SyncContributors(c.Request.Context(), id, owner, name); err != nil {
		h.logger.Error("failed to sync contributors",
			zap.Int64("repo_id", id),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to sync contributors"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "contributors synced successfully"})
}

// GetTopContributors retrieves top contributors for a repository
// GET /api/v1/repositories/:id/contributors/top
func (h *Handlers) GetTopContributors(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository id"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if limit <= 0 || limit > 100 {
		limit = 10
	}

	contributors, err := h.githubService.GetTopContributors(c.Request.Context(), id, limit)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "contributors not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": contributors})
}

// SyncPullRequests syncs pull requests for a repository
// POST /api/v1/repositories/:id/pull-requests/sync
func (h *Handlers) SyncPullRequests(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository id"})
		return
	}

	owner := c.Query("owner")
	name := c.Query("name")
	if owner == "" || name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "owner and name query params are required"})
		return
	}

	if err := h.githubService.SyncPullRequests(c.Request.Context(), id, owner, name); err != nil {
		h.logger.Error("failed to sync pull requests",
			zap.Int64("repo_id", id),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to sync pull requests"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "pull requests synced successfully"})
}

// AnalyzeRepository runs a full analysis on a repository
// POST /api/v1/repositories/:id/analyze
func (h *Handlers) AnalyzeRepository(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository id"})
		return
	}

	metrics, err := h.githubService.AnalyzeRepository(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("failed to analyze repository",
			zap.Int64("repo_id", id),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to analyze repository"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": metrics})
}

// AnalyzeContributions runs a contribution analysis for a repository
// POST /api/v1/repositories/:id/contributions/analyze
func (h *Handlers) AnalyzeContributions(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository id"})
		return
	}

	analyses, err := h.githubService.AnalyzeContributions(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("failed to analyze contributions",
			zap.Int64("repo_id", id),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to analyze contributions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": analyses})
}

// HealthCheck returns the health status of the service
// GET /health
func (h *Handlers) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"service":   "github-service",
	})
}

// ReadyCheck returns the readiness status
// GET /ready
func (h *Handlers) ReadyCheck(c *gin.Context) {
	// In a real implementation, this would check database connectivity
	// and external service availability
	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
	})
}
