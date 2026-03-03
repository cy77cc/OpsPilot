## 灰度发布与回滚 Runbook

### 目标
在不影响现网集群创建成功率的前提下，灰度发布参数化 bootstrap（动态版本、mirror、VIP provider、external etcd）。

### 发布前检查
- 后端已完成迁移：`20260303_000024`、`20260303_000025`
- `openspec validate --changes --json` 通过
- `go test ./...` 通过
- 前端构建 `npm -C web run build` 通过
- 目标环境已准备 mirror repo / image registry / VIP 网段

### 灰度步骤
1. 在内网测试环境开启新路径（`BOOTSTRAP_INIT_MODE=config`）。
2. 先验证 `repo_mode=online + nodeIP + stacked` 基础链路。
3. 验证 `repo_mode=mirror + image_repository + kube-vip` 主路径。
4. 验证 `endpoint_mode=vip + keepalived` 兼容路径。
5. 验证 `etcd_mode=external` 路径（含 TLS 预检）。
6. 观察指标：创建成功率、平均耗时、失败域（repo/endpoint/etcd/version）。
7. 扩大到 10% -> 30% -> 100% 用户组。

### 失败判定阈值
- 15 分钟窗口内创建失败率 > 5%
- 单一失败域（repo/endpoint/etcd）占比 > 50%
- `control-plane-init` 步骤连续失败 >= 3 次

### 回滚策略
1. 将 `BOOTSTRAP_INIT_MODE` 切回 `legacy`（保留旧 `kubeadm init` 参数路径）。
2. 暂时隐藏前端高级配置入口（profile/mirror/vip/external-etcd）。
3. 禁用 `bootstrap-prechecks` 与 `vip-provider` 步骤（通过配置开关或脚本短路）。
4. 保留任务记录中的 `resolved_config_json` 和 `diagnostics_json` 用于复盘。

### 复盘与恢复
- 汇总失败任务的 `validation_issues` 与 `diagnostics_json`
- 按失败域拆分修复项：repo、registry、endpoint、etcd、version-matrix
- 在测试环境复现后再恢复灰度流量
