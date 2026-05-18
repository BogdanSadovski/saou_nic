import React from 'react';

interface ProtectedRouteProps {
  children: React.ReactNode;
  isAuthenticated?: boolean;
  redirectPath?: string;
}

const ProtectedRoute: React.FC<ProtectedRouteProps> = ({
  children,
  isAuthenticated = false,
  redirectPath = '/login',
}) => {
  // Placeholder: in a real app, redirect logic would use React Router's Navigate
  if (!isAuthenticated) {
    return (
      <div className="protected-route">
        <p>Для доступа к этой странице нужно войти в систему.</p>
        <a href={redirectPath}>Перейти ко входу</a>
      </div>
    );
  }

  return <>{children}</>;
};

export default ProtectedRoute;
