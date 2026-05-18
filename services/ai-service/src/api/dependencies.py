"""Dependency injection providers for FastAPI."""

import logging
from functools import lru_cache
from typing import Any, Union

from src.config import get_settings
from src.core.embeddings import EmbeddingService
from src.core.llm_client import LLMClient, LLMPool, LLMRouter
from src.core.prompt_templates import PromptTemplateService
from src.services.analysis_service import AnalysisService
from src.services.question_service import QuestionService
from src.services.transcription_service import TranscriptionService

logger = logging.getLogger(__name__)


class DIContainer:
    """Simple dependency injection container."""

    def __init__(self) -> None:
        self._singletons: dict[str, Any] = {}

    def get_llm_client(self) -> Union[LLMClient, LLMRouter]:
        """Return the LLM client cascade.

        Builds up to 4 OpenAI-compatible clients (Tier 1–4) from
        env-driven config and wraps them in an :class:`LLMRouter`. If
        only Tier 1 is configured, returns the single client directly
        for minimal overhead. Cascade order:

            1. ``LLM_*``  (primary — e.g. Groq paid / OpenAI)
            2. ``LLM_SECONDARY_*``  (e.g. OpenRouter free)
            3. ``LLM_TERTIARY_*``  (e.g. DeepSeek free)
            4. ``LLM_QUATERNARY_*``  (last-resort free endpoint)
        """
        if "llm_client" in self._singletons:
            return self._singletons["llm_client"]

        settings = get_settings()
        tiers: list[LLMClient | LLMPool] = []

        def _make_client(api_key: str, model: str, base_url: str | None) -> LLMClient:
            return LLMClient(
                api_key=api_key,
                model=model,
                temperature=settings.llm_temperature,
                max_tokens=settings.llm_max_tokens,
                base_url=base_url,
            )

        # Tier 1 — primary (always present).
        tiers.append(
            _make_client(
                api_key=settings.llm_api_key,
                model=settings.llm_model,
                base_url=settings.llm_base_url,
            )
        )

        # Tier 2 — secondary (single key).
        if settings.llm_secondary_api_key:
            tiers.append(
                _make_client(
                    api_key=settings.llm_secondary_api_key,
                    model=settings.llm_secondary_model or settings.llm_model,
                    base_url=settings.llm_secondary_base_url,
                )
            )

        # Tier 3 — DeepSeek via OpenRouter (key pool). Designed for
        # 5× OpenRouter keys on the same free DeepSeek model. Defaults
        # picked so only ``LLM_TERTIARY_API_KEYS`` needs setting.
        tertiary_keys = settings.tertiary_keys()
        if tertiary_keys:
            t3_model = settings.llm_tertiary_model or "deepseek/deepseek-chat-v3-0324:free"
            t3_base = settings.llm_tertiary_base_url or "https://openrouter.ai/api/v1"
            t3_pool = [_make_client(k, t3_model, t3_base) for k in tertiary_keys]
            if len(t3_pool) == 1:
                tiers.append(t3_pool[0])
            else:
                tiers.append(LLMPool(t3_pool))

        # Tier 4 — last-resort safety net (also supports key pool).
        quaternary_keys = settings.quaternary_keys()
        if quaternary_keys:
            t4_model = settings.llm_quaternary_model or settings.llm_model
            t4_base = settings.llm_quaternary_base_url
            t4_pool = [_make_client(k, t4_model, t4_base) for k in quaternary_keys]
            if len(t4_pool) == 1:
                tiers.append(t4_pool[0])
            else:
                tiers.append(LLMPool(t4_pool))

        if len(tiers) == 1:
            logger.info("LLM cascade: 1 tier (primary only — no fallback)")
            self._singletons["llm_client"] = tiers[0]
        else:
            def _describe(tier: LLMClient | LLMPool) -> str:
                if isinstance(tier, LLMPool):
                    return f"{tier.model}×{tier.size}"
                return tier.model

            logger.info(
                "LLM cascade: %d tiers configured — %s",
                len(tiers),
                " → ".join(_describe(t) for t in tiers),
            )
            self._singletons["llm_client"] = LLMRouter(tiers)

        return self._singletons["llm_client"]

    def get_prompt_templates(self) -> PromptTemplateService:
        if "prompt_templates" not in self._singletons:
            self._singletons["prompt_templates"] = PromptTemplateService()
        return self._singletons["prompt_templates"]

    def get_embedding_service(self) -> EmbeddingService:
        if "embedding_service" not in self._singletons:
            settings = get_settings()
            self._singletons["embedding_service"] = EmbeddingService(
                api_key=settings.llm_api_key,
                model=settings.embedding_model,
                dimensions=settings.embedding_dimensions,
                base_url=settings.llm_base_url,
            )
        return self._singletons["embedding_service"]

    def get_question_service(self) -> QuestionService:
        if "question_service" not in self._singletons:
            self._singletons["question_service"] = QuestionService(
                llm_client=self.get_llm_client(),
                prompt_templates=self.get_prompt_templates(),
            )
        return self._singletons["question_service"]

    def get_analysis_service(self) -> AnalysisService:
        if "analysis_service" not in self._singletons:
            self._singletons["analysis_service"] = AnalysisService(
                llm_client=self.get_llm_client(),
                prompt_templates=self.get_prompt_templates(),
                embedding_service=self.get_embedding_service(),
            )
        return self._singletons["analysis_service"]

    def get_transcription_service(self) -> TranscriptionService:
        if "transcription_service" not in self._singletons:
            settings = get_settings()
            self._singletons["transcription_service"] = TranscriptionService(
                api_key=settings.llm_api_key,
                model=settings.transcription_model,
                max_file_size_mb=settings.transcription_max_file_size_mb,
                base_url=settings.llm_base_url,
            )
        return self._singletons["transcription_service"]


@lru_cache
def get_di_container() -> DIContainer:
    """Return a cached DI container instance."""
    return DIContainer()
