"""Обработчики команд, сообщений, callback'ов и inline-режима бота."""

from __future__ import annotations

import asyncio
import logging
import re
from datetime import datetime, timedelta, timezone
from typing import Optional

import httpx
from aiogram import Router, F
from aiogram.filters import Command, CommandObject, CommandStart
from aiogram.types import (
    CallbackQuery,
    InlineQuery,
    InlineQueryResultArticle,
    InputTextMessageContent,
    Message,
)

from . import content, db, news, questions, scoring
from .auth import auth_headers
from .config import settings
from .keyboards import (
    after_answer_keyboard,
    after_question_keyboard,
    menu_keyboard,
    upgrade_keyboard,
)
from .session_state import remember_question, get_last_question

logger = logging.getLogger(__name__)

router = Router(name="realsync")


async def _score_system_design(question: str, answer: str) -> Optional[dict]:
    """LLM-оценка system-design ответа по 4 критериям.

    Возвращает тот же контракт, что и /softskills/score:
    {score, verdict, feedback}.
    """
    if not settings.llm_api_key or not settings.llm_base_url:
        return {"score": None, "verdict": "partial",
                "feedback": "LLM не настроен, оценка пропущена. Ответ сохранён в логе."}
    system = (
        "Ты опытный staff-engineer. Оцени ответ кандидата на system-design "
        "задачу по 4 критериям (по 25 баллов каждый):\n"
        "1) Покрытие компонентов\n2) Trade-offs и обоснование\n"
        "3) Масштабирование и узкие места\n4) Failure modes / observability\n\n"
        "Верни строго JSON: {\"score\": <0-100>, \"verdict\": \"correct|partial|wrong\", "
        "\"feedback\": \"<2-3 фразы по-русски, что хорошо и что улучшить>\"}."
    )
    user = f"ЗАДАЧА:\n{question}\n\nОТВЕТ КАНДИДАТА:\n{answer}"
    try:
        async with httpx.AsyncClient(timeout=30) as cli:
            r = await cli.post(
                settings.llm_base_url.rstrip("/") + "/chat/completions",
                headers={"Authorization": f"Bearer {settings.llm_api_key}"},
                json={
                    "model": settings.llm_model,
                    "messages": [
                        {"role": "system", "content": system},
                        {"role": "user", "content": user},
                    ],
                    "response_format": {"type": "json_object"},
                    "max_tokens": 400,
                    "temperature": 0.3,
                },
            )
        if r.status_code != 200:
            return None
        import json as _j
        body = r.json()["choices"][0]["message"]["content"]
        out = _j.loads(body)
        # Нормализуем поля
        score = out.get("score")
        return {
            "score": float(score) if isinstance(score, (int, float)) else None,
            "verdict": (out.get("verdict") or "partial").lower(),
            "feedback": out.get("feedback") or "Без подробного комментария.",
        }
    except Exception as exc:  # noqa: BLE001
        logger.warning("system-design score failed: %s", exc)
        return None


VERDICT_EMOJI = {
    "correct": "✅",
    "partial": "🟡",
    "wrong": "❌",
    "skipped": "⏭️",
    "off_topic": "↪️",
}


def _link_required_msg() -> str:
    return (
        "Ваш Telegram не привязан к аккаунту RealSync. "
        "Зайдите на сайт → Профиль → «Подключить Telegram» — получите ссылку с токеном."
    )


def render_summary(summary, header: str = "📊 <b>Сводка по практике</b>") -> str:
    if summary.total == 0:
        return (
            f"{header}\n\n"
            f"За последние {summary.days} дн. ответов пока не было.\n"
            "Запросите /question — давайте начнём."
        )

    avg = f"{summary.avg_score:.0f}/100" if summary.avg_score is not None else "—"
    lines = [
        header,
        "",
        f"<b>Период:</b> последние {summary.days} дн.",
        f"<b>Ответов:</b> {summary.total}",
        f"<b>Средний скор:</b> {avg}",
        "",
        f"✅ Верных: {summary.correct}",
        f"🟡 Частичных: {summary.partial}",
        f"❌ Неверных / off-topic: {summary.wrong}",
    ]

    if summary.by_topic:
        ranked = sorted(summary.by_topic, key=lambda t: t[1], reverse=True)
        strong = [t for t in ranked if t[1] >= 70][:3]
        weak = [t for t in ranked if t[1] < 60][-3:]
        if strong:
            lines += ["", "<b>💪 Сильные темы:</b>"] + [f"  • {t} — {s:.0f}/100 (×{c})" for t, s, c in strong]
        if weak:
            lines += ["", "<b>📌 Что подтянуть:</b>"] + [f"  • {t} — {s:.0f}/100 (×{c})" for t, s, c in weak]

    lines.append("")
    lines.append("Хочется ещё практики? /question")
    return "\n".join(lines)


