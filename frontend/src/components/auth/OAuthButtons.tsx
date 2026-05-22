import React from "react";

interface OAuthButtonsProps {
  /** Опциональный override — иначе используем API-base из глобальной
   *  переменной окружения VITE_API_URL (или fallback на `/api`). */
  apiBase?: string;
  onOAuthLogin?: (provider: string) => void;
  providers?: Array<"Google" | "GitHub">;
}

const resolveApiBase = (override?: string): string => {
  if (override) return override.replace(/\/+$/, "");
  // Vite exposes VITE_* env at build time. Дефолт — относительный `/api`,
  // который проксируется api-gateway'ом.
  const fromEnv = (import.meta as unknown as { env?: { VITE_API_URL?: string } })
    .env?.VITE_API_URL;
  return (fromEnv || "/api").replace(/\/+$/, "");
};

const OAuthButtons: React.FC<OAuthButtonsProps> = ({
  apiBase,
  onOAuthLogin,
  providers = ["Google", "GitHub"],
}) => {
  const handleOAuthClick = (provider: string) => {
    onOAuthLogin?.(provider);
    const base = resolveApiBase(apiBase);
    const slug = provider.toLowerCase();
    // Бэк (user-service) выдаёт 307 redirect на провайдера. Браузер сам
    // следует за ним; после callback бэк перенаправит обратно на
    // /auth/oauth-callback с access_token/refresh_token в URL.
    window.location.href = `${base}/v1/auth/oauth/${slug}`;
  };

  return (
    <div className="oauth-buttons">
      <p className="oauth-buttons__label">Или войти через</p>
      <div className="oauth-buttons__list">
        {providers.map((provider) => (
          <button
            key={provider}
            className={`oauth-button oauth-button--${provider.toLowerCase()}`}
            onClick={() => handleOAuthClick(provider)}
            type="button"
          >
            <span className="oauth-button__icon" aria-hidden="true">
              {provider === "Google" ? "G" : "GH"}
            </span>
            <span className="oauth-button__text">{provider}</span>
          </button>
        ))}
      </div>
    </div>
  );
};

export default OAuthButtons;
