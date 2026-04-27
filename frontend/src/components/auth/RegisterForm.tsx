import React, { useState, FormEvent } from 'react';

interface RegisterFormProps {
  onRegister?: (name: string, email: string, password: string) => void;
  isLoading?: boolean;
  error?: string;
}

const RegisterForm: React.FC<RegisterFormProps> = ({
  onRegister,
  isLoading = false,
  error,
}) => {
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault();
    if (password !== confirmPassword) {
      return;
    }
    onRegister?.(name, email, password);
  };

  return (
    <form className="register-form" onSubmit={handleSubmit}>
      <h2 className="register-form__title">Create Account</h2>

      {error && <div className="register-form__error">{error}</div>}

      <div className="register-form__field">
        <label htmlFor="name" className="register-form__label">
          Full Name
        </label>
        <input
          id="name"
          type="text"
          className="register-form__input"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="John Doe"
          required
        />
      </div>

      <div className="register-form__field">
        <label htmlFor="email" className="register-form__label">
          Email
        </label>
        <input
          id="email"
          type="email"
          className="register-form__input"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          placeholder="you@example.com"
          required
        />
      </div>

      <div className="register-form__field">
        <label htmlFor="password" className="register-form__label">
          Password
        </label>
        <input
          id="password"
          type="password"
          className="register-form__input"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          placeholder="••••••••"
          required
        />
      </div>

      <div className="register-form__field">
        <label htmlFor="confirmPassword" className="register-form__label">
          Confirm Password
        </label>
        <input
          id="confirmPassword"
          type="password"
          className="register-form__input"
          value={confirmPassword}
          onChange={(e) => setConfirmPassword(e.target.value)}
          placeholder="••••••••"
          required
        />
      </div>

      <button type="submit" className="register-form__submit" disabled={isLoading}>
        {isLoading ? 'Creating account...' : 'Create Account'}
      </button>
    </form>
  );
};

export default RegisterForm;
