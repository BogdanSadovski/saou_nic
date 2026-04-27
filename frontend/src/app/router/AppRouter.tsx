import { Suspense, lazy } from "react";
import { Navigate, Route, Routes } from "react-router-dom";

import { Loader } from "@/shared/ui";
import { ProtectedRoute } from "./ProtectedRoute";
import { AppShell } from "./AppShell";

const HomePage = lazy(() => import("@/pages/Home/index"));
const AuthPage = lazy(() => import("@/pages/Auth/index"));
const DashboardPage = lazy(() => import("@/pages/Dashboard/index"));
const CareerCenterPage = lazy(() => import("@/pages/CareerCenter/index"));
const InterviewSetupPage = lazy(() => import("@/pages/InterviewSetup/index"));
const InterviewSessionPage = lazy(() => import("@/pages/InterviewSession/index"));
const InterviewResultPage = lazy(() => import("@/pages/InterviewResult/index"));
const ProfilePage = lazy(() => import("@/pages/Profile/index"));
const PublicProfilePage = lazy(() => import("@/pages/PublicProfile/index"));
const ReportsPage = lazy(() => import("@/pages/Reports/index"));
const ResumePage = lazy(() => import("@/pages/Resume/index"));
const AdminPage = lazy(() => import("@/pages/Admin/index"));
const ErrorPage = lazy(() => import("@/pages/Error/index"));

export function AppRouter() {
  return (
    <Suspense fallback={<div className="full-loader"><Loader /></div>}>
      <Routes>
        <Route element={<AppShell />}>
          <Route index element={<HomePage />} />
          <Route path="auth" element={<AuthPage />} />
          <Route
            path="dashboard"
            element={
              <ProtectedRoute>
                <DashboardPage />
              </ProtectedRoute>
            }
          />
          <Route
            path="career-center"
            element={
              <ProtectedRoute>
                <CareerCenterPage />
              </ProtectedRoute>
            }
          />
          <Route
            path="interview"
            element={
              <ProtectedRoute>
                <InterviewSetupPage />
              </ProtectedRoute>
            }
          />
          <Route
            path="interview/session/:sessionId"
            element={
              <ProtectedRoute>
                <InterviewSessionPage />
              </ProtectedRoute>
            }
          />
          <Route
            path="interview/result/:sessionId"
            element={
              <ProtectedRoute>
                <InterviewResultPage />
              </ProtectedRoute>
            }
          />
          <Route
            path="profile"
            element={
              <ProtectedRoute>
                <ProfilePage />
              </ProtectedRoute>
            }
          />
          <Route path="public-profile" element={<PublicProfilePage />} />
          <Route
            path="reports"
            element={
              <ProtectedRoute>
                <ReportsPage />
              </ProtectedRoute>
            }
          />
          <Route
            path="resume"
            element={
              <ProtectedRoute>
                <ResumePage />
              </ProtectedRoute>
            }
          />
          <Route
            path="admin"
            element={
              <ProtectedRoute>
                <AdminPage />
              </ProtectedRoute>
            }
          />
          <Route path="404" element={<ErrorPage />} />
          <Route path="*" element={<Navigate replace to="/404" />} />
        </Route>
      </Routes>
    </Suspense>
  );
}
