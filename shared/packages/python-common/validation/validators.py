"""
Input validation utilities.

Provides standalone validator functions for common input validation
scenarios: required fields, email format, length constraints,
and regex pattern matching.
"""

import re
from typing import Any, Optional


class ValidationError(Exception):
    """Raised when input fails validation."""

    def __init__(self, field: str, message: str):
        self.field = field
        self.message = message
        super().__init__(f"Validation error for '{field}': {message}")


# Precompiled regex for email validation (RFC 5322 simplified)
EMAIL_REGEX = re.compile(
    r"^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@"
    r"[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?"
    r"(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$"
)


def validate_required(
    value: Any,
    field: str = "field",
    allow_empty_string: bool = False,
) -> Any:
    """
    Validate that a value is present (not None).

    Args:
        value: The value to check.
        field: Field name for error messages.
        allow_empty_string: If False, empty strings are treated as missing.

    Returns:
        The original value if valid.

    Raises:
        ValidationError: If the value is None or an empty string.

    Example:
        >>> validate_required("hello", "username")
        'hello'
        >>> validate_required(None, "username")
        ValidationError: Validation error for 'username': is required
    """
    if value is None:
        raise ValidationError(field, "is required")
    if not allow_empty_string and isinstance(value, str) and value.strip() == "":
        raise ValidationError(field, "cannot be empty")
    return value


def validate_email(
    value: str,
    field: str = "email",
    max_length: int = 254,
) -> str:
    """
    Validate that a string is a properly formatted email address.

    Args:
        value: The email string to validate.
        field: Field name for error messages.
        max_length: Maximum allowed email length per RFC 5321.

    Returns:
        The trimmed, lowercased email string.

    Raises:
        ValidationError: If the email format is invalid.

    Example:
        >>> validate_email("user@example.com")
        'user@example.com'
        >>> validate_email("not-an-email")
        ValidationError: Validation error for 'email': invalid email format
    """
    validate_required(value, field)

    email = value.strip().lower()

    if len(email) > max_length:
        raise ValidationError(
            field, f"exceeds maximum length of {max_length} characters"
        )

    if not EMAIL_REGEX.match(email):
        raise ValidationError(field, "invalid email format")

    return email


def validate_length(
    value: str,
    field: str = "field",
    min_length: int = 0,
    max_length: Optional[int] = None,
) -> str:
    """
    Validate that a string falls within length bounds.

    Args:
        value: The string to validate.
        field: Field name for error messages.
        min_length: Minimum required length (inclusive).
        max_length: Maximum allowed length (inclusive). None for no limit.

    Returns:
        The original string if valid.

    Raises:
        ValidationError: If the length is out of bounds.

    Example:
        >>> validate_length("hello", "name", min_length=2, max_length=50)
        'hello'
        >>> validate_length("a", "name", min_length=2)
        ValidationError: Validation error for 'name': minimum length is 2
    """
    validate_required(value, field)

    length = len(value)

    if length < min_length:
        raise ValidationError(
            field, f"minimum length is {min_length} (got {length})"
        )

    if max_length is not None and length > max_length:
        raise ValidationError(
            field, f"maximum length is {max_length} (got {length})"
        )

    return value


def validate_regex(
    value: str,
    pattern: str,
    field: str = "field",
    flags: int = 0,
) -> str:
    """
    Validate that a string matches a regex pattern.

    Args:
        value: The string to validate.
        pattern: Regex pattern to match against.
        field: Field name for error messages.
        flags: Optional regex flags (e.g., re.IGNORECASE).

    Returns:
        The original string if it matches.

    Raises:
        ValidationError: If the string does not match the pattern.

    Example:
        >>> validate_regex("ABC-123", r"^[A-Z]{3}-\d{3}$", "code")
        'ABC-123'
        >>> validate_regex("abc", r"^[A-Z]+$", "code")
        ValidationError: Validation error for 'code': does not match required pattern
    """
    validate_required(value, field)

    if not re.match(pattern, value, flags):
        raise ValidationError(field, f"does not match required pattern: {pattern}")

    return value
