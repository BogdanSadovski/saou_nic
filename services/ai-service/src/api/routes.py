"""API route definitions."""

import logging
import re
from typing import Annotated

from fastapi import APIRouter, Depends, HTTPException, Query, UploadFile, File
from fastapi.responses import JSONResponse

from src.api.dependencies import DIContainer, get_di_container
from src.core.embeddings import get_local_embedding_client
from src.models.responses import (
    AnalysisScores,
    AnalysisResponse,
    DeveloperInsightsResponse,
    ResumeInsightsResponse,
    DeveloperInterviewTrack,
    DeveloperLanguageInsight,
    DeveloperRoleRecommendation,
    ErrorResponse,
    HealthResponse,
    NextQuestionResponse,
    PostAnalysisResponse,
    QuestionGenerationResponse,
    TranscriptionResponse,
    ValidateOutputResponse,
)
from src.models.schemas import (
    AnalysisRequest,
    DeveloperInsightsRequest,
    ResumeInsightsRequest,
    InterviewHistoryMessage,
    NextQuestionRequest,
    PostAnalysisRequest,
    QuestionGenerationRequest,
    SimilarityRequest,
    ValidateOutputRequest,
)
from src.utils.validators import validate_text_length

logger = logging.getLogger(__name__)

router = APIRouter(tags=["ai-service"])


def _get_container(
    container: Annotated[DIContainer, Depends(get_di_container)],
) -> DIContainer:
    return container


@router.post(
    "/questions/generate",
    response_model=QuestionGenerationResponse,
    responses={
        400: {"model": ErrorResponse},
        422: {"model": ErrorResponse},
        500: {"model": ErrorResponse},
    },
)
async def generate_questions(
    request: QuestionGenerationRequest,
    container: Annotated[DIContainer, Depends(_get_container)],
) -> QuestionGenerationResponse:
    """Generate AI-powered questions based on the provided topic and context."""
    try:
        service = container.get_question_service()
        result = await service.generate_questions(
            topic=request.topic,
            context=request.context,
            num_questions=request.num_questions,
            difficulty=request.difficulty,
            question_types=request.question_types,
        )
        return result
    except ValueError as exc:
        logger.warning("Invalid request: %s", exc)
        raise HTTPException(status_code=400, detail=str(exc)) from exc
    except Exception as exc:
        logger.exception("Unexpected error generating questions")
        raise HTTPException(
            status_code=500, detail="Internal server error"
        ) from exc


@router.post(
    "/analysis/answer",
    response_model=AnalysisResponse,
    responses={
        400: {"model": ErrorResponse},
        422: {"model": ErrorResponse},
        500: {"model": ErrorResponse},
    },
)
async def analyze_answer(
    request: AnalysisRequest,
    container: Annotated[DIContainer, Depends(_get_container)],
) -> AnalysisResponse:
    """Analyze a student's answer for correctness, completeness, and quality."""
    validate_text_length(request.answer, max_length=10000, field_name="answer")
    validate_text_length(
        request.expected_answer, max_length=10000, field_name="expected_answer"
    )

    try:
        service = container.get_analysis_service()
        result = await service.analyze_answer(
            question=request.question,
            answer=request.answer,
            expected_answer=request.expected_answer,
            rubric=request.rubric,
        )
        return result
    except ValueError as exc:
        logger.warning("Invalid request: %s", exc)
        raise HTTPException(status_code=400, detail=str(exc)) from exc
    except Exception as exc:
        logger.exception("Unexpected error analyzing answer")
        return AnalysisResponse(
            scores=AnalysisScores(
                correctness=0,
                completeness=0,
                clarity=0,
                relevance=0,
            ),
            overall_score=0,
            feedback="Анализ временно недоступен. Пожалуйста, попробуйте позже.",
            strengths=[],
            weaknesses=["Сервис анализа ИИ сейчас недоступен"],
            suggested_improvements=[
                "Повторите запрос на анализ через несколько минут"
            ],
        )


@router.post(
    "/analysis/similarity",
    response_model=dict[str, float],
)
async def compute_similarity(
    request: SimilarityRequest,
    container: Annotated[DIContainer, Depends(_get_container)],
) -> dict[str, float]:
    """Compute semantic similarity between two text passages."""
    try:
        service = container.get_analysis_service()
        score = await service.compute_similarity(
            text_a=request.text_a,
            text_b=request.text_b,
        )
        return {"similarity": score}
    except ValueError as exc:
        raise HTTPException(status_code=400, detail=str(exc)) from exc
    except Exception as exc:
        logger.exception("Unexpected error computing similarity")
        raise HTTPException(
            status_code=500, detail="Internal server error"
        ) from exc


@router.post(
    "/transcription",
    response_model=TranscriptionResponse,
)
async def transcribe_audio(
    file: Annotated[UploadFile, File(description="Audio file to transcribe")],
    language: Annotated[
        str | None, Query(description="Language code (e.g. 'en', 'ru')")
    ] = None,
    container: Annotated[DIContainer, Depends(_get_container)] = None,
) -> TranscriptionResponse:
    """Transcribe audio content to text."""
    if container is None:
        container = get_di_container()

    if not file.content_type or not file.content_type.startswith("audio/"):
        raise HTTPException(
            status_code=400,
            detail="Invalid file type. Only audio files are accepted.",
        )

    try:
        audio_bytes = await file.read()
        service = container.get_transcription_service()
        result = await service.transcribe(
            audio_data=audio_bytes, language=language
        )
        return result
    except ValueError as exc:
        raise HTTPException(status_code=400, detail=str(exc)) from exc
    except Exception as exc:
        logger.exception("Unexpected error during transcription")
        raise HTTPException(
            status_code=500, detail="Internal server error"
        ) from exc


_INTERVIEW_OUTPUT_SCHEMA = {
    "name": "interviewer_turn_response",
    "strict": True,
    "schema": {
        "type": "object",
        "additionalProperties": False,
        "required": [
            "question",
            "topic",
            "difficulty_delta",
            "pressure_level",
            "flags",
        ],
        "properties": {
            "question": {"type": "string", "minLength": 8, "maxLength": 700},
            "topic": {"type": "string", "minLength": 2, "maxLength": 120},
            "difficulty_delta": {"type": "integer", "minimum": -2, "maximum": 2},
            "pressure_level": {"type": "integer", "minimum": 1, "maximum": 5},
            "flags": {
                "type": "object",
                "additionalProperties": False,
                "required": [
                    "contains_explanation",
                    "contains_solution",
                    "policy_violation",
                ],
                "properties": {
                    "contains_explanation": {"type": "boolean"},
                    "contains_solution": {"type": "boolean"},
                    "policy_violation": {"type": "boolean"},
                },
            },
        },
    },
}


def _sanitize_interviewer_question(text: str) -> tuple[bool, list[str], str]:
    violations: list[str] = []
    cleaned = text.strip()
    cleaned = re.sub(r"\s+", " ", cleaned)
    if len(cleaned) < 8:
        violations.append("too_short")
    if len(cleaned) > 700:
        cleaned = cleaned[:700].rstrip()
        violations.append("trimmed_to_max_length")

    if cleaned and not cleaned.endswith("?"):
        cleaned = cleaned.rstrip(".!") + "?"

    is_valid = "too_short" not in violations
    return is_valid, violations, cleaned


def _question_fingerprint(text: str) -> str:
    normalized = re.sub(r"[^\w\s]", " ", text.lower())
    normalized = re.sub(r"\s+", " ", normalized).strip()
    return normalized


def _semantic_threshold_for_role(role: str) -> float:
    role_key = (role or "").strip().lower()
    thresholds = {
        "backend": 0.78,
        "frontend": 0.76,
        "devops": 0.77,
        "sre": 0.77,
        "platform": 0.77,
        "ml": 0.79,
        "data": 0.79,
        "mobile": 0.76,
    }
    for key, value in thresholds.items():
        if key in role_key:
            return value
    return 0.77


