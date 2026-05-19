package api

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// dev.by / jobs.devby.io scraper.
//
// dev.by нет публичного API для вакансий, но `robots.txt` парсинг
// списочной страницы не запрещает. Достаём HTML, выдираем нужные
// данные регулярками (структура классов стабильна с 2023+ года):
//
//   div.vacancies-list-item
//     a.vacancies-list-item__link_block[href=/vacancies/{id}]   — title + ссылка
//     div.vacancies-list-item__company                          — компания (HTML может содержать
//                                                                   <span class="...label">Удал.</span>)
//     div.vacancies-list-item__salary                           — зарплата (опционально)
//     span.vacancies-list-item__technology-tag__name            — теги технологий
//
// Кэшируем в Redis на 30 минут — там тысячи запросов в день
// допустимы, но смысла бить чаще нет.

const (
	devByListURL      = "https://jobs.devby.io/vacancies"
	devByDetailBase   = "https://jobs.devby.io"
	devByCacheTTL     = 30 * time.Minute
	devByMaxResults   = 10
	devByMinFromDevBy = 1 // как минимум 1 dev.by-вакансия в выдаче (если есть)
)

type DevByVacancy struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	URL        string   `json:"url"`
	Company    string   `json:"company"`
	Remote     bool     `json:"remote"`
	Salary     string   `json:"salary,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	Source     string   `json:"source"` // always "dev.by"
	RelevanceScore float64 `json:"relevance_score,omitempty"`
}

type DevByResponse struct {
	Query    string         `json:"query"`
	Total    int            `json:"total"`
	Items    []DevByVacancy `json:"items"`
	CachedAt time.Time      `json:"cached_at"`
}

// --- Parser regexes (compiled once) ---

var (
	devByItemRe = regexp.MustCompile(
		`(?s)<div class="vacancies-list-item(?:[^"]*)?">.*?` +
			`<a class="vacancies-list-item__link_block"\s+href="([^"]+)"[^>]*>` +
			`([^<]+)</a>.*?` +
			`<div class="vacancies-list-item__company"[^>]*>(.*?)</div>` +
			`(?:.*?<div class="vacancies-list-item__salary"[^>]*>(.*?)</div>)?` +
			`(?:.*?<div class="vacancies-list-item__technology-tags"[^>]*>(.*?)</div>)?`,
	)
	devByTagRe   = regexp.MustCompile(`<span class="vacancies-list-item__technology-tag__name"[^>]*>([^<]+)</span>`)
	devByLabelRe = regexp.MustCompile(`<span class="vacancies-list-item__label"[^>]*>([^<]+)</span>`)
	devByWhiteRe = regexp.MustCompile(`\s+`)
)

