## Context

当前 `ClusterBootstrapWizard` 与 `internal/service/cluster/logic_bootstrap.go` 对 Kubernetes 版本和脚本路径存在强绑定（`v1.28`），无法随上游版本节奏演进。现有创建流程缺少对生产常见需求的统一建模：
- 弱离线安装（内网 apt/yum 源 + 内网镜像仓库）
- 高可用控制平面入口（VIP/LB）
- external etcd 场景

已有实现基础：
- 前后端已具备 bootstrap 任务与步骤跟踪框架
- `init.sh` 已支持 `CONTROL_PLANE_ENDPOINT` 参数
- 平台已有脚本物料目录和任务诊断载体，可承载参数化扩展

约束：
- 保持 `/api/v1/clusters/bootstrap/*` 兼容，已有请求不应因新字段失败
- 不在首阶段引入“强离线完全断网”能力，只覆盖 mirror 模式
- 需要在创建流程中提供可解释的校验和失败诊断

## Goals / Non-Goals

**Goals:**
- 将集群创建从写死版本切换为“动态版本目录 + 可固定版本”的参数化模型。
- 支持弱离线模式：配置内网包源与 `imageRepository`，避免默认公网依赖。
- 支持控制平面入口配置（`nodeIP`/`vip`/`lbDNS`）并自动化 VIP provider（优先 kube-vip，兼容 keepalived）。
- 支持 `stacked` 与 `external` etcd 模式选择，并在请求阶段完成关键字段校验。
- 保持 bootstrap 任务可观测：关键参数、预检结果、失败原因可追踪。

**Non-Goals:**
- 不在本变更内交付“强离线（完全无网络）物料分发体系”。
- 不在本变更内覆盖所有 Kubernetes 小版本脚本自动生成，只实现受支持版本目录。
- 不在首批范围内交付多控制平面全自动扩缩容编排。

## Decisions

### Decision 1: 版本来源采用“服务端动态目录 + 前端选择”
- 方案：新增后端版本目录接口，聚合官方通道（如 stable/stable-1）并允许固定版本回填；前端下拉不再写死。
- 选择理由：避免前端发布节奏绑定版本更新，统一策略与缓存控制。
- 备选方案：仅前端拉官方接口。未选原因：跨域、缓存、稳定性与治理不可控。

### Decision 2: kubeadm 初始化切换到配置文件驱动
- 方案：从“命令参数拼接”转为生成 kubeadm config（包含 `controlPlaneEndpoint`、`imageRepository`、`etcd` 配置）后执行 init。
- 选择理由：对 VIP、external etcd、镜像仓库等复杂输入更稳定，降低脚本分支爆炸。
- 备选方案：继续追加命令行参数。未选原因：可维护性差，参数组合校验复杂。

### Decision 3: 安装源分层为 `online|mirror`
- 方案：`packages.repo_mode` 定义安装源模式，`mirror` 强制要求内网源配置；`images.image_repository` 与包源策略解耦。
- 选择理由：弱离线场景下包源与镜像源常常独立治理，需要分别可控。
- 备选方案：单一“离线开关”。未选原因：表达力不足，无法覆盖实际企业环境差异。

### Decision 4: VIP provider 抽象，默认 kube-vip
- 方案：定义 `vip.provider` 抽象层，首选 `kube-vip`，同时预留 `keepalived` 分支。
- 选择理由：kube-vip 与 kubeadm 路径一致性更高；keepalived 便于兼容传统网络团队。
- 备选方案：只做一种 provider。未选原因：适配性不足，无法覆盖多类基础设施。

### Decision 5: etcd 模式显式建模并强校验
- 方案：新增 `etcd.mode=stacked|external`；external 必须带 endpoints 与证书组。
- 选择理由：避免“隐式切换”导致初始化后期失败，尽早在请求/预检阶段暴露错误。
- 备选方案：后续迭代再支持 external。未选原因：用户明确需要，且与 kubeadm config 化强耦合。

## Risks / Trade-offs

- [Risk] 版本目录与脚本支持矩阵不一致（可选版本多于可执行脚本）
  → Mitigation：版本目录返回“可选但不可引导”的标记；提交时做硬校验并给出可用版本建议。

- [Risk] mirror 源可达但包不完整，安装在中途失败
  → Mitigation：预检增加“关键包解析检查”（kubeadm/kubelet/kubectl/containerd）。

- [Risk] VIP 已配置但网络/端口转发未闭环，API 不可用
  → Mitigation：VIP provider 完成后执行端到端探测（`controlPlaneEndpoint:6443`）并回写诊断。

- [Risk] external etcd 参数合法但证书链不匹配
  → Mitigation：在预检阶段尝试 etcd TLS 握手并输出具体证书错误。

- [Trade-off] 引入配置文件生成器提升可维护性，但初期实现复杂度上升
  → Mitigation：先收敛最小字段集，按 Phase 增量扩展。

## Migration Plan

1. API 兼容扩展：为 bootstrap 请求增加新字段，旧字段保持可用并映射到默认配置。  
2. 后端实现双轨：保留旧执行路径开关，新路径使用 kubeadm config 生成器。  
3. 前端灰度：高级配置项默认折叠，仅在需要时展开。  
4. 环境灰度：先在内网测试环境验证 `mirror + kube-vip` 主路径，再逐步推广。  
5. 回滚策略：出现高故障率时切回旧路径开关，禁用新字段入口并保留任务日志用于复盘。  

## Open Questions

- 版本目录接口是否仅暴露“平台验证通过版本”，还是允许“官方最新 + 平台支持标记”并存？
- keepalived provider 是否首批交付，还是在 kube-vip 稳定后再启用？
- mirror 模式下是否强制要求 `imageRepository`，还是允许警告后继续？
- external etcd 证书素材的存储与脱敏策略是否复用现有 credential 管理域？
