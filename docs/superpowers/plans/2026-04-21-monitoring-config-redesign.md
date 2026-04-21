# Monitoring Configuration Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Consolidate monitoring configuration pages into a unified, high-density configuration center.

**Architecture:** Create a `MonitorConfigLayout` wrapper component with a compact sidebar. Update existing configuration pages to use `size="small"` for tables and standard headers. Move the "Sync Rules" action to the Rules page.

**Tech Stack:** React, Ant Design, Tailwind CSS.

---

### Task 1: Create MonitorConfigLayout Component

**Files:**
- Create: `web/src/pages/Monitor/MonitorConfigLayout.tsx`

- [ ] **Step 1: Write the component code**

```tsx
import React from 'react';
import { Layout, Menu } from 'antd';
import { useNavigate, useLocation } from 'react-router-dom';
import {
  SettingOutlined,
  NotificationOutlined,
  BranchesOutlined,
  HistoryOutlined,
} from '@ant-design/icons';

const { Sider, Content } = Layout;

const MonitorConfigLayout: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const navigate = useNavigate();
  const location = useLocation();

  const menuItems = [
    {
      key: '/monitor/rules',
      icon: <SettingOutlined />,
      label: '告警规则',
    },
    {
      key: '/monitor/channels',
      icon: <NotificationOutlined />,
      label: '通知渠道',
    },
    {
      key: '/monitor/routing',
      icon: <BranchesOutlined />,
      label: '路由策略',
    },
    {
      key: '/monitor/deliveries',
      icon: <HistoryOutlined />,
      label: '投递记录',
    },
  ];

  return (
    <Layout style={{ background: '#fff', minHeight: 'calc(100vh - 64px)' }}>
      <Sider width={200} style={{ background: '#fff', borderRight: '1px solid #f0f0f0' }}>
        <Menu
          mode="inline"
          selectedKeys={[location.pathname]}
          style={{ height: '100%', borderRight: 0 }}
          items={menuItems}
          onClick={({ key }) => navigate(key)}
        />
      </Sider>
      <Content style={{ padding: '0 24px', minHeight: 280 }}>
        <div style={{ padding: '24px 0' }}>{children}</div>
      </Content>
    </Layout>
  );
};

export default MonitorConfigLayout;
```

- [ ] **Step 2: Commit**

```bash
git add web/src/pages/Monitor/MonitorConfigLayout.tsx
git commit -m "feat(monitor): add MonitorConfigLayout for unified configuration"
```

### Task 2: Redesign RulesConfigPage

**Files:**
- Modify: `web/src/pages/Monitor/RulesConfigPage.tsx`

- [ ] **Step 1: Update component to use MonitorConfigLayout and size="small" table**

```tsx
// ... imports ...
import MonitorConfigLayout from './MonitorConfigLayout';
import { SyncOutlined } from '@ant-design/icons';

// In RulesConfigPage component:
// 1. Wrap the return with <MonitorConfigLayout>
// 2. Update Table to size="small"
// 3. Add handleSyncRules function
// 4. Add "Sync Rules" button to Card extra

const handleSyncRules = async () => {
  try {
    await Api.monitoring.syncAlertRules();
    message.success('规则同步成功');
    void load();
  } catch (error: any) {
    message.error(error?.message || '规则同步失败');
  }
};

// ... in return ...
return (
  <MonitorConfigLayout>
    <Card
      title="告警规则配置"
      size="small"
      extra={(
        <Space size="small">
          <ScopeSelector value={scope} onChange={setScope} />
          <Button icon={<SyncOutlined />} onClick={handleSyncRules}>
            同步规则
          </Button>
          <Button type="primary" onClick={() => setCreateOpen(true)}>
            新增规则
          </Button>
        </Space>
      )}
    >
      <Table
        size="small"
        // ... other props ...
      />
      {/* ... Modals and Drawer ... */}
    </Card>
  </MonitorConfigLayout>
);
```

- [ ] **Step 2: Run verification**
Manual verification of the UI layout and "Sync Rules" functionality.

- [ ] **Step 3: Commit**

```bash
git add web/src/pages/Monitor/RulesConfigPage.tsx
git commit -m "style(monitor): compact RulesConfigPage and move Sync Rules button"
```