def _is_avoided_question(
    question: str,
    avoid_questions: list[str],
    role: str = "",
    strict: bool = False,
) -> bool:
    """Check if question is too similar to avoided questions using semantic similarity."""
    if not question or not avoid_questions:
        return False

    # First try exact fingerprint (fast path)
    candidate_fp = _question_fingerprint(question)
    if not candidate_fp:
        return False

    avoid_fps = {_question_fingerprint(item) for item in avoid_questions if item}
    if candidate_fp in avoid_fps:
        logger.debug("Exact fingerprint match found, rejecting question")
        return True

    threshold = _semantic_threshold_for_role(role)
    if strict:
        threshold = max(0.70, threshold - 0.05)

    # Then check semantic similarity (more expensive)
    try:
        embedding_client = get_local_embedding_client()
        candidate_embedding = embedding_client.embed(question)
        if candidate_embedding is None:
            # Model not available, fall back to fingerprint only
            return False

        for avoid_q in avoid_questions:
            if not avoid_q or not avoid_q.strip():
                continue
            avoid_embedding = embedding_client.embed(avoid_q)
            if avoid_embedding is None:
                continue

            similarity = embedding_client.cosine_similarity(candidate_embedding, avoid_embedding)
            if similarity > threshold:
                logger.debug(
                    "Semantic similarity %.3f > %.2f, rejecting question for role=%s. "
                    "Original: %s, Candidate: %s",
                    similarity,
                    threshold,
                    role,
                    avoid_q[:100],
                    question[:100],
                )
                return True
    except Exception as e:
        logger.warning("Semantic similarity check failed: %s", e)
        return False

    return False


def _classify_tutor_like_text(text: str) -> tuple[bool, float]:
    lowered = text.lower()
    signals = 0
    patterns = [
        r"\bправильн(ый|ое|ая)\b",
        r"\bрешение\b",
        r"\bобъясн(ю|им|ение)\b",
        r"\bшаг(и| за шагом)\b",
        r"\bиспользуйте\b",
        r"\bвот код\b",
        r"\bпример кода\b",
    ]
    for pattern in patterns:
        if re.search(pattern, lowered):
            signals += 1

    # Additional classifier feature: many imperative verbs and long narrative.
    imperative_markers = ["сделайте", "возьмите", "реализуйте", "постройте"]
    signals += sum(1 for marker in imperative_markers if marker in lowered)
    if len(text) > 280:
        signals += 1

    confidence = min(1.0, signals / 4.0)
    return signals >= 2, confidence


def _policy_flags_from_text(text: str) -> dict[str, bool]:
    lowered = text.lower()
    contains_solution = any(marker in lowered for marker in ["вот код", "готовый код", "final code"])
    contains_explanation = any(marker in lowered for marker in ["потому что", "объяснение", "теория", "например"])
    policy_violation = False
    return {
        "contains_explanation": contains_explanation,
        "contains_solution": contains_solution,
        "policy_violation": policy_violation,
    }


def _fallback_question(req: NextQuestionRequest, answer_profile: dict[str, object] | None = None) -> str:
    profile = answer_profile or {}
    recent_topics = [topic.strip() for topic in (req.recent_topics or []) if topic and topic.strip()]
    current_topic = (req.current_topic or "").strip()
    candidate_topic = current_topic
    for topic in recent_topics:
        if topic.lower() != current_topic.lower():
            candidate_topic = topic
            break

    session_context = (req.session_context or "").strip()
    if profile.get("is_weak"):
        if session_context:
            return (
                "Ответ пока слабый или уклончивый. "
                "Уточните reasoning, назовите конкретный шаг и объясните, как бы вы проверили результат. "
                f"Контекст: {session_context.splitlines()[0]}"
            )
        return (
            f"Ответ пока слабый для роли {req.role} уровня {req.level}. "
            "Объясните reasoning, конкретный trade-off и один risk/failure mode."
        )

    if profile.get("is_partial"):
        if session_context:
            return (
                "Ответ частичный, нужно больше глубины. "
                "Раскройте детали реализации, edge cases и критерий проверки результата. "
                f"Контекст: {session_context.splitlines()[0]}"
            )
        return (
            f"Ответ частичный для роли {req.role} уровня {req.level}. "
            "Добавьте детали реализации, edge cases и критерий проверки результата."
        )

    if session_context:
        return (
            "Нужен более точный ответ с учетом контекста сессии. "
            f"{session_context.splitlines()[0]}"
        )

    if candidate_topic:
        return (
            f"Сместимся от темы {candidate_topic} к соседней области: "
            "опишите trade-offs, риски и критерий проверки результата."
        )

    return (
        f"Вы претендуете на роль {req.role} уровня {req.level}. "
        "Опишите архитектурное решение, которое вы выбрали бы в реальном продакшене, "
        "и какие trade-offs вы считаете ключевыми?"
    )


def _build_session_context(request: NextQuestionRequest) -> str:
    if request.session_context and request.session_context.strip():
        return request.session_context.strip()

    focus_areas = ", ".join(request.focus_areas or []) or "нет"
    primary_skills = ", ".join(request.primary_skills or []) or "нет"
    theory_focus = ", ".join(request.theory_focus or []) or "нет"
    practice_focus = ", ".join(request.practice_focus or []) or "нет"
    vacancy_title = request.vacancy_title or "нет"
    vacancy_category = request.vacancy_category or "нет"
    current_topic = request.current_topic or "general"
    last_answer = (request.last_candidate_answer or "(первый вопрос)").strip()
    if len(last_answer) > 500:
        last_answer = last_answer[:500].rstrip() + "..."

    lines = [
        f"роль: {request.role}",
        f"уровень: {request.level}",
        f"вакансия: {vacancy_title}",
        f"категория вакансии: {vacancy_category}",
        f"режим: {request.interview_mode or 'practice'}",
        f"текущая тема: {current_topic}",
        f"сложность: {request.difficulty}/10",
        f"давление: {request.pressure_level}/5",
        f"осталось секунд: {request.time_left_sec}",
        f"осталось вопросов: {request.questions_left}",
        f"фокус-области: {focus_areas}",
        f"ключевые навыки: {primary_skills}",
        f"теоретический фокус: {theory_focus}",
        f"практический фокус: {practice_focus}",
        f"последний ответ: {last_answer}",
        f"история ходов: {len(request.history or [])}",
        f"последние темы: {', '.join(request.recent_topics or []) or 'нет'}",
        f"избегать вопросов: {len(request.avoid_questions or [])}",
    ]
    return "\n".join(lines)


def _inspect_last_answer(answer: str, topic: str) -> dict[str, object]:
    answer_text = (answer or "").strip()
    tokens = _tokenize(answer_text)
    word_count = len(tokens)
    structure_score = _structure_score(answer_text)
    evidence_score = _evidence_score(answer_text)
    topical_overlap = _lexical_overlap(topic or "", answer_text) if topic else 0.0
    deflective = _contains_deflection(answer_text)

    if not answer_text:
        return {
            "is_empty": True,
            "is_weak": True,
            "is_partial": False,
            "summary": "Последний ответ отсутствует.",
            "guidance": "Кандидат не ответил. Сформулируй короткий, прямой вопрос и попроси ответить по сути без отступлений.",
            "tone": "короткий, строгий, нейтральный",
            "next_action": "задать короткий прямой вопрос",
            "follow_up_focus": "получить базовый ответ без лишних вступлений",
        }

    is_weak = deflective or word_count < 8 or topical_overlap < 0.12
    is_partial = not is_weak and (word_count < 24 or structure_score < 0.35 or evidence_score < 0.15)

    if is_weak:
        summary = "Ответ слабый, уклончивый или мимо темы."
        guidance = (
            "Последний ответ слабый. Не переходи к новой теме как будто ответ был достаточным. "
            "Сделай точечный follow-up: попроси объяснить reasoning, назвать конкретный шаг, риск, компромисс или пример. "
            "Если кандидат снова уходит в сторону, переключись на более базовую проверку понимания."
        )
        tone = "короткий, строгий, без похвалы"
        next_action = "задать точечный follow-up"
        follow_up_focus = "reasoning, конкретный шаг, риск, компромисс или пример"
    elif is_partial:
        summary = "Ответ частичный: тема затронута, но не раскрыта достаточно глубоко."
        guidance = (
            "Последний ответ частичный. Уточни детали реализации, trade-offs, edge cases или критерий проверки результата."
        )
        tone = "деловой, уточняющий"
        next_action = "уточнить недостающие детали"
        follow_up_focus = "детали реализации, trade-offs, edge cases или критерий проверки результата"
    else:
        summary = "Ответ достаточно сильный, можно идти глубже."
        guidance = (
            "Последний ответ выглядит сильным. Можно повышать глубину: спроси про масштабирование, отказоустойчивость, альтернативы и продакшен-риски."
        )
        tone = "уверенный, без лишнего обучения"
        next_action = "углубить вопрос"
        follow_up_focus = "масштабирование, отказоустойчивость, альтернативы и продакшен-риски"

    return {
        "is_empty": False,
        "is_weak": is_weak,
        "is_partial": is_partial,
        "summary": summary,
        "guidance": guidance,
        "tone": tone,
        "next_action": next_action,
        "follow_up_focus": follow_up_focus,
    }


def _blend_difficulty_delta(requested_delta: int, answer_profile: dict[str, object]) -> int:
    delta = max(-2, min(2, int(requested_delta)))
    if answer_profile.get("is_weak"):
        delta = min(delta, -1)
    elif answer_profile.get("is_partial"):
        delta = max(-1, min(1, delta))
    else:
        delta = max(delta, 0)
    return delta


