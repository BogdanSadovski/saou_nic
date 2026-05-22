"""Code-executor FastAPI service.

Endpoints:
  GET  /health                — простой healthcheck для docker
  GET  /api/v1/sandbox/languages — список поддерживаемых языков
  POST /api/v1/sandbox/execute  — синхронный запуск, ждёт результат целиком
  WS   /api/v1/sandbox/stream   — то же, но с потоковой отдачей stdout/stderr
                                  (для красивой live-визуализации в редакторе)
"""

from __future__ import annotations

import asyncio
import json
import logging

from fastapi import FastAPI, WebSocket, WebSocketDisconnect
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel, Field

from .config import LANGUAGES, settings
from .runner import execute, run_in_docker, run_sqlite

logging.basicConfig(
    level=getattr(logging, settings.log_level.upper(), logging.INFO),
    format="%(asctime)s %(levelname)s %(name)s :: %(message)s",
)
logger = logging.getLogger("code-executor")

app = FastAPI(title="RealSync · Code Executor")

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=False,
    allow_methods=["*"],
    allow_headers=["*"],
)


class ExecuteRequest(BaseModel):
    language: str
    code: str = Field(..., max_length=64 * 1024)
    stdin: str | None = Field(None, max_length=8 * 1024)


class ExecuteResponse(BaseModel):
    language: str
    stdout: str
    stderr: str
    exit_code: int
    duration_ms: int
    timed_out: bool
    error: str | None = None


@app.get("/health")
async def health() -> dict[str, str]:
    return {"status": "ok"}


@app.get("/api/v1/sandbox/languages")
async def list_languages() -> dict:
    return {
        "languages": [
            {"id": spec.id, "label": spec.label, "in_process": spec.image == "builtin"}
            for spec in LANGUAGES.values()
        ],
        "limits": {
            "wall_timeout_sec": settings.wall_timeout_sec,
            "memory": settings.memory_limit,
            "cpu": settings.cpu_limit,
            "output_byte_cap": settings.output_byte_cap,
        },
    }


@app.post("/api/v1/sandbox/execute", response_model=ExecuteResponse)
async def post_execute(req: ExecuteRequest) -> ExecuteResponse:
    res = await execute(req.language, req.code, req.stdin)
    return ExecuteResponse(**res.__dict__)


# ───────────────────────── WebSocket: live-streaming ─────────────────────────


@app.websocket("/api/v1/sandbox/stream")
async def ws_stream(ws: WebSocket) -> None:
    """Live-стриминг stdout/stderr через WS.

    Протокол:
      Клиент шлёт первое сообщение JSON {language, code, stdin?}.
      Сервер шлёт обратно последовательность сообщений:
        {"type":"started",  "language": "..."}
        {"type":"stdout",   "chunk": "..."}    — может быть несколько
        {"type":"stderr",   "chunk": "..."}
        {"type":"done",     "exit_code": N, "duration_ms": N, "timed_out": bool}
    """
    await ws.accept()
    try:
        raw = await ws.receive_text()
        payload = json.loads(raw)
        language = payload.get("language", "")
        code = payload.get("code", "")
        stdin_data = payload.get("stdin")
    except Exception as exc:  # noqa: BLE001
        await ws.send_text(json.dumps({"type": "error", "error": f"bad request: {exc}"}))
        await ws.close()
        return

    await ws.send_text(json.dumps({"type": "started", "language": language}))

    if language == "sql":
        res = await run_sqlite(code)
        if res.stdout:
            await ws.send_text(json.dumps({"type": "stdout", "chunk": res.stdout}))
        if res.stderr:
            await ws.send_text(json.dumps({"type": "stderr", "chunk": res.stderr}))
        await ws.send_text(json.dumps({
            "type": "done",
            "exit_code": res.exit_code,
            "duration_ms": res.duration_ms,
            "timed_out": res.timed_out,
        }))
        await ws.close()
        return

    spec = LANGUAGES.get(language)
    if spec is None or spec.image == "builtin":
        await ws.send_text(json.dumps({"type": "error", "error": f"unsupported language: {language}"}))
        await ws.close()
        return

    # Для docker-based — пока поток грубее: запускаем sync, потом
    # отправляем накопленный stdout/stderr. v2 — построчный stream через
    # `docker logs -f`. На UX это влияет минимально, потому что лимит 5s.
    res = await run_in_docker(spec, code, stdin_data)
    if res.stdout:
        await ws.send_text(json.dumps({"type": "stdout", "chunk": res.stdout}))
    if res.stderr:
        await ws.send_text(json.dumps({"type": "stderr", "chunk": res.stderr}))
    await ws.send_text(json.dumps({
        "type": "done",
        "exit_code": res.exit_code,
        "duration_ms": res.duration_ms,
        "timed_out": res.timed_out,
    }))
    try:
        await ws.close()
    except Exception:  # noqa: BLE001
        pass


# Утилитарная пустышка, чтобы asyncio.create_task в будущем не падал
# из-за неимпортированного asyncio.
_ = asyncio
