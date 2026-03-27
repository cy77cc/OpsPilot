import React from 'react';
import type { ReactNode } from 'react';
import { Spin } from 'antd';
import { usePermission } from './PermissionContext';
import AccessDeniedPage from '../Auth/AccessDeniedPage';

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
    return <Spin size="large" className="flex justify-center items-center py-8" />;
  }

  if (!hasPermission(resource, action)) {
    return fallback || <AccessDeniedPage compact />;
  }

  return <>{children}</>;
};

// 权限验证工具函数
export const checkPermission = async (resource: string, action: string): Promise<boolean> => {
  const permissions = [
    'host:read',
    'host:write',
    'task:read',
    'task:write',
    'kubernetes:read',
    'kubernetes:write',
    'monitoring:read',
    'config:read',
    'config:write',
    'rbac:read',
    'rbac:write'
  ];
  return permissions.includes(`${resource}:${action}`);
};

export default Authorized;
