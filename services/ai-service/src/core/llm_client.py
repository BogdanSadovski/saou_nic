"""LLM client wrapper for OpenAI-compatible APIs."""

import asyncio
import json
import logging
import re
from typing import Any, Optional

from openai import AsyncOpenAI, OpenAIError, RateLimitError

logger = logging.getLogger(__name__)


# Hard cap on a Retry-After hint so a malformed server response can't
# stall the whole interview for minutes.
_MAX_RETRY_AFTER_SECONDS = 20.0


def _extract_retry_after(exc: BaseException) -> float:
    """Pull a sleep hint out of a RateLimitError.

    Both Groq and OpenRouter return JSON like
        ... 'Please try again in 10.815s' ...
    inside the error message. We parse that string first because the
    `Retry-After` header isn't always exposed through the SDK's
    `headers` attribute on async clients. Falls back to 5s.
    """
    text = str(exc)
    # 'try again in 10.815s' / 'retry after 12 seconds'
    m = re.search(r"(?:try again in|retry after)\s+([0-9]+(?:\.[0-9]+)?)\s*s", text, re.IGNORECASE)
    if m:
        try:
            return min(_MAX_RETRY_AFTER_SECONDS, float(m.group(1)))
        except ValueError:
            pass
    return 5.0


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
        # OpenRouter recommends HTTP-Referer + X-Title for app attribution.
        # OpenAI / Groq ignore unknown headers.
        default_headers = {
            "HTTP-Referer": "https://github.com/BogdanSadovski/saou_nic",
            "X-Title": "RealSync Interview Platform",
        }
        self._client = AsyncOpenAI(
            api_key=api_key,
            base_url=base_url,
            default_headers=default_headers,
        )

    async def generate(
        self,
        prompt: str,
        system_prompt: Optional[str] = None,
        temperature: Optional[float] = None,
        max_tokens: Optional[int] = None,
        response_format: Optional[dict[str, Any]] = None,
    ) -> str:
        """Generate a completion. Retries on 429 (rate-limit / TPM) and
        bubbles everything else as RuntimeError.

        Groq's free tier in particular caps Tokens-Per-Minute at 12k
        which the large resume / github prompts blow through quickly.
        Without this retry the very first burst of resume/github
        analyses 429s and falls through to the local heuristic
        fallback — which is the "AI didn't analyze it" complaint.
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

        max_attempts = 3
        last_exc: Optional[BaseException] = None
        for attempt in range(1, max_attempts + 1):
            try:
                response = await self._client.chat.completions.create(**kwargs)
                content = response.choices[0].message.content
                if content is None:
                    raise RuntimeError("LLM returned empty response")
                logger.debug(
                    "LLM request completed: tokens_used=%s attempt=%d",
                    response.usage.total_tokens if response.usage else "unknown",
                    attempt,
                )
                return content
            except RateLimitError as exc:
                last_exc = exc
                wait = _extract_retry_after(exc)
                if attempt < max_attempts:
                    logger.warning(
                        "LLM rate-limit on attempt %d/%d, sleeping %.2fs",
                        attempt, max_attempts, wait,
                    )
                    await asyncio.sleep(wait)
                    continue
                logger.error("LLM rate-limit persisted after %d attempts", max_attempts)
                raise RuntimeError(f"LLM API error: {exc}") from exc
            except OpenAIError as exc:
                logger.error("LLM API error: %s", exc)
                raise RuntimeError(f"LLM API error: {exc}") from exc

        # Defensive — should never hit because we either return or raise above.
        raise RuntimeError(f"LLM API error: {last_exc}") from last_exc

    async def generate_json(
        self,
        prompt: str,
        system_prompt: Optional[str] = None,
        schema: Optional[dict[str, Any]] = None,
    ) -> dict[str, Any]:
        """Generate a JSON response, optionally validated against a schema."""
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
