package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Subscription quota enforcement.
//
// Лимиты применяются к двум критичным точкам:
//   • POST /interviews/sessions — создание новой сессии
//   • POST /resume/import       — анализ резюме
//
// Счётчики хранятся в Redis по ключу:
//   quota:<resource>:<user_id>:<YYYY-MM>
// и автоматически сбрасываются каждый календарный месяц
// (новый ключ — новые ноль). TTL 35 дней — на 5 дней дольше месяца,
// чтобы переход через границу не терял счётчик мгновенно.
//
// Тариф пользователя берётся через переданный в JWT claim `tier`.
// Если claim отсутствует — считаем trial (минимальные лимиты),
// поэтому даже для legacy-токенов без subscription enforcement
// работает консервативно.

const (
	resourceInterview = "interview"
	resourceResume    = "resume"
	resourceGitHub    = "github_import"
)

// quotaLimits описывает сколько действий доступно каждому тарифу
// за календарный месяц. Значение `-1` означает безлимит.
type quotaPlan struct {
	Interviews    int
	Resumes       int
	GitHubImports int
}

var quotaPlans = map[string]quotaPlan{
	"trial":      {Interviews: 5, Resumes: 3, GitHubImports: 1},
	"free":       {Interviews: 5, Resumes: 3, GitHubImports: 1},
	"basic":      {Interviews: 10, Resumes: 10, GitHubImports: 5},
	"starter":    {Interviews: 10, Resumes: 10, GitHubImports: 5},
	"pro":        {Interviews: 30, Resumes: -1, GitHubImports: 10},
	"platinum":   {Interviews: -1, Resumes: -1, GitHubImports: -1},
	"team":       {Interviews: -1, Resumes: -1, GitHubImports: -1},
	"enterprise": {Interviews: -1, Resumes: -1, GitHubImports: -1},
}

// resolveLimit возвращает лимит для конкретного ресурса по тарифу.
// Неизвестный тариф трактуется как самый жёсткий (trial).
func resolveLimit(tier string, resource string) int {
	plan, ok := quotaPlans[strings.ToLower(strings.TrimSpace(tier))]
	if !ok {
		plan = quotaPlans["trial"]
	}
	switch resource {
	case resourceInterview:
		return plan.Interviews
	case resourceResume:
		return plan.Resumes
	case resourceGitHub:
		return plan.GitHubImports
	}
	return -1
}

func quotaKey(userID, resource string) string {
	return fmt.Sprintf("quota:%s:%s:%s", resource, userID, time.Now().UTC().Format("2006-01"))
}

// QuotaStatus — то, что отдаём наружу (handler + JSON для UI).
type QuotaStatus struct {
	Resource  string `json:"resource"`
	Tier      string `json:"tier"`
	Limit     int    `json:"limit"`      // -1 == unlimited
	Used      int    `json:"used"`
	Remaining int    `json:"remaining"`  // -1 == unlimited
	Allowed   bool   `json:"allowed"`
}

func (h *Handler) currentUsage(ctx context.Context, userID, resource string) int {
	if h.redis == nil {
		return 0 // без Redis — не enforce'им, чтобы не блокировать поток
	}
	val, err := h.redis.Get(ctx, quotaKey(userID, resource)).Int()
	if err != nil {
		return 0
	}
	return val
}

// checkQuota — read-only. Используется handler'ом GET /quota/me и при
// формировании ответа после consume'а, чтобы клиент сразу видел
// remaining.
func (h *Handler) checkQuota(ctx context.Context, userID, tier, resource string) QuotaStatus {
	limit := resolveLimit(tier, resource)
	used := h.currentUsage(ctx, userID, resource)
	status := QuotaStatus{
		Resource: resource,
		Tier:     tier,
		Limit:    limit,
		Used:     used,
	}
	if limit < 0 {
		status.Allowed = true
		status.Remaining = -1
		return status
	}
	remaining := limit - used
	if remaining < 0 {
		remaining = 0
	}
	status.Remaining = remaining
	status.Allowed = used < limit
	return status
}

