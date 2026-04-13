import React from 'react';
import { Navigate, Route } from 'react-router-dom';
import type { WithAuth } from './routeGuards';
import { K8sPage } from './pages';

interface LegacyRoutesProps {
  withAuth: WithAuth;
}

export function renderLegacyRoutes({ withAuth }: LegacyRoutesProps) {
  return (
    <>
    <Route path="/k8s" element={<Navigate to="/deployment" replace />} />
    <Route path="/k8s/:cluster" element={<Navigate to="/deployment" replace />} />
    <Route path="/k8s-legacy" element={withAuth('kubernetes', 'read', <K8sPage />)} />
    <Route path="/hosts" element={<Navigate to="/deployment/infrastructure/hosts" replace />} />
    <Route path="/hosts/onboarding" element={<Navigate to="/deployment/infrastructure/hosts/onboarding" replace />} />
    <Route path="/hosts/keys" element={<Navigate to="/deployment/infrastructure/hosts/keys" replace />} />
    <Route path="/hosts/cloud-import" element={<Navigate to="/deployment/infrastructure/hosts/cloud-import" replace />} />
    <Route path="/hosts/virtualization" element={<Navigate to="/deployment/infrastructure/hosts/virtualization" replace />} />
      <Route path="/hosts/detail/:id" element={<Navigate to="/deployment/infrastructure/hosts/:id" replace />} />
      <Route path="/hosts/terminal/:id" element={<Navigate to="/deployment/infrastructure/hosts/:id/terminal" replace />} />
    </>
  );
}
