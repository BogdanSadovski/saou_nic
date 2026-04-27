"""
Structured logging module using structlog.

Provides centralized logging configuration with JSON output,
request correlation IDs, and environment-aware log levels.
"""

import logging
import sys
from enum import Enum
from typing import Optional

import structlog
from structlog.types import Processor


class LogLevel(str, Enum):
    """Supported log levels."""

    DEBUG = "debug"
    INFO = "info"
    WARN = "warn"
    ERROR = "error"


def _get_log_level(level: LogLevel) -> int:
    """Map LogLevel enum to Python logging constants."""
    return {
        LogLevel.DEBUG: logging.DEBUG,
        LogLevel.INFO: logging.INFO,
        LogLevel.WARN: logging.WARNING,
        LogLevel.ERROR: logging.ERROR,
    }[level]


def setup_logging(
    level: LogLevel = LogLevel.INFO,
    environment: str = "development",
    service_name: str = "unknown",
    json_format: bool = True,
) -> None:
    """
    Configure structlog with processors and logging handlers.

    Args:
        level: Minimum log level to output.
        environment: Deployment environment (development, staging, production).
        service_name: Name of the service for log identification.
        json_format: If True, output JSON; if False, output console-friendly format.
    """
    processors: list[Processor] = [
        structlog.contextvars.merge_contextvars,
        structlog.processors.add_log_level,
        structlog.processors.StackInfoRenderer(),
        structlog.processors.TimeStamper(fmt="iso"),
        structlog.processors.UnicodeDecoder(),
    ]

    if json_format:
        processors.extend(
            [
                structlog.processors.format_exc_info,
                structlog.processors.JSONRenderer(),
            ]
        )
    else:
        processors.extend(
            [
                structlog.dev.ConsoleRenderer(colors=True),
            ]
        )

    structlog.configure(
        processors=processors,
        wrapper_class=structlog.make_filtering_bound_logger(_get_log_level(level)),
        context_class=dict,
        cache_logger_on_first_use=True,
        logger_factory=structlog.PrintLoggerFactory(),
    )

    # Configure standard logging to also route through structlog
    logging.basicConfig(
        format="%(message)s",
        level=_get_log_level(level),
        stream=sys.stdout,
    )

    # Bind service-level context to all log entries
    structlog.contextvars.bind_contextvars(
        service=service_name,
        environment=environment,
    )


def get_logger(
    name: Optional[str] = None,
    **kwargs: str,
) -> structlog.stdlib.BoundLogger:
    """
    Get a structured logger instance with optional context binding.

    Args:
        name: Logger name (typically module or class name).
        **kwargs: Additional context to bind to all log entries.

    Returns:
        Configured structlog BoundLogger instance.

    Example:
        >>> logger = get_logger(__name__, request_id="abc-123")
        >>> logger.info("user_logged_in", user_id=42)
    """
    if name:
        kwargs["logger"] = name
    return structlog.contextvars.bind_contextvars(**kwargs)
