"""API integration tests."""

from unittest.mock import AsyncMock, MagicMock, patch

import pytest
from fastapi.testclient import TestClient

from src.main import create_app


@pytest.fixture
def client():
    """Create a test client with mocked dependencies."""
    with patch("src.config.get_settings") as mock_settings:
        mock_settings.return_value = MagicMock(
            app_name="ai-service-test",
            app_version="1.0.0-test",
            debug=True,
            host="127.0.0.1",
            port=8001,
            log_level="DEBUG",
            llm_api_key="test-key",
            llm_model="gpt-4o-mini",
            llm_temperature=0.7,
            llm_max_tokens=2048,
            llm_base_url=None,
            embedding_model="text-embedding-3-small",
            embedding_dimensions=1536,
            transcription_model="whisper-1",
            transcription_max_file_size_mb=25,
            rate_limit_per_minute=60,
            cors_origins=["http://localhost:3000"],
        )
        app = create_app()
        with TestClient(app) as test_client:
            yield test_client


class TestHealthEndpoint:
    """Tests for the health check endpoint."""

    def test_health_check_returns_200(self, client):
        response = client.get("/health")
        assert response.status_code == 200
        data = response.json()
        assert data["status"] == "healthy"

    def test_health_check_returns_version(self, client):
        response = client.get("/health")
        data = response.json()
        assert "version" in data


class TestQuestionGenerationEndpoint:
    """Tests for the question generation endpoint."""

    @patch("src.services.question_service.QuestionService.generate_questions")
    def test_generate_questions_success(self, mock_generate, client):
        from src.models.responses import QuestionGenerationResponse, QuestionItem

        mock_generate.return_value = QuestionGenerationResponse(
            questions=[
                QuestionItem(
                    text="What is Python?",
                    type="open_ended",
                    difficulty="easy",
                    expected_answer="Python is a programming language",
                    explanation="Python is a high-level programming language",
                )
            ],
            topic="Python basics",
            total_count=1,
        )

        response = client.post(
            "/api/v1/questions/generate",
            json={
                "topic": "Python basics",
                "num_questions": 1,
                "difficulty": "easy",
            },
        )

        assert response.status_code == 200
        data = response.json()
        assert data["topic"] == "Python basics"
        assert len(data["questions"]) == 1

    def test_generate_questions_missing_topic(self, client):
        response = client.post(
            "/api/v1/questions/generate",
            json={"num_questions": 5},
        )
        assert response.status_code == 422

    def test_generate_questions_invalid_difficulty(self, client):
        response = client.post(
            "/api/v1/questions/generate",
            json={"topic": "Python", "difficulty": "extreme"},
        )
        assert response.status_code == 422

    def test_generate_questions_num_questions_out_of_range(self, client):
        response = client.post(
            "/api/v1/questions/generate",
            json={"topic": "Python", "num_questions": 100},
        )
        assert response.status_code == 422


class TestAnalysisEndpoint:
    """Tests for the answer analysis endpoint."""

    @patch("src.services.analysis_service.AnalysisService.analyze_answer")
    def test_analyze_answer_success(self, mock_analyze, client):
        from src.models.responses import AnalysisResponse, AnalysisScores

        mock_analyze.return_value = AnalysisResponse(
            scores=AnalysisScores(
                correctness=85,
                completeness=70,
                clarity=90,
                relevance=95,
            ),
            overall_score=85,
            feedback="Good answer with room for improvement",
            strengths=["Clear explanation"],
            weaknesses=["Missing key details"],
            suggested_improvements=["Add more examples"],
        )

        response = client.post(
            "/api/v1/analysis/answer",
            json={
                "question": "What is OOP?",
                "answer": "Object-oriented programming uses classes",
                "expected_answer": "OOP is a paradigm based on objects and classes",
            },
        )

        assert response.status_code == 200
        data = response.json()
        assert "scores" in data
        assert "overall_score" in data

    def test_analyze_answer_missing_fields(self, client):
        response = client.post(
            "/api/v1/analysis/answer",
            json={"question": "What is OOP?"},
        )
        assert response.status_code == 422

    @patch("src.services.analysis_service.AnalysisService.compute_similarity")
    def test_compute_similarity_success(self, mock_similarity, client):
        mock_similarity.return_value = 0.85

        response = client.post(
            "/api/v1/analysis/similarity",
            json={
                "text_a": "Python is a programming language",
                "text_b": "Python is a high-level language",
            },
        )

        assert response.status_code == 200
        data = response.json()
        assert "similarity" in data


class TestTranscriptionEndpoint:
    """Tests for the transcription endpoint."""

    def test_transcribe_invalid_file_type(self, client):
        response = client.post(
            "/api/v1/transcription",
            files={"file": ("test.txt", b"not audio", "text/plain")},
        )
        assert response.status_code == 400


