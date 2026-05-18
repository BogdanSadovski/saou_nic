# Гайд по тестированию платежной системы

## Быстрый старт

### Предусловия
- Backend user-service запущен (`go run ./cmd/main.go`)
- Frontend работает (`npm run dev`)
- PostgreSQL с миграциями выполнена (миграция `003_create_billing_tables`)
- Авторизованный пользователь

### Тестирование через Frontend

#### Шаг 1: Открыть страницу оплаты
```
http://localhost:5173/billing
```

#### Шаг 2: Выбрать план
- Trial (бесплатно, 14 дней, автоактивация)
- Pro ($9.99/мес, 30 интервью/мес)
- Platinum ($29.99/мес, неограниченные интервью)

#### Шаг 3: Оплатить тестово
1. Нажать "Upgrade to Pro/Platinum"
2. В модальном окне "Тестовая оплата" увидеть:
   - Intent ID
   - Client Secret
   - Срок истечения (30 мин)
3. Нажать "Подтвердить тестовую оплату"
4. Проверить результат:
   - ✅ Карточка "Текущая подписка" обновилась
   - ✅ Подписка отображает выбранный план и дату окончания

#### Шаг 4: Проверить историю платежей
- Внизу страницы "История транзакций"
- Должна содержать только что созданную запись

---

## API endpoints для ручного тестирования

Все endpoints требуют заголовка:
```
Authorization: Bearer {JWT_token}
```

### 1. GET /api/v1/billing/plans
Получить список всех доступных тарифов.

**Запрос:**
```bash
curl -H "Authorization: Bearer $JWT" \
  http://localhost:8000/api/v1/billing/plans
```

**Ответ:**
```json
{
  "plans": [
    {
      "id": "plan_trial",
      "tier": "trial",
      "name": "Trial",
      "description": "14 дней бесплатно",
      "price": 0,
      "billing_cycle": "one-time",
      "features": [
        "5 интервью в месяц",
        "Базовая аналитика"
      ],
      "limits": {
        "interviews_per_month": 5,
        "resume_storage_gb": 1
      }
    },
    {
      "id": "plan_pro",
      "tier": "pro",
      "name": "Pro",
      "description": "Профессиональный тарифчик",
      "price": 9.99,
      "billing_cycle": "monthly",
      "features": [
        "30 интервью в месяц",
        "Расширенная аналитика",
        "Приоритетная поддержка"
      ],
      "limits": {
        "interviews_per_month": 30,
        "resume_storage_gb": 5
      }
    },
    ...
  ],
  "test_mode": true
}
```

### 2. GET /api/v1/billing/subscription
Получить текущую подписку пользователя.

**Запрос:**
```bash
curl -H "Authorization: Bearer $JWT" \
  http://localhost:8000/api/v1/billing/subscription
```

**Ответ (если есть активная подписка):**
```json
{
  "subscription": {
    "id": "sub_abc123xyz",
    "user_id": "user_456",
    "tier": "pro",
    "status": "active",
    "billing_cycle": "monthly",
    "start_date": "2026-05-04T10:00:00Z",
    "end_date": "2026-06-04T10:00:00Z",
    "renewal_date": "2026-06-04T10:00:00Z",
    "is_active": true,
    "plan": {
      "id": "plan_pro",
      "tier": "pro",
      "name": "Pro",
      "price": 9.99,
      "billing_cycle": "monthly"
    }
  }
}
```

**Ответ (если нет подписки):**
```json
{
  "subscription": null,
  "message": "no active subscription"
}
```

### 3. POST /api/v1/billing/checkout-intents
Создать checkout intent (payment intent) для оплаты.

**Запрос:**
```bash
curl -X POST \
  -H "Authorization: Bearer $JWT" \
  -H "Content-Type: application/json" \
  -d '{
    "tier": "pro",
    "billing_cycle": "monthly",
    "payment_method_id": "pm_test_visa_4242"
  }' \
  http://localhost:8000/api/v1/billing/checkout-intents
```

**Параметры:**
- `tier` (required): `"trial"`, `"pro"`, или `"platinum"`
- `billing_cycle` (optional): `"monthly"` или `"yearly"` (default: `"monthly"`)
- `payment_method_id` (optional): test payment method ID

**Ответ:**
```json
{
  "intent": {
    "id": "pi_test_1234567890",
    "user_id": "user_456",
    "tier": "pro",
    "billing_cycle": "monthly",
    "amount_cents": 999,
    "currency": "USD",
    "status": "requires_confirmation",
    "provider": "test-gateway",
    "client_secret": "test_secret_abcd1234xyz",
    "expires_at": "2026-05-04T15:00:00Z",
    "created_at": "2026-05-04T14:30:00Z"
  },
  "test_mode": true,
  "next_step": "POST /api/v1/billing/checkout-intents/{intentID} to confirm"
}
```

