import React from 'react';

interface OAuthButtonsProps {
  onOAuthLogin?: (provider: string) => void;
  providers?: string[];
}

const OAuthButtons: React.FC<OAuthButtonsProps> = ({
  onOAuthLogin,
  providers = ['Google', 'GitHub'],
}) => {
  const handleOAuthClick = (provider: string) => {
    onOAuthLogin?.(provider);
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
            <span className="oauth-button__icon" />
            <span className="oauth-button__text">{provider}</span>
          </button>
        ))}
      </div>
    </div>
  );
};

export default OAuthButtons;
