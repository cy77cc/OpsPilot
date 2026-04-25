import React, { useEffect, useMemo, useState } from 'react';
import { Layout, Tabs, Card, Statistic, Row, Col, Button, Space } from 'antd';
import { useNavigate, useLocation, Outlet } from 'react-router-dom';
import {
  DashboardOutlined,
  BellOutlined,
  SettingOutlined,
  NotificationOutlined,
  BranchesOutlined,
  HistoryOutlined,
  AlertOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import { Api } from '../../api';
import { MonitorRefreshProvider, useMonitorRefresh } from './MonitorRefreshContext';

const { Content } = Layout;

// Separate component for stats to isolate state and prevent re-renders on route change
const MonitorStats: React.FC = () => {
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
    const interval = setInterval(loadStats, 30000);
    return () => clearInterval(interval);
  }, []);

  return (
    <Row gutter={[12, 12]} style={{ marginBottom: 12 }}>
      <Col span={6}>
        <Card bordered={false} style={{ boxShadow: '0 1px 2px 0 rgba(0, 0, 0, 0.03)' }}>
          <Statistic title="当前活跃告警" value={stats.firing} prefix={<AlertOutlined />} styles={{ content: { fontSize: 20, color: stats.firing > 0 ? '#ff4d4f' : 'inherit' } }} />
        </Card>
      </Col>
      <Col span={6}>
        <Card bordered={false} style={{ boxShadow: '0 1px 2px 0 rgba(0, 0, 0, 0.03)' }}>
          <Statistic title="告警规则总数" value={stats.rules} styles={{ content: { fontSize: 20 } }} />
        </Card>
      </Col>
    </Row>
  );
};

const MonitorTabsWithRefresh: React.FC<{ tabItems: any[], activeKey: string, navigate: (key: string) => void }> = ({ tabItems, activeKey, navigate }) => {
  const { onRefresh, loading } = useMonitorRefresh();

  return (
    <div style={{ background: '#fff', padding: '0 16px', borderRadius: 8, marginBottom: 12, boxShadow: '0 1px 2px 0 rgba(0, 0, 0, 0.03)' }}>
      <Tabs
        activeKey={activeKey}
        items={tabItems}
        onChange={(key) => navigate(key)}
        tabBarStyle={{ marginBottom: 0 }}
        tabBarExtraContent={
          <Space style={{ height: 46 }}>
            <Button 
              icon={<ReloadOutlined />} 
              loading={loading} 
              onClick={onRefresh || undefined}
              disabled={!onRefresh}
            >
              刷新
            </Button>
          </Space>
        }
      />
    </div>
  );
};

const MonitorCenterLayout: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();

  const tabItems = useMemo(() => [
    { key: '/observability/monitor/dashboard', label: <span><DashboardOutlined /> 实时大盘</span> },
    { key: '/observability/monitor/alerts', label: <span><BellOutlined /> 告警历史</span> },
    { key: '/observability/monitor/rules', label: <span><SettingOutlined /> 规则配置</span> },
    { key: '/observability/monitor/channels', label: <span><NotificationOutlined /> 通知渠道</span> },
    { key: '/observability/monitor/routing', label: <span><BranchesOutlined /> 路由策略</span> },
    { key: '/observability/monitor/deliveries', label: <span><HistoryOutlined /> 投递记录</span> },
  ], []);

  const getSelectedKey = () => {
    const path = location.pathname;
    if (path === '/observability/monitor' || path === '/observability/monitor/') {
      return '/observability/monitor/dashboard';
    }
    if (path.startsWith('/observability/monitor/dashboard')) {return '/observability/monitor/dashboard';}
    if (path.startsWith('/observability/monitor/alerts')) {return '/observability/monitor/alerts';}
    if (path.startsWith('/observability/monitor/rules')) {return '/observability/monitor/rules';}
    if (path.startsWith('/observability/monitor/channels')) {return '/observability/monitor/channels';}
    if (path.startsWith('/observability/monitor/routing')) {return '/observability/monitor/routing';}
    if (path.startsWith('/observability/monitor/deliveries')) {return '/observability/monitor/deliveries';}
    return path;
  };

  const activeKey = useMemo(() => getSelectedKey(), [location.pathname]);

  return (
    <Layout style={{ background: 'transparent', minHeight: 'calc(100vh - 64px)' }}>
      <Content style={{ padding: '12px' }}>
        <MonitorStats />
        
        <MonitorTabsWithRefresh 
          tabItems={tabItems} 
          activeKey={activeKey} 
          navigate={navigate} 
        />

        <Outlet />
      </Content>
    </Layout>
  );
};

const MonitorCenterRoot: React.FC = () => (
  <MonitorRefreshProvider>
    <MonitorCenterLayout />
  </MonitorRefreshProvider>
);

export default MonitorCenterRoot;


