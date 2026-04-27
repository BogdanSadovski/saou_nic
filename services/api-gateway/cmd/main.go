package main

import (
	"log"
	"math"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

type bucketEntry struct {
	tokens float64
	last   time.Time
}

type tokenBucketLimiter struct {
	mu      sync.Mutex
	rate    float64
	burst   float64
	buckets map[string]*bucketEntry
}

func newTokenBucketLimiter(rate float64, burst int) *tokenBucketLimiter {
	return &tokenBucketLimiter{
		rate:    rate,
		burst:   float64(burst),
		buckets: make(map[string]*bucketEntry),
	}
}

func (l *tokenBucketLimiter) allow(key string) bool {
	now := time.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	b, ok := l.buckets[key]
	if !ok {
		l.buckets[key] = &bucketEntry{tokens: l.burst - 1, last: now}
		return true
	}

	elapsed := now.Sub(b.last).Seconds()
	b.tokens = math.Min(l.burst, b.tokens+elapsed*l.rate)
	b.last = now

	if b.tokens < 1 {
		return false
	}

	b.tokens -= 1
	return true
}

func mustURL(raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		log.Fatalf("invalid upstream url %q: %v", raw, err)
	}
	return u
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	default:
		return a + b
	}
}

func makeProxy(target *url.URL) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = target.Host
	}
	return proxy
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-User-ID")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func withRateLimit(limiter *tokenBucketLimiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r.RemoteAddr)
		user := strings.TrimSpace(r.Header.Get("X-User-ID"))
		session := extractSessionID(r.URL.Path)
		key := strings.Join([]string{ip, user, session}, "|")

		if !limiter.allow(key) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"rate limit exceeded"}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}

func clientIP(remoteAddr string) string {
	if idx := strings.LastIndex(remoteAddr, ":"); idx > 0 {
		return remoteAddr[:idx]
	}
	if remoteAddr == "" {
		return "unknown"
	}
	return remoteAddr
}

func extractSessionID(path string) string {
	parts := strings.Split(path, "/")
	for i := 0; i < len(parts)-1; i++ {
		if parts[i] == "sessions" && parts[i+1] != "" {
			return parts[i+1]
		}
	}
	return "-"
}

func rewriteAndProxy(proxy *httputil.ReverseProxy, transform func(string) string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = transform(r.URL.Path)
		proxy.ServeHTTP(w, r)
	})
}

func stripPrefix(prefix string) func(string) string {
	return func(path string) string {
		trimmed := strings.TrimPrefix(path, prefix)
		if trimmed == "" {
			return "/"
		}
		if !strings.HasPrefix(trimmed, "/") {
			return "/" + trimmed
		}
		return trimmed
	}
}

