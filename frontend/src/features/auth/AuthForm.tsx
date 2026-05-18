import { useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";

import { useAuthStore } from "@/app/store";
import { useTranslation } from "@/shared/i18n";
import { useToast } from "@/shared/ui";

type AuthMode = "signin" | "signup";

export function AuthForm() {
  const login = useAuthStore((state) => state.login);
  const register = useAuthStore((state) => state.register);
  const { pushToast } = useToast();
  const navigate = useNavigate();
  const location = useLocation();
  const t = useTranslation();

  const [mode, setMode] = useState<AuthMode>("signin");
  const [fullName, setFullName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const submit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    if (!email.includes("@")) {
      setError(t.validEmailRequired);
      return;
    }

    if (password.length < 6) {
      setError(t.passwordMinLength);
      return;
    }

    setError(null);
    setSubmitting(true);

    try {
      if (mode === "signup") {
        await register(email, password, fullName);
        pushToast(t.accountCreated);
      } else {
        await login(email, password);
        pushToast(t.signedIn);
      }
    } catch (e) {
      const err = e as {
        code?: string;
        message?: string;
        response?: { status?: number; data?: { error?: string; message?: string } };
      };
      const status = err.response?.status;
      const code = err.code ?? "";
      const isNetworkDown =
        !err.response ||
        code === "ERR_NETWORK" ||
        code === "ECONNREFUSED" ||
        code === "ECONNABORTED" ||
        code === "ETIMEDOUT" ||
        err.message === "Network Error";

      if (isNetworkDown) {
        setError(
          "Бэкенд недоступен. Запустите api-gateway (порт 8000) — например, `make dev-up` или docker compose.",
        );
      } else if (status === 404) {
        setError(
          "Эндпоинт авторизации не найден (404). Проверьте api-gateway и VITE_API_BASE_URL.",
        );
      } else if (status === 401 || status === 403) {
        setError(mode === "signin" ? "Неверный email или пароль." : "Регистрация отклонена сервером.");
      } else if (status === 409) {
        setError("Пользователь с таким email уже существует.");
      } else if (status && status >= 500) {
        setError("Сервер авторизации перезапускается. Подождите 5–10 секунд и попробуйте снова.");
      } else {
        const serverMessage = err.response?.data?.error ?? err.response?.data?.message;
        setError(serverMessage || "Не удалось выполнить запрос. Проверьте подключение и данные.");
      }
      setSubmitting(false);
      return;
    }

    setSubmitting(false);
    const nextPath = (location.state as { from?: string } | null)?.from ?? "/workspace";
    navigate(nextPath, { replace: true });
  };

  return (
    <section className="profile-card auth-form">
      <span className="eyebrow">{mode === "signin" ? "Вход в систему" : "Создание аккаунта"}</span>
      <h2 style={{ fontSize: 32, marginTop: 8 }}>
        {mode === "signin" ? t.welcomeBack : t.createAccount}
      </h2>
      <p className="muted" style={{ marginTop: 6, fontSize: 14, maxWidth: "44ch" }}>
        {t.continueFullyLocal}
      </p>

      <div className="segmented" role="tablist" aria-label="Режим" style={{ marginTop: 20 }}>
        <button
          className={mode === "signin" ? "is-active" : ""}
          onClick={() => { setMode("signin"); setError(null); }}
          type="button"
        >
          {t.signIn}
        </button>
        <button
          className={mode === "signup" ? "is-active" : ""}
          onClick={() => { setMode("signup"); setError(null); }}
          type="button"
        >
          {t.signUp}
        </button>
      </div>

      <form onSubmit={submit} style={{ display: "grid", gap: 14, marginTop: 18 }}>
        {mode === "signup" && (
          <div className="field">
            <label htmlFor="auth-fullname">{t.fullName}</label>
            <input
              id="auth-fullname"
              className="input"
              autoComplete="name"
              value={fullName}
              onChange={(event) => setFullName(event.target.value)}
              placeholder="Иван Иванов"
            />
          </div>
        )}
        <div className="field">
          <label htmlFor="auth-email">{t.email}</label>
          <input
            id="auth-email"
            className="input"
            type="email"
            autoComplete="email"
            value={email}
            onChange={(event) => setEmail(event.target.value)}
            placeholder="you@example.com"
          />
        </div>
        <div className="field">
          <label htmlFor="auth-password">{t.password}</label>
          <input
            id="auth-password"
            className="input"
            type="password"
            autoComplete={mode === "signin" ? "current-password" : "new-password"}
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            placeholder="••••••••"
          />
        </div>

        {error && (
          <p
            className="mono"
            style={{
              fontSize: 12,
              padding: "10px 12px",
              borderRadius: "var(--r-1)",
              background: "oklch(0.93 0.08 25)",
              color: "oklch(0.30 0.14 25)",
              border: "1px solid oklch(0.80 0.14 25)",
            }}
          >
            {error}
          </p>
        )}

        <button className="btn btn--primary" type="submit" disabled={submitting} style={{ marginTop: 4 }}>
          {submitting
            ? mode === "signin"
              ? "Входим…"
              : "Создаём аккаунт…"
            : mode === "signin"
              ? t.enterRealSync
              : t.createAccount}
        </button>

        <p className="mono" style={{ fontSize: 11, color: "var(--muted)", marginTop: 4, letterSpacing: "0.04em" }}>
          Нажимая кнопку, вы соглашаетесь с условиями использования RealSync.
        </p>
      </form>
    </section>
  );
}