**Сроки истечения:**
- Trial: тут же активируется (нет payment intent)
- Pro/Platinum: 30 минут на подтверждение

### 4. POST /api/v1/billing/checkout-intents/{intentID}
Подтвердить checkout intent и активировать подписку.

**Запрос:**
```bash
curl -X POST \
  -H "Authorization: Bearer $JWT" \
  -H "Content-Type: application/json" \
  -d '{
    "payment_method_id": "pm_test_visa_4242"
  }' \
  http://localhost:8000/api/v1/billing/checkout-intents/pi_test_1234567890
```

**Параметры:**
- `payment_method_id` (optional): подтверждающий payment method

**Ответ:**
```json
{
  "intent": {
    "id": "pi_test_1234567890",
    "status": "succeeded",
    "confirmed_at": "2026-05-04T14:31:00Z"
  },
  "subscription": {
    "id": "sub_new_xyz",
    "tier": "pro",
    "status": "active",
    "start_date": "2026-05-04T14:31:00Z",
    "end_date": "2026-06-04T14:31:00Z"
  },
  "transaction": {
    "id": "txn_verified_123",
    "status": "succeeded",
    "amount_cents": 999,
    "currency": "USD"
  },
  "test_mode": true
}
```

### 5. GET /api/v1/billing/transactions
Получить историю всех платежей пользователя.

**Запрос:**
```bash
curl -H "Authorization: Bearer $JWT" \
  "http://localhost:8000/api/v1/billing/transactions?limit=20"
```

**Параметры query:**
- `limit` (optional): количество записей (default: 20, max: 100)
- `offset` (optional): пропустить N записей (для pagination)

**Ответ:**
```json
{
  "transactions": [
    {
      "id": "txn_verified_123",
      "intent_id": "pi_test_1234567890",
      "amount_cents": 999,
      "currency": "USD",
      "status": "succeeded",
      "provider": "test-gateway",
      "external_reference": "txn_test_ref_1",
      "description": "Pro subscription payment (monthly)",
      "created_at": "2026-05-04T14:31:00Z"
    }
  ],
  "total": 1,
  "limit": 20,
  "offset": 0
}
```

### 6. POST /api/v1/billing/webhooks/test
**[ТОЛЬКО ДЛЯ ТЕСТИРОВАНИЯ]** Симулировать webhook от платежного провайдера.

**Запрос:**
```bash
curl -X POST \
  -H "Authorization: Bearer $JWT" \
  -H "Content-Type: application/json" \
  -d '{
    "intent_id": "pi_test_1234567890",
    "status": "failed"
  }' \
  http://localhost:8000/api/v1/billing/webhooks/test
```

**Параметры:**
- `intent_id` (required): ID checkout intent
- `status` (required): `"succeeded"`, `"failed"`, или `"canceled"`

**Ответ при failed:**
```json
{
  "intent": {
    "id": "pi_test_1234567890",
    "status": "failed"
  },
  "message": "subscription not activated due to payment failure",
  "test_mode": true
}
```

### 7. POST /api/v1/billing/subscription/cancel
Отменить текущую активную подписку.

**Запрос:**
```bash
curl -X POST \
  -H "Authorization: Bearer $JWT" \
  http://localhost:8000/api/v1/billing/subscription/cancel
```

**Ответ:**
```json
{
  "subscription": {
    "id": "sub_abc123xyz",
    "tier": "pro",
    "status": "canceled",
    "canceled_at": "2026-05-04T14:45:00Z",
    "end_date": "2026-06-04T10:00:00Z"
  },
  "message": "subscription canceled successfully"
}
```

---

## Сценарии тестирования

### Сценарий 1: Базовый flow (успешный платёж)

```bash
# Получить JWT (через frontend login или другой механизм)
export JWT="your_jwt_token"

# 1. Посмотреть доступные тарифы
curl -H "Authorization: Bearer $JWT" \
  http://localhost:8000/api/v1/billing/plans

# 2. Создать checkout intent для Pro
curl -X POST \
  -H "Authorization: Bearer $JWT" \
  -H "Content-Type: application/json" \
  -d '{"tier": "pro", "billing_cycle": "monthly"}' \
  http://localhost:8000/api/v1/billing/checkout-intents

# Сохранить intent ID из ответа
export INTENT_ID="pi_test_..."

# 3. Подтвердить платёж
curl -X POST \
  -H "Authorization: Bearer $JWT" \
  -H "Content-Type: application/json" \
  -d '{}' \
  http://localhost:8000/api/v1/billing/checkout-intents/$INTENT_ID

# 4. Проверить активную подписку
curl -H "Authorization: Bearer $JWT" \
  http://localhost:8000/api/v1/billing/subscription

# Ожидаемый результат:
# - status: "active"
# - tier: "pro"
# - start_date: today
# - end_date: today + 1 month
```

