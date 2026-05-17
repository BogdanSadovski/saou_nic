"""Dependency injection providers for FastAPI."""

from functools import lru_cache
from typing import Any

from src.config import get_settings
from src.core.embeddings import EmbeddingService
from src.core.llm_client import LLMClient
from src.core.prompt_templates import PromptTemplateService
from src.services.analysis_service import AnalysisService
from src.services.question_service import QuestionService
from src.services.transcription_service import TranscriptionService


class DIContainer:
    """Simple dependency injection container."""

    def __init__(self) -> None:
        self._singletons: dict[str, Any] = {}

    def get_llm_client(self) -> LLMClient:
        if "llm_client" not in self._singletons:
            settings = get_settings()
            self._singletons["llm_client"] = LLMClient(
                api_key=settings.llm_api_key,
                model=settings.llm_model,
                temperature=settings.llm_temperature,
                max_tokens=settings.llm_max_tokens,
                base_url=settings.llm_base_url,
            )
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
