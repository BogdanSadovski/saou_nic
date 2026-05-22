package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// HH.ru Vacancies Search API (official OAuth integration).
//
// Endpoint:    GET https://api.hh.ru/vacancies
// Token URL:   POST https://api.hh.ru/token  (client_credentials grant)
// Docs:        https://api.hh.ru/openapi/redoc#tag/Poisk-vakansij
//              https://github.com/hhru/api/blob/master/docs/authorization.md
//
// Раньше работали анонимно (только User-Agent с e-mail) — публичный
// эндпоинт это позволяет, но HH-антибот периодически режет такие
// запросы по IP (особенно в Docker-средах с общим egress). Теперь
// ходим через зарегистрированное приложение из https://dev.hh.ru/admin:
//
//   HH_CLIENT_ID / HH_CLIENT_SECRET — выдаются при регистрации.
//     При наличии обоих сервис сам получает access_token по схеме
//     client_credentials и обновляет его за минуту до истечения.
//   HH_ACCESS_TOKEN — можно прокинуть готовый токен вручную (например,
//     долгоживущий «приложенческий» токен).  В этом случае автообмен
//     по client_credentials не выполняется.
//   HH_API_USER_AGENT — по-прежнему обязателен; HH требует e-mail
//     контактного лица в формате "AppName/Version (email)".
//
// Area IDs: 16=Беларусь · 113=Россия · 1=Москва · 2=СПб ·
// 1002=Минск · 1003=Гомель · 1004=Могилёв · 1005=Витебск ·
// 1006=Гродно · 1007=Брест.  Omit param for worldwide.

const (
	hhAPIBase       = "https://api.hh.ru/vacancies"
	hhTokenURL      = "https://api.hh.ru/token"
	hhCacheTTL      = time.Hour
	hhDefaultArea   = "1002" // Минск — где сосредоточено ≥70% IT-вакансий Беларуси.
	hhRequestPerPag = 12
)

// ----------------- OAuth client_credentials token cache -----------------
//
// HH-токен на client_credentials живёт обычно 14 дней, но мы не
// полагаемся на это — храним `expires_at` из ответа и обновляем за
// минуту до конца.  Кеш — process-local; при рестарте контейнера
// получаем токен заново (одна лишняя HTTP-операция за деплой).

type hhTokenCache struct {
	mu        sync.Mutex
	token     string
	expiresAt time.Time
}

var hhToken hhTokenCache

type hhTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// fetchHHAccessToken — получает application-токен по client_credentials.
// Безопасен к параллельным вызовам: внутренний mutex сериализует
// обновления.  Если HH_ACCESS_TOKEN задан явно — используется он
// и сетевой обмен пропускается.
func fetchHHAccessToken(ctx context.Context, userAgent string) (string, error) {
	if manual := strings.TrimSpace(os.Getenv("HH_ACCESS_TOKEN")); manual != "" {
		return manual, nil
	}
	clientID := strings.TrimSpace(os.Getenv("HH_CLIENT_ID"))
	clientSecret := strings.TrimSpace(os.Getenv("HH_CLIENT_SECRET"))
	if clientID == "" || clientSecret == "" {
		return "", nil // OAuth выключен — пойдём анонимно (только UA).
	}

	hhToken.mu.Lock()
	defer hhToken.mu.Unlock()
	if hhToken.token != "" && time.Until(hhToken.expiresAt) > time.Minute {
		return hhToken.token, nil
	}

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, hhTokenURL,
		strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("hh-token: build: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", userAgent)

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("hh-token: do: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", fmt.Errorf("hh-token: status %d: %s", resp.StatusCode, string(body))
	}

	var tr hhTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return "", fmt.Errorf("hh-token: decode: %w", err)
	}
	if tr.AccessToken == "" {
		return "", fmt.Errorf("hh-token: empty access_token")
	}
	hhToken.token = tr.AccessToken
	ttl := time.Duration(tr.ExpiresIn) * time.Second
	if ttl <= 0 {
		ttl = 14 * 24 * time.Hour // дефолт по докам HH
	}
	hhToken.expiresAt = time.Now().Add(ttl)
	return tr.AccessToken, nil
}

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

	// HH-API правила User-Agent (docs hh.ru):
	//   1. Должен быть установлен (без него 403);
	//   2. Формат "AppName/Version (real-contact-email)";
	//   3. E-mail должен совпадать с тем, что указан при регистрации
	//      приложения в https://dev.hh.ru/admin.
	userAgent := strings.TrimSpace(os.Getenv("HH_API_USER_AGENT"))
	if userAgent == "" {
		userAgent = "RealSync-Interview-Platform/1.0 (realsync.platform+hh@gmail.com)"
	}

	// Получаем application-токен (cached). Если CLIENT_ID/SECRET не
	// заданы — функция вернёт пустую строку и пойдём анонимно.
	accessToken, tokErr := fetchHHAccessToken(ctx, userAgent)
	if tokErr != nil {
		h.logger.WithError(tokErr).Warn("hh: token fetch failed, falling back to anonymous mode")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("hh: build request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "ru-BY,ru;q=0.9,en;q=0.5")
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("hh: do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		hint := ""
		switch resp.StatusCode {
		case 401:
			hint = " (HH_ACCESS_TOKEN истёк/невалиден — проверьте HH_CLIENT_ID/HH_CLIENT_SECRET в .env)"
			// сбрасываем кеш, чтобы следующий запрос обновил токен
			hhToken.mu.Lock()
			hhToken.token = ""
			hhToken.mu.Unlock()
		case 403:
			hint = " (приложение не зарегистрировано или e-mail в UA не совпадает с dev.hh.ru/admin)"
		case 429:
			hint = " (rate-limit; повторите через минуту)"
		}
		return nil, fmt.Errorf("hh: status %d%s: %s", resp.StatusCode, hint, string(body))
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
