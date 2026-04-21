import React, { useEffect, useState } from 'react';
import { Layout, Menu, Card, Statistic, Row, Col } from 'antd';
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

  const getSelectedKey = () => {
    if (location.pathname.startsWith('/monitor/dashboard')) return '/monitor/dashboard';
    if (location.pathname.startsWith('/monitor/alerts')) return '/monitor/alerts';
    if (location.pathname.startsWith('/monitor/rules')) return '/monitor/rules';
    if (location.pathname.startsWith('/monitor/channels')) return '/monitor/channels';
    if (location.pathname.startsWith('/monitor/routing')) return '/monitor/routing';
    if (location.pathname.startsWith('/monitor/deliveries')) return '/monitor/deliveries';
    return location.pathname;
  };

  return (
    <Layout style={{ background: '#f0f2f5', minHeight: 'calc(100vh - 64px)' }}>
      <Sider width={200} theme="light" style={{ borderRight: '1px solid #f0f0f0' }}>
        <Menu
          mode="inline"
          selectedKeys={[getSelectedKey()]}
          style={{ height: '100%' }}
          items={menuItems}
          onClick={({ key }) => navigate(key)}
        />
      </Sider>
      <Content style={{ padding: '12px' }}>
        <Row gutter={[12, 12]} style={{ marginBottom: 12 }}>
          <Col span={6}>
            <Card size="small">
              <Statistic title="当前活跃告警" value={stats.firing} prefix={<AlertOutlined />} valueStyle={{ fontSize: 20, color: stats.firing > 0 ? '#ff4d4f' : 'inherit' }} />
            </Card>
          </Col>
          <Col span={6}>
            <Card size="small">
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
