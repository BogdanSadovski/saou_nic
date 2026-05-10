package api

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/interview-platform/interview-service/internal/domain"
	"github.com/interview-platform/interview-service/internal/repository"
	"github.com/interview-platform/interview-service/internal/service"
	"github.com/interview-platform/interview-service/pkg/codeexecutor"
	pdf "github.com/ledongthuc/pdf"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

type Handler struct {
	interviewService  *service.InterviewService
	sessionManager    *service.SessionManager
	moduleStore       modulePersistence
	repo              dataPersistence
	codeExecutor      codeExecutorClient
	logger            *logrus.Logger
	redis             *redis.Client
	resumeMu          sync.RWMutex
	resumeHistory     map[string][]resumeImportResponse
	moduleMu          sync.RWMutex
	moduleSessions    map[uuid.UUID]*InterviewModuleSession
	moduleReports     map[uuid.UUID]*InterviewModuleReport
	moduleWS          map[uuid.UUID]map[*websocket.Conn]struct{}
	requestLog        map[string]string
	aiServiceURL      string
	secondaryAIURL    string
	scoringServiceURL string
	scoreQueue        chan scoreJob
	cbMu              sync.Mutex
	aiCircuit         aiCircuitState
	llmLatencyMs      atomic.Int64
	policyViolations  atomic.Int64
	fallbackRate      atomic.Int64
	wsReconnectCount  atomic.Int64
	reportGenMs       atomic.Int64
	reconnectAttempts atomic.Int64
	reconnectSuccess  atomic.Int64
	metricsMu         sync.Mutex
	llmLatencySamples []int64
	reportLatency     []int64
}

type modulePersistence interface {
	CreateInterviewModuleSession(ctx context.Context, session *domain.InterviewModuleSession) error
	GetInterviewModuleSessionByID(ctx context.Context, id uuid.UUID) (*domain.InterviewModuleSession, error)
	ListInterviewModuleSessionsByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.InterviewModuleSession, error)
	UpdateInterviewModuleSession(ctx context.Context, session *domain.InterviewModuleSession) error
	CreateInterviewModuleMessage(ctx context.Context, message *domain.InterviewModuleMessage) error
	ListInterviewModuleMessagesBySessionID(ctx context.Context, sessionID uuid.UUID, limit, offset int) ([]*domain.InterviewModuleMessage, error)
	UpsertInterviewModuleReport(ctx context.Context, report *domain.InterviewModuleReport) error
	GetInterviewModuleReportBySessionID(ctx context.Context, sessionID uuid.UUID) (*domain.InterviewModuleReport, error)
}

type dataPersistence interface {
	CreateCodeSubmission(ctx context.Context, submission *repository.CodeSubmission) (*repository.CodeSubmission, error)
	CreateCodeExecutionResult(ctx context.Context, result *repository.CodeExecutionResult) (*repository.CodeExecutionResult, error)
	ListCodeSubmissionsBySession(ctx context.Context, sessionID uuid.UUID) ([]*repository.CodeSubmission, error)
	AddCollaborator(ctx context.Context, sessionID, userID uuid.UUID, role domain.CollaboratorRole) (*domain.InterviewCollaborator, error)
	ListCollaborators(ctx context.Context, sessionID uuid.UUID) ([]*domain.InterviewCollaborator, error)
	AddNote(ctx context.Context, note *domain.CollaborationNote) (*domain.CollaborationNote, error)
	ListNotes(ctx context.Context, sessionID uuid.UUID, limit, offset int) ([]*domain.CollaborationNote, error)
	SubmitScore(ctx context.Context, score *domain.InterviewerScore) (*domain.InterviewerScore, error)
	GetScores(ctx context.Context, sessionID uuid.UUID) ([]*domain.InterviewerScore, error)
	CalculateConsensus(ctx context.Context, sessionID uuid.UUID) (*domain.InterviewConsensus, error)
	GetConsensus(ctx context.Context, sessionID uuid.UUID) (*domain.InterviewConsensus, error)
}

type codeExecutorClient interface {
	Execute(ctx context.Context, req *codeexecutor.CodeExecutionRequest) (*codeexecutor.CodeExecutionResult, error)
	HealthCheck(ctx context.Context) error
}

type scoreJob struct {
	SessionID uuid.UUID
	Feedback  string
}

type aiCircuitState struct {
	Failures  int
	OpenUntil time.Time
}

type githubAPIError struct {
	StatusCode int
	Message    string
}

func (e *githubAPIError) Error() string {
	if strings.TrimSpace(e.Message) == "" {
		return fmt.Sprintf("github api status: %d", e.StatusCode)
	}
	return fmt.Sprintf("github api status: %d, message: %s", e.StatusCode, e.Message)
}

func NewHandler(interviewService *service.InterviewService, sessionManager *service.SessionManager, moduleStore modulePersistence, repo dataPersistence, codeExecutor codeExecutorClient, redisClient *redis.Client, logger *logrus.Logger) *Handler {
	aiURL := os.Getenv("AI_SERVICE_URL")
	if aiURL == "" {
		aiURL = "http://ai-service:8001"
	}

	secondaryAIURL := strings.TrimSpace(os.Getenv("SECONDARY_AI_SERVICE_URL"))

	scoringURL := os.Getenv("SCORING_SERVICE_URL")
	if scoringURL == "" {
		scoringURL = "http://scoring-service:8080"
	}

	h := &Handler{
		interviewService:  interviewService,
		sessionManager:    sessionManager,
		moduleStore:       moduleStore,
		repo:              repo,
		codeExecutor:      codeExecutor,
		redis:             redisClient,
		logger:            logger,
		resumeHistory:     make(map[string][]resumeImportResponse),
		moduleSessions:    make(map[uuid.UUID]*InterviewModuleSession),
		moduleReports:     make(map[uuid.UUID]*InterviewModuleReport),
		moduleWS:          make(map[uuid.UUID]map[*websocket.Conn]struct{}),
		requestLog:        make(map[string]string),
		aiServiceURL:      strings.TrimRight(aiURL, "/"),
		secondaryAIURL:    strings.TrimRight(secondaryAIURL, "/"),
		scoringServiceURL: strings.TrimRight(scoringURL, "/"),
		scoreQueue:        make(chan scoreJob, 256),
	}

	go h.runScoringWorker()
	return h
}

type InterviewModuleSession struct {
	SessionID        uuid.UUID              `json:"session_id"`
	UserID           string                 `json:"user_id"`
	Role             string                 `json:"role"`
	VacancyTitle     string                 `json:"vacancy_title,omitempty"`
	VacancyCategory  string                 `json:"vacancy_category,omitempty"`
	InterviewMode    string                 `json:"interview_mode,omitempty"`
	FocusAreas       []string               `json:"focus_areas,omitempty"`
	PrimarySkills    []string               `json:"primary_skills,omitempty"`
	TheoryFocus      []string               `json:"theory_focus,omitempty"`
	PracticeFocus    []string               `json:"practice_focus,omitempty"`
	Level            string                 `json:"level"`
	Status           string                 `json:"status"`
	DurationMinutes  int                    `json:"duration_minutes"`
	QuestionLimit    int                    `json:"question_limit"`
	CurrentTopic     string                 `json:"current_topic"`
	Difficulty       int                    `json:"difficulty"`
	PressureLevel    int                    `json:"pressure_level"`
	TopicStats       map[string]int         `json:"topic_stats,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
	StartedAt        time.Time              `json:"started_at"`
	ExpiresAt        time.Time              `json:"expires_at"`
	FinishedAt       *time.Time             `json:"finished_at,omitempty"`
	Messages         []InterviewChatMessage `json:"messages"`
	WeakAnswerStreak int                    `json:"-"`
	TopicCursor      int                    `json:"-"`
}

type InterviewChatMessage struct {
	MessageID  uuid.UUID `json:"message_id"`
	Sender     string    `json:"sender"`
	Content    string    `json:"content"`
	Topic      string    `json:"topic,omitempty"`
	Difficulty int       `json:"difficulty,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	// AI verdict attached after the next-question AI call evaluates
	// THIS message. Only set on Sender=="user". UI renders ✅/⚠️/❌
	// badge based on Verdict, with VerdictReason as tooltip.
	Verdict       string `json:"verdict,omitempty"`
	VerdictReason string `json:"verdict_reason,omitempty"`
}

type InterviewModuleReport struct {
	SessionID       uuid.UUID `json:"session_id"`
	Correctness     float64   `json:"correctness"`
	Clarity         float64   `json:"clarity"`
	Completeness    float64   `json:"completeness"`
	Relevance       float64   `json:"relevance"`
	OverallScore    float64   `json:"overall_score"`
	Strengths       []string  `json:"strengths"`
	Weaknesses      []string  `json:"weaknesses"`
	Recommendations []string  `json:"recommendations"`
	GeneratedAt     time.Time `json:"generated_at"`
}

type userInterviewTotals struct {
	TotalInterviews      int     `json:"total_interviews"`
	CompletedInterviews  int     `json:"completed_interviews"`
	InProgressInterviews int     `json:"in_progress_interviews"`
	ExpiredInterviews    int     `json:"expired_interviews"`
	CompletionRate       float64 `json:"completion_rate"`
}

type userInterviewPerformance struct {
	AverageScore      float64 `json:"average_score"`
	BestScore         float64 `json:"best_score"`
	LatestScore       float64 `json:"latest_score"`
	ReportsGenerated  int     `json:"reports_generated"`
	AvgQuestionCount  float64 `json:"avg_question_count"`
	AvgSessionMinutes float64 `json:"avg_session_minutes"`
}

type userInterviewTimelinePoint struct {
	Date      string `json:"date"`
	Started   int    `json:"started"`
	Completed int    `json:"completed"`
}

type userInterviewEntry struct {
	SessionID       uuid.UUID  `json:"session_id"`
	Role            string     `json:"role"`
	Level           string     `json:"level"`
	VacancyTitle    string     `json:"vacancy_title,omitempty"`
	InterviewMode   string     `json:"interview_mode"`
	Status          string     `json:"status"`
	CurrentTopic    string     `json:"current_topic,omitempty"`
	DurationMinutes int        `json:"duration_minutes"`
	QuestionLimit   int        `json:"question_limit"`
	MessagesTotal   int        `json:"messages_total"`
	AIMessages      int        `json:"ai_messages"`
	UserMessages    int        `json:"user_messages"`
	StartedAt       time.Time  `json:"started_at"`
	ExpiresAt       time.Time  `json:"expires_at"`
	FinishedAt      *time.Time `json:"finished_at,omitempty"`
	OverallScore    *float64   `json:"overall_score,omitempty"`
	Strengths       []string   `json:"strengths,omitempty"`
	Weaknesses      []string   `json:"weaknesses,omitempty"`
}

type userInterviewAnalyticsReport struct {
	UserID               string                       `json:"user_id"`
	GeneratedAt          time.Time                    `json:"generated_at"`
	Totals               userInterviewTotals          `json:"totals"`
	Performance          userInterviewPerformance     `json:"performance"`
	RoleDistribution     []githubChartPoint           `json:"role_distribution"`
	ModeDistribution     []githubChartPoint           `json:"mode_distribution"`
	Timeline             []userInterviewTimelinePoint `json:"timeline"`
	TopStrengths         []string                     `json:"top_strengths"`
	TopWeaknesses        []string                     `json:"top_weaknesses"`
	TopRecommendations   []string                     `json:"top_recommendations"`
	CompletedInterviews  []userInterviewEntry         `json:"completed_interviews"`
	IncompleteInterviews []userInterviewEntry         `json:"incomplete_interviews"`
	RecentInterviews     []userInterviewEntry         `json:"recent_interviews"`
}

type nextQuestionRequest struct {
	Role                string                 `json:"role"`
	Level               string                 `json:"level"`
	VacancyTitle        string                 `json:"vacancy_title,omitempty"`
	VacancyCategory     string                 `json:"vacancy_category,omitempty"`
	InterviewMode       string                 `json:"interview_mode,omitempty"`
	SessionContext      string                 `json:"session_context,omitempty"`
	RecentTopics        []string               `json:"recent_topics,omitempty"`
	FocusAreas          []string               `json:"focus_areas,omitempty"`
	PrimarySkills       []string               `json:"primary_skills,omitempty"`
	TheoryFocus         []string               `json:"theory_focus,omitempty"`
	PracticeFocus       []string               `json:"practice_focus,omitempty"`
	CurrentTopic        string                 `json:"current_topic,omitempty"`
	Difficulty          int                    `json:"difficulty"`
	PressureLevel       int                    `json:"pressure_level"`
	TimeLeftSec         int64                  `json:"time_left_sec"`
	QuestionsLeft       int                    `json:"questions_left"`
	LastCandidateAnswer string                 `json:"last_candidate_answer,omitempty"`
	History             []InterviewChatMessage `json:"history,omitempty"`
	AvoidQuestions      []string               `json:"avoid_questions,omitempty"`
	TurnNonce           string                 `json:"turn_nonce,omitempty"`
}

type nextQuestionResponse struct {
	Question        string          `json:"question"`
	Topic           string          `json:"topic"`
	DifficultyDelta int             `json:"difficulty_delta"`
	PressureLevel   int             `json:"pressure_level"`
	ShouldEnd       bool            `json:"should_end"`
	Flags           map[string]bool `json:"flags,omitempty"`
	// AI verdict on the candidate's previous answer:
	// correct / partial / wrong / skipped / off_topic / none.
	// Surfaced to the chat UI as a per-message badge so the
	// candidate immediately sees if the answer landed.
	LastAnswerVerdict string `json:"last_answer_verdict,omitempty"`
	LastAnswerReason  string `json:"last_answer_reason,omitempty"`
}

type createInterviewModuleSessionRequest struct {
	Role            string   `json:"role"`
	Level           string   `json:"level"`
	DurationMinutes int      `json:"duration_minutes"`
	QuestionLimit   int      `json:"question_limit"`
	ResumeID        string   `json:"resume_id"`
	GitHubProfile   string   `json:"github_profile"`
	VacancyTitle    string   `json:"vacancy_title"`
	VacancyCategory string   `json:"vacancy_category"`
	InterviewMode   string   `json:"interview_mode"`
	FocusAreas      []string `json:"focus_areas"`
	PrimarySkills   []string `json:"primary_skills"`
	TheoryFocus     []string `json:"theory_focus"`
	PracticeFocus   []string `json:"practice_focus"`
}

type githubImportRequest struct {
	ProfileURL      string   `json:"profile_url"`
	MaxRepos        int      `json:"max_repos,omitempty"`
	RolePreferences []string `json:"role_preferences,omitempty"`
}

type githubChartPoint struct {
	Label string `json:"label"`
	Value int    `json:"value"`
}

type githubContributionDay struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

type githubTopRepository struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
	Language    string `json:"language,omitempty"`
	Stars       int    `json:"stars"`
	Forks       int    `json:"forks"`
	OpenIssues  int    `json:"open_issues"`
	LastPush    string `json:"last_push,omitempty"`
}

type githubImportStats struct {
	Followers       int `json:"followers"`
	Following       int `json:"following"`
	PublicRepos     int `json:"public_repos"`
	SampledRepos    int `json:"sampled_repos"`
	TotalStars      int `json:"total_stars"`
	TotalForks      int `json:"total_forks"`
	TotalOpenIssues int `json:"total_open_issues"`
}

type githubImportCharts struct {
	LanguageDistribution []githubChartPoint      `json:"language_distribution"`
	MonthlyActivity      []githubChartPoint      `json:"monthly_activity"`
	ContributionDays     []githubContributionDay `json:"contribution_days"`
}

type githubRoleRecommendation struct {
	Role      string `json:"role"`
	FitScore  int    `json:"fit_score"`
	Rationale string `json:"rationale"`
}

type githubLanguageInsight struct {
	Language        string   `json:"language"`
	Confidence      int      `json:"confidence"`
	Evidence        string   `json:"evidence"`
	InterviewTopics []string `json:"interview_topics"`
}

type githubInterviewTrack struct {
	Role            string   `json:"role"`
	Mode            string   `json:"mode"`
	Level           string   `json:"level"`
	DurationMinutes int      `json:"duration_minutes"`
	FocusAreas      []string `json:"focus_areas"`
	PrimarySkills   []string `json:"primary_skills"`
	Rationale       string   `json:"rationale"`
}

type githubAIInsights struct {
	Summary              string                     `json:"summary"`
	Strengths            []string                   `json:"strengths"`
	Risks                []string                   `json:"risks"`
	ActionPlan           []string                   `json:"action_plan"`
	LanguageInsights     []githubLanguageInsight    `json:"language_insights"`
	InterviewTracks      []githubInterviewTrack     `json:"interview_tracks"`
	RecommendedPositions []githubRoleRecommendation `json:"recommended_positions"`
}

type githubImportResponse struct {
	Username        string                `json:"username"`
	ProfileURL      string                `json:"profile_url"`
	ProfileName     string                `json:"profile_name,omitempty"`
	Bio             string                `json:"bio,omitempty"`
	AvatarURL       string                `json:"avatar_url,omitempty"`
	Stats           githubImportStats     `json:"stats"`
	Charts          githubImportCharts    `json:"charts"`
	TopRepositories []githubTopRepository `json:"top_repositories"`
	AIInsights      githubAIInsights      `json:"ai_insights"`
}

type githubUserAPIResponse struct {
	Login       string `json:"login"`
	Name        string `json:"name"`
	Bio         string `json:"bio"`
	Followers   int    `json:"followers"`
	Following   int    `json:"following"`
	PublicRepos int    `json:"public_repos"`
	HTMLURL     string `json:"html_url"`
	AvatarURL   string `json:"avatar_url"`
}

type githubRepoAPIResponse struct {
	Name            string `json:"name"`
	HTMLURL         string `json:"html_url"`
	Description     string `json:"description"`
	Language        string `json:"language"`
	StargazersCount int    `json:"stargazers_count"`
	ForksCount      int    `json:"forks_count"`
	OpenIssuesCount int    `json:"open_issues_count"`
	PushedAt        string `json:"pushed_at"`
	Archived        bool   `json:"archived"`
	Fork            bool   `json:"fork"`
}

type developerInsightsRequest struct {
	GitHubUsername       string                `json:"github_username"`
	ProfileName          string                `json:"profile_name,omitempty"`
	Bio                  string                `json:"bio,omitempty"`
	RolePreferences      []string              `json:"role_preferences,omitempty"`
	Followers            int                   `json:"followers"`
	Following            int                   `json:"following"`
	PublicRepos          int                   `json:"public_repos"`
	SampledRepos         int                   `json:"sampled_repos"`
	TotalStars           int                   `json:"total_stars"`
	TotalForks           int                   `json:"total_forks"`
	TotalOpenIssues      int                   `json:"total_open_issues"`
	LanguageDistribution []githubChartPoint    `json:"language_distribution,omitempty"`
	MonthlyActivity      []githubChartPoint    `json:"monthly_activity,omitempty"`
	TopRepositories      []githubTopRepository `json:"top_repositories,omitempty"`
}

type resumeImportStats struct {
	WordCount         int `json:"word_count"`
	CharacterCount    int `json:"character_count"`
	EstimatedPages    int `json:"estimated_pages"`
	SkillsCount       int `json:"skills_count"`
	LanguageCount     int `json:"language_count"`
	ExperienceEntries int `json:"experience_entries"`
	EducationEntries  int `json:"education_entries"`
}

type resumeImportCharts struct {
	LanguageDistribution []githubChartPoint `json:"language_distribution"`
	SkillsDistribution   []githubChartPoint `json:"skills_distribution"`
}

type resumeAIInsights struct {
	Summary              string                     `json:"summary"`
	StrongPoints         []string                   `json:"strong_points"`
	ImprovementPoints    []string                   `json:"improvement_points"`
	ActionPlan           []string                   `json:"action_plan"`
	LanguageInsights     []githubLanguageInsight    `json:"language_insights"`
	InterviewTracks      []githubInterviewTrack     `json:"interview_tracks"`
	RecommendedPositions []githubRoleRecommendation `json:"recommended_positions"`
}

type resumeImportResponse struct {
	ReportID         string                  `json:"report_id"`
	CreatedAt        time.Time               `json:"created_at"`
	FileName         string                  `json:"file_name"`
	FileSize         int64                   `json:"file_size"`
	ContentType      string                  `json:"content_type"`
	DetectedFormat   string                  `json:"detected_format"`
	Stats            resumeImportStats       `json:"stats"`
	Charts           resumeImportCharts      `json:"charts"`
	ExtractedSkills  []string                `json:"extracted_skills"`
	ProcessingStages []resumeProcessingStage `json:"processing_stages"`
	AIInsights       resumeAIInsights        `json:"ai_insights"`
}

type resumeProcessingStage struct {
	Code       string `json:"code"`
	Title      string `json:"title"`
	Status     string `json:"status"`
	DurationMs int64  `json:"duration_ms"`
}

type resumeInsightsRequest struct {
	FileName          string   `json:"file_name"`
	ContentType       string   `json:"content_type,omitempty"`
	RolePreferences   []string `json:"role_preferences,omitempty"`
	WordCount         int      `json:"word_count"`
	CharacterCount    int      `json:"character_count"`
	Skills            []string `json:"skills,omitempty"`
	Languages         []string `json:"languages,omitempty"`
	ExperienceEntries int      `json:"experience_entries"`
	EducationEntries  int      `json:"education_entries"`
	TextExcerpt       string   `json:"text_excerpt,omitempty"`
}

type resumeAIInsightsPayload struct {
	Summary              string                     `json:"summary"`
	StrongPoints         []string                   `json:"strong_points"`
	ImprovementPoints    []string                   `json:"improvement_points"`
	ActionPlan           []string                   `json:"action_plan"`
	LanguageInsights     []githubLanguageInsight    `json:"language_insights"`
	InterviewTracks      []githubInterviewTrack     `json:"interview_tracks"`
	RecommendedPositions []githubRoleRecommendation `json:"recommended_positions"`
}

type resumeExtractionResult struct {
	Text              string
	DetectedFormat    string
	NormalizedType    string
	Skills            []string
	Languages         []string
	SkillDistribution []githubChartPoint
	LangDistribution  []githubChartPoint
	ExperienceEntries int
	EducationEntries  int
}

