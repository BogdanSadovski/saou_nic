"""Direct postgres access to telegram_links + users.

Бот владеет таблицей telegram_links и читает users только для отображения
имени в /stats. Никаких записей в users.
"""

from __future__ import annotations

import asyncio
import logging
import secrets
import uuid
from dataclasses import dataclass
from datetime import datetime, timedelta, timezone
from typing import Optional

import asyncpg

from .config import settings

logger = logging.getLogger(__name__)

_pool: Optional[asyncpg.Pool] = None
_pool_lock = asyncio.Lock()


async def get_pool() -> asyncpg.Pool:
    global _pool
    if _pool is None:
        async with _pool_lock:
            if _pool is None:
                _pool = await asyncpg.create_pool(
                    settings.pg_dsn,
                    min_size=1,
                    max_size=5,
                    command_timeout=10,
                )
                logger.info("postgres pool ready")
    return _pool


# ───────────────── Models ─────────────────


@dataclass
class TelegramLink:
    id: uuid.UUID
    user_id: uuid.UUID
    chat_id: Optional[int]
    tg_username: Optional[str]
    notifications_paused: bool
    daily_push_hour_utc: int
    last_pushed_at: Optional[datetime]
    linked_at: Optional[datetime]


@dataclass
class LinkedUser(TelegramLink):
    email: str
    role: str
    full_name: str


# ───────────────── Token issuance ─────────────────


async def issue_link_token(user_id: uuid.UUID, ttl_minutes: int = 30) -> str:
    """Создать или перезаписать одноразовый token для конкретного user_id.

    Returns:
        token string suitable for t.me/<bot>?start=<token>.
    """
    token = secrets.token_urlsafe(24)
    expires = datetime.now(timezone.utc) + timedelta(minutes=ttl_minutes)
    pool = await get_pool()
    async with pool.acquire() as conn:
        await conn.execute(
            """
            INSERT INTO telegram_links (user_id, link_token, link_token_expires_at, updated_at)
            VALUES ($1, $2, $3, NOW())
            ON CONFLICT (user_id) DO UPDATE
            SET link_token = EXCLUDED.link_token,
                link_token_expires_at = EXCLUDED.link_token_expires_at,
                updated_at = NOW()
            """,
            user_id,
            token,
            expires,
        )
    return token


# ───────────────── /start binding ─────────────────


async def bind_chat_to_token(
    token: str, chat_id: int, tg_username: Optional[str]
) -> Optional[uuid.UUID]:
    """Ищет user по token, привязывает chat_id, обнуляет token.

    Returns:
        user_id если token валиден и не истёк, иначе None.
    """
    pool = await get_pool()
    async with pool.acquire() as conn:
        row = await conn.fetchrow(
            """
            UPDATE telegram_links
            SET chat_id = $2,
                tg_username = $3,
                link_token = NULL,
                link_token_expires_at = NULL,
                linked_at = NOW(),
                updated_at = NOW()
            WHERE link_token = $1
              AND link_token_expires_at > NOW()
            RETURNING user_id
            """,
            token,
            chat_id,
            tg_username,
        )
    return row["user_id"] if row else None


# ───────────────── chat_id lookups ─────────────────


async def get_link_by_chat(chat_id: int) -> Optional[LinkedUser]:
    pool = await get_pool()
    async with pool.acquire() as conn:
        row = await conn.fetchrow(
            """
            SELECT tl.id, tl.user_id, tl.chat_id, tl.tg_username,
                   tl.notifications_paused, tl.daily_push_hour_utc,
                   tl.last_pushed_at, tl.linked_at,
                   u.email, u.role,
                   COALESCE(NULLIF(TRIM(CONCAT(u.first_name, ' ', u.last_name)), ''),
                            u.username, u.email) AS full_name
            FROM telegram_links tl
            JOIN users u ON u.id = tl.user_id
            WHERE tl.chat_id = $1
            """,
            chat_id,
        )
    if row is None:
        return None
    return LinkedUser(**dict(row))


async def list_due_pushes(now_hour_utc: int) -> list[LinkedUser]:
    """Все привязанные юзеры, чей daily_push_hour_utc == now_hour_utc
    и кому сегодня ещё не отправляли."""
    pool = await get_pool()
    async with pool.acquire() as conn:
        rows = await conn.fetch(
            """
            SELECT tl.id, tl.user_id, tl.chat_id, tl.tg_username,
                   tl.notifications_paused, tl.daily_push_hour_utc,
                   tl.last_pushed_at, tl.linked_at,
                   u.email, u.role,
                   COALESCE(NULLIF(TRIM(CONCAT(u.first_name, ' ', u.last_name)), ''),
                            u.username, u.email) AS full_name
            FROM telegram_links tl
            JOIN users u ON u.id = tl.user_id
            WHERE tl.chat_id IS NOT NULL
              AND tl.daily_push_hour_utc = $1
              AND NOT tl.notifications_paused
              AND (tl.last_pushed_at IS NULL
                   OR tl.last_pushed_at < date_trunc('day', NOW() AT TIME ZONE 'UTC'))
            """,
            now_hour_utc,
        )
    return [LinkedUser(**dict(r)) for r in rows]


