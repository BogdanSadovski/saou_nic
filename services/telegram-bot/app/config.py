"""Telegram-bot service config.

Все секреты приходят из переменных окружения (см. docker-compose.yml,
блок `telegram-bot.environment`). В коде только сами имена.
"""

from __future__ import annotations

from pydantic import Field
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(env_file=None, case_sensitive=False)

    tg_bot_token: str = Field(..., description="Telegram bot token from @BotFather")
    tg_bot_username: str = Field(
        "realsync_practice_bot",
        description="Bot username без @ — используется в deep-link.",
    )

    # Postgres user_service DB
    pg_dsn: str = Field(
        "postgresql://postgres:postgres_secret@postgres:5432/user_service",
        description="DSN to user_service DB (хранит telegram_links).",
    )

    # AI / scoring backends
    interview_service_url: str = Field(
        "http://interview-service:8082",
        description="Где живёт soft-skills-score endpoint и список вопросов.",
    )
    softskills_service_url: str = Field(
        "http://softskills-service:8090",
        description="Прямой URL ML-сервиса soft-skills.",
    )

    # Daily push
    daily_push_default_hour_utc: int = Field(
        6, description="6 UTC ≈ 09:00 MSK / 09:00 Минск."
    )

    # LLM для daily-digest и оценки challenge'ов. Берём те же ключи,
    # что и весь стек (см. infrastructure/docker/.env).
    llm_api_key: str | None = Field(None)
    llm_base_url: str | None = Field(None)
    llm_model: str = Field("deepseek-chat")

    # Прочие URL'ы — для /resume и /github команд.
    resume_service_url: str = Field("http://resume-service:8080")
    api_gateway_url: str = Field("http://api-gateway:8000")

    # Кнопка [🚀 Открыть платформу] — Telegram WebApp.
    # Должен быть публичный HTTPS-URL вашего фронта. На dev — пусть пусто,
    # тогда кнопку не показываем.
    web_app_url: str | None = Field(None)

    # HMAC-секрет для push-webhook от других сервисов.
    push_webhook_secret: str = Field("change_me_in_env")

    # Тот же JWT_SECRET, что и у user-service. Используется ботом для
    # самостоятельной выдачи access-токена за привязанного пользователя —
    # чтобы вызовы /resume/import, /github/import, /billing/me шли с
    # валидным Bearer-токеном (api-gateway требует JWT, наш header
    # X-User-ID не доверенный).
    jwt_secret: str = Field("super_secret_jwt_key_change_in_production")
    jwt_issuer: str = Field("telegram-bot")
    jwt_ttl_sec: int = Field(900)  # 15 минут — больше чем нужно для S2S

    # Long-polling vs webhook. На dev — polling.
    use_webhook: bool = Field(False)
    webhook_url: str | None = Field(None)

    log_level: str = Field("INFO")


settings = Settings()
