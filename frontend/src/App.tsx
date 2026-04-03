import { BrowserRouter, Routes, Route } from "react-router-dom";
import { lazy, Suspense } from "react";
import AuthProvider from "./store/context/AuthProvider.tsx";
import { useRealtime } from "./hooks/useRealtime";
import MainLayout from "./components/layout/MainLayout";
import ErrorBoundary from "./components/layout/ErrorBoundary";
import ProtectedRoute from "./components/auth/ProtectedRoute";

// Lazy load route components for code splitting
const Home = lazy(() => import("./pages/Home.tsx"));
const LoginPage = lazy(() => import("./pages/Login.tsx"));
const AdminPanel = lazy(() => import("./pages/AdminPanel.tsx"));
const AdminSettings = lazy(() => import("./pages/AdminSettings.tsx"));
const Profile = lazy(() => import("./pages/Profile.tsx"));
const Secrets = lazy(() => import("./pages/Secrets"));
const Webhooks = lazy(() => import("./pages/Webhooks"));

// Loading fallback component
const RouteLoader = () => (
  <div className="flex items-center justify-center min-h-screen">
    <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-cyan-500"></div>
  </div>
);

const App = () => {
  useRealtime();

  return (
    <BrowserRouter>
      <AuthProvider>
        <Suspense fallback={<RouteLoader />}>
          <ErrorBoundary>
            <Routes>
              <Route element={<MainLayout />}>
                <Route path="/" element={<Home />} />
                <Route path="/login" element={<LoginPage />} />
                <Route path="/profile" element={<Profile />} />
                
                {/* Admin and Settings endpoints heavily rely on backend permissions */}
                <Route element={<ProtectedRoute allowedRoles={["admin"]} />}>
                  <Route path="/admin" element={<AdminPanel />} />
                  <Route path="/admin/settings" element={<AdminSettings />} />
                  <Route path="/secrets" element={<Secrets />} />
                  <Route path="/webhooks" element={<Webhooks />} />
                </Route>
              </Route>
            </Routes>
          </ErrorBoundary>
        </Suspense>
      </AuthProvider>
    </BrowserRouter>
  );
};

export default App;
