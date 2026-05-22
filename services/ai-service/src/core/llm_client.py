"""LLM client wrapper for OpenAI-compatible APIs."""

import asyncio
import json
import logging
import re
from typing import Any, Optional

from openai import AsyncOpenAI, OpenAIError, RateLimitError

logger = logging.getLogger(__name__)


# How long we're willing to wait for a Retry-After hint before giving
# up and letting the router try the next tier. With a 4-tier cascade
# there's no point sleeping 55 minutes on Tier 1 — the next provider
# can answer in 2-5 seconds. Anything beyond this cap signals
# "exhausted for the day", and we surface the error so LLMRouter
# moves on immediately.
_MAX_RETRY_AFTER_SECONDS = 3.0


def _extract_retry_after(exc: BaseException) -> float:
    """Pull a sleep hint out of a RateLimitError.

    Both Groq and OpenRouter return JSON like
        ... 'Please try again in 10.815s' ...
    inside the error message. We parse that string first because the
    `Retry-After` header isn't always exposed through the SDK's
    `headers` attribute on async clients. Returns the *raw* requested
    wait in seconds, or 0 if the message didn't contain one. The
    caller decides whether to honour it (small wait → retry, big wait
    → give up and fail through to next tier).
    """
    text = str(exc)
    # 'try again in 10.815s' / 'retry after 12 seconds'
    m = re.search(r"(?:try again in|retry after)\s+([0-9]+(?:\.[0-9]+)?)\s*s", text, re.IGNORECASE)
    if m:
        try:
            return float(m.group(1))
        except ValueError:
            pass
    # 'try again in 1m30s' style — convert minutes to seconds.
    m = re.search(r"try again in\s+([0-9]+)m([0-9]+(?:\.[0-9]+)?)s", text, re.IGNORECASE)
    if m:
        try:
            return float(m.group(1)) * 60 + float(m.group(2))
        except ValueError:
            pass
    return 0.0


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
        self._base_url = (base_url or "").lower()
        # DeepSeek-у нет смысла отправлять response_format=json_schema:
        # /chat/completions у них сейчас возвращает 400 "This
        # response_format type is unavailable now". На каждый запрос
        # это пустые 3–5 секунд round-trip перед обязательным retry.
        # Помечаем такие клиенты, чтобы generate_json сразу шёл в
        # json_object со схемой в user-prompt.
        self._skip_json_schema = "deepseek.com" in self._base_url
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

        # With a multi-tier router behind us we only retry briefly on
        # rate-limit. If the provider asks us to wait more than
        # _MAX_RETRY_AFTER_SECONDS, give up — the next tier will
        # answer in 2-5 seconds, no point sleeping minutes.
        max_attempts = 2
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
                # Daily quota exhausted (Retry-After in minutes/hours) →
                # fail fast so the router can move to the next provider.
                if wait > _MAX_RETRY_AFTER_SECONDS or wait == 0:
                    logger.warning(
                        "LLM %s rate-limit: Retry-After=%.1fs exceeds cap %.1fs — failing through to next tier",
                        self._model, wait, _MAX_RETRY_AFTER_SECONDS,
                    )
                    raise RuntimeError(f"LLM API error: {exc}") from exc
                if attempt < max_attempts:
                    logger.warning(
                        "LLM %s rate-limit on attempt %d/%d, sleeping %.2fs",
                        self._model, attempt, max_attempts, wait,
                    )
                    await asyncio.sleep(wait)
                    continue
                logger.error("LLM %s rate-limit persisted after %d attempts", self._model, max_attempts)
                raise RuntimeError(f"LLM API error: {exc}") from exc
            except OpenAIError as exc:
                logger.error("LLM %s API error: %s", self._model, exc)
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
        # Для провайдеров, которые точно не поддерживают json_schema
        # (DeepSeek), пропускаем заведомо проигрышный первый запрос и
        # сразу строим запрос с json_object + схемой в user-prompt.
        # Экономит ~3–5 сек round-trip на каждом вызове.
        if schema and self._skip_json_schema:
            schema_hint = ""
            try:
                schema_body = schema.get("schema") if isinstance(schema, dict) else None
                if schema_body:
                    schema_hint = (
                        "\n\nReturn ONLY a JSON object that strictly matches "
                        "the following JSON Schema. Fill EVERY field with "
                        "values derived from the input above — do not echo "
                        "field names, descriptions, or example placeholders "
                        "as values:\n```json\n"
                        + json.dumps(schema_body, ensure_ascii=False, indent=2)
                        + "\n```"
                    )
            except Exception:
                schema_hint = ""
            raw = await self.generate(
                prompt=prompt + schema_hint,
                system_prompt=system_prompt or default_system,
                response_format={"type": "json_object"},
            )
            try:
                parsed = json.loads(raw)
                logger.debug("LLM JSON parsed ok (%d chars)", len(raw))
                return parsed
            except json.JSONDecodeError as exc:
                logger.error("Failed to parse LLM JSON response: %s", raw[:500])
                raise ValueError(f"LLM returned invalid JSON: {exc}") from exc

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
            # Без json_schema провайдер не знает форму ответа и склонен
            # вернуть структуру с placeholder-полями вместо реальных
            # данных (наблюдалось у DeepSeek: язык = "primary",
            # "secondary"; у Groq Llama-3.3-70b — то же самое). Подкладываем
            # JSON-схему прямо в user-prompt, чтобы модель видела
            # ожидаемые поля и описания.
            schema_hint = ""
            try:
                schema_body = schema.get("schema") if isinstance(schema, dict) else None
                if schema_body:
                    schema_hint = (
                        "\n\nReturn ONLY a JSON object that strictly matches "
                        "the following JSON Schema. Fill EVERY field with "
                        "values derived from the input above — do not echo "
                        "field names, descriptions, or example placeholders "
                        "as values:\n```json\n"
                        + json.dumps(schema_body, ensure_ascii=False, indent=2)
                        + "\n```"
                    )
            except Exception:
                schema_hint = ""
            raw = await self.generate(
                prompt=prompt + schema_hint,
                system_prompt=system_prompt or default_system,
                response_format={"type": "json_object"},
            )

        try:
            parsed = json.loads(raw)
            logger.debug("LLM JSON parsed ok (%d chars)", len(raw))
            return parsed
        except json.JSONDecodeError as exc:
            logger.error("Failed to parse LLM JSON response: %s", raw[:500])
            raise ValueError(
                f"LLM returned invalid JSON: {exc}"
            ) from exc

    @property
    def model(self) -> str:
        return self._model


