import React from 'react';
import { Navigate, Route, Routes } from 'react-router-dom';
import type { WithAuth } from './routeGuards';
import { renderWorkloadRoutes } from './workloads.routes';
import { renderDeploymentRoutes } from './deployment.routes';
import { renderInfrastructureRoutes } from './infrastructure.routes';
import { renderObservabilityRoutes } from './observability.routes';
import { renderLegacyRoutes } from './legacy.routes';
import { renderPlatformRoutes } from './platform.routes';

interface ProtectedRoutesProps {
  withAuth: WithAuth;
  governanceMenuEnabled: boolean;
}

const ProtectedRoutes: React.FC<ProtectedRoutesProps> = ({ withAuth, governanceMenuEnabled }) => (
  <Routes>
    {renderWorkloadRoutes(withAuth)}
    {renderDeploymentRoutes(withAuth)}
    {renderInfrastructureRoutes(withAuth)}
    {renderObservabilityRoutes(withAuth)}
    {renderLegacyRoutes({ withAuth })}
    {renderPlatformRoutes({ withAuth, governanceMenuEnabled })}
    <Route path="*" element={<Navigate to="/" replace />} />
  </Routes>
);

export default ProtectedRoutes;
