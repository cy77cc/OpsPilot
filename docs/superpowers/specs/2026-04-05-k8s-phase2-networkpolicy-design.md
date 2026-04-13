# K8s Phase 2 设计：多 CNI NetworkPolicy 可视化编排

## 1. 背景与目标

Phase 1 已完成集群详情页"可操作闭环"。Phase 2 聚焦网络与策略治理，目标是交付"多 CNI NetworkPolicy 可视化编排"能力，并达到"读写 + 仿真校验"的生产可用门槛。

本阶段明确约束：

- 流量治理方向：`Gateway API` 为主，`Ingress` 仅兼容过渡。
- 网络策略方向：`Cilium` 为主，兼容 `Calico` 与 `Flannel`。
- 发布策略：安全与效率均衡（双指标达标）。
- 架构策略：先中心化控制，预留后续集群侧执行器下沉接口。

## 2. 范围与非目标

### 2.1 本阶段范围

1. 多 CNI 策略可视化：命名空间、工作负载与策略关联拓扑。
2. 策略读写闭环：创建、编辑、发布、回滚、审计追踪。
3. 发布前仿真：冲突检测、影响面分析、风险评分。
4. CNI 适配层：Cilium/Calico/Flannel 的能力矩阵与翻译校验。
5. 与现有治理体系整合：RBAC、审批、审计、操作中心。

### 2.2 非目标

1. 不在本阶段交付 Service Mesh 深度治理（灰度/熔断/限流）。
2. 不在本阶段交付完整 L7 流量治理编排。
3. 不在本阶段交付跨云全自动策略迁移。

## 3. 总体架构

采用"统一策略模型 + 多插件适配层 + 中心化仿真引擎"的方案。

分层如下：

1. `Policy API`：对外提供策略与发布能力接口。
2. `Policy Core`：统一 DSL、规则归一化、能力标签判定。
3. `CNI Adapter`：Cilium/Calico/Flannel 翻译与兼容校验。
4. `Simulation Engine`：冲突检测、影响面与风险评估。
5. `Governance`：RBAC、审批、审计与操作链路追踪。
6. `UI`：策略可视化、编排、仿真结果对比、发布/回滚操作。

控制面演进：

- 当前：中心服务执行仿真与发布。
- 预留：后续引入集群侧执行器时复用统一执行接口，不破坏上层产品体验。

## 4. 组件设计与职责

### 4.1 policy-definition-service

- 负责统一策略 DSL 的草稿、版本、生效态管理。
- 提供模板能力，确保跨 CNI 策略描述一致。

#### 4.1.1 统一 DSL 设计

```yaml
apiVersion: opspilot.io/v1alpha1
kind: NetworkPolicyDefinition
metadata:
  name: <policy-name>
  namespace: <namespace>
spec:
  # 目标选择器（统一抽象）
  target:
    podSelector: { matchLabels: {...} }
    namespaceSelector: { matchLabels: {...} }
  
  # 策略类型
  policyTypes: ["Ingress", "Egress"]
  
  # 入站规则
  ingress:
  - name: <rule-name>
    action: Allow | Deny
    from:
      podSelector: { matchLabels: {...} }
      namespaceSelector: { matchLabels: {...} }
      ipBlock: { cidr: "...", except: [...] }
      serviceAccount: <name>  # Calico 特有
    ports:
    - protocol: TCP | UDP | SCTP
      port: <number> | <name>
      endPort: <number>
    # L7 规则（Cilium 特有）
    http:
    - method: GET | POST | PUT | DELETE
      path: <regex>
    dns:
    - matchPattern: <domain-pattern>
  
  # 出站规则
  egress:
  - name: <rule-name>
    action: Allow | Deny
    to:
      podSelector: { matchLabels: {...} }
      namespaceSelector: { matchLabels: {...} }
      ipBlock: { cidr: "...", except: [...] }
      fqdn: <domain-pattern>  # Cilium 特有
    ports:
    - protocol: TCP | UDP | SCTP
      port: <number> | <name>
  
  # 高级选项
  advanced:
    order: <number>           # Calico 优先级
    doNotTrack: boolean       # Calico 性能 hint
    applyOnForward: boolean   # Calico 转发流量
```

