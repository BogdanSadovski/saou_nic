package api

import (
	"net/http"
	"strings"
)

// Router handles HTTP routing
type Router struct {
	handler *Handler
	mux     *http.ServeMux
}

// NewRouter creates a new Router with all routes configured
func NewRouter(handler *Handler) *Router {
	router := &Router{
		handler: handler,
		mux:     http.NewServeMux(),
	}

	router.setupRoutes()
	return router
}

// ServeHTTP implements http.Handler
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Apply middleware
	w = r.applyCORS(w)
	r.applyLogging(req)

	r.mux.ServeHTTP(w, req)
}

// setupRoutes configures all HTTP routes
func (r *Router) setupRoutes() {
	// Health check
	r.mux.HandleFunc("GET /health", r.handler.HealthCheck)
	r.mux.HandleFunc("GET /ready", r.handler.HealthCheck)

	// API v1 routes
	r.mux.HandleFunc("GET /api/v1/stats", r.withAuth(r.handler.GetStats))
	r.mux.HandleFunc("GET /api/v1/resumes", r.withAuth(r.handler.GetUserResumes))
	r.mux.HandleFunc("POST /api/v1/resumes", r.withAuth(r.handler.CreateResume))
	r.mux.HandleFunc("GET /api/v1/resumes/{id}", r.withAuth(r.handler.GetResume))
	r.mux.HandleFunc("PUT /api/v1/resumes/{id}", r.withAuth(r.handler.UpdateResume))
	r.mux.HandleFunc("DELETE /api/v1/resumes/{id}", r.withAuth(r.handler.DeleteResume))
	r.mux.HandleFunc("POST /api/v1/resumes/{id}/reparse", r.withAuth(r.handler.ReparseResume))
	r.mux.HandleFunc("GET /api/v1/resumes/{id}/download", r.withAuth(r.handler.GetResumeFileURL))
}

func (r *Router) withAuth(next http.HandlerFunc) http.HandlerFunc {
	protected := Authentication(next)
	return func(w http.ResponseWriter, req *http.Request) {
		protected.ServeHTTP(w, req)
	}
}

// applyCORS adds CORS headers to the response
func (r *Router) applyCORS(w http.ResponseWriter) http.ResponseWriter {
	if rw, ok := w.(interface{ Header() http.Header }); ok {
		header := rw.Header()
		header.Set("Access-Control-Allow-Origin", "*")
		header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		header.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-User-ID")
		header.Set("Access-Control-Max-Age", "86400")
	}
	return w
}

// applyLogging logs request information
func (r *Router) applyLogging(req *http.Request) {
	// In production, integrate with structured logging (e.g., zap, slog)
	// Example: log.Info("request", "method", req.Method, "path", req.URL.Path)
}

// Middleware is a function that wraps an http.Handler
type Middleware func(http.Handler) http.Handler

// Chain applies a chain of middleware to a handler
func Chain(handler http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

// Authentication middleware (placeholder)
func Authentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// In production, validate JWT token or session
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "missing authorization header", http.StatusUnauthorized)
			return
		}

		// Extract and validate token
		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "invalid authorization header format", http.StatusUnauthorized)
			return
		}

		// TODO: Validate token and set user context
		next.ServeHTTP(w, r)
	})
}

// RateLimit middleware (placeholder)
func RateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// In production, implement rate limiting using a token bucket or sliding window
		next.ServeHTTP(w, r)
	})
}

// RequestID middleware adds a unique ID to each request
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Generate and set request ID
		// In production, use uuid.New().String()
		next.ServeHTTP(w, r)
	})
}