# ─────────────────────────── /start ───────────────────────────


@router.message(CommandStart(deep_link=True))
async def on_start_with_token(message: Message, command: CommandObject) -> None:
    token = (command.args or "").strip()
    if not token:
        await message.answer(_link_required_msg(), reply_markup=menu_keyboard())
        return
    chat_id = message.chat.id
    tg_username = message.from_user.username if message.from_user else None
    user_id = await db.bind_chat_to_token(token, chat_id, tg_username)
    if user_id is None:
        await message.answer(
            "❌ Токен недействителен или истёк. Откройте Профиль на сайте и нажмите «Подключить Telegram» ещё раз."
        )
        return
    await message.answer(
        "✅ Аккаунт привязан!\n\n"
        "🎯 /question — вопрос для практики\n"
        "📊 /report — сводка за 7 дней\n"
        "🔥 /streak — твоя серия дней\n"
        "🧠 /term — термин дня\n"
        "📰 /news — IT-дайджест\n"
        "🏗 /challenge — system design (по пятницам авто)\n"
        "🍅 /focus 25 — Pomodoro 25 мин\n"
        "📄 /resume — загрузить резюме\n"
        "🐙 /github username — анализ профиля\n"
        "💳 /subscription — текущий тариф\n"
        "❓ /help — всё сразу",
        reply_markup=menu_keyboard(),
    )


@router.message(CommandStart())
async def on_start_plain(message: Message) -> None:
    link = await db.get_link_by_chat(message.chat.id)
    if link:
        await message.answer(
            f"Привет, {link.full_name}! Я — RealSync compagnon.",
            reply_markup=menu_keyboard(),
        )
    else:
        await message.answer(_link_required_msg(), reply_markup=menu_keyboard())


# ─────────────────────────── /question ───────────────────────────


async def _send_question(message_or_chat_id, *, topic_hint: Optional[str] = None) -> None:
    """Универсальный путь: и для команды /question, и для callback'ов."""
    chat_id = (
        message_or_chat_id.chat.id
        if isinstance(message_or_chat_id, Message)
        else int(message_or_chat_id)
    )
    bot = (
        message_or_chat_id.bot
        if isinstance(message_or_chat_id, Message)
        else None
    )

    link = await db.get_link_by_chat(chat_id)
    if link is None:
        text = _link_required_msg()
        if isinstance(message_or_chat_id, Message):
            await message_or_chat_id.answer(text)
        return

    topic, q = questions.pick(topic_hint)
    remember_question(chat_id, topic, q)
    text = (
        f"<b>Тема:</b> {topic}\n\n{q}\n\n"
        "Просто ответьте текстом — я оценю и пришлю фидбэк."
    )
    if isinstance(message_or_chat_id, Message):
        await message_or_chat_id.answer(text, parse_mode="HTML",
                                        reply_markup=after_question_keyboard())
    elif bot is not None:
        await bot.send_message(chat_id, text, parse_mode="HTML",
                               reply_markup=after_question_keyboard())


@router.message(Command("question"))
async def on_question(message: Message) -> None:
    await _send_question(message)


# ─────────────────────────── /report ───────────────────────────


@router.message(Command("report"))
async def on_report(message: Message, command: CommandObject) -> None:
    link = await db.get_link_by_chat(message.chat.id)
    if link is None:
        await message.answer(_link_required_msg())
        return
    days = 7
    arg = (command.args or "").strip()
    if arg.isdigit():
        days = max(1, min(60, int(arg)))
    summary = await db.summarize_practice(message.chat.id, days=days)
    await message.answer(render_summary(summary), parse_mode="HTML")


# ─────────────────────────── /streak ───────────────────────────


