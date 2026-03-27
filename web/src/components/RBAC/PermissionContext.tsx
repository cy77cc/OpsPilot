import React, { createContext, useContext, useState, useEffect } from 'react';
import type { ReactNode } from 'react';
import { rbacApi } from '../../api/modules/rbac';
import { useAuth } from '../Auth/AuthContext';

// 权限上下文类型
interface PermissionContextType {
  permissions: string[];
  loading: boolean;
  hasPermission: (resource: string, action: string) => boolean;
  refreshPermissions: () => Promise<void>;
}

// 创建权限上下文
const PermissionContext = createContext<PermissionContextType | undefined>(undefined);

// 权限上下文提供者属性
interface PermissionProviderProps {
  children: ReactNode;
}

// 权限上下文提供者组件
export const PermissionProvider: React.FC<PermissionProviderProps> = ({ children }) => {
  const { isAuthenticated } = useAuth();
  const [permissions, setPermissions] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);

  // 加载用户权限
  const loadPermissions = async () => {
    try {
      setLoading(true);
      if (!isAuthenticated) {
        setPermissions([]);
        return;
      }
      const res = await rbacApi.getMyPermissions();
      setPermissions(res.data || []);
    } catch (error) {
      console.error('加载权限失败:', error);
      setPermissions([]);
    } finally {
      setLoading(false);
    }
  };

  // 初始化加载权限
  useEffect(() => {
    loadPermissions();
  }, [isAuthenticated]);

  // 检查权限
  const hasPermission = (resource: string, action: string): boolean => {
    const permissionCode = `${resource}:${action}`;
    return permissions.includes(permissionCode) ||
      permissions.includes(`${resource}:*`) ||
      permissions.includes('*:*');
  };

  // 刷新权限
  const refreshPermissions = async () => {
    await loadPermissions();
  };

  const value: PermissionContextType = {
    permissions,
    loading,
    hasPermission,
    refreshPermissions
  };

  return (
    <PermissionContext.Provider value={value}>
      {children}
    </PermissionContext.Provider>
  );
};

// 权限上下文钩子
export const usePermission = (): PermissionContextType => {
  const context = useContext(PermissionContext);
  if (context === undefined) {
    throw new Error('usePermission must be used within a PermissionProvider');
  }
  return context;
};

export default PermissionContext;
