"""In-memory state «какой вопрос был последним для chat_id».

Лёгкая замена redis для прототипа — поскольку у нас один pod бота с
long-polling, race-conditions нет. При рестарте теряется — это ок:
юзер просто запросит /question заново.
"""

from __future__ import annotations

import threading
from typing import Optional

_lock = threading.Lock()
_state: dict[int, tuple[str, str]] = {}  # chat_id -> (topic, question)


def remember_question(chat_id: int, topic: str, question: str) -> None:
    with _lock:
        _state[chat_id] = (topic, question)


def get_last_question(chat_id: int) -> Optional[tuple[str, str]]:
    with _lock:
        return _state.get(chat_id)


def clear(chat_id: int) -> None:
    with _lock:
        _state.pop(chat_id, None)
