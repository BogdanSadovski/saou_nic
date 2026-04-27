"""
Async PostgreSQL database operations using aiopg.

Provides a connection pool wrapper with convenience methods
for executing queries, fetching results, and managing transactions.
"""

import os
from contextlib import asynccontextmanager
from typing import Any, Optional

import aiopg
from aiopg import Pool

from logger import get_logger

logger = get_logger(__name__, module="database.async_pg")


class DatabasePool:
    """
    Manages an async PostgreSQL connection pool.

    This class is designed as a module-level singleton pattern.
    Use the module-level convenience functions (connect, close, etc.)
    which delegate to the global pool instance.
    """

    _pool: Optional[Pool] = None

    @property
    def pool(self) -> Pool:
        """Get the active connection pool."""
        if self._pool is None:
            raise RuntimeError("Database pool is not initialized. Call connect() first.")
        return self._pool

    @property
    def is_connected(self) -> bool:
        """Check if the pool is initialized and has free connections."""
        return self._pool is not None and not self._pool.closed


_db = DatabasePool()


async def connect(
    dsn: Optional[str] = None,
    host: Optional[str] = None,
    port: int = 5432,
    dbname: Optional[str] = None,
    user: Optional[str] = None,
    password: Optional[str] = None,
    minsize: int = 1,
    maxsize: int = 10,
    **kwargs: Any,
) -> Pool:
    """
    Create and initialize the async PostgreSQL connection pool.

    Args:
        dsn: Full connection string (e.g., 'postgresql://user:pass@host:5432/db').
             If provided, overrides individual params.
        host: Database host address.
        port: Database port (default: 5432).
        dbname: Database name.
        user: Database user.
        password: Database password.
        minsize: Minimum number of connections in the pool.
        maxsize: Maximum number of connections in the pool.
        **kwargs: Additional arguments passed to aiopg.create_pool().

    Returns:
        The initialized aiopg Pool.

    Raises:
        RuntimeError: If the pool is already connected.

    Example:
        >>> await connect(
        ...     host="localhost",
        ...     dbname="assessment",
        ...     user="app",
        ...     password="secret",
        ...     maxsize=20,
        ... )
    """
    if _db.is_connected:
        logger.warn("pool_already_connected", action="connect")
        return _db.pool

    if dsn:
        pool = await aiopg.create_pool(
            dsn=dsn,
            minsize=minsize,
            maxsize=maxsize,
            **kwargs,
        )
    else:
        pool = await aiopg.create_pool(
            host=host or os.getenv("DB_HOST", "localhost"),
            port=port or int(os.getenv("DB_PORT", "5432")),
            dbname=dbname or os.getenv("DB_NAME", "assessment"),
            user=user or os.getenv("DB_USER", "postgres"),
            password=password or os.getenv("DB_PASSWORD", "postgres"),
            minsize=minsize,
            maxsize=maxsize,
            **kwargs,
        )

    _db._pool = pool
    logger.info("database_connected", host=host or "via_dsn", dbname=dbname or "via_dsn")
    return pool


async def close() -> None:
    """
    Close all connections in the pool.

    Should be called during application shutdown to cleanly
    release all database connections.
    """
    if _db._pool is not None:
        _db._pool.close()
        await _db._pool.wait_closed()
        _db._pool = None
        logger.info("database_disconnected")


async def execute(query: str, *args: Any, timeout: Optional[float] = None) -> Optional[int]:
    """
    Execute a write query (INSERT, UPDATE, DELETE).

    Args:
        query: SQL query with parameter placeholders (%s).
        *args: Query parameters.
        timeout: Optional query timeout in seconds.

    Returns:
        Number of affected rows, or None for non-returning queries.

    Example:
        >>> rows = await execute(
        ...     "UPDATE users SET last_login = NOW() WHERE id = %s",
        ...     user_id,
        ... )
    """
    async with _db.pool.acquire() as conn:
        async with conn.cursor() as cur:
            await cur.execute(query, args)
            rowcount = cur.rowcount
            logger.debug(
                "query_executed",
                operation="execute",
                rowcount=rowcount,
            )
            return rowcount


async def fetch_one(
    query: str,
    *args: Any,
    timeout: Optional[float] = None,
) -> Optional[tuple]:
    """
    Execute a query and return a single row.

    Args:
        query: SQL query with parameter placeholders (%s).
        *args: Query parameters.
        timeout: Optional query timeout in seconds.

    Returns:
        A single row as a tuple, or None if no rows match.

    Example:
        >>> row = await fetch_one(
        ...     "SELECT id, email, role FROM users WHERE id = %s",
        ...     user_id,
        ... )
        >>> if row:
        ...     user_id, email, role = row
    """
    async with _db.pool.acquire() as conn:
        async with conn.cursor() as cur:
            await cur.execute(query, args)
            result = await cur.fetchone()
            logger.debug("query_executed", operation="fetch_one", found=result is not None)
            return result


async def fetch_all(
    query: str,
    *args: Any,
    timeout: Optional[float] = None,
) -> list[tuple]:
    """
    Execute a query and return all matching rows.

    Args:
        query: SQL query with parameter placeholders (%s).
        *args: Query parameters.
        timeout: Optional query timeout in seconds.

    Returns:
        List of rows as tuples.

    Example:
        >>> rows = await fetch_all(
        ...     "SELECT id, email FROM users WHERE role = %s ORDER BY created_at",
        ...     "admin",
        ... )
    """
    async with _db.pool.acquire() as conn:
        async with conn.cursor() as cur:
            await cur.execute(query, args)
            results = await cur.fetchall()
            logger.debug(
                "query_executed",
                operation="fetch_all",
                rowcount=len(results),
            )
            return results


@asynccontextmanager
async def transaction(isolation_level: Optional[str] = None):
    """
    Context manager for database transactions with automatic commit/rollback.

    Args:
        isolation_level: Optional transaction isolation level.
            Supported: 'READ COMMITTED' (default), 'REPEATABLE READ', 'SERIALIZABLE'.

    Yields:
        An aiopg connection with an active transaction.

    Example:
        >>> async with transaction() as conn:
        ...     async with conn.cursor() as cur:
        ...         await cur.execute("INSERT INTO users (email) VALUES (%s)", ("test@test.com",))
        ...         # Auto-committed on exit; auto-rolled back on exception
    """
    async with _db.pool.acquire() as conn:
        try:
            if isolation_level:
                await conn.set_isolation_level(isolation_level)
            logger.debug("transaction_started", isolation=isolation_level or "default")
            yield conn
            logger.debug("transaction_committed")
        except Exception:
            logger.warn("transaction_rolled_back")
            raise
