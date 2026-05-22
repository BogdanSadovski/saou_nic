"""Минт JWT от лица привязанного пользователя.

Используем тот же JWT_SECRET, что и user-service (см. compose env).
Сервисы внутри платформы (interview-service, resume-service,
admin-service) валидируют этот секрет через jwt.Parse(secret) — нам
не нужно ходить в user-service для получения токена.

Claims-структура совпадает с user-service/pkg/jwt/token.go:
    {"user_id":"<uuid>","email":"...","role":"...", iat, exp, iss}
"""

from __future__ import annotations

import time
import uuid

import jwt

from .config import settings


def mint_user_token(
    user_id: uuid.UUID | str,
    email: str = "",
    role: str = "candidate",
) -> str:
    now = int(time.time())
    payload = {
        "user_id": str(user_id),
        "email": email,
        "role": role,
        "iat": now,
        "exp": now + settings.jwt_ttl_sec,
        "iss": settings.jwt_issuer,
    }
    return jwt.encode(payload, settings.jwt_secret, algorithm="HS256")


def auth_headers(user_id: uuid.UUID | str, email: str = "", role: str = "candidate") -> dict[str, str]:
    return {"Authorization": f"Bearer {mint_user_token(user_id, email, role)}"}
