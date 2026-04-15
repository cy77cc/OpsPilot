// RBAC组件索引文件
export { default as Authorized } from './Authorized';
export { PermissionProvider, usePermission } from './PermissionContext';

import Authorized from './Authorized';
import { PermissionProvider, usePermission } from './PermissionContext';

export default {
  Authorized,
  PermissionProvider,
  usePermission,
};
