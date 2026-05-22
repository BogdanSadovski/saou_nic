"""Запуск кода в ephemeral docker-контейнере с жёсткими лимитами.

Ключевая идея: на каждый /execute мы создаём временную директорию,
пишем туда код, монтируем её read-only в контейнер и стартуем образ
конкретного языка с лимитами CPU/RAM/network/pids.

Безопасность (НЕ для prod, но достаточно для диплома/демо):
  • --network=none       — без сети
  • --memory=128m        — ограничение RAM
  • --cpus=0.5           — ½ ядра
  • --pids-limit=64      — не даём форк-бомбе
  • --read-only          — корневая ФС read-only
  • --tmpfs /tmp:size=16m — единственное место, куда можно писать
  • --cap-drop=ALL       — никаких capabilities
  • --security-opt=no-new-privileges
  • -u nobody (где возможно) — не root
  • wall-time через python-таймаут (subprocess.run timeout=N)

При timeout: SIGKILL → контейнер удаляется, возвращается duration=N сек
и exit_code=124 (классика timeout(1)).
"""

from __future__ import annotations

import asyncio
import logging
import os
import shutil
import sqlite3
import tempfile
import time
import uuid
from dataclasses import dataclass

from .config import LANGUAGES, LanguageSpec, settings

logger = logging.getLogger(__name__)


@dataclass
class ExecResult:
    language: str
    stdout: str
    stderr: str
    exit_code: int
    duration_ms: int
    timed_out: bool
    error: str | None = None


def _trim(data: bytes) -> str:
    """Обрезаем output до лимита, чтобы не вернуть 10MB в JSON."""
    cap = settings.output_byte_cap
    if len(data) > cap:
        return data[:cap].decode("utf-8", errors="replace") + f"\n…[output truncated at {cap} bytes]"
    return data.decode("utf-8", errors="replace")


async def _run_subprocess(
    cmd: list[str],
    timeout: int,
    stdin_data: str | None = None,
) -> tuple[bytes, bytes, int, bool]:
    """Запуск subprocess с wall-time таймаутом.

    Возвращает (stdout_bytes, stderr_bytes, returncode, timed_out).
    """
    proc = await asyncio.create_subprocess_exec(
        *cmd,
        stdin=asyncio.subprocess.PIPE if stdin_data else asyncio.subprocess.DEVNULL,
        stdout=asyncio.subprocess.PIPE,
        stderr=asyncio.subprocess.PIPE,
    )
    try:
        stdout, stderr = await asyncio.wait_for(
            proc.communicate(input=stdin_data.encode() if stdin_data else None),
            timeout=timeout,
        )
        return stdout, stderr, proc.returncode or 0, False
    except asyncio.TimeoutError:
        # Кидаем SIGKILL — это надёжнее, docker сам уберёт контейнер
        # (мы запускаем с --rm).
        try:
            proc.kill()
        except ProcessLookupError:
            pass
        try:
            stdout, stderr = await proc.communicate()
        except Exception:  # noqa: BLE001
            stdout, stderr = b"", b""
        return stdout, stderr, 124, True


