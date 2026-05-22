import { useCallback, useEffect, useState } from "react";

import { apiClient } from "@/shared/api/client";
import { useToast } from "@/shared/ui";

type TelegramStatus = {
  linked: boolean;
  chat_id?: number | null;
  tg_username?: string | null;
  notifications_paused: boolean;
  daily_push_hour_utc: number;
  linked_at?: string | null;
};

type LinkTokenResponse = {
  token: string;
  expires_at: string;
  bot_username: string;
  deep_link: string;
};

/**
 * Карточка «Подключить Telegram» в Profile.
 *
 * Flow:
 *   1. Юзер жмёт «Получить ссылку» → POST /integrations/telegram/link-token
 *   2. Backend выдаёт {deep_link, expires_at} (token живёт 30 мин)
 *   3. Юзер открывает t.me/<bot>?start=<token> → бот биндит chat_id
 *   4. После биндинга юзер возвращается, жмёт «Обновить статус» —
 *      карточка тянет /integrations/telegram/status и видит linked=true
 */
export function TelegramConnectCard() {
  const { pushToast } = useToast();
  const [status, setStatus] = useState<TelegramStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [link, setLink] = useState<LinkTokenResponse | null>(null);
  const [issuing, setIssuing] = useState(false);
  const [unlinking, setUnlinking] = useState(false);

  const loadStatus = useCallback(async () => {
    setLoading(true);
    try {
      const { data } = await apiClient.get<TelegramStatus>(
        "/integrations/telegram/status",
      );
      setStatus(data);
    } catch {
      setStatus(null);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadStatus();
  }, [loadStatus]);

  const issueLink = async () => {
    setIssuing(true);
    try {
      const { data } = await apiClient.post<LinkTokenResponse>(
        "/integrations/telegram/link-token",
        {},
      );
      setLink(data);
    } catch (e) {
      console.error(e);
      pushToast("Не удалось получить ссылку для привязки Telegram");
    } finally {
      setIssuing(false);
    }
  };

  const copyLink = async () => {
    if (!link?.deep_link) return;
    try {
      await navigator.clipboard.writeText(link.deep_link);
      pushToast("Ссылка скопирована");
    } catch {
      pushToast("Не получилось скопировать. Кликните по ссылке вручную.");
    }
  };

  const unlink = async () => {
    setUnlinking(true);
    try {
      await apiClient.delete("/integrations/telegram");
      pushToast("Telegram отвязан");
      setLink(null);
      await loadStatus();
    } catch {
      pushToast("Не удалось отвязать Telegram");
    } finally {
      setUnlinking(false);
    }
  };

  const qrSrc = link
    ? `https://api.qrserver.com/v1/create-qr-code/?size=180x180&margin=0&data=${encodeURIComponent(link.deep_link)}`
    : null;

  return (
    <section className="profile-card">
      <header className="row-between" style={{ alignItems: "baseline" }}>
        <div>
          <span className="eyebrow">Telegram</span>
          <h2 style={{ fontSize: 28, marginTop: 4 }}>Ежедневная практика</h2>
        </div>
        {status?.linked ? (
          <span className="tag tag--lime">CONNECTED</span>
        ) : (
          <span className="tag">OFF</span>
        )}
      </header>

      {loading ? (
        <p className="muted" style={{ fontSize: 13 }}>Загрузка…</p>
      ) : status?.linked ? (
        <div style={{ display: "grid", gap: 10 }}>
          <p className="muted" style={{ fontSize: 13, lineHeight: 1.55 }}>
            Telegram подключён{status.tg_username ? ` к @${status.tg_username}` : ""}.
            Бот присылает один вопрос каждый день в{" "}
            <strong>{String(status.daily_push_hour_utc).padStart(2, "0")}:00 UTC</strong>{" "}
            {status.notifications_paused ? "· сейчас на паузе" : "· активно"}.
          </p>
          <div className="row" style={{ gap: 10, marginTop: 8 }}>
            <button
              className="btn btn--ghost btn--sm"
              type="button"
              onClick={unlink}
              disabled={unlinking}
            >
              {unlinking ? "Отвязываем…" : "Отвязать Telegram"}
            </button>
            <button className="btn btn--ghost btn--sm" type="button" onClick={() => void loadStatus()}>
              Обновить статус
            </button>
          </div>
        </div>
      ) : (
        <div style={{ display: "grid", gap: 14 }}>
          <p className="muted" style={{ fontSize: 13, lineHeight: 1.55 }}>
            Получите 1 вопрос для практики каждое утро прямо в Telegram. Ответьте
            текстом — бот оценит ответ soft-skills моделью и пришлёт фидбэк.
          </p>

          {link ? (
            <div
              style={{
                padding: 18,
                border: "1px solid var(--line)",
                borderRadius: "var(--r-1)",
                background: "var(--paper-2)",
                display: "grid",
                gridTemplateColumns: "auto 1fr",
                gap: 18,
                alignItems: "center",
              }}
            >
              {qrSrc ? (
                <img
                  src={qrSrc}
                  alt="QR-код привязки"
                  width={140}
                  height={140}
                  style={{ borderRadius: 8, background: "white", padding: 6 }}
                />
              ) : null}
              <div style={{ display: "grid", gap: 8, minWidth: 0 }}>
                <strong style={{ fontSize: 14 }}>1. Отсканируйте QR или откройте ссылку</strong>
                <a
                  href={link.deep_link}
                  target="_blank"
                  rel="noreferrer"
                  className="mono"
                  style={{ fontSize: 12, color: "var(--ink)", wordBreak: "break-all" }}
                >
                  {link.deep_link}
                </a>
                <p className="muted" style={{ fontSize: 12, margin: 0 }}>
                  2. Нажмите «START» в боте → аккаунт привяжется автоматически.
                  Затем вернитесь и нажмите «Обновить статус».
                </p>
                <div className="row" style={{ gap: 8, marginTop: 4, flexWrap: "wrap" }}>
                  <button className="btn btn--ghost btn--sm" type="button" onClick={copyLink}>
                    Скопировать ссылку
                  </button>
                  <button
                    className="btn btn--primary btn--sm"
                    type="button"
                    onClick={() => void loadStatus()}
                  >
                    Обновить статус
                  </button>
                </div>
                <p className="mono muted" style={{ fontSize: 10, marginTop: 4 }}>
                  Действует до {new Date(link.expires_at).toLocaleTimeString("ru-RU", { hour: "2-digit", minute: "2-digit" })}
                </p>
              </div>
            </div>
          ) : (
            <button
              className="btn btn--primary btn--sm"
              type="button"
              onClick={issueLink}
              disabled={issuing}
              style={{ justifySelf: "start" }}
            >
              {issuing ? "Готовим ссылку…" : "Подключить Telegram"}
            </button>
          )}
        </div>
      )}
    </section>
  );
}
