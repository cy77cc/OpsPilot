import React, { Suspense, lazy } from 'react';
import { BrowserRouter, Navigate, Route, Routes, useLocation } from 'react-router-dom';
import { AuthProvider, useAuth } from './components/Auth/AuthContext';
import { PermissionProvider } from './components/RBAC/PermissionContext';

const LoginPage = lazy(() => import('./pages/Auth/LoginPage'));
const RegisterPage = lazy(() => import('./pages/Auth/RegisterPage'));
const ProtectedApp = lazy(() => import('./ProtectedApp'));
const AIAssistantPage = lazy(() => import('./pages/AI/Assistant'));
const AIDiagnosisReportPage = lazy(() => import('./pages/AI/DiagnosisReport'));

const RouteFallback: React.FC = () => (
  <div className="min-h-screen flex items-center justify-center">加载中...</div>
);

const ProtectedRoute: React.FC<{ children: React.ReactElement }> = ({ children }) => {
  const { isAuthenticated, loading } = useAuth();
  const location = useLocation();

  if (loading) {
    return <RouteFallback />;
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace state={{ from: location.pathname }} />;
  }

  return children;
};

const App: React.FC = () => {
  return (
    <AuthProvider>
      <BrowserRouter>
        <Suspense fallback={<RouteFallback />}>
          <Routes>
            <Route path="/login" element={<LoginPage />} />
            <Route path="/register" element={<RegisterPage />} />
            <Route
              path="/ai"
              element={
                <ProtectedRoute>
                  <PermissionProvider>
                    <AIAssistantPage />
                  </PermissionProvider>
                </ProtectedRoute>
              }
            />
            <Route
              path="/ai/diagnosis/:reportId"
              element={
                <ProtectedRoute>
                  <PermissionProvider>
                    <AIDiagnosisReportPage />
                  </PermissionProvider>
                </ProtectedRoute>
              }
            />
            <Route
              path="/*"
              element={
                <ProtectedRoute>
                  <ProtectedApp />
                </ProtectedRoute>
              }
            />
          </Routes>
        </Suspense>
      </BrowserRouter>
    </AuthProvider>
  );
};

export default App;