async def mark_pushed(link_id: uuid.UUID) -> None:
    pool = await get_pool()
    async with pool.acquire() as conn:
        await conn.execute(
            "UPDATE telegram_links SET last_pushed_at = NOW(), updated_at = NOW() WHERE id = $1",
            link_id,
        )


async def set_paused(chat_id: int, paused: bool) -> bool:
    pool = await get_pool()
    async with pool.acquire() as conn:
        row = await conn.fetchrow(
            """
            UPDATE telegram_links
            SET notifications_paused = $2, updated_at = NOW()
            WHERE chat_id = $1
            RETURNING id
            """,
            chat_id,
            paused,
        )
    return row is not None


async def unbind_chat(chat_id: int) -> bool:
    pool = await get_pool()
    async with pool.acquire() as conn:
        row = await conn.fetchrow(
            """
            UPDATE telegram_links
            SET chat_id = NULL, tg_username = NULL, linked_at = NULL, updated_at = NOW()
            WHERE chat_id = $1
            RETURNING id
            """,
            chat_id,
        )
    return row is not None


# ───────────────── Practice log (для /report и weekly digest) ─────────────────


async def log_practice(
    *,
    user_id: Optional[uuid.UUID],
    chat_id: int,
    topic: str,
    question: str,
    answer: str,
    score: Optional[float],
    verdict: Optional[str],
    feedback: Optional[str],
) -> None:
    """Пишем результат каждого оценённого ответа. Игнорируем ошибки,
    чтобы не ломать пользователю сценарий ответа."""
    pool = await get_pool()
    try:
        async with pool.acquire() as conn:
            await conn.execute(
                """
                INSERT INTO telegram_practice_log
                    (user_id, chat_id, topic, question, answer, score, verdict, feedback)
                VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
                """,
                user_id,
                chat_id,
                topic,
                question,
                answer,
                score,
                verdict,
                feedback,
            )
    except Exception as exc:  # noqa: BLE001
        logger.warning("practice log insert failed: %s", exc)


@dataclass
class PracticeSummary:
    days: int
    total: int
    avg_score: Optional[float]
    correct: int
    partial: int
    wrong: int
    by_topic: list[tuple[str, float, int]]  # (topic, avg_score, count)


async def summarize_practice(chat_id: int, days: int = 7) -> PracticeSummary:
    """Агрегаты по chat_id за последние `days` дней."""
    pool = await get_pool()
    async with pool.acquire() as conn:
        totals = await conn.fetchrow(
            """
            SELECT
                COUNT(*)::int                                                 AS total,
                AVG(score)::float                                             AS avg_score,
                COUNT(*) FILTER (WHERE verdict = 'correct')::int              AS correct,
                COUNT(*) FILTER (WHERE verdict = 'partial')::int              AS partial,
                COUNT(*) FILTER (WHERE verdict IN ('wrong','off_topic'))::int AS wrong
            FROM telegram_practice_log
            WHERE chat_id = $1
              AND created_at >= NOW() - ($2::int || ' days')::interval
            """,
            chat_id,
            days,
        )
        rows = await conn.fetch(
            """
            SELECT topic,
                   AVG(score)::float AS avg_score,
                   COUNT(*)::int     AS cnt
            FROM telegram_practice_log
            WHERE chat_id = $1
              AND created_at >= NOW() - ($2::int || ' days')::interval
              AND score IS NOT NULL
            GROUP BY topic
            ORDER BY cnt DESC, avg_score DESC
            """,
            chat_id,
            days,
        )
    return PracticeSummary(
        days=days,
        total=totals["total"] or 0,
        avg_score=totals["avg_score"],
        correct=totals["correct"] or 0,
        partial=totals["partial"] or 0,
        wrong=totals["wrong"] or 0,
        by_topic=[(r["topic"], float(r["avg_score"] or 0), r["cnt"]) for r in rows],
    )


# ───────────────── Streak counter ─────────────────


@dataclass
class StreakInfo:
    current: int
    best: int
    last_day: Optional[str]   # ISO date