def _blend_pressure_level(requested_pressure: int, base_pressure: int, answer_profile: dict[str, object]) -> int:
    pressure = max(1, min(5, int(requested_pressure)))
    if answer_profile.get("is_weak"):
        pressure = min(5, max(pressure, int(base_pressure) + 1))
    elif answer_profile.get("is_partial"):
        pressure = max(1, min(5, max(pressure, int(base_pressure))))
    else:
        pressure = max(1, min(5, max(pressure, int(base_pressure))))
    return pressure


def _answer_guidance(answer: str, topic: str) -> str:
    profile = _inspect_last_answer(answer, topic)
    return (
        f"{profile['summary']} {profile['guidance']} "
        f"Тон: {profile['tone']}. "
        f"Следующее действие: {profile['next_action']}. "
        f"Фокус уточнения: {profile['follow_up_focus']}."
    )


def _response_policy(answer: str, topic: str) -> str:
    profile = _inspect_last_answer(answer, topic)
    if profile.get("is_empty"):
        return (
            "Задай один короткий прямой вопрос без вступлений и без похвалы. "
            "Не меняй тему, пока не получишь базовый ответ."
        )
    if profile.get("is_weak"):
        return (
            "Сделай один точечный follow-up по reasoning, конкретному шагу, риску, компромиссу или примеру. "
            "Не переходи к новой теме и не смягчай слабый ответ похвалой."
        )
    if profile.get("is_partial"):
        return (
            "Уточни недостающие детали, trade-offs, edge cases или критерий проверки результата. "
            "Держи вопрос коротким и сфокусированным на одном пробеле."
        )
    return (
        "Углубляйся в масштабирование, отказоустойчивость, альтернативы и production-риски. "
        "Сохраняй деловой тон и избегай длинных вступлений."
    )


def _normalize_topic_label(value: str) -> str:
    return re.sub(r"\s+", " ", (value or "").strip().lower())


def _topic_repetition_detected(question: str, current_topic: str, recent_topics: list[str]) -> bool:
    normalized_question = _normalize_topic_label(question)
    normalized_current = _normalize_topic_label(current_topic)
    if not normalized_question or not normalized_current:
        return False

    same_topic_count = sum(1 for topic in recent_topics or [] if _normalize_topic_label(topic) == normalized_current)
    if same_topic_count < 2:
        return False

    return normalized_current in normalized_question


def _format_interviewer_history(messages: list[InterviewHistoryMessage]) -> str:
    if not messages:
        return "- История отсутствует"

    recent_messages = messages[-8:]
    lines: list[str] = []
    for message in recent_messages:
        sender = message.sender.strip().lower()
        label = "Кандидат" if sender == "user" else "Интервьюер" if sender == "ai" else "Система"
        content = " ".join(message.content.split())
        if len(content) > 240:
            content = content[:237].rstrip() + "..."
        topic = f" | тема: {message.topic}" if message.topic else ""
        lines.append(f"- {label}{topic}: {content}")

    return "\n".join(lines)


def _format_avoid_questions(questions: list[str]) -> str:
    cleaned = [" ".join(q.split()) for q in (questions or []) if q and q.strip()]
    if not cleaned:
        return "- Нет"

    limited = cleaned[-20:]
    return "\n".join(f"- {item[:220]}" for item in limited)


def _format_list_block(values: list[str], empty_label: str = "- Нет") -> str:
    cleaned = [" ".join(value.split()) for value in (values or []) if value and value.strip()]
    if not cleaned:
        return empty_label
    return "\n".join(f"- {value[:180]}" for value in cleaned[:12])


def _interview_mode_guidance(mode: str) -> str:
    normalized = (mode or "").strip().lower()
    if normalized in {"practice", "practical", "coding", "code"}:
        return (
            "Режим practice (live coding): давай конкретное coding-задание, проверяй присланное решение, "
            "если решение корректно — переходи к следующему заданию, если нет — дай правильный ответ, короткое "
            "объяснение и предложи, что исправить."
        )
    if normalized in {"theory", "theoretical", "concept", "conceptual"}:
        return (
            "Режим theory: задавай вопросы с упором на концепции, trade-offs, архитектуру, принципы, "
            "обоснование решений и проверку понимания."
        )
    return "Режим practice (live coding): работай как интервьюер по live coding с проверкой решения."


def _interviewer_style_profile(role: str, level: str) -> str:
    normalized_role = role.lower()
    normalized_level = level.lower()

    profiles = {
        "backend": "жесткий, архитектурный, через API, транзакции, consistency, масштабирование и отказоустойчивость",
        "frontend": "прагматичный, про UX, рендеринг, состояние, производительность и регрессии",
        "devops": "операционный, про релизы, инциденты, observability, rollback и blast radius",
        "ml": "метрический, про данные, качество модели, leakage, drift и воспроизводимость",
        "data": "аналитический, про пайплайны, качество данных, SLA и reliability",
        "mobile": "продуктовый, про офлайн, синхронизацию, стабильность и release safety",
    }

    style = "строгий, короткий, без обучения"
    for key, value in profiles.items():
        if key in normalized_role:
            style = value
            break

    if normalized_level in {"senior", "staff", "principal"}:
        style += ", с повышенным давлением на trade-offs и edge cases"
    elif normalized_level in {"junior", "intern"}:
        style += ", но без снисходительного тона"

    return style


def _hard_mode_follow_up(role: str, topic: str, question: str) -> str:
    role_key = role.lower()
    topic_key = (topic or "").lower()
    text = question.lower()

    if "backend" in role_key:
        if any(marker in text for marker in ["нагруз", "p95", "p99", "отказ", "latency"]):
            return ""
        probes = [
            "Как вы подтвердите это метриками p95/p99 и тестом с частичными отказами?",
            "Какой сценарий деградации вы заложите первым и как проверите его на нагрузочном тесте?",
            "Какие два failure mode вы считаете критичными и как воспроизведете их до релиза?",
        ]
    elif "frontend" in role_key:
        if any(marker in text for marker in ["ux", "web vitals", "lcp", "cls", "inp", "перформ"]):
            return ""
        probes = [
            "Как вы докажете, что UX не просел: какие Web Vitals и где измерите?",
            "Какой регрессионный сценарий производительности вы проверите в CI в первую очередь?",
            "Какие пользовательские метрики вы возьмете, чтобы подтвердить эффект в проде?",
        ]
    elif any(key in role_key for key in ["devops", "sre", "platform"]):
        if any(marker in text for marker in ["blast radius", "rollback", "инцидент", "risk", "риск"]):
            return ""
        probes = [
            "Какой риск вы считаете самым опасным и как уменьшите blast radius?",
            "Какую проверку перед релизом нельзя пропустить, чтобы избежать массового инцидента?",
            "Что вы откатите первым при деградации и как сократите время восстановления?",
        ]
    elif "ml" in role_key or "data" in role_key:
        if any(marker in text for marker in ["drift", "auc", "f1", "precision", "recall", "смещ"]):
            return ""
        probes = [
            "Как вы проверите устойчивость на новых данных и сигнал drift до падения метрик?",
            "Какие срезы метрик покажут, что модель деградирует именно на целевом сегменте?",
            "Какой guardrail вы поставите, чтобы не пропустить деградацию качества в проде?",
        ]
    else:
        return ""

    seed = f"{role_key}|{topic_key}|{question}"
    idx = sum(ord(ch) for ch in seed) % len(probes)
    return probes[idx]


