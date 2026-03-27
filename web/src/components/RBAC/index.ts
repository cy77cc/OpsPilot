// RBAC组件索引文件
export { default as Authorized, checkPermission } from './Authorized';
export { PermissionProvider, usePermission } from './PermissionContext';

import Authorized, { checkPermission } from './Authorized';
import { PermissionProvider, usePermission } from './PermissionContext';

export default {
  Authorized,
  PermissionProvider,
  usePermission,
  checkPermission
};