### 4.2 policy-simulation-service

- 对比 `base_version` 与 `candidate_version`。
- 输出阻断项、告警项、影响面摘要与风险等级。

#### 4.2.1 仿真引擎算法

**冲突检测规则**：

| 冲突类型 | 检测条件 | 严重级别 |
|---------|---------|---------|
| 规则覆盖冲突 | 同一 Pod 被多个策略选中且规则互斥 | HIGH |
| 优先级冲突 | Calico 多策略 `order` 值相同 | MEDIUM |
| 语义降级 | DSL 特性在目标 CNI 不支持 | BLOCKING |
| 关键 IP 误阻断 | `ipBlock` 包含 Kubernetes API/节点网段 | BLOCKING |

**影响面分析**：

```
影响面 = 受影响 Pod 数量 × 规则变更密度

受影响 Pod 数量 = kubectl get pods --selector=<podSelector> | wc -l
规则变更密度 = (新增规则数 + 删除规则数 + 修改规则数) / 总规则数
```

**风险评分算法**：

```go
risk_score = 0

// 关键命名空间阻断 (40 分)
if blocks_critical_namespace {
    risk_score += 40
}

// L7 规则复杂度 (20 分)
if has_l7_rules {
    risk_score += min(http_rules_count * 2, 20)
}

// CNI 能力缺口 (30 分)
if cni_capability_gap {
    risk_score += 30
}

// 无回滚保障 (10 分)
if !rollback_available {
    risk_score += 10
}

// 风险等级判定
if risk_score >= 70 {
    return "CRITICAL"
} else if risk_score >= 40 {
    return "HIGH"
} else if risk_score >= 20 {
    return "MEDIUM"
} else {
    return "LOW"
}
```

### 4.3 policy-adapter-service

#### 4.3.1 Cilium Adapter

**CNI 能力**：
- API Group: `cilium.io/v2`
- 资源类型：`CiliumNetworkPolicy`, `CiliumClusterwideNetworkPolicy`
- 核心字段：`endpointSelector`, `toPorts`, `toFQDNs`, `toEndpoints`
- L7 支持：HTTP (method/path), DNS (matchPattern/matchName), TLS 感知

**翻译映射表**：

| 统一 DSL | Cilium 字段 | 备注 |
|---------|-----------|------|
| `target.podSelector` | `spec.endpointSelector.matchLabels` | 直接映射 |
| `ingress.from.podSelector` | `spec.ingress.fromEndpoints.matchLabels` | 直接映射 |
| `ingress.ports` | `spec.ingress.toPorts.ports` | 直接映射 |
| `ingress.http` | `spec.ingress.toPorts.rules.http` | L7 HTTP 规则 |
| `egress.fqdn` | `spec.egress.toFQDNs` | Cilium 特有 |
| `egress.dns` | `spec.egress.toEndpoints + toPorts.rules.dns` | DNS 规则 |

**阻断条件**：
- `serviceAccountSelector` 使用（Cilium 不支持）
- `order` 优先级字段（Cilium 不支持）

#### 4.3.2 Calico Adapter

**CNI 能力**：
- API Group: `crd.projectcalico.org/v1`
- 资源类型：`NetworkPolicy`, `GlobalNetworkPolicy`
- 核心字段：`selector`, `serviceAccountSelector`, `order`, `types`
- 特殊能力：策略优先级、Log Action、DoNotTrack

**翻译映射表**：

| 统一 DSL | Calico 字段 | 备注 |
|---------|-----------|------|
| `target.podSelector` | `spec.selector` | 转换为 Calico 选择器语法 |
| `target.serviceAccount` | `spec.serviceAccountSelector` | 直接映射 |
| `policyTypes` | `spec.types` | 直接映射 |
| `advanced.order` | `spec.order` | 直接映射 |
| `advanced.doNotTrack` | `spec.doNotTrack` | 直接映射 |
| `ingress.action` | `spec.ingress[].action` | Allow/Deny |

