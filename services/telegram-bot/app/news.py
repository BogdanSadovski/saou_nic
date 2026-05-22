"""Daily IT-news digest.

Тянем RSS из нескольких источников, фильтруем по дате, через DeepSeek
сжимаем 3-5 заголовков в одно сообщение. Результат кешируется на сутки.

Источники подобраны так, чтобы был баланс ru/en и разный фокус:
  • Hacker News Top — global tech
  • dev.to top — практический dev-контент
  • Habr — русскоязычный
"""

from __future__ import annotations

import asyncio
import logging
import re
import xml.etree.ElementTree as ET
from dataclasses import dataclass
from datetime import datetime, timezone

import httpx

from .config import settings

logger = logging.getLogger(__name__)


FEEDS = [
    # Hacker News best of the day
    ("Hacker News", "https://hnrss.org/best"),
    # Habr Top-Daily — русскоязычные айтишные новости/туториалы
    ("Habr", "https://habr.com/ru/rss/best/daily/?fl=ru"),
    # dev.to top week
    ("dev.to", "https://dev.to/feed/"),
]


@dataclass
class NewsItem:
    source: str
    title: str
    link: str
    pub_date: str


def _strip_html(s: str) -> str:
    return re.sub(r"<[^>]+>", "", s or "").strip()


async def _fetch_feed(name: str, url: str, limit: int = 5) -> list[NewsItem]:
    try:
        async with httpx.AsyncClient(timeout=8) as cli:
            resp = await cli.get(url, headers={"User-Agent": "RealSync-Digest/1.0"})
        if resp.status_code >= 400:
            return []
        root = ET.fromstring(resp.text)
        # RSS 2.0 — items живут под channel.item
        items = root.findall(".//item")[:limit]
        out: list[NewsItem] = []
        for it in items:
            title = _strip_html(it.findtext("title") or "")
            link = (it.findtext("link") or "").strip()
            pub = (it.findtext("pubDate") or "").strip()
            if not title or not link:
                continue
            out.append(NewsItem(source=name, title=title, link=link, pub_date=pub))
        return out
    except Exception as exc:  # noqa: BLE001
        logger.warning("feed %s fetch failed: %s", name, exc)
        return []


async def collect_news(per_source: int = 4) -> list[NewsItem]:
    """Собрать ~12 топ-новостей из всех источников."""
    results = await asyncio.gather(*(_fetch_feed(n, u, per_source) for n, u in FEEDS))
    flat: list[NewsItem] = []
    for r in results:
        flat.extend(r)
    return flat


async def summarize_with_llm(items: list[NewsItem], limit: int = 5) -> str:
    """Просим DeepSeek сжать 12 заголовков в 5 пунктов на русском.

    Если DeepSeek недоступен (нет ключа / 4xx) — возвращаем raw-список
    без суммаризации, тоже информативно.
    """
    if not items:
        return "Сегодня релевантных новостей не нашлось — попробуйте позже."

    # Plain-list fallback (без LLM)
    def _plain():
        lines = ["📰 <b>IT-дайджест дня</b>", ""]
        for i, it in enumerate(items[:limit], 1):
            lines.append(f"{i}. <a href=\"{it.link}\">{it.title}</a>")
            lines.append(f"   <i>· {it.source}</i>")
        return "\n".join(lines)

    api_key = settings.llm_api_key
    api_url = settings.llm_base_url
    model = settings.llm_model
    if not api_key or not api_url:
        return _plain()

    headlines = "\n".join(f"- [{i.source}] {i.title}" for i in items[:12])
    system = (
        "Ты IT-редактор. Из списка заголовков RSS выбери самые интересные "
        "5 для разработчика, сгруппируй смыслово и напиши по 1-2 предложения "
        "пояснения на русском. Без воды, без оценочных слов. Не выдумывай факты."
    )
    user = (
        f"Заголовки за сегодня:\n{headlines}\n\n"
        "Сформулируй дайджест. Формат каждой записи:\n"
        "1. <b>Заголовок</b>\n   Объяснение в 1-2 фразы.\n"
        "В конце добавь блок \"Подробнее по ссылкам\" с url'ами top-5."
    )
    try:
        async with httpx.AsyncClient(timeout=20) as cli:
            resp = await cli.post(
                api_url.rstrip("/") + "/chat/completions",
                headers={"Authorization": f"Bearer {api_key}"},
                json={
                    "model": model,
                    "messages": [
                        {"role": "system", "content": system},
                        {"role": "user", "content": user},
                    ],
                    "max_tokens": 600,
                    "temperature": 0.4,
                },
            )
        if resp.status_code != 200:
            logger.warning("llm digest %s: %s", resp.status_code, resp.text[:200])
            return _plain()
        data = resp.json()
        body = (data.get("choices", [{}])[0].get("message", {}).get("content") or "").strip()
        if not body:
            return _plain()
        # дополняем линками если LLM забыл
        if "http" not in body:
            body += "\n\n<b>Ссылки:</b>\n" + "\n".join(
                f"• <a href=\"{i.link}\">{i.title}</a>" for i in items[:5]
            )
        return f"📰 <b>IT-дайджест дня</b>\n\n{body}"
    except Exception as exc:  # noqa: BLE001
        logger.warning("llm digest exception: %s", exc)
        return _plain()
