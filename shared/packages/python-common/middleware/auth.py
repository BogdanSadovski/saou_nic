"""
JWT Authentication Middleware for FastAPI.

Provides token extraction, validation, and user context injection
for protected API endpoints.
"""

import os
from typing import Optional

import jwt
from fastapi import Depends, HTTPException, Request, status
from fastapi.security import HTTPAuthorizationCredentials, HTTPBearer
from pydantic import BaseModel

from logger import get_logger

logger = get_logger(__name__, module="middleware.auth")

JWT_SECRET = os.getenv("JWT_SECRET", "change-me-in-production")
JWT_ALGORITHM = os.getenv("JWT_ALGORITHM", "HS256")
AUTH_HEADER_PREFIX = "Bearer"

security = HTTPBearer(auto_error=False)


class UserContext(BaseModel):
    """Authenticated user context extracted from JWT token."""

    user_id: str
    email: str
    role: str = "user"
    tenant_id: Optional[str] = None


class JWTAuthMiddleware:
    """
    FastAPI middleware for JWT-based authentication.

    Extracts the Bearer token from the Authorization header,
    validates it, and attaches user context to request state.

    Usage:
        from fastapi import FastAPI
        from middleware.auth import JWTAuthMiddleware

        app = FastAPI()
        app.middleware("http")(JWTAuthMiddleware.dispatch)
    """

    @staticmethod
    async def dispatch(request: Request, call_next):
        """
        Process each incoming HTTP request.

        Skips authentication for paths in the whitelist (e.g., health checks,
        public endpoints, login/registration). For all other paths, validates
        the JWT token and sets `request.state.user` with the UserContext.
        """
        whitelist = ["/health", "/healthz", "/ready", "/metrics", "/docs", "/openapi.json"]
        if any(request.url.path.startswith(path) for path in whitelist):
            return await call_next(request)

        # Skip auth for POST to /auth endpoints (login, register, refresh)
        if request.url.path.startswith("/auth") and request.method in ("POST",):
            return await call_next(request)

        credentials = await security(request)

        if credentials is None:
            raise HTTPException(
                status_code=status.HTTP_401_UNAUTHORIZED,
                detail="Missing authentication token",
                headers={"WWW-Authenticate": "Bearer"},
            )

        try:
            user = decode_token(credentials.credentials)
            request.state.user = user
            logger.debug(
                "authenticated_request",
                user_id=user.user_id,
                path=request.url.path,
            )
        except jwt.ExpiredSignatureError:
            raise HTTPException(
                status_code=status.HTTP_401_UNAUTHORIZED,
                detail="Token has expired",
                headers={"WWW-Authenticate": "Bearer"},
            )
        except jwt.InvalidTokenError as e:
            logger.warn("invalid_token", error=str(e))
            raise HTTPException(
                status_code=status.HTTP_401_UNAUTHORIZED,
                detail="Invalid authentication token",
                headers={"WWW-Authenticate": "Bearer"},
            )

        return await call_next(request)


def decode_token(token: str) -> UserContext:
    """
    Decode and validate a JWT token, returning UserContext.

    Args:
        token: The raw JWT token string.

    Returns:
        UserContext with user_id, email, role, and optional tenant_id.

    Raises:
        jwt.InvalidTokenError: If the token is malformed or invalid.
    """
    payload = jwt.decode(
        token,
        JWT_SECRET,
        algorithms=[JWT_ALGORITHM],
        options={
            "require": ["sub", "email", "role"],
            "verify_signature": True,
        },
    )

    return UserContext(
        user_id=str(payload["sub"]),
        email=payload["email"],
        role=payload.get("role", "user"),
        tenant_id=payload.get("tenant_id"),
    )


async def get_current_user(
    request: Request,
) -> UserContext:
    """
    FastAPI dependency to get the authenticated user from request state.

    Use this as a dependency in route handlers to require authentication.

    Example:
        @app.get("/profile")
        async def get_profile(user: UserContext = Depends(get_current_user)):
            return {"user_id": user.user_id, "email": user.email}

    Raises:
        HTTPException: 401 if user is not authenticated.
    """
    user = getattr(request.state, "user", None)
    if user is None:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Not authenticated",
        )
    return user


async def require_roles(
    allowed_roles: list[str],
) -> callable:
    """
    Create a dependency that requires the user to have one of the allowed roles.

    Example:
        @app.delete("/users/{user_id}")
        async def delete_user(
            user: UserContext = Depends(require_roles(["admin", "moderator"]))
        ):
            ...
    """

    async def _check_role(request: Request) -> UserContext:
        user = await get_current_user(request)
        if user.role not in allowed_roles:
            raise HTTPException(
                status_code=status.HTTP_403_FORBIDDEN,
                detail=f"Role '{user.role}' is not authorized. Required: {allowed_roles}",
            )
        return user

    return _check_role


async def require_auth(
    request: Request,
) -> UserContext:
    """Alias for get_current_user for backward compatibility."""
    return await get_current_user(request)