**Calico 选择器语法转换**：

```go
// K8s LabelSelector -> Calico Selector 表达式
// { matchLabels: { "app": "web", "env": "prod" } }
// => "app == 'web' && env == 'prod'"

func toCalicoSelector(matchLabels map[string]string) string {
    var exprs []string
    for k, v := range matchLabels {
        exprs = append(exprs, fmt.Sprintf("%s == '%s'", k, v))
    }
    return strings.Join(exprs, " && ")
}
```

**阻断条件**：
- `fqdn` 字段（Calico 不支持 FQDN 策略）
- L7 HTTP 规则（Calico 仅支持 L3/L4）

#### 4.3.3 Flannel Adapter

**CNI 能力**：
- API Group: `networking.k8s.io/v1`（标准 K8s NetworkPolicy）
- 核心限制：仅支持 L3/L4，不支持 L7
- 前置条件：需启用 `netpol.enabled=true`

**错误码定义**：

| 错误码 | HTTP 状态 | 说明 | 建议操作 |
|-------|----------|------|---------|
| `FLANNEL_NETPOL_DISABLED` | 400 | Flannel 网络策略控制器未启用 | `helm upgrade flannel --set netpol.enabled=true` |
| `FLANNEL_L7_NOT_SUPPORTED` | 400 | Flannel 不支持 L7 策略 | 升级到 Cilium 或移除 L7 规则 |
| `FLANNEL_ONLY_STANDARD_NP` | 200 | 仅支持标准 K8s NetworkPolicy | 使用基础 L3/L4 规则 |

### 4.4 policy-release-service

- 管理发布流水：仿真 -> 审批 -> 下发 -> 回滚。
- 统一结果状态并关联审计记录与操作中心链接。

#### 4.4.1 发布状态机

```
┌─────────────┐
│   draft     │
└──────┬──────┘
       │ 用户保存草稿
       ▼
┌─────────────────┐
│ simulation_pending│
└──────┬──────────┘
       │ 执行仿真
       ├─── 失败 ───┐
       ▼            ▼
┌─────────────────┐  ┌───────────────────┐
│ simulation_passed│  │ simulation_failed │
└──────┬──────────┘  └───────────────────┘
       │ 发起发布申请
       ▼
┌─────────────────┐
│ approval_required│
└──────┬──────────┘
       │ 审批决策
       ├─── 拒绝 ───┐
       ▼            ▼
┌─────────────────┐  ┌───────────────────┐
│    applying     │  │ approval_rejected │
└──────┬──────────┘  └───────────────────┘
       │ 下发执行
       ├─── 失败 ───┐
       ▼            ▼
┌─────────────────┐  ┌───────────────────┐
│    applied      │  │    apply_failed   │
└──────┬──────────┘  └─────────┬─────────┘
       │                       │ 回滚
       ▼                       ▼
┌─────────────────┐  ┌───────────────────┐
│   active        │  │  rollback_applied │
└─────────────────┘  └───────────────────┘
```

#### 4.4.2 发布记录结构

```yaml
apiVersion: opspilot.io/v1alpha1
kind: PolicyRelease
metadata:
  name: <release-id>
spec:
  policyRef:
    apiVersion: opspilot.io/v1alpha1
    kind: NetworkPolicyDefinition
    name: <policy-name>
    namespace: <namespace>
  
  version: <semantic-version>
  previousStableVersion: <previous-version>
  
  targetCluster:
    clusterId: <cluster-id>
    cniType: cilium | calico | flannel
    cniVersion: <version>
  
  status:
    phase: draft | simulation_passed | applying | applied | rollback_applied
    riskScore: <0-100>
    riskLevel: LOW | MEDIUM | HIGH | CRITICAL
    
  simulation:
    jobId: <simulation-job-id>
    passedAt: <timestamp>
    blockingIssues: [...]
    warnings: [...]
    impactSummary:
      affectedPods: <count>
      affectedNamespaces: [...]
  
  approval:
    required: boolean
    approvers: [...]
    approvedAt: <timestamp>
    approvalToken: <token>
  
  audit:
    createdAt: <timestamp>
    createdBy: <user-id>
    appliedAt: <timestamp>
    rollbackAt: <timestamp>
```

