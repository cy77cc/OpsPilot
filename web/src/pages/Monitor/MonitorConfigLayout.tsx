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
