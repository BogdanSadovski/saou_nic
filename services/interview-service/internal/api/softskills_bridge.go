package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

// Soft-skills режим интервью — без отдельного ML-сервиса.
//
// Раньше тут был HTTP-клиент к services/softskills-service (PyTorch
// rubert-tiny2 + custom regressor). Этот сервис был удалён — оценка
// и выбор вопроса теперь идут через основной LLM-каскад (DeepSeek и
// т.п.), что:
//   • убирает отдельный 1.2GB-контейнер из стека
//   • даёт более «человечный» фидбэк (полный текст, не только score)
//   • использует те же fallback-тиры, что и весь остальной интервью-пайплайн

// Встроенный банк soft-skill вопросов. Раньше жил в softskills-service.
// Не выносим в JSON-файл, чтобы избежать копий и rebuild'ов — список
// меняется крайне редко.
var softSkillsBank = []string{
	"Расскажите о ситуации, когда вы конструктивно разрешили конфликт в команде. Какие шаги вы предприняли?",
	"Опишите случай, когда вам пришлось работать с дедлайном и неполной информацией. Как вы расставляли приоритеты?",
	"Приведите пример, когда вы получили жёсткий feedback. Как отреагировали и что изменили?",
	"Расскажите про самый сложный проект за последний год. В чём заключалась главная трудность и как вы её преодолели?",
	"Опишите случай, когда вам нужно было убедить коллегу или менеджера принять ваше техническое решение.",
	"Расскажите, как вы делегируете задачи, когда работаете с менее опытным разработчиком.",
	"Опишите, как вы выходите из тупика, когда отладка занимает слишком много времени и непонятно, куда копать.",
	"Расскажите, как вы балансируете между скоростью доставки и качеством кода.",
	"Опишите ситуацию, когда вы ошиблись в production. Как реагировали, что починили, какие выводы сделали?",
	"Приведите пример эффективной коммуникации с не-техническим стейкхолдером (продакт, бизнес, клиент).",
	"Расскажите про самую полезную книгу/курс/выступление по soft-skills за последние 2 года. Что применяете в работе?",
	"Опишите, как вы готовитесь к сложному 1-on-1 с менеджером или подчинённым.",
	"Расскажите про моменты профессионального выгорания и что помогло из него выйти.",
	"Опишите, как вы оцениваете технические долги и аргументируете время на их выплату перед продактом.",
	"Приведите пример успешного code-review, который сильно улучшил решение коллеги без обид.",
}

// pickSoftSkillsQuestion выбирает вопрос, которого ещё не было в сессии.
func pickSoftSkillsQuestion(session *InterviewModuleSession) string {
	asked := map[string]struct{}{}
	if session != nil {
		for _, m := range session.Messages {
			if m.Sender == "ai" {
				asked[strings.TrimSpace(m.Content)] = struct{}{}
			}
		}
	}
	// Перемешиваем индексы и берём первый не-дубликат.
	indices := rand.Perm(len(softSkillsBank))
	for _, i := range indices {
		q := softSkillsBank[i]
		if _, dup := asked[q]; !dup {
			return q
		}
	}
	// Все спрошены — отдаём случайный (поведение «после полного круга»).
	return softSkillsBank[rand.Intn(len(softSkillsBank))]
}

// softSkillsScore — DTO, который раньше отдавал отдельный ML-сервис.
// Сохраняем структуру для бинарной совместимости с остальным кодом.
type softSkillsScoreResponse struct {
	Score    float64 `json:"score"`
	Feedback string  `json:"feedback"`
	Verdict  string  `json:"verdict"`
}