async def run_in_docker(spec: LanguageSpec, code: str, stdin_data: str | None) -> ExecResult:
    """Запуск произвольного языка в одноразовом контейнере."""
    os.makedirs(settings.sandbox_scratch_dir, exist_ok=True)
    workdir = tempfile.mkdtemp(prefix="sbx-", dir=settings.sandbox_scratch_dir)
    # ВАЖНО: dockerd на хосте получает этот путь дословно. Поскольку
    # docker-compose монтирует /var/realsync-sandbox с одинаковым именем
    # на хост и в наш контейнер, path совпадает в обоих namespace'ах.
    try:
        os.chmod(workdir, 0o755)
    except OSError:
        pass
    try:
        path = os.path.join(workdir, spec.filename)
        with open(path, "w", encoding="utf-8") as f:
            f.write(code)

        # Go 1.22+ требует go.mod даже для одного файла. Создаём
        # минимальный модуль рядом, чтобы `go run main.go` сработал.
        if spec.id == "go":
            with open(os.path.join(workdir, "go.mod"), "w", encoding="utf-8") as f:
                f.write("module sandbox\n\ngo 1.22\n")

        container_name = f"sbx-{uuid.uuid4().hex[:12]}"
        # Лимиты: язык может переопределить (компилятор Go нуждается
        # в ≥ 250 MB, иначе OOM; cold compile ему нужно больше CPU).
        mem = spec.memory_limit or settings.memory_limit
        timeout = spec.wall_timeout_sec or settings.wall_timeout_sec
        cpu = spec.cpu_limit or settings.cpu_limit
        cmd = [
            settings.docker_bin, "run",
            "--rm",
            "--name", container_name,
            "--network=none",
            "--memory", mem,
            "--memory-swap", mem,
            "--cpus", cpu,
            "--pids-limit", str(settings.pids_limit),
            "--read-only",
            # /tmp = tmpfs 128MB — для Go-компилятора этого мало для
            # stdlib сборки с нуля ("no space left"), поэтому даём
            # больше; общий лимит RAM всё равно ограничен --memory.
            "--tmpfs", "/tmp:rw,size=128m,exec",
            # /sandbox — каталог с пользовательским файлом, read-only.
            "-v", f"{workdir}:/sandbox:ro",
            # Постоянный gocache между запусками — иначе каждый Go-run
            # компилирует stdlib с нуля (≥ 30 сек). Named volume
            # сохраняется dockerd'ом, не зависит от наших путей.
            "-v", "realsync_sandbox_gocache:/tmp/gocache",
            "-v", "realsync_sandbox_npm:/tmp/npm",
            "-w", "/sandbox",
            "--cap-drop=ALL",
            "--security-opt", "no-new-privileges",
            # для Node/Go/Python нужно $HOME — задаём в /tmp.
            "-e", "HOME=/tmp",
            "-e", "GOCACHE=/tmp/gocache",
            "-e", "GOMODCACHE=/tmp/gomodcache",
            "-e", "GOFLAGS=-mod=mod",
            "-e", "npm_config_cache=/tmp/npm",
            spec.image,
            *spec.run_cmd,
        ]

        start = time.perf_counter()
        try:
            stdout, stderr, rc, timed_out = await _run_subprocess(
                cmd, timeout=timeout, stdin_data=stdin_data,
            )
        finally:
            duration_ms = int((time.perf_counter() - start) * 1000)
            # Если контейнер ещё жив (timeout), убиваем по имени.
            if duration_ms >= timeout * 1000:
                try:
                    proc = await asyncio.create_subprocess_exec(
                        settings.docker_bin, "rm", "-f", container_name,
                        stdout=asyncio.subprocess.DEVNULL,
                        stderr=asyncio.subprocess.DEVNULL,
                    )
                    await asyncio.wait_for(proc.wait(), timeout=2)
                except Exception:  # noqa: BLE001
                    pass

        return ExecResult(
            language=spec.id,
            stdout=_trim(stdout),
            stderr=_trim(stderr),
            exit_code=rc,
            duration_ms=duration_ms,
            timed_out=timed_out,
            error=None,
        )
    finally:
        shutil.rmtree(workdir, ignore_errors=True)


# ─────────────────────────── SQLite (in-process) ───────────────────────────

# Демо-схема, на которой можно выполнять SELECT/INSERT/UPDATE/CREATE.
# Перед каждым прогоном схема загружается в новую in-memory БД.
SQL_SEED = """
CREATE TABLE users (
    id          INTEGER PRIMARY KEY,
    email       TEXT UNIQUE NOT NULL,
    full_name   TEXT NOT NULL,
    created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE orders (
    id          INTEGER PRIMARY KEY,
    user_id     INTEGER NOT NULL REFERENCES users(id),
    amount      REAL NOT NULL,
    status      TEXT NOT NULL CHECK(status IN ('pending','paid','refunded','cancelled')),
    created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

INSERT INTO users (id, email, full_name) VALUES
    (1, 'alice@example.com', 'Alice Ivanova'),
    (2, 'bob@example.com',   'Bob Petrov'),
    (3, 'carol@example.com', 'Carol Sidorova'),
    (4, 'dave@example.com',  'Dave Egorov'),
    (5, 'eve@example.com',   'Eve Mironova');

INSERT INTO orders (user_id, amount, status, created_at) VALUES
    (1, 120.50, 'paid',     datetime('now','-30 days')),
    (1,  35.00, 'paid',     datetime('now','-7 days')),
    (2, 250.00, 'paid',     datetime('now','-3 days')),
    (2,  10.00, 'refunded', datetime('now','-1 days')),
    (3,  75.00, 'pending',  datetime('now','-2 hours')),
    (4, 500.00, 'paid',     datetime('now','-15 days')),
    (4,  88.00, 'paid',     datetime('now','-1 days')),
    (5, 999.00, 'cancelled',datetime('now','-5 days'));
"""