class LLMPool:
    """Round-robin pool of LLM clients that share a provider.

    Used when you want to multiply the daily quota of a single
    provider (e.g. 5 OpenRouter API keys pointing at the same free
    DeepSeek model). Each call picks the next client in sequence; if
    that client raises, the pool transparently tries the next one
    until either someone succeeds or the whole pool is exhausted.

    The pool exposes the same `generate` / `generate_json` / `model`
    surface as :class:`LLMClient` so it slots into :class:`LLMRouter`
    as a single tier — a tier can be either a `LLMClient` or a
    `LLMPool`, the router doesn't care.

    Note: round-robin is per-process. With multiple ai-service
    replicas behind a load balancer each replica has its own pointer,
    which is fine — it still spreads load roughly evenly.
    """

    def __init__(self, clients: list["LLMClient"]) -> None:
        if not clients:
            raise ValueError("LLMPool requires at least one LLMClient")
        self._clients = clients
        self._cursor = 0
        self._lock = asyncio.Lock()

    @property
    def model(self) -> str:
        return self._clients[0].model

    @property
    def size(self) -> int:
        return len(self._clients)

    async def _next_index(self) -> int:
        async with self._lock:
            i = self._cursor
            self._cursor = (self._cursor + 1) % len(self._clients)
            return i

    async def generate(
        self,
        prompt: str,
        system_prompt: Optional[str] = None,
        temperature: Optional[float] = None,
        max_tokens: Optional[int] = None,
        response_format: Optional[dict[str, Any]] = None,
    ) -> str:
        start = await self._next_index()
        last_exc: Optional[BaseException] = None
        for offset in range(len(self._clients)):
            idx = (start + offset) % len(self._clients)
            client = self._clients[idx]
            try:
                return await client.generate(
                    prompt=prompt,
                    system_prompt=system_prompt,
                    temperature=temperature,
                    max_tokens=max_tokens,
                    response_format=response_format,
                )
            except (RuntimeError, OpenAIError) as exc:
                last_exc = exc
                # 404 = the model itself doesn't exist. All keys in the
                # pool point at the same model, so retrying the other
                # N-1 slots will produce the identical 404. Fail fast
                # so LLMRouter can move to the next tier in <1s.
                if "404" in str(exc) or "No endpoints found" in str(exc) or "does not exist" in str(exc):
                    logger.warning(
                        "LLMPool (%s) model not found (404) — skipping remaining %d slots",
                        client.model, len(self._clients) - 1,
                    )
                    raise RuntimeError(f"LLMPool model 404 ({client.model}): {exc}") from exc
                logger.warning(
                    "LLMPool slot %d/%d (%s) failed, trying next: %s",
                    idx + 1, len(self._clients), client.model, exc,
                )
                continue
        raise RuntimeError(f"LLMPool exhausted ({len(self._clients)} keys): {last_exc}")

    async def generate_json(
        self,
        prompt: str,
        system_prompt: Optional[str] = None,
        schema: Optional[dict[str, Any]] = None,
    ) -> dict[str, Any]:
        start = await self._next_index()
        last_exc: Optional[BaseException] = None
        for offset in range(len(self._clients)):
            idx = (start + offset) % len(self._clients)
            client = self._clients[idx]
            try:
                return await client.generate_json(
                    prompt=prompt,
                    system_prompt=system_prompt,
                    schema=schema,
                )
            except (RuntimeError, ValueError, OpenAIError) as exc:
                last_exc = exc
                if "404" in str(exc) or "No endpoints found" in str(exc) or "does not exist" in str(exc):
                    logger.warning(
                        "LLMPool (%s) model not found (404) — skipping remaining %d slots",
                        client.model, len(self._clients) - 1,
                    )
                    raise RuntimeError(f"LLMPool model 404 ({client.model}): {exc}") from exc
                logger.warning(
                    "LLMPool slot %d/%d (%s) JSON failed, trying next: %s",
                    idx + 1, len(self._clients), client.model, exc,
                )
                continue
        raise RuntimeError(f"LLMPool exhausted ({len(self._clients)} keys): {last_exc}")


