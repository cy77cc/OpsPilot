# 设计文档

## 技术方案

### 1. 路由修复

在 `AppRoutes.tsx` 添加：

```typescript
const ClusterImportWizard = lazy(() => import('../pages/Deployment/Infrastructure/ClusterImportWizard'));

// 路由配置
<Route
  path="/deployment/infrastructure/clusters/import"
  element={
    <AppLayout>
      <LazyPage>
        <ClusterImportWizard />
      </LazyPage>
    </AppLayout>
  }
/>
```

### 2. 认证方式设计

```
┌─────────────────────────────────────────────────────────────┐
│  AuthMethod = 'kubeconfig' | 'certificate' | 'token'        │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  kubeconfig:                                                │
│    └── kubeconfig: string (YAML 内容)                       │
│                                                             │
│  certificate:                                               │
│    ├── endpoint: string (API Server URL)                    │
│    ├── ca_cert: string (CA 证书, PEM/Base64)                │
│    ├── cert: string (客户端证书)                             │
│    └── key: string (客户端私钥)                              │
│                                                             │
│  token:                                                     │
│    ├── endpoint: string (API Server URL)                    │
│    ├── ca_cert?: string (CA 证书, 可选)                      │
│    ├── token: string (Bearer Token)                         │
│    └── skip_tls_verify?: boolean (跳过 TLS 验证)            │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 3. 向导步骤

**Step 1: 基本信息**
- 集群名称 (必填)
- 描述 (可选)

**Step 2: 认证方式**
- Radio 选择: Kubeconfig / 证书 / Token
- 每种方式的说明和适用场景

**Step 3: 连接配置**
- 根据 auth_method 动态渲染表单
- kubeconfig: TextArea + 文件上传
- certificate: endpoint + ca_cert + cert + key
- token: endpoint + ca_cert(可选) + token + skip_tls_verify

**Step 4: 连接测试**
- 调用 validateImport API
- 显示测试结果: 连通性、版本信息
- 失败时显示错误原因

**Step 5: 确认导入**
- 配置摘要展示
- 确认按钮

### 4. 组件结构

```tsx
ClusterImportWizard
├── State
│   ├── currentStep: number
│   ├── authMethod: 'kubeconfig' | 'certificate' | 'token'
│   ├── validationResult: ValidationResult | null
│   └── form: FormInstance
│
├── Steps
│   ├── Step 0: BasicInfoStep
│   ├── Step 1: AuthMethodStep
│   ├── Step 2: ConnectionConfigStep (动态)
│   ├── Step 3: ConnectionTestStep
│   └── Step 4: ConfirmStep
│
└── Handlers
    ├── handleValidate()
    ├── handleImport()
    └── handleNext/Prev()
```

### 5. 表单字段映射

```typescript
interface ClusterImportFormValues {
  // 基本信息
  name: string;
  description?: string;

  // 认证方式
  auth_method: 'kubeconfig' | 'certificate' | 'token';

  // Kubeconfig 方式
  kubeconfig?: string;

  // 证书方式
  endpoint?: string;
  ca_cert?: string;
  cert?: string;
  key?: string;

  // Token 方式
  token?: string;
  skip_tls_verify?: boolean;
}
```

### 6. API 调用

```typescript
// 验证连接
Api.cluster.validateImport({
  kubeconfig?: string;
  endpoint?: string;
  ca_cert?: string;
  cert?: string;
  key?: string;
  token?: string;
})

// 导入集群
Api.cluster.importCluster({
  name: string;
  description?: string;
  kubeconfig?: string;
  endpoint?: string;
  ca_cert?: string;
  cert?: string;
  key?: string;
  token?: string;
  auth_method?: string;
})
```