@router.message(Command("streak"))
async def on_streak(message: Message) -> None:
    link = await db.get_link_by_chat(message.chat.id)
    if link is None:
        await message.answer(_link_required_msg())
        return
    s = await db.get_streak(link.user_id)
    emoji = "🔥" if s.current >= 7 else "✨" if s.current >= 3 else "🌱"
    last = s.last_day or "—"
    await message.answer(
        f"{emoji} <b>Текущая серия:</b> {s.current} дн.\n"
        f"<b>Лучший рекорд:</b> {s.best} дн.\n"
        f"<b>Последний ответ:</b> {last}\n\n"
        f"{'Не теряй темп — ответь хотя бы один вопрос сегодня.' if s.current < 7 else 'Огонь, продолжай в том же духе! 💪'}",
        parse_mode="HTML",
    )


# ─────────────────────────── /term ───────────────────────────


@router.message(Command("term"))
async def on_term(message: Message) -> None:
    link = await db.get_link_by_chat(message.chat.id)
    if link is None:
        await message.answer(_link_required_msg())
        return
    used = await db.get_used_terms(link.user_id)
    term = content.pick_term(used)
    if term is None:
        await message.answer("Банк терминов пуст. Сообщите админу.")
        return
    await db.mark_term_used(link.user_id, term["id"])
    await message.answer(content.render_term(term), parse_mode="HTML")


# ─────────────────────────── /challenge (system design) ───────────────────────────


@router.message(Command("challenge"))
async def on_challenge(message: Message) -> None:
    link = await db.get_link_by_chat(message.chat.id)
    if link is None:
        await message.answer(_link_required_msg())
        return
    ch = content.pick_challenge()
    if ch is None:
        await message.answer("Банк челленджей пуст.")
        return
    # Запомним, что юзер сейчас работает над challenge — следующий
    # текстовый ответ пойдёт в AI как system-design eval.
    remember_question(message.chat.id, "system_design", ch["brief"])
    await message.answer(content.render_challenge(ch), parse_mode="HTML")


# ─────────────────────────── /news ───────────────────────────


@router.message(Command("news"))
async def on_news(message: Message) -> None:
    link = await db.get_link_by_chat(message.chat.id)
    if link is None:
        await message.answer(_link_required_msg())
        return
    await message.bot.send_chat_action(chat_id=message.chat.id, action="typing")
    items = await news.collect_news(per_source=4)
    text = await news.summarize_with_llm(items, limit=5)
    history = content.history_for_today()
    if history:
        text += f"\n\n📅 <i>{history}</i>"
    await message.answer(text, parse_mode="HTML", disable_web_page_preview=True)


# ─────────────────────────── /focus (Pomodoro) ───────────────────────────


@router.message(Command("focus"))
async def on_focus(message: Message, command: CommandObject) -> None:
    link = await db.get_link_by_chat(message.chat.id)
    if link is None:
        await message.answer(_link_required_msg())
        return

    args = (command.args or "").strip().split()
    work = 25
    rest = 5
    if args and args[0].isdigit():
        work = max(5, min(90, int(args[0])))
    if len(args) > 1 and args[1].isdigit():
        rest = max(1, min(30, int(args[1])))

    until_work = datetime.now(timezone.utc) + timedelta(minutes=work)
    await db.set_pomodoro(link.user_id, message.chat.id, until_work, "focus")
    await message.answer(
        f"🍅 <b>Pomodoro {work}/{rest}</b> запущен.\n"
        f"Сфокусируйся на одной задаче. Пришлю звон, когда время.",
        parse_mode="HTML",
    )

    async def _alarm():
        await asyncio.sleep(work * 60)
        try:
            await message.bot.send_message(
                message.chat.id,
                f"⏰ <b>Focus-блок завершён ({work} мин).</b>\n"
                f"Сделай {rest}-минутный перерыв. После — /focus {work} {rest} снова.",
                parse_mode="HTML",
            )
        except Exception as exc:  # noqa: BLE001
            logger.warning("focus alarm failed: %s", exc)
    asyncio.create_task(_alarm())


# ─────────────────────────── /subscription ───────────────────────────


