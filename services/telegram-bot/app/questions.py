"""Источник вопросов для daily-push и /question.

Сейчас — статичный JSON-банк. В будущем, когда появится spaced-repetition
таблица, заменим выбор на «худшая тема за неделю». Бэк изолируем здесь.
"""

from __future__ import annotations

import json
import logging
import random
from pathlib import Path
from typing import Optional

logger = logging.getLogger(__name__)

_BANK_PATH = Path(__file__).resolve().parent.parent / "questions" / "bank.json"
_BANK: dict[str, list[str]] = {}


def _load() -> dict[str, list[str]]:
    global _BANK
    if _BANK:
        return _BANK
    try:
        _BANK = json.loads(_BANK_PATH.read_text(encoding="utf-8"))
    except Exception as exc:  # noqa: BLE001
        logger.error("failed to load questions bank: %s", exc)
        _BANK = {"soft_skills": ["Расскажите о себе."]}
    return _BANK


def pick(topic: Optional[str] = None) -> tuple[str, str]:
    """Возвращает (topic, question)."""
    bank = _load()
    topics = list(bank.keys())
    if topic and topic in bank:
        chosen = topic
    else:
        chosen = random.choice(topics)
    pool = bank.get(chosen) or [""]
    return chosen, random.choice(pool)