### 4.5 policy-visualization-ui

#### 4.5.1 核心视图

1. **策略拓扑视图**
   - 命名空间维度：展示命名空间间的流量允许/拒绝关系
   - 工作负载维度：展示 Pod/Deployment 级别的策略覆盖
   - 策略维度：展示单一策略影响的范围

2. **策略编辑器**
   - 可视化规则构建器（拖拽式）
   - YAML 双向同步编辑
   - 实时语法校验与 CNI 兼容性提示

3. **仿真 Diff 视图**
   - 变更前后规则对比（高亮新增/删除/修改）
   - 影响面热力图
   - 风险评分可视化

4. **发布操作视图**
   - 审批流程状态
   - 发布历史时间线
   - 一键回滚入口

#### 4.5.2 Gateway API 关联展示

```
┌─────────────────────────────────────────────────────────┐
│                    Gateway API 视图                      │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  GatewayClass: nginx                                    │
│  └── Gateway: prod-gateway                                │
│      └── HTTPRoute: app-route                           │
│          ├── matches: [{ path: /api/* }]                │
│          ├── backendRefs: [{ name: app-svc, port: 80 }] │
│          └── 关联 NetworkPolicy: app-ingress-policy     │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

### 4.6 policy-observability-bridge

- 汇聚策略命中、拒绝流量摘要与关键告警。
- 支持按 `release_id` 追踪。

#### 4.6.1 可观测性指标

| 指标名称 | 类型 | 说明 |
|---------|------|------|
| `policy_hit_total` | Counter | 策略命中次数（按策略名、动作、方向） |
| `policy_deny_total` | Counter | 策略拒绝次数（用于发现误配置） |
| `policy_release_duration_seconds` | Histogram | 发布流水耗时 |
| `simulation_evaluation_duration_seconds` | Histogram | 仿真评估耗时 |
| `cni_adapter_translation_errors` | Counter | 翻译错误次数 |

## 5. 关键数据流与状态机

### 5.1 策略变更主流程

```
┌──────────────────────────────────────────────────────────────────┐
│                        策略变更主流程                              │
└──────────────────────────────────────────────────────────────────┘

1. 用户编辑策略草稿
   └── POST /api/v1/policies/{name}/draft
   
2. 执行仿真（强制门禁）
   └── POST /api/v1/policies/{name}/simulate
       ├── 输入：base_version, candidate_version, cluster.cni_type
       └── 输出：blocking_issues, warnings, impact_summary, risk_score
       
3. 仿真通过后发起发布申请
   └── POST /api/v1/releases
       ├── 前置检查：risk_score < 70 (CRITICAL 阈值)
       └── 创建 release_id，记录 previous_stable_version
       
4. 审批通过后执行下发
   └── POST /api/v1/releases/{release_id}/apply
       ├── 审批令牌校验
       └── 调用 CNI Adapter 下发策略
       
5. 记录审计并回链操作中心
   └── 审计事件写入 + 操作中心状态更新
   
6. 成功生效或失败回滚
   └── 失败时自动触发回滚至 previous_stable_version
```

### 5.2 仿真流程

**输入**：
- `base_version`: 当前生效的策略版本
- `candidate_version`: 待发布的策略草稿
- `cluster.cni_type`: 目标集群 CNI 类型
- `cluster.namespaces`: 命名空间列表（用于影响面分析）

**输出**：
```yaml
simulation_result:
  passed: boolean
  blocking_issues:
  - code: string
    message: string
    severity: BLOCKING | HIGH | MEDIUM | LOW
    suggestion: string
  warnings:
  - code: string
    message: string
  impact_summary:
    affected_pods: number
    affected_namespaces: [string]
    new_denied_flows: [string]
  risk_score: number  # 0-100
  risk_level: LOW | MEDIUM | HIGH | CRITICAL