func unescapeHTMLText(s string) string {
	s = html.UnescapeString(s)
	s = devByWhiteRe.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

func parseDevByList(htmlBody string) []DevByVacancy {
	matches := devByItemRe.FindAllStringSubmatch(htmlBody, -1)
	out := make([]DevByVacancy, 0, len(matches))
	for i, m := range matches {
		if len(m) < 4 {
			continue
		}
		href := m[1]
		title := unescapeHTMLText(m[2])
		companyRaw := m[3]
		salaryRaw := ""
		if len(m) > 4 {
			salaryRaw = m[4]
		}
		tagsRaw := ""
		if len(m) > 5 {
			tagsRaw = m[5]
		}

		// company может содержать <span class="...label">Удал.</span>
		// — вытащим как индикатор remote, потом удалим из company.
		remote := false
		if labelMatch := devByLabelRe.FindStringSubmatch(companyRaw); len(labelMatch) > 1 {
			lbl := strings.ToLower(unescapeHTMLText(labelMatch[1]))
			if strings.Contains(lbl, "удал") || strings.Contains(lbl, "remote") {
				remote = true
			}
			companyRaw = devByLabelRe.ReplaceAllString(companyRaw, "")
		}
		company := unescapeHTMLText(companyRaw)

		// tags
		var tags []string
		for _, t := range devByTagRe.FindAllStringSubmatch(tagsRaw, -1) {
			if len(t) > 1 {
				tag := unescapeHTMLText(t[1])
				if tag != "" {
					tags = append(tags, tag)
				}
			}
		}

		// vacancy id из href вида /vacancies/27873
		id := strings.TrimPrefix(href, "/vacancies/")
		id = strings.TrimSpace(strings.TrimSuffix(id, "/"))

		// link absolute
		linkURL := href
		if strings.HasPrefix(linkURL, "/") {
			linkURL = devByDetailBase + linkURL
		}

		out = append(out, DevByVacancy{
			ID:             id,
			Title:          title,
			URL:            linkURL,
			Company:        company,
			Remote:         remote,
			Salary:         unescapeHTMLText(salaryRaw),
			Tags:           tags,
			Source:         "dev.by",
			RelevanceScore: 1.0 - float64(i)*0.04,
		})
		if len(out) >= devByMaxResults*3 {
			break // достаём с запасом, потом отфильтруем по скиллам
		}
	}
	return out
}

// rankDevByByQuery ранжирует выдачу по совпадению тегов/тайтла с
// резюме-скиллами кандидата. dev.by не даёт поиска по тегам через
// URL, поэтому делаем in-memory ранжирование над всей лентой.
func rankDevByByQuery(items []DevByVacancy, skills []string) []DevByVacancy {
	if len(skills) == 0 {
		return items
	}
	lowerSkills := make([]string, 0, len(skills))
	for _, s := range skills {
		s = strings.ToLower(strings.TrimSpace(s))
		if s != "" {
			lowerSkills = append(lowerSkills, s)
		}
	}
	if len(lowerSkills) == 0 {
		return items
	}

	type scored struct {
		v     DevByVacancy
		match int
	}
	scoredList := make([]scored, 0, len(items))
	for _, v := range items {
		hay := strings.ToLower(v.Title + " " + strings.Join(v.Tags, " ") + " " + v.Company)
		m := 0
		for _, sk := range lowerSkills {
			if strings.Contains(hay, sk) {
				m++
			}
		}
		scoredList = append(scoredList, scored{v: v, match: m})
	}
	// sort: больше совпадений → выше; при равенстве сохраняем порядок dev.by.
	for i := 1; i < len(scoredList); i++ {
		for j := i; j > 0 && scoredList[j-1].match < scoredList[j].match; j-- {
			scoredList[j-1], scoredList[j] = scoredList[j], scoredList[j-1]
		}
	}
	out := make([]DevByVacancy, 0, len(scoredList))
	for i, s := range scoredList {
		if s.match > 0 {
			s.v.RelevanceScore = 0.95 - float64(i)*0.04
			if s.v.RelevanceScore < 0.4 {
				s.v.RelevanceScore = 0.4
			}
		} else {
			s.v.RelevanceScore = 0.3
		}
		out = append(out, s.v)
	}
	return out
}

func (h *Handler) fetchDevByVacancies(ctx context.Context, skills []string) (*DevByResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, devByListURL, nil)
	if err != nil {
		return nil, fmt.Errorf("devby: build request: %w", err)
	}
	// Реалистичный browser UA — dev.by фильтрует пустые/curl UA.
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 14_0) AppleWebKit/537.36 (KHTML, like Gecko) RealSync/1.0 Safari/605.1.15")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("Accept-Language", "ru-BY,ru;q=0.9,en;q=0.5")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("devby: do request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("devby: status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024)) // hard 5MB cap
	if err != nil {
		return nil, fmt.Errorf("devby: read: %w", err)
	}

	items := parseDevByList(string(body))
	items = rankDevByByQuery(items, skills)
	if len(items) > devByMaxResults {
		items = items[:devByMaxResults]
	}

	queryPreview := strings.Join(skills, ", ")
	if len(queryPreview) > 80 {
		queryPreview = queryPreview[:80] + "…"
	}

	return &DevByResponse{
		Query:    queryPreview,
		Total:    len(items),
		Items:    items,
		CachedAt: time.Now(),
	}, nil
}

// GetMatchingDevByVacancies — handler для
//
//	GET /api/v1/resume/devby/{report_id}
func (h *Handler) GetMatchingDevByVacancies(w http.ResponseWriter, r *http.Request) {
	userID := h.userIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	reportID := strings.TrimSpace(mux.Vars(r)["report_id"])
	if reportID == "" {
		writeError(w, http.StatusBadRequest, "report_id is required")
		return
	}

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

	cacheKey := fmt.Sprintf("devby:vacancies:%s", hashStr(strings.Join(found.ExtractedSkills, ",")))
	if h.redis != nil {
		if cached, err := h.redis.Get(r.Context(), cacheKey).Result(); err == nil && cached != "" {
			var resp DevByResponse
			if json.Unmarshal([]byte(cached), &resp) == nil {
				writeJSON(w, http.StatusOK, &resp)
				return
			}
		}
	}

	resp, err := h.fetchDevByVacancies(r.Context(), found.ExtractedSkills)
	if err != nil {
		h.logger.WithError(err).Warn("devby: fetch failed")
		writeError(w, http.StatusBadGateway, "dev.by unavailable: "+err.Error())
		return
	}

	if h.redis != nil {
		if payload, err := json.Marshal(resp); err == nil {
			h.redis.Set(r.Context(), cacheKey, payload, devByCacheTTL)
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

