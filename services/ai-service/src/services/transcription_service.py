"""Speech-to-text transcription service."""

import io
import logging
from typing import Optional

from openai import AsyncOpenAI, OpenAIError

from src.models.responses import TranscriptionResponse

logger = logging.getLogger(__name__)


class TranscriptionService:
    """Service for transcribing audio to text using Whisper."""

    def __init__(
        self,
        api_key: str,
        model: str = "whisper-1",
        max_file_size_mb: int = 25,
        base_url: Optional[str] = None,
    ) -> None:
        self._model = model
        self._max_file_size_bytes = max_file_size_mb * 1024 * 1024
        self._client = AsyncOpenAI(
            api_key=api_key,
            base_url=base_url,
        )

    async def transcribe(
        self,
        audio_data: bytes,
        language: Optional[str] = None,
    ) -> TranscriptionResponse:
        """Transcribe audio data to text.

        Args:
            audio_data: Raw audio bytes.
            language: Optional language code for transcription.

        Returns:
            TranscriptionResponse with transcribed text.

        Raises:
            ValueError: If audio data is empty or too large.
            RuntimeError: If the API call fails.
        """
        self._validate_audio(audio_data)

        file_obj = io.BytesIO(audio_data)
        file_obj.name = "audio.wav"

        kwargs: dict = {
            "file": file_obj,
            "model": self._model,
        }

        if language:
            kwargs["language"] = language

        try:
            response = await self._client.audio.transcriptions.create(
                **kwargs
            )
            text = response.text

            logger.info(
                "Transcription completed: %d characters", len(text)
            )

            return TranscriptionResponse(
                text=text,
                language=language,
            )
        except OpenAIError as exc:
            logger.error("Transcription API error: %s", exc)
            raise RuntimeError(f"Transcription failed: {exc}") from exc

    def _validate_audio(self, audio_data: bytes) -> None:
        """Validate audio data before sending for transcription.

        Args:
            audio_data: Raw audio bytes.

        Raises:
            ValueError: If audio is invalid.
        """
        if not audio_data:
            raise ValueError("Audio data cannot be empty")

        if len(audio_data) > self._max_file_size_bytes:
            max_mb = self._max_file_size_bytes / (1024 * 1024)
            raise ValueError(
                f"Audio file too large. Maximum size: {max_mb:.0f}MB"
            )
