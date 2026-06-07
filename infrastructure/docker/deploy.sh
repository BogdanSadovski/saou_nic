#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────
# Скрипт первичного деплоя RealSync на AWS EC2 (или любой Linux-хост).
#
# Что делает:
#   1. Проверяет наличие Docker + Docker Compose v2.
#   2. Подсовывает .env.production как активный .env (если ещё нет).
#   3. Собирает все образы (frontend, api-gateway, все Go/Python сервисы).
#   4. Поднимает стек одной командой `docker compose up -d`.
#   5. Ждёт, пока все health-checks станут healthy.
#   6. Печатает финальную проверку доступности http://<domain>/.
#
# Запуск на сервере:
#     cd ~/real_ass/infrastructure/docker
#     cp .env.production .env       # ОДИН РАЗ, потом редактируем .env
#     vim .env                      # подставить секреты
#     ./deploy.sh
# ─────────────────────────────────────────────────────────────────────
set -euo pipefail

cd "$(dirname "$0")"

DOMAIN="${PUBLIC_DOMAIN:-realsync.ddns.net}"

bold() { printf "\033[1m%s\033[0m\n" "$*"; }
fail() { printf "\033[31m✗ %s\033[0m\n" "$*"; exit 1; }
ok()   { printf "\033[32m✓ %s\033[0m\n" "$*"; }

# 1) docker
command -v docker >/dev/null || fail "Docker не установлен. См. https://docs.docker.com/engine/install/"
docker compose version >/dev/null 2>&1 || fail "Нужен Docker Compose v2 (плагин docker-compose-plugin)."
ok "Docker и Compose v2 доступны"

# 2) .env — копируем .env.production, если ещё нет (он содержит готовые
#    секреты; редактировать руками теперь нужно ТОЛЬКО при необходимости).
if [ ! -f .env ]; then
  if [ ! -f .env.production ]; then
    fail ".env.production отсутствует — что-то не так с репозиторием"
  fi
  cp .env.production .env
  ok ".env создан из .env.production"
fi

# Жёсткая валидация: если кто-то всё-таки оставил __CHANGE_ME_* — стоп.
if grep -qE "__CHANGE_ME_" .env; then
  bold "В .env остались плейсхолдеры __CHANGE_ME_*:"
  grep -nE "__CHANGE_ME_" .env || true
  fail "Замените все плейсхолдеры на реальные значения."
fi

# Мягкое предупреждение: GitHub PAT — нужен только для аналитики профилей.
# Если стоят дефолтные плейсхолдеры (ghp_your-github-..., your-github-app-...),
# просто предупреждаем — основной флоу интервью/резюме работает без него.
if grep -qE "ghp_your-github-personal-access-token|your-github-app-client" .env; then
  printf "\033[33m⚠  GitHub PAT не настроен — аналитика репозиториев в Профиле работать не будет.\033[0m\n"
  printf "   Это НЕ блокирует деплой. Чтобы починить позже: получить токен в\n"
  printf "   https://github.com/settings/personal-access-tokens и заменить GITHUB_TOKEN в .env\n\n"
fi

ok ".env готов"

# 3) build (без cache на первый раз — гарантирует свежие слои)
bold "Сборка образов (может занять 5–10 минут на первый раз)…"
docker compose -f docker-compose.yml --env-file .env build --pull
ok "Образы собраны"

# 4) up
bold "Поднимаем стек…"
docker compose -f docker-compose.yml --env-file .env up -d
ok "Стек запущен"

# 5) ждём health
bold "Ждём, пока все сервисы станут healthy (тайм-аут 6 минут)…"
deadline=$(( $(date +%s) + 360 ))
while [ "$(date +%s)" -lt "$deadline" ]; do
  unhealthy=$(docker compose ps --format '{{.Name}}\t{{.Health}}' | awk '$2!="healthy" && $2!="" {print $1}' | wc -l | tr -d ' ')
  total=$(docker compose ps --format '{{.Name}}' | wc -l | tr -d ' ')
  if [ "$unhealthy" = "0" ] && [ "$total" -gt "0" ]; then
    ok "Все $total сервисов healthy"
    break
  fi
  printf "  ⋯ ждём… (не-healthy: %s/%s)\n" "$unhealthy" "$total"
  sleep 10
done

# 6) финальный пинг
bold "Финальная проверка фронта…"
if curl -fsS "http://localhost/nginx-health" >/dev/null; then
  ok "nginx внутри хоста отвечает 200 на /nginx-health"
else
  fail "nginx не отвечает на localhost. Проверьте \`docker compose logs frontend\`"
fi

if curl -fsS "http://localhost/api/health" >/dev/null; then
  ok "api-gateway отвечает 200 на /api/health"
else
  printf "  ⚠  /api/health не отвечает — проверьте \`docker compose logs api-gateway\`\n"
fi

cat <<EOF

──────────────────────────────────────────────────────────────────────
✓ Деплой выполнен.

Откройте в браузере:    http://${DOMAIN}/
Health фронта:          http://${DOMAIN}/nginx-health
Health api-gateway:     http://${DOMAIN}/api/health

Логи:
    docker compose logs -f                  # все
    docker compose logs -f api-gateway      # один сервис
    docker compose logs -f ai-service

Стоп/старт:
    docker compose down                     # с сохранением томов
    docker compose down -v                  # ПОЛНОЕ удаление БД (осторожно)
    docker compose up -d                    # снова поднять

Перезапустить после правки .env:
    docker compose --env-file .env up -d --build
──────────────────────────────────────────────────────────────────────
EOF
