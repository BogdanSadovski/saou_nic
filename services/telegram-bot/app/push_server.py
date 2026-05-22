"""HTTP-webhook сервер внутри бота для приёма событий с платформы.

Другие сервисы (interview-service, resume-service, admin-service)
шлют POST /push с HMAC-подписью. Бот форвардит сообщение в чат
привязанного юзера.

Endpoint:
  POST /push
  Headers: X-Signature: hmac-sha256(secret, body)
  Body:    {"user_id": "...", "kind": "interview_finished|resume_ready|quota_low|generic",
            "text": "🎯 Интервью завершено: скор 78. /workspace/reports/abc",
            "buttons"?: [{"text": "Открыть", "url": "..."}]}

Запускается в том же процессе, что и aiogram polling, на отдельном порту.
"""

from __future__ import annotations

import asyncio
import hmac
import hashlib
import json
import logging

from aiogram import Bot
from aiogram.types import InlineKeyboardButton, InlineKeyboardMarkup
from aiohttp import web

from . import db
from .config import settings

logger = logging.getLogger(__name__)


def _verify(body: bytes, signature: str) -> bool:
    if not signature:
        return False
    expected = hmac.new(
        settings.push_webhook_secret.encode(),
        body,
        hashlib.sha256,
    ).hexdigest()
    return hmac.compare_digest(expected, signature)


async def _handle_push(bot: Bot, request: web.Request) -> web.Response:
    body = await request.read()
    sig = request.headers.get("X-Signature", "")
    if not _verify(body, sig):
        return web.json_response({"error": "invalid signature"}, status=401)

    try:
        payload = json.loads(body)
    except Exception:  # noqa: BLE001
        return web.json_response({"error": "bad json"}, status=400)

    user_id = (payload.get("user_id") or "").strip()
    text = (payload.get("text") or "").strip()
    if not user_id or not text:
        return web.json_response({"error": "user_id and text required"}, status=400)

    # ищем chat_id по user_id напрямую в БД
    pool = await db.get_pool()
    async with pool.acquire() as conn:
        row = await conn.fetchrow(
            "SELECT chat_id, notifications_paused FROM telegram_links WHERE user_id = $1",
            user_id,
        )
    if row is None or row["chat_id"] is None:
        return web.json_response({"ok": True, "delivered": False, "reason": "not_linked"}, status=200)
    if row["notifications_paused"]:
        return web.json_response({"ok": True, "delivered": False, "reason": "paused"}, status=200)

    buttons = payload.get("buttons") or []
    reply_markup = None
    if isinstance(buttons, list) and buttons:
        kb_rows = []
        for b in buttons[:6]:
            if not isinstance(b, dict): continue
            label, url = b.get("text"), b.get("url")
            if label and url:
                kb_rows.append([InlineKeyboardButton(text=label, url=url)])
        if kb_rows:
            reply_markup = InlineKeyboardMarkup(inline_keyboard=kb_rows)

    try:
        await bot.send_message(row["chat_id"], text, parse_mode="HTML",
                               reply_markup=reply_markup,
                               disable_web_page_preview=True)
    except Exception as exc:  # noqa: BLE001
        logger.warning("push send failed: %s", exc)
        return web.json_response({"ok": False, "error": str(exc)}, status=500)
    return web.json_response({"ok": True, "delivered": True})


async def _health(_req: web.Request) -> web.Response:
    return web.json_response({"status": "ok"})


async def start_push_server(bot: Bot, port: int = 8096) -> None:
    app = web.Application()
    app.router.add_get("/health", _health)
    app.router.add_post("/push", lambda r: _handle_push(bot, r))
    runner = web.AppRunner(app)
    await runner.setup()
    site = web.TCPSite(runner, "0.0.0.0", port)
    await site.start()
    logger.info("push-webhook server listening on :%d", port)
    # держим вечно
    while True:
        await asyncio.sleep(3600)