async def bump_streak(user_id: uuid.UUID, chat_id: int) -> StreakInfo:
    """Вызывается на каждый успешный ответ. Подсчитывает серию подряд
    дней. Если вчера был ответ — +1, если сегодня уже был — без
    изменений, если разрыв > 1 дня — сброс на 1."""
    pool = await get_pool()
    async with pool.acquire() as conn:
        row = await conn.fetchrow(
            "SELECT streak_current, streak_best, streak_last_day FROM telegram_user_state WHERE user_id = $1",
            user_id,
        )
        from datetime import date, timedelta
        today = date.today()
        if row is None:
            cur, best = 1, 1
        else:
            cur = row["streak_current"] or 0
            best = row["streak_best"] or 0
            last = row["streak_last_day"]
            if last == today:
                pass  # уже считали сегодня
            elif last == today - timedelta(days=1):
                cur += 1
            else:
                cur = 1
            if cur > best:
                best = cur

        await conn.execute(
            """
            INSERT INTO telegram_user_state (user_id, chat_id, streak_current, streak_best, streak_last_day, updated_at)
            VALUES ($1, $2, $3, $4, $5, NOW())
            ON CONFLICT (user_id) DO UPDATE
            SET streak_current = EXCLUDED.streak_current,
                streak_best = EXCLUDED.streak_best,
                streak_last_day = EXCLUDED.streak_last_day,
                chat_id = EXCLUDED.chat_id,
                updated_at = NOW()
            """,
            user_id, chat_id, cur, best, today,
        )
        return StreakInfo(current=cur, best=best, last_day=today.isoformat())


async def get_streak(user_id: uuid.UUID) -> StreakInfo:
    pool = await get_pool()
    async with pool.acquire() as conn:
        row = await conn.fetchrow(
            "SELECT streak_current, streak_best, streak_last_day FROM telegram_user_state WHERE user_id = $1",
            user_id,
        )
    if row is None:
        return StreakInfo(current=0, best=0, last_day=None)
    last = row["streak_last_day"]
    # Если последний ответ был ДО вчера — сброс current до 0.
    from datetime import date, timedelta
    if last and last < date.today() - timedelta(days=1):
        return StreakInfo(current=0, best=row["streak_best"] or 0, last_day=last.isoformat())
    return StreakInfo(
        current=row["streak_current"] or 0,
        best=row["streak_best"] or 0,
        last_day=last.isoformat() if last else None,
    )


# ───────────────── Anti-repeat for term/news/history ─────────────────


async def mark_term_used(user_id: uuid.UUID, term_id: int) -> None:
    pool = await get_pool()
    async with pool.acquire() as conn:
        await conn.execute(
            """
            INSERT INTO telegram_user_state (user_id, used_term_ids, last_term_date, updated_at)
            VALUES ($1, ARRAY[$2]::int[], CURRENT_DATE, NOW())
            ON CONFLICT (user_id) DO UPDATE
            SET used_term_ids = (SELECT ARRAY(SELECT DISTINCT unnest(telegram_user_state.used_term_ids || ARRAY[$2]::int[])))[GREATEST(1, array_length(telegram_user_state.used_term_ids, 1) - 200):],
                last_term_date = CURRENT_DATE,
                updated_at = NOW()
            """,
            user_id, term_id,
        )


async def get_used_terms(user_id: uuid.UUID) -> set[int]:
    pool = await get_pool()
    async with pool.acquire() as conn:
        row = await conn.fetchrow(
            "SELECT used_term_ids FROM telegram_user_state WHERE user_id = $1",
            user_id,
        )
    if row is None or row["used_term_ids"] is None:
        return set()
    return set(row["used_term_ids"])


# ───────────────── Pomodoro ─────────────────


async def set_pomodoro(user_id: uuid.UUID, chat_id: int, until: 'datetime', phase: str) -> None:
    pool = await get_pool()
    async with pool.acquire() as conn:
        await conn.execute(
            """
            INSERT INTO telegram_user_state (user_id, chat_id, pomodoro_until, pomodoro_phase, updated_at)
            VALUES ($1, $2, $3, $4, NOW())
            ON CONFLICT (user_id) DO UPDATE
            SET chat_id = EXCLUDED.chat_id,
                pomodoro_until = EXCLUDED.pomodoro_until,
                pomodoro_phase = EXCLUDED.pomodoro_phase,
                updated_at = NOW()
            """,
            user_id, chat_id, until, phase,
        )


async def list_active_chats() -> list[LinkedUser]:
    """Все привязанные не-paused юзеры — для weekly digest."""
    pool = await get_pool()
    async with pool.acquire() as conn:
        rows = await conn.fetch(
            """
            SELECT tl.id, tl.user_id, tl.chat_id, tl.tg_username,
                   tl.notifications_paused, tl.daily_push_hour_utc,
                   tl.last_pushed_at, tl.linked_at,
                   u.email, u.role,
                   COALESCE(NULLIF(TRIM(CONCAT(u.first_name, ' ', u.last_name)), ''),
                            u.username, u.email) AS full_name
            FROM telegram_links tl
            JOIN users u ON u.id = tl.user_id
            WHERE tl.chat_id IS NOT NULL
              AND NOT tl.notifications_paused
            """
        )
    return [LinkedUser(**dict(r)) for r in rows]
