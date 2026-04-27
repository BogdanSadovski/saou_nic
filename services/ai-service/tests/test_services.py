"""Service layer unit tests."""

from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from src.core.embeddings import EmbeddingService
from src.core.llm_client import LLMClient
from src.core.prompt_templates import PromptTemplate, PromptTemplateService
from src.services.analysis_service import AnalysisService
from src.services.question_service import QuestionService
from src.services.transcription_service import TranscriptionService
from src.utils.helpers import (
    compute_text_hash,
    format_duration,
    sanitize_text,
    truncate_text,
)
from src.utils.validators import (
    ValidationError,
    validate_difficulty,
    validate_language_code,
    validate_question_count,
    validate_text_length,
    validate_topic,
)


# ─── LLM Client Tests ───────────────────────────────────────────────


class TestLLMClient:
    """Tests for LLMClient."""

    @patch("src.core.llm_client.AsyncOpenAI")
    def test_generate_calls_api(self, mock_openai):
        mock_response = MagicMock()
        mock_response.choices = [MagicMock()]
        mock_response.choices[0].message.content = "Test response"
        mock_response.usage = MagicMock(total_tokens=50)
        mock_openai.return_value.chat.completions.create = AsyncMock(
            return_value=mock_response
        )

        client = LLMClient(api_key="test-key", model="gpt-4o-mini")

        import asyncio

        result = asyncio.get_event_loop().run_until_complete(
            client.generate(prompt="Hello")
        )
        assert result == "Test response"


# ─── Prompt Template Tests ──────────────────────────────────────────


class TestPromptTemplates:
    """Tests for prompt template service."""

    def test_format_template(self):
        template = PromptTemplate(
            name="test",
            template="Hello, $name! You are $age years old.",
            system_prompt="You are a test bot.",
        )
        user_prompt, system_prompt = template.format(
            name="World", age="42"
        )
        assert user_prompt == "Hello, World! You are 42 years old."
        assert system_prompt == "You are a test bot."

    def test_template_service_get(self):
        service = PromptTemplateService()
        template = service.get_template("question_generation")
        assert template.name == "question_generation"

    def test_template_service_unknown_template(self):
        service = PromptTemplateService()
        with pytest.raises(KeyError, match="Unknown template"):
            service.get_template("nonexistent")

    def test_template_service_register(self):
        service = PromptTemplateService()
        new_template = PromptTemplate(
            name="custom", template="Custom: $var"
        )
        service.register_template(new_template)
        assert "custom" in service.list_templates()

    def test_template_service_list(self):
        service = PromptTemplateService()
        templates = service.list_templates()
        assert len(templates) >= 2
        assert "question_generation" in templates
        assert "answer_analysis" in templates
        assert "interviewer_turn" in templates

    def test_interviewer_template_includes_response_policy(self):
        service = PromptTemplateService()
        template = service.get_template("interviewer_turn")
        assert "$response_policy" in template.template


# ─── Question Service Tests ──────────────────────────────────────────


class TestQuestionService:
    """Tests for QuestionService."""

    @pytest.fixture
    def question_service(self):
        llm = MagicMock(spec=LLMClient)
        llm.generate_json = AsyncMock(return_value={
            "questions": [
                {
                    "text": "What is 2+2?",
                    "type": "short_answer",
                    "difficulty": "easy",
                    "expected_answer": "4",
                    "explanation": "Basic addition",
                }
            ]
        })
        templates = PromptTemplateService()
        return QuestionService(llm_client=llm, prompt_templates=templates)

    def test_generate_questions(self, question_service):
        import asyncio

        result = asyncio.get_event_loop().run_until_complete(
            question_service.generate_questions(
                topic="Math basics",
                num_questions=1,
                difficulty="easy",
            )
        )
        assert result.topic == "Math basics"
        assert len(result.questions) == 1
        assert result.questions[0].text == "What is 2+2?"


# ─── Analysis Service Tests ─────────────────────────────────────────


