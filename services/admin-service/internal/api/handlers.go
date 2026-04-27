package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/real-ass/admin-service/internal/domain"
	"github.com/real-ass/admin-service/internal/service"
)

// Handler holds dependencies for HTTP handlers.
type Handler struct {
	adminService    *service.AdminService
	userService     *service.UserService
	subService      *service.SubscriptionService
	auditService    *service.AuditService
}

// NewHandler creates a new Handler.
func NewHandler(
	adminService *service.AdminService,
	userService *service.UserService,
	subService *service.SubscriptionService,
	auditService *service.AuditService,
) *Handler {
	return &Handler{
		adminService: adminService,
		userService:  userService,
		subService:   subService,
		auditService: auditService,
	}
}

// ==================== Dashboard Handlers ====================

// GetDashboardStats returns dashboard statistics.
func (h *Handler) GetDashboardStats(c *gin.Context) {
	stats, err := h.adminService.GetDashboardStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get dashboard stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetHealth returns system health status.
func (h *Handler) GetHealth(c *gin.Context) {
	health := h.adminService.GetSystemHealth(c.Request.Context())
	c.JSON(http.StatusOK, health)
}

// ==================== User Handlers ====================

// CreateUser handles user creation.
func (h *Handler) CreateUser(c *gin.Context) {
	var req domain.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	adminID, err := GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	user, err := h.userService.CreateUser(c.Request.Context(), req, adminID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, user)
}

// GetUser retrieves a user by ID.
func (h *Handler) GetUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	user, err := h.userService.GetUser(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// UpdateUser handles user updates.
func (h *Handler) UpdateUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	var req domain.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	adminID, err := GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	user, err := h.userService.UpdateUser(c.Request.Context(), id, req, adminID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

// DeleteUser handles user deletion.
func (h *Handler) DeleteUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	adminID, err := GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if err := h.userService.DeleteUser(c.Request.Context(), id, adminID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user deleted successfully"})
}

// ListUsers returns a paginated list of users.
func (h *Handler) ListUsers(c *gin.Context) {
	var query domain.ListUsersQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid query parameters", "details": err.Error()})
		return
	}

	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize < 1 {
		query.PageSize = 20
	}
	if query.PageSize > 100 {
		query.PageSize = 100
	}

	users, total, err := h.userService.ListUsers(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list users"})
		return
	}

	pagination := service.CalculatePagination(total, query.Page, query.PageSize)
	pagination.Items = users

	c.JSON(http.StatusOK, pagination)
}

// SuspendUser suspends a user account.
func (h *Handler) SuspendUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	adminID, err := GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if err := h.userService.SuspendUser(c.Request.Context(), id, adminID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user suspended successfully"})
}

// BanUser bans a user account.
func (h *Handler) BanUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	adminID, err := GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if err := h.userService.BanUser(c.Request.Context(), id, adminID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user banned successfully"})
}

// ActivateUser activates a user account.
func (h *Handler) ActivateUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	adminID, err := GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if err := h.userService.ActivateUser(c.Request.Context(), id, adminID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user activated successfully"})
}

// ChangeUserRole changes a user's role.
func (h *Handler) ChangeUserRole(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	var req struct {
		Role domain.UserRole `json:"role" binding:"required,oneof=super_admin admin moderator user"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	adminID, err := GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if err := h.userService.ChangeUserRole(c.Request.Context(), id, req.Role, adminID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user role changed successfully"})
}

// BulkUpdateUsers handles bulk user status updates.
func (h *Handler) BulkUpdateUsers(c *gin.Context) {
	var req struct {
		UserIDs []uuid.UUID      `json:"user_ids" binding:"required"`
		Status  domain.UserStatus `json:"status" binding:"required,oneof=active inactive suspended banned"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	updated, err := h.adminService.BulkUpdateStatus(c.Request.Context(), req.UserIDs, req.Status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to bulk update users"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "bulk update completed",
		"updated": updated,
		"total":   len(req.UserIDs),
	})
}

// ExportUsers exports user data.
func (h *Handler) ExportUsers(c *gin.Context) {
	var query domain.ListUsersQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid query parameters", "details": err.Error()})
		return
	}

	users, err := h.adminService.ExportUsers(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to export users"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"count": len(users),
	})
}

// ==================== Subscription Handlers ====================