_DEVELOPER_INSIGHTS_SCHEMA = {
    "name": "developer_insights_response",
    "strict": True,
    "schema": {
        "type": "object",
        "additionalProperties": False,
        "required": [
            "summary",
            "strengths",
            "risks",
            "action_plan",
            "language_insights",
            "interview_tracks",
            "recommended_positions",
        ],
        "properties": {
            "summary": {"type": "string", "minLength": 40, "maxLength": 1200},
            "strengths": {
                "type": "array",
                "minItems": 2,
                "maxItems": 6,
                "items": {"type": "string", "minLength": 4, "maxLength": 220},
            },
            "risks": {
                "type": "array",
                "minItems": 1,
                "maxItems": 5,
                "items": {"type": "string", "minLength": 4, "maxLength": 220},
            },
            "action_plan": {
                "type": "array",
                "minItems": 2,
                "maxItems": 6,
                "items": {"type": "string", "minLength": 6, "maxLength": 220},
            },
            "language_insights": {
                "type": "array",
                "minItems": 1,
                "maxItems": 6,
                "items": {
                    "type": "object",
                    "additionalProperties": False,
                    "required": ["language", "confidence", "evidence", "interview_topics"],
                    "properties": {
                        "language": {"type": "string", "minLength": 1, "maxLength": 80},
                        "confidence": {"type": "integer", "minimum": 0, "maximum": 100},
                        "evidence": {"type": "string", "minLength": 10, "maxLength": 260},
                        "interview_topics": {
                            "type": "array",
                            "minItems": 1,
                            "maxItems": 6,
                            "items": {"type": "string", "minLength": 3, "maxLength": 120},
                        },
                    },
                },
            },
            "interview_tracks": {
                "type": "array",
                "minItems": 1,
                "maxItems": 4,
                "items": {
                    "type": "object",
                    "additionalProperties": False,
                    "required": [
                        "role",
                        "mode",
                        "level",
                        "duration_minutes",
                        "focus_areas",
                        "primary_skills",
                        "rationale",
                    ],
                    "properties": {
                        "role": {"type": "string", "minLength": 2, "maxLength": 120},
                        "mode": {"type": "string", "minLength": 4, "maxLength": 40},
                        "level": {"type": "string", "minLength": 4, "maxLength": 40},
                        "duration_minutes": {"type": "integer", "minimum": 10, "maximum": 120},
                        "focus_areas": {
                            "type": "array",
                            "minItems": 2,
                            "maxItems": 8,
                            "items": {"type": "string", "minLength": 3, "maxLength": 120},
                        },
                        "primary_skills": {
                            "type": "array",
                            "minItems": 2,
                            "maxItems": 10,
                            "items": {"type": "string", "minLength": 2, "maxLength": 80},
                        },
                        "rationale": {"type": "string", "minLength": 12, "maxLength": 320},
                    },
                },
            },
            "recommended_positions": {
                "type": "array",
                "minItems": 2,
                "maxItems": 5,
                "items": {
                    "type": "object",
                    "additionalProperties": False,
                    "required": ["role", "fit_score", "rationale"],
                    "properties": {
                        "role": {"type": "string", "minLength": 2, "maxLength": 140},
                        "fit_score": {"type": "integer", "minimum": 0, "maximum": 100},
                        "rationale": {"type": "string", "minLength": 12, "maxLength": 300},
                    },
                },
            },
        },
    },
}


_RESUME_INSIGHTS_SCHEMA = {
    "name": "resume_insights_response",
    "strict": True,
    "schema": {
        "type": "object",
        "additionalProperties": False,
        "required": [
            "summary",
            "strong_points",
            "improvement_points",
            "action_plan",
            "language_insights",
            "interview_tracks",
            "recommended_positions",
        ],
        "properties": {
            "summary": {"type": "string", "minLength": 40, "maxLength": 1400},
            "strong_points": {
                "type": "array",
                "minItems": 2,
                "maxItems": 8,
                "items": {"type": "string", "minLength": 4, "maxLength": 240},
            },
            "improvement_points": {
                "type": "array",
                "minItems": 1,
                "maxItems": 8,
                "items": {"type": "string", "minLength": 4, "maxLength": 240},
            },
            "action_plan": {
                "type": "array",
                "minItems": 2,
                "maxItems": 8,
                "items": {"type": "string", "minLength": 6, "maxLength": 240},
            },
            "language_insights": _DEVELOPER_INSIGHTS_SCHEMA["schema"]["properties"]["language_insights"],
            "interview_tracks": _DEVELOPER_INSIGHTS_SCHEMA["schema"]["properties"]["interview_tracks"],
            "recommended_positions": _DEVELOPER_INSIGHTS_SCHEMA["schema"]["properties"]["recommended_positions"],
        },
    },
}


def _fallback_resume_positions(request: ResumeInsightsRequest) -> list[DeveloperRoleRecommendation]:
    stack = " ".join((request.skills or []) + (request.languages or [])).lower()
    base = [
        DeveloperRoleRecommendation(
            role="Backend Engineer",
            fit_score=66,
            rationale="Резюме содержит инженерные признаки, подходящие для backend-собеседований.",
        ),
        DeveloperRoleRecommendation(
            role="Fullstack Engineer",
            fit_score=64,
            rationale="Профиль выглядит универсальным и подходит для смешанного технического интервью.",
        ),
    ]

    if any(item in stack for item in ["react", "typescript", "javascript", "frontend"]):
        base[1] = DeveloperRoleRecommendation(
            role="Frontend Engineer",
            fit_score=72,
            rationale="В резюме есть стек и формулировки, характерные для frontend-разработки.",
        )
    if any(item in stack for item in ["go", "java", "python", "rust", "backend"]):
        base[0] = DeveloperRoleRecommendation(
            role="Backend Engineer",
            fit_score=74,
            rationale="Ключевые навыки и опыт больше соответствуют backend-направлению.",
        )

    return base


def _build_resume_prompt(request: ResumeInsightsRequest) -> str:
    role_preferences = ", ".join(request.role_preferences or []) or "не указаны"
    skills = "\n".join(f"- {item}" for item in (request.skills or [])[:20]) or "- нет данных"
    languages = "\n".join(f"- {item}" for item in (request.languages or [])[:12]) or "- нет данных"
    excerpt = (request.text_excerpt or "").strip()[:6000] or "нет"

    return (
        "Сформируй аналитический отчет по резюме кандидата.\\n"
        "Пиши только на русском языке.\\n"
        "Не выдумывай факты: опирайся на входные данные и фрагмент резюме.\\n"
        "Дай: сильные стороны, зоны роста, конкретный план улучшения и рекомендации для интервью.\\n"
        "Добавь language_insights и interview_tracks, где первый track — лучший старт для кандидата.\\n\\n"
        f"Файл: {request.file_name}\\n"
        f"Content-Type: {request.content_type or 'нет'}\\n"
        f"Предпочтительные роли: {role_preferences}\\n"
        f"Слов в резюме: {request.word_count}\\n"
        f"Символов в резюме: {request.character_count}\\n"
        f"Опытов (entries): {request.experience_entries}\\n"
        f"Образований (entries): {request.education_entries}\\n\\n"
        "Навыки:\\n"
        f"{skills}\\n\\n"
        "Языки программирования:\\n"
        f"{languages}\\n\\n"
        "Фрагмент резюме:\\n"
        f"{excerpt}"
    )


