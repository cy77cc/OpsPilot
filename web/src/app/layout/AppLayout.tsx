import React, { useEffect, useState } from 'react';
import { Layout, Menu, Avatar, Dropdown, Input, Tooltip, Button, Drawer, Switch } from 'antd';
import type { MenuProps } from 'antd';
import {
  LayoutDashboard,
  HardDrive,
  Settings,
  Cloud,
  PanelLeftClose,
  PanelLeftOpen,
  Search,
  User,
  LogOut,
  HelpCircle,
  Box,
  Menu as MenuIcon,
} from 'lucide-react';
import SparklesIcon from '../../components/common/SparklesIcon';
import { useLocation, useNavigate } from 'react-router-dom';
import { useAuth } from '../../components/Auth/AuthContext';
import ProjectSwitcher from '../../components/Project/ProjectSwitcher';
import { NotificationBell } from '../../components/Notification';
import { AICopilotButton, AISurfaceBoundary, CopilotSurface } from '../../components/AI';
import '../../components/Notification/notification.css';
import { useI18n } from '../../i18n';
import { usePermission } from '../../components/RBAC';
import CommandPalette from '../../components/CommandPalette';
import KeyboardShortcutsHelp from '../../components/KeyboardShortcutsHelp';
import { useKeyboardShortcuts } from '../../hooks/useKeyboardShortcuts';
import PageTransition from '../../components/PageTransition';
import { buildMenuSections } from './navigation.config';
import {
  findSectionPath,
  getActiveMenuKey,
  getBreadcrumbItems,
  menuRouteOverrides,
} from './navigation.helpers';