var resumeSkillKeywords = map[string][]string{
	"Go":         {"golang", " go ", "go/"},
	"Python":     {"python", "django", "fastapi", "flask"},
	"Java":       {"java", "spring", "kafka"},
	"JavaScript": {"javascript", "node.js", "nodejs", "vue", "angular"},
	"TypeScript": {"typescript", "ts-node", "nest"},
	"React":      {"react", "redux", "next.js", "nextjs"},
	"SQL":        {"sql", "postgres", "mysql", "sqlite"},
	"Docker":     {"docker", "container", "compose"},
	"Kubernetes": {"kubernetes", "k8s", "helm"},
	"AWS":        {"aws", "s3", "ec2", "lambda", "eks"},
	"CI/CD":      {"ci/cd", "gitlab ci", "github actions", "jenkins"},
	"Testing":    {"testing", "pytest", "jest", "unit test", "integration test"},
}

var programmingLanguagesOrder = []string{
	"Go", "Python", "Java", "JavaScript", "TypeScript", "C#", "C++", "Rust", "Kotlin", "Swift", "PHP", "Ruby",
}

// Response represents a standard API response
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeSuccess(w http.ResponseWriter, status int, data interface{}) {
	writeJSON(w, status, Response{
		Success: true,
		Data:    data,
	})
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, Response{
		Success: false,
		Error:   message,
	})
}

func writeServerError(w http.ResponseWriter, err error) {
	writeError(w, http.StatusInternalServerError, "internal server error")
}

func (h *Handler) parseUUIDVar(w http.ResponseWriter, r *http.Request, key string) (uuid.UUID, bool) {
	vars := mux.Vars(r)
	value := vars[key]
	parsed, err := uuid.Parse(value)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid %s", strings.ReplaceAll(key, "_", " ")))
		return uuid.Nil, false
	}
	return parsed, true
}

func normalizeInterviewMode(mode string) string {
	normalized := strings.ToLower(strings.TrimSpace(mode))
	switch normalized {
	case "practice", "theory":
		return normalized
	default:
		return "practice"
	}
}

func copyStringsWithFallback(values []string, fallback ...string) []string {
	if len(values) > 0 {
		return append([]string(nil), values...)
	}
	if len(fallback) == 0 {
		return nil
	}
	return append([]string(nil), fallback...)
}

func (h *Handler) buildInterviewModuleSession(userID string, req createInterviewModuleSessionRequest, sessionID uuid.UUID, now time.Time, initialDifficulty, initialPressure, topicSeed int) *InterviewModuleSession {
	session := &InterviewModuleSession{
		SessionID:       sessionID,
		UserID:          userID,
		Role:            req.Role,
		VacancyTitle:    strings.TrimSpace(req.VacancyTitle),
		VacancyCategory: strings.TrimSpace(req.VacancyCategory),
		InterviewMode:   normalizeInterviewMode(req.InterviewMode),
		FocusAreas:      copyStringsWithFallback(req.FocusAreas),
		PrimarySkills:   copyStringsWithFallback(req.PrimarySkills),
		TheoryFocus:     copyStringsWithFallback(req.TheoryFocus),
		PracticeFocus:   copyStringsWithFallback(req.PracticeFocus),
		Level:           req.Level,
		Status:          "active",
		DurationMinutes: req.DurationMinutes,
		QuestionLimit:   req.QuestionLimit,
		CurrentTopic:    h.nextTopic(req.Role, topicSeed-1),
		Difficulty:      initialDifficulty,
		PressureLevel:   initialPressure,
		TopicStats:      map[string]int{},
		CreatedAt:       now,
		StartedAt:       now,
		ExpiresAt:       now.Add(time.Duration(req.DurationMinutes) * time.Minute),
		Messages:        make([]InterviewChatMessage, 0, req.QuestionLimit*2),
		TopicCursor:     topicSeed,
	}

	if session.VacancyCategory == "" {
		session.VacancyCategory = h.roleKey(req.Role)
	}
	if session.InterviewMode == "" {
		session.InterviewMode = "practice"
	}
	if len(session.FocusAreas) == 0 {
		session.FocusAreas = []string{session.VacancyTitle, session.VacancyCategory}
	}
	if len(session.PrimarySkills) == 0 {
		session.PrimarySkills = []string{session.VacancyCategory}
	}

	return session
}

func (h *Handler) requestInterviewQuestionFromAI(session *InterviewModuleSession, topic string, lastAnswer string, mode string) (*nextQuestionResponse, error) {
	if session == nil {
		return nil, fmt.Errorf("session is nil")
	}

	normalizedMode := strings.ToLower(strings.TrimSpace(mode))
	if normalizedMode == "" {
		normalizedMode = strings.ToLower(strings.TrimSpace(session.InterviewMode))
	}
	if normalizedMode == "" {
		normalizedMode = "practice"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	timeLeft := time.Until(session.ExpiresAt)
	if timeLeft < 0 {
		timeLeft = 0
	}

	questionsLeft := session.QuestionLimit - h.countAIMessages(session.Messages)
	if questionsLeft < 0 {
		questionsLeft = 0
	}

	body := nextQuestionRequest{
		Role:                session.Role,
		Level:               session.Level,
		VacancyTitle:        session.VacancyTitle,
		VacancyCategory:     session.VacancyCategory,
		InterviewMode:       normalizedMode,
		SessionContext:      h.buildInterviewSessionContext(session, topic, lastAnswer, normalizedMode),
		RecentTopics:        h.recentAITopics(session, 5),
		FocusAreas:          session.FocusAreas,
		PrimarySkills:       session.PrimarySkills,
		TheoryFocus:         session.TheoryFocus,
		PracticeFocus:       session.PracticeFocus,
		CurrentTopic:        topic,
		Difficulty:          session.Difficulty,
		PressureLevel:       session.PressureLevel,
		TimeLeftSec:         int64(timeLeft / time.Second),
		QuestionsLeft:       questionsLeft,
		LastCandidateAnswer: h.sanitizeCandidateText(lastAnswer),
		History:             session.Messages,
		AvoidQuestions:      h.collectAvoidQuestions(ctx, session, 20),
		TurnNonce:           uuid.NewString(),
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	out, err := h.callAIWithFailover(ctx, session, payload)
	if err != nil {
		return nil, err
	}
	if out == nil {
		return nil, fmt.Errorf("empty ai response")
	}
	if strings.TrimSpace(out.Topic) == "" {
		out.Topic = topic
	}
	return out, nil
}

// ==================== Interview Handlers ====================

func (h *Handler) CreateInterview(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateInterviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	interview, err := h.interviewService.CreateInterview(r.Context(), &req)
	if err != nil {
		h.logger.WithError(err).Error("failed to create interview")
		writeServerError(w, err)
		return
	}

	writeSuccess(w, http.StatusCreated, interview)
}

func (h *Handler) GetInterview(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseUUIDVar(w, r, "id")
	if !ok {
		return
	}

	response, err := h.interviewService.GetInterview(r.Context(), id)
	if err != nil {
		h.logger.WithError(err).Error("failed to get interview")
		writeError(w, http.StatusNotFound, "interview not found")
		return
	}

	writeSuccess(w, http.StatusOK, response)
}

func (h *Handler) ListInterviews(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	limit := 20
	offset := 0

	if l := query.Get("limit"); l != "" {
		parsed, err := strconv.Atoi(l)
		if err != nil || parsed <= 0 {
			writeError(w, http.StatusBadRequest, "invalid limit")
			return
		}
		limit = parsed
	}

	if o := query.Get("offset"); o != "" {
		parsed, err := strconv.Atoi(o)
		if err != nil || parsed < 0 {
			writeError(w, http.StatusBadRequest, "invalid offset")
			return
		}
		offset = parsed
	}

	var interviewerID, candidateID *uuid.UUID
	var status *domain.InterviewStatus

	if iid := query.Get("interviewer_id"); iid != "" {
		id, err := uuid.Parse(iid)
		if err == nil {
			interviewerID = &id
		}
	}

	if cid := query.Get("candidate_id"); cid != "" {
		id, err := uuid.Parse(cid)
		if err == nil {
			candidateID = &id
		}
	}

	if s := query.Get("status"); s != "" {
		st := domain.InterviewStatus(s)
		status = &st
	}

	interviews, err := h.interviewService.ListInterviews(r.Context(), interviewerID, candidateID, status, limit, offset)
	if err != nil {
		h.logger.WithError(err).Error("failed to list interviews")
		writeServerError(w, err)
		return
	}

	writeSuccess(w, http.StatusOK, interviews)
}

func (h *Handler) UpdateInterview(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseUUIDVar(w, r, "id")
	if !ok {
		return
	}

	var req domain.UpdateInterviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	interview, err := h.interviewService.UpdateInterview(r.Context(), id, &req)
	if err != nil {
		h.logger.WithError(err).Error("failed to update interview")
		writeServerError(w, err)
		return
	}

	writeSuccess(w, http.StatusOK, interview)
}

func (h *Handler) DeleteInterview(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseUUIDVar(w, r, "id")
	if !ok {
		return
	}

	if err := h.interviewService.DeleteInterview(r.Context(), id); err != nil {
		h.logger.WithError(err).Error("failed to delete interview")
		writeServerError(w, err)
		return
	}

	writeSuccess(w, http.StatusOK, nil)
}

func (h *Handler) CancelInterview(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseUUIDVar(w, r, "id")
	if !ok {
		return
	}

	if err := h.interviewService.CancelInterview(r.Context(), id); err != nil {
		h.logger.WithError(err).Error("failed to cancel interview")
		writeServerError(w, err)
		return
	}

	writeSuccess(w, http.StatusOK, nil)
}

// ==================== Session Handlers ====================

func (h *Handler) StartSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	interviewID, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid interview id")
		return
	}

	session, err := h.sessionManager.StartSession(r.Context(), interviewID)
	if err != nil {
		h.logger.WithError(err).Error("failed to start session")
		writeServerError(w, err)
		return
	}

	writeSuccess(w, http.StatusCreated, session)
}

func (h *Handler) GetSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID, err := uuid.Parse(vars["session_id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid session id")
		return
	}

	session, err := h.sessionManager.GetSession(r.Context(), sessionID)
	if err != nil {
		h.logger.WithError(err).Error("failed to get session")
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	writeSuccess(w, http.StatusOK, session)
}

func (h *Handler) SubmitAnswer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID, err := uuid.Parse(vars["session_id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid session id")
		return
	}

	var req struct {
		QuestionID uuid.UUID `json:"question_id"`
		Code       string    `json:"code"`
		Language   string    `json:"language"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	answer, err := h.sessionManager.SubmitAnswer(r.Context(), sessionID, req.QuestionID, req.Code, req.Language)
	if err != nil {
		h.logger.WithError(err).Error("failed to submit answer")
		writeServerError(w, err)
		return
	}

	writeSuccess(w, http.StatusCreated, answer)
}

func (h *Handler) EndSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID, err := uuid.Parse(vars["session_id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid session id")
		return
	}

	var req struct {
		Feedback string `json:"feedback"`
	}

	json.NewDecoder(r.Body).Decode(&req)

	session, err := h.sessionManager.EndSession(r.Context(), sessionID, req.Feedback)
	if err != nil {
		h.logger.WithError(err).Error("failed to end session")
		writeServerError(w, err)
		return
	}

	writeSuccess(w, http.StatusOK, session)
}

func (h *Handler) GetSessionResults(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID, err := uuid.Parse(vars["session_id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid session id")
		return
	}

	results, err := h.sessionManager.GetSessionResults(r.Context(), sessionID)
	if err != nil {
		h.logger.WithError(err).Error("failed to get session results")
		writeServerError(w, err)
		return
	}

	writeSuccess(w, http.StatusOK, results)
}

// HealthCheck returns the health status of the service
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	llmP95 := h.p95(h.llmLatencySamples)
	reportP95 := h.p95(h.reportLatency)
	reconnectRate := h.successRate(h.reconnectSuccess.Load(), h.reconnectAttempts.Load())

	writeSuccess(w, http.StatusOK, map[string]interface{}{
		"status":  "healthy",
		"service": "interview-service",
		"metrics": map[string]int64{
			"llm_latency_ms":            h.llmLatencyMs.Load(),
			"policy_violation_count":    h.policyViolations.Load(),
			"fallback_rate":             h.fallbackRate.Load(),
			"ws_reconnect_count":        h.wsReconnectCount.Load(),
			"report_generation_time_ms": h.reportGenMs.Load(),
		},
		"slo": map[string]interface{}{
			"p95_question_generation_ms": llmP95,
			"p95_report_generation_ms":   reportP95,
			"ws_reconnect_success_rate":  reconnectRate,
			"targets": map[string]interface{}{
				"question_generation_p95_lt_ms": 2500,
				"report_generation_p95_lt_ms":   20000,
				"ws_reconnect_success_gt":       0.99,
			},
			"compliant": map[string]bool{
				"question_generation": llmP95 > 0 && llmP95 < 2500,
				"report_generation":   reportP95 > 0 && reportP95 < 20000,
				"ws_reconnect":        reconnectRate >= 0.99,
			},
		},
	})

}

func (h *Handler) ImportGitHubProfile(w http.ResponseWriter, r *http.Request) {
	userID := h.userIDFromContext(r.Context())
	if userID == "anonymous" {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var req githubImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	username, normalizedProfileURL, err := parseGitHubUsername(req.ProfileURL)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	maxRepos := req.MaxRepos
	if maxRepos <= 0 {
		maxRepos = 12
	}
	if maxRepos > 30 {
		maxRepos = 30
	}

	ctx, cancel := context.WithTimeout(r.Context(), 18*time.Second)
	defer cancel()

	result, err := h.fetchGitHubProfileAnalytics(ctx, username, normalizedProfileURL, maxRepos)
	if err != nil {
		var ghErr *githubAPIError
		if errors.As(err, &ghErr) {
			if ghErr.StatusCode == http.StatusNotFound {
				writeError(w, http.StatusNotFound, "GitHub-профиль не найден")
				return
			}
			writeError(w, http.StatusBadGateway, "Ошибка GitHub API")
			return
		}

		h.logger.WithError(err).WithFields(logrus.Fields{
			"username": username,
			"user_id":  userID,
		}).Warn("github profile import failed")
		writeError(w, http.StatusBadGateway, "failed to import github profile")
		return
	}

	aiInsights, err := h.requestDeveloperInsights(ctx, result, req.RolePreferences)
	if err != nil {
		h.logger.WithError(err).WithField("username", username).Warn("developer insights request failed, fallback used")
		aiInsights = h.buildLocalDeveloperInsights(result, req.RolePreferences)
	}
	result.AIInsights = aiInsights

	writeSuccess(w, http.StatusOK, result)
}

func parseGitHubUsername(raw string) (string, string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", "", fmt.Errorf("profile_url is required")
	}

	value = strings.TrimPrefix(value, "@")

	if !strings.Contains(value, "/") && !strings.Contains(value, ".") {
		if !isValidGitHubUsername(value) {
			return "", "", fmt.Errorf("invalid github username")
		}
		return strings.ToLower(value), fmt.Sprintf("https://github.com/%s", strings.ToLower(value)), nil
	}

	if !strings.Contains(strings.ToLower(value), "github.com") {
		return "", "", fmt.Errorf("profile_url must point to github.com")
	}

	if !strings.HasPrefix(strings.ToLower(value), "http://") && !strings.HasPrefix(strings.ToLower(value), "https://") {
		value = "https://" + value
	}

	parsed, err := url.Parse(value)
	if err != nil {
		return "", "", fmt.Errorf("invalid github profile url")
	}

	host := strings.ToLower(parsed.Hostname())
	if host != "github.com" && host != "www.github.com" {
		return "", "", fmt.Errorf("profile_url must point to github.com")
	}

	segments := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(segments) == 0 || strings.TrimSpace(segments[0]) == "" {
		return "", "", fmt.Errorf("github username not found in url")
	}

	username := strings.TrimSpace(segments[0])
	if !isValidGitHubUsername(username) {
		return "", "", fmt.Errorf("invalid github username")
	}

	username = strings.ToLower(username)
	return username, fmt.Sprintf("https://github.com/%s", username), nil
}

func isValidGitHubUsername(username string) bool {
	if len(username) == 0 || len(username) > 39 {
		return false
	}
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,37}[a-zA-Z0-9])?$`, username)
	return matched
}

func (h *Handler) fetchGitHubProfileAnalytics(ctx context.Context, username string, profileURL string, maxRepos int) (*githubImportResponse, error) {
	var user githubUserAPIResponse
	if err := h.fetchGitHubAPIJSON(ctx, fmt.Sprintf("https://api.github.com/users/%s", username), &user); err != nil {
		return nil, err
	}

	reposURL := fmt.Sprintf("https://api.github.com/users/%s/repos?sort=updated&per_page=%d&type=owner", username, maxRepos)
	var repos []githubRepoAPIResponse
	if err := h.fetchGitHubAPIJSON(ctx, reposURL, &repos); err != nil {
		return nil, err
	}

	languageCounts := map[string]int{}
	monthlyActivity := map[string]int{}
	contributionCount := map[string]int{}
	today := time.Now().UTC()
	dayWindow := 180
	for i := 0; i < dayWindow; i++ {
		day := today.AddDate(0, 0, -i).Format("2006-01-02")
		contributionCount[day] = 0
	}

	for i := 0; i < 12; i++ {
		month := today.AddDate(0, -i, 0).Format("2006-01")
		monthlyActivity[month] = 0
	}

	totalStars := 0
	totalForks := 0
	totalOpenIssues := 0
	topRepositories := make([]githubTopRepository, 0, len(repos))

	for _, repo := range repos {
		if repo.Archived {
			continue
		}

		totalStars += repo.StargazersCount
		totalForks += repo.ForksCount
		totalOpenIssues += repo.OpenIssuesCount

		if lang := strings.TrimSpace(repo.Language); lang != "" {
			languageCounts[lang]++
		}

		if pushedAt, err := time.Parse(time.RFC3339, repo.PushedAt); err == nil {
			monthKey := pushedAt.UTC().Format("2006-01")
			if _, ok := monthlyActivity[monthKey]; ok {
				monthlyActivity[monthKey]++
			}

			dayKey := pushedAt.UTC().Format("2006-01-02")
			if _, ok := contributionCount[dayKey]; ok {
				contributionCount[dayKey]++
			}
		}

		topRepositories = append(topRepositories, githubTopRepository{
			Name:        repo.Name,
			URL:         repo.HTMLURL,
			Description: strings.TrimSpace(repo.Description),
			Language:    strings.TrimSpace(repo.Language),
			Stars:       repo.StargazersCount,
			Forks:       repo.ForksCount,
			OpenIssues:  repo.OpenIssuesCount,
			LastPush:    repo.PushedAt,
		})
	}

	sort.Slice(topRepositories, func(i, j int) bool {
		if topRepositories[i].Stars != topRepositories[j].Stars {
			return topRepositories[i].Stars > topRepositories[j].Stars
		}
		return topRepositories[i].LastPush > topRepositories[j].LastPush
	})
	if len(topRepositories) > 8 {
		topRepositories = topRepositories[:8]
	}

	languageDistribution := make([]githubChartPoint, 0, len(languageCounts))
	for label, value := range languageCounts {
		languageDistribution = append(languageDistribution, githubChartPoint{Label: label, Value: value})
	}
	sort.Slice(languageDistribution, func(i, j int) bool {
		return languageDistribution[i].Value > languageDistribution[j].Value
	})
	if len(languageDistribution) > 8 {
		languageDistribution = languageDistribution[:8]
	}

	monthlyChart := make([]githubChartPoint, 0, len(monthlyActivity))
	monthKeys := make([]string, 0, len(monthlyActivity))
	for key := range monthlyActivity {
		monthKeys = append(monthKeys, key)
	}
	sort.Strings(monthKeys)
	for _, key := range monthKeys {
		monthlyChart = append(monthlyChart, githubChartPoint{Label: key, Value: monthlyActivity[key]})
	}

	contributionDays := make([]githubContributionDay, 0, len(contributionCount))
	dayKeys := make([]string, 0, len(contributionCount))
	for key := range contributionCount {
		dayKeys = append(dayKeys, key)
	}
	sort.Strings(dayKeys)
	for _, key := range dayKeys {
		contributionDays = append(contributionDays, githubContributionDay{Date: key, Count: contributionCount[key]})
	}

	result := &githubImportResponse{
		Username:    username,
		ProfileURL:  profileURL,
		ProfileName: strings.TrimSpace(user.Name),
		Bio:         strings.TrimSpace(user.Bio),
		AvatarURL:   strings.TrimSpace(user.AvatarURL),
		Stats: githubImportStats{
			Followers:       user.Followers,
			Following:       user.Following,
			PublicRepos:     user.PublicRepos,
			SampledRepos:    len(repos),
			TotalStars:      totalStars,
			TotalForks:      totalForks,
			TotalOpenIssues: totalOpenIssues,
		},
		Charts: githubImportCharts{
			LanguageDistribution: languageDistribution,
			MonthlyActivity:      monthlyChart,
			ContributionDays:     contributionDays,
		},
		TopRepositories: topRepositories,
		AIInsights:      githubAIInsights{},
	}

	if result.ProfileName == "" {
		result.ProfileName = username
	}

	return result, nil
}

