# CMDB 增强版详细设计 (CMDB Enhanced Detailed Design v2.0)

## 1. 背景与目标
本方案在 v1.0 基础上，引入身份对齐、属性权属治理和标准化接入契约，旨在构建一个高性能、可扩展、真实反映现状且具备治理能力的资产底座。

## 2. 核心概念与约束
- **CI (Configuration Item)**：受控的配置项实体。
- **Identity**：CI 的身份标识。一个 CI 可关联多个外部系统的 Identity。
- **Relationship**：有向边，具备来源、状态和置信度。
- **Attribute Ownership**：
    - `Discovered`: 自动发现驱动，实时覆盖。
    - `Managed`: 人工维护，发现系统不可覆盖。
    - `Computed`: 系统根据规则自动计算。

## 3. 数据模型设计

### 3.1 配置项表 (`cmdb_cis`)
增加高频过滤字段的结构化存储。
| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| `id` | SERIAL | 内部主键 |
| `ci_uid` | VARCHAR | 平台全局唯一标识 (Stable ID) |
| `ci_type` | VARCHAR | 资产类型 (host, cluster, service...) |
| `name` | VARCHAR | 显示名称 |
| `status` | VARCHAR | 运营状态 (active, stale, inactive, archived) |
| `env` | VARCHAR | 环境 (prod, staging, dev) |
| `region` | VARCHAR | 地域/机房 |
| `project_id` | UINT | 所属项目 (冗余索引，由关系派生) |
| `owner_id` | UINT | 负责人 ID |
| `source_main` | VARCHAR | 主来源名称 |
| `last_seen_at`| TIMESTAMP| 最后一次被发现的时间 |
| `attrs_json` | JSONB | 扩展属性 (非索引字段) |
| `attr_meta` | JSONB | 属性权属元数据 (标记哪些字段是 Managed) |

### 3.2 身份映射表 (`cmdb_ci_identities`)
解决多源发现同一对象的问题。
| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| `ci_id` | UINT | 关联 `cmdb_cis.id` |
| `source` | VARCHAR | 来源系统名称 (k8s, tencent_cloud, zabbix) |
| `ext_id` | VARCHAR | 外部系统中的唯一标识 |
| `confidence` | FLOAT | 匹配置信度 |

### 3.3 关系表 (`cmdb_relations`)
| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| `from_ci_id` | UINT | 源 CI |
| `to_ci_id` | UINT | 目标 CI |
| `rel_type` | VARCHAR | 关系类型 (depends_on, runs_on, part_of) |
| `source` | VARCHAR | 关系来源 (discovery, manual, rule) |
| `status` | VARCHAR | 关系状态 (active, broken) |
| `confidence` | FLOAT | 关系置信度 |
| `last_seen_at`| TIMESTAMP| 最后一次探测到关系的时间 |

## 4. 数据接入与同步流程 (Ingestion Pipeline)

### 4.1 标准接入契约 (Ingestion DTO)
所有发现插件必须上报统一格式：
```json
{
  "source": "k8s-cluster-01",
  "object_type": "node",
  "external_id": "i-bbcc123",
  "attributes": { "ip": "10.0.0.1", "cpu": 16 },
  "relations": [
    { "target_type": "cluster", "target_ext_id": "cls-xyz", "type": "part_of" }
  ]
}
```

### 4.2 同步处理链
1. **Fetch**: 从源模块拉取 DTO。
2. **Match**: 根据 `external_id` 或定义好的 `match_keys` 在 `cmdb_ci_identities` 中查找 CI。
3. **Merge**: 
    - 若匹配，更新 `last_seen_at` 和 `Discovered` 属性。
    - 若不匹配，创建新 CI 并建立 Identity 绑定。
4. **Relationship Sync**: 建立/更新关系，标记 `last_seen_at`。
5. **Sweep**: 对该 Source 下长时间未见的 CI/Relation，按生命周期策略执行 `Stale` 或 `Offline` 操作。

## 5. 生命周期管理 (LCM)
针对不同 `ci_type` 配置不同的老化策略：
- **Host**: 24h 未见 ➔ `Stale`, 7d 未见 ➔ `Inactive`。
- **Service Instance**: 15m 未见 ➔ `Stale` (跟随注册中心频率)。
- **Project**: 不自动失效。

## 6. 查询与拓扑 API 设计

### 6.1 树形导航 (`GET /api/cmdb/v1/tree`)
- 入参: `parent_id`, `view_type` (business/resource)。
- 场景: UI 左侧资产树、分层展开。

### 6.2 局部拓扑 (`GET /api/cmdb/v1/topology/subgraph`)
- 入参: `root_id`, `depth`, `direction`, `rel_types`。
- 场景: 影响分析、服务依赖图。

### 6.3 全局搜索 (`GET /api/cmdb/v1/search`)
- 入参: `keyword`, `ci_types`, `env`。
- 场景: 资产快速定位。

## 7. 审计与安全性
- **字段级审计**: 记录 `(field, old_value, new_value, source, operator)`。
- **变更快照**: 每次同步任务完成后，记录变更摘要。
