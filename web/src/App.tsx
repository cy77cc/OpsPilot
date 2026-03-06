import React, { Suspense, lazy } from 'react';
import { BrowserRouter, Navigate, Route, Routes, useLocation } from 'react-router-dom';
import { message } from 'antd';
import { AuthProvider, useAuth } from './components/Auth/AuthContext';
import { PermissionProvider } from './components/RBAC/PermissionContext';

const LoginPage = lazy(() => import('./pages/Auth/LoginPage'));
const RegisterPage = lazy(() => import('./pages/Auth/RegisterPage'));
const ProtectedApp = lazy(() => import('./ProtectedApp'));

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

/**
 * 旧 /ai 路由重定向组件
 * 显示提示后重定向到首页
 */
const AIRedirect: React.FC = () => {
  React.useEffect(() => {
    message.info('AI 助手已移至右上角，点击 "AI 助手" 按钮或按 Cmd+/ 打开');
  }, []);

  return <Navigate to="/" replace />;
};

const App: React.FC = () => {
  return (
    <AuthProvider>
      <BrowserRouter>
        <Suspense fallback={<RouteFallback />}>
          <Routes>
            <Route path="/login" element={<LoginPage />} />
            <Route path="/register" element={<RegisterPage />} />
            {/* 旧 /ai 路由重定向到首页 */}
            <Route
              path="/ai"
              element={
                <ProtectedRoute>
                  <PermissionProvider>
                    <AIRedirect />
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
