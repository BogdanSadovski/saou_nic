# Подключение LLM-провайдеров

Платформа работает через каскад провайдеров. Запрос пытается выполниться на
Tier 1; если 429/5xx — автоматически уходит на Tier 2, потом Tier 3, потом
Tier 4. Внутри Tier 3 и Tier 4 можно положить **пул из нескольких ключей**
одного провайдера — пул раздаёт запросы по round-robin между ключами,
утраивая (или больше) суточный лимит.

Минимальная конфигурация — только Tier 1, всё работает как раньше.
Дополнительные уровни включаются добавлением переменных в `.env`.

---

## Текущая рекомендуемая конфигурация

| Уровень | Провайдер | Модель | Зачем |
|---------|-----------|--------|-------|
| Tier 1 | **Groq** | `llama-3.3-70b-versatile` | Очень быстро (2-3 с/запрос), хорошее качество. ~14k TPM / 100k TPD free. |
| Tier 2 | **OpenRouter** (ключ A) | `meta-llama/llama-3.3-70b-instruct:free` | Подстраховка от Groq. Другой провайдер, другие лимиты. |
| Tier 3 | **OpenRouter × 5 ключей** | `deepseek/deepseek-chat-v3-0324:free` | Пул из 5 ключей → 5× суточный лимит, DeepSeek качества gpt-4o-mini. |
| Tier 4 | Cerebras / Together / др. | — | Опционально, last-resort. |

Tier 3 — **главная новая фишка**. 5 ключей OpenRouter на одну DeepSeek-модель,
round-robin размазывает нагрузку. Если один ключ упёрся в дневной лимит —
пул сам пропускает мёртвый слот и пробует следующий.

---

## 1️⃣ Tier 1 — Groq (primary)

1. Зарегистрируйтесь: <https://console.groq.com/>
2. Создайте ключ: <https://console.groq.com/keys>
3. В `.env`:
   ```bash
   LLM_API_KEY=gsk_...
   LLM_BASE_URL=https://api.groq.com/openai/v1
   LLM_MODEL=llama-3.3-70b-versatile
   ```

Лимиты free-тарифа: ~14k TPM, 100k TPD. Подробнее: <https://console.groq.com/settings/limits>

---

## 2️⃣ Tier 2 — OpenRouter (secondary, 1 ключ)

1. Зарегистрируйтесь: <https://openrouter.ai/>
2. Создайте API-ключ: <https://openrouter.ai/keys>
3. В `.env`:
   ```bash
   LLM_SECONDARY_API_KEY=sk-or-v1-...
   LLM_SECONDARY_BASE_URL=https://openrouter.ai/api/v1
   LLM_SECONDARY_MODEL=meta-llama/llama-3.3-70b-instruct:free
   ```

Каталог free-моделей: <https://openrouter.ai/models?q=free>
Лимиты free: ~50 запросов/день на ключ.

---

## 3️⃣ Tier 3 — DeepSeek через OpenRouter (пул из 5 ключей)

**Главная идея:** один аккаунт OpenRouter позволяет создать несколько API-ключей.
Создаём 5 ключей, кладём в пул — получаем 5× суточный лимит. Это разрешено
ToS OpenRouter (в отличие от мульти-аккаунтинга DeepSeek напрямую).

### Получение 5 ключей

1. Залогиньтесь в OpenRouter (тот же аккаунт, что для Tier 2): <https://openrouter.ai/keys>
2. Нажмите **Create Key**. Дайте имя, например `realsync-deepseek-1`. Скопируйте ключ.
3. Повторите 5 раз — `realsync-deepseek-1`, `…-2`, `…-3`, `…-4`, `…-5`.
4. Все 5 ключей будут вида `sk-or-v1-XXXXXX...`

### Конфигурация в `.env`

Запятыми, без пробелов:

```bash
LLM_TERTIARY_API_KEYS=sk-or-v1-key1,sk-or-v1-key2,sk-or-v1-key3,sk-or-v1-key4,sk-or-v1-key5
```

`LLM_TERTIARY_BASE_URL` и `LLM_TERTIARY_MODEL` можно не задавать — подставятся
дефолты `https://openrouter.ai/api/v1` и `deepseek/deepseek-chat-v3-0324:free`.

Если нужна reasoning-модель — задайте явно:
```bash
LLM_TERTIARY_MODEL=deepseek/deepseek-r1:free
```

Каталог free DeepSeek-моделей в OpenRouter: <https://openrouter.ai/models?q=deepseek+free>

---

## 4️⃣ Tier 4 — last-resort (опционально)

Выберите один из вариантов или пропустите.

### Cerebras (быстрый, рекомендую)

1. Зарегистрируйтесь: <https://cloud.cerebras.ai/>
2. Ключ: <https://cloud.cerebras.ai/platform> → API Keys
3. Docs: <https://inference-docs.cerebras.ai/>

```bash
LLM_QUATERNARY_API_KEY=csk-...
LLM_QUATERNARY_BASE_URL=https://api.cerebras.ai/v1
LLM_QUATERNARY_MODEL=llama-3.3-70b
```

