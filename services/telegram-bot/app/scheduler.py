"""Простейший почасовой scheduler без отдельных зависимостей.

Раз в N минут (по умолчанию 5) смотрим текущий UTC-час; для всех
телеграм-линков, где `daily_push_hour_utc == now_utc.hour` и сегодня
ещё не было пуша — отправляем вопрос и помечаем last_pushed_at.
"""

from __future__ import annotations

import asyncio
import logging
from datetime import datetime, timezone

from aiogram import Bot

from . import content, db, news, questions
from .handlers import render_summary
from .session_state import remember_question

logger = logging.getLogger(__name__)


# Регулярные «контентные» отправки:
#   • 06:00 UTC — daily-push с вопросом (по user-настройке)
#   • 06:30 UTC — IT-news + Tech-history дайджест
#   • 17:00 UTC — Term of the Day
#   • Воскресенье 18:00 UTC — недельная сводка
#   • Пятница 12:00 UTC — System Design Challenge

_WEEKLY_DOW = 6        # Monday=0, Sunday=6
_WEEKLY_HOUR_UTC = 18
_NEWS_HOUR_UTC = 7     # 10:00 МСК
_TERM_HOUR_UTC = 17    # 20:00 МСК
_CHALLENGE_DOW = 4     # Friday
_CHALLENGE_HOUR_UTC = 12

_last_weekly_run_date: str | None = None
_last_news_date: str | None = None
_last_term_date: str | None = None
_last_challenge_date: str | None = None


async def run_daily_loop(bot: Bot, poll_interval_sec: int = 300) -> None:
    logger.info(
        "scheduler started (every %ss): daily-push, news @%d:00, term @%d:00, "
        "weekly @ Sun %d:00, challenge @ Fri %d:00 UTC",
        poll_interval_sec, _NEWS_HOUR_UTC, _TERM_HOUR_UTC,
        _WEEKLY_HOUR_UTC, _CHALLENGE_HOUR_UTC,
    )
    while True:
        try:
            await _tick(bot)
            await _weekly_tick(bot)
            await _news_tick(bot)
            await _term_tick(bot)
            await _challenge_tick(bot)
        except Exception as exc:  # noqa: BLE001
            logger.exception("scheduler tick failed: %s", exc)
        await asyncio.sleep(poll_interval_sec)


async def _tick(bot: Bot) -> None:
    now = datetime.now(timezone.utc)
    due = await db.list_due_pushes(now.hour)
    if not due:
        return
    logger.info("daily-push tick: %d users due", len(due))

    for link in due:
        try:
            topic, q = questions.pick()
            await bot.send_message(
                chat_id=link.chat_id,
                text=(
                    f"🌅 Доброе утро, {link.full_name}!\n"
                    f"Вопрос дня — тема <b>{topic}</b>:\n\n{q}\n\n"
                    "Ответьте текстом — я оценю."
                ),
                parse_mode="HTML",
            )
            remember_question(link.chat_id, topic, q)
            await db.mark_pushed(link.id)
        except Exception as exc:  # noqa: BLE001
            logger.warning("push to chat %s failed: %s", link.chat_id, exc)


async def _weekly_tick(bot: Bot) -> None:
    """Если сейчас вс 18:00 UTC и сегодня ещё не отправляли — шлём
    еженедельную сводку всем активным юзерам."""
    global _last_weekly_run_date
    now = datetime.now(timezone.utc)
    if now.weekday() != _WEEKLY_DOW or now.hour != _WEEKLY_HOUR_UTC:
        return

    today_key = now.strftime("%Y-%m-%d")
    if _last_weekly_run_date == today_key:
        return  # уже отправили в этот час

    chats = await db.list_active_chats()
    if not chats:
        _last_weekly_run_date = today_key
        return

    logger.info("weekly digest tick: sending to %d users", len(chats))
    for link in chats:
        try:
            summary = await db.summarize_practice(link.chat_id, days=7)
            text = render_summary(
                summary,
                header=f"📅 <b>Итоги недели, {link.full_name}</b>",
            )
            await bot.send_message(chat_id=link.chat_id, text=text, parse_mode="HTML")
        except Exception as exc:  # noqa: BLE001
            logger.warning("weekly digest to chat %s failed: %s", link.chat_id, exc)

    _last_weekly_run_date = today_key


async def _news_tick(bot: Bot) -> None:
    """Утренний IT-дайджест: 07:00 UTC. Один общий summary на день,
    рассылается всем активным юзерам."""
    global _last_news_date
    now = datetime.now(timezone.utc)
    if now.hour != _NEWS_HOUR_UTC:
        return
    today_key = now.strftime("%Y-%m-%d")
    if _last_news_date == today_key:
        return

    chats = await db.list_active_chats()
    if not chats:
        _last_news_date = today_key
        return

    logger.info("news tick: %d users", len(chats))
    # Один LLM-вызов на всех — экономим токены.
    items = await news.collect_news(per_source=4)
    digest = await news.summarize_with_llm(items, limit=5)
    history = content.history_for_today()
    if history:
        digest += f"\n\n📅 <i>{history}</i>"

    for link in chats:
        try:
            await bot.send_message(chat_id=link.chat_id, text=digest,
                                   parse_mode="HTML", disable_web_page_preview=True)
        except Exception as exc:  # noqa: BLE001
            logger.warning("news push to %s failed: %s", link.chat_id, exc)
    _last_news_date = today_key


async def _term_tick(bot: Bot) -> None:
    """Term of the day, 17:00 UTC. Для каждого юзера — свой случайный
    термин из тех, что он ещё не видел."""
    global _last_term_date
    now = datetime.now(timezone.utc)
    if now.hour != _TERM_HOUR_UTC:
        return
    today_key = now.strftime("%Y-%m-%d")
    if _last_term_date == today_key:
        return

    chats = await db.list_active_chats()
    if not chats:
        _last_term_date = today_key
        return

    logger.info("term tick: %d users", len(chats))
    for link in chats:
        try:
            used = await db.get_used_terms(link.user_id)
            term = content.pick_term(used)
            if not term:
                continue
            await db.mark_term_used(link.user_id, term["id"])
            await bot.send_message(
                chat_id=link.chat_id,
                text="🧠 <b>Термин дня</b>\n\n" + content.render_term(term),
                parse_mode="HTML",
            )
        except Exception as exc:  # noqa: BLE001
            logger.warning("term push to %s failed: %s", link.chat_id, exc)
    _last_term_date = today_key


async def _challenge_tick(bot: Bot) -> None:
    """System Design Friday: пятница 12:00 UTC. Один общий challenge."""
    global _last_challenge_date
    now = datetime.now(timezone.utc)
    if now.weekday() != _CHALLENGE_DOW or now.hour != _CHALLENGE_HOUR_UTC:
        return
    today_key = now.strftime("%Y-%m-%d")
    if _last_challenge_date == today_key:
        return

    chats = await db.list_active_chats()
    ch = content.pick_challenge()
    if not chats or not ch:
        _last_challenge_date = today_key
        return

    logger.info("challenge tick: %d users", len(chats))
    text = content.render_challenge(ch)
    for link in chats:
        try:
            await bot.send_message(chat_id=link.chat_id, text=text, parse_mode="HTML")
            remember_question(link.chat_id, "system_design", ch["brief"])
        except Exception as exc:  # noqa: BLE001
            logger.warning("challenge push to %s failed: %s", link.chat_id, exc)
    _last_challenge_date = today_key