@router.post(
    "/resume/insights",
    response_model=ResumeInsightsResponse,
    responses={400: {"model": ErrorResponse}, 500: {"model": ErrorResponse}},
)
async def resume_insights(
    request: ResumeInsightsRequest,
    container: Annotated[DIContainer, Depends(_get_container)],
) -> ResumeInsightsResponse:
    """Generate AI insights and interview recommendations from resume metadata/text."""
    llm = container.get_llm_client()
    prompt = _build_resume_prompt(request)
    system_prompt = (
        "Ты технический карьерный аналитик и интервьюер. "
        "Выводи только строгий JSON, практичный и применимый на интервью."
    )

    try:
        raw = await llm.generate_json(
            prompt=prompt,
            system_prompt=system_prompt,
            schema=_RESUME_INSIGHTS_SCHEMA,
        )

        strong_points = [str(item).strip() for item in raw.get("strong_points", []) if str(item).strip()]
        improvement_points = [
            str(item).strip() for item in raw.get("improvement_points", []) if str(item).strip()
        ]
        action_plan = [str(item).strip() for item in raw.get("action_plan", []) if str(item).strip()]

        language_insights: list[DeveloperLanguageInsight] = []
        for item in raw.get("language_insights", []):
            language = str(item.get("language", "")).strip()
            evidence = str(item.get("evidence", "")).strip()
            if not language or not evidence:
                continue
            confidence = int(item.get("confidence", 0))
            topics = [str(topic).strip() for topic in item.get("interview_topics", []) if str(topic).strip()]
            language_insights.append(
                DeveloperLanguageInsight(
                    language=language,
                    confidence=max(0, min(100, confidence)),
                    evidence=evidence,
                    interview_topics=topics[:6],
                )
            )

        interview_tracks: list[DeveloperInterviewTrack] = []
        for item in raw.get("interview_tracks", []):
            role = str(item.get("role", "")).strip()
            rationale = str(item.get("rationale", "")).strip()
            if not role or not rationale:
                continue
            mode = str(item.get("mode", "practice")).strip() or "practice"
            level = str(item.get("level", "Middle")).strip() or "Middle"
            duration_minutes = max(10, min(120, int(item.get("duration_minutes", 30))))
            focus_areas = [
                str(value).strip() for value in item.get("focus_areas", []) if str(value).strip()
            ]
            primary_skills = [
                str(value).strip() for value in item.get("primary_skills", []) if str(value).strip()
            ]
            interview_tracks.append(
                DeveloperInterviewTrack(
                    role=role,
                    mode=mode,
                    level=level,
                    duration_minutes=duration_minutes,
                    focus_areas=focus_areas[:8],
                    primary_skills=primary_skills[:10],
                    rationale=rationale,
                )
            )

        positions: list[DeveloperRoleRecommendation] = []
        for item in raw.get("recommended_positions", []):
            role = str(item.get("role", "")).strip()
            rationale = str(item.get("rationale", "")).strip()
            if not role or not rationale:
                continue
            fit_score = max(0, min(100, int(item.get("fit_score", 0))))
            positions.append(
                DeveloperRoleRecommendation(
                    role=role,
                    fit_score=fit_score,
                    rationale=rationale,
                )
            )

        if not strong_points:
            strong_points = [
                "Резюме содержит структурированный технический стек.",
                "Есть достаточный объем данных для построения интервью-плана.",
            ]
        if not improvement_points:
            improvement_points = [
                "Добавьте больше измеримых результатов и конкретных метрик по проектам.",
            ]
        if not action_plan:
            action_plan = [
                "Уточните достижения в формате 'действие -> эффект -> метрика'.",
                "Подготовьте 2-3 project case для практического интервью по сильному стеку.",
            ]
        if not language_insights:
            seed_language = (request.languages[0] if request.languages else "General").strip() or "General"
            language_insights = [
                DeveloperLanguageInsight(
                    language=seed_language,
                    confidence=62,
                    evidence="Язык указан в навыках резюме и может быть использован для интервью-фокуса.",
                    interview_topics=["архитектура", "тестирование", "производительность"],
                )
            ]
        if not positions:
            positions = _fallback_resume_positions(request)
        if not interview_tracks:
            top_role = positions[0].role if positions else "Software Engineer"
            primary_skills = [item.language for item in language_insights[:4]]
            if not primary_skills:
                primary_skills = (request.skills or [])[:4] or ["алгоритмы", "системный дизайн"]
            interview_tracks = [
                DeveloperInterviewTrack(
                    role=top_role,
                    mode="practice",
                    level="Middle",
                    duration_minutes=35,
                    focus_areas=["live coding", "разбор trade-offs", "production reliability"],
                    primary_skills=primary_skills,
                    rationale="Трек построен на сильных сторонах резюме и наиболее релевантном интервью-направлении.",
                )
            ]

        summary = str(raw.get("summary", "")).strip()
        if not summary:
            summary = (
                "Резюме демонстрирует достаточную техническую базу для целевого интервью, "
                "но требует усиления конкретными достижениями и метриками результатов."
            )

        return ResumeInsightsResponse(
            summary=summary,
            strong_points=strong_points[:8],
            improvement_points=improvement_points[:8],
            action_plan=action_plan[:8],
            language_insights=language_insights[:6],
            interview_tracks=interview_tracks[:4],
            recommended_positions=positions[:5],
        )
    except Exception:
        logger.exception("resume insights generation failed, fallback will be used")
        positions = _fallback_resume_positions(request)
        top_role = positions[0].role if positions else "Software Engineer"
        seed_language = (request.languages[0] if request.languages else "General").strip() or "General"
        return ResumeInsightsResponse(
            summary=(
                "Не удалось получить полный AI-анализ резюме, поэтому возвращен надежный fallback-отчет "
                "с базовыми рекомендациями для подготовки к интервью."
            ),
            strong_points=[
                "Резюме успешно обработано и может быть использовано для интервью-профилирования.",
                "Есть базовые сигналы по стеку и направлениям для старта подготовки.",
            ],
            improvement_points=[
                "Добавьте количественные метрики влияния в опыте работы.",
                "Уточните глубину владения ключевыми технологиями на конкретных кейсах.",
            ],
            action_plan=[
                "Сформируйте 2-3 истории проектов по схеме проблема-решение-результат.",
                "Пройдите практическое интервью по самому сильному языку/стеку.",
            ],
            language_insights=[
                DeveloperLanguageInsight(
                    language=seed_language,
                    confidence=60,
                    evidence="Язык определен из навыков резюме и подходит для стартового интервью-фокуса.",
                    interview_topics=["основы языка", "тестирование", "отладка"],
                )
            ],
            interview_tracks=[
                DeveloperInterviewTrack(
                    role=top_role,
                    mode="practice",
                    level="Middle",
                    duration_minutes=30,
                    focus_areas=["практика", "архитектура", "качество кода"],
                    primary_skills=[seed_language],
                    rationale="Fallback-track построен по доступному стеку и позиции с максимальным fit.",
                )
            ],
            recommended_positions=positions,
        )


def _fallback_role_recommendations(request: DeveloperInsightsRequest) -> list[DeveloperRoleRecommendation]:
    lang_labels = [item.label.lower() for item in request.language_distribution or []]
    top_lang = lang_labels[0] if lang_labels else ""
    repos = request.sampled_repos or 0
    stars = request.total_stars or 0

    defaults = [
        DeveloperRoleRecommendation(
            role="Backend Engineer",
            fit_score=68,
            rationale="Есть признаки практики в продуктовой разработке и базовой инженерной глубины.",
        ),
        DeveloperRoleRecommendation(
            role="Fullstack Engineer",
            fit_score=65,
            rationale="Профиль выглядит универсальным и подходит для смешанных задач в небольших командах.",
        ),
    ]

    if any(lang in top_lang for lang in ["python", "go", "rust", "java"]):
        defaults[0] = DeveloperRoleRecommendation(
            role="Backend Engineer",
            fit_score=74,
            rationale="Основной стек и активность репозиториев указывают на сильную backend-направленность.",
        )

    if any(lang in top_lang for lang in ["typescript", "javascript", "react"]):
        defaults[1] = DeveloperRoleRecommendation(
            role="Frontend Engineer",
            fit_score=72,
            rationale="Преобладание JavaScript/TypeScript сигнализирует о хорошем потенциале во frontend-ролях.",
        )

    if repos >= 8 and stars >= 100:
        defaults.append(
            DeveloperRoleRecommendation(
                role="Senior Software Engineer",
                fit_score=70,
                rationale="Количество репозиториев и социальный сигнал по звездам подтверждают устойчивую практику.",
            )
        )
    else:
        defaults.append(
            DeveloperRoleRecommendation(
                role="Middle Software Engineer",
                fit_score=67,
                rationale="Профиль показывает стабильную инженерную активность и потенциал к росту.",
            )
        )

    return defaults[:3]


def _build_developer_prompt(request: DeveloperInsightsRequest) -> str:
    role_preferences = ", ".join(request.role_preferences or []) or "не указаны"
    language_distribution = "\n".join(
        f"- {item.label}: {item.value}" for item in (request.language_distribution or [])[:8]
    ) or "- нет данных"
    monthly_activity = "\n".join(
        f"- {item.label}: {item.value}" for item in (request.monthly_activity or [])[-12:]
    ) or "- нет данных"
    top_repositories = "\n".join(
        (
            f"- {repo.name} | lang={repo.language or 'n/a'} | stars={repo.stars} | "
            f"forks={repo.forks} | issues={repo.open_issues} | push={repo.last_push or 'n/a'}"
        )
        for repo in (request.top_repositories or [])[:10]
    ) or "- нет данных"

    return (
        "Сформируй аналитический профиль разработчика по GitHub-данным.\\n"
        "Пиши только на русском языке.\\n"
        "Не выдумывай факты: опирайся только на входные метрики.\\n"
        "Оцени объективно: без завышений и без чрезмерного негатива.\\n"
        "Рекомендованные позиции должны быть конкретными и реалистичными для собеседования.\\n\\n"
        "Сделай акцент на языках программирования: по каждому ключевому языку дай уверенность, доказательства и темы для интервью.\\n"
        "Сформируй interview_tracks так, чтобы первый track был самым сильным направлением для кандидата.\\n"
        f"GitHub username: {request.github_username}\\n"
        f"Имя профиля: {request.profile_name or 'нет'}\\n"
        f"Bio: {(request.bio or 'нет')[:500]}\\n"
        f"Предпочтительные роли: {role_preferences}\\n"
        f"Followers: {request.followers}\\n"
        f"Following: {request.following}\\n"
        f"Public repos: {request.public_repos}\\n"
        f"Sampled repos: {request.sampled_repos}\\n"
        f"Total stars: {request.total_stars}\\n"
        f"Total forks: {request.total_forks}\\n"
        f"Total open issues: {request.total_open_issues}\\n\\n"
        "Language distribution:\\n"
        f"{language_distribution}\\n\\n"
        "Monthly activity:\\n"
        f"{monthly_activity}\\n\\n"
        "Top repositories:\\n"
        f"{top_repositories}"
    )