def _format_sqlite_rows(cursor: sqlite3.Cursor) -> str:
    """Превращаем результат SELECT в красивую ASCII-табличку."""
    headers = [c[0] for c in cursor.description] if cursor.description else []
    rows = cursor.fetchall()
    if not headers:
        return ""
    # Конвертим значения в строки.
    str_rows = [[str(v) if v is not None else "NULL" for v in row] for row in rows]
    widths = [len(h) for h in headers]
    for row in str_rows:
        for i, cell in enumerate(row):
            widths[i] = max(widths[i], len(cell))
    sep = "+".join("-" * (w + 2) for w in widths)
    sep = f"+{sep}+"
    header_line = "| " + " | ".join(h.ljust(widths[i]) for i, h in enumerate(headers)) + " |"
    out = [sep, header_line, sep]
    for row in str_rows:
        out.append("| " + " | ".join(row[i].ljust(widths[i]) for i in range(len(row))) + " |")
    out.append(sep)
    out.append(f"({len(str_rows)} rows)")
    return "\n".join(out)


async def run_sqlite(code: str) -> ExecResult:
    """In-memory SQLite с предзагруженной демо-схемой."""
    start = time.perf_counter()

    def _exec_sync() -> tuple[str, str, int]:
        conn = sqlite3.connect(":memory:")
        conn.row_factory = sqlite3.Row
        try:
            cur = conn.cursor()
            cur.executescript(SQL_SEED)
            conn.commit()

            chunks: list[str] = []
            # Разбиваем пользовательский SQL на стейтменты по ';'.
            stmts = [s.strip() for s in code.split(";") if s.strip()]
            for stmt in stmts:
                try:
                    cur.execute(stmt)
                    if cur.description:
                        chunks.append(f"-- {stmt[:60].replace(chr(10), ' ')}…")
                        chunks.append(_format_sqlite_rows(cur))
                        chunks.append("")
                    else:
                        chunks.append(f"-- {stmt[:60].replace(chr(10), ' ')}…")
                        chunks.append(f"({cur.rowcount} rows affected)")
                        chunks.append("")
                except sqlite3.Error as e:
                    return ("\n".join(chunks), f"SQL error: {e}\n  in statement: {stmt}", 1)
            conn.commit()
            return ("\n".join(chunks), "", 0)
        finally:
            conn.close()

    loop = asyncio.get_event_loop()
    try:
        stdout, stderr, rc = await asyncio.wait_for(
            loop.run_in_executor(None, _exec_sync),
            timeout=settings.wall_timeout_sec,
        )
        timed_out = False
    except asyncio.TimeoutError:
        stdout, stderr, rc, timed_out = "", "SQL execution timed out", 124, True

    duration_ms = int((time.perf_counter() - start) * 1000)
    return ExecResult(
        language="sql",
        stdout=_trim(stdout.encode("utf-8")),
        stderr=_trim(stderr.encode("utf-8")),
        exit_code=rc,
        duration_ms=duration_ms,
        timed_out=timed_out,
        error=None,
    )


# ─────────────────────────── public dispatch ───────────────────────────


async def execute(language: str, code: str, stdin_data: str | None = None) -> ExecResult:
    """Главный entrypoint. Выбирает между sqlite и docker по `language`."""
    if not language or not code:
        return ExecResult(
            language=language or "",
            stdout="", stderr="empty code", exit_code=2, duration_ms=0,
            timed_out=False, error="empty_code",
        )

    if language == "sql":
        return await run_sqlite(code)

    spec = LANGUAGES.get(language)
    if spec is None or spec.image == "builtin":
        return ExecResult(
            language=language,
            stdout="", stderr=f"unsupported language: {language}",
            exit_code=2, duration_ms=0, timed_out=False, error="unsupported_language",
        )

    return await run_in_docker(spec, code, stdin_data)