```

### 5.3 发布状态机详细转换条件

| 当前状态 | 触发动作 | 目标状态 | 条件 |
|---------|---------|---------|------|
| `draft` | 保存草稿 | `draft` | - |
| `draft` | 执行仿真 | `simulation_pending` | - |
| `simulation_pending` | 仿真完成 | `simulation_passed` | 无 blocking_issues |
| `simulation_pending` | 仿真失败 | `simulation_failed` | 存在 blocking_issues |
| `simulation_passed` | 发起发布 | `approval_required` | risk_score < 70 |
| `approval_required` | 审批通过 | `applying` | 审批令牌有效 |
| `approval_required` | 审批拒绝 | `approval_rejected` | - |
| `applying` | 下发成功 | `applied` | CNI 下发校验通过 |
| `applying` | 下发失败 | `apply_failed` | - |
| `apply_failed` | 回滚 | `rollback_applied` | previous_stable_version 存在 |
| `applied` | 回滚 | `rollback_applied` | - |

## 6. 跨 CNI 支持策略

### 6.1 CNI 能力矩阵

| 能力 | K8s Native | Cilium | Calico | Flannel |
|------|------------|--------|--------|---------|
| L3 策略 (podSelector) | ✅ | ✅ | ✅ | ✅ |
| L4 策略 (ports) | ✅ | ✅ | ✅ | ✅ |
| L7 HTTP 策略 | ❌ | ✅ | ❌ | ❌ |
| L7 DNS 策略 | ❌ | ✅ | ❌ | ❌ |
| FQDN 策略 | ❌ | ✅ | ❌ | ❌ |
| 策略优先级 (order) | ❌ | ❌ | ✅ | ❌ |
| ServiceAccount 选择 | ❌ | ❌ | ✅ | ❌ |
| 命名空间策略 | ✅ | ✅ | ✅ | ✅ |
| Global 级别策略 | ❌ | ✅ (CCNP) | ✅ (GNP) | ❌ |
| Log 动作 | ❌ | ✅ | ✅ | ❌ |

### 6.2 CNI 分级验收标准

#### 6.2.1 Cilium（主验收线）

- [ ] 读写 + 仿真全通过
- [ ] L7 HTTP/DNS 规则翻译正确
- [ ] FQDN 策略支持
- [ ] `CiliumNetworkPolicy` 和 `CiliumClusterwideNetworkPolicy` 双资源支持

#### 6.2.2 Calico

- [ ] 读写 + 仿真通过
- [ ] `order` 优先级字段支持
- [ ] `serviceAccountSelector` 支持
- [ ] `GlobalNetworkPolicy` 支持
- [ ] 选择器语法转换正确

#### 6.2.3 Flannel

- [ ] 只读可视化通过
- [ ] `netpol.enabled=false` 时发布阻断通过
- [ ] 错误提示文案清晰且包含修复建议
- [ ] 标准 K8s NetworkPolicy 兼容通过

## 7. 错误处理与风险控制

### 7.1 阻断级错误（必须停止发布）

| 错误码 | 说明 | 建议操作 |
|-------|------|---------|
| `SIMULATION_BLOCKING_CONFLICT` | 高风险冲突将导致关键命名空间流量误阻断 | 调整策略规则或影响范围 |
| `CNI_SEMANTIC_GAP` | 语义转译后高风险不等价 | 移除不兼容特性或更换 CNI |
| `APPROVAL_TOKEN_INVALID` | 审批未通过或审批令牌无效 | 重新发起审批流程 |
| `APPLY_VALIDATION_FAILED` | 下发校验失败 | 检查集群连接与 CNI 状态 |
| `FLANNEL_NETPOL_DISABLED` | Flannel 网络策略控制器未启用 | 启用 netpol 控制器 |
| `CRITICAL_NAMESPACE_BLOCKED` | 策略误阻断 kube-system/default 命名空间 | 添加例外规则 |

### 7.2 告警级问题（允许继续但需确认）

| 告警码 | 说明 | 确认要求 |
|-------|------|---------|
| `CNI_CAPABILITY_DOWNGRADE` | CNI 能力降级 | 二次确认弹窗 |
| `IMPACT_SCOPE_EXPANDED` | 影响面扩大但在白名单范围内 | 展示影响 Pod 列表 |
| `NON_CRITICAL_NAMESPACE_CHANGE` | 非关键命名空间覆盖变化 | 记录审计日志 |
| `L7_RULE_SIMPLIFIED` | L7 规则在目标 CNI 被简化 | 展示简化前后对比 |

### 7.3 回滚机制

1. 每次发布生成 `release_id` 与 `previous_stable_version`。
2. 发布失败或健康检查异常支持一键回滚。
3. 回滚全链路必须审计可追溯。

**回滚 API**：
```
POST /api/v1/releases/{release_id}/rollback
Content-Type: application/json