// CreateSubscription creates a new subscription.
func (h *Handler) CreateSubscription(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	var req struct {
		Tier domain.SubscriptionTier `json:"tier" binding:"required,oneof=free basic pro enterprise"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	adminID, err := GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	sub, err := h.subService.CreateSubscription(c.Request.Context(), userID, req.Tier, adminID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, sub)
}

// GetSubscription retrieves a subscription by ID.
func (h *Handler) GetSubscription(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subscription ID"})
		return
	}

	sub, err := h.subService.GetSubscription(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
		return
	}

	c.JSON(http.StatusOK, sub)
}

// GetUserSubscription retrieves a user's subscription.
func (h *Handler) GetUserSubscription(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	sub, err := h.subService.GetSubscriptionByUserID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
		return
	}

	c.JSON(http.StatusOK, sub)
}

// UpdateSubscription updates a subscription.
func (h *Handler) UpdateSubscription(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subscription ID"})
		return
	}

	var req struct {
		Tier      domain.SubscriptionTier `json:"tier" binding:"required,oneof=free basic pro enterprise"`
		AutoRenew bool                    `json:"auto_renew"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	adminID, err := GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	sub, err := h.subService.UpdateSubscription(c.Request.Context(), id, req.Tier, req.AutoRenew, adminID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, sub)
}

// CancelSubscription cancels a subscription.
func (h *Handler) CancelSubscription(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subscription ID"})
		return
	}

	adminID, err := GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if err := h.subService.CancelSubscription(c.Request.Context(), id, adminID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "subscription canceled successfully"})
}

// RenewSubscription renews a subscription.
func (h *Handler) RenewSubscription(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subscription ID"})
		return
	}

	adminID, err := GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	sub, err := h.subService.RenewSubscription(c.Request.Context(), id, adminID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, sub)
}

// ListSubscriptions lists subscriptions by status.
func (h *Handler) ListSubscriptions(c *gin.Context) {
	status := c.Query("status")
	if status == "" {
		status = string(domain.SubscriptionActive)
	}

	subs, err := h.subService.ListSubscriptionsByStatus(c.Request.Context(), domain.SubscriptionStatus(status))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list subscriptions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"subscriptions": subs,
		"count":         len(subs),
	})
}

// ExpireSubscriptions triggers expiration of old subscriptions.
func (h *Handler) ExpireSubscriptions(c *gin.Context) {
	adminID, err := GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	expired, err := h.subService.ExpireOldSubscriptions(c.Request.Context(), adminID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to expire subscriptions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "subscription expiration completed",
		"expired": expired,
	})
}

// ==================== Audit Log Handlers ====================

// GetAuditLog retrieves an audit log entry.
func (h *Handler) GetAuditLog(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid audit log ID"})
		return
	}

	log, err := h.auditService.GetAuditLog(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "audit log not found"})
		return
	}

	c.JSON(http.StatusOK, log)
}

// ListAuditLogs lists audit logs with filters.
func (h *Handler) ListAuditLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	filters := domain.AuditLogFilters{
		Page:         page,
		PageSize:     pageSize,
		ResourceType: c.Query("resource_type"),
	}

	if action := c.Query("action"); action != "" {
		a := domain.AuditAction(action)
		filters.Action = &a
	}

	if startDate := c.Query("start_date"); startDate != "" {
		filters.StartDate = &startDate
	}

	if endDate := c.Query("end_date"); endDate != "" {
		filters.EndDate = &endDate
	}

	logs, total, err := h.auditService.ListAuditLogs(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list audit logs"})
		return
	}

	pagination := service.CalculatePagination(total, page, pageSize)
	pagination.Items = logs

	c.JSON(http.StatusOK, pagination)
}

// GetAdminActivity retrieves activity summary for an admin.
func (h *Handler) GetAdminActivity(c *gin.Context) {
	adminID, err := uuid.Parse(c.Param("admin_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid admin ID"})
		return
	}

	summary, err := h.auditService.GetAdminActivitySummary(c.Request.Context(), adminID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get admin activity"})
		return
	}

	c.JSON(http.StatusOK, summary)
}

// CleanupAuditLogs triggers cleanup of old audit logs.
func (h *Handler) CleanupAuditLogs(c *gin.Context) {
	var req struct {
		Days int `json:"days" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	adminID, err := GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	deleted, err := h.auditService.CleanupOldLogs(c.Request.Context(), req.Days, adminID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cleanup audit logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "audit log cleanup completed",
		"deleted": deleted,
	})
}