// scoreSoftSkillsAnswer — LLM-оценка soft-skill ответа.
//
// Использует тот же LLM-ключ/URL, что и весь стек (LLM_API_KEY,
// LLM_BASE_URL — DeepSeek по умолчанию). Прямой HTTP-вызов в
// OpenAI-совместимый /chat/completions с response_format=json_object.
//
// Раньше тут был callAIWithFailover, но он dereferences session.*
// (которое у нас nil для standalone-скоринга) → panic. Прямой
// вызов проще и не зависит от внутренних структур интервью.
func (h *Handler) scoreSoftSkillsAnswer(ctx context.Context, question, answer string) (*softSkillsScoreResponse, error) {
	apiKey := strings.TrimSpace(os.Getenv("LLM_API_KEY"))
	apiURL := strings.TrimRight(strings.TrimSpace(os.Getenv("LLM_BASE_URL")), "/")
	model := strings.TrimSpace(os.Getenv("LLM_MODEL"))
	if model == "" {
		model = "deepseek-chat"
	}
	if apiKey == "" || apiURL == "" {
		// Без LLM нечем оценивать — отдадим heuristic.
		return &softSkillsScoreResponse{
			Score:    heuristicScore(answer),
			Verdict:  heuristicVerdict(answer),
			Feedback: "LLM не настроен, использована резервная эвристика по длине ответа.",
		}, nil
	}

	systemPrompt := "Ты опытный карьерный коуч / интервьюер. Оцени ответ кандидата " +
		"на поведенческий (soft-skills) вопрос по шкале 0..100. Учитывай: " +
		"STAR-структуру (situation/task/action/result), конкретику и метрики, " +
		"рефлексию, краткость без воды. Верни строго JSON: " +
		"{\"score\": <0-100>, \"verdict\": \"correct|partial|wrong|skipped|off_topic\", " +
		"\"feedback\": \"<2-3 фразы по-русски: что сильно, что улучшить>\"}."

	body := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": fmt.Sprintf("ВОПРОС:\n%s\n\nОТВЕТ КАНДИДАТА:\n%s",
				strings.TrimSpace(question), strings.TrimSpace(answer))},
		},
		"response_format": map[string]string{"type": "json_object"},
		"max_tokens":      400,
		"temperature":     0.3,
	}
	payload, _ := json.Marshal(body)

	reqCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, apiURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 25 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return &softSkillsScoreResponse{
			Score:    heuristicScore(answer),
			Verdict:  heuristicVerdict(answer),
			Feedback: fmt.Sprintf("LLM вернул %d, использована эвристика.", resp.StatusCode),
		}, nil
	}

	var raw struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	if len(raw.Choices) == 0 {
		return nil, fmt.Errorf("softskills LLM: empty choices")
	}
	score, verdict, feedback := parseScoreJSON(raw.Choices[0].Message.Content)
	if verdict == "" {
		verdict = heuristicVerdict(answer)
	}
	if feedback == "" {
		feedback = "Без подробного комментария."
	}
	return &softSkillsScoreResponse{
		Score:    score,
		Verdict:  verdict,
		Feedback: feedback,
	}, nil
}

// parseScoreJSON пытается достать score/verdict/feedback из произвольного
// LLM-output. Поддерживает «грязный» JSON, обёрнутый в ```json``` блоки.
func parseScoreJSON(raw string) (float64, string, string) {
	// Удаляем markdown-fence и BOM.
	clean := strings.TrimSpace(raw)
	clean = strings.TrimPrefix(clean, "```json")
	clean = strings.TrimPrefix(clean, "```")
	clean = strings.TrimSuffix(clean, "```")
	clean = strings.TrimSpace(clean)

	// Ищем первую JSON-фигурную скобку.
	start := strings.Index(clean, "{")
	end := strings.LastIndex(clean, "}")
	if start < 0 || end <= start {
		return 0, "", ""
	}
	var parsed struct {
		Score    float64 `json:"score"`
		Verdict  string  `json:"verdict"`
		Feedback string  `json:"feedback"`
	}
	if err := json.Unmarshal([]byte(clean[start:end+1]), &parsed); err != nil {
		return 0, "", ""
	}
	return parsed.Score, strings.ToLower(strings.TrimSpace(parsed.Verdict)), parsed.Feedback
}

// heuristicVerdict — резерв, если LLM не отдал structured ответ.
// Не пытается быть умным; просто проверяет, что ответ непустой и
// содержит хоть какую-то длину.
func heuristicVerdict(answer string) string {
	a := strings.TrimSpace(answer)
	if a == "" {
		return "skipped"
	}
	if len(a) < 40 {
		return "wrong"
	}
	if len(a) < 200 {
		return "partial"
	}
	return "correct"
}

// heuristicScore — числовой эквивалент того же эвристического
// разбиения. Используется ТОЛЬКО когда LLM недоступен.
func heuristicScore(answer string) float64 {
	switch heuristicVerdict(answer) {
	case "correct":
		return 75
	case "partial":
		return 55
	case "wrong":
		return 25
	default:
		return 0
	}
}

// isSoftSkillsSession — helper, чтобы handlers ветвились без
// размытых string-сравнений.
func isSoftSkillsSession(session *InterviewModuleSession) bool {
	if session == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(session.InterviewMode), "softskills")
}

// buildSoftSkillsNextQuestion — soft-skills эквивалент requestNextQuestion.
// Оценивает предыдущий ответ через LLM, выбирает следующий вопрос из
// локального банка. Без отдельного ML-контейнера.
func (h *Handler) buildSoftSkillsNextQuestion(
	session *InterviewModuleSession,
	lastAnswer string,
) (*nextQuestionResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	verdict := "none"
	reason := ""
	prevQuestion := ""
	for i := len(session.Messages) - 1; i >= 0; i-- {
		m := session.Messages[i]
		if m.Sender == "ai" {
			prevQuestion = strings.TrimSpace(m.Content)
			break
		}
	}
	if strings.TrimSpace(lastAnswer) != "" && prevQuestion != "" {
		if res, err := h.scoreSoftSkillsAnswer(ctx, prevQuestion, lastAnswer); err == nil && res != nil {
			verdict = res.Verdict
			if res.Feedback != "" {
				reason = fmt.Sprintf("%s · %.0f%%", res.Feedback, res.Score)
			}
		}
	}

	picked := pickSoftSkillsQuestion(session)
	return &nextQuestionResponse{
		Question:          picked,
		Topic:             "soft_skills",
		DifficultyDelta:   0,
		PressureLevel:     1,
		ShouldEnd:         false,
		LastAnswerVerdict: verdict,
		LastAnswerReason:  reason,
	}, nil
}