{
  "reason": "<回滚原因>",
  "targetVersion": "<previous_stable_version>"  // 可选，默认为上一个稳定版本
}
```

### 7.4 护栏

1. 禁止绕过仿真直接发布。
2. 高风险命名空间（kube-system, default）启用二次审批。
3. Flannel 能力缺口按配置执行强阻断。
4. CRITICAL 风险等级（>=70 分）禁止发布。

## 8. 测试与验收标准

### 8.1 测试分层

#### 8.1.1 单元测试

| 测试模块 | 覆盖内容 | 目标覆盖率 |
|---------|---------|-----------|
| DSL 解析器 | YAML 解析、字段校验、默认值填充 | 95%+ |
| 翻译引擎 | CNI 字段映射、选择器语法转换 | 90%+ |
| 冲突检测 | 规则覆盖冲突、优先级冲突 | 90%+ |
| 风险评分 | 评分规则、阈值判定 | 95%+ |

#### 8.1.2 集成测试

| 测试场景 | CNI 类型 | 验证点 |
|---------|---------|-------|
| 策略创建 | Cilium | CNPR 资源创建成功 |
| 策略创建 | Calico | NetworkPolicy 资源创建成功 |
| 策略创建 | Flannel | K8s NetworkPolicy 创建成功 |
| 策略更新 | 全部 | 版本递增、历史记录 |
| 策略删除 | 全部 | 资源清理、审计记录 |
| 发布流水 | 全部 | 状态机流转正确 |

#### 8.1.3 端到端测试

1. **策略编辑到操作中心追踪闭环**
   - 创建策略草稿
   - 执行仿真
   - 发起审批
   - 审批通过
   - 策略下发
   - 操作中心查看审计记录

2. **回滚场景**
   - 发布失败自动回滚
   - 手动回滚至指定版本

#### 8.1.4 兼容测试

- Gateway API 资源识别与关联展示
- Ingress 资源向后兼容

### 8.2 验收门槛（均衡）

#### 8.2.1 安全正确性

- [ ] 高风险冲突 0 漏拦截
- [ ] 跨 CNI 语义降级必须显式提示
- [ ] CRITICAL 风险等级发布阻断率 100%
- [ ] 回滚成功率 99%+

#### 8.2.2 交付效率

| 指标 | 目标值 | 说明 |
|------|-------|------|
| 仿真评估时延 | < 2s | 100 条规则内 |
| 发布流水时延 | < 10s | 审批通过后下发 |
| 策略生效时延 | < 30s | 下发到 CNI 生效 |

#### 8.2.3 闭环能力

- [ ] 每次发布均有 `release_id`、审计记录、可回滚版本
- [ ] 操作中心可按策略/集群/操作者/时间追踪
- [ ] 回滚操作 100% 可追溯

### 8.3 语义等价性测试

**测试用例**：

| DSL 特性 | Cilium 预期 | Calico 预期 | Flannel 预期 |
|---------|-----------|-----------|------------|
| podSelector | endpointSelector 等价 | selector 表达式等价 | podSelector 等价 |
| L4 端口规则 | toPorts 等价 | ports 等价 | ports 等价 |
| L7 HTTP 规则 | 原生支持 | 阻断提示 | 阻断提示 |
| ipBlock | 不支持（需转换） | 不支持（需转换） | 原生支持 |

## 9. 里程碑建议（Phase 2 内）

| 里程碑 | 交付内容 | 验收标准 | 预计周期 |
|-------|---------|---------|---------|
| **M1** | 统一 DSL + Cilium 适配 + 基础仿真引擎 | Cilium 读写 + 仿真全通过 | 3 周 |
| **M2** | Calico 适配 + 发布流水与审批审计打通 | Calico 读写 + 仿真通过 | 2 周 |
| **M3** | Flannel 能力缺口阻断 + 可视化与操作中心联动 | Flannel 阻断逻辑正确 | 1 周 |
| **M4** | 回归测试、验收、灰度上线准备 | 全部验收标准通过 | 1 周 |

## 10. 关键参考（官方文档）

### 10.1 Kubernetes 核心

1. **Kubernetes NetworkPolicy**
   - https://kubernetes.io/docs/concepts/services-networking/network-policies/
   - API: `networking.k8s.io/v1`
   - 字段：`podSelector`, `policyTypes`, `ingress`, `egress`

2. **Gateway API**
   - https://gateway-api.sigs.k8s.io/
   - 资源：`Gateway`, `HTTPRoute`, `GRPCRoute`
   - 字段：`backendRefs`, `matches`, `filters`

### 10.2 CNI 插件

3. **Cilium NetworkPolicy**
   - https://docs.cilium.io/en/stable/network/kubernetes/policy.html
   - API: `cilium.io/v2`
   - 资源：`CiliumNetworkPolicy`, `CiliumClusterwideNetworkPolicy`
   - 特有字段：`endpointSelector`, `toFQDNs`, `toPorts.rules.http`, `toPorts.rules.dns`

4. **Calico NetworkPolicy**
   - https://docs.tigera.io/calico/latest/reference/resources/networkpolicy
   - https://docs.tigera.io/calico/latest/reference/resources/globalnetworkpolicy
   - API: `crd.projectcalico.org/v1`
   - 资源：`NetworkPolicy`, `GlobalNetworkPolicy`
   - 特有字段：`selector`, `serviceAccountSelector`, `order`, `doNotTrack`, `applyOnForward`

5. **Flannel 与策略引擎**
   - https://github.com/flannel-io/flannel/blob/master/Documentation/netpol.md
   - 限制：仅支持标准 K8s NetworkPolicy
   - 前置条件：`helm install flannel --set netpol.enabled=true`

### 10.3 错误码索引

| 错误码前缀 | 来源模块 |
|-----------|---------|
| `SIMULATION_` | policy-simulation-service |
| `CNI_` | policy-adapter-service |
| `APPROVAL_` | policy-release-service |
| `APPLY_` | policy-release-service |
| `FLANNEL_` | policy-adapter-service (Flannel Adapter) |

---

## 附录 A：DSL 字段与 CNI 映射完整对照表

```yaml
# 统一 DSL -> Cilium 映射
target.podSelector           -> spec.endpointSelector.matchLabels
ingress.from.podSelector     -> spec.ingress.fromEndpoints.matchLabels
ingress.ports                -> spec.ingress.toPorts.ports
ingress.http.method          -> spec.ingress.toPorts.rules.http.method
ingress.http.path            -> spec.ingress.toPorts.rules.http.path
egress.fqdn                  -> spec.egress.toFQDNs
egress.dns.matchPattern      -> spec.egress.toEndpoints + toPorts.rules.dns

