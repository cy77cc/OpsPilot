# K8s 管理平台路线规划图（Roadmap Chart）

## 1. 总览路线图（Phase 1 -> Phase 3）

```mermaid
timeline
    title K8s 管理平台路线图（建议）
    Phase 1 (6-8周) : 集群详情页可操作闭环
                    : 节点/工作负载/Service-Ingress 基础操作
                    : 最小 RBAC + 审批 + 审计 + 任务状态机
    Phase 2 (8-12周) : 网络与可观测增强
                     : NetworkPolicy 可视化
                     : 指标/日志/链路统一排障入口
    Phase 3 (12周+) : 安全与交付平台化
                    : 准入与镜像扫描门禁
                    : 运行时安全与 GitOps/应用市场
```

## 2. 能力域推进节奏图（按阶段）

```mermaid
flowchart LR
    P1[Phase 1\n可操作闭环] --> P2[Phase 2\n深度治理]
    P2 --> P3[Phase 3\n平台化]

    subgraph D1[多集群与生命周期]
      D1A[P1: 节点操作/升级入口]
      D1B[P2: 节点池弹性与版本策略]
      D1C[P3: 备份恢复标准化]
      D1A --> D1B --> D1C
    end

    subgraph D2[资源与工作负载]
      D2A[P1: Deployment/StatefulSet/Pod 基础操作]
      D2B[P2: HPA 深化]
      D2C[P3: VPA 生产化]
      D2A --> D2B --> D2C
    end

    subgraph D3[网络与流量]
      D3A[P1: Service/Ingress 基础治理]
      D3B[P2: NetworkPolicy 可视化]
      D3C[P3: Mesh 灰度/限流/熔断]
      D3A --> D3B --> D3C
    end

    subgraph D4[可观测与AIOps]
      D4A[P1: 操作可追踪与审计可视]
      D4B[P2: 指标+日志+链路联动]
      D4C[P3: 智能诊断闭环]
      D4A --> D4B --> D4C
    end

    subgraph D5[安全与合规]
      D5A[P1: RBAC 最小闭环]
      D5B[P2: 审批策略细化]
      D5C[P3: 准入与运行时安全]
      D5A --> D5B --> D5C
    end

    subgraph D6[应用与交付]
      D6A[P1: 预留交付接口]
      D6B[P2: 交付链路接入]
      D6C[P3: Helm市场+GitOps标准化]
      D6A --> D6B --> D6C
    end
```

## 3. Phase 1（6-8周）执行甘特图

```mermaid
gantt
    title Phase 1 执行计划（6-8周）
    dateFormat  YYYY-MM-DD
    axisFormat  %m/%d

    section 基础能力（30%）
    统一任务状态机与操作协议     :a1, 2026-04-08, 10d
    RBAC与审批最小闭环           :a2, after a1, 8d
    审计链路与操作中心打通       :a3, 2026-04-15, 14d

    section 可交付能力（70%）
    节点操作闭环（cordon等）     :b1, 2026-04-08, 14d
    工作负载基础操作             :b2, 2026-04-22, 14d
    Service/Ingress基础操作      :b3, 2026-05-06, 12d

    section 验收与上线
    回归测试与稳定性修复         :c1, 2026-05-18, 10d
    灰度上线与验收               :c2, after c1, 4d
```

## 4. 里程碑与验收点

- M1（第2周）：节点操作能力可用，具备基础状态反馈。
- M2（第4周）：工作负载核心操作可用，审批与审计链路可跑通。
- M3（第6周）：Service/Ingress 基础能力可用，关键错误路径可恢复。
- M4（第8周）：完成回归、验收与上线。

## 5. 使用说明

- 该文档仅用于“路线规划图”可视化沟通。
- 需求边界、非目标、测试细节以主规格文档为准：
  - `docs/superpowers/specs/2026-04-04-k8s-platform-roadmap-design.md`

## 6. 当前执行状态（2026-04-05）

- Phase 1 任务进度：`8/8`（其中 Task 8 已完成收口记录与发布控制判断）
- 已闭环能力：
  - 节点操作（cordon/uncordon/drain/remove + 标签/污点）
  - 工作负载基础操作（重启/扩缩容/删除）
  - Service/Ingress 基础治理
  - 操作中心追踪与详情回链
  - 高风险失败恢复指引 + runbook
- 发布判定：
  - Phase 1 聚焦回归：通过
  - 仓库全量回归/构建：存在既有非 Phase 1 阻塞，需独立治理
- 对应验收记录：
  - `docs/superpowers/specs/2026-04-05-k8s-phase1-release-readiness.md`