@router.message(Command("subscription"))
async def on_subscription(message: Message) -> None:
    link = await db.get_link_by_chat(message.chat.id)
    if link is None:
        await message.answer(_link_required_msg())
        return
    # Берём через api-gateway → admin-service /billing/me/subscription.
    # Юзер-id из linked записи. Для S2S не делаем JWT — admin-service
    # достанет user из header'а если он есть, иначе вернёт 401 — тогда
    # покажем generic-сообщение.
    msg = "📦 <b>Текущий тариф:</b> Trial (бесплатный)"
    try:
        async with httpx.AsyncClient(timeout=8) as cli:
            # Минтим JWT за этого юзера тем же секретом, что и user-service —
            # admin-service валидирует Bearer-токен и достаёт user_id.
            resp = await cli.get(
                settings.api_gateway_url.rstrip("/") + "/api/billing/me/subscription",
                headers=auth_headers(link.user_id, link.email, link.role),
            )
        if resp.status_code == 200:
            sub = resp.json()
            tier = (sub.get("tier") or "trial").upper()
            status = sub.get("status") or "active"
            end = sub.get("end_date") or "—"
            msg = (
                f"📦 <b>Текущий тариф:</b> {tier}\n"
                f"<b>Статус:</b> {status}\n"
                f"<b>Действует до:</b> {end}"
            )
        elif resp.status_code == 404:
            msg = "📦 <b>Текущий тариф:</b> Trial (бесплатный)\nАпгрейд → /upgrade или кнопка ниже."
    except Exception as exc:  # noqa: BLE001
        logger.warning("subscription read failed: %s", exc)

    kb = upgrade_keyboard()
    await message.answer(msg, parse_mode="HTML", **({"reply_markup": kb} if kb else {}))


@router.message(Command("upgrade"))
async def on_upgrade(message: Message) -> None:
    kb = upgrade_keyboard()
    text = (
        "💎 Управление подпиской — на сайте, в разделе «Подписка».\n"
        "Тарифы: Trial (бесплатно), Pro (65 Br/мес), Platinum (159 Br/мес)."
    )
    if not kb:
        text += "\n\n<i>WEB_APP_URL не сконфигурирован — откройте раздел через основной сайт.</i>"
    await message.answer(text, parse_mode="HTML", **({"reply_markup": kb} if kb else {}))


# ─────────────────────────── /resume ───────────────────────────


@router.message(Command("resume"))
async def on_resume(message: Message) -> None:
    link = await db.get_link_by_chat(message.chat.id)
    if link is None:
        await message.answer(_link_required_msg())
        return
    await message.answer(
        "📄 <b>Анализ резюме через Telegram</b>\n\n"
        "Просто пришлите мне DOCX, TXT или RTF файл с резюме. Я отправлю его "
        "в AI-сервис RealSync и пришлю краткую сводку: топ-3 сильных стороны, "
        "топ-3 зоны роста, рекомендация по позициям.\n\n"
        "<i>PDF временно не принимается из-за нестабильного парсера.</i>",
        parse_mode="HTML",
    )


@router.message(F.document)
async def on_document(message: Message) -> None:
    link = await db.get_link_by_chat(message.chat.id)
    if link is None:
        await message.answer(_link_required_msg())
        return
    doc = message.document
    fname = (doc.file_name or "resume").lower()
    if not any(fname.endswith(ext) for ext in (".docx", ".txt", ".rtf")):
        await message.answer(
            "🛑 Принимаю только DOCX / TXT / RTF. PDF — пока нет."
        )
        return
    if doc.file_size and doc.file_size > 8 * 1024 * 1024:
        await message.answer("🛑 Файл слишком большой (макс 8 MB).")
        return

    await message.bot.send_chat_action(chat_id=message.chat.id, action="typing")
    # 1. скачать файл из Telegram
    file = await message.bot.get_file(doc.file_id)
    buf = await message.bot.download_file(file.file_path)
    data = buf.read() if hasattr(buf, "read") else bytes(buf)

    # 2. forward в interview-service /resume/import (тот же endpoint, что и web)
    try:
        async with httpx.AsyncClient(timeout=60) as cli:
            files = {"file": (doc.file_name, data, doc.mime_type or "application/octet-stream")}
            resp = await cli.post(
                settings.api_gateway_url.rstrip("/") + "/api/resume/import",
                headers=auth_headers(link.user_id, link.email, link.role),
                files=files,
            )
    except Exception as exc:  # noqa: BLE001
        await message.answer(f"⚠️ Не удалось отправить файл: {exc}")
        return

    if resp.status_code >= 400:
        await message.answer(
            f"⚠️ Сервис анализа вернул {resp.status_code}. Попробуйте через сайт."
        )
        return
    body = resp.json().get("data") if isinstance(resp.json(), dict) else None
    if not body:
        await message.answer("⚠️ Пустой ответ от сервиса.")
        return

    insights = body.get("ai_insights") or {}
    strong = insights.get("strong_points") or []
    improve = insights.get("improvement_points") or []
    positions = insights.get("recommended_positions") or []

    text = ["📄 <b>Анализ резюме готов</b>\n"]
    text.append(f"<i>{(insights.get('summary') or '')[:280]}</i>\n")
    if strong:
        text.append("<b>💪 Сильные стороны:</b>")
        for s in strong[:3]:
            text.append(f"  • {s}")
    if improve:
        text.append("\n<b>📌 Что улучшить:</b>")
        for s in improve[:3]:
            text.append(f"  • {s}")
    if positions:
        text.append("\n<b>🎯 Подходящие позиции:</b>")
        for p in positions[:3]:
            text.append(f"  • {p.get('role','?')} (fit {p.get('fit_score', '?')}%)")
    text.append("\n📊 Полный отчёт открыт на сайте → /workspace/resume")
    await message.answer("\n".join(text), parse_mode="HTML")


