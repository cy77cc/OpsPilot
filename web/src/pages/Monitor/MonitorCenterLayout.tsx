import React, { useEffect, useState } from 'react';
import { Layout, Tabs, Card, Statistic, Row, Col } from 'antd';
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

const { Content } = Layout;

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
    const interval = setInterval(loadStats, 30000);
    return () => clearInterval(interval);
  }, []);

  const tabItems = [
    { key: '/observability/monitor/dashboard', label: <span><DashboardOutlined /> 实时大盘</span> },
    { key: '/observability/monitor/alerts', label: <span><BellOutlined /> 告警历史</span> },
    { key: '/observability/monitor/rules', label: <span><SettingOutlined /> 规则配置</span> },
    { key: '/observability/monitor/channels', label: <span><NotificationOutlined /> 通知渠道</span> },
    { key: '/observability/monitor/routing', label: <span><BranchesOutlined /> 路由策略</span> },
    { key: '/observability/monitor/deliveries', label: <span><HistoryOutlined /> 投递记录</span> },
  ];

  const getSelectedKey = () => {
    if (location.pathname.startsWith('/observability/monitor/dashboard')) {return '/observability/monitor/dashboard';}
    if (location.pathname.startsWith('/observability/monitor/alerts')) {return '/observability/monitor/alerts';}
    if (location.pathname.startsWith('/observability/monitor/rules')) {return '/observability/monitor/rules';}
    if (location.pathname.startsWith('/observability/monitor/channels')) {return '/observability/monitor/channels';}
    if (location.pathname.startsWith('/observability/monitor/routing')) {return '/observability/monitor/routing';}
    if (location.pathname.startsWith('/observability/monitor/deliveries')) {return '/observability/monitor/deliveries';}
    return location.pathname;
  };

  return (
    <Layout style={{ background: '#f0f2f5', minHeight: 'calc(100vh - 64px)' }}>
      <Content style={{ padding: '12px' }}>
        <Row gutter={[12, 12]} style={{ marginBottom: 12 }}>
          <Col span={6}>
            <Card>
              <Statistic title="当前活跃告警" value={stats.firing} prefix={<AlertOutlined />} styles={{ content: { fontSize: 20, color: stats.firing > 0 ? '#ff4d4f' : 'inherit' } }} />
            </Card>
          </Col>
          <Col span={6}>
            <Card>
              <Statistic title="告警规则总数" value={stats.rules} styles={{ content: { fontSize: 20 } }} />
            </Card>
          </Col>
          {/* Add more stats if needed */}
        </Row>
        
        <div style={{ background: '#fff', padding: '0 16px', borderRadius: 8, marginBottom: 12, boxShadow: '0 1px 2px 0 rgba(0, 0, 0, 0.03)' }}>
          <Tabs
            activeKey={getSelectedKey()}
            items={tabItems}
            onChange={(key) => navigate(key)}
            tabBarStyle={{ marginBottom: 0 }}
          />
        </div>

        <Outlet />
      </Content>
    </Layout>
  );
};

export default MonitorCenterLayout;
