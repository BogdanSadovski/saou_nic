import { AuthForm } from "@/features/auth/AuthForm";
import { GithubConnectCard } from "@/features/github-connect/GithubConnectCard";

export default function AuthPage() {
  return (
    <main className="page" data-screen-label="Auth">
      <div className="sysbar reveal" style={{ marginBottom: 24 }}>
        <span><span className="dot"></span><span className="k">аутентификация</span><span className="v">v2</span></span>
        <span><span className="k">провайдер</span><span className="v">jwt + bcrypt</span></span>
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
