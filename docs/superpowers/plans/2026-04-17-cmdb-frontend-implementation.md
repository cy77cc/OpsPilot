# CMDB 前端模块实施计划 (CMDB Frontend Implementation Plan)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现一个基于 AntV G6 的三位一体交互式 CMDB 前端页面，支持树状导航、层级拓扑和详情查看。

**Architecture:** 采用分栏布局，左侧 Sidebar 树作为资产选择器，中间画布负责拓扑渲染，右侧 Drawer 负责详情展示。使用 AntV G6 引擎处理大规模图谱。

**Tech Stack:** React (TypeScript), Ant Design, AntV G6, SWR/React Query.

---

### Task 1: API 封装与 Hooks 定义

**Files:**
- Modify: `web/src/api/modules/cmdb.ts`
- Create: `web/src/hooks/useCMDB.ts`

- [ ] **Step 1: 更新 API 模型定义**
添加 `getTree`, `getTopology`, `getAssetDetail` 等接口。

- [ ] **Step 2: 封装 `useCMDBTree` Hook**
支持异步加载逻辑，管理树的展开状态。

- [ ] **Step 3: 封装 `useCMDBTopology` Hook**
根据选中的节点 ID 获取局部拓扑数据，处理 G6 所需的数据格式转换。

- [ ] **Step 4: Commit**
```bash
git add web/src/api/modules/cmdb.ts web/src/hooks/useCMDB.ts
git commit -m "feat(web/cmdb): encapsulate api and hooks for tree/topology"
```

---

### Task 2: 分栏布局与侧边资产树 (Layout & Sidebar Tree)

**Files:**
- Modify: `web/src/pages/CMDB/CMDBPage.tsx`
- Create: `web/src/pages/CMDB/components/AssetTree.tsx`

- [ ] **Step 1: 实现响应式分栏布局**
使用 Ant Design `Layout` 或 `Splitter` 划分 Tree、Graph、Drawer 区域。

- [ ] **Step 2: 实现 `AssetTree` 组件**
集成 `useCMDBTree`，实现点击节点触发全局状态更新。

- [ ] **Step 3: Commit**
```bash
git add web/src/pages/CMDB/CMDBPage.tsx web/src/pages/CMDB/components/AssetTree.tsx
git commit -m "feat(web/cmdb): implement split layout and hierarchy tree"
```

---

### Task 3: AntV G6 拓扑画布集成 (Graph Component)

**Files:**
- Create: `web/src/pages/CMDB/components/TopologyGraph.tsx`
- Create: `web/src/pages/CMDB/utils/graph-helper.ts`

- [ ] **Step 1: 初始化 G6 图表**
设置基础配置（Dagre 布局、缩放、平移、交互行为）。

- [ ] **Step 2: 实现自定义节点与连线样式**
根据 `ui_hints` (icon, color, glow) 动态渲染节点。

- [ ] **Step 3: 实现画布交互**
双击节点展开、单击查看详情。

- [ ] **Step 4: Commit**
```bash
git add web/src/pages/CMDB/components/TopologyGraph.tsx
git commit -m "feat(web/cmdb): integrate AntV G6 for topology visualization"
```

---

### Task 4: 详情抽屉与审计视图 (Detail Drawer)

**Files:**
- Create: `web/src/pages/CMDB/components/AssetDetailDrawer.tsx`

- [ ] **Step 1: 实现属性列表展示**
格式化展示 `attrs_json` 中的动态属性。

- [ ] **Step 2: 实现审计历史时间轴**
调用审计接口，展示 CI 的历史变更记录。

- [ ] **Step 3: Commit**
```bash
git add web/src/pages/CMDB/components/AssetDetailDrawer.tsx
git commit -m "feat(web/cmdb): add detail drawer and audit history"
```
