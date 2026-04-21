# Monitoring Center Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Transform the monitoring module into a unified, high-density Monitoring Center with a integrated dashboard and configuration layout.

**Architecture:** Implement a parent `MonitorCenterLayout` component with side navigation and top stats. Group all `/monitor/*` routes under this layout. Optimize all sub-pages for density.

**Tech Stack:** React, Ant Design, Tailwind CSS.

---

### Task 1: Create MonitorCenterLayout Component

**Files:**
- Create: `web/src/pages/Monitor/MonitorCenterLayout.tsx`
- Modify: `web/src/pages/Monitor/index.ts` (if needed to export)

- [ ] **Step 1: Write the component code**

```tsx
import React, { useEffect, useState } from 'react';
import { Layout, Menu, Card, Statistic, Row, Col, Space } from 'antd';
import { useNavigate, useLocation, Outlet } from 'react-router-dom';
import {
  DashboardOutlined,
  BellOutlined,
  SettingOutlined,
  NotificationOutlined,
  BranchesOutlined,
  HistoryOutlined,
  AlertOutlined,
} from '@ant-design/icons';
import { Api } from '../../api';

const { Sider, Content } = Layout;

const MonitorCenterLayout: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const [stats, setStats] = useState({ firing: 0, rules: 0 });

  useEffect(() => {
    const loadStats = async () => {
      try {
        const [alertRes, ruleRes] = await Promise.all([
          Api.monitoring.getAlertList({ page: 1, pageSize: 1 }),
          Api.monitoring.getAlertRuleList({ page: 1, pageSize: 1 }),
        ]);
        setStats({
          firing: alertRes.data?.total || 0,
          rules: ruleRes.data?.total || 0,
        });
      } catch (err) {
        console.error('Failed to load monitor stats', err);
      }
    };
    loadStats();
  }, [location.pathname]);

  const menuItems = [
    { key: '/monitor/dashboard', icon: <DashboardOutlined />, label: '实时大盘' },
    { key: '/monitor/alerts', icon: <BellOutlined />, label: '告警历史' },
    { key: '/monitor/rules', icon: <SettingOutlined />, label: '规则配置' },
    { key: '/monitor/channels', icon: <NotificationOutlined />, label: '通知渠道' },
    { key: '/monitor/routing', icon: <BranchesOutlined />, label: '路由策略' },
    { key: '/monitor/deliveries', icon: <HistoryOutlined />, label: '投递记录' },
  ];

  return (
    <Layout style={{ background: '#f0f2f5', minHeight: 'calc(100vh - 64px)' }}>
      <Sider width={200} theme="light" style={{ borderRight: '1px solid #f0f0f0' }}>
        <Menu
          mode="inline"
          selectedKeys={[location.pathname.startsWith('/monitor/dashboard') ? '/monitor/dashboard' : location.pathname]}
          style={{ height: '100%' }}
          items={menuItems}
          onClick={({ key }) => navigate(key)}
        />
      </Sider>
      <Content style={{ padding: '12px' }}>
        <Row gutter={[12, 12]} style={{ marginBottom: 12 }}>
          <Col span={6}>
            <Card size="small" bodyStyle={{ padding: '8px 12px' }}>
              <Statistic title="当前活跃告警" value={stats.firing} prefix={<AlertOutlined />} valueStyle={{ fontSize: 20, color: stats.firing > 0 ? '#ff4d4f' : 'inherit' }} />
            </Card>
          </Col>
          <Col span={6}>
            <Card size="small" bodyStyle={{ padding: '8px 12px' }}>
              <Statistic title="告警规则总数" value={stats.rules} valueStyle={{ fontSize: 20 }} />
            </Card>
          </Col>
          {/* Add more stats if needed */}
        </Row>
        <Outlet />
      </Content>
    </Layout>
  );
};

export default MonitorCenterLayout;
```

- [ ] **Step 2: Export from index**
Check `web/src/pages/Monitor/index.ts` and add `export { default as MonitorCenterLayout } from './MonitorCenterLayout';`.

- [ ] **Step 3: Commit**

```bash
git add web/src/pages/Monitor/MonitorCenterLayout.tsx web/src/pages/Monitor/index.ts
git commit -m "feat(monitor): create MonitorCenterLayout for integrated experience"
```

### Task 2: Update Routing Configuration

**Files:**
- Modify: `web/src/app/routes/observability.routes.tsx`

- [ ] **Step 1: Wrap monitor routes with MonitorCenterLayout**

```tsx
// In renderObservabilityRoutes:
<Route path="/monitor" element={withAuth('monitoring', 'read', <MonitorCenterLayout />)}>
  <Route index element={<MonitorPage />} />
  <Route path="dashboard" element={<MonitorPage />} />
  <Route path="alerts" element={<AlertsPage />} />
  <Route path="alerts/:alertId" element={<AlertDetailPage />} />
  <Route path="rules" element={<RulesConfigPage />} />
  <Route path="channels" element={<ChannelsConfigPage />} />
  <Route path="routing" element={<RoutingConfigPage />} />
  <Route path="deliveries" element={<DeliveriesPage />} />
</Route>
```

- [ ] **Step 2: Commit**

```bash
git add web/src/app/routes/observability.routes.tsx
git commit -m "refactor(monitor): group routes under MonitorCenterLayout"
```

### Task 3: Redesign MonitorPage (Dashboard)

**Files:**
- Modify: `web/src/pages/Monitor/MonitorPage.tsx`

- [ ] **Step 1: Compact UI and remove redundant buttons**
- Remove the top `Space` with `Link` buttons.
- Update `Card`, `Statistic`, `Progress`, `Table` to `size="small"`.
- Standardize the layout to be more dense.

- [ ] **Step 2: Commit**

```bash
git add web/src/pages/Monitor/MonitorPage.tsx
git commit -m "style(monitor): compact Dashboard UI"
```

### Task 4: Update All Config Pages (Rules, Channels, Routes, Deliveries)

**Files:**
- Modify: `web/src/pages/Monitor/RulesConfigPage.tsx`
- Modify: `web/src/pages/Monitor/ChannelsConfigPage.tsx`
- Modify: `web/src/pages/Monitor/RoutingConfigPage.tsx`
- Modify: `web/src/pages/Monitor/DeliveriesPage.tsx`

- [ ] **Step 1: Remove old Layout wrappers and apply size="small"**
- Since these are now rendered inside `MonitorCenterLayout`, remove any redundant wrappers if they existed (like the `MonitorConfigLayout` from my previous failed attempt if it was partially applied).
- Ensure all `Table` components use `size="small"`.
- Move "Sync Rules" to `RulesConfigPage` header if not already there.

- [ ] **Step 2: Commit**

```bash
git add web/src/pages/Monitor/RulesConfigPage.tsx web/src/pages/Monitor/ChannelsConfigPage.tsx web/src/pages/Monitor/RoutingConfigPage.tsx web/src/pages/Monitor/DeliveriesPage.tsx
git commit -m "style(monitor): update config pages for high-density integration"
```

### Task 5: Final Polishing and Verification

- [ ] **Step 1: Verify overall navigation and layout consistency**
- [ ] **Step 2: Check global stats update behavior**
- [ ] **Step 3: Final build check**

```bash
npm run build
```