class TestInterviewerEndpoints:
    """Tests for interviewer next-question and post-analysis endpoints."""

    @patch("src.core.llm_client.LLMClient.generate_json", new_callable=AsyncMock)
    def test_next_question_uses_fallback_for_weak_answer(self, mock_generate_json, client):
        mock_generate_json.side_effect = RuntimeError("llm offline")

        response = client.post(
            "/api/v1/interviewer/next-question",
            json={
                "role": "Backend Engineer",
                "level": "middle",
                "current_topic": "caching",
                "difficulty": 6,
                "pressure_level": 2,
                "time_left_sec": 900,
                "questions_left": 5,
                "last_candidate_answer": "не знаю, затрудняюсь",
                "history": [
                    {"sender": "ai", "content": "Объясните cache invalidation"},
                    {"sender": "user", "content": "не знаю"},
                ],
                "recent_topics": ["caching", "database"],
                "avoid_questions": ["Что такое кэш?"],
            },
        )

        assert response.status_code == 200
        data = response.json()
        assert data["flags"]["answer_was_weak"] is True
        assert data["flags"]["answer_was_partial"] is False
        assert "Ответ пока слабый" in data["question"]

    @patch("src.services.analysis_service.AnalysisService.compute_similarity", new_callable=AsyncMock)
    def test_post_analysis_highlights_weak_dialogue_patterns(self, mock_similarity, client):
        mock_similarity.return_value = 0.08

        response = client.post(
            "/api/v1/interviewer/post-analysis",
            json={
                "session_id": "sess-weak-1",
                "role": "Backend Engineer",
                "level": "middle",
                "messages": [
                    {"sender": "ai", "content": "Как обеспечите надежность сервиса?"},
                    {"sender": "user", "content": "не знаю"},
                    {"sender": "ai", "content": "Какие метрики выберете?"},
                    {"sender": "user", "content": "кажется, p95"},
                    {"sender": "ai", "content": "Как будете валидировать решение?"},
                    {"sender": "user", "content": "примерно тестами"},
                ],
            },
        )

        assert response.status_code == 200
        data = response.json()
        assert data["overall_score"] < 60
        assert "Слишком много коротких ответов без объяснения reasoning" in data["weaknesses"]
        assert "Есть уклончивые ответы вместо прямого решения" in data["weaknesses"]
        assert "Если не уверены, озвучивайте гипотезу и план проверки, а не уходите в общие фразы" in data[
            "recommendations"
        ]

    def test_post_analysis_no_messages_returns_zero_scores(self, client):
        response = client.post(
            "/api/v1/interviewer/post-analysis",
            json={
                "session_id": "sess-empty-1",
                "role": "Backend Engineer",
                "level": "middle",
                "messages": [],
            },
        )

        assert response.status_code == 200
        data = response.json()
        assert data["overall_score"] == 0
        assert data["weaknesses"] == ["Нет ответов кандидата для анализа"]

    @patch("src.api.routes._score_interview_turn", new_callable=AsyncMock)
    def test_post_analysis_strong_dialogue_adds_advanced_recommendation(self, mock_score_turn, client):
        mock_score_turn.return_value = {
            "correctness": 88.0,
            "clarity": 86.0,
            "completeness": 84.0,
            "relevance": 89.0,
        }

        response = client.post(
            "/api/v1/interviewer/post-analysis",
            json={
                "session_id": "sess-strong-1",
                "role": "Backend Engineer",
                "level": "senior",
                "messages": [
                    {"sender": "ai", "content": "Как обеспечите отказоустойчивость API?"},
                    {
                        "sender": "user",
                        "content": (
                            "Сначала выделю критичные failure modes, затем введу retry policy с jitter, "
                            "идемпотентность и circuit breaker, после чего проверю p95/p99 и стратегию rollback."
                        ),
                    },
                    {"sender": "ai", "content": "Как проверите решение на нагрузке?"},
                    {
                        "sender": "user",
                        "content": (
                            "Построю нагрузочный сценарий с частичными отказами, сравню baseline и новую версию, "
                            "зафиксирую trade-offs по latency и error rate, затем валидирую SLO и план отката."
                        ),
                    },
                    {"sender": "ai", "content": "Какие метрики и алерты поставите?"},
                    {
                        "sender": "user",
                        "content": (
                            "Вынесу p95/p99, saturation, error budget burn и алерты по деградации, "
                            "а также контрольные проверки после релиза с явным критерием успешности."
                        ),
                    },
                ],
            },
        )

        assert response.status_code == 200
        data = response.json()
        assert data["overall_score"] >= 75
        assert "Ответы остаются в фокусе вопроса" in data["strengths"]
        assert "Добавьте продвинутый уровень: обсуждайте failure modes, метрики и стратегию отката" in data[
            "recommendations"
        ]
