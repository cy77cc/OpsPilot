import React from 'react';
import { Navigate, useLocation } from 'react-router-dom';
import { Api } from '../../api';

interface LegacyGovernanceRedirectProps {
  to: string;
}

const LegacyGovernanceRedirect: React.FC<LegacyGovernanceRedirectProps> = ({ to }) => {
  const location = useLocation();

  React.useEffect(() => {
    void Api.rbac.recordMigrationEvent({
      eventType: 'legacy_redirect',
      fromPath: location.pathname,
      toPath: to,
      status: 'redirected',
    }).catch(() => undefined);
  }, [location.pathname, to]);

  return <Navigate to={to} replace />;
};

export default LegacyGovernanceRedirect;
