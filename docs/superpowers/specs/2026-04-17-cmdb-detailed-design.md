# CMDB 模块详细设计 (CMDB Detailed Design)

## 1. 背景与目标
OpsPilot 的 CMDB（配置管理数据库）旨在提供一个“反映现状”的资产治理中心。通过自动发现各模块资源并建立关联，实现业务资产的透明化管理和可视化拓扑展示。

## 2. 核心架构设计

### 2.1 数据存储策略
- **数据库**：PostgreSQL
- **核心模型 (`cmdb_cis`)**：
    - `id`: 自增主键
    - `ci_uid`: 唯一标识 (格式: `ciType:externalID`)
    - `ci_type`: 类型 (host, cluster, service, project, team 等)
    - `name`: 名称
    - `status`: 状态 (active, stale, inactive, archived)
    - `project_id`, `team_id`: 归属关系（扁平化索引）
    - `attrs_json`: JSONB 类型，存储不同 CI 类型的扩展属性。
- **关系模型 (`cmdb_relations`)**：
    - `from_ci_id`, `to_ci_id`: 关联 CI
    - `relation_type`: 关系类型 (`part_of`, `runs_on`, `depends_on`, `member_of`)

### 2.2 四层资产分层 (Hierarchy)
为了支持可视化的树状展开，资产按以下逻辑分层：
1. **L1 业务组织层 (Root)**：Project, Team。作为资产查看的入口。
2. **L2 逻辑服务层 (Application)**：Service。关注业务依赖关系 (`depends_on`)。
3. **L3 部署运行层 (Runtime)**：DeploymentTarget, Cluster。连接逻辑服务与物理资源。
4. **L4 物理基础设施层 (Infrastructure)**：Node/Host。最底层的计算/存储资源。

## 3. 详细功能设计

### 3.1 树状拓扑接口 (Tree-Expansion API)
- **Endpoint**: `GET /api/cmdb/v1/topology`
- **参数**:
    - `parent_id`: 展开的父节点 ID。为空则返回 L1 根节点（项目/团队）。
    - `view_type`: `business` (默认, A 路径) 或 `resource` (B 路径)。
- **返回结构**:
    ```json
    {
      "nodes": [
        { "id": 1, "name": "Order-Service", "ci_type": "service", "ui_hints": { "icon": "box", "color": "purple", "expandable": true } }
      ],
      "edges": [
        { "from": 101, "to": 1, "type": "part_of" }
      ]
    }
    ```

### 3.2 自动发现与生命周期 (LCM)
- **同步机制 (Discovery)**：
    - 定期从 Node、Cluster、Service 模块拉取数据。
    - **Mark & Sweep 算法**：同步时标记活跃 CI。若 CI 在连续 3 次同步中未出现，状态变更为 `stale`（失联）。
- **状态流转**:
    - `Active`: 自动发现在线或人工确认。
    - `Stale`: 自动发现连续失败，但尚未确认下线。
    - `Inactive`: 人工标记或资源已正式释放。
    - `Archived`: 资源已删除，仅保留历史记录。

### 3.3 审计与校准
- **Audit Log**: 记录所有 CI 属性的变更，支持“变更前 vs 变更后”对比。
- **差异同步**: 当自动发现的数据与 CMDB 存储的属性不一致时，以“客观现状”为准自动更新。

## 4. 可视化友好特性 (UI Friendly)
- **UI Hints**: 后端根据 `ci_type` 和 `status` 预计算渲染参数（如图标、发光状态）。
- **Lazy Loading**: 树状结构仅在用户点击时请求子节点，解决大规模资产渲染卡顿问题。

## 5. 后续扩展计划
- **逻辑依赖自动发现**: 通过应用配置或流量分析自动建立 Service ➔ Service 的依赖线。
- **自定义 Schema**: 允许管理员在 UI 上为特定 CI 类型增加自定义字段。