Free: 30 RPM, 60k TPM, 14400 RPD.

### Together AI

1. Регистрация: <https://www.together.ai/>
2. Ключ: <https://api.together.ai/settings/api-keys>
3. Docs: <https://docs.together.ai/reference/chat-completions-1>

```bash
LLM_QUATERNARY_API_KEY=...
LLM_QUATERNARY_BASE_URL=https://api.together.xyz/v1
LLM_QUATERNARY_MODEL=meta-llama/Llama-3.3-70B-Instruct-Turbo-Free
```

### Google Gemini

1. AI Studio: <https://aistudio.google.com/>
2. Ключ: <https://aistudio.google.com/apikey>

```bash
LLM_QUATERNARY_API_KEY=AIza...
LLM_QUATERNARY_BASE_URL=https://generativelanguage.googleapis.com/v1beta/openai
LLM_QUATERNARY_MODEL=gemini-2.0-flash
```

Free: 15 RPM, 1500 RPD.

### Пул нескольких ключей в Tier 4

Если хотите второй пул — используйте `LLM_QUATERNARY_API_KEYS` (plural):
```bash
LLM_QUATERNARY_API_KEYS=key1,key2,key3
```

---

## 📋 Итоговый `.env` (полная конфигурация)

```bash
# ─── Tier 1: Groq (primary) ───
LLM_API_KEY=gsk_xxxxxxxxxxxx
LLM_BASE_URL=https://api.groq.com/openai/v1
LLM_MODEL=llama-3.3-70b-versatile

# ─── Tier 2: OpenRouter (secondary, 1 ключ) ───
LLM_SECONDARY_API_KEY=sk-or-v1-aaaaaaaaaa
LLM_SECONDARY_BASE_URL=https://openrouter.ai/api/v1
LLM_SECONDARY_MODEL=meta-llama/llama-3.3-70b-instruct:free

# ─── Tier 3: DeepSeek via OpenRouter (пул 5 ключей) ───
LLM_TERTIARY_API_KEYS=sk-or-v1-bbbb,sk-or-v1-cccc,sk-or-v1-dddd,sk-or-v1-eeee,sk-or-v1-ffff
# base_url и model — дефолты подставятся, можно не указывать
# LLM_TERTIARY_BASE_URL=https://openrouter.ai/api/v1
# LLM_TERTIARY_MODEL=deepseek/deepseek-chat-v3-0324:free

# ─── Tier 4 (опционально): Cerebras ───
LLM_QUATERNARY_API_KEY=csk-xxxxxxxxxxxx
LLM_QUATERNARY_BASE_URL=https://api.cerebras.ai/v1
LLM_QUATERNARY_MODEL=llama-3.3-70b
```

---

## 🚀 Запуск

После правки `.env`:

```bash
cd /Users/bogdan./Documents/учеба/дипломчик/real_ass
make dev-down
make dev-up
```

Важно: **именно `down` + `up`**, не `restart`. Env-переменные перечитываются
только при пересоздании контейнера.

### Проверка, что каскад поднялся

```bash
docker logs ai-service 2>&1 | grep "LLM cascade"
```

Должно появиться примерно:

```
LLM cascade: 4 tiers configured — llama-3.3-70b-versatile → meta-llama/llama-3.3-70b-instruct:free → deepseek/deepseek-chat-v3-0324:free×5 → llama-3.3-70b
```

`×5` после третьего тира — подтверждение, что пул из 5 ключей собрался.

### Прямой тест endpoint

```bash
curl -X POST http://localhost:8001/api/v1/interviewer/next-question \
  -H "Content-Type: application/json" \
  -d '{"session_id":"test","role":"Backend","level":"Middle","interview_mode":"theory","topic":"databases","messages":[]}'
```

В логах ai-service увидите, какой тир ответил. Если Tier 1 (Groq) живой —
ответит он; если рейт-лимит — увидите warning `LLM Tier 1 failed, falling
over to Tier 2`.

---

## 🔧 Поведение

- **Tier 3 пул:** запросы внутри пула раздаются round-robin между 5 ключами.
  Если ключ #2 упёрся в лимит — пул логирует warning и пробует #3, #4, #5.
  Только когда **все 5 ключей** в пуле упали — переход на Tier 4.

- **Каскад между тирами:** Tier 1 (Groq) → Tier 2 (OpenRouter одиночный) →
  Tier 3 (пул 5×OpenRouter+DeepSeek) → Tier 4. Каждый раз начинается
  заново с Tier 1, чтобы при восстановлении основного провайдера автоматически
  возвращаться к лучшему качеству.

- **Без falsy-данных:** если ВСЕ тиры упали — пробрасывается реальная ошибка
  до бизнес-логики. Никаких выдуманных ответов «AI типа сработал».

- **Resume / GitHub эндпоинты:** имеют свой эвристический fallback в коде
  (`_fallback_resume_positions`, `_fallback_role_recommendations`). Если
  каскад полностью упал, они вернут разумный детерминистический ответ.
