"""Application configuration using pydantic-settings."""

from functools import lru_cache
from typing import Optional

from pydantic import Field, field_validator
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    """Application settings loaded from environment variables and .env file."""

    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        case_sensitive=False,
        extra="ignore",
    )

    # Application
    app_name: str = "ai-service"
    app_version: str = "1.0.0"
    debug: bool = False
    host: str = "0.0.0.0"
    port: int = 8001
    log_level: str = "INFO"

    # LLM Configuration — primary provider (Tier 1)
    llm_api_key: str = Field(..., description="Primary LLM API key (Tier 1)")
    llm_model: str = "gpt-4o-mini"
    llm_temperature: float = 0.7
    llm_max_tokens: int = 2048
    llm_base_url: Optional[str] = None

    # Secondary provider (Tier 2) — used when Tier 1 returns 429/5xx
    # (currently typically OpenRouter free Llama).
    llm_secondary_api_key: Optional[str] = None
    llm_secondary_base_url: Optional[str] = None
    llm_secondary_model: Optional[str] = None

    # Tertiary provider (Tier 3) — pool with round-robin. Designed for
    # 5× OpenRouter accounts/keys pointing at the same free DeepSeek
    # model, but works with any provider. Provide a comma-separated
    # list of keys via ``LLM_TERTIARY_API_KEYS`` (plural). The single
    # ``LLM_TERTIARY_API_KEY`` is still accepted as a 1-key fallback.
    # base_url defaults to OpenRouter, model defaults to free DeepSeek.
    llm_tertiary_api_key: Optional[str] = None
    llm_tertiary_api_keys: Optional[str] = None   # comma-separated
    llm_tertiary_base_url: Optional[str] = None
    llm_tertiary_model: Optional[str] = None

    # Quaternary provider (Tier 4) — last-resort safety net. Use any
    # OpenAI-compatible free endpoint (Together AI free, Cerebras free,
    # Mistral free, HuggingFace router, etc.). Also supports a key pool
    # via ``LLM_QUATERNARY_API_KEYS``.
    llm_quaternary_api_key: Optional[str] = None
    llm_quaternary_api_keys: Optional[str] = None
    llm_quaternary_base_url: Optional[str] = None
    llm_quaternary_model: Optional[str] = None

    def tertiary_keys(self) -> list[str]:
        """Return the list of Tier 3 API keys (plural form takes priority)."""
        if self.llm_tertiary_api_keys:
            return [k.strip() for k in self.llm_tertiary_api_keys.split(",") if k.strip()]
        if self.llm_tertiary_api_key:
            return [self.llm_tertiary_api_key]
        return []

    def quaternary_keys(self) -> list[str]:
        """Return the list of Tier 4 API keys (plural form takes priority)."""
        if self.llm_quaternary_api_keys:
            return [k.strip() for k in self.llm_quaternary_api_keys.split(",") if k.strip()]
        if self.llm_quaternary_api_key:
            return [self.llm_quaternary_api_key]
        return []

    # Embeddings
    embedding_model: str = "text-embedding-3-small"
    embedding_dimensions: int = 1536

    # Transcription
    transcription_model: str = "whisper-1"
    transcription_max_file_size_mb: int = 25

    # Rate limiting
    rate_limit_per_minute: int = 60

    # CORS
    cors_origins: list[str] = Field(
        default_factory=lambda: ["http://localhost:3000"],
    )

    @field_validator("log_level")
    @classmethod
    def validate_log_level(cls, v: str) -> str:
        valid_levels = {"DEBUG", "INFO", "WARNING", "ERROR", "CRITICAL"}
        upper = v.upper()
        if upper not in valid_levels:
            raise ValueError(
                f"Invalid log level: {v}. Must be one of {valid_levels}"
            )
        return upper


@lru_cache
def get_settings() -> Settings:
    """Cached singleton for application settings."""
    return Settings()
