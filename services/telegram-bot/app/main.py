"""Точка входа telegram-bot service.

aiogram long-polling + background scheduler в одном процессе. Webhook
поддержка оставлена как заглушка (use_webhook=False по дефолту).
"""

from __future__ import annotations

import asyncio
import logging

from aiogram import Bot, Dispatcher
from aiogram.client.default import DefaultBotProperties
from aiogram.enums import ParseMode

from .config import settings
from .handlers import router
from .push_server import start_push_server
from .scheduler import run_daily_loop


def _setup_logging() -> None:
    logging.basicConfig(
        level=getattr(logging, settings.log_level.upper(), logging.INFO),
        format="%(asctime)s %(levelname)s %(name)s :: %(message)s",
    )


async def main() -> None:
    _setup_logging()
    log = logging.getLogger("main")
    log.info("starting telegram-bot @%s", settings.tg_bot_username)

    bot = Bot(
        token=settings.tg_bot_token,
        default=DefaultBotProperties(parse_mode=ParseMode.HTML),
    )
    dp = Dispatcher()
    dp.include_router(router)

    # Background-задачи: scheduler + push-webhook server.
    scheduler_task = asyncio.create_task(run_daily_loop(bot))
    push_task = asyncio.create_task(start_push_server(bot))

    try:
        # Снимаем висящий webhook на всякий случай, переходим в polling.
        await bot.delete_webhook(drop_pending_updates=False)
        await dp.start_polling(bot, allowed_updates=dp.resolve_used_update_types())
    finally:
        scheduler_task.cancel()
        push_task.cancel()
        await bot.session.close()


if __name__ == "__main__":
    asyncio.run(main())
