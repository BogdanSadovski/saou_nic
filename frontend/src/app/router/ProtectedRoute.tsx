import { Navigate, useLocation } from "react-router-dom";
import type { PropsWithChildren } from "react";

import { useAuthStore } from "@/app/store";
import { Loader } from "@/shared/ui";

export function ProtectedRoute({ children }: PropsWithChildren) {
  const location = useLocation();
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  const isInitialized = useAuthStore((state) => state.isInitialized);

  if (!isInitialized) {
    return (
      <div className="full-loader">
        <Loader />
      </div>
    );
  }

  if (!isAuthenticated) {
    return <Navigate replace state={{ from: location.pathname }} to="/auth" />;
  }

  return <>{children}</>;
}
