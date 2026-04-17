import React, { Suspense } from 'react';
import { PermissionProvider } from './components/RBAC';
import { useAuth } from './components/Auth/AuthContext';
import { NotificationProvider } from './contexts/NotificationContext';
import { PageTransition } from './components/Motion';
import { PageSkeleton } from './components/LoadingSkeleton';
import AppLayout from './app/layout/AppLayout';
import ProtectedRoutes from './app/routes';
import { createWithAuth } from './app/routes/routeGuards';

export default function ProtectedApp() {
  const { user } = useAuth();
  const governanceMenuEnabled = import.meta.env.VITE_FEATURE_GOVERNANCE_MENU !== 'false';
  const withAuth = createWithAuth();

  return (
    <PermissionProvider>
      <NotificationProvider userId={user?.id}>
        <AppLayout>
          <Suspense fallback={<PageSkeleton />}>
            <ProtectedRoutes withAuth={withAuth} governanceMenuEnabled={governanceMenuEnabled} />
          </Suspense>
        </AppLayout>
      </NotificationProvider>
    </PermissionProvider>
  );
}