# ─────────────────────────── /github ───────────────────────────


@router.message(Command("github"))
async def on_github(message: Message, command: CommandObject) -> None:
    link = await db.get_link_by_chat(message.chat.id)
    if link is None:
        await message.answer(_link_required_msg())
        return
    username = (command.args or "").strip().lstrip("@")
    if not username or not re.match(r"^[A-Za-z0-9-]{1,39}$", username):
        await message.answer(
            "Использование: <code>/github username</code>\n"
            "Например: <code>/github octocat</code>",
            parse_mode="HTML",
        )
        return

    await message.bot.send_chat_action(chat_id=message.chat.id, action="typing")
    try:
        async with httpx.AsyncClient(timeout=90) as cli:
            headers = auth_headers(link.user_id, link.email, link.role)
            headers["Content-Type"] = "application/json"
            resp = await cli.post(
                settings.api_gateway_url.rstrip("/") + "/api/github/import",
                headers=headers,
                json={"profile_url": username},
            )
    except Exception as exc:  # noqa: BLE001
        await message.answer(f"⚠️ Не удалось импортировать: {exc}")
        return
    if resp.status_code >= 400:
        await message.answer(f"⚠️ GitHub-импорт вернул {resp.status_code}.")
        return
    body = resp.json().get("data") if isinstance(resp.json(), dict) else None
    if not body:
        await message.answer("⚠️ Пустой ответ.")
        return

    stats = body.get("github_stats") or body.get("stats") or {}
    insights = body.get("ai_insights") or {}
    strong = insights.get("strengths") or insights.get("strong_points") or []

    text = [
        f"🐙 <b>GitHub · @{body.get('username', username)}</b>",
        f"<i>{(insights.get('summary') or '')[:240]}</i>",
        "",
        f"📦 Репозиториев: <b>{stats.get('public_repos') or stats.get('repo_count') or '?'}</b>",
        f"⭐ Звёзд: <b>{stats.get('total_stars') or stats.get('stars') or '?'}</b>",
        f"👥 Подписчиков: <b>{stats.get('followers') or '?'}</b>",
    ]
    if strong:
        text.append("\n<b>💪 Сильные стороны:</b>")
        for s in strong[:3]:
            text.append(f"  • {s}")
    text.append("\n📊 Полный отчёт → /workspace/profile")
    await message.answer("\n".join(text), parse_mode="HTML")


# ─────────────────────────── /pause /resume_daily /unlink ───────────────────────────


@router.message(Command("pause"))
async def on_pause(message: Message) -> None:
    if await db.set_paused(message.chat.id, True):
        await message.answer("⏸ Ежедневные вопросы поставлены на паузу. Возобновить — /resume_daily.")
    else:
        await message.answer(_link_required_msg())


@router.message(Command("resume_daily"))
async def on_resume_daily(message: Message) -> None:
    if await db.set_paused(message.chat.id, False):
        await message.answer("▶ Ежедневные вопросы возобновлены.")
    else:
        await message.answer(_link_required_msg())


@router.message(Command("unlink"))
async def on_unlink(message: Message) -> None:
    if await db.unbind_chat(message.chat.id):
        await message.answer("🔌 Telegram отвязан от аккаунта RealSync.")
    else:
        await message.answer("Telegram уже не привязан — нечего отвязывать.")


# ─────────────────────────── /stats ───────────────────────────


