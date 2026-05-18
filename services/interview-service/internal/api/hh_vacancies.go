package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// HH.ru public Vacancies Search API.
//
// Endpoint: GET https://api.hh.ru/vacancies
// Docs:    https://api.hh.ru/openapi/redoc#tag/Poisk-vakansij/operation/get-vacancies
//
// Key points:
//   - Public read-only — no API key required for vacancy search
//   - User-Agent header is mandatory; HH bans empty / browser UAs.
//     Must be of the form "AppName/Version (contact-email)"
//   - Rate limit: documented as "reasonable", typically ~30 RPS for an
//     anonymous client. We cache results in Redis for an hour per query
//     fingerprint, so user-facing requests rarely hit HH directly.
//   - Area IDs: 16=Беларусь · 113=Россия · 1=Москва · 2=СПб ·
//     omit param for worldwide.

const (
	hhAPIBase       = "https://api.hh.ru/vacancies"
	hhCacheTTL      = time.Hour
	hhDefaultArea   = "16" // Belarus, since this is a Belarusian thesis project.
	hhRequestPerPag = 12
)

// HHVacancy is the trimmed-down vacancy shape we expose to the
// frontend. Full HH.ru response has 40+ fields; we keep only what the
// resume page actually renders.
type HHVacancy struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	URL             string  `json:"url"`
	Employer        string  `json:"employer"`
	Area            string  `json:"area"`
	Experience      string  `json:"experience"`
	Schedule        string  `json:"schedule"`
	Employment      string  `json:"employment"`
	SalaryFrom      *int    `json:"salary_from,omitempty"`
	SalaryTo        *int    `json:"salary_to,omitempty"`
	SalaryCurrency  string  `json:"salary_currency,omitempty"`
	SalaryGross     bool    `json:"salary_gross,omitempty"`
	Snippet         string  `json:"snippet,omitempty"`
	PublishedAt     string  `json:"published_at"`
	RelevanceScore  float64 `json:"relevance_score,omitempty"`
}

// HHVacanciesResponse is what `GET /resume/vacancies` returns.
type HHVacanciesResponse struct {
	Query    string      `json:"query"`
	Area     string      `json:"area"`
	Total    int         `json:"total"`
	Items    []HHVacancy `json:"items"`
	CachedAt time.Time   `json:"cached_at"`
}

// ---------------------------- HH raw shapes ----------------------------

type hhRawResponse struct {
	Items []hhRawVacancy `json:"items"`
	Found int            `json:"found"`
}

type hhRawVacancy struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	URL         string         `json:"alternate_url"`
	Employer    *hhRawEmployer `json:"employer"`
	Area        *hhRawNamed    `json:"area"`
	Experience  *hhRawNamed    `json:"experience"`
	Schedule    *hhRawNamed    `json:"schedule"`
	Employment  *hhRawNamed    `json:"employment"`
	Salary      *hhRawSalary   `json:"salary"`
	Snippet     *hhRawSnippet  `json:"snippet"`
	PublishedAt string         `json:"published_at"`
}

type hhRawEmployer struct {
	Name string `json:"name"`
}
type hhRawNamed struct {
	Name string `json:"name"`
}
type hhRawSalary struct {
	From     *int   `json:"from"`
	To       *int   `json:"to"`
	Currency string `json:"currency"`
	Gross    bool   `json:"gross"`
}
type hhRawSnippet struct {
	Requirement    string `json:"requirement"`
	Responsibility string `json:"responsibility"`
}

// ---------------------------- query builder ----------------------------

// buildHHQuery constructs an HH.ru `text=` parameter from the resume's
// recommended role + top extracted skills. HH treats space as AND, so
// we keep the query tight: "<role> AND (skill1 OR skill2)".
//
// Examples:
//
//	role="Backend Engineer", skills=[Go, PostgreSQL, gRPC]
//	→ "Backend AND (Go OR PostgreSQL OR gRPC)"
//
//	role="", skills=[Python, FastAPI]
//	→ "Python OR FastAPI"
//
// We strip multi-word noise like "Engineer / Developer" to keep HH's
// relevance ranker happy.
func buildHHQuery(role string, skills []string) string {
	roleClean := strings.TrimSpace(role)
	// Drop trailing "Engineer / Developer / Programmer" because HH
	// ranking handles role tokens itself, and "Engineer" matches
	// almost every IT vacancy and pollutes results.
	roleClean = strings.NewReplacer(
		" Engineer", "",
		" Developer", "",
		" Programmer", "",
		" engineer", "",
		" developer", "",
	).Replace(roleClean)
	roleClean = strings.TrimSpace(roleClean)

	uniqSkills := make([]string, 0, len(skills))
	seen := map[string]struct{}{}
	for _, s := range skills {
		v := strings.TrimSpace(s)
		if v == "" {
			continue
		}
		k := strings.ToLower(v)
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		uniqSkills = append(uniqSkills, v)
		if len(uniqSkills) >= 4 {
			break
		}
	}

	skillsExpr := ""
	if len(uniqSkills) > 0 {
		if len(uniqSkills) == 1 {
			skillsExpr = uniqSkills[0]
		} else {
			skillsExpr = "(" + strings.Join(uniqSkills, " OR ") + ")"
		}
	}

	switch {
	case roleClean != "" && skillsExpr != "":
		return roleClean + " AND " + skillsExpr
	case roleClean != "":
		return roleClean
	case skillsExpr != "":
		return skillsExpr
	default:
		return "developer"
	}
}

