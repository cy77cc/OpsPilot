import React from 'react';
import { Navigate, Route, Routes } from 'react-router-dom';
import type { WithAuth } from './routeGuards';
import { renderWorkloadRoutes } from './workloads.routes';
import { renderDeliveryRoutes } from './delivery.routes';
import { renderResourceRoutes } from './resource.routes';
import { renderObservabilityRoutes } from './observability.routes';
import { renderLegacyRoutes } from './legacy.routes';
import { renderGovernanceRoutes } from './governance.routes';
import { renderAIRoutes } from './ai.routes';

interface ProtectedRoutesProps {
  withAuth: WithAuth;
  governanceMenuEnabled: boolean;
}

const ProtectedRoutes: React.FC<ProtectedRoutesProps> = ({ withAuth, governanceMenuEnabled }) => (
  <Routes>
    {renderOverviewRoutes(withAuth)}
    {renderDeliveryRoutes(withAuth)}
    {renderResourceRoutes(withAuth)}
    {renderObservabilityRoutes(withAuth)}
    {renderLegacyRoutes({ withAuth })}
    {renderGovernanceRoutes({ withAuth, governanceMenuEnabled })}
    {renderAIRoutes({ withAuth })}
    <Route path="*" element={<Navigate to="/" replace />} />
  </Routes>
);

export default ProtectedRoutes;