@router.message(Command("stats"))
async def on_stats(message: Message) -> None:
    link = await db.get_link_by_chat(message.chat.id)
    if link is None:
        await message.answer(_link_required_msg())
        return
    status = "⏸ на паузе" if link.notifications_paused else "▶ активна"
    last_push = link.last_pushed_at.strftime("%Y-%m-%d %H:%M UTC") if link.last_pushed_at else "—"
    linked_at = link.linked_at.strftime("%Y-%m-%d") if link.linked_at else "—"
    await message.answer(
        f"<b>Профиль RealSync</b>\n"
        f"• Имя: {link.full_name}\n"
        f"• Email: {link.email}\n"
        f"• Привязан: {linked_at}\n\n"
        f"<b>Ежедневные вопросы:</b> {status}\n"
        f"• Час отправки: {link.daily_push_hour_utc:02d}:00 UTC\n"
        f"• Последний push: {last_push}",
        parse_mode="HTML",
    )


# ─────────────────────────── /help ───────────────────────────


@router.message(Command("help"))
async def on_help(message: Message) -> None:
    await message.answer(
        "🤖 <b>RealSync · Telegram-companion</b>\n\n"
        "<b>Практика:</b>\n"
        "/question — вопрос для практики · /report [N] — сводка · /streak — серия дней\n"
        "/term — термин дня · /challenge — system-design челлендж\n\n"
        "<b>Импорты:</b>\n"
        "/resume — пришлите DOCX и я разберу резюме\n"
        "/github <username> — анализ GitHub-профиля\n\n"
        "<b>Информация:</b>\n"
        "/news — IT-дайджест дня (новости + history-on-this-day)\n"
        "/subscription — текущий тариф · /focus 25 — pomodoro\n\n"
        "<b>Управление:</b>\n"
        "/stats · /pause · /resume_daily · /unlink",
        parse_mode="HTML",
        reply_markup=menu_keyboard(),
    )


# ─────────────────────────── callback queries ───────────────────────────


@router.callback_query(F.data == "next:same")
async def cb_next_same(cq: CallbackQuery) -> None:
    last = get_last_question(cq.message.chat.id)
    topic = last[0] if last else None
    await cq.answer()
    await _send_question(cq.message, topic_hint=topic)


@router.callback_query(F.data == "next:other")
async def cb_next_other(cq: CallbackQuery) -> None:
    await cq.answer()
    await _send_question(cq.message, topic_hint=None)


@router.callback_query(F.data == "skip")
async def cb_skip(cq: CallbackQuery) -> None:
    await cq.answer("Пропускаем…")
    await _send_question(cq.message)


@router.callback_query(F.data == "hint")
async def cb_hint(cq: CallbackQuery) -> None:
    last = get_last_question(cq.message.chat.id)
    if not last:
        await cq.answer("Сначала запроси вопрос /question.")
        return
    await cq.answer()
    await cq.message.answer(
        "💡 <b>Подсказка:</b> начните с конкретного примера из вашего опыта, "
        "затем структурируйте ответ как situation → action → result.",
        parse_mode="HTML",
    )


@router.callback_query(F.data == "cancel")
async def cb_cancel(cq: CallbackQuery) -> None:
    await cq.answer("Ок, заходи когда захочешь — /menu")
    await cq.message.answer("До встречи. /menu — главное меню.")


@router.callback_query(F.data == "report:7")
async def cb_report(cq: CallbackQuery) -> None:
    summary = await db.summarize_practice(cq.message.chat.id, days=7)
    await cq.answer()
    await cq.message.answer(render_summary(summary), parse_mode="HTML")


@router.callback_query(F.data.startswith("cmd:"))
async def cb_menu(cq: CallbackQuery) -> None:
    cmd = cq.data.split(":", 1)[1]
    await cq.answer()
    fake_msg = cq.message
    if cmd == "question":
        await _send_question(fake_msg)
    elif cmd == "report":
        summary = await db.summarize_practice(fake_msg.chat.id, days=7)
        await fake_msg.answer(render_summary(summary), parse_mode="HTML")
    elif cmd == "streak":
        link = await db.get_link_by_chat(fake_msg.chat.id)
        if not link:
            await fake_msg.answer(_link_required_msg()); return
        s = await db.get_streak(link.user_id)
        await fake_msg.answer(f"🔥 {s.current} дн. подряд (рекорд: {s.best})", parse_mode="HTML")
    elif cmd == "term":
        link = await db.get_link_by_chat(fake_msg.chat.id)
        if not link:
            await fake_msg.answer(_link_required_msg()); return
        used = await db.get_used_terms(link.user_id)
        term = content.pick_term(used)
        if term:
            await db.mark_term_used(link.user_id, term["id"])
            await fake_msg.answer(content.render_term(term), parse_mode="HTML")
    elif cmd == "news":
        items = await news.collect_news()
        text = await news.summarize_with_llm(items)
        await fake_msg.answer(text, parse_mode="HTML", disable_web_page_preview=True)
    elif cmd == "challenge":
        ch = content.pick_challenge()
        if ch:
            remember_question(fake_msg.chat.id, "system_design", ch["brief"])
            await fake_msg.answer(content.render_challenge(ch), parse_mode="HTML")
    elif cmd == "subscription":
        await on_subscription(fake_msg)