@router.post(
    "/developer/insights",
    response_model=DeveloperInsightsResponse,
    responses={400: {"model": ErrorResponse}, 500: {"model": ErrorResponse}},
)
async def developer_insights(
    request: DeveloperInsightsRequest,
    container: Annotated[DIContainer, Depends(_get_container)],
) -> DeveloperInsightsResponse:
    """Generate AI insights and role recommendations from GitHub profile metrics."""
    llm = container.get_llm_client()
    prompt = _build_developer_prompt(request)
    system_prompt = (
        "Ты технический interviewer-аналитик. Выводи только строгий JSON. "
        "Рекомендации должны быть практичными для оценки на собеседовании."
    )

    try:
        raw = await llm.generate_json(
            prompt=prompt,
            system_prompt=system_prompt,
            schema=_DEVELOPER_INSIGHTS_SCHEMA,
        )

        positions: list[DeveloperRoleRecommendation] = []
        for item in raw.get("recommended_positions", []):
            role = str(item.get("role", "")).strip()
            rationale = str(item.get("rationale", "")).strip()
            if not role or not rationale:
                continue
            fit_score = int(item.get("fit_score", 0))
            fit_score = max(0, min(100, fit_score))
            positions.append(
                DeveloperRoleRecommendation(
                    role=role,
                    fit_score=fit_score,
                    rationale=rationale,
                )
            )

        if not positions:
            positions = _fallback_role_recommendations(request)

        strengths = [str(item).strip() for item in raw.get("strengths", []) if str(item).strip()]
        risks = [str(item).strip() for item in raw.get("risks", []) if str(item).strip()]
        action_plan = [str(item).strip() for item in raw.get("action_plan", []) if str(item).strip()]

        language_insights: list[DeveloperLanguageInsight] = []
        for item in raw.get("language_insights", []):
            language = str(item.get("language", "")).strip()
            evidence = str(item.get("evidence", "")).strip()
            if not language or not evidence:
                continue
            confidence = int(item.get("confidence", 0))
            topics = [
                str(topic).strip()
                for topic in item.get("interview_topics", [])
                if str(topic).strip()
            ]
            language_insights.append(
                DeveloperLanguageInsight(
                    language=language,
                    confidence=max(0, min(100, confidence)),
                    evidence=evidence,
                    interview_topics=topics[:6],
                )
            )

        interview_tracks: list[DeveloperInterviewTrack] = []
        for item in raw.get("interview_tracks", []):
            role = str(item.get("role", "")).strip()
            mode = str(item.get("mode", "")).strip() or "practice"
            level = str(item.get("level", "")).strip() or "Middle"
            rationale = str(item.get("rationale", "")).strip()
            if not role or not rationale:
                continue

            focus_areas = [
                str(value).strip()
                for value in item.get("focus_areas", [])
                if str(value).strip()
            ]
            primary_skills = [
                str(value).strip()
                for value in item.get("primary_skills", [])
                if str(value).strip()
            ]
            duration_minutes = int(item.get("duration_minutes", 30))

            interview_tracks.append(
                DeveloperInterviewTrack(
                    role=role,
                    mode=mode,
                    level=level,
                    duration_minutes=max(10, min(120, duration_minutes)),
                    focus_areas=focus_areas[:8],
                    primary_skills=primary_skills[:10],
                    rationale=rationale,
                )
            )

        if not strengths:
            strengths = [
                "Есть практическая инженерная активность в публичных репозиториях.",
                "Профиль содержит сигналы для технической оценки на интервью.",
            ]
        if not risks:
            risks = ["Ограниченный контекст: часть навыков может быть не отражена в публичных репозиториях."]
        if not action_plan:
            action_plan = [
                "Начните с практического интервью по ключевому стеку из топ-репозиториев.",
                "Отработайте объяснение архитектурных решений и trade-offs на примерах своих проектов.",
            ]
        if not language_insights:
            language_insights = [
                DeveloperLanguageInsight(
                    language=(request.language_distribution[0].label if request.language_distribution else "General"),
                    confidence=64,
                    evidence="Язык встречается в активных репозиториях и отражен в недавней активности.",
                    interview_topics=["архитектурные компромиссы", "тестирование", "производительность"],
                )
            ]
        if not interview_tracks:
            top_role = positions[0].role if positions else "Software Engineer"
            interview_tracks = [
                DeveloperInterviewTrack(
                    role=top_role,
                    mode="practice",
                    level="Middle",
                    duration_minutes=35,
                    focus_areas=["практическая реализация", "разбор trade-offs", "production reliability"],
                    primary_skills=[item.label for item in (request.language_distribution or [])[:3]] or ["алгоритмы", "системный дизайн"],
                    rationale="Track выстроен по наиболее сильному стеку и типу задач в публичных репозиториях.",
                )
            ]

        summary = str(raw.get("summary", "")).strip()
        if not summary:
            summary = (
                "Профиль демонстрирует рабочую инженерную активность и подходит для технического интервью "
                "по ролям, близким к текущему стеку."
            )

        return DeveloperInsightsResponse(
            summary=summary,
            strengths=strengths[:6],
            risks=risks[:5],
            action_plan=action_plan[:6],
            language_insights=language_insights[:6],
            interview_tracks=interview_tracks[:4],
            recommended_positions=positions[:5],
        )
    except Exception:
        logger.exception("developer insights generation failed, fallback will be used")
        return DeveloperInsightsResponse(
            summary=(
                "Не удалось получить полный AI-анализ, но базовая оценка показывает "
                "потенциал для собеседований по инженерным ролям."
            ),
            strengths=[
                "Есть данные о репозиториях и активности для предварительного профилирования.",
                "Публичный профиль позволяет сформировать стартовый список интервью-ролей.",
            ],
            risks=[
                "Оценка частичная: не учтены приватные репозитории и командные артефакты.",
            ],
            action_plan=[
                "Выберите интервью по наиболее активному стеку и начните с режима practice.",
                "Подготовьте примеры решений с объяснением компромиссов и метрик качества.",
            ],
            language_insights=[
                DeveloperLanguageInsight(
                    language=(request.language_distribution[0].label if request.language_distribution else "General"),
                    confidence=60,
                    evidence="Определено по языковому распределению репозиториев.",
                    interview_topics=["архитектура", "тестирование", "отладка"],
                )
            ],
            interview_tracks=[
                DeveloperInterviewTrack(
                    role="Backend Engineer",
                    mode="practice",
                    level="Middle",
                    duration_minutes=30,
                    focus_areas=["кодинг", "проектирование API", "надежность"],
                    primary_skills=[item.label for item in (request.language_distribution or [])[:3]] or ["алгоритмы", "системное мышление"],
                    rationale="Fallback-track построен на доступной языковой статистике профиля.",
                )
            ],
            recommended_positions=_fallback_role_recommendations(request),
        )