// consumeQuota увеличивает счётчик. Возвращает финальный статус.
// Если лимит уже исчерпан до инкремента — Allowed=false, счётчик не
// растёт (rollback через DECR). Если лимит = -1 (unlimited) — INCR
// делаем всё равно, чтобы видеть статистику использования.
func (h *Handler) consumeQuota(ctx context.Context, userID, tier, resource string) (QuotaStatus, error) {
	limit := resolveLimit(tier, resource)
	if h.redis == nil {
		// Без Redis enforce невозможен — отдаём «разрешено», но логируем.
		h.logger.Warn("quota: redis unavailable, allowing without enforcement")
		return QuotaStatus{Resource: resource, Tier: tier, Limit: limit, Allowed: true, Remaining: -1}, nil
	}
	key := quotaKey(userID, resource)
	newVal, err := h.redis.Incr(ctx, key).Result()
	if err != nil {
		h.logger.WithError(err).Warn("quota: redis incr failed")
		return QuotaStatus{Resource: resource, Tier: tier, Limit: limit, Allowed: true, Remaining: -1}, nil
	}
	// первая постановка ключа — выставим TTL на 35 дней
	if newVal == 1 {
		_ = h.redis.Expire(ctx, key, 35*24*time.Hour).Err()
	}
	used := int(newVal)
	status := QuotaStatus{
		Resource: resource,
		Tier:     tier,
		Limit:    limit,
		Used:     used,
	}
	if limit < 0 {
		status.Allowed = true
		status.Remaining = -1
		return status, nil
	}
	if used > limit {
		// откат: лимит исчерпан, не должен был пройти
		_ = h.redis.Decr(ctx, key).Err()
		status.Used = used - 1
		status.Remaining = 0
		status.Allowed = false
		return status, nil
	}
	status.Remaining = limit - used
	status.Allowed = true
	return status, nil
}

// userTierFromContext возвращает тариф пользователя из:
//   1. JWT claim `tier`     (если user-service его проставит)
//   2. HTTP header X-Subscription-Tier (api-gateway может прокидывать)
//   3. role=admin → platinum bypass (админам лимиты не нужны)
//   4. default → trial
func (h *Handler) userTierFromContext(ctx context.Context) string {
	if v := ctx.Value(contextKeyTier); v != nil {
		if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
			return s
		}
	}
	if v := ctx.Value(ContextKeyRole); v != nil {
		if role, ok := v.(string); ok {
			if strings.EqualFold(role, "admin") || strings.EqualFold(role, "moderator") {
				return "platinum"
			}
		}
	}
	return "trial"
}

// enforceQuotaJSON — удобная обёртка для handler'ов: если лимит
// исчерпан, пишет 402 Payment Required с подробным телом и возвращает
// false. Иначе возвращает true и обновлённый статус.
func (h *Handler) enforceQuotaJSON(w http.ResponseWriter, r *http.Request, userID, resource string) (QuotaStatus, bool) {
	tier := h.userTierFromContext(r.Context())
	status, err := h.consumeQuota(r.Context(), userID, tier, resource)
	if err != nil {
		h.logger.WithError(err).WithFields(logrus.Fields{
			"user_id":  userID,
			"resource": resource,
		}).Warn("quota: consume error (allowing)")
		return QuotaStatus{Allowed: true}, true
	}
	if !status.Allowed {
		writeJSON(w, http.StatusPaymentRequired, map[string]any{
			"success": false,
			"error":   "quota_exceeded",
			"message": fmt.Sprintf("Лимит «%s» на тарифе %s исчерпан (%d/%d). Обновите подписку.", resource, tier, status.Used, status.Limit),
			"data":    status,
		})
		return status, false
	}
	return status, true
}
