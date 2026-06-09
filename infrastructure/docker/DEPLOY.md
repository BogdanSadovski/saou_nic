# RealSync — деплой на AWS EC2 (v1.0)

Инструкция для **уже подготовленного** инстанса:
Docker, Docker Compose v2, git, npm — установлены.

Публичный URL: **http://realsync.ddns.net** (NoIP, A-record → IP инстанса).
Открытые порты на инстансе: **22** (SSH, только твой IP) и **80** (HTTP, `0.0.0.0/0`).

---

## 1. Клонирование репозитория

Репозиторий на GitHub называется `saou_nic`, но внутренние пути в коде и
docker-compose ожидают рабочую директорию **`real_ass`** (`context:
../../services/...` в compose отсчитывается от неё). Поэтому при клоне
**сразу переименовываем папку**:

```bash
cd ~
git clone https://github.com/BogdanSadovski/saou_nic.git real_ass
cd real_ass
```

(Это эквивалентно `git clone … saou_nic && mv saou_nic real_ass` — просто
коротко одной командой.)

Если уже клонировал с дефолтным именем:

```bash
mv ~/saou_nic ~/real_ass
cd ~/real_ass
```

---

## 2. Обновление GitHub OAuth callback (один раз)

Открой [github.com/settings/developers](https://github.com/settings/developers)
→ OAuth App **`Ov23li3jJpAiEif2d4WW`** →
поле **Authorization callback URL** должно содержать:

```
http://realsync.ddns.net/api/auth/oauth/github/callback
```

GitHub поддерживает несколько callback URL в одном приложении — можно
оставить локальный `http://localhost:3000/...` рядом для разработки.

---

## 3. Запуск

```bash
cd ~/real_ass/infrastructure/docker
./deploy.sh
```

Скрипт сам:
1. Скопирует `.env.production` → `.env` (если ещё нет).
2. Проверит, что не осталось плейсхолдеров `__CHANGE_ME_*`.
3. Соберёт все docker-образы (`docker compose build --pull`).
4. Поднимет стек (`docker compose up -d`).
5. Подождёт до 6 минут, пока все сервисы станут `healthy`.
6. Проверит, что фронт и api-gateway отвечают.

Первый запуск занимает **5–10 минут** (нужно скачать базовые образы,
собрать Go-сервисы из исходников, собрать React-бандл). Дальнейшие
старты — **~30 секунд** (используется кэш слоёв).

---

## 4. Проверка

После завершения скрипта:

| URL | Ожидаемый ответ |
|---|---|
| `http://realsync.ddns.net/` | Лендинг RealSync |
| `http://realsync.ddns.net/nginx-health` | `ok` |
| `http://realsync.ddns.net/api/health` | `{"status":"healthy"}` |

В браузере:
1. Регистрация по email / Логин.
2. «Войти через GitHub» — должен пройти OAuth-цикл и вернуть в приложение.
3. Создать интервью → DevTools → Network → WebSocket → подключение к
   `ws://realsync.ddns.net/api/interviews/sessions/<id>/ws` со статусом
   `101 Switching Protocols`.

---

## 5. Полезные команды

```bash
# Логи всего стека
docker compose logs -f

# Логи одного сервиса
docker compose logs -f api-gateway
docker compose logs -f ai-service
docker compose logs -f interview-service

# Список контейнеров со статусом health
docker compose ps

# Остановить (с сохранением данных)
docker compose down

# ВНИМАНИЕ: ниже стирает Postgres/Redis/MinIO!
docker compose down -v

# Пересобрать после правок кода
docker compose up -d --build

# Применить новый .env
docker compose --env-file .env up -d
```

---

## 6. Что НЕ надо делать

- ❌ Менять `localhost` в `frontend/src/shared/config/env.ts` — фронт сам
  определяет хост из `window.location.host`.
- ❌ Открывать порты 8000, 8001…8083 на EC2 — это внутренние порты
  сервисов, доступны только внутри docker-сети.
- ❌ Настраивать CORS — фронт и API на одном домене (same-origin).
- ❌ Запускать миграции БД руками — `init-scripts/` и сами сервисы
  применяют их при старте Postgres.

---

## 7. Если что-то не работает

| Симптом | Где смотреть |
|---|---|
| Фронт открывается, но `/api/*` отвечают 502 | `docker compose logs api-gateway` |
| Логин падает с 500 | `docker compose logs user-service` |
| Интервью не присылает первый вопрос | `docker compose logs ai-service interview-service` |
| WS обрывается через минуту | проверь `proxy_read_timeout` в `frontend/nginx.conf` (должно быть `24h`) |
| `realsync.ddns.net` не резолвится | NoIP может через 5–30 минут после смены IP, либо проверь A-record в [my.noip.com](https://my.noip.com) |
| `Cannot connect to the Docker daemon` | `sudo systemctl start docker` + `sudo usermod -aG docker $USER && newgrp docker` |
| `port is already allocated` 80 | `sudo lsof -i :80` — посмотри кто занял; на свежем EC2 обычно никто |

---

## 8. Когда придёт время HTTPS

Заготовка уже в репо: `infrastructure/docker/docker-compose.https.yml` +
`frontend/nginx.https.conf`.

```bash
# 1. AWS Security Group → открыть TCP 443
# 2. поставить certbot и выдать сертификат
sudo apt update && sudo apt install -y certbot
cd ~/real_ass/infrastructure/docker
docker compose stop frontend
sudo certbot certonly --standalone -d realsync.ddns.net \
  --non-interactive --agree-tos --email bogdan.minsk@icloud.com

# 3. подменить nginx.conf на HTTPS-вариант
cp ~/real_ass/frontend/nginx.https.conf ~/real_ass/frontend/nginx.conf

# 4. .env: переключить URL'ы на https://
sed -i 's|^PUBLIC_BASE_URL=.*|PUBLIC_BASE_URL=https://realsync.ddns.net|' .env
sed -i 's|^FRONTEND_BASE_URL=.*|FRONTEND_BASE_URL=https://realsync.ddns.net|' .env
sed -i 's|^OAUTH_GITHUB_REDIRECT_URL=.*|OAUTH_GITHUB_REDIRECT_URL=https://realsync.ddns.net/api/auth/oauth/github/callback|' .env

# 5. поднять с HTTPS-override
docker compose -f docker-compose.yml -f docker-compose.https.yml \
  --env-file .env up -d --build --force-recreate frontend

# 6. в GitHub OAuth-приложении обновить callback URL на https://
# 7. (опционально) cron на auto-renew
echo '0 3 * * * certbot renew --quiet --pre-hook "docker stop platform-frontend-1" --post-hook "docker start platform-frontend-1"' | sudo crontab -
```

Фронт сам подхватит — он реагирует на `window.location.protocol` и
переключит `ws://` → `wss://`.

---

## 9. SEO и красивые превью ссылок

В репо уже лежат:
- `frontend/index.html` — Open Graph + Twitter Card + JSON-LD
- `frontend/public/og-cover.png` — баннер 1200×1200
- `frontend/public/robots.txt`
- `frontend/public/sitemap.xml`
- `frontend/public/google023d82ba69f6f2e3.html` — Google Search Console verification

Что делает на свежем деплое: после `docker compose up -d --build frontend`
все эти файлы автоматически доступны:
- https://realsync.ddns.net/og-cover.png
- https://realsync.ddns.net/sitemap.xml
- https://realsync.ddns.net/robots.txt
- https://realsync.ddns.net/google023d82ba69f6f2e3.html

Что нужно сделать руками после деплоя:
1. Открыть [Google Search Console](https://search.google.com/search-console) →
   Add property → URL prefix → `https://realsync.ddns.net/` → Verify.
2. В разделе Sitemaps добавить `sitemap.xml`.
3. В Telegram: бот `@WebpageBot` → отправить URL → кеш Telegram сбросится.

---

## 10. Демо-данные

Один SQL-сидер `seed_demo_user.sql` (в корне репо) заполняет пользователя
`demo_user_test@realsnc.com` синтетикой: профиль, резюме, 3 интервью,
GitHub-кеш, биллинг. Запуск:

```bash
docker compose exec -T postgres psql -U postgres < ~/real_ass/seed_demo_user.sql
```

Идемпотентно — повторный запуск ничего не сломает.