func addAPIV1(path string) string {
	trimmed := strings.TrimPrefix(path, "/api")
	if trimmed == "" {
		trimmed = "/"
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	return singleJoiningSlash("/api/v1", trimmed)
}

func main() {
	port := getEnv("PORT", "8000")

	userProxy := makeProxy(mustURL(getEnv("USER_SERVICE_URL", "http://user-service:8080")))
	interviewProxy := makeProxy(mustURL(getEnv("INTERVIEW_SERVICE_URL", "http://interview-service:8082")))
	resumeProxy := makeProxy(mustURL(getEnv("RESUME_SERVICE_URL", "http://resume-service:8080")))
	reportProxy := makeProxy(mustURL(getEnv("REPORT_SERVICE_URL", "http://report-service:8080")))
	adminProxy := makeProxy(mustURL(getEnv("ADMIN_SERVICE_URL", "http://admin-service:8080")))
	githubProxy := makeProxy(mustURL(getEnv("GITHUB_SERVICE_URL", "http://github-service:8082")))
	aiProxy := makeProxy(mustURL(getEnv("AI_SERVICE_URL", "http://ai-service:8001")))
	scoringProxy := makeProxy(mustURL(getEnv("SCORING_SERVICE_URL", "http://scoring-service:8080")))
	wsProxy := makeProxy(mustURL(getEnv("INTERVIEW_SERVICE_URL", "http://interview-service:8082")))

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","service":"api-gateway"}`))
	})

	mux.Handle("/ws", wsProxy)

	mux.Handle("/api/auth/", rewriteAndProxy(userProxy, addAPIV1))
	mux.Handle("/api/users/", rewriteAndProxy(userProxy, addAPIV1))

	mux.Handle("/api/interviews", rewriteAndProxy(interviewProxy, addAPIV1))
	mux.Handle("/api/interviews/", rewriteAndProxy(interviewProxy, addAPIV1))
	mux.Handle("/api/sessions/", rewriteAndProxy(interviewProxy, addAPIV1))
	mux.Handle("/api/answers/", rewriteAndProxy(interviewProxy, addAPIV1))

	mux.Handle("/api/resumes", rewriteAndProxy(resumeProxy, addAPIV1))
	mux.Handle("/api/resumes/", rewriteAndProxy(resumeProxy, addAPIV1))
	mux.Handle("/api/stats", rewriteAndProxy(resumeProxy, addAPIV1))

	mux.Handle("/api/reports", rewriteAndProxy(reportProxy, addAPIV1))
	mux.Handle("/api/reports/", rewriteAndProxy(reportProxy, addAPIV1))
	mux.Handle("/api/candidates/", rewriteAndProxy(reportProxy, addAPIV1))

	mux.Handle("/api/admin/", rewriteAndProxy(adminProxy, func(path string) string {
		return singleJoiningSlash("/api/v1", stripPrefix("/api/admin")(path))
	}))

	// GitHub profile import is implemented in interview-service.
	mux.Handle("/api/github/import", rewriteAndProxy(interviewProxy, func(string) string {
		return "/api/v1/github/import"
	}))

	// Resume import analytics is implemented in interview-service.
	mux.Handle("/api/resume/import", rewriteAndProxy(interviewProxy, func(string) string {
		return "/api/v1/resume/import"
	}))
	// Compatibility for legacy frontend fallback paths (/api/v1/resume/*).
	mux.Handle("/api/v1/resume/import", rewriteAndProxy(interviewProxy, func(string) string {
		return "/api/v1/resume/import"
	}))
	mux.Handle("/api/resume/history", rewriteAndProxy(interviewProxy, func(string) string {
		return "/api/v1/resume/history"
	}))
	mux.Handle("/api/v1/resume/history", rewriteAndProxy(interviewProxy, func(string) string {
		return "/api/v1/resume/history"
	}))
	mux.Handle("/api/resume/history/", rewriteAndProxy(interviewProxy, func(path string) string {
		id := strings.TrimPrefix(path, "/api/resume/history/")
		if id == "" {
			return "/api/v1/resume/history"
		}
		return "/api/v1/resume/history/" + id
	}))
	mux.Handle("/api/v1/resume/history/", rewriteAndProxy(interviewProxy, func(path string) string {
		id := strings.TrimPrefix(path, "/api/v1/resume/history/")
		if id == "" {
			return "/api/v1/resume/history"
		}
		return "/api/v1/resume/history/" + id
	}))

	mux.Handle("/api/github/", rewriteAndProxy(githubProxy, func(path string) string {
		return singleJoiningSlash("/api/v1", stripPrefix("/api/github")(path))
	}))

	// AI service exposes health on /health, while business endpoints are versioned under /api/v1.
	mux.Handle("/api/ai/health", rewriteAndProxy(aiProxy, func(string) string {
		return "/health"
	}))

	mux.Handle("/api/ai/", rewriteAndProxy(aiProxy, func(path string) string {
		return singleJoiningSlash("/api/v1", stripPrefix("/api/ai")(path))
	}))

	mux.Handle("/api/scoring/", rewriteAndProxy(scoringProxy, func(path string) string {
		return singleJoiningSlash("/api/v1/scoring", stripPrefix("/api/scoring")(path))
	}))

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           withRateLimit(newTokenBucketLimiter(10, 40), withCORS(mux)),
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("api-gateway listening on :%s", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("api-gateway failed: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