class TestAnalysisService:
    """Tests for AnalysisService."""

    @pytest.fixture
    def analysis_service(self):
        llm = MagicMock(spec=LLMClient)
        llm.generate_json = AsyncMock(
            return_value={
                "scores": {
                    "correctness": 80,
                    "completeness": 70,
                    "clarity": 90,
                    "relevance": 85,
                },
                "overall_score": 81,
                "feedback": "Good answer overall.",
                "strengths": ["Clear writing"],
                "weaknesses": ["Missing details"],
                "suggested_improvements": ["Expand on key points"],
            }
        )
        templates = PromptTemplateService()
        embeddings = MagicMock(spec=EmbeddingService)
        embeddings.similarity = AsyncMock(return_value=0.85)
        return AnalysisService(
            llm_client=llm,
            prompt_templates=templates,
            embedding_service=embeddings,
        )

    def test_analyze_answer(self, analysis_service):
        import asyncio

        result = asyncio.get_event_loop().run_until_complete(
            analysis_service.analyze_answer(
                question="What is Python?",
                answer="A programming language",
                expected_answer="Python is a high-level programming language",
            )
        )
        assert result.overall_score == 36.25
        assert result.scores.correctness == 35.0
        assert result.scores.relevance == 40.0
        assert len(result.weaknesses) >= 3

    def test_compute_similarity(self, analysis_service):
        import asyncio

        score = asyncio.get_event_loop().run_until_complete(
            analysis_service.compute_similarity("text a", "text b")
        )
        assert score == 0.85


class TestInterviewCommunicationHelpers:
    """Tests for interviewer communication heuristics."""

    def test_inspect_last_answer_flags_weak_response(self):
        from src.api.routes import _inspect_last_answer, _response_policy

        profile = _inspect_last_answer("не знаю, кажется что-то там", "Python")

        assert profile["is_weak"] is True
        assert profile["next_action"] == "задать точечный follow-up"
        assert "reasoning" in profile["follow_up_focus"]
        assert "follow-up" in _response_policy("не знаю, кажется что-то там", "Python")

    def test_inspect_last_answer_flags_strong_response(self):
        from src.api.routes import _inspect_last_answer, _response_policy

        profile = _inspect_last_answer(
            "Для backend performance, кеширования, нагрузки p95 и отказоустойчивости я сначала уменьшу число обращений к БД, затем проверю кеширование, а потом сниму метрики нагрузки и частичных сбоев.",
            "backend performance кеширование нагрузка p95 отказоустойчивость",
        )

        assert profile["is_weak"] is False
        assert profile["is_partial"] is False
        assert profile["next_action"] == "углубить вопрос"
        assert "масштабирование" in _response_policy(
            "Для backend performance, кеширования, нагрузки p95 и отказоустойчивости я сначала уменьшу число обращений к БД, затем проверю кеширование, а потом сниму метрики нагрузки и частичных сбоев.",
            "backend performance кеширование нагрузка p95 отказоустойчивость",
        )


class TestPostAnalysisCommunicationHelpers:
    """Tests for post-analysis narrative quality helpers."""

    def test_post_analysis_weaknesses_highlight_short_and_deflective_answers(self):
        from src.api.routes import _build_post_analysis_weaknesses

        weaknesses = _build_post_analysis_weaknesses(
            correctness=45,
            clarity=50,
            completeness=48,
            relevance=52,
            turn_pairs=[
                ("Q1", "не знаю"),
                ("Q2", "кажется, это что-то про кеш"),
                ("Q3", "примерно так"),
            ],
        )

        assert "Слишком много коротких ответов без объяснения reasoning" in weaknesses
        assert "Есть уклончивые ответы вместо прямого решения" in weaknesses
        assert "Кандидат часто избегает конкретики под давлением интервью" in weaknesses

    def test_post_analysis_recommendations_include_structured_answer_pattern(self):
        from src.api.routes import _build_post_analysis_recommendations

        recommendations = _build_post_analysis_recommendations(
            correctness=55,
            clarity=54,
            completeness=50,
            relevance=53,
            turn_pairs=[
                ("Q1", "не знаю"),
                ("Q2", "кажется"),
            ],
        )

        assert "Давайте ответ в 3 шага: решение, почему оно выбрано, как проверите результат" in recommendations
        assert "Если не уверены, озвучивайте гипотезу и план проверки, а не уходите в общие фразы" in recommendations
        assert "Продлите интервью, чтобы накопить больше данных для оценки" in recommendations

    def test_post_analysis_strengths_and_recommendations_for_high_scores(self):
        from src.api.routes import (
            _build_post_analysis_recommendations,
            _build_post_analysis_strengths,
        )

        strengths = _build_post_analysis_strengths(
            correctness=86,
            clarity=84,
            completeness=83,
            relevance=87,
        )
        recommendations = _build_post_analysis_recommendations(
            correctness=86,
            clarity=84,
            completeness=83,
            relevance=87,
            turn_pairs=[
                ("Q1", "Я объяснил архитектурный выбор, сравнил альтернативы и отметил риски."),
                ("Q2", "Я описал метрики p95/p99, способ валидации и план отката."),
                ("Q3", "Я добавил edge cases и критерии готовности решения к продакшену."),
            ],
        )

        assert "Кандидат стабильно держит уровень и может обсуждать сложные trade-offs" in strengths
        assert "Добавьте продвинутый уровень: обсуждайте failure modes, метрики и стратегию отката" in recommendations