### Сценарий 2: Webhook симуляция (отказ платежа)

```bash
# Создать intent
curl -X POST \
  -H "Authorization: Bearer $JWT" \
  -H "Content-Type: application/json" \
  -d '{"tier": "platinum"}' \
  http://localhost:8000/api/v1/billing/checkout-intents

export INTENT_ID="pi_test_..."

# Симулировать failed webhook
curl -X POST \
  -H "Authorization: Bearer $JWT" \
  -H "Content-Type: application/json" \
  -d '{"intent_id": "'$INTENT_ID'", "status": "failed"}' \
  http://localhost:8000/api/v1/billing/webhooks/test

# Проверить статус (должен быть failed, подписка не активирована)
curl -H "Authorization: Bearer $JWT" \
  http://localhost:8000/api/v1/billing/subscription
```

### Сценарий 3: Апгрейд подписки

```bash
# 1. Активировать Pro
curl -X POST \
  -H "Authorization: Bearer $JWT" \
  -H "Content-Type: application/json" \
  -d '{"tier": "pro"}' \
  http://localhost:8000/api/v1/billing/checkout-intents
# Подтвердить (see scenario 1)

# 2. Апгрейдить на Platinum
curl -X POST \
  -H "Authorization: Bearer $JWT" \
  -H "Content-Type: application/json" \
  -d '{"tier": "platinum"}' \
  http://localhost:8000/api/v1/billing/checkout-intents

# 3. Новый intent создан, старая подписка отменена после подтверждения

# 4. Проверить историю транзакций (обе должны быть)
curl -H "Authorization: Bearer $JWT" \
  "http://localhost:8000/api/v1/billing/transactions?limit=50"
```

### Сценарий 4: Отмена подписки

```bash
# 1. Активировать любой платный план
# (см. сценарий 1)

# 2. Отменить
curl -X POST \
  -H "Authorization: Bearer $JWT" \
  http://localhost:8000/api/v1/billing/subscription/cancel

# 3. Проверить статус (должен быть canceled)
curl -H "Authorization: Bearer $JWT" \
  http://localhost:8000/api/v1/billing/subscription

# Ожидаемый результат:
# - status: "canceled"
# - canceled_at: now
```

---

## Отладка

### Проблема: 401 Unauthorized

**Причина:** JWT токен отсутствует или истёк  
**Решение:** Авторизуйтесь в frontend, скопируйте token из localStorage:
```javascript
// В console браузера
console.log(localStorage.getItem('auth_token'))
```

### Проблема: 404 Not Found на /api/v1/billing

**Причина:** Backend не запущен или путь неправильный  
**Решение:**
```bash
# Проверить, что user-service работает
curl http://localhost:8000/health

# Проверить роуты
curl -H "Authorization: Bearer $JWT" \
  http://localhost:8000/api/v1/billing/plans
```

### Проблема: Intent expires too quickly

**Причина:** Intent истёк (более 30 минут прошло)  
**Решение:** Создать новый intent (старый больше не подойдёт)

### Проблема: Database constraint errors

**Причина:** Миграция не выполнена  
**Решение:**
```bash
# Выполнить миграции
cd services/user-service
migrate -path migrations \
  -database "postgres://user:pass@localhost:5432/real_ass?sslmode=disable" \
  up
```

---

## Интеграция с реальным платёжным провайдером (Phase 2)

После тестирования в test-gateway режиме:

1. Установить Stripe SDK:
   ```bash
   go get github.com/stripe/stripe-go/v76
   ```

2. Заменить в `payment_service.go`:
   - `provider := "test-gateway"` → `provider := "stripe"`
   - `status := "requires_confirmation"` → Stripe API call
   - Webhook listener вместо `/webhooks/test`

3. Обновить env vars:
   ```
   STRIPE_SECRET_KEY=sk_test_...
   STRIPE_PUBLIC_KEY=pk_test_...
   ```

4. Frontend: заменить симуляцию на Stripe Elements/Payment Request Button

---

**Статус:** ✅ Ready for testing  
**Последнее обновление:** 2026-05-04