# 统一 DSL -> Calico 映射
target.podSelector           -> spec.selector (转换为表达式语法)
target.serviceAccount        -> spec.serviceAccountSelector
policyTypes                  -> spec.types
ingress.action               -> spec.ingress[].action (Allow/Deny)
advanced.order               -> spec.order
advanced.doNotTrack          -> spec.doNotTrack

# 统一 DSL -> Flannel/K8s Native 映射
target.podSelector           -> spec.podSelector
policyTypes                  -> spec.policyTypes
ingress/from/podSelector     -> spec.ingress[].from.podSelector
ingress.ports                -> spec.ingress[].ports
egress/to/podSelector        -> spec.egress[].to.podSelector
egress.ports                 -> spec.egress[].ports
```

## 附录 B：CNI 选择器语法对照

| K8s LabelSelector | Calico Selector | 说明 |
|------------------|-----------------|------|
| `{matchLabels: {"app": "web"}}` | `app == 'web'` | 等值匹配 |
| `{matchExpressions: [{"key": "env", "operator": "In", "values": ["prod", "staging"]}]}` | `env in {'prod', 'staging'}` | 集合匹配 |
| `{matchExpressions: [{"key": "tier", "operator": "Exists"}]}` | `has(tier)` | 存在性检查 |
| `{matchExpressions: [{"key": "debug", "operator": "DoesNotExist"}]}` | `!has(debug)` | 不存在检查 |

## 附录 C：API 端点索引

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/v1/policies` | GET | 策略列表 |
| `/api/v1/policies/{name}` | GET | 策略详情 |
| `/api/v1/policies/{name}` | PUT | 更新策略 |
| `/api/v1/policies/{name}/draft` | POST | 创建草稿 |
| `/api/v1/policies/{name}/simulate` | POST | 执行仿真 |
| `/api/v1/releases` | POST | 创建发布 |
| `/api/v1/releases/{release_id}` | GET | 发布详情 |
| `/api/v1/releases/{release_id}/apply` | POST | 执行发布 |
| `/api/v1/releases/{release_id}/rollback` | POST | 回滚发布 |
| `/api/v1/clusters/{cluster_id}/cni-info` | GET | 获取 CNI 能力信息 |

