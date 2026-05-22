"""Config: лимиты и образы для каждого языка.

Все env переопределяемы — но дефолты подобраны для безопасной dev-инсталляции.
"""

from __future__ import annotations

from dataclasses import dataclass

from pydantic import Field
from pydantic_settings import BaseSettings, SettingsConfigDict


@dataclass(frozen=True)
class LanguageSpec:
    """Параметры одного language-runner."""
    id: str
    label: str
    image: str
    filename: str           # как сохраняется код в /sandbox/code
    run_cmd: list[str]      # команда внутри контейнера, $FILE = подставляется
    compile_cmd: list[str] | None = None  # опционально (Java)
    # Per-language overrides — компилируемые языки требуют больше RAM
    # (Go-компилятор ≥ 200 MB), интерпретируемые довольствуются 64.
    memory_limit: str | None = None
    wall_timeout_sec: int | None = None
    cpu_limit: str | None = None


# Алиасы для frontend'а — он шлёт "go", "python", "js", ...
LANGUAGES: dict[str, LanguageSpec] = {
    "python": LanguageSpec(
        id="python",
        label="Python 3.12",
        image="python:3.12-alpine",
        filename="main.py",
        run_cmd=["python", "/sandbox/main.py"],
    ),
    "go": LanguageSpec(
        id="go",
        label="Go 1.22",
        image="golang:1.22-alpine",
        filename="main.go",
        run_cmd=["go", "run", "/sandbox/main.go"],
        memory_limit="512m",  # компилятор кушает ~250MB, +stdlib
        wall_timeout_sec=45,  # cold start compile в Docker Desktop ≥ 30с
        cpu_limit="2",        # компилятор однопоточный, но stdlib параллельно
    ),
    "javascript": LanguageSpec(
        id="javascript",
        label="Node 20",
        image="node:20-alpine",
        filename="main.js",
        run_cmd=["node", "/sandbox/main.js"],
    ),
    "typescript": LanguageSpec(
        id="typescript",
        label="TypeScript (tsx)",
        image="node:20-alpine",
        filename="main.ts",
        # Используем `tsx`-loader через npx — компилит и сразу запускает.
        # Чтобы не качать пакеты каждый раз, --yes гарантирует non-interactive.
        run_cmd=["npx", "--yes", "tsx", "/sandbox/main.ts"],
        memory_limit="256m",
        wall_timeout_sec=15,
    ),
    "bash": LanguageSpec(
        id="bash",
        label="Bash",
        image="bash:5.2",
        filename="main.sh",
        run_cmd=["bash", "/sandbox/main.sh"],
    ),
    # SQL обрабатывается отдельно в-процессе через sqlite3 stdlib.
    # Здесь только метка для list-API.
    "sql": LanguageSpec(
        id="sql",
        label="SQLite (in-memory)",
        image="builtin",
        filename="main.sql",
        run_cmd=[],
    ),
}


class Settings(BaseSettings):
    model_config = SettingsConfigDict(env_file=None, case_sensitive=False)

    # Лимиты на ВЫПОЛНЕНИЕ (передаются в docker run).
    cpu_limit: str = Field("0.5", description="--cpus")
    memory_limit: str = Field("128m", description="--memory")
    pids_limit: int = Field(64)
    wall_timeout_sec: int = Field(5)
    output_byte_cap: int = Field(64 * 1024, description="trim stdout+stderr to this size")

    # docker binary path
    docker_bin: str = Field("docker")

    # Куда писать временные файлы. ВАЖНО: должен быть РАВЕН пути,
    # под которым этот же каталог смонтирован на хосте (docker.sock
    # шлёт пути dockerd'у, который не знает про наш контейнер).
    sandbox_scratch_dir: str = Field("/tmp/realsync-sandbox")

    log_level: str = Field("INFO")


settings = Settings()
