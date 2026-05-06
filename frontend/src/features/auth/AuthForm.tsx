import { useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";

import { useAuthStore } from "@/app/store";
import { useTranslation } from "@/shared/i18n";
import { FloatingInput, GlassButton, GlassCard, useToast } from "@/shared/ui";

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

    try {
      if (mode === "signup") {
        await register(email, password, fullName);
        pushToast(t.accountCreated);
      } else {
        await login(email, password);
        pushToast(t.signedIn);
      }
    } catch {
      setError("Ошибка авторизации. Проверьте доступность бэкенда и корректность данных.");
      return;
    }

    const nextPath = (location.state as { from?: string } | null)?.from ?? "/dashboard";
    navigate(nextPath, { replace: true });
  };

  return (
    <GlassCard className="auth-form">
      <h2>{mode === "signin" ? t.welcomeBack : t.createAccount}</h2>
      <p className="muted">{t.continueFullyLocal}</p>

      <div className="auth-mode-switch">
        <button
          className={mode === "signin" ? "mode-active" : ""}
          onClick={() => setMode("signin")}
          type="button"
        >
          {t.signIn}
        </button>
        <button
          className={mode === "signup" ? "mode-active" : ""}
          onClick={() => setMode("signup")}
          type="button"
        >
          {t.signUp}
        </button>
      </div>

      <form onSubmit={submit}>
        {mode === "signup" && (
          <FloatingInput
            autoComplete="name"
            label={t.fullName}
            onChange={(event) => setFullName(event.target.value)}
            value={fullName}
          />
        )}
        <FloatingInput
          autoComplete="email"
          label={t.email}
          onChange={(event) => setEmail(event.target.value)}
          value={email}
        />
        <FloatingInput
          autoComplete="current-password"
          label={t.password}
          onChange={(event) => setPassword(event.target.value)}
          type="password"
          value={password}
        />

        {error && <p className="form-error">{error}</p>}

        <GlassButton type="submit">
          {mode === "signin" ? t.enterRealSync : t.createAccount}
        </GlassButton>
      </form>
    </GlassCard>
  );
}