@router.post(
    "/interviewer/next-question",
    response_model=NextQuestionResponse,
    responses={400: {"model": ErrorResponse}, 500: {"model": ErrorResponse}},
)
async def interviewer_next_question(
    request: NextQuestionRequest,
    container: Annotated[DIContainer, Depends(_get_container)],
) -> NextQuestionResponse:
    """Generate next strict interviewer question without teaching or solutions."""
    last_answer = (request.last_candidate_answer or "").strip()
    templates = container.get_prompt_templates()
    llm = container.get_llm_client()
    interviewer_template = templates.get_template("interviewer_turn")
    prompt, system_prompt = interviewer_template.format(
        role=request.role,
        vacancy_title=request.vacancy_title or request.role,
        vacancy_category=request.vacancy_category or request.role,
        interview_mode=request.interview_mode or "practice",
        mode_guidance=_interview_mode_guidance(request.interview_mode),
        session_context=_build_session_context(request),
        answer_guidance=_answer_guidance(last_answer, request.current_topic or request.role),
        response_policy=_response_policy(last_answer, request.current_topic or request.role),
        practice_focus=_format_list_block(request.practice_focus or []),
        theory_focus=_format_list_block(request.theory_focus or []),
        primary_skills=_format_list_block(request.primary_skills or []),
        focus_areas=_format_list_block(request.focus_areas or []),
        level=request.level,
        style_profile=_interviewer_style_profile(request.role, request.level),
        topic=request.current_topic or "general",
        difficulty=str(request.difficulty),
        pressure=str(request.pressure_level),
        time_left_sec=str(request.time_left_sec),
        questions_left=str(request.questions_left),
        turn_nonce=request.turn_nonce or "none",
        history=_format_interviewer_history(request.history or []),
        avoid_questions=_format_avoid_questions(request.avoid_questions or []),
        candidate_answer=last_answer or "(первый вопрос)",
    )

    accepted_question: str | None = None
    accepted_topic = request.current_topic or "core"
    accepted_delta = 0
    accepted_pressure = request.pressure_level
    accepted_flags = {
        "contains_explanation": False,
        "contains_solution": False,
        "policy_violation": False,
    }
    answer_profile = _inspect_last_answer(last_answer, request.current_topic or request.role)

    regen_reason = ""
    max_attempts = 3

    try:
        for attempt in range(1, max_attempts + 1):
            attempt_prompt = prompt
            if regen_reason:
                attempt_prompt += f"\n\nПредыдущий ответ отклонен: {regen_reason}. Сгенерируй корректный новый вопрос."

            raw = await llm.generate_json(
                prompt=attempt_prompt,
                system_prompt=system_prompt,
                schema=_INTERVIEW_OUTPUT_SCHEMA,
            )

            question = str(raw.get("question", "")).strip()
            topic = str(raw.get("topic", accepted_topic)).strip() or accepted_topic
            difficulty_delta = int(raw.get("difficulty_delta", 0))
            pressure_level = int(raw.get("pressure_level", request.pressure_level))
            llm_flags = raw.get("flags", {})

            is_valid, violations, sanitized = _sanitize_interviewer_question(question)
            derived_flags = _policy_flags_from_text(question)
            merged_flags = {
                "contains_explanation": bool(llm_flags.get("contains_explanation", False) or derived_flags["contains_explanation"]),
                "contains_solution": bool(llm_flags.get("contains_solution", False) or derived_flags["contains_solution"]),
                "policy_violation": bool(llm_flags.get("policy_violation", False) or derived_flags["policy_violation"]),
            }

            if _is_avoided_question(sanitized, request.avoid_questions or [], request.role):
                is_valid = False
                violations.append("duplicate_from_avoid_list")

            if is_valid:
                accepted_question = sanitized
                accepted_topic = topic
                accepted_delta = _blend_difficulty_delta(difficulty_delta, answer_profile)
                accepted_pressure = _blend_pressure_level(pressure_level, request.pressure_level, answer_profile)
                accepted_flags = merged_flags
                accepted_flags["answer_was_weak"] = bool(answer_profile["is_weak"])
                accepted_flags["answer_was_partial"] = bool(answer_profile["is_partial"])
                break

            regen_reason = ",".join(violations) if violations else "invalid_format"
            logger.warning(
                "interviewer response rejected on attempt %d/%d: %s",
                attempt,
                max_attempts,
                regen_reason,
            )
    except Exception:
        logger.exception("interviewer generation failed, fallback will be used")

    if not accepted_question:
        accepted_question = _fallback_question(request, answer_profile)
        accepted_topic = request.current_topic or "core"
        accepted_delta = _blend_difficulty_delta(0, answer_profile)
        accepted_pressure = _blend_pressure_level(request.pressure_level, request.pressure_level, answer_profile)
        accepted_flags = {
            "contains_explanation": False,
            "contains_solution": False,
            "policy_violation": False,
            "answer_was_weak": bool(answer_profile["is_weak"]),
            "answer_was_partial": bool(answer_profile["is_partial"]),
        }

    if request.difficulty >= 7:
        follow_up = _hard_mode_follow_up(request.role, accepted_topic, accepted_question)
        if follow_up:
            accepted_question = accepted_question.rstrip("?.!") + ". " + follow_up

    should_end = request.questions_left <= 1 or request.time_left_sec <= 30
    pressure = min(5, max(1, accepted_pressure + (1 if request.difficulty >= 6 else 0)))
    difficulty_delta = accepted_delta

    return NextQuestionResponse(
        question=accepted_question,
        topic=accepted_topic,
        difficulty_delta=difficulty_delta,
        pressure_level=pressure,
        should_end=should_end,
        flags=accepted_flags,
    )


@router.post(
    "/interviewer/validate-output",
    response_model=ValidateOutputResponse,
    responses={400: {"model": ErrorResponse}},
)
async def interviewer_validate_output(
    request: ValidateOutputRequest,
) -> ValidateOutputResponse:
    """Validate interviewer output by format, size, and anti-repeat rules."""
    is_valid, violations, sanitized = _sanitize_interviewer_question(
        request.draft_response
    )

    if _is_avoided_question(
        sanitized,
        request.avoid_questions or [],
        request.role or "",
        strict=True,
    ):
        is_valid = False
        violations.append("semantic_duplicate")

    if _topic_repetition_detected(
        sanitized,
        request.current_topic or "",
        request.recent_topics or [],
    ):
        is_valid = False
        violations.append("topic_repeat")

    return ValidateOutputResponse(
        is_valid=is_valid,
        violations=violations,
        sanitized_question=sanitized,
    )


@router.post(
    "/interviewer/post-analysis",
    response_model=PostAnalysisResponse,
    responses={400: {"model": ErrorResponse}},
)
async def interviewer_post_analysis(
    request: PostAnalysisRequest,
    container: Annotated[DIContainer, Depends(_get_container)],
) -> PostAnalysisResponse:
    """Compute post-interview scores from the full conversation history."""
    service = container.get_analysis_service()
    turn_pairs = _pair_interview_turns(request.messages)

    if not turn_pairs:
        return PostAnalysisResponse(
            session_id=request.session_id,
            correctness=0,
            clarity=0,
            completeness=0,
            relevance=0,
            overall_score=0,
            strengths=[],
            weaknesses=["Нет ответов кандидата для анализа"],
            recommendations=["Пройдите интервью до конца для формирования отчета"],
        )

    turn_scores: list[dict[str, float]] = []
    for question_text, answer_text in turn_pairs:
        if not answer_text.strip():
            continue
        turn_scores.append(
            await _score_interview_turn(
                service=service,
                question=question_text,
                answer=answer_text,
                level=request.level,
            )
        )

    if not turn_scores:
        return PostAnalysisResponse(
            session_id=request.session_id,
            correctness=0,
            clarity=0,
            completeness=0,
            relevance=0,
            overall_score=0,
            strengths=[],
            weaknesses=["Ответы кандидата пустые или не содержат анализируемого текста"],
            recommendations=["Попросите кандидата отвечать развернуто и по сути вопроса"],
        )

    correctness = _average_score(turn_scores, "correctness")
    clarity = _average_score(turn_scores, "clarity")
    completeness = _average_score(turn_scores, "completeness")
    relevance = _average_score(turn_scores, "relevance")
    overall = round((correctness + clarity + completeness + relevance) / 4.0, 2)

    strengths = _build_post_analysis_strengths(
        correctness=correctness,
        clarity=clarity,
        completeness=completeness,
        relevance=relevance,
    )
    weaknesses = _build_post_analysis_weaknesses(
        correctness=correctness,
        clarity=clarity,
        completeness=completeness,
        relevance=relevance,
        turn_pairs=turn_pairs,
    )
    recommendations = _build_post_analysis_recommendations(
        correctness=correctness,
        clarity=clarity,
        completeness=completeness,
        relevance=relevance,
        turn_pairs=turn_pairs,
    )

    return PostAnalysisResponse(
        session_id=request.session_id,
        correctness=correctness,
        clarity=clarity,
        completeness=completeness,
        relevance=relevance,
        overall_score=overall,
        strengths=strengths,
        weaknesses=weaknesses,
        recommendations=recommendations,
    )


def _pair_interview_turns(messages: list[InterviewHistoryMessage]) -> list[tuple[str, str]]:
    pairs: list[tuple[str, str]] = []
    last_ai_question = ""

    for message in messages:
        sender = (message.sender or "").strip().lower()
        content = (message.content or "").strip()
        if sender == "ai":
            last_ai_question = content
            continue
        if sender == "user":
            pairs.append((last_ai_question, content))

    return pairs


