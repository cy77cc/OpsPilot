# 集群导入功能完善

## 概述

完善集群创建/导入功能，解决"只有骨架没有内容"的问题，使其能够真正用于导入和管理外部 Kubernetes 集群。

## 背景

当前存在以下问题：
1. **路由缺失**: `AppRoutes.tsx` 缺少 `/deployment/infrastructure/clusters/import` 路由
2. **导入方式单一**: 仅支持 kubeconfig 一种认证方式
3. **表单字段不完整**: API 定义了 endpoint/ca_cert/cert/key/token 等字段，但 UI 未提供输入
4. **流程过于简单**: 缺少认证方式选择和完整的连接测试流程

## 目标

- 修复路由，使导入页面可访问
- 支持三种认证方式：Kubeconfig / API地址+证书 / ServiceAccount Token
- 完善表单字段，覆盖后端 API 定义
- 添加连接测试步骤，提升用户体验

## 范围

### 包含
- `web/src/routes/AppRoutes.tsx` - 添加 import 路由
- `web/src/pages/Deployment/Infrastructure/ClusterImportWizard.tsx` - 重构向导组件
- `web/src/types/cluster.types.ts` - 更新类型定义（如需要）

### 不包含
- 后端 API 修改（假设已支持所有字段）
- 集群创建 (Bootstrap) 功能修改
- 集群详情页修改

## 成功标准

1. 访问 `/deployment/infrastructure/clusters/import` 能正常显示导入向导
2. 用户可以选择三种认证方式之一
3. 根据选择的认证方式，显示对应的表单字段
4. 能够成功导入外部 Kubernetes 集群
5. 导入后能在集群列表看到新导入的集群
