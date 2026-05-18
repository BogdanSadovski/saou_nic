package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// softskills-service is a separate Python+PyTorch micro-service that
// hosts the candidate's personal ML model (rubert-tiny2 embeddings +
// custom regressor). interview-service talks to it when a session's
// interview_mode == "softskills" instead of calling the LLM cascade.

func softSkillsServiceURL() string {
	v := os.Getenv("SOFTSKILLS_SERVICE_URL")
	if strings.TrimSpace(v) == "" {
		return "http://softskills-service:8090"
	}
	return strings.TrimRight(v, "/")
}

// softSkillsScoreRequest is the JSON we send to /api/v1/score.
type softSkillsScoreRequest struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type softSkillsScoreResponse struct {
	Score    float64 `json:"score"`
	Feedback string  `json:"feedback"`
	Verdict  string  `json:"verdict"`
}

type softSkillsQuestionsResponse struct {
	Questions []string `json:"questions"`
}

// fetchSoftSkillsQuestions pulls N random soft-skill questions from the
// ML service. Falls back to an empty slice if the service is down —
// caller decides how to recover.
func (h *Handler) fetchSoftSkillsQuestions(ctx context.Context, n int) ([]string, error) {
	if n <= 0 {
		n = 5
	}
	url := fmt.Sprintf("%s/api/v1/questions?n=%d", softSkillsServiceURL(), n)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("softskills-service: status %d", resp.StatusCode)
	}
	var out softSkillsQuestionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Questions, nil
}

// scoreSoftSkillsAnswer asks the ML service to rate a single
// question/answer pair. Returns score in [0, 100] plus a short
// feedback string and a verdict bucket (correct/partial/wrong/skipped).
func (h *Handler) scoreSoftSkillsAnswer(ctx context.Context, question, answer string) (*softSkillsScoreResponse, error) {
	body, _ := json.Marshal(softSkillsScoreRequest{Question: question, Answer: answer})
	url := softSkillsServiceURL() + "/api/v1/score"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("softskills-service score: status %d", resp.StatusCode)
	}
	var out softSkillsScoreResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

// isSoftSkillsSession is a small helper so handlers can branch
// without sprinkling string comparisons everywhere.
func isSoftSkillsSession(session *InterviewModuleSession) bool {
	if session == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(session.InterviewMode), "softskills")
}

// buildSoftSkillsNextQuestion is the soft-skills equivalent of
// `requestNextQuestion`. It scores the previous answer via the ML
// regressor, then pulls the next question from the same service. If
// the service is unreachable, returns a transparent system message
// so the user knows to start the softskills-service container.
func (h *Handler) buildSoftSkillsNextQuestion(
	session *InterviewModuleSession,
	lastAnswer string,
) (*nextQuestionResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// 1. Score the previous answer (if any) — drives the verdict badge.
	//    The "current question" for the previous turn is the most
	//    recent AI message in the chat history.
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
			reason = fmt.Sprintf("%s · %.0f%%", res.Feedback, res.Score)
		}
	}

	// 2. Pull next question from the bank, avoiding repeats within
	//    the session by asking for a small pool and skipping any
	//    we've already shown.
	questions, err := h.fetchSoftSkillsQuestions(ctx, 8)
	if err != nil || len(questions) == 0 {
		return &nextQuestionResponse{
			Question: "🤖 Сервис soft-skills недоступен. Убедитесь, что контейнер `softskills-service` запущен " +
				"(`docker compose ... up -d softskills-service`). После запуска нажмите «Следующий вопрос».",
			Topic:             "soft_skills",
			DifficultyDelta:   0,
			PressureLevel:     1,
			ShouldEnd:         false,
			LastAnswerVerdict: "skipped",
			LastAnswerReason:  "softskills-service offline",
		}, nil
	}

	asked := map[string]struct{}{}
	for _, m := range session.Messages {
		if m.Sender == "ai" {
			asked[strings.TrimSpace(m.Content)] = struct{}{}
		}
	}
	picked := ""
	for _, q := range questions {
		q = strings.TrimSpace(q)
		if q == "" {
			continue
		}
		if _, dup := asked[q]; dup {
			continue
		}
		picked = q
		break
	}
	if picked == "" {
		picked = strings.TrimSpace(questions[0])
	}

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
