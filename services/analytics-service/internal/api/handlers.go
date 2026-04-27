package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"analytics-service/internal/domain"
	"analytics-service/internal/service"
)

// Handler holds dependencies for HTTP request handling.
type Handler struct {
	analyticsSvc *service.AnalyticsService
	dashboardSvc *service.DashboardService
	exportSvc    *service.ExportService
}

// NewHandler creates a new Handler with the given services.
func NewHandler(
	analyticsSvc *service.AnalyticsService,
	dashboardSvc *service.DashboardService,
	exportSvc *service.ExportService,
) *Handler {
	return &Handler{
		analyticsSvc: analyticsSvc,
		dashboardSvc: dashboardSvc,
		exportSvc:    exportSvc,
	}
}

// Response helpers.

type successResponse struct {
	Data   interface{} `json:"data"`
	Total  int64       `json:"total,omitempty"`
	Page   int         `json:"page,omitempty"`
	PerPage int        `json:"per_page,omitempty"`
}

type errorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err string) {
	writeJSON(w, status, errorResponse{Error: err})
}

func writeErrorWithMessage(w http.ResponseWriter, status int, err, msg string) {
	writeJSON(w, status, errorResponse{Error: err, Message: msg})
}

func extractTenantID(r *http.Request) string {
	// Extract from header or context.
	if id := r.Header.Get("X-Tenant-ID"); id != "" {
		return id
	}
	if id := r.URL.Query().Get("tenant_id"); id != "" {
		return id
	}
	return ""
}

func parsePagination(r *http.Request) (limit, offset int) {
	limit, _ = strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ = strconv.Atoi(r.URL.Query().Get("offset"))

	if limit <= 0 {
		limit = 50
	}
	if limit > 1000 {
		limit = 1000
	}
	if offset < 0 {
		offset = 0
	}

	return limit, offset
}

// HealthCheck returns service health status.
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"service":   "analytics-service",
	})
}

// ==================== Event Handlers ====================

