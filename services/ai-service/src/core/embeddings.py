"""Embedding utilities for semantic similarity and text vectorization."""

import logging
from typing import Optional

import numpy as np
from openai import AsyncOpenAI, OpenAIError

try:
    from sentence_transformers import SentenceTransformer
except ImportError:
    SentenceTransformer = None

logger = logging.getLogger(__name__)


class EmbeddingService:
    """Service for generating text embeddings and computing similarity."""

    def __init__(
        self,
        api_key: str,
        model: str = "text-embedding-3-small",
        dimensions: int = 1536,
        base_url: Optional[str] = None,
    ) -> None:
        self._model = model
        self._dimensions = dimensions
        self._client = AsyncOpenAI(
            api_key=api_key,
            base_url=base_url,
        )

    async def embed(self, text: str) -> list[float]:
        """Generate an embedding vector for the given text.

        Args:
            text: Input text to embed.

        Returns:
            Embedding vector as a list of floats.

        Raises:
            ValueError: If text is empty.
            RuntimeError: If the API call fails.
        """
        if not text or not text.strip():
            raise ValueError("Cannot embed empty text")

        try:
            response = await self._client.embeddings.create(
                model=self._model,
                input=text,
                dimensions=self._dimensions,
            )
            return response.data[0].embedding
        except OpenAIError as exc:
            logger.error("Embedding API error: %s", exc)
            raise RuntimeError(f"Embedding API error: {exc}") from exc

    async def embed_batch(self, texts: list[str]) -> list[list[float]]:
        """Generate embeddings for multiple texts.

        Args:
            texts: List of input texts.

        Returns:
            List of embedding vectors.
        """
        if not texts:
            return []

        try:
            response = await self._client.embeddings.create(
                model=self._model,
                input=texts,
                dimensions=self._dimensions,
            )
            # Sort by index to ensure order matches input
            sorted_data = sorted(response.data, key=lambda x: x.index)
            return [item.embedding for item in sorted_data]
        except OpenAIError as exc:
            logger.error("Batch embedding API error: %s", exc)
            raise RuntimeError(f"Embedding API error: {exc}") from exc

    @staticmethod
    def cosine_similarity(vec_a: list[float], vec_b: list[float]) -> float:
        """Compute cosine similarity between two vectors.

        Args:
            vec_a: First vector.
            vec_b: Second vector.

        Returns:
            Similarity score between -1.0 and 1.0.
        """
        a = np.array(vec_a)
        b = np.array(vec_b)

        norm_a = np.linalg.norm(a)
        norm_b = np.linalg.norm(b)

        if norm_a == 0 or norm_b == 0:
            return 0.0

        return float(np.dot(a, b) / (norm_a * norm_b))

    async def similarity(
        self, text_a: str, text_b: str
    ) -> float:
        """Compute semantic similarity between two texts.

        Args:
            text_a: First text.
            text_b: Second text.

        Returns:
            Similarity score between 0.0 and 1.0.
        """
        embeddings = await self.embed_batch([text_a, text_b])
        sim = self.cosine_similarity(embeddings[0], embeddings[1])
        # Normalize from [-1, 1] to [0, 1]
        return round((sim + 1) / 2, 4)

    @property
    def model(self) -> str:
        return self._model

    @property
    def dimensions(self) -> int:
        return self._dimensions


class LocalEmbeddingClient:
    """Lightweight local semantic embedding model for question similarity (no API calls)."""

    def __init__(self, model_name: str = "sentence-transformers/all-MiniLM-L6-v2"):
        """Initialize local embedding model (lazy-loaded on first use)."""
        self._model: Optional[SentenceTransformer] = None
        self._model_name = model_name
        self._embedding_cache: dict[str, list[float]] = {}

    def _ensure_loaded(self) -> bool:
        """Load model on first use. Returns True if successful."""
        if SentenceTransformer is None:
            logger.warning("SentenceTransformer not available, semantic dedup disabled")
            return False

        if self._model is None:
            try:
                self._model = SentenceTransformer(self._model_name)
                logger.info("Local embedding model loaded: %s", self._model_name)
            except Exception as e:
                logger.error("Failed to load local embedding model: %s", e)
                return False

        return True

    def embed(self, text: str) -> Optional[list[float]]:
        """Generate embedding for text. Returns None if model unavailable."""
        if not text or not text.strip():
            return None

        cache_key = text.strip().lower()
        if cache_key in self._embedding_cache:
            return self._embedding_cache[cache_key]

        if not self._ensure_loaded():
            return None

        try:
            embedding = self._model.encode(text, convert_to_numpy=True)
            result = embedding.tolist()
            self._embedding_cache[cache_key] = result
            return result
        except Exception as e:
            logger.warning("Embedding failed for text: %s", e)
            return None

    def batch_embed(self, texts: list[str]) -> dict[str, Optional[list[float]]]:
        """Generate embeddings for multiple texts efficiently."""
        if not texts:
            return {}

        if not self._ensure_loaded():
            return {text: None for text in texts}

        result: dict[str, Optional[list[float]]] = {}
        texts_to_embed = []
        text_mapping = {}

        # Check cache
        for text in texts:
            cache_key = text.strip().lower()
            if cache_key in self._embedding_cache:
                result[text] = self._embedding_cache[cache_key]
            else:
                texts_to_embed.append(text)
                text_mapping[text] = cache_key

        if not texts_to_embed:
            return result

        try:
            embeddings = self._model.encode(texts_to_embed, convert_to_numpy=True)
            for original_text, embedding in zip(texts_to_embed, embeddings):
                embedding_list = embedding.tolist()
                cache_key = text_mapping[original_text]
                self._embedding_cache[cache_key] = embedding_list
                result[original_text] = embedding_list
        except Exception as e:
            logger.warning("Batch embedding failed: %s", e)
            for text in texts_to_embed:
                result[text] = None

        return result

    @staticmethod
    def cosine_similarity(vec1: Optional[list[float]], vec2: Optional[list[float]]) -> float:
        """Compute cosine similarity between two embedding vectors (0-1)."""
        if vec1 is None or vec2 is None:
            return 0.0

        try:
            v1 = np.array(vec1)
            v2 = np.array(vec2)
            dot_product = float(np.dot(v1, v2))
            norm1 = float(np.linalg.norm(v1))
            norm2 = float(np.linalg.norm(v2))

            if norm1 == 0.0 or norm2 == 0.0:
                return 0.0

            return dot_product / (norm1 * norm2)
        except Exception as e:
            logger.warning("Cosine similarity computation failed: %s", e)
            return 0.0


# Global local embedding client
_local_embedding_client: Optional[LocalEmbeddingClient] = None


def get_local_embedding_client() -> LocalEmbeddingClient:
    """Get or create global local embedding client."""
    global _local_embedding_client
    if _local_embedding_client is None:
        _local_embedding_client = LocalEmbeddingClient()
    return _local_embedding_client

