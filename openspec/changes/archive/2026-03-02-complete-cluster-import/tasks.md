# 任务清单

## 任务列表

### Task 1: 修复路由配置
- [x] 在 `AppRoutes.tsx` 中添加 ClusterImportWizard 懒加载
- [x] 在 `AppRoutes.tsx` 中添加 `/deployment/infrastructure/clusters/import` 路由

**文件**: `src/routes/AppRoutes.tsx`

---

### Task 2: 重构 ClusterImportWizard 组件
- [x] 添加 authMethod 状态，支持三种认证方式选择
- [x] 重构为 5 步向导流程
- [x] 实现 Step 0: 基本信息
- [x] 实现 Step 1: 认证方式选择
- [x] 实现 Step 2: 连接配置（根据 authMethod 动态渲染）
- [x] 实现 Step 3: 连接测试
- [x] 实现 Step 4: 确认导入

**文件**: `src/pages/Deployment/Infrastructure/ClusterImportWizard.tsx`

---

### Task 3: 更新后端验证逻辑
- [x] 更新 ValidateImport 函数支持三种认证方式
- [x] 添加 testConnection 通用连接测试函数
- [x] 添加 buildRestConfigFromRequest 函数
- [x] 添加 syncClusterNodesWithCred 函数
- [x] 更新 ImportCluster 函数支持所有认证方式

**文件**: `internal/service/cluster/logic_import.go`

---

## 完成状态

✅ 所有任务已完成

## 验证方式

1. 访问 `/deployment/infrastructure/clusters/import` 页面正常显示
2. 能够选择不同认证方式并看到对应表单
3. 填写信息后能够测试连接
4. 能够成功导入集群
