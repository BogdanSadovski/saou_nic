"""LLM-оценка soft-skill ответа.

Раньше тут был HTTP-клиент к отдельному softskills-service (PyTorch).
Сервис удалён — оценка теперь через тот же DeepSeek/OpenAI-совместимый
endpoint, что и весь остальной AI-функционал бота.

Возвращаемый контракт {score, verdict, feedback} сохранён — handlers
кода в боте не знают, что под капотом сменился источник.
"""

from __future__ import annotations

import json
import logging
import re
from typing import Optional

import httpx

from .config import settings

logger = logging.getLogger(__name__)


_SYSTEM_PROMPT = (
    "Ты опытный карьерный коуч / интервьюер. Оцени ответ кандидата на "
    "поведенческий (soft-skills) вопрос по шкале 0..100. Учитывай:\n"
    "  • STAR-структуру (situation/task/action/result)\n"
    "  • конкретику и метрики вместо общих фраз\n"
    "  • рефлексию (что узнал из ситуации, что бы сделал иначе)\n"
    "  • краткость без воды\n\n"
    "Верни строго JSON: {\"score\": <0-100>, "
    "\"verdict\": \"correct|partial|wrong|skipped|off_topic\", "
    "\"feedback\": \"<2-3 фразы по-русски: что сильно, что улучшить>\"}."
)


def _heuristic(answer: str) -> dict:
    """Fallback, когда LLM недоступен — простая длинная-строка эвристика."""
    a = (answer or "").strip()
    if not a:
        return {"score": 0, "verdict": "skipped",
                "feedback": "Ответ пустой — попробуйте ещё раз."}
    if len(a) < 40:
        return {"score": 25, "verdict": "wrong",
                "feedback": "Очень коротко. Опишите ситуацию подробнее по структуре STAR."}
    if len(a) < 200:
        return {"score": 55, "verdict": "partial",
                "feedback": "Есть основа, но не хватает деталей: добавьте конкретные действия и измеримый результат."}
    return {"score": 75, "verdict": "correct",
            "feedback": "Развёрнутый ответ. Подсветите измеримый итог и личный вывод из ситуации."}


def _extract_json(raw: str) -> Optional[dict]:
    """LLM иногда оборачивает ответ в ```json``` или болтает до/после.
    Достаём первый JSON-объект."""
    if not raw:
        return None
    clean = raw.strip()
    clean = re.sub(r"^```(?:json)?\s*", "", clean)
    clean = re.sub(r"\s*```$", "", clean)
    start = clean.find("{")
    end = clean.rfind("}")
    if start < 0 or end <= start:
        return None
    try:
        return json.loads(clean[start:end + 1])
    except Exception:  # noqa: BLE001
        return None


async def score_answer(question: str, answer: str) -> Optional[dict]:
    """Возвращает {score, verdict, feedback} или None при ошибке."""
    api_key = settings.llm_api_key
    api_url = settings.llm_base_url
    model = settings.llm_model
    if not api_key or not api_url:
        logger.warning("LLM not configured — using heuristic")
        return _heuristic(answer)

    user_prompt = (
        f"ВОПРОС:\n{question.strip()}\n\n"
        f"ОТВЕТ КАНДИДАТА:\n{(answer or '').strip()}"
    )

    try:
        async with httpx.AsyncClient(timeout=20) as client:
            resp = await client.post(
                api_url.rstrip("/") + "/chat/completions",
                headers={"Authorization": f"Bearer {api_key}"},
                json={
                    "model": model,
                    "messages": [
                        {"role": "system", "content": _SYSTEM_PROMPT},
                        {"role": "user", "content": user_prompt},
                    ],
                    "response_format": {"type": "json_object"},
                    "max_tokens": 400,
                    "temperature": 0.3,
                },
            )
        if resp.status_code != 200:
            logger.warning("LLM score returned %s: %s", resp.status_code, resp.text[:200])
            return _heuristic(answer)
        body = resp.json()
        content = (body.get("choices", [{}])[0].get("message", {}).get("content") or "").strip()
        parsed = _extract_json(content)
        if not parsed:
            return _heuristic(answer)
        # Нормализация
        score = parsed.get("score")
        try:
            score_val = float(score) if score is not None else None
        except (TypeError, ValueError):
            score_val = None
        verdict = (parsed.get("verdict") or "").lower().strip()
        if verdict not in {"correct", "partial", "wrong", "skipped", "off_topic"}:
            verdict = "partial"
        feedback = (parsed.get("feedback") or "").strip() or "Без подробного комментария."
        return {"score": score_val, "verdict": verdict, "feedback": feedback}
    except Exception as exc:  # noqa: BLE001
        logger.warning("LLM score request failed: %s — falling back to heuristic", exc)
        return _heuristic(answer)
