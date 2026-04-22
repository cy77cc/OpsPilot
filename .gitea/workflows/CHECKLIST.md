# OpsPilot CI/CD 快速配置清单

## 📋 5分钟快速配置

### 必做项

- [ ] **进入仓库Settings** > Secrets and variables > Actions
  
- [ ] 添加Secrets（可选，跳过则不推送Docker镜像）:
  ```
  REGISTRY_USER = <你的Docker Registry用户名>
  REGISTRY_PASSWORD = <你的Docker Registry密码/Token>
  ```

- [ ] **编辑** `.gitea/workflows/docker.yaml` 的 `env` 部分:
  ```yaml
  REGISTRY: registry.example.com  # 改为你的Registry地址
  IMAGE_NAME: k8s-manage         # 改为你的镜像名称
  ```

### 验证配置

- [ ] Push一个test commit到develop分支
  ```bash
  git commit --allow-empty -m "test: trigger CI"
  git push origin develop
  ```

- [ ] 在Gitea Actions页面检查是否都通过 ✅

---

## 📊 工作流概览

| 工作流 | 触发条件 | 主要功能 | 需要配置 |
|--------|--------|--------|--------|
| **ci.yaml** | Push/PR到main或develop | Lint + Test + Build | ❌ 无 |
| **docker.yaml** | Push到main/develop或Tag | Docker构建 + 安全扫描 | ⚠️ Registry信息 |
| **deploy.yaml** | Tag推送或手动trigger | K8s部署 + 通知 | ⚠️ K8s配置 |

---

## 🔑 Secrets清单

### 必需（Docker镜像推送）
```
REGISTRY_USER        # Docker Registry 用户名
REGISTRY_PASSWORD    # Docker Registry 密码
```

### 可选（部署用）
```
KUBE_CONFIG          # Kubernetes配置 (base64)
WEBHOOK_URL          # 通知地址 (Slack/钉钉)
```

---

## 🚀 常用命令

### 触发CI工作流
```bash
git push origin develop        # 触发ci.yaml
git push origin main           # 触发ci.yaml + docker.yaml
```

### 跳过CI工作流
```bash
git commit -m "docs: update [skip ci]"
```

### 查看工作流运行日志
```
Gitea > 仓库 > Actions > 选择工作流 > 查看日志
```

---

## ⚙️ 配置文件位置

```
.gitea/workflows/
├── ci.yaml           # CI流程（必需）
├── docker.yaml       # Docker构建（可选）
├── deploy.yaml       # K8s部署（可选）
├── SETUP.md          # 详细配置指南
└── CHECKLIST.md      # 本文件
```

---

## ✅ 配置完成后的样子

```
Push to main/develop
    ↓
ci.yaml 工作流启动
    ├─ Lint ✅
    ├─ Test ✅
    ├─ Build ✅
    └─ Security Scan ✅
    ↓
(仅在main分支) docker.yaml 启动
    ├─ Build Docker Image ✅
    └─ Security Scan ✅
    
✨ 全自动！
```

---

## 📞 需要帮助？

- 详细配置说明见: `SETUP.md`
- 工作流改动需要Push后才能生效
- 所有artifact会保存7-30天供下载

