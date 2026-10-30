import {
  createBrowserRouter,
  createRoutesFromElements,
  RouterProvider,
  Route,
  Navigate,
} from "react-router-dom";
import { lazy, Suspense } from "react";
import AuthProvider from "./store/context/AuthProvider";
import { useRealtime } from "./hooks/useRealtime";
import MainLayout from "./components/layout/MainLayout";
import AdminLayout from "./components/layout/AdminLayout";
import ErrorBoundary from "./components/layout/ErrorBoundary";
import ProtectedRoute from "./components/auth/ProtectedRoute";

const Home = lazy(() => import("./pages/Home"));
const LoginPage = lazy(() => import("./pages/Login"));
const SpaceAccess = lazy(() => import("./pages/SpaceAccess"));
const AdminPanel = lazy(() => import("./pages/AdminPanel"));
const AdminSettings = lazy(() => import("./pages/AdminSettings"));
const SpaceManagement = lazy(() => import("./components/SpaceManagement"));
const GroupManagement = lazy(() => import("./components/GroupManagement"));
const Profile = lazy(() => import("./pages/Profile"));
const Secrets = lazy(() => import("./pages/Secrets"));
const Webhooks = lazy(() => import("./pages/Webhooks"));

const RouteLoader = () => (
  <div
    role="status"
    className="flex items-center justify-center min-h-64 text-slate-300"
  >
    Loading page…
  </div>
);

const router = createBrowserRouter(
  createRoutesFromElements(
    <Route
      element={
        <AuthProvider>
          <ErrorBoundary>
            <Suspense fallback={<RouteLoader />}>
              <MainLayout />
            </Suspense>
          </ErrorBoundary>
        </AuthProvider>
      }
    >
      <Route path="/" element={<Home />} />
      <Route path="/login" element={<LoginPage />} />
      <Route path="/profile" element={<Profile />} />
      <Route element={<ProtectedRoute allowedRoles={["admin"]} />}>
        <Route path="/admin" element={<AdminLayout />}>
          <Route index element={<Navigate to="users" replace />} />
          <Route path="users" element={<AdminPanel />} />
          <Route path="groups" element={<GroupManagement />} />
          <Route path="spaces" element={<SpaceManagement />} />
          <Route path="spaces/:spaceId" element={<SpaceAccess />} />
          <Route path="secrets" element={<Secrets />} />
          <Route path="webhooks" element={<Webhooks />} />
          <Route
            path="settings"
            element={<Navigate to="authentication" replace />}
          />
          <Route path="settings/:section" element={<AdminSettings />} />
        </Route>
        <Route
          path="/secrets"
          element={<Navigate to="/admin/secrets" replace />}
        />
        <Route
          path="/webhooks"
          element={<Navigate to="/admin/webhooks" replace />}
        />
      </Route>
    </Route>,
  ),
);

const App = () => {
  useRealtime();
  return <RouterProvider router={router} />;
};

export default App;