class LLMRouter:
    """Cascading wrapper over multiple OpenAI-compatible LLM clients.

    Same public surface as :class:`LLMClient` (`generate`, `generate_json`,
    `model`) so call sites that depend on the existing client need no
    changes — only the DI container swaps in a router instead of a
    single client.

    The first client (Tier 1) is the primary provider. On
    :class:`RuntimeError` (rate-limit exhausted, server 5xx, JSON
    decode failure, etc.) the router moves to Tier 2, then Tier 3,
    then Tier 4. Only the *last* tier's exception propagates — earlier
    failures are logged as warnings so the caller still sees a useful
    error if everything is down.

    The cascade fires for every request independently; there is no
    sticky failover, no circuit breaker. Free tiers recover quickly,
    so always retrying from Tier 1 keeps quality high when Tier 1 is
    healthy.
    """

    def __init__(self, tiers: "list[LLMClient | LLMPool]") -> None:
        if not tiers:
            raise ValueError("LLMRouter requires at least one tier")
        self._clients = tiers

    @property
    def model(self) -> str:
        # Report the primary model; downstream logs are still accurate
        # because each client logs its own model on use.
        return self._clients[0].model

    async def generate(
        self,
        prompt: str,
        system_prompt: Optional[str] = None,
        temperature: Optional[float] = None,
        max_tokens: Optional[int] = None,
        response_format: Optional[dict[str, Any]] = None,
    ) -> str:
        last_exc: Optional[BaseException] = None
        for idx, client in enumerate(self._clients, start=1):
            try:
                return await client.generate(
                    prompt=prompt,
                    system_prompt=system_prompt,
                    temperature=temperature,
                    max_tokens=max_tokens,
                    response_format=response_format,
                )
            except (RuntimeError, OpenAIError) as exc:
                last_exc = exc
                if idx < len(self._clients):
                    logger.warning(
                        "LLM Tier %d (%s) failed, falling over to Tier %d: %s",
                        idx, client.model, idx + 1, exc,
                    )
                    continue
                logger.error(
                    "LLM Tier %d (%s) failed — no more providers in cascade: %s",
                    idx, client.model, exc,
                )
                raise
        # Defensive — should be unreachable.
        raise RuntimeError(f"LLM cascade exhausted: {last_exc}")

    async def generate_json(
        self,
        prompt: str,
        system_prompt: Optional[str] = None,
        schema: Optional[dict[str, Any]] = None,
    ) -> dict[str, Any]:
        last_exc: Optional[BaseException] = None
        for idx, client in enumerate(self._clients, start=1):
            try:
                return await client.generate_json(
                    prompt=prompt,
                    system_prompt=system_prompt,
                    schema=schema,
                )
            except (RuntimeError, ValueError, OpenAIError) as exc:
                last_exc = exc
                if idx < len(self._clients):
                    logger.warning(
                        "LLM Tier %d (%s) JSON failed, falling over to Tier %d: %s",
                        idx, client.model, idx + 1, exc,
                    )
                    continue
                logger.error(
                    "LLM Tier %d (%s) JSON failed — cascade exhausted: %s",
                    idx, client.model, exc,
                )
                raise
        raise RuntimeError(f"LLM cascade exhausted: {last_exc}")