## 11. Phase 2 验收记录（2026-04-05）

### 11.1 聚焦回归结果

- 后端聚焦回归：通过  
  - `GOCACHE=/tmp/go-build-cache go test ./internal/service/cluster/... ./internal/service/governance/...`
- 前端聚焦回归：通过  
  - `cd web && npx vitest run src/pages/Deployment/Infrastructure/ClusterPolicyCenterPage.test.tsx src/pages/Deployment/Infrastructure/ClusterOperationCenterPage.test.tsx src/api/modules/cluster.policy.test.ts --testTimeout=60000`
- E2E lite 流程回归：通过  
  - `cd web && npx vitest run src/e2e/policy-release-flow.test.ts --testTimeout=120000`

### 11.2 全量门禁结果（非本阶段阻塞记录）

- `go test ./...`：未通过（存在既有非 Phase 2 阻塞）
  - `internal/dao/ai`：缺失 migration fixture（`storage/migrations/20260320_0003_add_ai_failed_session_persistence.sql`、`20260321_0004_fix_ai_run_contents_utf8mb4.sql`）
  - `internal/modules/ai/handler`：多条 SSE 回放相关用例失败（事件重放顺序断言不满足）
  - `storage/migration`：缺失 migration fixture（`20260317_0003_create_ai_approval_tasks.sql`）
- `cd web && npm run test`：未通过（存在既有前端测试失败与环境错误）
  - Notification、Deployment、Cluster 相关若干历史测试超时/断言不匹配
  - 运行期出现 `window is not defined`、`MutationObserver is not a constructor` 的测试环境问题
- `cd web && npm run build`：未通过（存在既有类型定义缺失）
  - `src/data/mockData.ts` 引用了 `../types` 中不存在的导出（`ConfigApp`、`ConfigItem`、`ConfigTemplate`、`Release`、`AuditLog`）

### 11.3 本阶段发布结论

- Phase 2 网络策略治理链路（NetworkPolicy + Gateway API 过渡方向 + Flannel 兼容约束）已具备“聚焦验收可发布”条件。
- 仓库全量门禁仍有历史阻塞项，需独立修复后再执行全仓统一放行。