// ---------------------------- HTTP fetch ----------------------------

func (h *Handler) fetchHHVacancies(ctx context.Context, query, area string) (*HHVacanciesResponse, error) {
	params := url.Values{}
	params.Set("text", query)
	params.Set("per_page", strconv.Itoa(hhRequestPerPag))
	params.Set("order_by", "relevance")
	params.Set("only_with_salary", "false")
	if area != "" && area != "world" {
		params.Set("area", area)
	}

	fullURL := hhAPIBase + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("hh: build request: %w", err)
	}
	// HH explicitly requires a User-Agent identifying the client with
	// an email contact. Without it requests fail with 403.
	req.Header.Set("User-Agent", "RealSync-Interview-Platform/1.0 (admin@realsync.local)")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "ru-RU,ru")

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("hh: do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("hh: status %d: %s", resp.StatusCode, string(body))
	}

	var raw hhRawResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("hh: decode: %w", err)
	}

	out := &HHVacanciesResponse{
		Query:    query,
		Area:     area,
		Total:    raw.Found,
		Items:    make([]HHVacancy, 0, len(raw.Items)),
		CachedAt: time.Now(),
	}
	for i, item := range raw.Items {
		v := HHVacancy{
			ID:          item.ID,
			Name:        item.Name,
			URL:         item.URL,
			PublishedAt: item.PublishedAt,
		}
		if item.Employer != nil {
			v.Employer = item.Employer.Name
		}
		if item.Area != nil {
			v.Area = item.Area.Name
		}
		if item.Experience != nil {
			v.Experience = item.Experience.Name
		}
		if item.Schedule != nil {
			v.Schedule = item.Schedule.Name
		}
		if item.Employment != nil {
			v.Employment = item.Employment.Name
		}
		if item.Salary != nil {
			v.SalaryFrom = item.Salary.From
			v.SalaryTo = item.Salary.To
			v.SalaryCurrency = item.Salary.Currency
			v.SalaryGross = item.Salary.Gross
		}
		if item.Snippet != nil {
			parts := []string{}
			if r := strings.TrimSpace(item.Snippet.Requirement); r != "" {
				parts = append(parts, r)
			}
			if r := strings.TrimSpace(item.Snippet.Responsibility); r != "" {
				parts = append(parts, r)
			}
			v.Snippet = strings.Join(parts, " · ")
		}
		// Relevance: top result = 1.0, last = ~0.6, simple linear.
		v.RelevanceScore = 1.0 - float64(i)*0.04
		if v.RelevanceScore < 0.4 {
			v.RelevanceScore = 0.4
		}
		out.Items = append(out.Items, v)
	}
	return out, nil
}

// ---------------------------- HTTP handler ----------------------------

// GetMatchingVacancies handles
//
//	GET /api/v1/resume/vacancies/{report_id}?area=16
//
// area is optional; if omitted, defaults to Belarus (16). Pass
// area=world to disable the filter entirely.
func (h *Handler) GetMatchingVacancies(w http.ResponseWriter, r *http.Request) {
	userID := h.userIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	reportID := strings.TrimSpace(vars["report_id"])
	if reportID == "" {
		writeError(w, http.StatusBadRequest, "report_id is required")
		return
	}

	area := strings.TrimSpace(r.URL.Query().Get("area"))
	if area == "" {
		area = hhDefaultArea
	}

	// Look up the report.
	h.resumeMu.RLock()
	history := h.resumeHistory[userID]
	var found *resumeImportResponse
	for i := range history {
		if history[i].ReportID == reportID {
			found = &history[i]
			break
		}
	}
	h.resumeMu.RUnlock()
	if found == nil {
		writeError(w, http.StatusNotFound, "report not found")
		return
	}

	// Build query from the AI-recommended role + extracted skills.
	role := ""
	if len(found.AIInsights.RecommendedPositions) > 0 {
		role = found.AIInsights.RecommendedPositions[0].Role
	}
	query := buildHHQuery(role, found.ExtractedSkills)

	// Redis cache lookup.
	cacheKey := fmt.Sprintf("hh:vacancies:%s:%s", area, hashStr(query))
	if h.redis != nil {
		if cached, err := h.redis.Get(r.Context(), cacheKey).Result(); err == nil && cached != "" {
			var resp HHVacanciesResponse
			if json.Unmarshal([]byte(cached), &resp) == nil {
				writeJSON(w, http.StatusOK, &resp)
				return
			}
		}
	}

	resp, err := h.fetchHHVacancies(r.Context(), query, area)
	if err != nil {
		h.logger.WithError(err).Warn("hh: fetch failed")
		writeError(w, http.StatusBadGateway, "hh.ru unavailable: "+err.Error())
		return
	}

	if h.redis != nil {
		if payload, err := json.Marshal(resp); err == nil {
			h.redis.Set(r.Context(), cacheKey, payload, hhCacheTTL)
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

// hashStr is a tiny FNV-style hash to fingerprint the query for cache
// keys. Avoids pulling crypto/sha256 for a non-security use.
func hashStr(s string) string {
	var h uint64 = 14695981039346656037
	for _, c := range s {
		h ^= uint64(c)
		h *= 1099511628211
	}
	return fmt.Sprintf("%x", h)
}

// Keep linter happy: logrus import is conditional based on usage above.
var _ = logrus.Fields{}
