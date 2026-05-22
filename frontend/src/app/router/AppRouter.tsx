import { Suspense, lazy } from "react";
import { Navigate, Route, Routes } from "react-router-dom";

import { Loader } from "@/shared/ui";
import { ProtectedRoute } from "./ProtectedRoute";
import { AppShell } from "./AppShell";

const HomePage = lazy(() => import("@/pages/Home/index"));
const AuthPage = lazy(() => import("@/pages/Auth/index"));
const WorkspaceLayout = lazy(() => import("@/pages/Workspace/WorkspaceLayout"));
const WorkspaceOverview = lazy(() => import("@/pages/Workspace/Overview"));
const CareerCenterPage = lazy(() => import("@/pages/CareerCenter/index"));
const InterviewSetupPage = lazy(() => import("@/pages/InterviewSetup/index"));
const InterviewSessionPage = lazy(() => import("@/pages/InterviewSession/index"));
const InterviewResultPage = lazy(() => import("@/pages/InterviewResult/index"));
const ProfilePage = lazy(() => import("@/pages/Profile/index"));
const PublicProfilePage = lazy(() => import("@/pages/PublicProfile/index"));
const ReportsPage = lazy(() => import("@/pages/Reports/index"));
const ResumePage = lazy(() => import("@/pages/Resume/index"));
const AdminPage = lazy(() => import("@/pages/Admin/index"));
const CheckoutPage = lazy(() => import("@/pages/Billing/Checkout"));
const BillingPage = lazy(() => import("@/pages/Billing/index"));
const ErrorPage = lazy(() => import("@/pages/Error/index"));

export function AppRouter() {
  return (
    <Suspense fallback={<div className="full-loader"><Loader /></div>}>
      <Routes>
        <Route element={<AppShell />}>
          <Route index element={<HomePage />} />
          <Route path="auth" element={<AuthPage />} />
          {/* Workspace — persistent rail with content swap via Outlet */}
          <Route
            path="workspace"
            element={
              <ProtectedRoute>
                <WorkspaceLayout />
              </ProtectedRoute>
            }
          >
            <Route index element={<WorkspaceOverview />} />
            <Route path="career" element={<CareerCenterPage />} />
            <Route path="profile" element={<ProfilePage />} />
            <Route path="resume" element={<ResumePage />} />
            <Route path="billing" element={<BillingPage />} />
            <Route path="admin" element={<AdminPage />} />
          </Route>

          {/* Legacy top-level redirects → workspace */}
          <Route path="dashboard" element={<Navigate replace to="/workspace" />} />
          <Route path="career-center" element={<Navigate replace to="/workspace/career" />} />
          <Route path="profile" element={<Navigate replace to="/workspace/profile" />} />
          <Route path="resume" element={<Navigate replace to="/workspace/resume" />} />
          <Route path="admin" element={<Navigate replace to="/workspace/admin" />} />

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
          <Route path="public-profile" element={<PublicProfilePage />} />
          <Route
            path="reports"
            element={
              <ProtectedRoute>
                <ReportsPage />
              </ProtectedRoute>
            }
          />
          <Route path="404" element={<ErrorPage />} />
        </Route>

        {/* External-style checkout — intentionally rendered OUTSIDE
            AppShell so navbar/sidebar/banner don't appear. The page
            looks like a 3rd-party hosted gateway (Stripe-like). */}
        <Route
          path="billing/checkout"
          element={
            <ProtectedRoute>
              <CheckoutPage />
            </ProtectedRoute>
          }
        />

        <Route path="*" element={<Navigate replace to="/404" />} />
      </Routes>
    </Suspense>
  );
}