# ─── Transcription Service Tests ─────────────────────────────────────


class TestTranscriptionService:
    """Tests for TranscriptionService."""

    def test_validate_audio_empty(self):
        service = TranscriptionService(
            api_key="test-key", max_file_size_mb=25
        )
        with pytest.raises(ValueError, match="cannot be empty"):
            import asyncio

            asyncio.get_event_loop().run_until_complete(
                service.transcribe(b"")
            )

    def test_validate_audio_too_large(self):
        service = TranscriptionService(
            api_key="test-key", max_file_size_mb=1
        )
        large_audio = b"0" * (2 * 1024 * 1024)  # 2MB, limit is 1MB
        with pytest.raises(ValueError, match="too large"):
            import asyncio

            asyncio.get_event_loop().run_until_complete(
                service.transcribe(large_audio)
            )


# ─── Validator Tests ────────────────────────────────────────────────


class TestValidators:
    """Tests for input validators."""

    def test_validate_topic_valid(self):
        assert validate_topic("Python programming") == "Python programming"

    def test_validate_topic_empty(self):
        with pytest.raises(ValueError, match="cannot be empty"):
            validate_topic("")

    def test_validate_topic_too_short(self):
        with pytest.raises(ValueError, match="at least 2 characters"):
            validate_topic("a")

    def test_validate_difficulty_valid(self):
        assert validate_difficulty("EASY") == "easy"
        assert validate_difficulty("Medium") == "medium"
        assert validate_difficulty("  HARD  ") == "hard"

    def test_validate_difficulty_invalid(self):
        with pytest.raises(ValueError, match="Invalid difficulty"):
            validate_difficulty("extreme")

    def test_validate_text_length_empty(self):
        with pytest.raises(ValueError, match="cannot be empty"):
            validate_text_length("")

    def test_validate_text_length_too_long(self):
        with pytest.raises(ValueError, match="must not exceed"):
            validate_text_length("x" * 10001, max_length=10000)

    def test_validate_language_code_valid(self):
        assert validate_language_code("en") == "en"
        assert validate_language_code("ru") == "ru"

    def test_validate_language_code_invalid(self):
        with pytest.raises(ValueError, match="Invalid language code"):
            validate_language_code("english")

    def test_validate_language_code_none(self):
        assert validate_language_code(None) is None

    def test_validate_question_count_valid(self):
        assert validate_question_count(5) == 5

    def test_validate_question_count_zero(self):
        with pytest.raises(ValueError, match="at least 1"):
            validate_question_count(0)

    def test_validate_question_count_too_high(self):
        with pytest.raises(ValueError, match="more than 50"):
            validate_question_count(51)


# ─── Helper Tests ────────────────────────────────────────────────────


class TestHelpers:
    """Tests for utility helpers."""

    def test_truncate_text_short(self):
        assert truncate_text("Hello", max_length=10) == "Hello"

    def test_truncate_text_long(self):
        result = truncate_text("Hello World", max_length=8)
        assert result == "Hello..."
        assert len(result) <= 8

    def test_sanitize_text(self):
        assert sanitize_text("  hello   world  ") == "hello world"

    def test_compute_text_hash(self):
        hash1 = compute_text_hash("test")
        hash2 = compute_text_hash("test")
        hash3 = compute_text_hash("different")
        assert hash1 == hash2
        assert hash1 != hash3

    def test_format_duration(self):
        assert format_duration(65) == "1m 5s"
        assert format_duration(3661) == "1h 1m 1s"
