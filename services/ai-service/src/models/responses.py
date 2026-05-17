"""Pydantic models for API responses."""

from typing import Literal, Optional

from pydantic import BaseModel, Field


class QuestionItem(BaseModel):
    """A single generated question."""

    text: str
    type: str  # multiple_choice, open_ended, true_false, short_answer
    difficulty: str  # easy, medium, hard
    expected_answer: str
    options: Optional[list[str]] = None
    explanation: Optional[str] = None


class QuestionGenerationResponse(BaseModel):
    """Response model for question generation."""

    questions: list[QuestionItem]
    topic: str
    total_count: int


class AnalysisScores(BaseModel):
    """Individual analysis scores."""

    correctness: float = Field(..., ge=0, le=100)
    completeness: float = Field(..., ge=0, le=100)
    clarity: float = Field(..., ge=0, le=100)
    relevance: float = Field(..., ge=0, le=100)


class AnalysisResponse(BaseModel):
    """Response model for answer analysis."""

    scores: AnalysisScores
    overall_score: float = Field(..., ge=0, le=100)
    feedback: str
    strengths: list[str] = Field(default_factory=list)
    weaknesses: list[str] = Field(default_factory=list)
    suggested_improvements: list[str] = Field(default_factory=list)


class TranscriptionResponse(BaseModel):
    """Response model for audio transcription."""

    text: str
    language: Optional[str] = None
    duration_seconds: Optional[float] = None
    confidence: Optional[float] = None


class ErrorResponse(BaseModel):
    """Standard error response model."""

    error: str
    detail: Optional[str] = None


class HealthResponse(BaseModel):
    """Health check response."""

    status: str
    version: Optional[str] = None


class NextQuestionResponse(BaseModel):
    """Response for strict interviewer next question."""

    question: str
    topic: str
    difficulty_delta: int = Field(default=0, ge=-2, le=2)
    pressure_level: int = Field(default=1, ge=1, le=5)
    should_end: bool = False
    flags: dict[str, bool] = Field(default_factory=dict)
    # Verdict on the candidate's LAST answer. Drives the inline
    # ✅/⚠️/❌ badge in the chat and the final-report aggregation:
    #   correct  — answer is technically right and complete enough
    #   partial  — partially correct, important pieces missing
    #   wrong    — factually incorrect
    #   skipped  — candidate said "не знаю" / "пропустить" / no answer
    #   off_topic — answer didn't address the question at all
    #   none     — no prior answer to grade (first turn of session)
    last_answer_verdict: Optional[
        Literal["correct", "partial", "wrong", "skipped", "off_topic", "none"]
    ] = None
    # One-line reason that the badge tooltip can show.
    last_answer_reason: Optional[str] = None


class ValidateOutputResponse(BaseModel):
    """Response for interviewer output policy validation."""

    is_valid: bool
    violations: list[str] = Field(default_factory=list)
    sanitized_question: str


class PostAnalysisResponse(BaseModel):
    """Response with post-interview quality metrics."""

    session_id: str
    correctness: float = Field(..., ge=0, le=100)
    clarity: float = Field(..., ge=0, le=100)
    completeness: float = Field(..., ge=0, le=100)
    relevance: float = Field(..., ge=0, le=100)
    overall_score: float = Field(..., ge=0, le=100)
    strengths: list[str] = Field(default_factory=list)
    weaknesses: list[str] = Field(default_factory=list)
    recommendations: list[str] = Field(default_factory=list)


class DeveloperRoleRecommendation(BaseModel):
    """Single recommended interview role based on GitHub profile."""

    role: str
    fit_score: int = Field(..., ge=0, le=100)
    rationale: str


class DeveloperLanguageInsight(BaseModel):
    """Language-specific insights with interview focus hints."""

    language: str
    confidence: int = Field(..., ge=0, le=100)
    evidence: str
    interview_topics: list[str] = Field(default_factory=list)


class DeveloperInterviewTrack(BaseModel):
    """Recommended interview setup based on profile analytics."""

    role: str
    mode: str
    level: str
    duration_minutes: int = Field(default=30, ge=10, le=120)
    focus_areas: list[str] = Field(default_factory=list)
    primary_skills: list[str] = Field(default_factory=list)
    rationale: str


class DeveloperInsightsResponse(BaseModel):
    """AI-generated developer profile insights and role recommendations."""

    summary: str
    strengths: list[str] = Field(default_factory=list)
    risks: list[str] = Field(default_factory=list)
    action_plan: list[str] = Field(default_factory=list)
    language_insights: list[DeveloperLanguageInsight] = Field(default_factory=list)
    interview_tracks: list[DeveloperInterviewTrack] = Field(default_factory=list)
    recommended_positions: list[DeveloperRoleRecommendation] = Field(default_factory=list)


class ResumeInsightsResponse(BaseModel):
    """AI-generated resume insights and interview recommendations."""

    summary: str
    strong_points: list[str] = Field(default_factory=list)
    improvement_points: list[str] = Field(default_factory=list)
    action_plan: list[str] = Field(default_factory=list)
    language_insights: list[DeveloperLanguageInsight] = Field(default_factory=list)
    interview_tracks: list[DeveloperInterviewTrack] = Field(default_factory=list)
    recommended_positions: list[DeveloperRoleRecommendation] = Field(default_factory=list)
