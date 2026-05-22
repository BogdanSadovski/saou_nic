"""Inline-keyboards — переиспользуемые блоки кнопок."""

from __future__ import annotations

from aiogram.types import InlineKeyboardButton, InlineKeyboardMarkup, WebAppInfo

from .config import settings


def after_answer_keyboard() -> InlineKeyboardMarkup:
    """Под каждым AI-вердиктом: следующий вопрос / сменить тему / стоп."""
    rows = [
        [
            InlineKeyboardButton(text="▶ Ещё вопрос",   callback_data="next:same"),
            InlineKeyboardButton(text="🔀 Сменить тему", callback_data="next:other"),
        ],
        [
            InlineKeyboardButton(text="📊 Отчёт за неделю", callback_data="report:7"),
            InlineKeyboardButton(text="🛑 Хватит", callback_data="cancel"),
        ],
    ]
    if settings.web_app_url:
        rows.append([
            InlineKeyboardButton(text="🚀 Открыть платформу", web_app=WebAppInfo(url=settings.web_app_url)),
        ])
    return InlineKeyboardMarkup(inline_keyboard=rows)


def after_question_keyboard() -> InlineKeyboardMarkup:
    """Под только что выданным вопросом — отказаться/пропустить."""
    return InlineKeyboardMarkup(inline_keyboard=[
        [
            InlineKeyboardButton(text="⏭ Пропустить", callback_data="skip"),
            InlineKeyboardButton(text="💡 Дать подсказку", callback_data="hint"),
        ],
    ])


def menu_keyboard() -> InlineKeyboardMarkup:
    """Главное меню — для /start и /help."""
    rows = [
        [
            InlineKeyboardButton(text="🎯 Вопрос",      callback_data="cmd:question"),
            InlineKeyboardButton(text="📊 Отчёт 7д",    callback_data="cmd:report"),
        ],
        [
            InlineKeyboardButton(text="🔥 Streak",     callback_data="cmd:streak"),
            InlineKeyboardButton(text="🧠 Термин дня", callback_data="cmd:term"),
        ],
        [
            InlineKeyboardButton(text="📰 IT-новости", callback_data="cmd:news"),
            InlineKeyboardButton(text="🏗 Challenge",  callback_data="cmd:challenge"),
        ],
        [
            InlineKeyboardButton(text="💳 Подписка",   callback_data="cmd:subscription"),
        ],
    ]
    if settings.web_app_url:
        rows.insert(0, [
            InlineKeyboardButton(text="🚀 Открыть платформу",
                                 web_app=WebAppInfo(url=settings.web_app_url)),
        ])
    return InlineKeyboardMarkup(inline_keyboard=rows)


def upgrade_keyboard() -> InlineKeyboardMarkup | None:
    """Под /subscription — кнопка перехода к биллингу.

    Если WEB_APP_URL не задан, кнопку не показываем — Telegram не
    разрешает url-кнопки без валидного https://домен.
    """
    if not settings.web_app_url:
        return None
    base = settings.web_app_url.rstrip("/")
    return InlineKeyboardMarkup(inline_keyboard=[
        [InlineKeyboardButton(text="💎 Управлять подпиской", url=base + "/workspace/billing")],
    ])
