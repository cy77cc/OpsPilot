## Why

当前集群创建流程将 Kubernetes 版本和部分安装路径写死为 `1.28.x`，且缺少对弱离线（内网仓库）和高可用入口（VIP/LB）的端到端配置能力，导致生产场景落地受限。现在需要将创建流程从“固定脚本”升级为“参数化、可演进”的 bootstrap 能力，以支撑真实企业环境。

## What Changes

- 将集群创建的 Kubernetes 版本来源改为动态版本目录（支持官方通道/固定版本），替代前后端写死版本。
- 扩展创建向导与后端 bootstrap 请求模型，新增控制平面入口配置（`nodeIP`/`vip`/`lbDNS`）与 `controlPlaneEndpoint` 参数。
- 引入弱离线安装模式（`online`/`mirror`），支持内网包源与可配置镜像仓库（含 `registry.aliyuncs.com/google_containers` 与自定义仓库）。
- 为 kubeadm 初始化引入配置文件化能力（如 `imageRepository`、`controlPlaneEndpoint`），减少硬编码命令参数。
- 增加 VIP 自动化编排能力，优先支持 `kube-vip`，并预留 `keepalived` provider 路径。
- 增加 external etcd 配置输入与校验能力，使 bootstrap 可切换 `stacked`/`external` etcd 模式。
- 强化 bootstrap 预检与诊断：对仓库可达性、镜像可用性、VIP 参数、etcd 参数给出结构化错误与修复提示。

## Capabilities

### New Capabilities
- `cluster-bootstrap-profile-management`: 定义并管理集群创建 profile（版本通道、仓库模式、VIP provider、etcd 模式）及跨字段校验策略。

### Modified Capabilities
- `deployment-infrastructure-management`: 扩展集群创建向导、bootstrap API 与执行流程，支持动态版本、VIP/LB 入口、镜像仓库与 external etcd。
- `environment-runtime-bootstrap`: 扩展运行时安装流程的弱离线模式（mirror repo）与安装前预检/诊断能力。

## Impact

- 后端：`internal/service/cluster/*` bootstrap 参数模型、执行编排与任务诊断逻辑需改造；可能新增 provider/validator 组件。
- 脚本与物料：`script/cluster/kubeadm/*` 需支持配置文件驱动、镜像仓库注入、VIP 自动化步骤与弱离线仓库策略。
- 前端：`web/src/pages/Deployment/Infrastructure/ClusterBootstrapWizard.tsx` 与 `web/src/api/modules/cluster.ts` 需新增高级配置项与校验提示。
- 数据与契约：bootstrap task 记录需扩展关键配置字段，便于审计与复盘。
- 运维依赖：需要内网仓库/镜像仓库可用性保障，以及 VIP 网络规划前置条件。