// IngestEvent handles a single event ingestion.
func (h *Handler) IngestEvent(w http.ResponseWriter, r *http.Request) {
	var event domain.Event
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if event.Type == "" {
		writeError(w, http.StatusBadRequest, "event type is required")
		return
	}

	if event.SessionID == "" {
		writeError(w, http.StatusBadRequest, "session_id is required")
		return
	}

	// Generate ID if not provided.
	if event.ID == "" {
		event.ID = generateID()
	}

	// Set tenant from header.
	if event.TenantID == "" {
		event.TenantID = extractTenantID(r)
	}

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	if err := h.analyticsSvc.ProcessEvent(r.Context(), &event); err != nil {
		writeErrorWithMessage(w, http.StatusInternalServerError, "failed to process event", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, successResponse{Data: map[string]string{"id": event.ID}})
}

// BatchIngestEvents handles batch event ingestion.
func (h *Handler) BatchIngestEvents(w http.ResponseWriter, r *http.Request) {
	var events []*domain.Event
	if err := json.NewDecoder(r.Body).Decode(&events); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(events) == 0 {
		writeError(w, http.StatusBadRequest, "no events provided")
		return
	}

	tenantID := extractTenantID(r)
	now := time.Now().UTC()

	for _, e := range events {
		if e.TenantID == "" {
			e.TenantID = tenantID
		}
		if e.ID == "" {
			e.ID = generateID()
		}
		if e.Timestamp.IsZero() {
			e.Timestamp = now
		}
	}

	if err := h.analyticsSvc.ProcessEvents(r.Context(), events); err != nil {
		writeErrorWithMessage(w, http.StatusInternalServerError, "failed to process events", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, successResponse{
		Data: map[string]interface{}{
			"ingested": len(events),
		},
	})
}

// QueryEvents returns events matching query parameters.
func (h *Handler) QueryEvents(w http.ResponseWriter, r *http.Request) {
	tenantID := extractTenantID(r)
	if tenantID == "" {
		writeError(w, http.StatusBadRequest, "tenant_id is required")
		return
	}

	filter := parseQueryFilter(r, tenantID)
	limit, offset := parsePagination(r)
	filter.Limit = limit
	filter.Offset = offset

	events, total, err := h.analyticsSvc.GetEvents(r.Context(), filter)
	if err != nil {
		writeErrorWithMessage(w, http.StatusInternalServerError, "failed to query events", err.Error())
		return
	}

	page := 1
	if offset > 0 && limit > 0 {
		page = (offset / limit) + 1
	}

	writeJSON(w, http.StatusOK, successResponse{
		Data:    events,
		Total:   total,
		Page:    page,
		PerPage: limit,
	})
}

// ==================== Metrics Handlers ====================

// GetTimeSeries returns time series data for metrics.
func (h *Handler) GetTimeSeries(w http.ResponseWriter, r *http.Request) {
	tenantID := extractTenantID(r)
	if tenantID == "" {
		writeError(w, http.StatusBadRequest, "tenant_id is required")
		return
	}

	filter := parseQueryFilter(r, tenantID)

	points, err := h.analyticsSvc.GetTimeSeries(r.Context(), filter)
	if err != nil {
		writeErrorWithMessage(w, http.StatusInternalServerError, "failed to get time series", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, successResponse{Data: points})
}

// GetMetricsSummary returns a summary of key metrics.
func (h *Handler) GetMetricsSummary(w http.ResponseWriter, r *http.Request) {
	tenantID := extractTenantID(r)
	if tenantID == "" {
		writeError(w, http.StatusBadRequest, "tenant_id is required")
		return
	}

	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	if from == "" {
		from = time.Now().UTC().AddDate(0, 0, -7).Format("2006-01-02")
	}
	if to == "" {
		to = time.Now().UTC().Format("2006-01-02")
	}

	summary, err := h.analyticsSvc.GetMetricsSummary(r.Context(), tenantID, from, to)
	if err != nil {
		writeErrorWithMessage(w, http.StatusInternalServerError, "failed to get metrics summary", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, successResponse{Data: summary})
}

// GetActiveUsers returns the count of active users.
func (h *Handler) GetActiveUsers(w http.ResponseWriter, r *http.Request) {
	tenantID := extractTenantID(r)
	if tenantID == "" {
		writeError(w, http.StatusBadRequest, "tenant_id is required")
		return
	}

	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	if from == "" {
		from = time.Now().UTC().AddDate(0, 0, -7).Format("2006-01-02")
	}
	if to == "" {
		to = time.Now().UTC().Format("2006-01-02")
	}

	count, err := h.analyticsSvc.GetActiveUsers(r.Context(), tenantID, from, to)
	if err != nil {
		writeErrorWithMessage(w, http.StatusInternalServerError, "failed to get active users", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, successResponse{Data: map[string]int64{"active_users": count}})
}

// GetTopUsers returns users ranked by activity.
func (h *Handler) GetTopUsers(w http.ResponseWriter, r *http.Request) {
	tenantID := extractTenantID(r)
	if tenantID == "" {
		writeError(w, http.StatusBadRequest, "tenant_id is required")
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 10
	}

	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	if from == "" {
		from = time.Now().UTC().AddDate(0, 0, -30).Format("2006-01-02")
	}
	if to == "" {
		to = time.Now().UTC().Format("2006-01-02")
	}

	users, err := h.analyticsSvc.GetTopUsers(r.Context(), tenantID, from, to, limit)
	if err != nil {
		writeErrorWithMessage(w, http.StatusInternalServerError, "failed to get top users", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, successResponse{Data: users})
}

// GetRealtimeMetrics returns metrics for the recent real-time window.
func (h *Handler) GetRealtimeMetrics(w http.ResponseWriter, r *http.Request) {
	tenantID := extractTenantID(r)
	if tenantID == "" {
		writeError(w, http.StatusBadRequest, "tenant_id is required")
		return
	}

	windowMinutes, _ := strconv.Atoi(r.URL.Query().Get("window"))
	if windowMinutes <= 0 {
		windowMinutes = 15
	}

	metrics, err := h.analyticsSvc.GetRealtimeMetrics(r.Context(), tenantID, windowMinutes)
	if err != nil {
		writeErrorWithMessage(w, http.StatusInternalServerError, "failed to get realtime metrics", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, successResponse{Data: metrics})
}

// ==================== Dashboard Handlers ====================

// CreateDashboard creates a new dashboard.
func (h *Handler) CreateDashboard(w http.ResponseWriter, r *http.Request) {
	var dashboard domain.Dashboard
	if err := json.NewDecoder(r.Body).Decode(&dashboard); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	dashboard.TenantID = extractTenantID(r)
	dashboard.CreatedBy = r.Header.Get("X-User-ID")

	created, err := h.dashboardSvc.CreateDashboard(r.Context(), &dashboard)
	if err != nil {
		writeErrorWithMessage(w, http.StatusInternalServerError, "failed to create dashboard", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, successResponse{Data: created})
}

// GetDashboard retrieves a dashboard by ID.
func (h *Handler) GetDashboard(w http.ResponseWriter, r *http.Request, id string) {
	dashboard, err := h.dashboardSvc.GetDashboard(r.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "dashboard not found")
			return
		}
		writeErrorWithMessage(w, http.StatusInternalServerError, "failed to get dashboard", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, successResponse{Data: dashboard})
}

// ListDashboards returns all dashboards for a tenant.
func (h *Handler) ListDashboards(w http.ResponseWriter, r *http.Request) {
	tenantID := extractTenantID(r)
	if tenantID == "" {
		writeError(w, http.StatusBadRequest, "tenant_id is required")
		return
	}

	dashboards, err := h.dashboardSvc.ListDashboards(r.Context(), tenantID)
	if err != nil {
		writeErrorWithMessage(w, http.StatusInternalServerError, "failed to list dashboards", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, successResponse{Data: dashboards})
}

// UpdateDashboard modifies an existing dashboard.
func (h *Handler) UpdateDashboard(w http.ResponseWriter, r *http.Request, id string) {
	var dashboard domain.Dashboard
	if err := json.NewDecoder(r.Body).Decode(&dashboard); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	dashboard.ID = id
	updated, err := h.dashboardSvc.UpdateDashboard(r.Context(), &dashboard)
	if err != nil {
		writeErrorWithMessage(w, http.StatusInternalServerError, "failed to update dashboard", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, successResponse{Data: updated})
}

// DeleteDashboard removes a dashboard.
func (h *Handler) DeleteDashboard(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.dashboardSvc.DeleteDashboard(r.Context(), id); err != nil {
		writeErrorWithMessage(w, http.StatusInternalServerError, "failed to delete dashboard", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, successResponse{Data: map[string]string{"status": "deleted"}})
}

// GetDashboardWidgetData returns data for a specific widget.
func (h *Handler) GetDashboardWidgetData(w http.ResponseWriter, r *http.Request, dashboardID, widgetID string) {
	dashboard, err := h.dashboardSvc.GetDashboard(r.Context(), dashboardID)
	if err != nil {
		writeError(w, http.StatusNotFound, "dashboard not found")
		return
	}

	var widget *domain.Widget
	for i := range dashboard.Widgets {
		if dashboard.Widgets[i].ID == widgetID {
			widget = &dashboard.Widgets[i]
			break
		}
	}

	if widget == nil {
		writeError(w, http.StatusNotFound, "widget not found")
		return
	}

	dateFrom := r.URL.Query().Get("from")
	dateTo := r.URL.Query().Get("to")

	if dateFrom == "" {
		dateFrom = time.Now().UTC().AddDate(0, 0, -7).Format("2006-01-02")
	}
	if dateTo == "" {
		dateTo = time.Now().UTC().Format("2006-01-02")
	}

	data, err := h.dashboardSvc.GetWidgetData(r.Context(), *widget, dashboard.TenantID, dateFrom, dateTo)
	if err != nil {
		writeErrorWithMessage(w, http.StatusInternalServerError, "failed to get widget data", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, successResponse{Data: data})
}

// ==================== Export Handlers ====================

// CreateExport initiates a data export.
func (h *Handler) CreateExport(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Format string          `json:"format"`
		Filter domain.QueryFilter `json:"filter"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tenantID := extractTenantID(r)
	if tenantID == "" {
		writeError(w, http.StatusBadRequest, "tenant_id is required")
		return
	}

	if req.Format == "" {
		req.Format = "csv"
	}

	export, err := h.exportSvc.CreateExportRequest(r.Context(), tenantID, req.Format, req.Filter)
	if err != nil {
		writeErrorWithMessage(w, http.StatusInternalServerError, "failed to create export", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, successResponse{Data: export})
}

// GetExportStatus returns the status of an export request.
func (h *Handler) GetExportStatus(w http.ResponseWriter, r *http.Request, id string) {
	export, err := h.exportSvc.GetExportStatus(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "export not found")
		return
	}

	writeJSON(w, http.StatusOK, successResponse{Data: export})
}

// ListExports returns export requests for a tenant.
func (h *Handler) ListExports(w http.ResponseWriter, r *http.Request) {
	tenantID := extractTenantID(r)
	if tenantID == "" {
		writeError(w, http.StatusBadRequest, "tenant_id is required")
		return
	}

	limit, offset := parsePagination(r)

	exports, err := h.exportSvc.ListExports(r.Context(), tenantID, limit, offset)
	if err != nil {
		writeErrorWithMessage(w, http.StatusInternalServerError, "failed to list exports", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, successResponse{Data: exports})
}

// GetSupportedFormats returns available export formats.
func (h *Handler) GetSupportedFormats(w http.ResponseWriter, r *http.Request) {
	formats := h.exportSvc.GetSupportedFormats()
	writeJSON(w, http.StatusOK, successResponse{Data: formats})
}

// ==================== Helpers ====================

func parseQueryFilter(r *http.Request, tenantID string) domain.QueryFilter {
	q := r.URL.Query()

	filter := domain.QueryFilter{
		TenantID: tenantID,
	}

	if t := q.Get("type"); t != "" {
		filter.EventTypes = []domain.EventType{domain.EventType(t)}
	}
	if types := q.Get("types"); types != "" {
		for _, t := range strings.Split(types, ",") {
			filter.EventTypes = append(filter.EventTypes, domain.EventType(t))
		}
	}

	filter.UserID = q.Get("user_id")
	filter.SessionID = q.Get("session_id")
	filter.URL = q.Get("url")
	filter.Country = q.Get("country")
	filter.Device = q.Get("device")
	filter.OS = q.Get("os")
	filter.Browser = q.Get("browser")
	filter.Granularity = q.Get("granularity")

	if from := q.Get("from"); from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			filter.DateFrom = t
		} else if t, err := time.Parse("2006-01-02", from); err == nil {
			filter.DateFrom = t
		}
	}

	if to := q.Get("to"); to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			filter.DateTo = t
		} else if t, err := time.Parse("2006-01-02", to); err == nil {
			filter.DateTo = t
		}
	}

	if filter.DateFrom.IsZero() {
		filter.DateFrom = time.Now().UTC().AddDate(0, 0, -7)
	}
	if filter.DateTo.IsZero() {
		filter.DateTo = time.Now().UTC()
	}

	return filter
}

func generateID() string {
	return time.Now().UTC().Format("20060102150405.000000000")
}

// Middleware helpers.

// RequireTenantID ensures a tenant ID is present in the request.
func RequireTenantID(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if extractTenantID(r) == "" {
			writeError(w, http.StatusUnauthorized, "missing tenant_id")
			return
		}
		next(w, r)
	}
}

// CORSMiddleware adds CORS headers to responses.
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Tenant-ID, X-User-ID")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// LoggingMiddleware logs request details.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Simple response writer wrapper for status code capture.
		next.ServeHTTP(w, r)

		_ = context.Background()
		_ = start
		// In production, use a proper logger (e.g., zap, logrus).
	})
}
