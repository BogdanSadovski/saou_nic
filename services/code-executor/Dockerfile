FROM python:3.12-slim

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1

# Нужен docker CLI, чтобы запускать ephemeral-runner'ы для каждого
# языка. docker.sock прокидывается с хоста (см. docker-compose.yml).
# Ставим только CLI-бинарник (не daemon) — никакого dockerd в этом
# контейнере. Используем официальный static-tarball от Docker, потому
# что debian-slim не всегда тянет docker.io корректно.
ARG DOCKER_VERSION=24.0.7
RUN apt-get update && apt-get install -y --no-install-recommends \
        curl ca-certificates sqlite3 \
    && curl -fsSL "https://download.docker.com/linux/static/stable/x86_64/docker-${DOCKER_VERSION}.tgz" \
        | tar -xz -C /tmp \
    && mv /tmp/docker/docker /usr/local/bin/docker \
    && rm -rf /tmp/docker \
    && chmod +x /usr/local/bin/docker \
    && apt-get purge -y curl \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt

COPY app ./app

EXPOSE 8095

HEALTHCHECK --interval=20s --timeout=5s --start-period=10s --retries=3 \
    CMD python -c "import urllib.request; urllib.request.urlopen('http://localhost:8095/health', timeout=3)"

CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8095", "--workers", "1"]