func (h *Handler) fetchGitHubAPIJSON(ctx context.Context, endpoint string, target interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "real-ass-interview-service")
	if token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		var payload struct {
			Message string `json:"message"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&payload)
		return &githubAPIError{
			StatusCode: resp.StatusCode,
			Message:    strings.TrimSpace(payload.Message),
		}
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return err
	}

	return nil
}

func (h *Handler) requestDeveloperInsights(ctx context.Context, profile *githubImportResponse, rolePreferences []string) (githubAIInsights, error) {
	payload := developerInsightsRequest{
		GitHubUsername:       profile.Username,
		ProfileName:          profile.ProfileName,
		Bio:                  profile.Bio,
		RolePreferences:      rolePreferences,
		Followers:            profile.Stats.Followers,
		Following:            profile.Stats.Following,
		PublicRepos:          profile.Stats.PublicRepos,
		SampledRepos:         profile.Stats.SampledRepos,
		TotalStars:           profile.Stats.TotalStars,
		TotalForks:           profile.Stats.TotalForks,
		TotalOpenIssues:      profile.Stats.TotalOpenIssues,
		LanguageDistribution: profile.Charts.LanguageDistribution,
		MonthlyActivity:      profile.Charts.MonthlyActivity,
		TopRepositories:      profile.TopRepositories,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return githubAIInsights{}, err
	}

	aiCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	endpoint := h.aiServiceURL + "/api/v1/developer/insights"
	req, err := http.NewRequestWithContext(aiCtx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return githubAIInsights{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return githubAIInsights{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return githubAIInsights{}, fmt.Errorf("ai insights status: %d", resp.StatusCode)
	}

	var out githubAIInsights
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return githubAIInsights{}, err
	}

	if strings.TrimSpace(out.Summary) == "" {
		return githubAIInsights{}, fmt.Errorf("empty ai summary")
	}

	if len(out.RecommendedPositions) == 0 {
		return githubAIInsights{}, fmt.Errorf("no ai positions")
	}

	if len(out.LanguageInsights) == 0 && len(profile.Charts.LanguageDistribution) > 0 {
		top := profile.Charts.LanguageDistribution[0]
		out.LanguageInsights = []githubLanguageInsight{
			{
				Language:        top.Label,
				Confidence:      62,
				Evidence:        "Язык чаще всего встречается в репозиториях и недавней активности.",
				InterviewTopics: []string{"архитектура", "тестирование", "производительность"},
			},
		}
	}

	if len(out.InterviewTracks) == 0 {
		primary := out.RecommendedPositions[0]
		track := githubInterviewTrack{
			Role:            primary.Role,
			Mode:            "practice",
			Level:           "Middle",
			DurationMinutes: 35,
			FocusAreas:      []string{"практическая реализация", "разбор trade-offs", "production reliability"},
			PrimarySkills:   []string{},
			Rationale:       "Сформировано из наиболее сильной AI-рекомендации и технических сигналов профиля.",
		}
		for _, lang := range profile.Charts.LanguageDistribution {
			if strings.TrimSpace(lang.Label) == "" {
				continue
			}
			track.PrimarySkills = append(track.PrimarySkills, lang.Label)
			if len(track.PrimarySkills) >= 4 {
				break
			}
		}
		if len(track.PrimarySkills) == 0 {
			track.PrimarySkills = []string{"алгоритмы", "системный дизайн"}
		}
		out.InterviewTracks = []githubInterviewTrack{track}
	}

	if len(out.ActionPlan) == 0 {
		out.ActionPlan = []string{
			"Выберите самый сильный interview track и начните с режима practice.",
			"Подготовьте 2-3 кейса из ваших репозиториев с фокусом на архитектурные компромиссы.",
		}
	}

	return out, nil
}

func (h *Handler) buildLocalDeveloperInsights(profile *githubImportResponse, rolePreferences []string) githubAIInsights {
	topLanguage := ""
	if len(profile.Charts.LanguageDistribution) > 0 {
		topLanguage = strings.ToLower(profile.Charts.LanguageDistribution[0].Label)
	}

	positions := []githubRoleRecommendation{
		{
			Role:      "Backend Engineer",
			FitScore:  68,
			Rationale: "Профиль демонстрирует базовую продуктовую и инженерную активность.",
		},
		{
			Role:      "Fullstack Engineer",
			FitScore:  65,
			Rationale: "Есть сигналы универсальности по стеку и рабочим задачам.",
		},
	}

	if strings.Contains(topLanguage, "typescript") || strings.Contains(topLanguage, "javascript") {
		positions[1] = githubRoleRecommendation{
			Role:      "Frontend Engineer",
			FitScore:  72,
			Rationale: "Основной язык в профиле ближе к frontend-разработке.",
		}
	}
	if strings.Contains(topLanguage, "python") || strings.Contains(topLanguage, "go") || strings.Contains(topLanguage, "java") {
		positions[0] = githubRoleRecommendation{
			Role:      "Backend Engineer",
			FitScore:  74,
			Rationale: "Ключевые технологии и репозитории лучше соответствуют backend-роли.",
		}
	}

	if len(rolePreferences) > 0 {
		for idx, pref := range rolePreferences {
			if idx >= 2 {
				break
			}
			pref = strings.TrimSpace(pref)
			if pref == "" {
				continue
			}
			positions = append([]githubRoleRecommendation{{
				Role:      pref,
				FitScore:  70,
				Rationale: "Роль совпадает с указанными предпочтениями и поддерживается текущим профилем активности.",
			}}, positions...)
		}
	}

	if len(positions) > 4 {
		positions = positions[:4]
	}

	languageInsights := make([]githubLanguageInsight, 0, 4)
	for _, item := range profile.Charts.LanguageDistribution {
		if strings.TrimSpace(item.Label) == "" {
			continue
		}
		confidence := 55 + item.Value*7
		if confidence > 92 {
			confidence = 92
		}
		languageInsights = append(languageInsights, githubLanguageInsight{
			Language:   item.Label,
			Confidence: confidence,
			Evidence:   "Язык стабильно встречается в репозиториях и последних изменениях.",
			InterviewTopics: []string{
				"архитектурные решения",
				"тестирование и качество",
				"оптимизация производительности",
			},
		})
		if len(languageInsights) >= 4 {
			break
		}
	}

	primarySkills := make([]string, 0, 5)
	for _, insight := range languageInsights {
		primarySkills = append(primarySkills, insight.Language)
	}
	if len(primarySkills) == 0 {
		primarySkills = []string{"алгоритмы", "системный дизайн"}
	}

	interviewTracks := []githubInterviewTrack{
		{
			Role:            positions[0].Role,
			Mode:            "practice",
			Level:           "Middle",
			DurationMinutes: 35,
			FocusAreas: []string{
				"практическая реализация",
				"надежность в продакшене",
				"аргументация trade-offs",
			},
			PrimarySkills: primarySkills,
			Rationale:     "Трек построен вокруг самой сильной роли и доминирующих языков программирования.",
		},
	}

	return githubAIInsights{
		Summary: "Профиль показывает достаточную публичную инженерную активность для таргетированных технических интервью.",
		Strengths: []string{
			"Есть измеримые сигналы по репозиториям, языкам и активности.",
			"Публичные проекты позволяют построить практический сценарий интервью.",
		},
		Risks: []string{
			"Оценка ограничена публичными данными и не учитывает приватные рабочие проекты.",
		},
		ActionPlan: []string{
			"Начните интервью с практического трека по наиболее сильному языку.",
			"Подготовьте краткие кейсы из репозиториев с акцентом на архитектуру и метрики.",
			"Во втором раунде добавьте теоретический блок на reliability и масштабирование.",
		},
		LanguageInsights:     languageInsights,
		InterviewTracks:      interviewTracks,
		RecommendedPositions: positions,
	}
}

func (h *Handler) ImportResumeProfile(w http.ResponseWriter, r *http.Request) {
	userID := h.userIDFromContext(r.Context())
	if userID == "anonymous" {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	uploadStart := time.Now()

	if err := r.ParseMultipartForm(12 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	if header.Size <= 0 {
		writeError(w, http.StatusBadRequest, "empty file")
		return
	}
	if header.Size > 10*1024*1024 {
		writeError(w, http.StatusBadRequest, "file is too large (max 10MB)")
		return
	}

	fileData, err := io.ReadAll(io.LimitReader(file, 10*1024*1024+1))
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read file")
		return
	}
	if len(fileData) == 0 {
		writeError(w, http.StatusBadRequest, "empty file")
		return
	}
	if len(fileData) > 10*1024*1024 {
		writeError(w, http.StatusBadRequest, "file is too large (max 10MB)")
		return
	}

	uploadDuration := time.Since(uploadStart).Milliseconds()

	rolePreferences := parseRolePreferencesField(r.FormValue("role_preferences"))

	extractStart := time.Now()
	extracted, err := extractResumeData(header.Filename, header.Header.Get("Content-Type"), fileData)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	extractDuration := time.Since(extractStart).Milliseconds()

	wordCount := countWords(extracted.Text)
	charCount := len([]rune(extracted.Text))
	estPages := 1
	if wordCount > 0 {
		estPages = int(math.Ceil(float64(wordCount) / 450.0))
	}

	response := resumeImportResponse{
		ReportID:       uuid.NewString(),
		CreatedAt:      time.Now().UTC(),
		FileName:       header.Filename,
		FileSize:       int64(len(fileData)),
		ContentType:    extracted.NormalizedType,
		DetectedFormat: extracted.DetectedFormat,
		Stats: resumeImportStats{
			WordCount:         wordCount,
			CharacterCount:    charCount,
			EstimatedPages:    estPages,
			SkillsCount:       len(extracted.Skills),
			LanguageCount:     len(extracted.Languages),
			ExperienceEntries: extracted.ExperienceEntries,
			EducationEntries:  extracted.EducationEntries,
		},
		Charts: resumeImportCharts{
			LanguageDistribution: extracted.LangDistribution,
			SkillsDistribution:   extracted.SkillDistribution,
		},
		ExtractedSkills: extracted.Skills,
		ProcessingStages: []resumeProcessingStage{
			{
				Code:       "upload",
				Title:      "Загрузка файла",
				Status:     "done",
				DurationMs: uploadDuration,
			},
			{
				Code:       "extract",
				Title:      "Извлечение текста",
				Status:     "done",
				DurationMs: extractDuration,
			},
			{
				Code:       "ai_analysis",
				Title:      "AI-анализ",
				Status:     "processing",
				DurationMs: 0,
			},
		},
		AIInsights: resumeAIInsights{},
	}

	ctx, cancel := context.WithTimeout(r.Context(), 18*time.Second)
	defer cancel()

	aiStart := time.Now()
	aiInsights, err := h.requestResumeInsights(ctx, &response, extracted.Text, extracted.Languages, rolePreferences)
	if err != nil {
		h.logger.WithError(err).WithFields(logrus.Fields{
			"file_name": header.Filename,
			"user_id":   userID,
		}).Warn("resume insights request failed, fallback used")
		aiInsights = h.buildLocalResumeInsights(&response, extracted.Languages, rolePreferences)
	}
	aiDuration := time.Since(aiStart).Milliseconds()

	response.AIInsights = aiInsights
	response.ProcessingStages[2].Status = "done"
	response.ProcessingStages[2].DurationMs = aiDuration

	h.saveResumeReport(userID, response)
	writeSuccess(w, http.StatusOK, response)
}

func (h *Handler) GetResumeImportHistory(w http.ResponseWriter, r *http.Request) {
	userID := h.userIDFromContext(r.Context())
	if userID == "anonymous" {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	h.resumeMu.RLock()
	history := append([]resumeImportResponse(nil), h.resumeHistory[userID]...)
	h.resumeMu.RUnlock()

	writeSuccess(w, http.StatusOK, map[string]interface{}{
		"items": history,
		"total": len(history),
	})
}

func (h *Handler) GetResumeImportReport(w http.ResponseWriter, r *http.Request) {
	userID := h.userIDFromContext(r.Context())
	if userID == "anonymous" {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	reportID := strings.TrimSpace(mux.Vars(r)["report_id"])
	if reportID == "" {
		writeError(w, http.StatusBadRequest, "report_id is required")
		return
	}

	h.resumeMu.RLock()
	defer h.resumeMu.RUnlock()
	for _, item := range h.resumeHistory[userID] {
		if item.ReportID == reportID {
			writeSuccess(w, http.StatusOK, item)
			return
		}
	}

	writeError(w, http.StatusNotFound, "resume report not found")
}

func (h *Handler) saveResumeReport(userID string, report resumeImportResponse) {
	h.resumeMu.Lock()
	defer h.resumeMu.Unlock()

	history := append([]resumeImportResponse{report}, h.resumeHistory[userID]...)
	if len(history) > 25 {
		history = history[:25]
	}
	h.resumeHistory[userID] = history
}

func parseRolePreferencesField(raw string) []string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil
	}

	if strings.HasPrefix(value, "[") {
		var parsed []string
		if err := json.Unmarshal([]byte(value), &parsed); err == nil {
			cleaned := make([]string, 0, len(parsed))
			for _, item := range parsed {
				item = strings.TrimSpace(item)
				if item != "" {
					cleaned = append(cleaned, item)
				}
			}
			return cleaned
		}
	}

	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

func extractResumeData(fileName string, contentType string, data []byte) (*resumeExtractionResult, error) {
	format, normalizedType, err := detectResumeFormat(fileName, contentType)
	if err != nil {
		return nil, err
	}

	var text string
	switch format {
	case "txt", "rtf":
		text = string(data)
	case "docx":
		parsed, parseErr := extractDOCXText(data)
		if parseErr != nil {
			return nil, fmt.Errorf("failed to parse DOCX file")
		}
		text = parsed
	case "pdf":
		parsed, parseErr := extractPDFText(data)
		if parseErr != nil {
			return nil, fmt.Errorf("failed to parse PDF file")
		}
		text = parsed
	default:
		return nil, fmt.Errorf("unsupported resume format")
	}

	text = normalizeText(text)
	if countWords(text) < 20 {
		return nil, fmt.Errorf("не удалось извлечь текст из файла. Попробуйте PDF/DOCX/TXT с текстовым слоем")
	}

	skillDist, langDist, skills, langs := extractSkillsAndLanguages(text)

	result := &resumeExtractionResult{
		Text:              text,
		DetectedFormat:    format,
		NormalizedType:    normalizedType,
		Skills:            skills,
		Languages:         langs,
		SkillDistribution: skillDist,
		LangDistribution:  langDist,
		ExperienceEntries: estimateExperienceEntries(text),
		EducationEntries:  estimateEducationEntries(text),
	}

	return result, nil
}

func detectResumeFormat(fileName, contentType string) (string, string, error) {
	ext := strings.ToLower(strings.TrimSpace(filepath.Ext(fileName)))
	ct := strings.ToLower(strings.TrimSpace(contentType))

	allowed := map[string]struct{}{
		".pdf":  {},
		".docx": {},
		".txt":  {},
		".rtf":  {},
	}
	if _, ok := allowed[ext]; !ok {
		return "", "", fmt.Errorf("неподдерживаемый формат файла: %s. Разрешены PDF, DOCX, TXT, RTF", ext)
	}

	switch {
	case ext == ".pdf" || strings.Contains(ct, "application/pdf"):
		return "pdf", "application/pdf", nil
	case ext == ".docx" || strings.Contains(ct, "application/vnd.openxmlformats-officedocument.wordprocessingml.document"):
		return "docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", nil
	case ext == ".txt" || strings.Contains(ct, "text/plain"):
		return "txt", "text/plain", nil
	case ext == ".rtf" || strings.Contains(ct, "application/rtf") || strings.Contains(ct, "text/rtf"):
		return "rtf", "application/rtf", nil
	default:
		return "", "", fmt.Errorf("некорректный content-type для выбранного формата. Разрешены PDF, DOCX, TXT, RTF")
	}
}

func extractDOCXText(data []byte) (string, error) {
	readerAt := bytes.NewReader(data)
	zr, err := zip.NewReader(readerAt, int64(len(data)))
	if err != nil {
		return "", err
	}

	parts := make([]string, 0, 4)
	for _, f := range zr.File {
		if !strings.HasPrefix(f.Name, "word/") || !strings.HasSuffix(f.Name, ".xml") {
			continue
		}
		rc, openErr := f.Open()
		if openErr != nil {
			continue
		}
		chunk, readErr := io.ReadAll(rc)
		_ = rc.Close()
		if readErr != nil {
			continue
		}
		xmlText := string(chunk)
		xmlText = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(xmlText, " ")
		xmlText = strings.NewReplacer("&amp;", "&", "&lt;", "<", "&gt;", ">", "&quot;", `"`).Replace(xmlText)
		parts = append(parts, xmlText)
	}

	if len(parts) == 0 {
		return "", fmt.Errorf("docx text not found")
	}

	return strings.Join(parts, "\n"), nil
}

func extractPDFText(data []byte) (string, error) {
	tmpFile, err := os.CreateTemp("", "resume-*.pdf")
	if err != nil {
		return "", err
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return "", err
	}
	if err := tmpFile.Close(); err != nil {
		return "", err
	}

	f, reader, err := pdf.Open(tmpPath)
	if err == nil {
		defer f.Close()
		var b strings.Builder
		for i := 1; i <= reader.NumPage(); i++ {
			page := reader.Page(i)
			if page.V.IsNull() {
				continue
			}
			content, pageErr := page.GetPlainText(nil)
			if pageErr != nil {
				continue
			}
			b.WriteString(content)
			b.WriteString("\n")
		}
		parsed := normalizeText(b.String())
		if countWords(parsed) >= 20 {
			return parsed, nil
		}
	}

	raw := string(data)
	var b strings.Builder
	b.Grow(len(raw))
	for _, ch := range raw {
		if ch == '\n' || ch == '\r' || ch == '\t' {
			b.WriteRune(' ')
			continue
		}
		if (ch >= 32 && ch <= 126) || (ch >= 0x0400 && ch <= 0x04FF) {
			b.WriteRune(ch)
		} else {
			b.WriteRune(' ')
		}
	}

	text := normalizeText(b.String())
	if countWords(text) >= 20 {
		return text, nil
	}

	matches := regexp.MustCompile(`\(([^\)]+)\)`).FindAllStringSubmatch(raw, 400)
	fragments := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		value := normalizeText(match[1])
		if len(value) >= 4 {
			fragments = append(fragments, value)
		}
	}
	return normalizeText(strings.Join(fragments, " ")), nil
}

func normalizeText(value string) string {
	value = strings.ReplaceAll(value, "\u0000", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\t", " ")
	value = regexp.MustCompile(`\s+`).ReplaceAllString(value, " ")
	return strings.TrimSpace(value)
}

func countWords(text string) int {
	if strings.TrimSpace(text) == "" {
		return 0
	}
	return len(strings.Fields(text))
}

func extractSkillsAndLanguages(text string) ([]githubChartPoint, []githubChartPoint, []string, []string) {
	lowered := " " + strings.ToLower(text) + " "
	skillCounts := map[string]int{}
	languageCounts := map[string]int{}

	for skill, markers := range resumeSkillKeywords {
		count := 0
		for _, marker := range markers {
			count += strings.Count(lowered, strings.ToLower(marker))
		}
		if count > 0 {
			skillCounts[skill] = count
		}
	}

	for _, language := range programmingLanguagesOrder {
		marker := strings.ToLower(language)
		count := strings.Count(lowered, " "+marker+" ")
		if language == "C#" {
			count += strings.Count(lowered, " csharp ")
		}
		if language == "C++" {
			count += strings.Count(lowered, " cpp ")
		}
		if count > 0 {
			languageCounts[language] = count
		}
	}

	if value, ok := skillCounts["TypeScript"]; ok {
		languageCounts["TypeScript"] += value
	}
	if value, ok := skillCounts["JavaScript"]; ok {
		languageCounts["JavaScript"] += value
	}
	if value, ok := skillCounts["Python"]; ok {
		languageCounts["Python"] += value
	}
	if value, ok := skillCounts["Go"]; ok {
		languageCounts["Go"] += value
	}

	skillDist := make([]githubChartPoint, 0, len(skillCounts))
	for label, count := range skillCounts {
		skillDist = append(skillDist, githubChartPoint{Label: label, Value: count})
	}
	sort.Slice(skillDist, func(i, j int) bool {
		if skillDist[i].Value == skillDist[j].Value {
			return skillDist[i].Label < skillDist[j].Label
		}
		return skillDist[i].Value > skillDist[j].Value
	})
	if len(skillDist) > 12 {
		skillDist = skillDist[:12]
	}

	langDist := make([]githubChartPoint, 0, len(languageCounts))
	for label, count := range languageCounts {
		langDist = append(langDist, githubChartPoint{Label: label, Value: count})
	}
	sort.Slice(langDist, func(i, j int) bool {
		if langDist[i].Value == langDist[j].Value {
			return langDist[i].Label < langDist[j].Label
		}
		return langDist[i].Value > langDist[j].Value
	})
	if len(langDist) > 8 {
		langDist = langDist[:8]
	}

	skills := make([]string, 0, len(skillDist))
	for _, item := range skillDist {
		skills = append(skills, item.Label)
	}

	languages := make([]string, 0, len(langDist))
	for _, item := range langDist {
		languages = append(languages, item.Label)
	}

	return skillDist, langDist, skills, languages
}

func estimateExperienceEntries(text string) int {
	lowered := strings.ToLower(text)
	yearHits := regexp.MustCompile(`(19|20)\d{2}`).FindAllString(lowered, -1)
	entries := len(yearHits) / 2
	if entries == 0 {
		entries = strings.Count(lowered, "experience") + strings.Count(lowered, "опыт") + strings.Count(lowered, "company")
	}
	if entries <= 0 {
		entries = 1
	}
	if entries > 12 {
		entries = 12
	}
	return entries
}

func estimateEducationEntries(text string) int {
	lowered := strings.ToLower(text)
	entries := strings.Count(lowered, "university") + strings.Count(lowered, "bachelor") + strings.Count(lowered, "master") +
		strings.Count(lowered, "образован") + strings.Count(lowered, "институт") + strings.Count(lowered, "университет")
	if entries <= 0 {
		entries = 1
	}
	if entries > 6 {
		entries = 6
	}
	return entries
}

func (h *Handler) requestResumeInsights(
	ctx context.Context,
	report *resumeImportResponse,
	resumeText string,
	languages []string,
	rolePreferences []string,
) (resumeAIInsights, error) {
	payload := resumeInsightsRequest{
		FileName:          report.FileName,
		ContentType:       report.ContentType,
		RolePreferences:   rolePreferences,
		WordCount:         report.Stats.WordCount,
		CharacterCount:    report.Stats.CharacterCount,
		Skills:            report.ExtractedSkills,
		Languages:         languages,
		ExperienceEntries: report.Stats.ExperienceEntries,
		EducationEntries:  report.Stats.EducationEntries,
		TextExcerpt:       truncateRunes(resumeText, 6000),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return resumeAIInsights{}, err
	}

	aiCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	endpoint := h.aiServiceURL + "/api/v1/resume/insights"
	req, err := http.NewRequestWithContext(aiCtx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return resumeAIInsights{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return resumeAIInsights{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return resumeAIInsights{}, fmt.Errorf("resume insights status: %d", resp.StatusCode)
	}

	var out resumeAIInsightsPayload
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return resumeAIInsights{}, err
	}

	if strings.TrimSpace(out.Summary) == "" {
		return resumeAIInsights{}, fmt.Errorf("empty resume ai summary")
	}
	if len(out.RecommendedPositions) == 0 {
		return resumeAIInsights{}, fmt.Errorf("no resume ai positions")
	}

	return resumeAIInsights(out), nil
}

func (h *Handler) buildLocalResumeInsights(
	report *resumeImportResponse,
	languages []string,
	rolePreferences []string,
) resumeAIInsights {
	positions := []githubRoleRecommendation{
		{
			Role:      "Backend Engineer",
			FitScore:  68,
			Rationale: "Профиль резюме подходит для backend-интервью с фокусом на инженерные задачи.",
		},
		{
			Role:      "Fullstack Engineer",
			FitScore:  64,
			Rationale: "Набор навыков выглядит универсальным и применимым для смешанных ролей.",
		},
	}

	if len(rolePreferences) > 0 {
		pref := strings.TrimSpace(rolePreferences[0])
		if pref != "" {
			positions = append([]githubRoleRecommendation{{
				Role:      pref,
				FitScore:  72,
				Rationale: "Роль совпадает с предпочтением пользователя и поддерживается содержимым резюме.",
			}}, positions...)
		}
	}

	languageInsights := make([]githubLanguageInsight, 0, 4)
	for idx, lang := range languages {
		if strings.TrimSpace(lang) == "" {
			continue
		}
		confidence := 78 - idx*7
		if confidence < 55 {
			confidence = 55
		}
		languageInsights = append(languageInsights, githubLanguageInsight{
			Language:   lang,
			Confidence: confidence,
			Evidence:   "Язык явно отражен в резюме и может быть основой интервью-сценария.",
			InterviewTopics: []string{
				"архитектурные решения",
				"качество и тестирование",
				"оптимизация и отладка",
			},
		})
		if len(languageInsights) >= 4 {
			break
		}
	}
	if len(languageInsights) == 0 {
		languageInsights = append(languageInsights, githubLanguageInsight{
			Language:        "General",
			Confidence:      58,
			Evidence:        "Языки в резюме указаны неполно, поэтому фокус задан как общий инженерный.",
			InterviewTopics: []string{"алгоритмы", "структуры данных", "системное мышление"},
		})
	}

	primarySkills := make([]string, 0, 5)
	for _, item := range languageInsights {
		primarySkills = append(primarySkills, item.Language)
	}
	if len(primarySkills) == 0 {
		primarySkills = report.ExtractedSkills
	}
	if len(primarySkills) == 0 {
		primarySkills = []string{"алгоритмы", "системный дизайн"}
	}

	tracks := []githubInterviewTrack{
		{
			Role:            positions[0].Role,
			Mode:            "practice",
			Level:           "Middle",
			DurationMinutes: 35,
			FocusAreas:      []string{"live coding", "разбор trade-offs", "production reliability"},
			PrimarySkills:   primarySkills,
			Rationale:       "Трек основан на найденных навыках резюме и наиболее релевантной позиции.",
		},
	}

	return resumeAIInsights{
		Summary: "Резюме успешно импортировано: профиль подходит для технического интервью, но стоит усилить раздел достижений конкретными метриками.",
		StrongPoints: []string{
			"В резюме присутствуют технические навыки, пригодные для целевого интервью.",
			"Структура документа позволяет выделить практические зоны для оценки.",
		},
		ImprovementPoints: []string{
			"Добавьте количественные результаты по ключевым проектам.",
			"Уточните вклад в архитектурные и продуктовые решения.",
		},
		ActionPlan: []string{
			"Обновите резюме в формате: действие -> технология -> измеримый результат.",
			"Подготовьте 2-3 проектных кейса для практического интервью по сильному стеку.",
			"Пройдите первый технический трек и зафиксируйте зоны роста для следующего раунда.",
		},
		LanguageInsights:     languageInsights,
		InterviewTracks:      tracks,
		RecommendedPositions: positions,
	}
}

func truncateRunes(value string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return strings.TrimSpace(string(runes[:max]))
}

func toStableUserUUID(userID string) uuid.UUID {
	trimmed := strings.TrimSpace(userID)
	if trimmed == "" {
		return uuid.Nil
	}
	if parsed, err := uuid.Parse(trimmed); err == nil {
		return parsed
	}
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(trimmed))
}

func levelToDomain(value string) domain.InterviewSessionLevel {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "junior":
		return domain.InterviewSessionLevelJunior
	case "senior":
		return domain.InterviewSessionLevelSenior
	default:
		return domain.InterviewSessionLevelMiddle
	}
}

