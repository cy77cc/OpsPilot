# Gitea Actions CI/CD 配置指南

本文档说明了OpsPilot项目的CI/CD工作流需要的所有配置项。

## 1. 仓库Secrets配置

在Gitea中访问：`你的仓库 > Settings > Secrets and variables > Actions`

### 必需Secrets（用于Docker镜像推送）

#### REGISTRY_USER
- **说明**: Docker Registry 登录用户名
- **示例**: `your-registry-username`
- **使用场景**: Docker镜像推送到私有仓库
- **是否必需**: 否（若为空，Docker push步骤会被跳过）

#### REGISTRY_PASSWORD
- **说明**: Docker Registry 登录密码/Token
- **示例**: `your-registry-token`
- **使用场景**: Docker镜像推送到私有仓库
- **是否必需**: 否（若为空，Docker push步骤会被跳过）

### 可选Secrets（用于通知和部署）

#### WEBHOOK_URL
- **说明**: 部署完成后的回调通知地址
- **示例**: `https://hooks.slack.com/services/YOUR/WEBHOOK/URL`
- **使用场景**: 发送Slack、钉钉等通知
- **是否必需**: 否

#### KUBE_CONFIG
- **说明**: Kubernetes集群配置（base64编码）
- **获取方法**: `cat ~/.kube/config | base64 | tr -d '\n'`
- **使用场景**: deploy.yaml中的Kubernetes部署
- **是否必需**: 仅部署时需要

## 2. 工作流变量配置

在Gitea中访问：`你的仓库 > Settings > Secrets and variables > Variables`

### REGISTRY (可选)
- **当前值**: `registry.example.com`
- **说明**: Docker Registry地址
- **修改路径**: `.gitea/workflows/docker.yaml` 中 `env.REGISTRY`
- **示例**: `docker.io`, `ghcr.io`, `your-private-registry.com`

### IMAGE_NAME (可选)
- **当前值**: `k8s-manage`
- **说明**: Docker镜像名称
- **修改路径**: `.gitea/workflows/docker.yaml` 中 `env.IMAGE_NAME`
- **示例**: `my-org/my-app`, `k8s-manage`

## 3. 具体配置步骤

### 步骤1: 基础配置（必需）
1. 访问仓库Settings
2. 在左侧菜单找到 "Secrets and variables" > "Actions"
3. 点击 "New secret" 并添加以下内容：

```
Secret Name: REGISTRY_USER
Value: your-docker-registry-username
```

```
Secret Name: REGISTRY_PASSWORD
Value: your-docker-registry-password
```

> 💡 提示：如果暂时不需要推送Docker镜像，可以跳过这步。CI流程仍会正常运行。

### 步骤2: Docker Registry配置（可选但推荐）

编辑 `.gitea/workflows/docker.yaml`，找到 `env` 部分：

```yaml
env:
  REGISTRY: registry.example.com      # 改为你的Registry地址
  IMAGE_NAME: k8s-manage             # 改为你的镜像名称
```

常见Registry配置：
- **Docker Hub**: `docker.io`, 镜像名: `your-username/k8s-manage`
- **GitHub Container Registry**: `ghcr.io`, 镜像名: `your-username/k8s-manage`
- **Aliyun**: `registry.cn-hangzhou.aliyuncs.com`, 镜像名: `your-namespace/k8s-manage`
- **Private Registry**: `your-private-registry.com`, 镜像名: `k8s-manage`

### 步骤3: 部署配置（仅需部署时）

如果需要使用 `deploy.yaml`，需要配置：

1. 添加Secrets:
```
Secret Name: KUBE_CONFIG
Value: <base64编码的kubeconfig>

Secret Name: WEBHOOK_URL
Value: <通知WebhookURL，如Slack、钉钉>
```

2. 编辑 `.gitea/workflows/deploy.yaml`，修改：
   - 第55行: `url: https://staging.example.com` → 实际地址
   - 第73行: 命令行改为实际的kubectl部署命令

## 4. 工作流说明

### ci.yaml (CI流程)
**触发条件**: Push到 `main` 或 `develop` 分支，或发起 PR

**执行步骤**:
1. ✅ **Lint** - Go代码检查 + 前端代码检查
2. ✅ **Test** - 后端单元测试 + 前端测试 + 覆盖率报告
3. ✅ **Build** - 编译后端二进制 + 前端构建
4. ✅ **Security Scan** - Go security 扫描

**产物**: 
- `coverage.html` - 测试覆盖率报告
- `binaries/` - 编译后的二进制文件 (amd64, arm64)
- `gosec-results.json` - 安全扫描结果

### docker.yaml (Docker构建)
**触发条件**: Push到 `main`/`develop` 分支、Tag推送、PR

**执行步骤**:
1. 🐳 **Build Docker Image** - 构建多架构镜像 (amd64, arm64)
2. 🔒 **Security Scan** - Trivy漏洞扫描

**自动Push条件**: 
- 只在**非PR**且**已配置REGISTRY_USER**时推送
- PR时仅构建不推送（使用本地缓存）

### deploy.yaml (部署流程)
**触发条件**: Tag推送 (v*) 或手动trigger

**执行步骤**:
1. 🚀 **Deploy** - 更新Kubernetes Deployment
2. 📢 **Notify** - 发送通知到Webhook

> ⚠️ 需要配置Kubernetes环境，见上方步骤3

## 5. 验证配置

### 检查点1: CI工作流
```bash
# 提交一个test commit
git commit --allow-empty -m "test: trigger CI"
git push origin develop

# 查看Gitea Actions页面，应该看到:
# - Lint: ✅ 通过
# - Test: ✅ 通过  
# - Build: ✅ 通过
# - Security Scan: ✅ 通过
```

### 检查点2: Docker构建（可选）
若已配置Registry Secret：
```bash
# Push到main分支
git push origin main

# Gitea Actions > docker.yaml 应该:
# - Build: ✅ 通过并推送镜像
# - Scan: ✅ 扫描完成
```

### 检查点3: 部署（可选）
若已配置KUBE_CONFIG：
```bash
# 创建Tag
git tag v1.0.0
git push origin v1.0.0

# Gitea Actions > deploy.yaml 应该:
# - Prepare: ✅ 识别环境
# - Deploy: ✅ 更新K8s
# - Notify: ✅ 发送通知
```

## 6. 常见问题

### Q: 为什么Docker push步骤被跳过了？
A: 检查是否已设置 `REGISTRY_USER` 和 `REGISTRY_PASSWORD` Secrets。

### Q: 前端测试失败，说找不到Node modules
A: 确保 `web/package-lock.json` 文件存在且是最新的。

### Q: Go版本不匹配
A: 编辑 `ci.yaml`，在 `env.GO_VERSION` 改为 `1.26.1`（已默认设置）。

### Q: 如何跳过某个工作流？
A: Commit message中包含 `[skip ci]` 或 `[ci skip]`：
```bash
git commit -m "docs: update README [skip ci]"
```

### Q: 覆盖率阈值多少？
A: 当前设置为40%（在Makefile中）。修改路径：
```makefile
# Makefile line 46
if [ $$(echo "$$coverage < 40" | bc -l) -eq 1 ]; then
```

## 7. 下一步

1. ✅ 配置必需Secrets（如果需要Docker push）
2. ✅ 修改Registry地址和镜像名称
3. ✅ Push一个测试commit验证CI流程
4. ✅ （可选）配置Kubernetes部署
5. ✅ （可选）配置Webhook通知

**所有配置完成后，CI/CD流程就能自动运行了！** 🎉
