"""Контент-фичи бота: term-of-day, system-design челлендж, tech-history.

Все банки — статичные JSON. Ротация без повторов: для каждого юзера
храним set из «уже виденных id» в telegram_user_state.used_term_ids.
"""

from __future__ import annotations

import json
import logging
import random
from datetime import date
from pathlib import Path
from typing import Optional

logger = logging.getLogger(__name__)

_QUESTIONS_DIR = Path(__file__).resolve().parent.parent / "questions"


def _load_json(name: str):
    try:
        return json.loads((_QUESTIONS_DIR / name).read_text(encoding="utf-8"))
    except Exception as exc:  # noqa: BLE001
        logger.error("failed to load %s: %s", name, exc)
        return None


_TERMS = _load_json("terms.json") or []
_CHALLENGES = _load_json("system_design.json") or []
_HISTORY = _load_json("tech_history.json") or {}


def pick_term(used: set[int]) -> Optional[dict]:
    """Случайный термин, которого ещё не было у этого юзера."""
    if not _TERMS:
        return None
    available = [t for t in _TERMS if t["id"] not in used]
    if not available:
        # все видел — сбрасываем ротацию
        available = _TERMS
    return random.choice(available)


def render_term(t: dict) -> str:
    return (
        f"🧠 <b>{t['term']}</b>\n\n"
        f"<b>Что:</b> {t['what']}\n\n"
        f"<b>Пример:</b> {t.get('example', '—')}\n\n"
        f"<b>Где применяют:</b> {t.get('where', '—')}"
    )


def pick_challenge() -> Optional[dict]:
    """System-design челлендж — ротация недельная, привязка к дате."""
    if not _CHALLENGES:
        return None
    # Стабильно: одна задача на ISO-week, чтобы все юзеры в пятницу
    # получали одинаковую (можно обсуждать с друзьями).
    iso_year, iso_week, _ = date.today().isocalendar()
    idx = (iso_week + iso_year) % len(_CHALLENGES)
    return _CHALLENGES[idx]


def render_challenge(c: dict) -> str:
    crit = "\n".join(f"  • {x}" for x in c.get("criteria", []))
    return (
        f"🏗 <b>System Design Friday</b>\n"
        f"<b>{c['title']}</b>\n\n"
        f"{c['brief']}\n\n"
        f"<b>На что обратить внимание:</b>\n{crit}\n\n"
        f"Ответьте текстом (5-10 предложений). AI оценит ваш разбор."
    )


def history_for_today() -> Optional[str]:
    key = date.today().strftime("%m-%d")
    return _HISTORY.get(key)