async def _score_interview_turn(service, question: str, answer: str, level: str) -> dict[str, float]:
    answer_text = answer.strip()
    question_text = question.strip()
    tokens = _tokenize(answer_text)
    word_count = len(tokens)
    unique_ratio = _unique_ratio(tokens)
    structure_score = _structure_score(answer_text)
    evidence_score = _evidence_score(answer_text)
    deflective = _contains_deflection(answer_text)

    if question_text:
        try:
            relevance_similarity = await service.compute_similarity(question_text, answer_text)
        except Exception:
            relevance_similarity = _lexical_overlap(question_text, answer_text)
    else:
        relevance_similarity = 0.35 if answer_text else 0.0

    semantic_alignment = relevance_similarity * 100
    length_signal = min(100.0, word_count * 4.5)
    level_factor = 1.0 if level.lower() in {"junior", "middle"} else 1.08

    correctness = _clamp(
        semantic_alignment * 0.6 + evidence_score * 25 + length_signal * 0.15 - (18 if deflective else 0),
    )
    clarity = _clamp(
        25 + structure_score * 40 + unique_ratio * 20 + min(25.0, length_signal * 0.2) - (8 if deflective else 0),
    )
    completeness = _clamp(
        semantic_alignment * 0.35 + min(60.0, length_signal) + evidence_score * 18 - (15 if deflective else 0),
    )
    relevance = _clamp(
        semantic_alignment * 0.75 + evidence_score * 10 - (18 if deflective else 0),
    )

    if word_count < 6:
        correctness = min(correctness, 35.0)
        clarity = min(clarity, 45.0)
        completeness = min(completeness, 30.0)
        relevance = min(relevance, 40.0)

    if "не знаю" in answer_text.lower() or "затрудняюсь" in answer_text.lower():
        correctness = min(correctness, 25.0)
        completeness = min(completeness, 25.0)
        relevance = min(relevance, 30.0)

    if level_factor > 1.0:
        correctness *= 0.98
        completeness *= 0.97

    return {
        "correctness": round(_clamp(correctness), 2),
        "clarity": round(_clamp(clarity), 2),
        "completeness": round(_clamp(completeness), 2),
        "relevance": round(_clamp(relevance), 2),
    }


def _average_score(turn_scores: list[dict[str, float]], field: str) -> float:
    if not turn_scores:
        return 0.0
    return round(sum(score[field] for score in turn_scores) / len(turn_scores), 2)


def _build_post_analysis_strengths(
    *,
    correctness: float,
    clarity: float,
    completeness: float,
    relevance: float,
) -> list[str]:
    strengths: list[str] = []
    if relevance >= 70:
        strengths.append("Ответы остаются в фокусе вопроса")
    if correctness >= 70:
        strengths.append("Есть уверенное понимание базовых концепций")
    if completeness >= 70:
        strengths.append("Кандидат раскрывает ответы достаточно полно")
    if clarity >= 70:
        strengths.append("Ответы в целом структурированы")
    if all(metric >= 80 for metric in (correctness, clarity, completeness, relevance)):
        strengths.append("Кандидат стабильно держит уровень и может обсуждать сложные trade-offs")
    return strengths


def _build_post_analysis_weaknesses(
    *,
    correctness: float,
    clarity: float,
    completeness: float,
    relevance: float,
    turn_pairs: list[tuple[str, str]],
) -> list[str]:
    weaknesses: list[str] = []
    short_answers = sum(1 for _, answer in turn_pairs if len(answer.split()) < 6)
    deflection_hits = sum(1 for _, answer in turn_pairs if _contains_deflection(answer))

    if relevance < 60:
        weaknesses.append("Часть ответов уходит в сторону от заданного вопроса")
    if correctness < 60:
        weaknesses.append("Не хватает точности или уверенности в фактах и выводах")
    if completeness < 60:
        weaknesses.append("Ответы часто недостаточно раскрывают тему")
    if clarity < 60:
        weaknesses.append("Формулировки местами слишком размытые или несистемные")
    if short_answers >= max(2, len(turn_pairs) // 2):
        weaknesses.append("Слишком много коротких ответов без объяснения reasoning")
    if deflection_hits > 0:
        weaknesses.append("Есть уклончивые ответы вместо прямого решения")
    if deflection_hits >= max(2, len(turn_pairs) // 2):
        weaknesses.append("Кандидат часто избегает конкретики под давлением интервью")
    return weaknesses or ["Критичных слабых мест не найдено"]


def _build_post_analysis_recommendations(
    *,
    correctness: float,
    clarity: float,
    completeness: float,
    relevance: float,
    turn_pairs: list[tuple[str, str]],
) -> list[str]:
    recommendations: list[str] = []
    short_answers = sum(1 for _, answer in turn_pairs if len(answer.split()) < 6)
    deflection_hits = sum(1 for _, answer in turn_pairs if _contains_deflection(answer))

    if relevance < 60:
        recommendations.append("Сначала отвечайте на сам вопрос, затем добавляйте детали и примеры")
    if correctness < 60:
        recommendations.append("Проверяйте факты и проговаривайте допущения")
    if completeness < 60:
        recommendations.append("Расширяйте ответы: критерии выбора, trade-offs и edge cases")
    if clarity < 60:
        recommendations.append("Структурируйте ответ: тезис, аргументы, вывод")
    if short_answers >= max(2, len(turn_pairs) // 2):
        recommendations.append("Давайте ответ в 3 шага: решение, почему оно выбрано, как проверите результат")
    if deflection_hits > 0:
        recommendations.append("Если не уверены, озвучивайте гипотезу и план проверки, а не уходите в общие фразы")
    if len(turn_pairs) < 3:
        recommendations.append("Продлите интервью, чтобы накопить больше данных для оценки")
    if all(metric >= 75 for metric in (correctness, clarity, completeness, relevance)):
        recommendations.append("Добавьте продвинутый уровень: обсуждайте failure modes, метрики и стратегию отката")
    return recommendations or ["Сохраняйте текущий уровень и добавляйте больше конкретики"]


def _contains_deflection(text: str) -> bool:
    lowered = text.lower()
    return any(
        marker in lowered
        for marker in (
            "не знаю",
            "не уверен",
            "затрудняюсь",
            "кажется",
            "примерно",
            "скорее всего",
            "не помню",
        )
    )


def _evidence_score(text: str) -> float:
    lowered = text.lower()
    markers = (
        "например",
        "потому что",
        "то есть",
        "сначала",
        "затем",
        "в итоге",
        "trade-off",
        "компромисс",
        "edge",
        "ошибка",
        "риск",
        "зависимость",
        "асинхрон",
        "кэш",
        "транзакц",
        "сложност",
    )
    hits = sum(1 for marker in markers if marker in lowered)
    return min(1.0, hits / 4.0)


def _tokenize(text: str) -> list[str]:
    return re.findall(r"[A-Za-zА-Яа-я0-9_]+", text.lower())


def _unique_ratio(tokens: list[str]) -> float:
    if not tokens:
        return 0.0
    return len(set(tokens)) / len(tokens)


def _structure_score(text: str) -> float:
    if not text:
        return 0.0
    paragraphs = text.count("\n")
    bullets = text.count("-") + text.count("*")
    punctuation = sum(text.count(char) for char in [".", ",", ";", ":"])
    raw = 0.25 * paragraphs + 0.25 * bullets + min(1.0, punctuation / max(1.0, len(text) / 40))
    return min(1.0, raw)


def _lexical_overlap(text_a: str, text_b: str) -> float:
    tokens_a = set(_tokenize(text_a))
    tokens_b = set(_tokenize(text_b))
    if not tokens_a or not tokens_b:
        return 0.0
    return len(tokens_a & tokens_b) / len(tokens_a | tokens_b)


def _clamp(value: float) -> float:
    return max(0.0, min(100.0, float(value)))


@router.post("/embeddings/question")
async def get_question_embedding(
    request: dict,
) -> dict:
    """Get embedding vector for a question. Used for semantic similarity checking."""
    question = request.get("question", "").strip()
    if not question:
        raise HTTPException(status_code=400, detail="Question text required")

    try:
        embedding_client = get_local_embedding_client()
        embedding = embedding_client.embed(question)
        if embedding is None:
            raise HTTPException(status_code=503, detail="Embedding model unavailable")
        
        return {
            "question": question,
            "embedding": embedding,
            "dimension": len(embedding),
        }
    except HTTPException:
        raise
    except Exception as exc:
        logger.exception("Embedding generation failed")
        raise HTTPException(status_code=500, detail="Internal server error") from exc


@router.post("/embeddings/compare")
async def compare_question_similarity(
    request: dict,
) -> dict:
    """Compare semantic similarity between two questions."""
    question1 = request.get("question1", "").strip()
    question2 = request.get("question2", "").strip()
    role = request.get("role", "")
    
    if not question1 or not question2:
        raise HTTPException(status_code=400, detail="Both questions required")

    try:
        embedding_client = get_local_embedding_client()
        emb1 = embedding_client.embed(question1)
        emb2 = embedding_client.embed(question2)
        
        if emb1 is None or emb2 is None:
            raise HTTPException(status_code=503, detail="Embedding model unavailable")
        
        similarity = embedding_client.cosine_similarity(emb1, emb2)
        threshold = _semantic_threshold_for_role(str(role))
        
        return {
            "question1": question1,
            "question2": question2,
            "similarity": round(similarity, 4),
            "threshold": threshold,
            "is_duplicate": similarity > threshold,
        }
    except HTTPException:
        raise
    except Exception as exc:
        logger.exception("Similarity comparison failed")
        raise HTTPException(status_code=500, detail="Internal server error") from exc
