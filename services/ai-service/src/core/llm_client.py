"""LLM client wrapper for OpenAI-compatible APIs."""

import json
import logging
from typing import Any, Optional

from openai import AsyncOpenAI, OpenAIError

logger = logging.getLogger(__name__)


class LLMClient:
    """Async client for interacting with OpenAI-compatible LLM APIs."""

    def __init__(
        self,
        api_key: str,
        model: str = "gpt-4o-mini",
        temperature: float = 0.7,
        max_tokens: int = 2048,
        base_url: Optional[str] = None,
    ) -> None:
        self._model = model
        self._temperature = temperature
        self._max_tokens = max_tokens
        self._client = AsyncOpenAI(
            api_key=api_key,
            base_url=base_url,
        )

    async def generate(
        self,
        prompt: str,
        system_prompt: Optional[str] = None,
        temperature: Optional[float] = None,
        max_tokens: Optional[int] = None,
        response_format: Optional[dict[str, Any]] = None,
    ) -> str:
        """Generate a completion from the LLM.

        Args:
            prompt: The user prompt.
            system_prompt: Optional system prompt.
            temperature: Override default temperature.
            max_tokens: Override default max tokens.
            response_format: Optional JSON schema for structured output.

        Returns:
            The generated text response.
        """
        messages: list[dict[str, str]] = []
        if system_prompt:
            messages.append({"role": "system", "content": system_prompt})
        messages.append({"role": "user", "content": prompt})

        kwargs: dict[str, Any] = {
            "model": self._model,
            "messages": messages,
            "temperature": temperature if temperature is not None else self._temperature,
            "max_tokens": max_tokens if max_tokens is not None else self._max_tokens,
        }

        if response_format:
            kwargs["response_format"] = response_format

        try:
            response = await self._client.chat.completions.create(**kwargs)
            content = response.choices[0].message.content
            if content is None:
                raise RuntimeError("LLM returned empty response")
            logger.debug(
                "LLM request completed: tokens_used=%s",
                response.usage.total_tokens if response.usage else "unknown",
            )
            return content
        except OpenAIError as exc:
            logger.error("LLM API error: %s", exc)
            raise RuntimeError(f"LLM API error: {exc}") from exc

    async def generate_json(
        self,
        prompt: str,
        system_prompt: Optional[str] = None,
        schema: Optional[dict[str, Any]] = None,
    ) -> dict[str, Any]:
        """Generate a JSON response, optionally validated against a schema.

        Args:
            prompt: The user prompt.
            system_prompt: Optional system prompt instructing JSON output.
            schema: Optional JSON schema for structured output.

        Returns:
            Parsed JSON as a dictionary.
        """
        default_system = (
            "You must respond with valid JSON only. No markdown, no explanations."
        )
        if schema:
            response_format = {"type": "json_schema", "json_schema": schema}
        else:
            response_format = {"type": "json_object"}

        try:
            raw = await self.generate(
                prompt=prompt,
                system_prompt=system_prompt or default_system,
                response_format=response_format,
            )
        except RuntimeError as exc:
            error_text = str(exc)
            should_retry_without_schema = schema is not None and (
                "response format `json_schema`" in error_text
                or "response_format" in error_text
                or "structured outputs" in error_text
            )
            if not should_retry_without_schema:
                raise

            logger.warning(
                "LLM model does not support json_schema response format, retrying with json_object"
            )
            raw = await self.generate(
                prompt=prompt,
                system_prompt=system_prompt or default_system,
                response_format={"type": "json_object"},
            )

        try:
            return json.loads(raw)
        except json.JSONDecodeError as exc:
            logger.error("Failed to parse LLM JSON response: %s", raw[:500])
            raise ValueError(
                f"LLM returned invalid JSON: {exc}"
            ) from exc

    @property
    def model(self) -> str:
        return self._model
