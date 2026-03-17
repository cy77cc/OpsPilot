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
  '/deployment/history': '/deployment',
};

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

  const baseMenuItems: MenuItem[] = [
    { key: '/', icon: <DashboardOutlined />, label: t('menu.dashboard') },
    {
      key: '/services',
      icon: <CloudServerOutlined />,
      label: t('menu.services'),
      children: [
        { key: '/services/all', label: '全部服务' },
        { key: '/services?service_kind=middleware', label: '中间件服务' },
        { key: '/services?service_kind=business', label: '业务服务' },
      ],
    },
    { key: '/cmdb', icon: <CloudServerOutlined />, label: t('menu.cmdb') },
    { key: '/automation', icon: <ToolOutlined />, label: t('menu.automation') },
    { key: '/cicd', icon: <ToolOutlined />, label: 'CI/CD' },
    { key: '/help', icon: <FileTextOutlined />, label: '帮助中心' },
    { key: '/config', icon: <SettingOutlined />, label: t('menu.config') },
    { key: '/tasks', icon: <ClockCircleOutlined />, label: t('menu.tasks') },
    {
      key: '/deployment',
      icon: <CloudOutlined />,
      label: '部署管理',
      children: [
        {
          key: 'infrastructure',
          label: '基础设施',
          children: [
            { key: '/deployment/infrastructure/clusters', label: '集群管理' },
            { key: '/deployment/infrastructure/credentials', label: '凭证管理' },
            { key: '/deployment/infrastructure/hosts', label: '主机管理' },
          ],
        },
        {
          key: 'targets',
          label: '部署目标',
          children: [
            { key: '/deployment/targets', label: '目标列表' },
            { key: '/deployment/targets/create', label: '创建目标' },
          ],
        },
        {
          key: 'releases',
          label: '发布管理',
          children: [
            { key: '/deployment/overview', label: '发布概览' },
            { key: '/deployment/create', label: '创建发布' },
            { key: '/deployment/history', label: '发布历史' },
            { key: '/deployment/approvals', label: '审批中心' },
          ],
        },
        {
          key: 'observability',
          label: '可观测性',
          children: [
            { key: '/deployment/observability/topology', label: '部署拓扑' },
            { key: '/deployment/observability/metrics', label: '指标仪表板' },
            { key: '/deployment/observability/audit-logs', label: '审计日志' },
            { key: '/deployment/observability/policies', label: '策略管理' },
            { key: '/deployment/observability/aiops', label: 'AIOps 洞察' },
          ],
        },
      ],
    },
    { key: '/monitor', icon: <AlertOutlined />, label: t('menu.monitor') },
    { key: '/tools', icon: <ToolOutlined />, label: t('menu.tools') },
    ...(governanceMenuEnabled ? [] : [{ key: '/settings', icon: <SettingOutlined />, label: '系统设置' }]),
  ];

  const governanceMenuItems: MenuItem[] =
    governanceMenuEnabled && canReadGovernance
      ? [
          {
            key: '/governance',
            icon: <UserOutlined />,
            label: '访问治理',
            children: [
              { key: '/governance/users', label: '用户管理' },
              { key: '/governance/roles', label: '角色管理' },
              { key: '/governance/permissions', label: '权限列表' },
            ],
          },
        ]
      : [];

  const menuItems = [...baseMenuItems, ...governanceMenuItems];

  const activeMenuKey = React.useMemo(() => {
    // 任务相关页面映射到任务中心
    if (location.pathname.startsWith('/jobs')) return '/tasks';
    // 配置中心旧路由映射
    if (location.pathname.startsWith('/configcenter')) return '/config';
    // K8s 相关页面映射到部署管理
    if (location.pathname.startsWith('/k8s')) return '/deployment';
    // 主机相关页面映射到部署管理>基础设施>主机管理
    if (location.pathname.startsWith('/deployment/infrastructure/hosts')) return '/deployment/infrastructure/hosts';
    // 旧主机路由重定向
    if (location.pathname.startsWith('/hosts')) return '/deployment/infrastructure/hosts';
    // 帮助中心
    if (location.pathname.startsWith('/help')) return '/help';
    if (location.pathname.startsWith('/services')) return '/services';
    // 访问治理子菜单
    if (location.pathname.startsWith('/governance/users')) return '/governance/users';
    if (location.pathname.startsWith('/governance/roles')) return '/governance/roles';
    if (location.pathname.startsWith('/governance/permissions')) return '/governance/permissions';
    return location.pathname;
  }, [location.pathname]);

  const userMenuItems = [
    { key: 'profile', icon: <UserOutlined />, label: '个人中心' },
    { key: 'settings', icon: <SettingOutlined />, label: '系统设置' },
    { type: 'divider' as const },
    { key: 'logout', icon: <LogoutOutlined />, label: '退出登录' },
  ];

  const getBreadcrumbItems = () => {
    const paths = location.pathname.split('/').filter(Boolean);
    const items = [{ title: '首页', path: '/' }];
    let currentPath = '';

    paths.forEach((path) => {
      currentPath += `/${path}`;
      const menuItem = menuItems.find((item) => item && (item as any).key === currentPath) as any;
      if (menuItem?.label) {
        items.push({ title: String(menuItem.label), path: currentPath });
      }
      const governanceRoot = menuItems.find((item) => item && (item as any).key === '/governance') as any;
      if (governanceRoot?.children && Array.isArray(governanceRoot.children)) {
        const child = governanceRoot.children.find((sub: any) => sub && sub.key === currentPath);
        if (child?.label) {
          items.push({ title: String(child.label), path: currentPath });
        }
      }
    });

    return items;
  };

  const handleMenuClick = (key: string) => {
    navigate(menuRouteOverrides[key] || key);
    if (isMobile) {
      setMobileDrawerOpen(false);
    }
  };

  // 3.1.2 & 3.1.3 侧边栏内容
  const sidebarContent = (
    <div className="flex flex-col h-full">
      {/* Logo 区域 */}
      <div className="h-16 flex-shrink-0 flex items-center justify-center border-b border-gray-200 px-4">
        <div className="flex items-center gap-3">
          <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-primary-500 to-primary-600 flex items-center justify-center shadow-sm">
            <CloudOutlined className="text-white text-base" />
          </div>
          {!collapsed && (
            <span className="text-gray-900 font-semibold text-lg">OpsPilot</span>
          )}
        </div>
      </div>

      {/* 菜单 - 可滚动区域 */}
      <div className="flex-1 overflow-y-auto overflow-x-hidden">
        <Menu
          theme="light"
          mode="inline"
          selectedKeys={[activeMenuKey]}
          items={menuItems}
          onClick={({ key }) => handleMenuClick(key)}
          className="border-none mt-2"
          style={{ background: 'transparent' }}
        />
      </div>

      {/* 折叠按钮 (仅桌面端) */}
      {!isMobile && (
        <div className="flex-shrink-0 p-4 border-t border-gray-200">
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