func parseStringListFromAny(value interface{}) []string {
	items, ok := value.([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		text := strings.TrimSpace(fmt.Sprintf("%v", item))
		if text != "" && text != "<nil>" {
			out = append(out, text)
		}
	}
	return out
}

func intFromMetadata(metadata map[string]interface{}, key string, fallback int) int {
	if metadata == nil {
		return fallback
	}
	raw, ok := metadata[key]
	if !ok {
		return fallback
	}
	switch value := raw.(type) {
	case float64:
		return int(value)
	case float32:
		return int(value)
	case int:
		return value
	case int64:
		return int(value)
	case int32:
		return int(value)
	case string:
		if parsed, err := strconv.Atoi(strings.TrimSpace(value)); err == nil {
			return parsed
		}
	}
	return fallback
}

func topicStatsFromMetadata(metadata map[string]interface{}) map[string]int {
	raw, ok := metadata["topic_stats"].(map[string]interface{})
	if !ok {
		return map[string]int{}
	}
	out := make(map[string]int, len(raw))
	for key, value := range raw {
		switch typed := value.(type) {
		case float64:
			out[key] = int(typed)
		case int:
			out[key] = typed
		case int64:
			out[key] = int(typed)
		}
	}
	return out
}

func (h *Handler) toDomainSession(session *InterviewModuleSession) *domain.InterviewModuleSession {
	if session == nil {
		return nil
	}

	metadata := map[string]interface{}{
		"vacancy_title":    session.VacancyTitle,
		"vacancy_category": session.VacancyCategory,
		"interview_mode":   session.InterviewMode,
		"focus_areas":      append([]string(nil), session.FocusAreas...),
		"primary_skills":   append([]string(nil), session.PrimarySkills...),
		"theory_focus":     append([]string(nil), session.TheoryFocus...),
		"practice_focus":   append([]string(nil), session.PracticeFocus...),
		"created_at":       session.CreatedAt,
		"expires_at":       session.ExpiresAt,
		"topic_cursor":     session.TopicCursor,
		"weak_streak":      session.WeakAnswerStreak,
		"topic_stats":      session.TopicStats,
		"api_user_id":      session.UserID,
	}

	return &domain.InterviewModuleSession{
		ID:              session.SessionID,
		UserID:          toStableUserUUID(session.UserID),
		Role:            session.Role,
		Level:           levelToDomain(session.Level),
		Status:          domain.InterviewSessionStatus(strings.ToLower(strings.TrimSpace(session.Status))),
		CurrentTopic:    session.CurrentTopic,
		DifficultyScore: session.Difficulty,
		PressureLevel:   session.PressureLevel,
		QuestionCount:   h.countAIMessages(session.Messages),
		QuestionLimit:   session.QuestionLimit,
		StartedAt:       session.StartedAt,
		EndedAt:         session.FinishedAt,
		DurationSeconds: session.DurationMinutes * 60,
		Metadata:        metadata,
		CreatedAt:       session.CreatedAt,
		UpdatedAt:       time.Now(),
	}
}

func (h *Handler) toAPISession(domainSession *domain.InterviewModuleSession, messages []InterviewChatMessage) *InterviewModuleSession {
	if domainSession == nil {
		return nil
	}

	meta := domainSession.Metadata
	apiUserID, _ := meta["api_user_id"].(string)
	if strings.TrimSpace(apiUserID) == "" {
		apiUserID = domainSession.UserID.String()
	}

	createdAt := domainSession.CreatedAt
	if raw, ok := meta["created_at"].(string); ok {
		if parsed, err := time.Parse(time.RFC3339Nano, raw); err == nil {
			createdAt = parsed
		}
	}
	if raw, ok := meta["created_at"].(time.Time); ok {
		createdAt = raw
	}

	expiresAt := domainSession.StartedAt.Add(time.Duration(domainSession.DurationSeconds) * time.Second)
	if raw, ok := meta["expires_at"].(string); ok {
		if parsed, err := time.Parse(time.RFC3339Nano, raw); err == nil {
			expiresAt = parsed
		}
	}
	if raw, ok := meta["expires_at"].(time.Time); ok {
		expiresAt = raw
	}

	session := &InterviewModuleSession{
		SessionID:        domainSession.ID,
		UserID:           apiUserID,
		Role:             domainSession.Role,
		VacancyTitle:     strings.TrimSpace(fmt.Sprintf("%v", meta["vacancy_title"])),
		VacancyCategory:  strings.TrimSpace(fmt.Sprintf("%v", meta["vacancy_category"])),
		InterviewMode:    normalizeInterviewMode(fmt.Sprintf("%v", meta["interview_mode"])),
		FocusAreas:       parseStringListFromAny(meta["focus_areas"]),
		PrimarySkills:    parseStringListFromAny(meta["primary_skills"]),
		TheoryFocus:      parseStringListFromAny(meta["theory_focus"]),
		PracticeFocus:    parseStringListFromAny(meta["practice_focus"]),
		Level:            strings.Title(string(domainSession.Level)),
		Status:           string(domainSession.Status),
		DurationMinutes:  int(math.Max(1, float64(domainSession.DurationSeconds/60))),
		QuestionLimit:    domainSession.QuestionLimit,
		CurrentTopic:     domainSession.CurrentTopic,
		Difficulty:       domainSession.DifficultyScore,
		PressureLevel:    domainSession.PressureLevel,
		TopicStats:       topicStatsFromMetadata(meta),
		CreatedAt:        createdAt,
		StartedAt:        domainSession.StartedAt,
		ExpiresAt:        expiresAt,
		FinishedAt:       domainSession.EndedAt,
		Messages:         messages,
		WeakAnswerStreak: intFromMetadata(meta, "weak_streak", 0),
		TopicCursor:      intFromMetadata(meta, "topic_cursor", 0),
	}

	if session.TopicStats == nil {
		session.TopicStats = map[string]int{}
	}
	if session.InterviewMode == "" {
		session.InterviewMode = "practice"
	}
	if session.DurationMinutes <= 0 {
		session.DurationMinutes = 30
	}
	if session.QuestionLimit <= 0 {
		session.QuestionLimit = 10
	}

	return session
}

func (h *Handler) toDomainMessage(sessionID uuid.UUID, msg InterviewChatMessage) *domain.InterviewModuleMessage {
	var topic *string
	trimmedTopic := strings.TrimSpace(msg.Topic)
	if trimmedTopic != "" {
		topic = &trimmedTopic
	}

	var difficulty *int
	if msg.Difficulty > 0 {
		difficulty = &msg.Difficulty
	}

	return &domain.InterviewModuleMessage{
		ID:         msg.MessageID,
		SessionID:  sessionID,
		Sender:     domain.MessageSender(strings.ToLower(strings.TrimSpace(msg.Sender))),
		Content:    msg.Content,
		Topic:      topic,
		Difficulty: difficulty,
		CreatedAt:  msg.CreatedAt,
	}
}

func toAPIMessage(message *domain.InterviewModuleMessage) InterviewChatMessage {
	if message == nil {
		return InterviewChatMessage{}
	}
	topic := ""
	if message.Topic != nil {
		topic = *message.Topic
	}
	difficulty := 0
	if message.Difficulty != nil {
		difficulty = *message.Difficulty
	}
	return InterviewChatMessage{
		MessageID:  message.ID,
		Sender:     string(message.Sender),
		Content:    message.Content,
		Topic:      topic,
		Difficulty: difficulty,
		CreatedAt:  message.CreatedAt,
	}
}

func (h *Handler) toDomainReport(report *InterviewModuleReport) *domain.InterviewModuleReport {
	if report == nil {
		return nil
	}
	toItems := func(values []string) []map[string]interface{} {
		items := make([]map[string]interface{}, 0, len(values))
		for _, value := range values {
			trimmed := strings.TrimSpace(value)
			if trimmed == "" {
				continue
			}
			items = append(items, map[string]interface{}{"text": trimmed})
		}
		return items
	}

	return &domain.InterviewModuleReport{
		SessionID:       report.SessionID,
		Correctness:     report.Correctness,
		Clarity:         report.Clarity,
		Completeness:    report.Completeness,
		Relevance:       report.Relevance,
		OverallScore:    report.OverallScore,
		Strengths:       toItems(report.Strengths),
		Weaknesses:      toItems(report.Weaknesses),
		Recommendations: toItems(report.Recommendations),
		GeneratedAt:     report.GeneratedAt,
	}
}

func toAPIReport(report *domain.InterviewModuleReport) *InterviewModuleReport {
	if report == nil {
		return nil
	}
	fromItems := func(values []map[string]interface{}) []string {
		items := make([]string, 0, len(values))
		for _, value := range values {
			if value == nil {
				continue
			}
			if text, ok := value["text"]; ok {
				trimmed := strings.TrimSpace(fmt.Sprintf("%v", text))
				if trimmed != "" && trimmed != "<nil>" {
					items = append(items, trimmed)
					continue
				}
			}
			raw, err := json.Marshal(value)
			if err == nil {
				items = append(items, string(raw))
			}
		}
		return items
	}

	return &InterviewModuleReport{
		SessionID:       report.SessionID,
		Correctness:     report.Correctness,
		Clarity:         report.Clarity,
		Completeness:    report.Completeness,
		Relevance:       report.Relevance,
		OverallScore:    report.OverallScore,
		Strengths:       fromItems(report.Strengths),
		Weaknesses:      fromItems(report.Weaknesses),
		Recommendations: fromItems(report.Recommendations),
		GeneratedAt:     report.GeneratedAt,
	}
}

func (h *Handler) persistSession(ctx context.Context, session *InterviewModuleSession) {
	if h.moduleStore == nil || session == nil {
		return
	}
	if err := h.moduleStore.UpdateInterviewModuleSession(ctx, h.toDomainSession(session)); err != nil {
		h.logger.WithError(err).WithField("session_id", session.SessionID).Warn("failed to persist module session")
	}
}

func (h *Handler) loadSessionFromStore(ctx context.Context, sessionID uuid.UUID) (*InterviewModuleSession, error) {
	if h.moduleStore == nil {
		return nil, errors.New("module store is disabled")
	}

	domainSession, err := h.moduleStore.GetInterviewModuleSessionByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	domainMessages, err := h.moduleStore.ListInterviewModuleMessagesBySessionID(ctx, sessionID, 5000, 0)
	if err != nil {
		return nil, err
	}

	messages := make([]InterviewChatMessage, 0, len(domainMessages))
	for _, message := range domainMessages {
		messages = append(messages, toAPIMessage(message))
	}

	return h.toAPISession(domainSession, messages), nil
}

func (h *Handler) CreateInterviewModuleSession(w http.ResponseWriter, r *http.Request) {
	var req createInterviewModuleSessionRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if strings.TrimSpace(req.Role) == "" || strings.TrimSpace(req.Level) == "" {
		writeError(w, http.StatusBadRequest, "role and level are required")
		return
	}
	if req.DurationMinutes <= 0 {
		req.DurationMinutes = 30
	}
	if req.QuestionLimit <= 0 {
		req.QuestionLimit = 10
	}

	userID := h.userIDFromContext(r.Context())
	if userID == "anonymous" {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey != "" {
		cacheKey := fmt.Sprintf("create:%s:%s", userID, idempotencyKey)
		h.moduleMu.RLock()
		cached, exists := h.requestLog[cacheKey]
		h.moduleMu.RUnlock()
		if exists {
			if parsed, err := uuid.Parse(cached); err == nil {
				writeSuccess(w, http.StatusCreated, map[string]interface{}{
					"session_id": parsed,
					"ws_url":     fmt.Sprintf("/api/interviews/sessions/%s/ws", parsed.String()),
					"idempotent": true,
					"reused":     true,
				})
				return
			}
		}
	}

	sessionID := uuid.New()
	now := time.Now()
	initialDifficulty, initialPressure := h.initialDifficultyAndPressure(req.Level)
	topicSeed := int(sessionID[1] % 5)
	session := h.buildInterviewModuleSession(userID, req, sessionID, now, initialDifficulty, initialPressure, topicSeed)
	if h.moduleStore != nil {
		if err := h.moduleStore.CreateInterviewModuleSession(r.Context(), h.toDomainSession(session)); err != nil {
			h.logger.WithError(err).WithField("session_id", sessionID).Error("failed to persist module session")
			writeError(w, http.StatusInternalServerError, "failed to create session")
			return
		}
	}

	h.moduleMu.Lock()
	h.moduleSessions[sessionID] = session
	if idempotencyKey != "" {
		h.requestLog[fmt.Sprintf("create:%s:%s", userID, idempotencyKey)] = sessionID.String()
	}
	h.moduleMu.Unlock()

	wsURL := fmt.Sprintf("/api/interviews/sessions/%s/ws", sessionID.String())
	writeSuccess(w, http.StatusCreated, map[string]interface{}{
		"session_id": sessionID,
		"ws_url":     wsURL,
		"expires_at": session.ExpiresAt,
	})

	go h.generateInitialQuestion(context.Background(), sessionID)
}

func (h *Handler) generateInitialQuestion(ctx context.Context, sessionID uuid.UUID) {
	h.moduleMu.RLock()
	session, ok := h.moduleSessions[sessionID]
	h.moduleMu.RUnlock()
	if !ok {
		return
	}

	h.broadcastSessionEvent(session.SessionID, "session.started", map[string]interface{}{
		"session_id": session.SessionID,
		"started_at": session.StartedAt,
	})
	h.broadcastSessionEvent(session.SessionID, "ai.typing.started", map[string]bool{"typing": true})

	firstQuestion := h.buildIntroQuestion(session)

	h.moduleMu.Lock()
	session.CurrentTopic = firstQuestion.Topic
	session.PressureLevel = firstQuestion.PressureLevel
	h.applyDifficultyDelta(session, firstQuestion.DifficultyDelta)
	aiMsg := InterviewChatMessage{
		MessageID:  uuid.New(),
		Sender:     "ai",
		Content:    h.sanitizeModelText(firstQuestion.Question),
		Topic:      session.CurrentTopic,
		Difficulty: session.Difficulty,
		CreatedAt:  time.Now(),
	}
	session.Messages = append(session.Messages, aiMsg)
	session.TopicStats[session.CurrentTopic]++
	h.moduleMu.Unlock()

	if h.moduleStore != nil {
		if err := h.moduleStore.CreateInterviewModuleMessage(ctx, h.toDomainMessage(session.SessionID, aiMsg)); err != nil {
			h.logger.WithError(err).WithField("session_id", session.SessionID).Warn("failed to persist initial ai message")
		}
		h.persistSession(ctx, session)
	}

	h.rememberAskedQuestion(ctx, session.UserID, session.Role, aiMsg.Content)

	h.broadcastSessionEvent(session.SessionID, "ai.typing.stopped", map[string]bool{"typing": false})
	h.streamAIMessageContent(session.SessionID, aiMsg.Content)
	h.broadcastSessionEvent(session.SessionID, "message.ai", aiMsg)
	h.recordAuditEvent(ctx, session.SessionID, "llm_response", map[string]interface{}{"topic": aiMsg.Topic, "difficulty": aiMsg.Difficulty, "pressure": session.PressureLevel})
}

func (h *Handler) GetInterviewModuleSession(w http.ResponseWriter, r *http.Request) {
	session, ok := h.getModuleSessionFromRequest(w, r)
	if !ok {
		return
	}
	writeSuccess(w, http.StatusOK, session)
}

func (h *Handler) GetInterviewModuleMessages(w http.ResponseWriter, r *http.Request) {
	session, ok := h.getModuleSessionFromRequest(w, r)
	if !ok {
		return
	}

	limit := len(session.Messages)
	if limit == 0 {
		limit = 50
	}
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	if h.moduleStore != nil {
		domainMessages, err := h.moduleStore.ListInterviewModuleMessagesBySessionID(r.Context(), session.SessionID, limit, offset)
		if err == nil {
			out := make([]InterviewChatMessage, 0, len(domainMessages))
			for _, message := range domainMessages {
				out = append(out, toAPIMessage(message))
			}
			writeSuccess(w, http.StatusOK, map[string]interface{}{
				"messages": out,
				"total":    len(session.Messages),
			})
			return
		}
		h.logger.WithError(err).WithField("session_id", session.SessionID).Warn("failed to list module messages from store")
	}

	if offset > len(session.Messages) {
		offset = len(session.Messages)
	}
	end := offset + limit
	if end > len(session.Messages) {
		end = len(session.Messages)
	}

	writeSuccess(w, http.StatusOK, map[string]interface{}{
		"messages": session.Messages[offset:end],
		"total":    len(session.Messages),
	})
}

func (h *Handler) AddInterviewModuleMessage(w http.ResponseWriter, r *http.Request) {
	session, ok := h.getModuleSessionFromRequest(w, r)
	if !ok {
		return
	}
	unlock, lockErr := h.acquireSessionLock(r.Context(), session.SessionID)
	if lockErr != nil {
		writeError(w, http.StatusConflict, "session is busy, retry")
		return
	}
	defer unlock()

	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	userID := h.userIDFromContext(r.Context())
	idemScopeKey := fmt.Sprintf("message:%s:%s:%s", userID, session.SessionID.String(), idempotencyKey)
	if idempotencyKey != "" {
		h.moduleMu.RLock()
		cached, exists := h.requestLog[idemScopeKey]
		h.moduleMu.RUnlock()
		if exists {
			writeSuccess(w, http.StatusAccepted, map[string]interface{}{
				"accepted":        true,
				"idempotent":      true,
				"cached_response": cached,
			})
			return
		}
	}

	if session.Status != "active" {
		writeError(w, http.StatusConflict, "session is not active")
		return
	}

	if time.Now().After(session.ExpiresAt) {
		h.finishModuleSession(r.Context(), session, "Interview timeout")
		writeError(w, http.StatusConflict, "session time limit reached")
		return
	}

	var req struct {
		Content         string `json:"content"`
		ClientMessageID string `json:"client_message_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		writeError(w, http.StatusBadRequest, "content is required")
		return
	}

	cleanContent := h.sanitizeCandidateText(req.Content)

	// Censorship: reject abusive language with a friendly 400 so the
	// chat doesn't ship vulgarities into the AI prompt. We also drop
	// outright off-topic ("solve me a math problem", "tell me a joke")
	// requests so the AI stays interview-focused.
	if isAbusive(cleanContent) {
		writeError(w, http.StatusBadRequest, "сообщение содержит недопустимые выражения. пожалуйста, перефразируйте.")
		return
	}

	// Short-answer guard. Anything under ~12 characters is either a
	// stray keystroke or "не знаю" — both should NOT count as a
	// graded turn. Surface a polite re-prompt instead of feeding it
	// to the LLM and inflating the final average.
	letters := utf8.RuneCountInString(strings.TrimSpace(cleanContent))
	if letters < 12 {
		writeError(w, http.StatusBadRequest,
			"пожалуйста, дайте развёрнутый ответ (хотя бы 1–2 предложения) или нажмите «следующий вопрос», чтобы пропустить.")
		return
	}

	msg := InterviewChatMessage{
		MessageID: uuid.New(),
		Sender:    "user",
		Content:   cleanContent,
		CreatedAt: time.Now(),
	}

	h.moduleMu.Lock()
	session.Messages = append(session.Messages, msg)
	questionsAsked := h.countAIMessages(session.Messages)
	h.moduleMu.Unlock()
	if h.moduleStore != nil {
		if err := h.moduleStore.CreateInterviewModuleMessage(r.Context(), h.toDomainMessage(session.SessionID, msg)); err != nil {
			h.logger.WithError(err).WithField("session_id", session.SessionID).Warn("failed to persist user message")
		}
		h.persistSession(r.Context(), session)
	}

	h.broadcastSessionEvent(session.SessionID, "message.user", msg)
	h.broadcastSessionEvent(session.SessionID, "ai.typing.started", map[string]bool{"typing": true})

	remainingQuestions := session.QuestionLimit - questionsAsked
	next, err := h.requestNextQuestion(r.Context(), session, cleanContent)
	if err != nil {
		h.logger.WithError(err).Warn("failed to generate follow-up question, using fallback")
		next = h.buildTechnicalFallbackQuestion(session, cleanContent)
		h.fallbackRate.Add(1)
	}

	// Attach AI verdict to the user message we just received, then
	// broadcast a dedicated message.user.evaluated event so the
	// chat UI can flip the bubble from "pending" to the verdict
	// badge. Anything not in the verdict allowlist is dropped.
	if verdict := normaliseVerdict(next.LastAnswerVerdict); verdict != "" {
		h.moduleMu.Lock()
		for i := len(session.Messages) - 1; i >= 0; i-- {
			if session.Messages[i].MessageID == msg.MessageID {
				session.Messages[i].Verdict = verdict
				session.Messages[i].VerdictReason = next.LastAnswerReason
				msg.Verdict = verdict
				msg.VerdictReason = next.LastAnswerReason
				break
			}
		}
		h.moduleMu.Unlock()
		h.broadcastSessionEvent(session.SessionID, "message.user.evaluated", map[string]interface{}{
			"message_id":     msg.MessageID,
			"verdict":        verdict,
			"verdict_reason": next.LastAnswerReason,
		})
	}

	if remainingQuestions <= 1 || next.ShouldEnd {
		h.finishModuleSession(r.Context(), session, "Interview completed")
		h.broadcastSessionEvent(session.SessionID, "session.finished", map[string]string{"reason": "limits reached"})
		writeSuccess(w, http.StatusAccepted, map[string]interface{}{"accepted": true, "message_id": msg.MessageID, "session_finished": true})
		return
	}

	h.moduleMu.Lock()
	session.CurrentTopic = next.Topic
	session.PressureLevel = next.PressureLevel
	h.applyDifficultyDelta(session, next.DifficultyDelta)
	aiMsg := InterviewChatMessage{
		MessageID:  uuid.New(),
		Sender:     "ai",
		Content:    h.sanitizeModelText(next.Question),
		Topic:      session.CurrentTopic,
		Difficulty: session.Difficulty,
		CreatedAt:  time.Now(),
	}
	session.Messages = append(session.Messages, aiMsg)
	session.TopicStats[session.CurrentTopic]++
	h.moduleMu.Unlock()
	if h.moduleStore != nil {
		if err := h.moduleStore.CreateInterviewModuleMessage(r.Context(), h.toDomainMessage(session.SessionID, aiMsg)); err != nil {
			h.logger.WithError(err).WithField("session_id", session.SessionID).Warn("failed to persist ai message")
		}
		h.persistSession(r.Context(), session)
	}

	h.rememberAskedQuestion(r.Context(), session.UserID, session.Role, aiMsg.Content)

	h.broadcastSessionEvent(session.SessionID, "ai.typing.stopped", map[string]bool{"typing": false})
	h.streamAIMessageContent(session.SessionID, aiMsg.Content)
	h.broadcastSessionEvent(session.SessionID, "message.ai", aiMsg)

	remainingSec := int(time.Until(session.ExpiresAt) / time.Second)
	if remainingSec > 0 && remainingSec <= 60 {
		h.broadcastSessionEvent(session.SessionID, "session.warning", map[string]interface{}{
			"reason":            "time_low",
			"remaining_seconds": remainingSec,
		})
	}

	responsePayload := map[string]interface{}{
		"accepted":   true,
		"message_id": msg.MessageID,
	}

	if idempotencyKey != "" {
		h.moduleMu.Lock()
		h.requestLog[idemScopeKey] = msg.MessageID.String()
		h.moduleMu.Unlock()
	}

	h.recordAuditEvent(r.Context(), session.SessionID, "llm_response", map[string]interface{}{"topic": aiMsg.Topic, "difficulty": aiMsg.Difficulty, "pressure": session.PressureLevel})

	writeSuccess(w, http.StatusAccepted, responsePayload)
}

func (h *Handler) FinishInterviewModuleSession(w http.ResponseWriter, r *http.Request) {
	session, ok := h.getModuleSessionFromRequest(w, r)
	if !ok {
		return
	}
	unlock, lockErr := h.acquireSessionLock(r.Context(), session.SessionID)
	if lockErr != nil {
		writeError(w, http.StatusConflict, "session is busy, retry")
		return
	}
	defer unlock()

	h.finishModuleSession(r.Context(), session, "Session finished by user")
	h.broadcastSessionEvent(session.SessionID, "session.finished", map[string]string{"reason": "manual"})

	writeSuccess(w, http.StatusOK, map[string]interface{}{
		"session_id": session.SessionID,
		"status":     session.Status,
	})
}

func (h *Handler) GetInterviewModuleReport(w http.ResponseWriter, r *http.Request) {
	session, ok := h.getModuleSessionFromRequest(w, r)
	if !ok {
		return
	}
	sessionID := session.SessionID

	h.moduleMu.RLock()
	report, ok := h.moduleReports[sessionID]
	h.moduleMu.RUnlock()
	if !ok && h.moduleStore != nil {
		domainReport, err := h.moduleStore.GetInterviewModuleReportBySessionID(r.Context(), sessionID)
		if err == nil && domainReport != nil {
			report = toAPIReport(domainReport)
			h.moduleMu.Lock()
			h.moduleReports[sessionID] = report
			h.moduleMu.Unlock()
			ok = true
		}
	}

	if !ok {
		writeError(w, http.StatusNotFound, "report not ready")
		return
	}

	writeSuccess(w, http.StatusOK, report)
}

func (h *Handler) GetMyInterviewAnalyticsReport(w http.ResponseWriter, r *http.Request) {
	userID := h.userIDFromContext(r.Context())
	if strings.TrimSpace(userID) == "" || userID == "anonymous" {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	now := time.Now()

	userSessions := make([]InterviewModuleSession, 0)
	userReports := make(map[uuid.UUID]InterviewModuleReport)

	if h.moduleStore != nil {
		storedSessions, err := h.moduleStore.ListInterviewModuleSessionsByUserID(r.Context(), toStableUserUUID(userID), 500, 0)
		if err != nil {
			h.logger.WithError(err).WithField("user_id", userID).Warn("failed to load user module sessions from store")
		} else {
			for _, stored := range storedSessions {
				if stored == nil {
					continue
				}
				domainMessages, msgErr := h.moduleStore.ListInterviewModuleMessagesBySessionID(r.Context(), stored.ID, 5000, 0)
				if msgErr != nil {
					h.logger.WithError(msgErr).WithField("session_id", stored.ID).Warn("failed to load module messages for analytics")
					continue
				}
				messages := make([]InterviewChatMessage, 0, len(domainMessages))
				for _, message := range domainMessages {
					messages = append(messages, toAPIMessage(message))
				}
				apiSession := h.toAPISession(stored, messages)
				if apiSession == nil || apiSession.UserID != userID {
					continue
				}
				userSessions = append(userSessions, *apiSession)

				report, reportErr := h.moduleStore.GetInterviewModuleReportBySessionID(r.Context(), stored.ID)
				if reportErr == nil && report != nil {
					mapped := toAPIReport(report)
					if mapped != nil {
						userReports[stored.ID] = *mapped
					}
				}
			}
		}
	}

	if len(userSessions) == 0 {
		h.moduleMu.RLock()
		for _, session := range h.moduleSessions {
			if session == nil || session.UserID != userID {
				continue
			}
			clone := *session
			clone.Messages = append([]InterviewChatMessage(nil), session.Messages...)
			clone.FocusAreas = append([]string(nil), session.FocusAreas...)
			clone.PrimarySkills = append([]string(nil), session.PrimarySkills...)
			clone.TheoryFocus = append([]string(nil), session.TheoryFocus...)
			clone.PracticeFocus = append([]string(nil), session.PracticeFocus...)
			if session.TopicStats != nil {
				clone.TopicStats = make(map[string]int, len(session.TopicStats))
				for k, v := range session.TopicStats {
					clone.TopicStats[k] = v
				}
			}
			userSessions = append(userSessions, clone)

			if report, ok := h.moduleReports[session.SessionID]; ok && report != nil {
				userReports[session.SessionID] = *report
			}
		}
		h.moduleMu.RUnlock()
	}

	report := userInterviewAnalyticsReport{
		UserID:               userID,
		GeneratedAt:          now,
		RoleDistribution:     []githubChartPoint{},
		ModeDistribution:     []githubChartPoint{},
		Timeline:             []userInterviewTimelinePoint{},
		TopStrengths:         []string{},
		TopWeaknesses:        []string{},
		TopRecommendations:   []string{},
		CompletedInterviews:  []userInterviewEntry{},
		IncompleteInterviews: []userInterviewEntry{},
		RecentInterviews:     []userInterviewEntry{},
	}

	if len(userSessions) == 0 {
		writeSuccess(w, http.StatusOK, report)
		return
	}

	roleCounters := map[string]int{}
	modeCounters := map[string]int{}
	timelineStarted := map[string]int{}
	timelineCompleted := map[string]int{}
	strengthCounters := map[string]int{}
	weaknessCounters := map[string]int{}
	recommendCounters := map[string]int{}

	completed := make([]userInterviewEntry, 0, len(userSessions))
	incomplete := make([]userInterviewEntry, 0, len(userSessions))
	recent := make([]userInterviewEntry, 0, len(userSessions))

	scoreTotal := 0.0
	bestScore := 0.0
	latestScore := 0.0
	scoredReports := 0
	questionCountTotal := 0
	durationMinutesTotal := 0
	latestScoreAt := time.Time{}

	for _, session := range userSessions {
		status := strings.ToLower(strings.TrimSpace(session.Status))
		if status == "" {
			status = "active"
		}
		if status != "finished" && now.After(session.ExpiresAt) {
			status = "expired"
		}

		entry := userInterviewEntry{
			SessionID:       session.SessionID,
			Role:            session.Role,
			Level:           session.Level,
			VacancyTitle:    session.VacancyTitle,
			InterviewMode:   normalizeInterviewMode(session.InterviewMode),
			Status:          status,
			CurrentTopic:    session.CurrentTopic,
			DurationMinutes: session.DurationMinutes,
			QuestionLimit:   session.QuestionLimit,
			MessagesTotal:   len(session.Messages),
			StartedAt:       session.StartedAt,
			ExpiresAt:       session.ExpiresAt,
			FinishedAt:      session.FinishedAt,
		}

		for _, msg := range session.Messages {
			sender := strings.ToLower(strings.TrimSpace(msg.Sender))
			if sender == "ai" {
				entry.AIMessages++
			} else if sender == "user" {
				entry.UserMessages++
			}
		}

		if moduleReport, ok := userReports[session.SessionID]; ok {
			score := moduleReport.OverallScore
			entry.OverallScore = &score
			entry.Strengths = append([]string(nil), moduleReport.Strengths...)
			entry.Weaknesses = append([]string(nil), moduleReport.Weaknesses...)

			scoreTotal += moduleReport.OverallScore
			scoredReports++
			if moduleReport.OverallScore > bestScore {
				bestScore = moduleReport.OverallScore
			}
			if moduleReport.GeneratedAt.After(latestScoreAt) {
				latestScoreAt = moduleReport.GeneratedAt
				latestScore = moduleReport.OverallScore
			}

			for _, strength := range moduleReport.Strengths {
				normalized := strings.TrimSpace(strings.ToLower(strength))
				if normalized != "" {
					strengthCounters[normalized]++
				}
			}
			for _, weakness := range moduleReport.Weaknesses {
				normalized := strings.TrimSpace(strings.ToLower(weakness))
				if normalized != "" {
					weaknessCounters[normalized]++
				}
			}
			for _, recommendation := range moduleReport.Recommendations {
				normalized := strings.TrimSpace(strings.ToLower(recommendation))
				if normalized != "" {
					recommendCounters[normalized]++
				}
			}
		}

		roleKey := strings.TrimSpace(session.Role)
		if roleKey == "" {
			roleKey = "unknown"
		}
		roleCounters[roleKey]++
		modeCounters[entry.InterviewMode]++

		startedDay := session.StartedAt.Format("2006-01-02")
		timelineStarted[startedDay]++
		if status == "finished" && session.FinishedAt != nil {
			finishedDay := session.FinishedAt.Format("2006-01-02")
			timelineCompleted[finishedDay]++
		}

		recent = append(recent, entry)
		if status == "finished" {
			completed = append(completed, entry)
		} else {
			incomplete = append(incomplete, entry)
		}

		questionCountTotal += entry.MessagesTotal
		durationMinutesTotal += session.DurationMinutes
	}

	sort.Slice(recent, func(i, j int) bool {
		return recent[i].StartedAt.After(recent[j].StartedAt)
	})
	sort.Slice(completed, func(i, j int) bool {
		return completed[i].StartedAt.After(completed[j].StartedAt)
	})
	sort.Slice(incomplete, func(i, j int) bool {
		return incomplete[i].StartedAt.After(incomplete[j].StartedAt)
	})

	totalInterviews := len(userSessions)
	completedCount := len(completed)
	inProgressCount := 0
	expiredCount := 0
	for _, entry := range incomplete {
		if entry.Status == "expired" {
			expiredCount++
		} else {
			inProgressCount++
		}
	}

	completionRate := 0.0
	if totalInterviews > 0 {
		completionRate = (float64(completedCount) / float64(totalInterviews)) * 100
	}

	averageScore := 0.0
	if scoredReports > 0 {
		averageScore = scoreTotal / float64(scoredReports)
	}

	avgQuestionCount := 0.0
	avgSessionMinutes := 0.0
	if totalInterviews > 0 {
		avgQuestionCount = float64(questionCountTotal) / float64(totalInterviews)
		avgSessionMinutes = float64(durationMinutesTotal) / float64(totalInterviews)
	}

	report.Totals = userInterviewTotals{
		TotalInterviews:      totalInterviews,
		CompletedInterviews:  completedCount,
		InProgressInterviews: inProgressCount,
		ExpiredInterviews:    expiredCount,
		CompletionRate:       math.Round(completionRate*100) / 100,
	}
	report.Performance = userInterviewPerformance{
		AverageScore:      math.Round(averageScore*100) / 100,
		BestScore:         math.Round(bestScore*100) / 100,
		LatestScore:       math.Round(latestScore*100) / 100,
		ReportsGenerated:  scoredReports,
		AvgQuestionCount:  math.Round(avgQuestionCount*100) / 100,
		AvgSessionMinutes: math.Round(avgSessionMinutes*100) / 100,
	}
	report.RoleDistribution = mapToSortedChart(roleCounters)
	report.ModeDistribution = mapToSortedChart(modeCounters)
	report.Timeline = buildTimeline(timelineStarted, timelineCompleted)
	report.TopStrengths = extractTopTextInsights(strengthCounters, 10)
	report.TopWeaknesses = extractTopTextInsights(weaknessCounters, 10)
	report.TopRecommendations = extractTopTextInsights(recommendCounters, 10)
	report.CompletedInterviews = completed
	report.IncompleteInterviews = incomplete
	if len(recent) > 30 {
		report.RecentInterviews = recent[:30]
	} else {
		report.RecentInterviews = recent
	}

	writeSuccess(w, http.StatusOK, report)
}

func mapToSortedChart(values map[string]int) []githubChartPoint {
	if len(values) == 0 {
		return []githubChartPoint{}
	}
	out := make([]githubChartPoint, 0, len(values))
	for label, value := range values {
		out = append(out, githubChartPoint{Label: label, Value: value})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Value == out[j].Value {
			return out[i].Label < out[j].Label
		}
		return out[i].Value > out[j].Value
	})
	return out
}

func extractTopTextInsights(values map[string]int, limit int) []string {
	if len(values) == 0 || limit <= 0 {
		return []string{}
	}
	type pair struct {
		text  string
		count int
	}
	pairs := make([]pair, 0, len(values))
	for text, count := range values {
		if strings.TrimSpace(text) == "" {
			continue
		}
		pairs = append(pairs, pair{text: text, count: count})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].count == pairs[j].count {
			return pairs[i].text < pairs[j].text
		}
		return pairs[i].count > pairs[j].count
	})
	if len(pairs) > limit {
		pairs = pairs[:limit]
	}
	out := make([]string, 0, len(pairs))
	for _, item := range pairs {
		clean := strings.TrimSpace(item.text)
		if clean == "" {
			continue
		}
		out = append(out, clean)
	}
	return out
}

func buildTimeline(started map[string]int, completed map[string]int) []userInterviewTimelinePoint {
	if len(started) == 0 && len(completed) == 0 {
		return []userInterviewTimelinePoint{}
	}
	allDays := make(map[string]struct{}, len(started)+len(completed))
	for day := range started {
		allDays[day] = struct{}{}
	}
	for day := range completed {
		allDays[day] = struct{}{}
	}
	days := make([]string, 0, len(allDays))
	for day := range allDays {
		days = append(days, day)
	}
	sort.Strings(days)
	out := make([]userInterviewTimelinePoint, 0, len(days))
	for _, day := range days {
		out = append(out, userInterviewTimelinePoint{
			Date:      day,
			Started:   started[day],
			Completed: completed[day],
		})
	}
	return out
}

func (h *Handler) HandleInterviewModuleWS(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID, err := uuid.Parse(vars["session_id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid session id")
		return
	}

	h.moduleMu.RLock()
	session, ok := h.moduleSessions[sessionID]
	h.moduleMu.RUnlock()
	if !ok && h.moduleStore != nil {
		loaded, loadErr := h.loadSessionFromStore(r.Context(), sessionID)
		if loadErr == nil && loaded != nil {
			h.moduleMu.Lock()
			h.moduleSessions[sessionID] = loaded
			h.moduleMu.Unlock()
			session = loaded
			ok = true
		}
	}
	if !ok {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	if session.UserID != h.userIDFromContext(r.Context()) {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}

	upgrader := websocket.Upgrader{CheckOrigin: func(_ *http.Request) bool { return true }}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.WithError(err).Warn("failed to upgrade websocket")
		return
	}

	h.moduleMu.Lock()
	if h.moduleWS[sessionID] == nil {
		h.moduleWS[sessionID] = make(map[*websocket.Conn]struct{})
	}
	if len(h.moduleWS[sessionID]) > 0 {
		h.wsReconnectCount.Add(1)
		h.reconnectAttempts.Add(1)
		go h.broadcastSessionEvent(sessionID, "session.warning", map[string]interface{}{
			"reason": "reconnecting",
		})
	}
	h.moduleWS[sessionID][conn] = struct{}{}
	h.reconnectSuccess.Add(1)
	h.moduleMu.Unlock()

	h.broadcastSessionEvent(sessionID, "session.started", map[string]interface{}{
		"session_id": sessionID,
		"resumed":    true,
	})

	h.moduleMu.RLock()
	moduleSession := h.moduleSessions[sessionID]
	h.moduleMu.RUnlock()
	if moduleSession != nil {
		remainingSec := int(time.Until(moduleSession.ExpiresAt) / time.Second)
		if remainingSec > 0 && remainingSec <= 60 {
			h.broadcastSessionEvent(sessionID, "session.warning", map[string]interface{}{
				"reason":            "time_low",
				"remaining_seconds": remainingSec,
			})
		}
	}

	defer func() {
		h.moduleMu.Lock()
		delete(h.moduleWS[sessionID], conn)
		h.moduleMu.Unlock()
		_ = conn.Close()
	}()

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (h *Handler) getModuleSessionFromRequest(w http.ResponseWriter, r *http.Request) (*InterviewModuleSession, bool) {
	vars := mux.Vars(r)
	sessionID, err := uuid.Parse(vars["session_id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid session id")
		return nil, false
	}

	h.moduleMu.RLock()
	session, ok := h.moduleSessions[sessionID]
	h.moduleMu.RUnlock()
	if !ok && h.moduleStore != nil {
		loaded, loadErr := h.loadSessionFromStore(r.Context(), sessionID)
		if loadErr == nil && loaded != nil {
			h.moduleMu.Lock()
			h.moduleSessions[sessionID] = loaded
			h.moduleMu.Unlock()
			session = loaded
			ok = true
		}
	}
	if !ok {
		writeError(w, http.StatusNotFound, "session not found")
		return nil, false
	}

	if session.UserID != h.userIDFromContext(r.Context()) {
		writeError(w, http.StatusForbidden, "access denied")
		return nil, false
	}

	return session, true
}

func (h *Handler) requestNextQuestion(ctx context.Context, session *InterviewModuleSession, lastAnswer string) (*nextQuestionResponse, error) {
	normalizedMode := strings.ToLower(strings.TrimSpace(session.InterviewMode))
	if normalizedMode == "" {
		normalizedMode = "practice"
	}

	if normalizedMode != "theory" {
		if intent := h.detectPracticeControlIntent(lastAnswer); intent != "" {
			return h.buildPracticeControlResponse(session, intent), nil
		}
	}

	answerSignal := h.classifyAnswer(lastAnswer)
	intent := h.detectCandidateIntent(lastAnswer)
	askedBefore := h.countAIMessages(session.Messages)
	h.moduleMu.Lock()
	h.updateDifficultyAndPressure(session, answerSignal)
	if session.CurrentTopic == "skills_overview" && askedBefore >= 1 {
		session.CurrentTopic = h.nextTopic(session.Role, session.TopicCursor)
		session.TopicCursor++
	}
	if h.shouldSwitchTopic(session, answerSignal) {
		session.CurrentTopic = h.nextTopic(session.Role, session.TopicCursor)
		session.TopicCursor++
	}
	h.moduleMu.Unlock()

	timeLeft := time.Until(session.ExpiresAt)
	if timeLeft < 0 {
		timeLeft = 0
	}

	asked := h.countAIMessages(session.Messages)
	requestCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()

	backoff := []time.Duration{250 * time.Millisecond, 650 * time.Millisecond}
	var lastErr error
	duplicateSeen := map[string]struct{}{}

	for attempt := 0; attempt < len(backoff); attempt++ {
		avoidQuestions := h.collectAvoidQuestions(requestCtx, session, 40)
		for q := range duplicateSeen {
			avoidQuestions = append(avoidQuestions, q)
		}

		body := nextQuestionRequest{
			Role:                session.Role,
			Level:               session.Level,
			VacancyTitle:        session.VacancyTitle,
			VacancyCategory:     session.VacancyCategory,
			InterviewMode:       normalizedMode,
			SessionContext:      h.buildInterviewSessionContext(session, session.CurrentTopic, lastAnswer, normalizedMode),
			RecentTopics:        h.recentAITopics(session, 5),
			FocusAreas:          session.FocusAreas,
			PrimarySkills:       session.PrimarySkills,
			TheoryFocus:         session.TheoryFocus,
			PracticeFocus:       session.PracticeFocus,
			CurrentTopic:        session.CurrentTopic,
			Difficulty:          session.Difficulty,
			PressureLevel:       session.PressureLevel,
			TimeLeftSec:         int64(timeLeft / time.Second),
			QuestionsLeft:       session.QuestionLimit - asked,
			LastCandidateAnswer: h.sanitizeCandidateText(lastAnswer),
			History:             session.Messages,
			AvoidQuestions:      avoidQuestions,
			TurnNonce:           uuid.NewString(),
		}

		payload, _ := json.Marshal(body)

		start := time.Now()
		out, err := h.callAIWithFailover(requestCtx, session, payload)
		latency := time.Since(start).Milliseconds()
		h.llmLatencyMs.Store(latency)

		h.logger.WithFields(logrus.Fields{
			"metric":  "llm_latency_ms",
			"value":   latency,
			"attempt": attempt + 1,
		}).Info("llm request measured")

		if err == nil && strings.TrimSpace(out.Question) != "" {
			h.applyAnswerSignalToResponse(out, session, answerSignal, intent, lastAnswer)

			if h.isQuestionRepeated(requestCtx, session, out.Question) {
				duplicateSeen[out.Question] = struct{}{}
				lastErr = fmt.Errorf("duplicate interviewer question")
				break
			}

			if out.Flags != nil && out.Flags["policy_violation"] {
				h.policyViolations.Add(1)
				h.recordAuditEvent(ctx, session.SessionID, "policy_decision", map[string]interface{}{"result": "violation", "flags": out.Flags})
			} else {
				h.recordAuditEvent(ctx, session.SessionID, "policy_decision", map[string]interface{}{"result": "ok"})
			}
			h.recordLatencySample(&h.llmLatencySamples, latency)
			return out, nil
		}

		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("empty interviewer question")
		}

		if requestCtx.Err() != nil {
			break
		}
		time.Sleep(backoff[attempt])
	}

	return nil, lastErr
}

func (h *Handler) finishModuleSession(ctx context.Context, session *InterviewModuleSession, feedback string) {
	now := time.Now()
	h.moduleMu.Lock()
	session.Status = "finished"
	session.FinishedAt = &now
	h.moduleMu.Unlock()
	h.persistSession(ctx, session)

	select {
	case h.scoreQueue <- scoreJob{SessionID: session.SessionID, Feedback: feedback}:
	default:
		h.logger.Warn("score queue full, using direct scoring fallback")
		report := h.requestScoringReport(ctx, session, feedback)
		h.moduleMu.Lock()
		h.moduleReports[session.SessionID] = report
		h.moduleMu.Unlock()
		if h.moduleStore != nil {
			if err := h.moduleStore.UpsertInterviewModuleReport(ctx, h.toDomainReport(report)); err != nil {
				h.logger.WithError(err).WithField("session_id", session.SessionID).Warn("failed to persist interview report")
			}
		}
		h.broadcastSessionEvent(session.SessionID, "report.ready", map[string]string{"session_id": session.SessionID.String()})
	}
}

func (h *Handler) callAIWithFailover(ctx context.Context, session *InterviewModuleSession, payload []byte) (*nextQuestionResponse, error) {
	urls := make([]string, 0, 2)
	if h.shouldUsePrimaryAI() {
		urls = append(urls, h.aiServiceURL)
	}
	if h.secondaryAIURL != "" {
		urls = append(urls, h.secondaryAIURL)
	}
	if len(urls) == 0 {
		urls = append(urls, h.aiServiceURL)
	}

	client := &http.Client{Timeout: 8 * time.Second}
	var lastErr error

	for _, baseURL := range urls {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/v1/interviewer/next-question", bytes.NewReader(payload))
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			h.recordAIProviderFailure(baseURL)
			continue
		}

		if resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests {
			lastErr = fmt.Errorf("provider %s returned status %d", baseURL, resp.StatusCode)
			resp.Body.Close()
			h.recordAIProviderFailure(baseURL)
			continue
		}
		if resp.StatusCode >= 300 {
			lastErr = fmt.Errorf("provider %s returned status %d", baseURL, resp.StatusCode)
			resp.Body.Close()
			continue
		}

		var out nextQuestionResponse
		decodeErr := json.NewDecoder(resp.Body).Decode(&out)
		resp.Body.Close()
		if decodeErr != nil {
			lastErr = decodeErr
			h.recordAIProviderFailure(baseURL)
			continue
		}

		validatedQuestion, err := h.validateInterviewerOutput(ctx, session, baseURL, out.Question)
		if err != nil {
			lastErr = err
			h.policyViolations.Add(1)
			h.recordAuditEvent(ctx, uuid.Nil, "policy_decision", map[string]interface{}{"result": "violation", "provider": baseURL, "error": err.Error()})
			h.recordAIProviderFailure(baseURL)
			continue
		}
		if strings.TrimSpace(validatedQuestion) != "" {
			out.Question = validatedQuestion
		}

		h.recordAIProviderSuccess(baseURL)
		return &out, nil
	}

	return nil, lastErr
}

func (h *Handler) shouldUsePrimaryAI() bool {
	h.cbMu.Lock()
	defer h.cbMu.Unlock()
	return !time.Now().Before(h.aiCircuit.OpenUntil)
}

func (h *Handler) recordAIProviderFailure(baseURL string) {
	if baseURL != h.aiServiceURL {
		return
	}
	h.cbMu.Lock()
	defer h.cbMu.Unlock()
	h.aiCircuit.Failures++
	if h.aiCircuit.Failures >= 3 {
		h.aiCircuit.OpenUntil = time.Now().Add(45 * time.Second)
		h.aiCircuit.Failures = 0
	}
}

func (h *Handler) recordAIProviderSuccess(baseURL string) {
	if baseURL != h.aiServiceURL {
		return
	}
	h.cbMu.Lock()
	defer h.cbMu.Unlock()
	h.aiCircuit.Failures = 0
	h.aiCircuit.OpenUntil = time.Time{}
}

func (h *Handler) runScoringWorker() {
	for job := range h.scoreQueue {
		start := time.Now()
		h.moduleMu.RLock()
		session := h.moduleSessions[job.SessionID]
		h.moduleMu.RUnlock()
		if session == nil && h.moduleStore != nil {
			loaded, err := h.loadSessionFromStore(context.Background(), job.SessionID)
			if err == nil {
				h.moduleMu.Lock()
				h.moduleSessions[job.SessionID] = loaded
				h.moduleMu.Unlock()
				session = loaded
			}
		}
		if session == nil {
			continue
		}

		report := h.requestScoringReport(context.Background(), session, job.Feedback)
		h.moduleMu.Lock()
		h.moduleReports[job.SessionID] = report
		h.moduleMu.Unlock()
		if h.moduleStore != nil {
			if err := h.moduleStore.UpsertInterviewModuleReport(context.Background(), h.toDomainReport(report)); err != nil {
				h.logger.WithError(err).WithField("session_id", job.SessionID).Warn("failed to persist generated report")
			}
		}

		elapsedMs := time.Since(start).Milliseconds()
		h.reportGenMs.Store(elapsedMs)
		h.recordLatencySample(&h.reportLatency, elapsedMs)
		h.logger.WithFields(logrus.Fields{"metric": "report_generation_time_ms", "value": elapsedMs}).Info("report generation completed")

		h.broadcastSessionEvent(job.SessionID, "report.ready", map[string]string{"session_id": job.SessionID.String()})
	}
}

func (h *Handler) requestScoringReport(ctx context.Context, session *InterviewModuleSession, feedback string) *InterviewModuleReport {
	body := map[string]interface{}{
		"session_id": session.SessionID.String(),
		"role":       session.Role,
		"level":      session.Level,
		"messages":   session.Messages,
		"feedback":   feedback,
	}
	payload, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.scoringServiceURL+"/api/v1/scoring/generate", bytes.NewReader(payload))
	if err != nil {
		return h.localFallbackReport(session)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return h.localFallbackReport(session)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return h.localFallbackReport(session)
	}

	var parsed InterviewModuleReport
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return h.localFallbackReport(session)
	}
	if parsed.GeneratedAt.IsZero() {
		parsed.GeneratedAt = time.Now()
	}
	if parsed.SessionID == uuid.Nil {
		parsed.SessionID = session.SessionID
	}
	return &parsed
}

// localFallbackReport is used when the scoring-service can't be
// reached. It MUST NOT invent scores — previously it returned
// 40+ baseline marks even for sessions with zero candidate answers,
// which is what the user saw as "средние баллы за пустое интервью".
//
// New behaviour:
//   - Count substantive answers (anything that passed the
//     length-guard, i.e. >=12 chars after trim). Anything shorter
//     was rejected at handler level, but defensively re-check here.
//   - If there are no substantive answers, return all zeros and an
//     honest "не было засчитанных ответов" weakness — the candidate
//     should not be rewarded for silence.
//   - Otherwise grade conservatively from length + technical-marker
//     density per answer, capped at 70. The real AI/LLM scoring
//     remains the source of truth; this fallback only fires when
//     scoring-service is offline.
func (h *Handler) localFallbackReport(session *InterviewModuleSession) *InterviewModuleReport {
	h.fallbackRate.Add(1)
	h.logger.WithFields(logrus.Fields{"metric": "fallback_rate", "value": h.fallbackRate.Load()}).Warn("fallback report used")

	type answerSample struct {
		length     int
		signalHits int
		uncertain  bool
	}
	var samples []answerSample
	techMarkers := []string{
		"trade-off", "tradeoff", "компромисс", "latency", "p95", "p99",
		"throughput", "qps", "метрик", "consistency", "идемпотент",
		"timeout", "retry", "cache", "кэш", "shard", "partition",
		"репликац", "deadlock", "race condition",
	}

	for _, msg := range session.Messages {
		if msg.Sender != "user" {
			continue
		}
		trimmed := strings.TrimSpace(msg.Content)
		runes := utf8.RuneCountInString(trimmed)
		if runes < 12 {
			continue // not a real answer
		}
		lower := strings.ToLower(trimmed)
		hits := 0
		for _, marker := range techMarkers {
			if strings.Contains(lower, marker) {
				hits++
			}
		}
		samples = append(samples, answerSample{
			length:     runes,
			signalHits: hits,
			uncertain:  h.isUncertainAnswer(trimmed),
		})
	}

	if len(samples) == 0 {
		return &InterviewModuleReport{
			SessionID:    session.SessionID,
			Correctness:  0,
			Clarity:      0,
			Completeness: 0,
			Relevance:    0,
			OverallScore: 0,
			Strengths:    []string{},
			Weaknesses: []string{
				"Кандидат не дал ни одного развёрнутого ответа — оценить нечего.",
			},
			Recommendations: []string{
				"Попробуйте пройти интервью снова, отвечая на каждый вопрос хотя бы 1–2 предложениями.",
			},
			GeneratedAt: time.Now(),
		}
	}

	// Per-sample score: length contributes up to 35, tech markers up
	// to 25 (capped at 5 hits), uncertainty subtracts 25. Average
	// across samples gives a conservative correctness estimate.
	totalCorrectness := 0.0
	totalClarity := 0.0
	totalCompleteness := 0.0
	for _, s := range samples {
		base := 20.0 + math.Min(35.0, float64(s.length)/8.0) + math.Min(25.0, float64(s.signalHits)*5.0)
		if s.uncertain {
			base -= 25.0
		}
		if base < 0 {
			base = 0
		}
		if base > 70 {
			base = 70 // never claim mastery from a heuristic fallback
		}
		totalCorrectness += base
		totalClarity += math.Max(0, base-3)
		totalCompleteness += math.Max(0, base-2)
	}
	n := float64(len(samples))
	correctness := math.Round(totalCorrectness / n)
	clarity := math.Round(totalClarity / n)
	completeness := math.Round(totalCompleteness / n)
	relevance := math.Round((correctness + completeness) / 2)
	overall := math.Round((correctness + clarity + completeness + relevance) / 4)

	weaknesses := []string{}
	recs := []string{}
	if correctness < 40 {
		weaknesses = append(weaknesses, "Ответы поверхностные — недостаточно конкретики и обоснований.")
		recs = append(recs, "Тренируйте структуру ответа: тезис → аргументы → пример → trade-offs.")
	} else if correctness < 60 {
		weaknesses = append(weaknesses, "Есть пробелы в trade-offs и edge-cases.")
		recs = append(recs, "Добавляйте в ответы метрики (latency/p95/throughput) и обоснование выбора.")
	}
	if len(weaknesses) == 0 {
		weaknesses = append(weaknesses, "Не все ответы подкреплены конкретными метриками результата.")
	}
	if len(recs) == 0 {
		recs = append(recs, "Продолжайте практику на интервью более высокого уровня.")
	}

	return &InterviewModuleReport{
		SessionID:       session.SessionID,
		Correctness:     correctness,
		Clarity:         clarity,
		Completeness:    completeness,
		Relevance:       relevance,
		OverallScore:    overall,
		Strengths:       []string{},
		Weaknesses:      weaknesses,
		Recommendations: recs,
		GeneratedAt:     time.Now(),
	}
}

func (h *Handler) applyDifficultyDelta(session *InterviewModuleSession, delta int) {
	session.Difficulty += delta
	if session.Difficulty < 1 {
		session.Difficulty = 1
	}
	if session.Difficulty > 10 {
		session.Difficulty = 10
	}
}

func (h *Handler) countAIMessages(messages []InterviewChatMessage) int {
	count := 0
	for _, msg := range messages {
		if msg.Sender == "ai" {
			count++
		}
	}
	return count
}

func (h *Handler) broadcastSessionEvent(sessionID uuid.UUID, eventType string, payload interface{}) {
	h.moduleMu.RLock()
	connections := h.moduleWS[sessionID]
	h.moduleMu.RUnlock()

	if len(connections) == 0 {
		return
	}

	message := map[string]interface{}{
		"version":   "v1",
		"type":      eventType,
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"payload":   payload,
	}
	data, _ := json.Marshal(message)

	for conn := range connections {
		_ = conn.WriteMessage(websocket.TextMessage, data)
	}
}

// streamAIMessageContent simulates token-by-token streaming over the
// existing WS channel. The frontend already handles ai.message.chunk
// events (see chatStore.pushStreamChunk), so by emitting word-grouped
// chunks with a small inter-chunk delay we get a typewriter-style
// experience without introducing a real LLM streaming hop.
//
// Behaviour:
//   - Splits the message preserving whitespace, then groups 1-3 tokens
//     per chunk so the buffer grows in human-readable bursts (not single
//     letters).
//   - Sleeps proportionally to chunk length, capped, so a 700-char
//     question streams in ~1.5-2.5s — perceived as "thinking, typing
//     fast" rather than "instant" or "stuck".
//   - Total stream is bounded by an upper deadline (maxDuration) so a
//     pathological message can never block downstream message.ai.
//   - Aborts immediately if the session has no live WS connections,
//     since chunks are useless without subscribers.
func (h *Handler) streamAIMessageContent(sessionID uuid.UUID, content string) {
	if strings.TrimSpace(content) == "" {
		return
	}

	h.moduleMu.RLock()
	hasSubscribers := len(h.moduleWS[sessionID]) > 0
	h.moduleMu.RUnlock()
	if !hasSubscribers {
		return
	}

	chunks := splitIntoStreamChunks(content)
	if len(chunks) == 0 {
		return
	}

	const (
		perChunkBaseMS  = 18
		perCharMS       = 4
		maxChunkPauseMS = 110
		maxDuration     = 2500 * time.Millisecond
	)

	deadline := time.Now().Add(maxDuration)
	for _, chunk := range chunks {
		h.broadcastSessionEvent(sessionID, "ai.message.chunk", map[string]string{"chunk": chunk})

		// Inter-chunk pause scales with chunk size, capped, and never exceeds
		// the overall deadline.
		pause := time.Duration(perChunkBaseMS+len(chunk)*perCharMS) * time.Millisecond
		if pause > time.Duration(maxChunkPauseMS)*time.Millisecond {
			pause = time.Duration(maxChunkPauseMS) * time.Millisecond
		}
		if remaining := time.Until(deadline); remaining < pause {
			pause = remaining
		}
		if pause <= 0 {
			continue
		}
		time.Sleep(pause)
	}
}

// splitIntoStreamChunks groups the message into 1-3 token bursts while
// preserving the trailing whitespace so the concatenated chunks
// reconstruct the original content byte-for-byte. Code/punctuation
// boundaries get their own chunk so JSON-ish or technical answers
// remain readable mid-stream.
func splitIntoStreamChunks(content string) []string {
	if content == "" {
		return nil
	}
	// Tokenise: word + trailing whitespace + trailing punctuation cluster.
	tokens := tokenizeForStream(content)
	if len(tokens) == 0 {
		return []string{content}
	}

	chunks := make([]string, 0, len(tokens))
	var buf strings.Builder
	tokensInBuf := 0
	for _, tok := range tokens {
		buf.WriteString(tok)
		tokensInBuf++
		// Flush every 1-3 tokens; on punctuation/newline we flush early
		// so the typewriter pauses naturally at sentence breaks.
		if tokensInBuf >= 3 || endsAtBoundary(tok) {
			chunks = append(chunks, buf.String())
			buf.Reset()
			tokensInBuf = 0
		}
	}
	if buf.Len() > 0 {
		chunks = append(chunks, buf.String())
	}
	return chunks
}

func tokenizeForStream(s string) []string {
	out := make([]string, 0, len(s)/4+1)
	var current strings.Builder
	flush := func() {
		if current.Len() > 0 {
			out = append(out, current.String())
			current.Reset()
		}
	}
	for _, r := range s {
		current.WriteRune(r)
		if r == ' ' || r == '\n' || r == '\t' {
			flush()
		}
	}
	flush()
	return out
}

func endsAtBoundary(token string) bool {
	if token == "" {
		return false
	}
	// Strip the trailing whitespace we kept on each token, then look at
	// the last meaningful rune.
	trimmed := strings.TrimRight(token, " \t\n")
	if trimmed == "" {
		return false
	}
	last := trimmed[len(trimmed)-1]
	switch last {
	case '.', '!', '?', ':', ';', ',', ')', ']', '}', '"':
		return true
	}
	return false
}

func (h *Handler) userIDFromContext(ctx context.Context) string {
	if v := ctx.Value(ContextKeyUserID); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return "anonymous"
}

func (h *Handler) acquireSessionLock(ctx context.Context, sessionID uuid.UUID) (func(), error) {
	if h.redis == nil {
		return func() {}, nil
	}

	lockKey := "interview:session:lock:" + sessionID.String()
	lockValue := uuid.NewString()
	acquired, err := h.redis.SetNX(ctx, lockKey, lockValue, 12*time.Second).Result()
	if err != nil {
		return nil, err
	}
	if !acquired {
		return nil, fmt.Errorf("lock already acquired")
	}

	unlock := func() {
		_ = h.redis.Del(context.Background(), lockKey).Err()
	}

	return unlock, nil
}

// classifyAnswer scores a candidate answer on a small set of textual
// signals and returns one of "weak" / "neutral" / "strong". The result
// drives difficulty/pressure adjustments and topic-switch decisions.
//
// Weighting (heuristic, intentionally bounded so a single signal can't
// dominate):
//   - Length tier (short < 80 chars → -2, long > 350 chars → +1)
//   - Uncertainty markers ("не знаю", "затрудняюсь" etc.) → -3
//   - Engineering-density vocabulary (trade-off, latency, p95, метрик,
//     consistency, idempot...) → +1 each, capped at +3
//   - Code block / fenced snippet → +2 (live coding answers)
//   - Numeric reasoning ("в 2 раза", "100ms", "10x") → +1
//   - Structural cues (bullets, "сначала ... затем") → +1
//
// Negative cumulative score → weak, ≥3 → strong, otherwise neutral.
func (h *Handler) classifyAnswer(answer string) string {
	trimmed := strings.TrimSpace(answer)
	if trimmed == "" {
		return "weak"
	}
	lowered := strings.ToLower(trimmed)
	score := 0

	switch {
	case len(trimmed) < 30:
		score -= 3
	case len(trimmed) < 80:
		score -= 2
	case len(trimmed) > 600:
		score += 2
	case len(trimmed) > 350:
		score += 1
	}

	if h.isUncertainAnswer(trimmed) {
		score -= 3
	}

	techMarkers := []string{
		"trade-off", "tradeoff", "компромисс",
		"latency", "p95", "p99", "throughput", "qps",
		"метрик", "monitoring", "observability",
		"consistency", "идемпотент", "idempot",
		"transaction", "транзакц",
		"timeout", "retry", "circuit breaker",
		"cache", "кэш", "ttl",
		"sharding", "partition", "репликац",
		"deadlock", "race condition", "гонк",
	}
	techHits := 0
	for _, marker := range techMarkers {
		if strings.Contains(lowered, marker) {
			techHits++
		}
	}
	if techHits > 3 {
		techHits = 3
	}
	score += techHits

	// Fenced code or inline backticks → strong signal of practice answer.
	if strings.Contains(trimmed, "```") || strings.Count(trimmed, "`") >= 4 {
		score += 2
	}

	// Numeric reasoning — answers that put a number on the claim.
	if matched, _ := regexp.MatchString(`\b\d+\s*(ms|сек|секунд|min|мин|%|x|раз|qps|rps|gb|mb)\b`, lowered); matched {
		score += 1
	}

	// Structural cues.
	if strings.Contains(trimmed, "\n- ") || strings.Contains(trimmed, "\n* ") {
		score += 1
	}
	if strings.Contains(lowered, "сначала") && (strings.Contains(lowered, "затем") || strings.Contains(lowered, "потом")) {
		score += 1
	}
	// Counter-questions show engagement.
	if strings.Contains(trimmed, "?") && len(trimmed) > 120 {
		score += 1
	}

	switch {
	case score <= -2:
		return "weak"
	case score >= 3:
		return "strong"
	default:
		return "neutral"
	}
}

func (h *Handler) updateDifficultyAndPressure(session *InterviewModuleSession, signal string) {
	if strings.EqualFold(session.Level, "senior") {
		session.PressureLevel = minInt(5, session.PressureLevel+1)
	}

	switch signal {
	case "strong":
		session.Difficulty = minInt(10, session.Difficulty+1)
		session.PressureLevel = minInt(5, session.PressureLevel+1)
		session.WeakAnswerStreak = 0
	case "weak":
		session.Difficulty = maxInt(1, session.Difficulty-1)
		session.PressureLevel = maxInt(1, session.PressureLevel-1)
		session.WeakAnswerStreak++
	default:
		session.WeakAnswerStreak = 0
	}
}

func (h *Handler) shouldSwitchTopic(session *InterviewModuleSession, signal string) bool {
	if session.TopicStats == nil {
		session.TopicStats = map[string]int{}
	}
	questionsOnTopic := session.TopicStats[session.CurrentTopic]
	threshold := 3
	if strings.EqualFold(session.Level, "senior") || signal == "strong" {
		threshold = 2
	}
	if questionsOnTopic >= threshold {
		return true
	}
	return session.WeakAnswerStreak >= 2
}

func (h *Handler) nextTopic(role string, cursor int) string {
	matrix := map[string][]string{
		"web":        {"frontend", "backend", "performance", "accessibility", "architecture"},
		"backend":    {"concurrency", "db", "api_design", "caching", "distributed_systems"},
		"frontend":   {"rendering", "state_management", "performance", "accessibility", "testing"},
		"devops":     {"ci_cd", "observability", "kubernetes", "incident_response", "security"},
		"ml":         {"data_quality", "evaluation", "inference", "drift", "training"},
		"data":       {"modeling", "pipelines", "data_quality", "storage", "serving"},
		"mobile":     {"architecture", "offline_sync", "performance", "testing", "release"},
		"game":       {"gameplay", "engine", "rendering", "networking", "tools"},
		"security":   {"threat_modeling", "secure_coding", "pentest", "incident_response", "reverse_engineering"},
		"systems":    {"os", "kernel", "embedded", "compiler", "desktop"},
		"enterprise": {"qa_automation", "erp", "salesforce", "low_code", "sdet"},
		"fintech":    {"hft", "quant", "blockchain", "risk", "latency"},
		"iot":        {"firmware", "connectivity", "robotics", "sensors", "reliability"},
		"management": {"product", "project", "analysis", "documentation", "devrel"},
		"fullstack":  {"api_design", "db", "frontend_performance", "caching", "distributed_systems"},
	}

	key := h.roleKey(role)
	topics, ok := matrix[key]
	if !ok || len(topics) == 0 {
		topics = []string{"experience", "system_design", "debugging", "scalability", "quality"}
	}

	if cursor < 0 {
		return topics[0]
	}
	return topics[(cursor+1)%len(topics)]
}

func (h *Handler) applyAnswerSignalToResponse(out *nextQuestionResponse, session *InterviewModuleSession, signal string, intent string, lastAnswer string) {
	out.Question = h.sanitizeModelText(out.Question)
	out.Topic = strings.TrimSpace(out.Topic)
	if out.Topic == "" {
		out.Topic = session.CurrentTopic
	}

	if strings.ToLower(strings.TrimSpace(session.InterviewMode)) != "theory" {
		out.Topic = "live_coding"
		if !h.isPracticeTaskQuestion(out.Question) {
			out.Question = h.buildPracticeTaskQuestion(session, out.Topic)
		}
		feedback := "Принято. Проверьте аккуратность обработки edge cases и оценку сложности."
		errors := ""
		hint := ""
		switch signal {
		case "weak":
			feedback = "Вижу ваш ход мысли. В текущем решении есть ошибки или пробелы. Подсказка: проверьте граничные случаи, обработку пустого ввода и корректность базовой логики."
			errors = "1) Неполная обработка edge cases. 2) Нет явной проверки пустого/некорректного ввода. 3) Не зафиксирована ожидаемая сложность."
			hint = "Разбейте решение на шаги: validate input -> core logic -> post-check invariants. Добавьте минимум 3 тест-кейса: empty, nominal, stress."
		case "strong":
			feedback = "Отличный прогресс. Решение выглядит уверенно, переходим к следующему заданию."
			hint = "Сохраните стиль: короткая сигнатура, явные проверки, комментарий по O(...)."
		default:
			hint = "Давайте докрутим решение: проверьте консистентность логики и убедитесь, что базовые кейсы не ломаются."
		}
		anchor := h.extractCandidateFocus(lastAnswer)
		if anchor != "" {
			feedback = feedback + " Отмечу ваш фокус: " + anchor + "."
		}
		out.Question = h.composePracticeTurn(feedback, errors, hint, out.Question)
		out.PressureLevel = maxInt(1, out.PressureLevel)
		return
	}

	if intent == "clarify" {
		hint := h.topicHint(session.Role, out.Topic)
		out.Question = "Понял, поясню проще. Что здесь важно: " + hint + " Теперь ответьте коротко в 3 шага: подход, риски, проверка результата."
		out.DifficultyDelta = -1
		out.PressureLevel = maxInt(1, session.PressureLevel-1)
		return
	}

	if intent == "switch" {
		out.Topic = h.nextTopic(session.Role, session.TopicCursor)
		session.TopicCursor++
		out.Question = "Ок, сменим угол. Новый вопрос по теме " + out.Topic + ": " + h.topicPrompt(session.Role, out.Topic)
		out.DifficultyDelta = -1
		out.PressureLevel = maxInt(1, session.PressureLevel-1)
		return
	}

	switch signal {
	case "strong":
		if out.DifficultyDelta < 1 {
			out.DifficultyDelta = 1
		}
		out.PressureLevel = minInt(5, maxInt(session.PressureLevel, out.PressureLevel))
	case "weak":
		if out.DifficultyDelta > 0 {
			out.DifficultyDelta = 0
		}
		if out.DifficultyDelta == 0 {
			out.DifficultyDelta = -1
		}
		out.Topic = session.CurrentTopic
		if !strings.Contains(strings.ToLower(out.Question), "шаг") {
			hint := h.topicHint(session.Role, out.Topic)
			out.Question = "Нормально, давайте без давления. Короткая подсказка: " + hint + " Ответьте в формате: 1) что делаем, 2) где риск, 3) как проверим."
		}
		out.PressureLevel = maxInt(1, minInt(session.PressureLevel, out.PressureLevel))
	default:
		out.PressureLevel = minInt(5, maxInt(1, out.PressureLevel))
	}

	out.Question = h.humanizeTheoryQuestion(out.Question, signal, intent, out.Topic, lastAnswer)
}

func (h *Handler) humanizeTheoryQuestion(question string, signal string, intent string, topic string, lastAnswer string) string {
	base := strings.TrimSpace(question)
	if base == "" {
		return base
	}

	seed := len(strings.TrimSpace(lastAnswer)) + len(strings.TrimSpace(topic))
	if seed < 0 {
		seed = -seed
	}

	prefixesStrong := []string{
		"Хороший разбор.",
		"Сильный ответ.",
		"Видно уверенное мышление.",
	}
	prefixesNeutral := []string{
		"Принял.",
		"Хорошо, двигаемся дальше.",
		"Окей, продолжаем.",
	}
	prefixesWeak := []string{
		"Нормально, идем шаг за шагом.",
		"Не страшно, давайте упростим фокус.",
		"Хорошо, помогу структурировать ответ.",
	}

	var prefix string
	switch signal {
	case "strong":
		prefix = prefixesStrong[seed%len(prefixesStrong)]
	case "weak":
		prefix = prefixesWeak[seed%len(prefixesWeak)]
	default:
		prefix = prefixesNeutral[seed%len(prefixesNeutral)]
	}

	if intent == "clarify" {
		prefix = "Отличный запрос на уточнение."
	}
	if intent == "switch" {
		prefix = "Хорошо, переключаемся на новый угол."
	}

	anchor := h.extractCandidateFocus(lastAnswer)
	if anchor != "" {
		prefix = prefix + " Вы правильно подняли тему: " + anchor + "."
	}

	return strings.TrimSpace(prefix + " " + base)
}

func (h *Handler) extractCandidateFocus(answer string) string {
	v := strings.ToLower(strings.TrimSpace(answer))
	if v == "" {
		return ""
	}

	focuses := []struct {
		keys  []string
		label string
	}{
		{keys: []string{"latency", "p95", "p99", "задерж"}, label: "latency/performance"},
		{keys: []string{"кеш", "cache", "кэш"}, label: "кэширование"},
		{keys: []string{"безопас", "security", "уязв"}, label: "безопасность"},
		{keys: []string{"тест", "test", "провер"}, label: "тестирование и валидация"},
		{keys: []string{"база", "db", "sql", "индекс"}, label: "работа с данными/БД"},
		{keys: []string{"масштаб", "scal", "нагруз"}, label: "масштабирование под нагрузкой"},
		{keys: []string{"trade-off", "компромисс"}, label: "осознанный trade-off"},
		{keys: []string{"ux", "интерфейс", "пользоват"}, label: "пользовательский опыт"},
	}

	for _, f := range focuses {
		for _, key := range f.keys {
			if strings.Contains(v, key) {
				return f.label
			}
		}
	}

	return ""
}

func (h *Handler) validateInterviewerOutput(ctx context.Context, session *InterviewModuleSession, baseURL, question string) (string, error) {
	body := map[string]interface{}{
		"draft_response":  h.sanitizeModelText(question),
		"role":            session.Role,
		"current_topic":   session.CurrentTopic,
		"session_context": h.buildInterviewSessionContext(session, session.CurrentTopic, "", session.InterviewMode),
		"recent_topics":   h.recentAITopics(session, 5),
		"avoid_questions": h.collectAvoidQuestions(ctx, session, 30),
	}
	payload, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+"/api/v1/interviewer/validate-output", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("policy validator status %d", resp.StatusCode)
	}

	var result struct {
		IsValid           bool     `json:"is_valid"`
		Violations        []string `json:"violations"`
		SanitizedQuestion string   `json:"sanitized_question"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if !result.IsValid {
		if strings.TrimSpace(result.SanitizedQuestion) != "" {
			return h.sanitizeModelText(result.SanitizedQuestion), nil
		}
		return "", fmt.Errorf("policy validation blocked output")
	}

	if strings.TrimSpace(result.SanitizedQuestion) != "" {
		return h.sanitizeModelText(result.SanitizedQuestion), nil
	}

	return h.sanitizeModelText(question), nil
}

func (h *Handler) sanitizeCandidateText(content string) string {
	clean := strings.TrimSpace(content)
	clean = strings.ReplaceAll(clean, "\u0000", "")
	clean = strings.ReplaceAll(clean, "\r", " ")
	clean = strings.ReplaceAll(clean, "\n\n\n", "\n\n")
	if len(clean) > 4000 {
		clean = clean[:4000]
	}
	return clean
}

func (h *Handler) sanitizeModelText(content string) string {
	clean := strings.TrimSpace(content)
	clean = strings.ReplaceAll(clean, "\u0000", "")
	clean = strings.ReplaceAll(clean, "\r", " ")
	clean = regexp.MustCompile(`\s+`).ReplaceAllString(clean, " ")
	clean = strings.ReplaceAll(clean, "сетевой сети", "сети")
	clean = h.dedupeQuestionSentences(clean)
	if len(clean) > 2000 {
		clean = clean[:2000]
	}
	return clean
}

func (h *Handler) dedupeQuestionSentences(content string) string {
	parts := regexp.MustCompile(`[^.!?]+[.!?]?`).FindAllString(content, -1)
	if len(parts) == 0 {
		return content
	}

	normalize := regexp.MustCompile(`[\s\p{P}]+`)
	seen := make(map[string]struct{}, len(parts))
	kept := make([]string, 0, len(parts))

	for _, part := range parts {
		sentence := strings.TrimSpace(part)
		if sentence == "" {
			continue
		}
		key := strings.TrimSpace(strings.ToLower(normalize.ReplaceAllString(sentence, " ")))
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		kept = append(kept, sentence)
	}

	if len(kept) == 0 {
		return strings.TrimSpace(content)
	}

	return strings.Join(kept, " ")
}

func (h *Handler) questionMemoryKey(userID string, role string) string {
	return fmt.Sprintf("interview:asked_questions:%s:%s", strings.TrimSpace(userID), h.roleKey(role))
}

func (h *Handler) questionFingerprint(text string) string {
	normalized := strings.ToLower(strings.TrimSpace(text))
	normalized = regexp.MustCompile(`[^a-zа-я0-9\s]+`).ReplaceAllString(normalized, " ")
	normalized = regexp.MustCompile(`\s+`).ReplaceAllString(normalized, " ")
	return strings.TrimSpace(normalized)
}

func (h *Handler) collectAvoidQuestions(ctx context.Context, session *InterviewModuleSession, limit int) []string {
	if limit <= 0 {
		limit = 20
	}

	items := make([]string, 0, limit)
	seen := map[string]struct{}{}

	for i := len(session.Messages) - 1; i >= 0 && len(items) < limit; i-- {
		msg := session.Messages[i]
		if msg.Sender != "ai" {
			continue
		}
		fp := h.questionFingerprint(msg.Content)
		if fp == "" {
			continue
		}
		if _, exists := seen[fp]; exists {
			continue
		}
		seen[fp] = struct{}{}
		items = append(items, msg.Content)
	}

	if h.redis == nil || strings.TrimSpace(session.UserID) == "" {
		return items
	}

	key := h.questionMemoryKey(session.UserID, session.Role)
	recent, err := h.redis.LRange(ctx, key, 0, int64(limit*2)).Result()
	if err != nil {
		return items
	}

	for _, q := range recent {
		if len(items) >= limit {
			break
		}
		fp := h.questionFingerprint(q)
		if fp == "" {
			continue
		}
		if _, exists := seen[fp]; exists {
			continue
		}
		seen[fp] = struct{}{}
		items = append(items, q)
	}

	return items
}

func (h *Handler) isQuestionRepeated(ctx context.Context, session *InterviewModuleSession, question string) bool {
	if session == nil {
		return false
	}

	fp := h.questionFingerprint(question)
	if fp == "" {
		return false
	}

	for i := len(session.Messages) - 1; i >= 0; i-- {
		msg := session.Messages[i]
		if msg.Sender != "ai" {
			continue
		}
		if h.questionFingerprint(msg.Content) == fp {
			return true
		}
	}

	canCheckRedis := h.redis != nil && strings.TrimSpace(session.UserID) != ""
	if canCheckRedis {
		key := h.questionMemoryKey(session.UserID, session.Role)
		recent, err := h.redis.LRange(ctx, key, 0, 100).Result()
		if err == nil {
			for _, q := range recent {
				if h.questionFingerprint(q) == fp {
					return true
				}
			}
		}
	}

	if h.aiServiceURL != "" && canCheckRedis {
		return h.isSemanticDuplicate(ctx, session, question)
	}

	return false
}

func (h *Handler) isSemanticDuplicate(ctx context.Context, session *InterviewModuleSession, question string) bool {
	if session == nil || h.redis == nil || strings.TrimSpace(session.UserID) == "" || strings.TrimSpace(h.aiServiceURL) == "" {
		return false
	}

	embKey := h.questionMemoryKey(session.UserID, session.Role) + ":embed"
	recent, err := h.redis.LRange(ctx, embKey, 0, 40).Result()
	if err != nil || len(recent) == 0 {
		return false
	}

	currentEmbedding, err := h.getEmbeddingWithCache(ctx, strings.TrimSpace(question), 24*time.Hour)
	if err != nil || len(currentEmbedding) == 0 {
		return false
	}

	semanticThreshold := h.semanticThresholdForRole(session.Role)
	for _, raw := range recent {
		var stored struct {
			Q         string    `json:"q"`
			E         []float64 `json:"e"`
			Embedding []float64 `json:"embedding"`
		}
		if err := json.Unmarshal([]byte(raw), &stored); err != nil {
			continue
		}

		vec := stored.E
		if len(vec) == 0 {
			vec = stored.Embedding
		}
		if len(vec) == 0 {
			continue
		}

		sim := h.cosineSimilarity(currentEmbedding, vec)
		if sim > semanticThreshold {
			h.logger.Debugf("Semantic duplicate detected (sim=%.3f > %.2f) role=%s: %s vs %s", sim, semanticThreshold, session.Role, question, stored.Q)
			return true
		}
	}

	return false
}

func (h *Handler) cosineSimilarity(vec1, vec2 []float64) float64 {
	if len(vec1) == 0 || len(vec2) == 0 || len(vec1) != len(vec2) {
		return 0.0
	}

	var dot, norm1, norm2 float64
	for i := range vec1 {
		dot += vec1[i] * vec2[i]
		norm1 += vec1[i] * vec1[i]
		norm2 += vec2[i] * vec2[i]
	}

	if norm1 == 0 || norm2 == 0 {
		return 0.0
	}

	return dot / (math.Sqrt(norm1) * math.Sqrt(norm2))
}

func (h *Handler) rememberAskedQuestion(ctx context.Context, userID string, role string, question string) {
	if h.redis == nil {
		return
	}
	if strings.TrimSpace(userID) == "" {
		return
	}
	clean := h.sanitizeModelText(question)
	if clean == "" {
		return
	}

	key := h.questionMemoryKey(userID, role)
	pipe := h.redis.TxPipeline()
	pipe.LPush(ctx, key, clean)
	pipe.LTrim(ctx, key, 0, 199)
	pipe.Expire(ctx, key, 14*24*time.Hour)
	_, _ = pipe.Exec(ctx)

	go h.storeQuestionWithEmbedding(context.Background(), userID, role, clean)
}

func (h *Handler) storeQuestionWithEmbedding(ctx context.Context, userID string, role string, question string) {
	if h.redis == nil || strings.TrimSpace(h.aiServiceURL) == "" {
		return
	}

	embedding, err := h.getEmbeddingWithCache(ctx, question, 24*time.Hour)
	if err != nil || len(embedding) == 0 {
		return
	}

	stored := map[string]interface{}{
		"q": question,
		"e": embedding,
	}
	storedJSON, err := json.Marshal(stored)
	if err != nil {
		return
	}

	key := h.questionMemoryKey(userID, role) + ":embed"
	pipe := h.redis.TxPipeline()
	pipe.LPush(ctx, key, string(storedJSON))
	pipe.LTrim(ctx, key, 0, 199)
	pipe.Expire(ctx, key, 14*24*time.Hour)
	_, _ = pipe.Exec(ctx)
}

func (h *Handler) semanticThresholdForRole(role string) float64 {
	key := h.roleKey(role)
	thresholds := map[string]float64{
		"backend":  0.78,
		"frontend": 0.76,
		"devops":   0.77,
		"ml":       0.79,
		"data":     0.79,
		"mobile":   0.76,
	}
	if v, ok := thresholds[key]; ok {
		return v
	}
	return 0.77
}

func (h *Handler) embeddingCacheKey(text string) string {
	normalized := strings.ToLower(strings.TrimSpace(text))
	sum := sha256.Sum256([]byte(normalized))
	return "interview:embedding:cache:" + hex.EncodeToString(sum[:])
}

func (h *Handler) getEmbeddingWithCache(ctx context.Context, text string, ttl time.Duration) ([]float64, error) {
	if h.redis == nil || strings.TrimSpace(h.aiServiceURL) == "" {
		return nil, fmt.Errorf("embedding cache unavailable")
	}

	key := h.embeddingCacheKey(text)
	if cached, err := h.redis.Get(ctx, key).Result(); err == nil && strings.TrimSpace(cached) != "" {
		var emb []float64
		if json.Unmarshal([]byte(cached), &emb) == nil && len(emb) > 0 {
			return emb, nil
		}
	}

	body, err := json.Marshal(map[string]string{"question": text})
	if err != nil {
		return nil, err
	}

	url := strings.TrimRight(h.aiServiceURL, "/") + "/embeddings/question"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", uuid.New().String())

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embedding endpoint status=%d", resp.StatusCode)
	}

	var embData struct {
		Embedding []float64 `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&embData); err != nil || len(embData.Embedding) == 0 {
		return nil, fmt.Errorf("invalid embedding payload")
	}

	if payload, err := json.Marshal(embData.Embedding); err == nil {
		_ = h.redis.Set(ctx, key, payload, ttl).Err()
	}

	return embData.Embedding, nil
}

func (h *Handler) detectCandidateIntent(answer string) string {
	v := strings.ToLower(strings.TrimSpace(answer))
	if v == "" {
		return ""
	}
	if strings.Contains(v, "раскрой") || strings.Contains(v, "подробнее") || strings.Contains(v, "объяс") || strings.Contains(v, "не понял") {
		return "clarify"
	}
	if strings.Contains(v, "другой вопрос") || strings.Contains(v, "смени вопрос") || strings.Contains(v, "другая тема") {
		return "switch"
	}
	return ""
}

func (h *Handler) topicHint(role string, topic string) string {
	_ = role
	trimmedTopic := strings.TrimSpace(topic)
	if trimmedTopic != "" {
		return "сфокусируйтесь на теме " + trimmedTopic + ", рисках, компромиссах и проверке результата"
	}
	return "сфокусируйтесь на рисках, компромиссах и проверке результата"
}

func (h *Handler) topicPrompt(role string, topic string) string {
	_ = role
	trimmedTopic := strings.TrimSpace(topic)
	if trimmedTopic != "" {
		return "как вы подойдете к теме " + trimmedTopic + ", чтобы решение было проверяемым и устойчивым?"
	}
	return "какой практический подход вы выберете и почему?"
}

func (h *Handler) uncertaintyFallbackQuestion(session *InterviewModuleSession, topic string) string {
	_ = session
	question := "Ок, упростим: опишите базовое решение, два основных риска и как вы докажете, что всё работает корректно."

	if topic != "" {
		question = strings.TrimSuffix(question, ".") + " По теме " + topic + " ответьте конкретно, без общих слов."
	}

	return question
}

func (h *Handler) recordAuditEvent(ctx context.Context, sessionID uuid.UUID, eventType string, payload map[string]interface{}) {
	if payload == nil {
		payload = map[string]interface{}{}
	}

	audit := map[string]interface{}{
		"ts":         time.Now().UTC().Format(time.RFC3339Nano),
		"session_id": sessionID.String(),
		"event_type": eventType,
		"payload":    payload,
	}
	b, _ := json.Marshal(audit)

	if h.redis != nil {
		rawKey := "interview:audit:raw:" + sessionID.String()
		_ = h.redis.RPush(ctx, rawKey, string(b)).Err()
		_ = h.redis.Expire(ctx, rawKey, 72*time.Hour).Err()
	}

	h.logger.WithFields(logrus.Fields{
		"audit_event": eventType,
		"session_id":  sessionID.String(),
		"payload":     h.maskPII(string(b)),
	}).Info("audit trail")
}

func (h *Handler) initialDifficultyAndPressure(level string) (int, int) {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "senior", "staff", "principal":
		return 7, 3
	case "middle", "mid":
		return 5, 2
	default:
		return 4, 1
	}
}

func (h *Handler) maskPII(in string) string {
	if strings.TrimSpace(in) == "" {
		return in
	}
	emailRe := regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)
	phoneRe := regexp.MustCompile(`\+?[0-9][0-9\-\s]{7,}[0-9]`)
	out := emailRe.ReplaceAllString(in, "[masked-email]")
	out = phoneRe.ReplaceAllString(out, "[masked-phone]")
	return out
}

func (h *Handler) recordLatencySample(target *[]int64, value int64) {
	h.metricsMu.Lock()
	defer h.metricsMu.Unlock()
	*target = append(*target, value)
	if len(*target) > 256 {
		*target = (*target)[len(*target)-256:]
	}
}

func (h *Handler) p95(samples []int64) int64 {
	h.metricsMu.Lock()
	defer h.metricsMu.Unlock()
	if len(samples) == 0 {
		return 0
	}
	cpy := append([]int64(nil), samples...)
	for i := 0; i < len(cpy); i++ {
		for j := i + 1; j < len(cpy); j++ {
			if cpy[j] < cpy[i] {
				cpy[i], cpy[j] = cpy[j], cpy[i]
			}
		}
	}
	idx := int(float64(len(cpy)-1) * 0.95)
	return cpy[idx]
}

func (h *Handler) successRate(success int64, attempts int64) float64 {
	if attempts <= 0 {
		return 1.0
	}
	return float64(success) / float64(attempts)
}

func (h *Handler) buildIntroQuestion(session *InterviewModuleSession) *nextQuestionResponse {
	role := strings.TrimSpace(session.Role)
	level := strings.TrimSpace(session.Level)
	mode := strings.ToLower(strings.TrimSpace(session.InterviewMode))

	if mode != "theory" {
		question := h.buildPracticeTaskQuestion(session, "live_coding")

		return &nextQuestionResponse{
			Question:        question,
			Topic:           "live_coding",
			DifficultyDelta: 0,
			PressureLevel:   maxInt(1, session.PressureLevel),
			ShouldEnd:       false,
		}
	}

	out, err := h.requestInterviewQuestionFromAI(session, "skills_overview", "", "theory")
	if err == nil && out != nil && strings.TrimSpace(out.Question) != "" {
		out.Topic = "skills_overview"
		out.DifficultyDelta = 0
		out.PressureLevel = maxInt(1, out.PressureLevel)
		return out
	}

	question := fmt.Sprintf("Коротко опишите ваш опыт по роли %s (%s): ключевые навыки, инструменты, один сильный проект и ваш личный вклад.", role, level)
	if strings.TrimSpace(session.VacancyTitle) != "" {
		if strings.Contains(strings.ToLower(question), "опишите") || strings.Contains(strings.ToLower(question), "расскажите") {
			question = strings.TrimSuffix(question, "?") + fmt.Sprintf(" для вакансии %s?", session.VacancyTitle)
		}
	}

	return &nextQuestionResponse{
		Question:        question,
		Topic:           "skills_overview",
		DifficultyDelta: 0,
		PressureLevel:   maxInt(1, session.PressureLevel),
		ShouldEnd:       false,
	}
}

// buildTechnicalFallbackQuestion is invoked when the primary AI call
// in requestNextQuestion fails. Previously it returned a hardcoded
// "теперь перейдем к технической части..." line which violated the
// "all questions must come from AI" requirement.
//
// Behaviour now:
//   - Try the secondary AI endpoint one more time (different topic /
//     mode hint) — this still produces a real LLM-generated question.
//   - If that also fails, surface a transparent AI-unavailable
//     message into the chat. The candidate sees "AI временно
//     недоступен, нажмите 'следующий вопрос' через 5–10 сек" rather
//     than a generic canned interview question that distorts the
//     conversation.
func (h *Handler) buildTechnicalFallbackQuestion(session *InterviewModuleSession, lastAnswer string) *nextQuestionResponse {
	topic := strings.TrimSpace(session.CurrentTopic)
	if topic == "" || strings.EqualFold(topic, "skills_overview") {
		topic = h.nextTopic(session.Role, session.TopicCursor)
		session.TopicCursor++
	}

	mode := strings.ToLower(strings.TrimSpace(session.InterviewMode))
	if mode == "" {
		mode = "practice"
	}

	if out, err := h.requestInterviewQuestionFromAI(session, topic, lastAnswer, mode); err == nil && out != nil && strings.TrimSpace(out.Question) != "" {
		if mode != "theory" {
			out.Topic = "live_coding"
		} else {
			out.Topic = topic
		}
		out.DifficultyDelta = 0
		out.PressureLevel = maxInt(1, out.PressureLevel)
		return out
	}

	// No more fallbacks. Send a system message so the candidate
	// knows the AI is offline and can retry — better than fabricating
	// an interview question from a hardcoded list.
	return &nextQuestionResponse{
		Question:        "🤖 AI-интервьюер сейчас недоступен. Подождите 5–10 секунд и нажмите «следующий вопрос», чтобы продолжить — ваши ответы сохранены.",
		Topic:           topic,
		DifficultyDelta: 0,
		PressureLevel:   maxInt(1, session.PressureLevel),
		ShouldEnd:       false,
	}
}

// buildPracticeTaskQuestion ALWAYS asks the AI for the task. If the
// AI is unavailable we surface a transparent message rather than
// returning a deterministic hard-coded coding problem — see
// buildTechnicalFallbackQuestion for the same policy.
func (h *Handler) buildPracticeTaskQuestion(session *InterviewModuleSession, topic string) string {
	mode := strings.ToLower(strings.TrimSpace(session.InterviewMode))
	if mode == "" {
		mode = "practice"
	}
	out, err := h.requestInterviewQuestionFromAI(session, strings.TrimSpace(topic), "", mode)
	if err == nil && out != nil && strings.TrimSpace(out.Question) != "" {
		candidate := h.sanitizeModelText(out.Question)
		if !h.isWeakPracticeTask(candidate) {
			return candidate
		}
	}
	return "🤖 AI-сервис временно не отвечает. Через 5–10 секунд нажмите «следующая задача» — задание сгенерируется заново."
}

func (h *Handler) isWeakPracticeTask(question string) bool {
	v := strings.ToLower(strings.TrimSpace(question))
	if v == "" {
		return true
	}

	weakMarkers := []string{
		"по теме",
		"live_coding",
		"o(...)",
		"укажи сложность",
		"реализуй решение",
	}
	for _, marker := range weakMarkers {
		if strings.Contains(v, marker) {
			return true
		}
	}

	hasStructure := strings.Contains(v, "вход") || strings.Contains(v, "input")
	hasStructure = hasStructure && (strings.Contains(v, "выход") || strings.Contains(v, "output"))
	hasStructure = hasStructure && strings.Contains(v, "пример")

	return !hasStructure || len(v) < 140
}

func (h *Handler) buildDeterministicPracticeTask(session *InterviewModuleSession, topic string) string {
	role := strings.ToLower(strings.TrimSpace(session.Role))
	level := strings.ToLower(strings.TrimSpace(session.Level))
	topicName := strings.ToLower(strings.TrimSpace(topic))

	if strings.Contains(role, "frontend") {
		return "Задание: реализуйте функцию compressRanges(nums []int) []string.\n" +
			"Условие: дан отсортированный массив целых чисел без ограничений по знаку. Нужно сжать последовательные диапазоны в строки вида \"a-b\", одиночные значения оставлять как \"x\".\n" +
			"Вход: nums []int.\n" +
			"Выход: []string с диапазонами в исходном порядке.\n" +
			"Edge cases: пустой массив, дубликаты подряд, отрицательные числа, большие значения.\n" +
			"Пример: [-3,-2,-1,2,4,5,6] -> [\"-3--1\",\"2\",\"4-6\"].\n" +
			"Требование: озвучьте сложность по времени и памяти."
	}

	if strings.Contains(role, "backend") || strings.Contains(role, "go") || strings.Contains(role, "platform") || strings.Contains(topicName, "api") || strings.Contains(topicName, "concurrency") {
		return "Задание: реализуйте LRU-кеш с TTL.\n" +
			"Интерфейс: type Cache interface { Get(key string) (string, bool); Set(key, value string, ttlSeconds int); Delete(key string); Len() int }.\n" +
			"Условие: все операции должны быть потокобезопасны. Get должен возвращать false для отсутствующих или протухших значений.\n" +
			"Вход: последовательность команд Set/Get/Delete.\n" +
			"Выход: результаты Get и итоговая длина кеша.\n" +
			"Edge cases: ttlSeconds <= 0, повторная запись ключа, одновременные чтения/записи, capacity pressure.\n" +
			"Пример: Set(a,1,1) -> sleep 2s -> Get(a) == miss.\n" +
			"Требование: назовите сложность операций и как предотвращается race condition."
	}

	if level == "senior" || level == "staff" || level == "principal" {
		return "Задание: реализуйте функцию mergeIntervals(intervals [][]int) [][]int.\n" +
			"Условие: каждый интервал задан парой [start,end], где start <= end. Нужно объединить пересекающиеся интервалы.\n" +
			"Вход: [][]int.\n" +
			"Выход: [][]int без пересечений, отсортированный по start.\n" +
			"Edge cases: пустой вход, вложенные интервалы, одинаковые границы, отрицательные значения.\n" +
			"Пример: [[1,3],[2,6],[8,10],[15,18]] -> [[1,6],[8,10],[15,18]].\n" +
			"Требование: укажите, почему итоговая сложность O(n log n)."
	}

	return "Задание: реализуйте функцию twoSum(nums []int, target int) []int.\n" +
		"Условие: вернуть индексы двух различных элементов, сумма которых равна target. Если решения нет, вернуть пустой массив.\n" +
		"Вход: nums []int, target int.\n" +
		"Выход: []int длины 2 или пустой массив.\n" +
		"Edge cases: пустой массив, дубликаты, отрицательные числа, несколько валидных пар.\n" +
		"Пример: nums=[2,7,11,15], target=9 -> [0,1].\n" +
		"Требование: озвучьте сложность O(n) и используемую память."
}

func (h *Handler) buildInterviewSessionContext(session *InterviewModuleSession, topic string, lastAnswer string, mode string) string {
	if session == nil {
		return ""
	}

	trimmedMode := strings.ToLower(strings.TrimSpace(mode))
	if trimmedMode == "" {
		trimmedMode = strings.ToLower(strings.TrimSpace(session.InterviewMode))
	}
	if trimmedMode == "" {
		trimmedMode = "practice"
	}

	timeLeft := time.Until(session.ExpiresAt)
	if timeLeft < 0 {
		timeLeft = 0
	}

	questionsLeft := session.QuestionLimit - h.countAIMessages(session.Messages)
	if questionsLeft < 0 {
		questionsLeft = 0
	}

	parts := []string{
		"роль: " + strings.TrimSpace(session.Role),
		"уровень: " + strings.TrimSpace(session.Level),
		"режим: " + trimmedMode,
		"текущая тема: " + strings.TrimSpace(topic),
		fmt.Sprintf("сложность: %d/10", session.Difficulty),
		fmt.Sprintf("давление: %d/5", session.PressureLevel),
		fmt.Sprintf("осталось времени: %d сек", int(timeLeft/time.Second)),
		fmt.Sprintf("осталось вопросов: %d", questionsLeft),
	}

	if v := strings.TrimSpace(session.VacancyTitle); v != "" {
		parts = append(parts, "вакансия: "+v)
	}
	if v := strings.TrimSpace(session.VacancyCategory); v != "" {
		parts = append(parts, "категория вакансии: "+v)
	}
	if len(session.FocusAreas) > 0 {
		parts = append(parts, "фокус-области: "+strings.Join(session.FocusAreas, ", "))
	}
	if len(session.PrimarySkills) > 0 {
		parts = append(parts, "ключевые навыки: "+strings.Join(session.PrimarySkills, ", "))
	}
	if len(session.TheoryFocus) > 0 {
		parts = append(parts, "теоретический фокус: "+strings.Join(session.TheoryFocus, ", "))
	}
	if len(session.PracticeFocus) > 0 {
		parts = append(parts, "практический фокус: "+strings.Join(session.PracticeFocus, ", "))
	}

	answer := strings.TrimSpace(h.sanitizeCandidateText(lastAnswer))
	if answer != "" {
		if len(answer) > 500 {
			answer = answer[:500]
		}
		parts = append(parts, "последний ответ: "+answer)
	}

	if len(session.Messages) > 0 {
		asked := h.countAIMessages(session.Messages)
		parts = append(parts, fmt.Sprintf("уже задано AI-вопросов: %d", asked))
	}

	recentTopics := h.recentAITopics(session, 5)
	if len(recentTopics) > 0 {
		parts = append(parts, "последние темы: "+strings.Join(recentTopics, ", "))
	}

	return strings.Join(parts, "\n")
}

func (h *Handler) recentAITopics(session *InterviewModuleSession, limit int) []string {
	if session == nil || limit <= 0 {
		return nil
	}

	topics := make([]string, 0, limit)
	seen := make(map[string]struct{}, limit)
	for i := len(session.Messages) - 1; i >= 0 && len(topics) < limit; i-- {
		msg := session.Messages[i]
		if msg.Sender != "ai" {
			continue
		}
		topic := strings.TrimSpace(msg.Topic)
		if topic == "" {
			topic = strings.TrimSpace(h.extractCurrentPracticeTask(session))
		}
		if topic == "" {
			continue
		}
		key := strings.ToLower(topic)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		topics = append(topics, topic)
	}
	return topics
}

func (h *Handler) isPracticeTaskQuestion(question string) bool {
	v := strings.ToLower(strings.TrimSpace(question))
	if v == "" {
		return false
	}

	markers := []string{
		"напиши",
		"реализ",
		"функц",
		"класс",
		"алгорит",
		"live coding",
		"код",
		"решение",
		"edge case",
	}
	for _, marker := range markers {
		if strings.Contains(v, marker) {
			return true
		}
	}
	return false
}

func (h *Handler) composePracticeTurn(feedback string, errors string, hint string, task string) string {
	cleanFeedback := h.sanitizeModelText(feedback)
	cleanErrors := h.sanitizeModelText(errors)
	cleanHint := h.sanitizeModelText(hint)
	cleanTask := h.sanitizeModelText(task)

	if strings.TrimSpace(cleanFeedback) == "" {
		cleanFeedback = "Проверка выполнена. Продолжаем."
	}
	if strings.TrimSpace(cleanTask) == "" {
		cleanTask = "Напишите функцию solve(input), обработайте edge cases и укажите сложность."
	}
	if strings.TrimSpace(cleanHint) == "" {
		cleanHint = "Проверьте edge cases и корректность на пустом/предельном вводе."
	}
	if strings.TrimSpace(cleanErrors) == "" {
		cleanErrors = "Нет критичных ошибок, но есть зоны улучшения по проверкам и устойчивости."
	}

	return strings.TrimSpace(
		"[FEEDBACK]\n" + cleanFeedback +
			"\n\n[ERRORS]\n" + cleanErrors +
			"\n\n[HINT]\n" + cleanHint +
			"\n\n[NEXT_TASK]\n" + cleanTask,
	)
}

func (h *Handler) detectPracticeControlIntent(answer string) string {
	v := strings.ToLower(strings.TrimSpace(answer))
	if strings.Contains(v, "[hint_request]") {
		return "hint"
	}
	if strings.Contains(v, "[test_case_request]") {
		return "tests"
	}
	return ""
}

func (h *Handler) buildPracticeControlResponse(session *InterviewModuleSession, intent string) *nextQuestionResponse {
	currentTask := h.extractCurrentPracticeTask(session)
	if strings.TrimSpace(currentTask) == "" {
		currentTask = h.buildPracticeTaskQuestion(session, "live_coding")
	}

	feedback := "Продолжаем текущую задачу."
	errors := "Код еще не проверялся: сначала отправьте попытку решения."
	hint := ""

	if intent == "hint" {
		feedback = "Подсказка к текущему заданию."
		hint = "Декомпозируйте решение: входные проверки, основная логика, проверка инвариантов. Сначала пройдите базовый кейс вручную."
	}
	if intent == "tests" {
		feedback = "Рекомендуемые тест-кейсы для самопроверки."
		hint = "Минимальный набор: empty input, one-item input, duplicate items, maximal boundary values, random stress sample."
	}

	return &nextQuestionResponse{
		Question:        h.composePracticeTurn(feedback, errors, hint, currentTask),
		Topic:           "live_coding",
		DifficultyDelta: 0,
		PressureLevel:   maxInt(1, session.PressureLevel),
		ShouldEnd:       false,
	}
}

func (h *Handler) extractCurrentPracticeTask(session *InterviewModuleSession) string {
	for i := len(session.Messages) - 1; i >= 0; i-- {
		msg := session.Messages[i]
		if msg.Sender != "ai" {
			continue
		}
		text := msg.Content
		idx := strings.Index(text, "[NEXT_TASK]")
		if idx >= 0 {
			task := strings.TrimSpace(text[idx+len("[NEXT_TASK]"):])
			if task != "" {
				return task
			}
		}
		if h.isPracticeTaskQuestion(text) {
			return strings.TrimSpace(text)
		}
	}
	return ""
}

func (h *Handler) roleKey(role string) string {
	value := strings.ToLower(strings.TrimSpace(role))
	switch {
	case strings.Contains(value, "web") || strings.Contains(value, "browser") || strings.Contains(value, "fullstack") || strings.Contains(value, "full stack"):
		return "web"
	case strings.Contains(value, "backend"):
		return "backend"
	case strings.Contains(value, "frontend"):
		return "frontend"
	case strings.Contains(value, "devops") || strings.Contains(value, "sre") || strings.Contains(value, "platform"):
		return "devops"
	case strings.Contains(value, "ml") || strings.Contains(value, "machine learning") || strings.Contains(value, "data science"):
		return "ml"
	case strings.Contains(value, "data"):
		return "data"
	case strings.Contains(value, "mobile") || strings.Contains(value, "android") || strings.Contains(value, "ios"):
		return "mobile"
	case strings.Contains(value, "game") || strings.Contains(value, "gaming") || strings.Contains(value, "unity"):
		return "game"
	case strings.Contains(value, "security") || strings.Contains(value, "cyber") || strings.Contains(value, "pentest") || strings.Contains(value, "soc") || strings.Contains(value, "reverse"):
		return "security"
	case strings.Contains(value, "system") || strings.Contains(value, "desktop") || strings.Contains(value, "embedded") || strings.Contains(value, "kernel") || strings.Contains(value, "compiler"):
		return "systems"
	case strings.Contains(value, "enterprise") || strings.Contains(value, "sap") || strings.Contains(value, "salesforce") || strings.Contains(value, "qa") || strings.Contains(value, "sdet") || strings.Contains(value, "low-code") || strings.Contains(value, "low code"):
		return "enterprise"
	case strings.Contains(value, "fintech") || strings.Contains(value, "trading") || strings.Contains(value, "hft") || strings.Contains(value, "quant") || strings.Contains(value, "blockchain"):
		return "fintech"
	case strings.Contains(value, "iot") || strings.Contains(value, "robot") || strings.Contains(value, "firmware"):
		return "iot"
	case strings.Contains(value, "manager") || strings.Contains(value, "product") || strings.Contains(value, "project") || strings.Contains(value, "analyst") || strings.Contains(value, "writer") || strings.Contains(value, "devrel") || strings.Contains(value, "scrum"):
		return "management"
	default:
		return value
	}
}

func (h *Handler) isUncertainAnswer(answer string) bool {
	v := strings.ToLower(strings.TrimSpace(answer))
	if v == "" {
		return false
	}
	markers := []string{"не знаю", "не уверен", "затрудняюсь", "без понятия", "i don't know", "dont know", "not sure"}
	for _, m := range markers {
		if strings.Contains(v, m) {
			return true
		}
	}
	return false
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
