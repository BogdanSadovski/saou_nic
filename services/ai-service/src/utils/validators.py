"""Input validation utilities."""

import re
from typing import Optional


class ValidationError(Exception):
    """Custom exception for validation errors."""

    def __init__(self, field: str, message: str) -> None:
        self.field = field
        self.message = message
        super().__init__(f"Validation error in '{field}': {message}")


def validate_text_length(
    text: str,
    min_length: int = 1,
    max_length: int = 10000,
    field_name: str = "text",
) -> None:
    """Validate text length is within bounds.

    Args:
        text: Text to validate.
        min_length: Minimum allowed length.
        max_length: Maximum allowed length.
        field_name: Name of the field for error messages.

    Raises:
        ValueError: If text length is out of bounds.
    """
    if not text:
        raise ValueError(f"{field_name} cannot be empty")
    if len(text) < min_length:
        raise ValueError(
            f"{field_name} must be at least {min_length} characters"
        )
    if len(text) > max_length:
        raise ValueError(
            f"{field_name} must not exceed {max_length} characters"
        )


def validate_topic(topic: str) -> str:
    """Validate and clean a topic string.

    Args:
        topic: Topic text to validate.

    Returns:
        Cleaned topic string.

    Raises:
        ValueError: If topic is invalid.
    """
    if not topic or not topic.strip():
        raise ValueError("Topic cannot be empty")

    cleaned = topic.strip()
    if len(cleaned) < 2:
        raise ValueError("Topic must be at least 2 characters")
    if len(cleaned) > 500:
        raise ValueError("Topic must not exceed 500 characters")

    # Check for excessive special characters
    special_char_ratio = len(re.findall(r"[^a-zA-Z0-9\s\u0400-\u04ff]", cleaned)) / len(cleaned)
    if special_char_ratio > 0.3:
        raise ValueError("Topic contains too many special characters")

    return cleaned


def validate_difficulty(level: str) -> str:
    """Validate difficulty level.

    Args:
        level: Difficulty string.

    Returns:
        Normalized difficulty level.

    Raises:
        ValueError: If level is invalid.
    """
    valid_levels = {"easy", "medium", "hard"}
    normalized = level.lower().strip()
    if normalized not in valid_levels:
        raise ValueError(
            f"Invalid difficulty: {level}. Must be one of {valid_levels}"
        )
    return normalized


def validate_language_code(code: Optional[str]) -> Optional[str]:
    """Validate an ISO 639-1 language code.

    Args:
        code: Language code (e.g., 'en', 'ru').

    Returns:
        Validated code or None.

    Raises:
        ValueError: If code format is invalid.
    """
    if code is None:
        return None

    if not re.match(r"^[a-z]{2}$", code):
        raise ValueError(
            f"Invalid language code: {code}. Must be a 2-letter ISO 639-1 code"
        )
    return code


def validate_question_count(count: int) -> int:
    """Validate the number of questions requested.

    Args:
        count: Number of questions.

    Returns:
        Validated count.

    Raises:
        ValueError: If count is out of range.
    """
    if count < 1:
        raise ValueError("Must request at least 1 question")
    if count > 50:
        raise ValueError("Cannot generate more than 50 questions at once")
    return count
