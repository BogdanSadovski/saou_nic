import React, { useState } from 'react';

interface GitHubConnectProps {
  onConnect?: (token: string) => void;
  isConnected?: boolean;
  onDisconnect?: () => void;
}

const GitHubConnect: React.FC<GitHubConnectProps> = ({
  onConnect,
  isConnected = false,
  onDisconnect,
}) => {
  const [token, setToken] = useState('');

  const handleConnect = () => {
    if (token.trim()) {
      onConnect?.(token.trim());
    }
  };

  if (isConnected) {
    return (
      <div className="github-connect github-connect--connected">
        <span className="github-connect__icon" />
        <span className="github-connect__status">GitHub подключен</span>
        <button className="github-connect__disconnect" onClick={onDisconnect}>
          Отключить
        </button>
      </div>
    );
  }

  return (
    <div className="github-connect">
      <h3 className="github-connect__title">Подключение к GitHub</h3>
      <p className="github-connect__description">
        Введите персональный токен GitHub для подключения аккаунта.
      </p>
      <div className="github-connect__input-group">
        <input
          type="password"
          className="github-connect__input"
          placeholder="ghp_xxxxxxxxxxxxxxxxxxxx"
          value={token}
          onChange={(e) => setToken(e.target.value)}
        />
        <button
          className="github-connect__button"
          onClick={handleConnect}
          disabled={!token.trim()}
        >
          Подключить
        </button>
      </div>
      <a
        href="https://github.com/settings/tokens"
        target="_blank"
        rel="noopener noreferrer"
        className="github-connect__link"
      >
        Сгенерировать токен
      </a>
    </div>
  );
};

export default GitHubConnect;
