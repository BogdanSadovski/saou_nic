package api

import (
	"net/http"

	"analytics-service/internal/config"
)

// Router sets up HTTP routes for the analytics service.
type Router struct {
	handler *Handler
	cfg     config.ServerConfig
	mux     *http.ServeMux
}

// NewRouter creates a new Router with all routes registered.
func NewRouter(handler *Handler, cfg config.ServerConfig) *Router {
	r := &Router{
		handler: handler,
		cfg:     cfg,
		mux:     http.NewServeMux(),
	}

	r.registerRoutes()
	return r
}

// ServeMux returns the configured http.ServeMux.
func (r *Router) ServeMux() *http.ServeMux {
	return r.mux
}

// Handler returns the mux wrapped with middleware.
func (r *Router) Handler() http.Handler {
	var h http.Handler = r.mux
	h = CORSMiddleware(h)
	h = LoggingMiddleware(h)
	return h
}

func (r *Router) registerRoutes() {
	// Health and metadata.
	r.mux.HandleFunc("/health", r.handler.HealthCheck)
	r.mux.HandleFunc("/api/v1/formats", r.handler.GetSupportedFormats)

	// Event endpoints.
	r.mux.HandleFunc("/api/v1/events", r.requireTenant(r.handler.IngestEvent, http.MethodPost, http.MethodOptions))
	r.mux.HandleFunc("/api/v1/events/batch", r.requireTenant(r.handler.BatchIngestEvents, http.MethodPost, http.MethodOptions))
	r.mux.HandleFunc("/api/v1/events/query", r.requireTenant(r.handler.QueryEvents, http.MethodGet, http.MethodOptions))

	// Metrics endpoints.
	r.mux.HandleFunc("/api/v1/metrics/timeseries", r.requireTenant(r.handler.GetTimeSeries, http.MethodGet, http.MethodOptions))
	r.mux.HandleFunc("/api/v1/metrics/summary", r.requireTenant(r.handler.GetMetricsSummary, http.MethodGet, http.MethodOptions))
	r.mux.HandleFunc("/api/v1/metrics/active-users", r.requireTenant(r.handler.GetActiveUsers, http.MethodGet, http.MethodOptions))
	r.mux.HandleFunc("/api/v1/metrics/top-users", r.requireTenant(r.handler.GetTopUsers, http.MethodGet, http.MethodOptions))
	r.mux.HandleFunc("/api/v1/metrics/realtime", r.requireTenant(r.handler.GetRealtimeMetrics, http.MethodGet, http.MethodOptions))

	// Dashboard endpoints.
	r.mux.HandleFunc("/api/v1/dashboards", r.requireTenant(r.handler.ListDashboards, http.MethodGet, http.MethodOptions))
	r.mux.HandleFunc("/api/v1/dashboards/create", r.requireTenant(r.handler.CreateDashboard, http.MethodPost, http.MethodOptions))

	// Dashboard detail endpoints (with path params).
	r.mux.HandleFunc("/api/v1/dashboards/{id}", r.dashboardDetailHandler)
	r.mux.HandleFunc("/api/v1/dashboards/{dashboardId}/widgets/{widgetId}/data", r.requireTenant(r.dashboardWidgetDataHandler, http.MethodGet, http.MethodOptions))

	// Export endpoints.
	r.mux.HandleFunc("/api/v1/exports", r.requireTenant(r.handler.ListExports, http.MethodGet, http.MethodOptions))
	r.mux.HandleFunc("/api/v1/exports/create", r.requireTenant(r.handler.CreateExport, http.MethodPost, http.MethodOptions))
	r.mux.HandleFunc("/api/v1/exports/{id}", r.requireTenant(r.exportDetailHandler, http.MethodGet, http.MethodOptions))
}

// requireTenant wraps a handler with tenant requirement for specific methods.
func (r *Router) requireTenant(fn http.HandlerFunc, allowedMethods ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		// Check allowed methods.
		methodAllowed := false
		for _, m := range allowedMethods {
			if req.Method == m {
				methodAllowed = true
				break
			}
		}
		if !methodAllowed {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// For OPTIONS (preflight), skip tenant check.
		if req.Method == http.MethodOptions {
			fn(w, req)
			return
		}

		RequireTenantID(fn)(w, req)
	}
}

// dashboardDetailHandler routes to GET/PUT/DELETE based on method.
func (r *Router) dashboardDetailHandler(w http.ResponseWriter, req *http.Request) {
	id := req.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "dashboard id is required")
		return
	}

	switch req.Method {
	case http.MethodGet:
		RequireTenantID(func(w http.ResponseWriter, req2 *http.Request) {
			r.handler.GetDashboard(w, req2, id)
		})(w, req)
	case http.MethodPut:
		RequireTenantID(func(w http.ResponseWriter, req2 *http.Request) {
			r.handler.UpdateDashboard(w, req2, id)
		})(w, req)
	case http.MethodDelete:
		RequireTenantID(func(w http.ResponseWriter, req2 *http.Request) {
			r.handler.DeleteDashboard(w, req2, id)
		})(w, req)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// exportDetailHandler routes GET requests for export status.
func (r *Router) exportDetailHandler(w http.ResponseWriter, req *http.Request) {
	id := req.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "export id is required")
		return
	}

	RequireTenantID(func(w http.ResponseWriter, req2 *http.Request) {
		r.handler.GetExportStatus(w, req2, id)
	})(w, req)
}

// dashboardWidgetDataHandler extracts path params for widget data.
func (r *Router) dashboardWidgetDataHandler(w http.ResponseWriter, req *http.Request) {
	dashboardID := req.PathValue("dashboardId")
	widgetID := req.PathValue("widgetId")

	if dashboardID == "" || widgetID == "" {
		writeError(w, http.StatusBadRequest, "dashboard_id and widget_id are required")
		return
	}

	RequireTenantID(func(w http.ResponseWriter, req2 *http.Request) {
		r.handler.GetDashboardWidgetData(w, req2, dashboardID, widgetID)
	})(w, req)
}
