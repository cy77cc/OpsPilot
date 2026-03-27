# Gitea Actions 配置说明

## 文件结构

```
.gitea/workflows/
├── ci.yaml       # Go CI: lint、test、build
├── docker.yaml   # Docker 镜像构建与推送
└── deploy.yaml   # Kubernetes 部署
```

## 工作流触发条件

| Workflow | 触发条件 |
|----------|----------|
| `ci.yaml` | push 到 main/develop，或 PR 到 main |
| `docker.yaml` | push 到 main，或推送 tag |
| `deploy.yaml` | 推送 tag (v*)，或手动触发 |

## 需要配置的 Secrets

在 Gitea 仓库设置中添加以下 Secrets：

```
# Docker Registry 认证
REGISTRY_USER=your-username
REGISTRY_PASSWORD=your-password-or-token

# Kubernetes 配置 (base64 编码)
KUBE_CONFIG=<base64-encoded-kubeconfig>

# 通知 Webhook (可选)
WEBHOOK_URL=https://your-webhook-url
```

## 使用方法

### 1. 日常开发 - 自动 CI

```bash
# 推送到 main/develop 分支自动触发 CI
git push origin main
```

CI 流程：Lint → Test → Build

### 2. 构建 Docker 镜像

```bash
# 推送 tag 触发镜像构建
git tag v1.0.0
git push origin v1.0.0
```

镜像标签规则：
- `v1.0.0` → `registry.example.com/owner/repo:v1.0.0`
- `v1.0.0` → `registry.example.com/owner/repo:1.0` (major.minor)
- `main` → `registry.example.com/owner/repo:main`

### 3. 部署到环境

```bash
# 方式1: 推送 tag 自动部署
git tag v1.0.0
git push origin v1.0.0

# 方式2: 手动触发
# 在 Gitea Actions 页面选择 "Deploy" workflow
# 选择目标环境: staging / production
```

## 配置自定义

### 修改 Registry 地址

编辑 `docker.yaml` 和 `deploy.yaml`：

```yaml
env:
  REGISTRY: registry.example.com  # 改为你的 Registry
```

### 修改 Kubernetes 部署

编辑 `deploy.yaml` 中的部署命令：

```yaml
- name: Deploy to Kubernetes
  run: |
    kubectl set image deployment/your-deployment-name \
      your-container-name=${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:${{ needs.prepare.outputs.image_tag }} \
      -n your-namespace
```

### 添加新的环境

编辑 `deploy.yaml`：

```yaml
inputs:
  environment:
    options:
      - staging
      - production
      - development  # 新增环境
```

## 本地测试

使用 act 在本地测试 workflow：

```bash
# 安装 act
go install github.com/nektos/act@latest

# 本地运行 CI
act -j lint

# 本地运行所有 job
act
```

## 注意事项

1. **Runner 要求**: 需要配置 Docker 和 kubectl
2. **镜像缓存**: 使用 buildx 缓存加速构建
3. **安全扫描**: Trivy 扫描镜像漏洞
4. **部署保护**: production 环境需要确认

## Gitea 特性

- 使用绝对 URL 引用 Actions: `uses: https://github.com/actions/checkout@v4`
- 支持 `@daily` 等 schedule 扩展语法
- 不支持 `concurrency`、`permissions` 等字段
