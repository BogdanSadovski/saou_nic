package api

import (
	"net/http"
	"strings"
)

// Router is a simple HTTP router
type Router struct {
	routes map[string]map[string]http.HandlerFunc
}

// NewRouter creates a new Router
func NewRouter() *Router {
	return &Router{
		routes: make(map[string]map[string]http.HandlerFunc),
	}
}

// Handle registers a handler for a method and path pattern
func (r *Router) Handle(method, pattern string, handler http.HandlerFunc) {
	method = strings.ToUpper(method)
	if r.routes[method] == nil {
		r.routes[method] = make(map[string]http.HandlerFunc)
	}
	r.routes[method][pattern] = handler
}

// ServeHTTP implements http.Handler
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	method := strings.ToUpper(req.Method)
	path := req.URL.Path

	// Try exact match first
	if handlers, ok := r.routes[method]; ok {
		if handler, ok := handlers[path]; ok {
			handler(w, req)
			return
		}
	}

	// Try prefix match for dynamic routes
	if handlers, ok := r.routes[method]; ok {
		for pattern, handler := range handlers {
			if matchesPattern(path, pattern) {
				handler(w, req)
				return
			}
		}
	}

	// Handle CORS preflight
	if req.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	http.NotFound(w, req)
}

// matchesPattern checks if a path matches a pattern with optional wildcards
func matchesPattern(path, pattern string) bool {
	pathParts := strings.Split(strings.Trim(path, "/"), "/")
	patternParts := strings.Split(strings.Trim(pattern, "/"), "/")

	if len(pathParts) != len(patternParts) {
		return false
	}

	for i, part := range patternParts {
		if part == "{id}" || part == "{candidate_id}" {
			continue // Wildcard matches any value
		}
		if part != pathParts[i] {
			return false
		}
	}

	return true
}

// SetupRoutes configures all routes for the report-service
func SetupRoutes(handlers *Handlers) http.Handler {
	router := NewRouter()

	// Health check
	router.Handle(http.MethodGet, "/health", handlers.HandleHealthCheck)

	// Report CRUD operations
	router.Handle(http.MethodPost, "/api/v1/reports", handlers.HandleCreateReport)
	router.Handle(http.MethodGet, "/api/v1/reports", handlers.HandleListReports)
	router.Handle(http.MethodGet, "/api/v1/reports/{id}", handlers.HandleGetReport)
	router.Handle(http.MethodDelete, "/api/v1/reports/{id}", handlers.HandleDeleteReport)

	// Report generation
	router.Handle(http.MethodPost, "/api/v1/reports/{id}/generate/pdf", handlers.HandleGeneratePDF)
	router.Handle(http.MethodPost, "/api/v1/reports/{id}/generate/docx", handlers.HandleGenerateDOCX)

	// Report download
	router.Handle(http.MethodGet, "/api/v1/reports/{id}/download", handlers.HandleDownloadReport)

	// Statistics
	router.Handle(http.MethodGet, "/api/v1/reports/stats", handlers.HandleGetStats)

	// Candidate reports
	router.Handle(http.MethodGet, "/api/v1/candidates/{candidate_id}/reports", handlers.HandleGetReportsByCandidate)

	// Middleware wrapper with CORS headers
	return corsMiddleware(router)
}

// corsMiddleware adds CORS headers to all responses
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Disposition, X-Total-Count")
		next.ServeHTTP(w, r)
	})
}