const { Header, Sider, Content } = Layout;
type MenuItem = Required<MenuProps>['items'][number];

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

  // Expose setCopilotOpen globally for QuickAccess and other components
  useEffect(() => {
    (window as any).openCopilot = () => setCopilotOpen(true);
    return () => {
      delete (window as any).openCopilot;
    };
  }, []);

  const [aiFormAssistEnabled, setAiFormAssistEnabled] = useState(
    typeof window !== 'undefined' ? localStorage.getItem('ai-form-assist-enabled') !== '0' : true
  );

  const toggleAiFormAssist = (enabled: boolean) => {
    setAiFormAssistEnabled(enabled);
    localStorage.setItem('ai-form-assist-enabled', enabled ? '1' : '0');
    // Force a reload or notify hooks if needed, but since it reads from localStorage 
    // on every render in the current implementation, it might just work if the component tree re-renders.
    // For a cleaner approach, a Context would be better, but this satisfies "Help me open".
    window.dispatchEvent(new Event('storage')); // Notify other tabs/hooks
  };

  const navigate = useNavigate();
  const location = useLocation();
  const { logout } = useAuth();
  const { hasPermission } = usePermission();
  const { t, lang, setLang } = useI18n();
  const governanceMenuEnabled = import.meta.env.VITE_FEATURE_GOVERNANCE_MENU !== 'false';
  const canReadGovernance = hasPermission('rbac', 'read');

  useKeyboardShortcuts({
    onOpenHelp: () => setHelpOpen(true),
    enableNavigation: true,
    enableListNavigation: false,
  });

  useEffect(() => {
    const checkMobile = () => {
      setIsMobile(window.innerWidth < 768);
    };
    checkMobile();
    window.addEventListener('resize', checkMobile);
    return () => window.removeEventListener('resize', checkMobile);
  }, []);

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

  const menuSections = React.useMemo(
    () =>
      buildMenuSections({
        t,
        governanceMenuEnabled,
        canReadGovernance,
      }),
    [t, governanceMenuEnabled, canReadGovernance],
  );

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

  const activeMenuKey = React.useMemo(() => getActiveMenuKey(location.pathname), [location.pathname]);
  const menuPath = React.useMemo(() => findSectionPath(menuSections, activeMenuKey), [menuSections, activeMenuKey]);
  const breadcrumbItems = React.useMemo(() => getBreadcrumbItems(menuPath), [menuPath]);

  const userMenuItems = [
    { key: 'profile', icon: <User size={16} />, label: '个人中心' },
    { key: 'settings', icon: <Settings size={16} />, label: '系统设置' },
    { type: 'divider' as const },
    { key: 'logout', icon: <LogOut size={16} />, label: '退出登录' },
  ];

  const handleMenuClick = (key: string) => {
    if (!key.startsWith('/')) {
      return;
    }
    navigate(menuRouteOverrides[key] || key);
    if (isMobile) {
      setMobileDrawerOpen(false);
    }
  };

  const sidebarContent = (
    <div className="flex flex-col h-full">
      <div className="h-14 flex-shrink-0 flex items-center justify-center border-b border-gray-200 px-3">
        <div className="flex items-center gap-2.5">
          <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-primary-500 to-primary-600 flex items-center justify-center shadow-sm">
            <Cloud size={18} className="text-white" />
          </div>
          {!collapsed && <span className="text-gray-900 font-semibold text-lg">OpsPilot</span>}
        </div>
      </div>

      <div className="flex-1 overflow-y-auto overflow-x-hidden py-1">
        <Menu
          theme="light"
          mode="inline"
          selectedKeys={[activeMenuKey]}
          items={menuItems}
          onClick={({ key }) => handleMenuClick(key)}
          className="border-none mt-0 [&_.ant-menu-item-group-title]:px-3 [&_.ant-menu-item-group-title]:pb-0.5 [&_.ant-menu-item-group-title]:pt-2 [&_.ant-menu-item-group-title]:text-[10px] [&_.ant-menu-item-group-title]:font-semibold [&_.ant-menu-item-group-title]:uppercase [&_.ant-menu-item-group-title]:tracking-[0.08em] [&_.ant-menu-item-group-title]:text-gray-400 [&_.ant-menu-item-group-list]:space-y-0 [&_.ant-menu-item]:mx-2 [&_.ant-menu-item]:my-0 [&_.ant-menu-item]:h-8 [&_.ant-menu-item]:leading-8 [&_.ant-menu-item]:px-3 [&_.ant-menu-item_.ant-menu-item-icon]:mr-1.5 [&_.ant-menu-item_.ant-menu-item-icon]:text-sm [&_.ant-menu-item-selected]:bg-blue-600! [&_.ant-menu-item-selected]:text-white! [&_.ant-menu-item-selected]:rounded-lg"
          style={{ background: 'transparent' }}
        />
      </div>

      {!isMobile && (
        <div className="flex-shrink-0 border-t border-gray-200">
          <div className="flex items-center px-1 py-1">
            <Button
              type="text"
              icon={<Settings size={18} />}
              onClick={() => navigate('/settings')}
              className={`flex-1 flex items-center justify-start h-9 text-gray-500 hover:text-gray-900 hover:bg-gray-100 ${collapsed ? 'justify-center px-0' : 'px-3'}`}
            >
              {!collapsed && <span className="ml-2 text-sm font-medium">系统设置</span>}
            </Button>
            <Button
              type="text"
              icon={collapsed ? <PanelLeftOpen size={18} /> : <PanelLeftClose size={18} />}
              onClick={() => setCollapsed(!collapsed)}
              className="w-10 h-9 text-gray-400 hover:text-gray-900 flex-shrink-0"
            />
          </div>
        </div>
      )}
    </div>
  );

  return (
    <Layout className="h-screen overflow-hidden">
      <CommandPalette open={commandPaletteOpen} onOpenChange={setCommandPaletteOpen} />
      <KeyboardShortcutsHelp open={helpOpen} onClose={() => setHelpOpen(false)} />

      {isMobile ? (
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
        <Sider
          trigger={null}
          collapsible
          collapsed={collapsed}
          width={240}
          theme="light"
          className="fixed left-0 top-0 bottom-0 z-50 shadow-sm"
          style={{
            background: '#ffffff',
            borderRight: '1px solid #f0f0f0',
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
          height: '100vh',
          display: 'flex',
          flexDirection: 'column',
        }}
      >
        <Header
          className="h-16 flex-shrink-0 px-4 md:px-6 flex items-center justify-between bg-white shadow-sm"
          style={{
            position: 'sticky',
            top: 0,
            zIndex: 40,
            borderBottom: '1px solid #f0f0f0',
          }}
        >
          <div className="flex items-center gap-4">
            {isMobile && (
              <Button
                type="text"
                icon={<MenuIcon size={20} />}
                onClick={() => setMobileDrawerOpen(true)}
                className="text-gray-600"
              />
            )}

            {!isMobile && breadcrumbItems.length > 0 && (
              <div className="flex items-baseline gap-2">
                <h1 className="text-xl font-semibold text-gray-900 m-0">
                  {breadcrumbItems[breadcrumbItems.length - 1].title}
                </h1>
                <span className="text-xs font-normal text-gray-400 whitespace-nowrap">
                  | {menuPath.map(m => m.title).join(' / ')}
                </span>
              </div>
            )}
          </div>

          <div className="flex items-center gap-2 md:gap-3">
            {!isMobile && <ProjectSwitcher />}

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

            {!isMobile && (
              <Input
                placeholder="搜索资源、应用、文档、命令..."
                prefix={<Search size={18} className="text-gray-400" />}
                suffix={
                  <div className="flex items-center gap-1 bg-gray-100 border border-gray-200 px-1.5 py-0.5 rounded text-[10px] text-gray-400 font-sans font-medium">
                    <span className="text-[12px]">⌘</span> K
                  </div>
                }
                className="w-80 cursor-pointer"
                readOnly
                onClick={() => setCommandPaletteOpen(true)}
                style={{
                  background: '#f8f9fa',
                  border: '1px solid #e9ecef',
                  borderRadius: '8px',
                }}
              />
            )}

            <Tooltip title="帮助文档 <kbd className='ml-1 text-xs'>?</kbd>">
              <Button
                type="text"
                icon={<HelpCircle size={20} />}
                className="text-gray-600 hover:text-primary-600"
                onClick={() => setHelpOpen(true)}
              />
            </Tooltip>

            <Tooltip title={aiFormAssistEnabled ? "关闭 AI 表单辅助" : "开启 AI 表单辅助"}>
              <div className="flex items-center gap-2 px-2 py-1 rounded-lg hover:bg-gray-100 transition-colors">
                <SparklesIcon active={aiFormAssistEnabled} />
                <Switch 
                  size="small" 
                  checked={aiFormAssistEnabled} 
                  onChange={toggleAiFormAssist} 
                />
              </div>
            </Tooltip>

            <NotificationBell onViewAll={() => navigate('/monitor')} />
            <AICopilotButton onOpen={() => setCopilotOpen(true)} />

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
                icon={<User size={18} />}
              />
            </Dropdown>
          </div>
        </Header>

        {isMobile && (
          <div className="flex-shrink-0 fixed bottom-0 left-0 right-0 h-16 bg-white border-t border-gray-200 flex items-center justify-around z-50 shadow-lg">
            <Button
              type="text"
              icon={<LayoutDashboard size={20} />}
              onClick={() => navigate('/')}
              className={location.pathname === '/' ? 'text-primary-600' : 'text-gray-600'}
            />
            <Button
              type="text"
              icon={<Box size={20} />}
              onClick={() => navigate('/services')}
              className={location.pathname.startsWith('/services') ? 'text-primary-600' : 'text-gray-600'}
            />
            <Button
              type="text"
              icon={<HardDrive size={20} />}
              onClick={() => navigate('/hosts')}
              className={location.pathname.startsWith('/hosts') ? 'text-primary-600' : 'text-gray-600'}
            />
            <Button
              type="text"
              icon={<Settings size={20} />}
              onClick={() => navigate('/settings')}
              className={location.pathname.startsWith('/settings') ? 'text-primary-600' : 'text-gray-600'}
            />
          </div>
        )}

        <Content
          className="bg-gray-50 flex-1 overflow-y-auto overflow-x-hidden relative"
          style={{
            height: isMobile ? 'calc(100vh - 128px)' : 'calc(100vh - 64px)',
          }}
        >
          <div className="flex min-h-full w-full flex-col p-6 pb-20 md:pb-6">
            <PageTransition>{children}</PageTransition>
          </div>
        </Content>
      </Layout>

      <AISurfaceBoundary>
        <CopilotSurface open={copilotOpen} onClose={() => setCopilotOpen(false)} />
      </AISurfaceBoundary>
    </Layout>
  );
};

export default AppLayout;
