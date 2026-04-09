---
name: modal-to-popconfirm-refactor
description: 将简单确认的 Modal.confirm 替换为 Popconfirm 以改善交互体验
type: project
---

# Modal.confirm 到 Popconfirm 重构设计

## 背景

系统中使用了大量 `Modal.confirm` 进行操作确认，但对于简单的删除/更新确认，全屏遮罩式对话框交互不够友好。Popconfirm 气泡确认框能保留上下文、操作更快、不打断流程。

## 设计决策

### 替换范围

**保留 Modal.confirm（需要输入的复杂场景）**：
- `HostListPage` - 批量命令执行（需输入命令）
- `HostListPage` - SSH 命令执行（需输入命令）
- `HostListPage` - 设为维护（需输入维护原因）
- `HostTerminalPage` - 重命名文件（需输入新名称）

**替换为 Popconfirm（简单确认场景）**：

| 文件 | 行号 | 操作 |
|------|------|------|
| HostDetailPage.tsx | 116 | 删除主机 |
| HostDetailPage.tsx | 246 | 删除主机（卡片内） |
| HostListPage.tsx | 298 | 删除主机 |
| HostCredentialsPage.tsx | 94 | 删除凭证 |
| HostCredentialsPage.tsx | 133 | 删除凭证 |
| HostTerminalPage.tsx | 466 | 删除文件 |
| HostTerminalPage.tsx | 531 | 放弃未保存修改 |
| UsersPage.tsx | 103 | 更新用户 |
| UsersPage.tsx | 131 | 删除用户 |
| RolesPage.tsx | 86 | 删除角色 |
| RolesPage.tsx | 179 | 更新角色权限 |
| PolicyManagementPage.tsx | 55 | 删除策略 |
| DeploymentDetailPage.tsx | 121 | 拒绝 Release |
| DeploymentDetailPage.tsx | 142 | 回滚 Release |
| ClusterDetailPage.tsx | 869 | 删除 Service |
| ClusterDetailPage.tsx | 888 | 删除 Ingress |
| ClusterDetailPage.tsx | 1024 | 续期证书 |
| ClusterDetailPage.tsx | 1045 | 升级集群 |
| ClusterDetailPage.tsx | 1089 | 删除 Deployment |
| ClusterDetailPage.tsx | 1130 | 删除 StatefulSet |
| ClusterDetailPage.tsx | 1145 | 删除 Pod |

### Popconfirm 样式

采用简洁风格，只显示标题，不显示描述：

```tsx
<Popconfirm
  title="确定删除此用户？"
  okText="确定"
  cancelText="取消"
  okButtonProps={{ danger: true }}
  onConfirm={async () => {
    // 原有逻辑
  }}
>
  <Button danger>删除</Button>
</Popconfirm>
```

## 实现要点

1. **导入更新**：添加 `Popconfirm` 到 antd 导入列表
2. **异步处理**：`onConfirm` 支持异步函数，Popconfirm 会自动显示加载状态
3. **位置调整**：使用 `placement` 属性确保气泡不超出视口
4. **危险操作**：保留 `okButtonProps={{ danger: true }}` 标识高风险操作

## 验收标准

- 所有简单确认场景替换为 Popconfirm
- 复杂输入场景保留 Modal.confirm
- 替换后功能正常运行
- 无新增 console.log 或调试代码