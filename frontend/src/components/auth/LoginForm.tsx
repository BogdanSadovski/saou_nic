import React, { useState, FormEvent } from 'react';

interface LoginFormProps {
  onLogin?: (email: string, password: string) => void;
  isLoading?: boolean;
  error?: string;
}

const LoginForm: React.FC<LoginFormProps> = ({ onLogin, isLoading = false, error }) => {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault();
    onLogin?.(email, password);
  };

  return (
    <form className="login-form" onSubmit={handleSubmit}>
      <h2 className="login-form__title">Войти</h2>

      {error && <div className="login-form__error">{error}</div>}

      <div className="login-form__field">
        <label htmlFor="email" className="login-form__label">
          Email
        </label>
        <input
          id="email"
          type="email"
          className="login-form__input"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          placeholder="you@example.com"
          required
        />
      </div>

      <div className="login-form__field">
        <label htmlFor="password" className="login-form__label">
          Пароль
        </label>
        <input
          id="password"
          type="password"
          className="login-form__input"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          placeholder="••••••••"
          required
        />
      </div>

      <button type="submit" className="login-form__submit" disabled={isLoading}>
        {isLoading ? 'Входим...' : 'Войти'}
      </button>
    </form>
  );
};

export default LoginForm;
