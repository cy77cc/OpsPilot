import React, { useState, useEffect } from 'react';
import { Layout, Menu, Breadcrumb, Avatar, Dropdown, Input, Tooltip, Button, Drawer } from 'antd';
import type { MenuProps } from 'antd';
import {
  DashboardOutlined,
  DesktopOutlined,
  SettingOutlined,
  AlertOutlined,
  CloudOutlined,
  ClockCircleOutlined,
  ToolOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  SearchOutlined,
  UserOutlined,
  LogoutOutlined,
  QuestionCircleOutlined,
  CloudServerOutlined,
  FileTextOutlined,
  MenuOutlined,
} from '@ant-design/icons';
import { useNavigate, useLocation } from 'react-router-dom';
import { useAuth } from '../Auth/AuthContext';
import ProjectSwitcher from '../Project/ProjectSwitcher';
import { NotificationBell } from '../Notification';
import { AICopilotButton, AISurfaceBoundary, CopilotSurface } from '../AI';
import '../Notification/notification.css';
import { useI18n } from '../../i18n';
import { usePermission } from '../RBAC';
import CommandPalette from '../CommandPalette';
import KeyboardShortcutsHelp from '../KeyboardShortcutsHelp';
import { useKeyboardShortcuts } from '../../hooks/useKeyboardShortcuts';
import PageTransition from '../PageTransition';

const { Header, Sider, Content } = Layout;
type MenuItem = Required<MenuProps>['items'][number];

const menuRouteOverrides: Record<string, string> = {
  '/services/all': '/services',
};

type MenuLeaf = {
  key: string;
  icon?: React.ReactNode;
  label: string;
};

type MenuSection = {
  key: string;
  title: string;
  items: MenuLeaf[];
};

function findSectionPath(sections: MenuSection[], targetKey: string): { title: string; key?: string }[] {
  for (const section of sections) {
    const item = section.items.find((entry) => entry.key === targetKey);
    if (item) {
      return [{ title: section.title }, { title: item.label, key: item.key }];
    }
  }
  return [];
}

interface AppLayoutProps {
  children: React.ReactNode;
}

