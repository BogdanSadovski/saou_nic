import React from 'react';

interface HeaderProps {
  title?: string;
  onMenuToggle?: () => void;
}

const Header: React.FC<HeaderProps> = ({ title = 'Панель', onMenuToggle }) => {
  return (
    <header className="header">
      <button
        className="header__menu-toggle"
        onClick={onMenuToggle}
        aria-label="Переключить меню"
      >
        <span className="header__menu-icon" />
      </button>
      <h1 className="header__title">{title}</h1>
      <div className="header__actions">
        {/* Placeholder for notifications, user avatar, etc. */}
      </div>
    </header>
  );
};

export default Header;
