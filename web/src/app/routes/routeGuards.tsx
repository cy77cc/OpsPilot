import React from 'react';
import { Authorized } from '../../components/RBAC';
import AccessDeniedPage from '../../components/Auth/AccessDeniedPage';

export type WithAuth = (
  resource: string,
  action: string,
  element: React.ReactElement,
) => React.ReactElement;

export const createWithAuth = (): WithAuth => (resource, action, element) => (
  <Authorized resource={resource} action={action} fallback={<AccessDeniedPage />}>
    {element}
  </Authorized>
);