const AppLayout: React.FC<AppLayoutProps> = ({ children }) => {
  const [collapsed, setCollapsed] = useState(false);
  const [mobileDrawerOpen, setMobileDrawerOpen] = useState(false);
  const [copilotOpen, setCopilotOpen] = useState(false);
  const [isMobile, setIsMobile] = useState(false);
  const [commandPaletteOpen, setCommandPaletteOpen] = useState(false);
  const [helpOpen, setHelpOpen] = useState(false);
  const navigate = useNavigate();
  const location = useLocation();
  const { logout } = useAuth();
  const { hasPermission } = usePermission();
  const { t, lang, setLang } = useI18n();
  const governanceMenuEnabled = import.meta.env.VITE_FEATURE_GOVERNANCE_MENU !== 'false';
  const canReadGovernance = hasPermission('rbac', 'read');

  // 4.1.9 使用键盘快捷键 Hook
  useKeyboardShortcuts({
    onOpenHelp: () => setHelpOpen(true),
    enableNavigation: true,
    enableListNavigation: false,
  });

  // 3.1.5 响应式布局检测
  useEffect(() => {
    const checkMobile = () => {
      setIsMobile(window.innerWidth < 768);
    };
    checkMobile();
    window.addEventListener('resize', checkMobile);
    return () => window.removeEventListener('resize', checkMobile);
  }, []);

  // 4.1.4 全局快捷键 Cmd+K / Ctrl+K
  useEffect(() => {
    const down = (e: KeyboardEvent) => {
      if (e.key === 'k' && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        setCommandPaletteOpen((open) => !open);
      }
    };

    document.addEventListener('keydown', down);
    return () => document.removeEventListener('keydown', down);
  }, []);

  const menuSections: MenuSection[] = [
    {
      key: 'overview',
      title: '总览',
      items: [
        { key: '/', icon: <DashboardOutlined />, label: t('menu.dashboard') },
      ],
    },
    {
      key: 'delivery',
      title: '研发交付',
      items: [
        { key: '/services', icon: <CloudServerOutlined />, label: t('menu.services') },
        { key: '/deployment', icon: <CloudOutlined />, label: '发布中心' },
        { key: '/deployment/targets', icon: <CloudOutlined />, label: '发布目标' },
        { key: '/cicd', icon: <ToolOutlined />, label: 'CI/CD' },
      ],
    },
    {
      key: 'infrastructure',
      title: '基础设施',
      items: [
        { key: '/deployment/infrastructure/clusters', icon: <CloudOutlined />, label: '集群管理' },
        { key: '/deployment/infrastructure/hosts', icon: <DesktopOutlined />, label: '主机管理' },
      ],
    },
    {
      key: 'observability',
      title: '观测治理',
      items: [
        { key: '/monitor', icon: <AlertOutlined />, label: t('menu.monitor') },
        { key: '/deployment/observability/topology', icon: <CloudOutlined />, label: '部署拓扑' },
        { key: '/deployment/observability/audit-logs', icon: <AlertOutlined />, label: '审计日志' },
        { key: '/deployment/observability/policies', icon: <AlertOutlined />, label: '策略管理' },
        { key: '/deployment/observability/aiops', icon: <AlertOutlined />, label: 'AIOps 洞察' },
      ],
    },
    {
      key: 'ops',
      title: '运维运营',
      items: [
        { key: '/automation', icon: <ToolOutlined />, label: t('menu.automation') },
        { key: '/tasks', icon: <ClockCircleOutlined />, label: t('menu.tasks') },
        { key: '/cmdb', icon: <CloudServerOutlined />, label: t('menu.cmdb') },
      ],
    },
    {
      key: 'support',
      title: '平台与支持',
      items: [
        { key: '/settings', icon: <SettingOutlined />, label: '基础设置' },
        { key: '/settings/ai-models', icon: <SettingOutlined />, label: 'AI 模型配置' },
        ...(!governanceMenuEnabled ? [
          { key: '/settings/users', icon: <UserOutlined />, label: '用户管理' },
          { key: '/settings/roles', icon: <UserOutlined />, label: '角色管理' },
          { key: '/settings/permissions', icon: <UserOutlined />, label: '权限列表' },
        ] : []),
        ...(governanceMenuEnabled && canReadGovernance
          ? [
              { key: '/governance/users', icon: <UserOutlined />, label: '访问治理' },
            ]
          : []),
        { key: '/tools', icon: <ToolOutlined />, label: t('menu.tools') },
        { key: '/help', icon: <FileTextOutlined />, label: '帮助中心' },
      ],
    },
  ];
  const menuItems: MenuItem[] = menuSections.map((section) => ({
    type: 'group',
    key: section.key,
    label: section.title,
    children: section.items.map((item) => ({
      key: item.key,
      icon: item.icon,
      label: item.label,
    })),
  }));

  const activeMenuKey = React.useMemo(() => {
    if (location.pathname.startsWith('/jobs')) return '/tasks';
    if (location.pathname.startsWith('/k8s')) return '/deployment';
    if (location.pathname.startsWith('/deployment/overview')) return '/deployment';
    if (location.pathname.startsWith('/deployment/create')) return '/deployment';
    if (location.pathname.startsWith('/deployment/approvals')) return '/deployment';
    if (location.pathname.startsWith('/deployment/targets')) return '/deployment/targets';
    if (location.pathname.startsWith('/deployment/infrastructure/clusters')) return '/deployment/infrastructure/clusters';
    if (location.pathname.startsWith('/deployment/infrastructure/credentials')) return '/deployment/infrastructure/clusters';
    if (location.pathname.startsWith('/deployment/infrastructure/hosts')) return '/deployment/infrastructure/hosts';
    if (location.pathname.startsWith('/hosts')) return '/deployment/infrastructure/hosts';
    if (location.pathname.startsWith('/deployment/observability/metrics')) return '/monitor';
    if (location.pathname.startsWith('/deployment/observability/topology')) return '/deployment/observability/topology';
    if (location.pathname.startsWith('/deployment/observability/audit-logs')) return '/deployment/observability/audit-logs';
    if (location.pathname.startsWith('/deployment/observability/policies')) return '/deployment/observability/policies';
    if (location.pathname.startsWith('/deployment/observability/aiops')) return '/deployment/observability/aiops';
    if (location.pathname.startsWith('/monitoring')) return '/monitor';
    if (location.pathname.startsWith('/monitor')) return '/monitor';
    if (location.pathname.startsWith('/automation')) return '/automation';
    if (location.pathname.startsWith('/cicd')) return '/cicd';
    if (location.pathname.startsWith('/cmdb')) return '/cmdb';
    if (location.pathname.startsWith('/tools')) return '/tools';
    if (location.pathname.startsWith('/services')) return '/services';
    if (location.pathname.startsWith('/settings/ai-models')) return '/settings/ai-models';
    if (location.pathname.startsWith('/governance')) return '/governance/users';
    if (location.pathname.startsWith('/settings/users')) return '/settings/users';
    if (location.pathname.startsWith('/settings/roles')) return '/settings/roles';
    if (location.pathname.startsWith('/settings/permissions')) return '/settings/permissions';
    if (location.pathname.startsWith('/settings')) return '/settings';
    if (location.pathname.startsWith('/help')) return '/help';
    if (location.pathname.startsWith('/deployment/') && location.pathname !== '/deployment/targets') return '/deployment';
    return location.pathname;
  }, [location.pathname]);

  const menuPath = React.useMemo(() => findSectionPath(menuSections, activeMenuKey), [menuSections, activeMenuKey]);

  const userMenuItems = [
    { key: 'profile', icon: <UserOutlined />, label: '个人中心' },
    { key: 'settings', icon: <SettingOutlined />, label: '系统设置' },
    { type: 'divider' as const },
    { key: 'logout', icon: <LogoutOutlined />, label: '退出登录' },
  ];

  const getBreadcrumbItems = () => {
    const items = [{ title: '首页', path: '/' }];
    for (const entry of menuPath) {
      items.push({ title: entry.title, path: entry.key });
    }
    return items;
  };

  const handleMenuClick = (key: string) => {
    if (!key.startsWith('/')) {
      return;
    }
    navigate(menuRouteOverrides[key] || key);
    if (isMobile) {
      setMobileDrawerOpen(false);
    }
  };

  // 3.1.2 & 3.1.3 侧边栏内容
  const sidebarContent = (
    <div className="flex flex-col h-full">
      {/* Logo 区域 */}
      <div className="h-14 flex-shrink-0 flex items-center justify-center border-b border-gray-200 px-3">
        <div className="flex items-center gap-2.5">
          <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-primary-500 to-primary-600 flex items-center justify-center shadow-sm">
            <CloudOutlined className="text-white text-base" />
          </div>
          {!collapsed && (
            <span className="text-gray-900 font-semibold text-lg">OpsPilot</span>
          )}
        </div>
      </div>

      {/* 菜单 - 可滚动区域 */}
      <div className="flex-1 overflow-y-auto overflow-x-hidden py-1">
        <Menu
          theme="light"
          mode="inline"
          selectedKeys={[activeMenuKey]}
          items={menuItems}
          onClick={({ key }) => handleMenuClick(key)}
          className="border-none mt-0 [&_.ant-menu-item-group-title]:px-3 [&_.ant-menu-item-group-title]:pb-1 [&_.ant-menu-item-group-title]:pt-2.5 [&_.ant-menu-item-group-title]:text-[10px] [&_.ant-menu-item-group-title]:font-semibold [&_.ant-menu-item-group-title]:uppercase [&_.ant-menu-item-group-title]:tracking-[0.08em] [&_.ant-menu-item-group-title]:text-gray-400 [&_.ant-menu-item-group-list]:space-y-0.5 [&_.ant-menu-item]:mx-2 [&_.ant-menu-item]:my-0 [&_.ant-menu-item]:h-9 [&_.ant-menu-item]:leading-9 [&_.ant-menu-item]:px-3 [&_.ant-menu-item_.ant-menu-item-icon]:mr-2.5"
          style={{ background: 'transparent' }}
        />
      </div>

      {/* 折叠按钮 (仅桌面端) */}
      {!isMobile && (
        <div className="flex-shrink-0 px-3 py-2.5 border-t border-gray-200">
          <Button
            type="text"
            icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
            onClick={() => setCollapsed(!collapsed)}
            className="w-full text-gray-500 hover:text-gray-900 hover:bg-gray-100"
          />
        </div>
      )}
    </div>
  );

  return (
    <Layout className="min-h-screen">
      {/* 4.1.2 & 4.1.3 命令面板 */}
      <CommandPalette open={commandPaletteOpen} onOpenChange={setCommandPaletteOpen} />

      {/* 4.1.13 快捷键帮助对话框 */}
      <KeyboardShortcutsHelp open={helpOpen} onClose={() => setHelpOpen(false)} />

      {/* 3.1.5 & 3.1.7 桌面端侧边栏 / 移动端抽屉 */}
      {isMobile ? (
        // 移动端抽屉式侧边栏
        <Drawer
          placement="left"
          onClose={() => setMobileDrawerOpen(false)}
          open={mobileDrawerOpen}
          closable={false}
          width={240}
          bodyStyle={{ padding: 0, height: '100%', display: 'flex', flexDirection: 'column' }}
          className="mobile-sidebar-drawer"
        >
          {sidebarContent}
        </Drawer>
      ) : (
        // 桌面端固定侧边栏
        <Sider
          trigger={null}
          collapsible
          collapsed={collapsed}
          width={240}
          theme="light"
          className="fixed left-0 top-0 bottom-0 z-50 shadow-sm"
          style={{
            background: '#ffffff',
            borderRight: '1px solid #e9ecef',
            height: '100vh',
          }}
        >
          {sidebarContent}
        </Sider>
      )}

      <Layout
        style={{
          marginLeft: isMobile ? 0 : collapsed ? 80 : 240,
          transition: 'margin-left 0.2s',
        }}
      >
        {/* 3.1.4 顶部导航 */}
        <Header
          className="h-16 px-4 md:px-6 flex items-center justify-between bg-white shadow-sm"
          style={{
            position: 'sticky',
            top: 0,
            zIndex: 40,
            borderBottom: '1px solid #e9ecef',
          }}
        >
          <div className="flex items-center gap-4">
            {/* 移动端菜单按钮 */}
            {isMobile && (
              <Button
                type="text"
                icon={<MenuOutlined />}
                onClick={() => setMobileDrawerOpen(true)}
                className="text-gray-600"
              />
            )}

            {/* 面包屑 (桌面端显示) */}
            {!isMobile && (
              <Breadcrumb
                items={getBreadcrumbItems().map((item, index) => ({
                  title:
                    index === getBreadcrumbItems().length - 1 ? (
                      <span className="text-gray-900 font-medium">{item.title}</span>
                    ) : !item.path ? (
                      <span className="text-gray-600">{item.title}</span>
                    ) : (
                      <a
                        onClick={() => navigate(item.path!)}
                        className="text-gray-600 hover:text-primary-600 cursor-pointer"
                      >
                        {item.title}
                      </a>
                    ),
                }))}
                separator="/"
              />
            )}
          </div>

          <div className="flex items-center gap-2 md:gap-3">
            {/* 项目切换器 (桌面端显示) */}
            {!isMobile && <ProjectSwitcher />}

            {/* 语言切换 (桌面端显示) */}
            {!isMobile && (
              <select
                value={lang}
                onChange={(e) => setLang(e.target.value as 'zh-CN' | 'en-US')}
                className="border border-gray-300 rounded-lg h-9 px-3 text-sm text-gray-700 bg-white hover:border-primary-500 focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-100"
              >
                <option value="zh-CN">{t('lang.zh')}</option>
                <option value="en-US">{t('lang.en')}</option>
              </select>
            )}

            {/* 搜索框 (桌面端显示) */}
            {!isMobile && (
              <Input
                placeholder="搜索..."
                prefix={<SearchOutlined className="text-gray-400" />}
                className="w-48 lg:w-64"
                style={{
                  background: '#f8f9fa',
                  border: '1px solid #e9ecef',
                  borderRadius: '8px',
                }}
              />
            )}

            {/* 帮助按钮 */}
            <Tooltip title={<span>帮助文档 <kbd className="ml-1 text-xs">?</kbd></span>}>
              <Button
                type="text"
                icon={<QuestionCircleOutlined />}
                className="text-gray-600 hover:text-primary-600"
                onClick={() => setHelpOpen(true)}
              />
            </Tooltip>

            {/* 通知 */}
            <NotificationBell onViewAll={() => navigate('/monitor')} />

            {/* AI 助手 */}
            <AICopilotButton onOpen={() => setCopilotOpen(true)} />

            {/* 用户菜单 */}
            <Dropdown
              menu={{
                items: userMenuItems,
                onClick: ({ key }) => {
                  if (key === 'logout') {
                    logout();
                    navigate('/login', { replace: true });
                  }
                  if (key === 'settings') {
                    navigate('/settings');
                  }
                },
              }}
              placement="bottomRight"
            >
              <Avatar
                className="bg-primary-500 cursor-pointer hover:bg-primary-600 transition-colors"
                icon={<UserOutlined />}
              />
            </Dropdown>
          </div>
        </Header>

        {/* 3.1.6 移动端底部导航 */}
        {isMobile && (
          <div className="fixed bottom-0 left-0 right-0 h-16 bg-white border-t border-gray-200 flex items-center justify-around z-50 shadow-lg">
            <Button
              type="text"
              icon={<DashboardOutlined />}
              onClick={() => navigate('/')}
              className={location.pathname === '/' ? 'text-primary-600' : 'text-gray-600'}
            />
            <Button
              type="text"
              icon={<CloudServerOutlined />}
              onClick={() => navigate('/services')}
              className={location.pathname.startsWith('/services') ? 'text-primary-600' : 'text-gray-600'}
            />
            <Button
              type="text"
              icon={<DesktopOutlined />}
              onClick={() => navigate('/hosts')}
              className={location.pathname.startsWith('/hosts') ? 'text-primary-600' : 'text-gray-600'}
            />
            <Button
              type="text"
              icon={<SettingOutlined />}
              onClick={() => navigate('/settings')}
              className={location.pathname.startsWith('/settings') ? 'text-primary-600' : 'text-gray-600'}
            />
          </div>
        )}

        <Content
          className="p-4 md:p-6 bg-gray-50"
          style={{
            minHeight: isMobile ? 'calc(100vh - 128px)' : 'calc(100vh - 64px)',
          }}
        >
          {/* 4.2.1 页面切换动画 */}
          <PageTransition>{children}</PageTransition>
        </Content>
      </Layout>

      <AISurfaceBoundary>
        <CopilotSurface open={copilotOpen} onClose={() => setCopilotOpen(false)} />
      </AISurfaceBoundary>
    </Layout>
  );
};

export default AppLayout;
