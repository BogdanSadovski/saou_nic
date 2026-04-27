"""Pydantic models for request schemas."""

from typing import Optional

from pydantic import BaseModel, Field, field_validator


class QuestionGenerationRequest(BaseModel):
    """Request model for question generation."""

    topic: str = Field(..., min_length=1, max_length=500, description="Topic for question generation")
    context: Optional[str] = Field(None, max_length=5000, description="Additional context")
    num_questions: int = Field(default=5, ge=1, le=50, description="Number of questions to generate")
    difficulty: str = Field(default="medium", pattern="^(easy|medium|hard)$")
    question_types: Optional[list[str]] = Field(
        default=None,
        description="Types of questions: multiple_choice, open_ended, true_false, short_answer",
    )

    @field_validator("question_types")
    @classmethod
    def validate_question_types(cls, v: Optional[list[str]]) -> Optional[list[str]]:
        valid_types = {"multiple_choice", "open_ended", "true_false", "short_answer"}
        if v is not None:
            for qt in v:
                if qt not in valid_types:
                    raise ValueError(f"Invalid question type: {qt}. Must be one of {valid_types}")
        return v


class AnalysisRequest(BaseModel):
    """Request model for answer analysis."""

    question: str = Field(..., min_length=1, max_length=2000)
    answer: str = Field(..., min_length=1, max_length=10000)
    expected_answer: str = Field(..., min_length=1, max_length=10000)
    rubric: Optional[str] = Field(None, max_length=2000, description="Custom grading rubric")


class SimilarityRequest(BaseModel):
    """Request model for similarity computation."""

    text_a: str = Field(..., min_length=1, max_length=10000)
    text_b: str = Field(..., min_length=1, max_length=10000)


class TranscriptionRequest(BaseModel):
    """Metadata for transcription requests."""

    language: Optional[str] = Field(None, max_length=10, description="Language code")


class InterviewHistoryMessage(BaseModel):
    """Single message in interview conversation history."""

    sender: str = Field(..., pattern="^(ai|user|system)$")
    content: str = Field(..., min_length=1, max_length=6000)
    topic: Optional[str] = Field(None, max_length=120)
    difficulty: Optional[int] = Field(None, ge=1, le=10)


class NextQuestionRequest(BaseModel):
    """Request schema for strict interviewer next-question generation."""

    role: str = Field(..., min_length=1, max_length=120)
    level: str = Field(..., min_length=1, max_length=40)
    vacancy_title: Optional[str] = Field(None, max_length=200)
    vacancy_category: Optional[str] = Field(None, max_length=120)
    interview_mode: Optional[str] = Field(None, max_length=40)
    session_context: Optional[str] = Field(None, max_length=5000)
    recent_topics: Optional[list[str]] = Field(default_factory=list)
    focus_areas: Optional[list[str]] = Field(default_factory=list)
    primary_skills: Optional[list[str]] = Field(default_factory=list)
    theory_focus: Optional[list[str]] = Field(default_factory=list)
    practice_focus: Optional[list[str]] = Field(default_factory=list)
    current_topic: Optional[str] = Field(None, max_length=120)
    difficulty: int = Field(default=5, ge=1, le=10)
    pressure_level: int = Field(default=1, ge=1, le=5)
    time_left_sec: int = Field(default=1800, ge=0, le=10800)
    questions_left: int = Field(default=10, ge=0, le=100)
    last_candidate_answer: Optional[str] = Field(None, max_length=10000)
    history: Optional[list[InterviewHistoryMessage]] = Field(default_factory=list)
    avoid_questions: Optional[list[str]] = Field(default_factory=list)
    turn_nonce: Optional[str] = Field(None, max_length=200)


class ValidateOutputRequest(BaseModel):
    """Request schema for interviewer output policy validation."""

    draft_response: str = Field(..., min_length=1, max_length=8000)
    role: Optional[str] = Field(None, max_length=120)
    current_topic: Optional[str] = Field(None, max_length=120)
    session_context: Optional[str] = Field(None, max_length=5000)
    recent_topics: Optional[list[str]] = Field(default_factory=list)
    avoid_questions: Optional[list[str]] = Field(default_factory=list)


class PostAnalysisRequest(BaseModel):
    """Request schema for post-interview analysis."""

    session_id: str = Field(..., min_length=1, max_length=100)
    role: str = Field(..., min_length=1, max_length=120)
    level: str = Field(..., min_length=1, max_length=40)
    messages: list[InterviewHistoryMessage] = Field(default_factory=list)


class DeveloperRepoSnapshot(BaseModel):
    """Repository snapshot for developer profile insights."""

    name: str = Field(..., min_length=1, max_length=200)
    language: Optional[str] = Field(None, max_length=80)
    stars: int = Field(default=0, ge=0, le=1000000)
    forks: int = Field(default=0, ge=0, le=1000000)
    open_issues: int = Field(default=0, ge=0, le=1000000)
    last_push: Optional[str] = Field(None, max_length=64)


class DeveloperChartPoint(BaseModel):
    """Simple chart point used for language and activity charts."""

    label: str = Field(..., min_length=1, max_length=120)
    value: int = Field(..., ge=0, le=100000000)


class DeveloperInsightsRequest(BaseModel):
    """Request schema for GitHub developer profile insights."""

    github_username: str = Field(..., min_length=1, max_length=80)
    profile_name: Optional[str] = Field(None, max_length=120)
    bio: Optional[str] = Field(None, max_length=2000)
    role_preferences: Optional[list[str]] = Field(default_factory=list)
    followers: int = Field(default=0, ge=0, le=100000000)
    following: int = Field(default=0, ge=0, le=100000000)
    public_repos: int = Field(default=0, ge=0, le=100000000)
    sampled_repos: int = Field(default=0, ge=0, le=1000000)
    total_stars: int = Field(default=0, ge=0, le=100000000)
    total_forks: int = Field(default=0, ge=0, le=100000000)
    total_open_issues: int = Field(default=0, ge=0, le=100000000)
    language_distribution: Optional[list[DeveloperChartPoint]] = Field(default_factory=list)
    monthly_activity: Optional[list[DeveloperChartPoint]] = Field(default_factory=list)
    top_repositories: Optional[list[DeveloperRepoSnapshot]] = Field(default_factory=list)


class ResumeInsightsRequest(BaseModel):
    """Request schema for resume file analysis insights."""

    file_name: str = Field(..., min_length=1, max_length=260)
    content_type: Optional[str] = Field(None, max_length=120)
    role_preferences: Optional[list[str]] = Field(default_factory=list)
    word_count: int = Field(default=0, ge=0, le=1000000)
    character_count: int = Field(default=0, ge=0, le=5000000)
    skills: Optional[list[str]] = Field(default_factory=list)
    languages: Optional[list[str]] = Field(default_factory=list)
    experience_entries: int = Field(default=0, ge=0, le=10000)
    education_entries: int = Field(default=0, ge=0, le=10000)
    text_excerpt: Optional[str] = Field(None, max_length=8000)