# ─────────────────────────── /menu ───────────────────────────


@router.message(Command("menu"))
async def on_menu(message: Message) -> None:
    await message.answer("Главное меню:", reply_markup=menu_keyboard())


# ─────────────────────────── plain text → score ───────────────────────────


@router.message(F.text & ~F.text.startswith("/"))
async def on_text_answer(message: Message) -> None:
    link = await db.get_link_by_chat(message.chat.id)
    if link is None:
        await message.answer(_link_required_msg())
        return
    last = get_last_question(message.chat.id)
    if not last:
        await message.answer("Активного вопроса нет — /question для нового.")
        return

    topic, question_text = last
    await message.bot.send_chat_action(chat_id=message.chat.id, action="typing")

    # System-design челлендж — soft-skills модель неадекватна для оценки
    # архитектурного разбора. Шлём в основной LLM по простому промпту.
    if topic == "system_design":
        result = await _score_system_design(question_text, message.text or "")
    else:
        result = await scoring.score_answer(question_text, message.text or "")
    if result is None:
        await message.answer("⚠️ Не удалось получить оценку, попробуйте позже.")
        return

    verdict = (result.get("verdict") or "partial").lower()
    score = result.get("score")
    feedback = result.get("feedback") or "Без подробного комментария."
    emoji = VERDICT_EMOJI.get(verdict, "📝")

    await db.log_practice(
        user_id=link.user_id, chat_id=message.chat.id, topic=topic,
        question=question_text, answer=message.text or "",
        score=float(score) if isinstance(score, (int, float)) else None,
        verdict=verdict, feedback=feedback,
    )
    # streak bump — каждый ответ продлевает серию
    streak = await db.bump_streak(link.user_id, message.chat.id)
    score_line = f"{score:.0f}/100" if isinstance(score, (int, float)) else "—"
    streak_line = ""
    if streak.current in {3, 7, 14, 30, 60, 100}:
        streak_line = f"\n\n🔥 <b>Streak {streak.current} дн.!</b>"

    await message.answer(
        f"{emoji} <b>{verdict.upper()}</b> · скор {score_line}\n\n{feedback}{streak_line}",
        parse_mode="HTML",
        reply_markup=after_answer_keyboard(),
    )

    next_topic, next_q = questions.pick(topic)
    remember_question(message.chat.id, next_topic, next_q)


# ─────────────────────────── Inline-mode ───────────────────────────


@router.inline_query()
async def on_inline_query(iq: InlineQuery) -> None:
    """Юзер пишет @bot <query> в любом чате — мы предлагаем 5 вопросов
    для практики, фильтруя по теме query."""
    q = (iq.query or "").lower().strip()
    out = []
    topics = list(questions._load().keys()) if hasattr(questions, "_load") else []  # noqa: SLF001
    seen = set()
    for _ in range(5):
        topic_hint = None
        if q:
            for topic in topics:
                if q in topic.lower():
                    topic_hint = topic
                    break
        topic, question_text = questions.pick(topic_hint)
        if question_text in seen:
            continue
        seen.add(question_text)
        out.append(
            InlineQueryResultArticle(
                id=f"{len(out)}-{hash(question_text) % 100000}",
                title=topic,
                description=question_text[:120],
                input_message_content=InputTextMessageContent(
                    message_text=f"🎯 <b>Вопрос для практики ({topic})</b>\n\n{question_text}",
                    parse_mode="HTML",
                ),
            )
        )
        if len(out) >= 5:
            break
    await iq.answer(out, cache_time=10, is_personal=True)