### Task 3: Redesign ChannelsConfigPage

**Files:**
- Modify: `web/src/pages/Monitor/ChannelsConfigPage.tsx`

- [ ] **Step 1: Update component to use MonitorConfigLayout and size="small" table**

```tsx
// ... imports ...
import MonitorConfigLayout from './MonitorConfigLayout';

// In ChannelsConfigPage component:
// 1. Wrap the return with <MonitorConfigLayout>
// 2. Update Tables to size="small"
// 3. Update Cards to size="small"

return (
  <MonitorConfigLayout>
    <Space orientation="vertical" size="small" style={{ width: '100%' }}>
      <Card
        title="通知渠道配置"
        size="small"
        extra={(
          <Space size="small">
            <ScopeSelector value={scope} onChange={setScope} />
            <Button type="primary" onClick={() => setCreateOpen(true)}>
              新增渠道
            </Button>
          </Space>
        )}
      >
        <Table size="small" ... />
      </Card>
      <Card title="渠道测试发送" size="small">
        <Form ... size="small">
          {/* ... */}
        </Form>
      </Card>
    </Space>
  </MonitorConfigLayout>
);
```

- [ ] **Step 2: Commit**

```bash
git add web/src/pages/Monitor/ChannelsConfigPage.tsx
git commit -m "style(monitor): compact ChannelsConfigPage"
```

### Task 4: Redesign RoutingConfigPage

**Files:**
- Modify: `web/src/pages/Monitor/RoutingConfigPage.tsx`

- [ ] **Step 1: Update component to use MonitorConfigLayout and size="small" table**

```tsx
// ... imports ...
import MonitorConfigLayout from './MonitorConfigLayout';

// In RoutingConfigPage component:
// 1. Wrap the return with <MonitorConfigLayout>
// 2. Update Table to size="small"
// 3. Update Card to size="small"

return (
  <MonitorConfigLayout>
    <Card
      title="路由配置"
      size="small"
      extra={(
        <Space size="small">
          <ScopeSelector value={scope} onChange={setScope} />
          <Button type="primary" onClick={() => setCreateOpen(true)}>
            新增路由
          </Button>
        </Space>
      )}
    >
      <Table size="small" ... />
    </Card>
  </MonitorConfigLayout>
);
```

- [ ] **Step 2: Commit**

```bash
git add web/src/pages/Monitor/RoutingConfigPage.tsx
git commit -m "style(monitor): compact RoutingConfigPage"
```

### Task 5: Redesign DeliveriesPage

**Files:**
- Modify: `web/src/pages/Monitor/DeliveriesPage.tsx`

- [ ] **Step 1: Update component to use MonitorConfigLayout and size="small" table**

```tsx
// ... imports ...
import MonitorConfigLayout from './MonitorConfigLayout';

// In DeliveriesPage component:
// 1. Wrap the return with <MonitorConfigLayout>
// 2. Update Table to size="small"
// 3. Update Card to size="small"

return (
  <MonitorConfigLayout>
    <Card title="投递记录" size="small">
      <Table size="small" ... />
    </Card>
  </MonitorConfigLayout>
);
```

- [ ] **Step 2: Commit**

```bash
git add web/src/pages/Monitor/DeliveriesPage.tsx
git commit -m "style(monitor): compact DeliveriesPage"
```

### Task 6: Cleanup MonitorPage

**Files:**
- Modify: `web/src/pages/Monitor/MonitorPage.tsx`

- [ ] **Step 1: Remove configuration links and Sync Rules button**

```tsx
// Remove configuration links and Sync button from the header Space
// Keep Reload button
<div className="flex justify-end">
  <Space>
    <Button icon={<ReloadOutlined />} loading={loading && !isInitialLoading} onClick={load}>刷新</Button>
  </Space>
</div>
```

- [ ] **Step 2: Commit**

```bash
git add web/src/pages/Monitor/MonitorPage.tsx
git commit -m "refactor(monitor): remove redundant config buttons from dashboard"
```

### Task 7: Final Verification

- [ ] **Step 1: Run dev build and check all monitoring pages**
- [ ] **Step 2: Verify all CRUD and Sync operations**
