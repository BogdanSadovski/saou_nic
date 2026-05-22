import { useEffect } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";

import { useUserStore } from "@/app/store";
import { AuthForm } from "@/features/auth/AuthForm";
import { GithubConnectCard } from "@/features/github-connect/GithubConnectCard";
import { useToast } from "@/shared/ui";

export default function AuthPage() {
  const navigate = useNavigate();
  const [params, setParams] = useSearchParams();
  const { pushToast } = useToast();
  const hydrateUser = useUserStore((s) => s.hydrate);

  // OAuth-callback: бэкенд после Google/GitHub редиректит сюда с
  // access_token и refresh_token в query. Кладём в localStorage,
  // зачищаем URL и тянем профиль → переходим в workspace.
  useEffect(() => {
    const accessToken = params.get("access_token");
    const refreshToken = params.get("refresh_token");
    const oauthError = params.get("oauth_error");
    const provider = params.get("oauth_provider");

    if (oauthError) {
      pushToast(
        oauthError === "missing_code"
          ? "OAuth-провайдер не вернул код авторизации."
          : "Не удалось завершить вход через провайдера. Попробуйте ещё раз.",
      );
      params.delete("oauth_error");
      params.delete("oauth_provider");
      setParams(params, { replace: true });
      return;
    }

    if (accessToken && refreshToken) {
      localStorage.setItem("realsync_token", accessToken);
      localStorage.setItem("realsync_refresh_token", refreshToken);
      params.delete("access_token");
      params.delete("refresh_token");
      params.delete("oauth_provider");
      setParams(params, { replace: true });
      pushToast(`Вход через ${provider === "google" ? "Google" : "GitHub"} выполнен.`);
      void hydrateUser().then(() => navigate("/workspace"));
    }
  }, [params, setParams, navigate, pushToast, hydrateUser]);

  return (
    <main className="page" data-screen-label="Auth">
      <div className="sysbar reveal" style={{ marginBottom: 24 }}>
        <span><span className="dot"></span><span className="k">аутентификация</span><span className="v">v2</span></span>
        <span><span className="k">провайдер</span><span className="v">jwt + bcrypt + oauth</span></span>
        <span><span className="k">сессии</span><span className="v">7d refresh</span></span>
        <span><span className="k">github</span><span className="v">опционально</span></span>
      </div>

      <span className="eyebrow">Доступ к платформе</span>
      <h1 className="expr-headline" style={{ fontSize: "clamp(44px, 5.5vw, 80px)", marginTop: 8 }}>
        <span className="bold">Войти</span> <span className="ital">или</span> <span className="bold">создать</span> аккаунт.
      </h1>

      <div
        style={{
          display: "grid",
          gridTemplateColumns: "minmax(0, 1fr) minmax(0, 1fr)",
          gap: 24,
          marginTop: 32,
          alignItems: "start",
        }}
        className="auth-grid"
      >
        <AuthForm />
        <GithubConnectCard />
      </div>
    </main>
  );
}
