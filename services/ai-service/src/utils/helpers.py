"""Utility helper functions."""

import hashlib
import uuid
from datetime import datetime, timezone
from typing import Any


def generate_id() -> str:
    """Generate a unique identifier."""
    return str(uuid.uuid4())


def compute_text_hash(text: str, algorithm: str = "sha256") -> str:
    """Compute a hash of the given text.

    Args:
        text: Input text to hash.
        algorithm: Hash algorithm to use.

    Returns:
        Hex digest of the hash.
    """
    return hashlib.new(algorithm, text.encode("utf-8")).hexdigest()


def truncate_text(text: str, max_length: int = 100, suffix: str = "...") -> str:
    """Truncate text to a maximum length.

    Args:
        text: Input text.
        max_length: Maximum length including suffix.
        suffix: Suffix to append when truncated.

    Returns:
        Truncated text.
    """
    if len(text) <= max_length:
        return text
    return text[: max_length - len(suffix)] + suffix


def sanitize_text(text: str) -> str:
    """Clean and normalize text content.

    - Strips leading/trailing whitespace
    - Collapses multiple spaces into one
    - Removes control characters except newlines

    Args:
        text: Input text to sanitize.

    Returns:
        Cleaned text.
    """
    import re

    # Remove control characters except newlines and tabs
    cleaned = re.sub(r"[\x00-\x08\x0b\x0c\x0e-\x1f\x7f]", "", text)
    # Collapse multiple spaces
    cleaned = re.sub(r" +", " ", cleaned)
    return cleaned.strip()


def format_duration(seconds: float) -> str:
    """Format seconds into a human-readable duration string.

    Args:
        seconds: Duration in seconds.

    Returns:
        Formatted string (e.g., "1h 23m 45s").
    """
    hours = int(seconds // 3600)
    minutes = int((seconds % 3600) // 60)
    secs = int(seconds % 60)

    parts = []
    if hours:
        parts.append(f"{hours}h")
    if minutes:
        parts.append(f"{minutes}m")
    parts.append(f"{secs}s")
    return " ".join(parts)


def now_utc() -> datetime:
    """Get the current UTC timestamp."""
    return datetime.now(timezone.utc)


def safe_get(data: dict[str, Any], *keys: str, default: Any = None) -> Any:
    """Safely access nested dictionary keys.

    Args:
        data: Dictionary to traverse.
        *keys: Sequence of keys to access.
        default: Value to return if any key is missing.

    Returns:
        The nested value or default.

    Example:
        safe_get(data, "scores", "correctness", default=0)
    """
    current: Any = data
    for key in keys:
        if isinstance(current, dict):
            current = current.get(key, default)
        else:
            return default
    return current
