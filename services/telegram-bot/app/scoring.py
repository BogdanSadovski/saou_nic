"""Клиент к softskills-service для оценки ответа.

Бэк уже умеет /api/v1/score (см. services/softskills-service). Здесь —
тонкая обёртка с таймаутом, чтобы бот не висел в polling-loop.
"""

from __future__ import annotations

import logging
from typing import Optional

import httpx

from .config import settings

logger = logging.getLogger(__name__)


async def score_answer(question: str, answer: str) -> Optional[dict]:
    """Возвращает {score, verdict, feedback} или None при ошибке."""
    payload = {"question": question, "answer": answer}
    url = settings.softskills_service_url.rstrip("/") + "/api/v1/score"
    try:
        async with httpx.AsyncClient(timeout=15) as client:
            resp = await client.post(url, json=payload)
        if resp.status_code != 200:
            logger.warning(
                "softskills score returned %s: %s",
                resp.status_code, resp.text[:200],
            )
            return None
        return resp.json()
    except Exception as exc:  # noqa: BLE001
        logger.warning("softskills score request failed: %s", exc)
        return None
