import React from 'react';
import type { ReactNode } from 'react';
import { usePermission } from './PermissionContext';
import AccessDeniedPage from '../Auth/AccessDeniedPage';
import { PageSkeleton } from '../LoadingSkeleton';

// 权限验证属性
export interface AuthorizedProps {
  resource: string;
  action: string;
  children: ReactNode;
  fallback?: ReactNode;
}

// 权限验证组件
const Authorized: React.FC<AuthorizedProps> = ({ 
  resource, 
  action, 
  children, 
  fallback 
}) => {
  const { loading, hasPermission } = usePermission();

  if (loading) {
    return <PageSkeleton />;
  }

  if (!hasPermission(resource, action)) {
    return fallback || <AccessDeniedPage compact />;
  }

  return <>{children}</>;
};

export default Authorized;
