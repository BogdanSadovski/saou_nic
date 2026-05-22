package api

import (
	"github.com/gin-gonic/gin"
	"github.com/real-ass/admin-service/internal/config"
	"github.com/real-ass/admin-service/internal/domain"
	"github.com/real-ass/admin-service/pkg/rbac"
)

// SetupRouter configures the HTTP router with all routes and middleware.
func SetupRouter(
	handler *Handler,
	cfg *config.Config,
) *gin.Engine {
	router := gin.New()

	// Global middleware
	router.Use(RequestIDMiddleware())
	router.Use(LoggerMiddleware())
	router.Use(RecoveryMiddleware())
	router.Use(CORSMiddleware())

	// Health check (no auth required)
	router.GET("/health", handler.GetHealth)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Auth middleware
		auth := NewAuthMiddleware(cfg.JWT.Secret)
		rbacMiddleware := NewRBACMiddleware(cfg.RBAC.Enabled)

		// Apply authentication to all v1 routes
		v1.Use(auth.Authenticate())

		// Dashboard routes — admin-only. Без middleware на /stats
		// любой авторизованный пользователь видел админ-метрики.
		dashboard := v1.Group("/dashboard")
		{
			dashboard.GET("/stats", rbacMiddleware.RequireRole(domain.RoleAdmin), handler.GetDashboardStats)
		}

		// User management routes
		users := v1.Group("/users")
		{
			users.GET("", rbacMiddleware.RequirePermission(rbac.ResourceUsers, rbac.ActionList), handler.ListUsers)
			users.POST("", rbacMiddleware.RequirePermission(rbac.ResourceUsers, rbac.ActionCreate), handler.CreateUser)
			users.POST("/bulk", rbacMiddleware.RequirePermission(rbac.ResourceUsers, rbac.ActionUpdate), handler.BulkUpdateUsers)
			users.GET("/export", rbacMiddleware.RequirePermission(rbac.ResourceUsers, rbac.ActionExport), handler.ExportUsers)
			users.GET("/:id", rbacMiddleware.RequirePermission(rbac.ResourceUsers, rbac.ActionRead), handler.GetUser)
			users.PUT("/:id", rbacMiddleware.RequirePermission(rbac.ResourceUsers, rbac.ActionUpdate), handler.UpdateUser)
			users.DELETE("/:id", rbacMiddleware.RequirePermission(rbac.ResourceUsers, rbac.ActionDelete), handler.DeleteUser)
			users.POST("/:id/suspend", rbacMiddleware.RequirePermission(rbac.ResourceUsers, rbac.ActionUpdate), handler.SuspendUser)
			users.POST("/:id/ban", rbacMiddleware.RequirePermission(rbac.ResourceUsers, rbac.ActionUpdate), handler.BanUser)
			users.POST("/:id/activate", rbacMiddleware.RequirePermission(rbac.ResourceUsers, rbac.ActionUpdate), handler.ActivateUser)
			users.POST("/:id/role", rbacMiddleware.RequirePermission(rbac.ResourceUsers, rbac.ActionUpdate), handler.ChangeUserRole)
		}

		// User-facing billing endpoints (no RBAC) — every authenticated
		// user can manage their own subscription.
		billing := v1.Group("/billing/me")
		{
			billing.GET("/subscription", handler.GetMySubscription)
			billing.POST("/subscription", handler.CreateMySubscription)
			billing.DELETE("/subscription", handler.CancelMySubscription)
		}

		// Subscription management routes
		subscriptions := v1.Group("/subscriptions")
		{
			subscriptions.GET("", rbacMiddleware.RequirePermission(rbac.ResourceSubscriptions, rbac.ActionList), handler.ListSubscriptions)
			subscriptions.POST("/expire", rbacMiddleware.RequirePermission(rbac.ResourceSubscriptions, rbac.ActionUpdate), handler.ExpireSubscriptions)
			subscriptions.POST("/user/:user_id", rbacMiddleware.RequirePermission(rbac.ResourceSubscriptions, rbac.ActionCreate), handler.CreateSubscription)
			subscriptions.GET("/user/:user_id", rbacMiddleware.RequirePermission(rbac.ResourceSubscriptions, rbac.ActionRead), handler.GetUserSubscription)
			subscriptions.GET("/:id", rbacMiddleware.RequirePermission(rbac.ResourceSubscriptions, rbac.ActionRead), handler.GetSubscription)
			subscriptions.PUT("/:id", rbacMiddleware.RequirePermission(rbac.ResourceSubscriptions, rbac.ActionUpdate), handler.UpdateSubscription)
			subscriptions.POST("/:id/cancel", rbacMiddleware.RequirePermission(rbac.ResourceSubscriptions, rbac.ActionUpdate), handler.CancelSubscription)
			subscriptions.POST("/:id/renew", rbacMiddleware.RequirePermission(rbac.ResourceSubscriptions, rbac.ActionUpdate), handler.RenewSubscription)
		}

		// Audit log routes
		auditLogs := v1.Group("/audit-logs")
		{
			auditLogs.GET("", rbacMiddleware.RequirePermission(rbac.ResourceAuditLogs, rbac.ActionList), handler.ListAuditLogs)
			auditLogs.GET("/:id", rbacMiddleware.RequirePermission(rbac.ResourceAuditLogs, rbac.ActionRead), handler.GetAuditLog)
			auditLogs.POST("/cleanup", rbacMiddleware.RequirePermission(rbac.ResourceAuditLogs, rbac.ActionDelete), handler.CleanupAuditLogs)
			auditLogs.GET("/admins/:admin_id/activity", rbacMiddleware.RequirePermission(rbac.ResourceAuditLogs, rbac.ActionRead), handler.GetAdminActivity)
		}
	}

	return router
}
