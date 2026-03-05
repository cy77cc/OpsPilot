import React from 'react';
import { Button, Drawer, Segmented, Space, Tabs, Typography } from 'antd';
import { MessageOutlined } from '@ant-design/icons';
import { useLocation } from 'react-router-dom';
import ChatInterface from './ChatInterface';
import CommandPanel from './CommandPanel';
import './ai-assistant.css';

const { Text } = Typography;
const MemoChatInterface = React.memo(ChatInterface);
const MemoCommandPanel = React.memo(CommandPanel);
const DRAWER_WIDTH_STORAGE_KEY = 'ai:assistant:drawer:width';
const MOBILE_BREAKPOINT = 768;
const DRAWER_MIN_WIDTH = 620;
const DRAWER_MAX_GUTTER = 24;

const getInitialDrawerWidth = (): number => {
  const maxWidth = window.innerWidth - DRAWER_MAX_GUTTER;
  const fallback = Math.min(920, maxWidth);
  const saved = Number(localStorage.getItem(DRAWER_WIDTH_STORAGE_KEY));
  if (Number.isFinite(saved) && saved > 0) {
    return Math.min(maxWidth, Math.max(DRAWER_MIN_WIDTH, saved));
  }
  return Math.min(maxWidth, Math.max(DRAWER_MIN_WIDTH, fallback));
};

const clampDrawerWidth = (width: number, viewportWidth: number): number => {
  const max = Math.max(DRAWER_MIN_WIDTH, viewportWidth - DRAWER_MAX_GUTTER);
  return Math.min(max, Math.max(DRAWER_MIN_WIDTH, width));
};

interface GlobalAIAssistantProps {
  inlineTrigger?: boolean;
}

const sceneFromPath = (pathname: string): string => {
  const segments = pathname.split('/').filter(Boolean);

  // 首页
  if (segments.length === 0) return 'scene:home';

  // 部署管理二级场景
  if (segments[0] === 'deployment') {
    if (segments[1] === 'infrastructure') {
      if (segments[2] === 'clusters') return 'scene:deployment:clusters';
      if (segments[2] === 'credentials') return 'scene:deployment:credentials';
      if (segments[2] === 'hosts') return 'scene:deployment:hosts';
    }
    if (segments[1] === 'targets') return 'scene:deployment:targets';
    if (segments[1] === 'approvals') return 'scene:deployment:approvals';
    if (segments[1] === 'observability') {
      if (segments[2] === 'topology') return 'scene:deployment:topology';
      if (segments[2] === 'metrics') return 'scene:deployment:metrics';
      if (segments[2] === 'audit-logs') return 'scene:deployment:audit';
      if (segments[2] === 'aiops') return 'scene:deployment:aiops';
    }
    return 'scene:deployment:releases';
  }

  // 服务管理二级场景
  if (segments[0] === 'services') {
    if (segments[1] === 'provision') return 'scene:services:provision';
    if (segments[1] === 'catalog') return 'scene:services:catalog';
    if (segments[2] === 'deploy') return 'scene:services:deploy';
    if (segments[1]) return 'scene:services:detail';
    return 'scene:services:list';
  }

  // 治理管理二级场景
  if (segments[0] === 'governance') {
    if (segments[1] === 'users') return 'scene:governance:users';
    if (segments[1] === 'roles') return 'scene:governance:roles';
    if (segments[1] === 'permissions') return 'scene:governance:permissions';
    return 'scene:governance';
  }

  // 一级场景扩展
  if (['configcenter', 'jobs', 'cicd', 'cmdb', 'automation'].includes(segments[0])) {
    return `scene:${segments[0]}`;
  }

  // 其他一级路由
  return `scene:${segments[0]}`;
};

const parseNumericFromPath = (pathname: string, pattern: RegExp): number | undefined => {
  const matched = pathname.match(pattern);
  if (!matched || !matched[1]) {
    return undefined;
  }
  const value = Number(matched[1]);
  return Number.isFinite(value) && value > 0 ? value : undefined;
};

const buildRuntimeContext = (pathname: string, search: string, scene: string): Record<string, any> => {
  const params = new URLSearchParams(search);
  const pageData: Record<string, any> = { scene };

  const pickNumber = (key: string): number | undefined => {
    const raw = params.get(key);
    if (!raw) return undefined;
    const v = Number(raw);
    return Number.isFinite(v) && v > 0 ? v : undefined;
  };

  pageData.cluster_id = pickNumber('cluster_id') ?? parseNumericFromPath(pathname, /\/clusters\/(\d+)/);
  pageData.service_id = pickNumber('service_id') ?? parseNumericFromPath(pathname, /\/services\/(\d+)/);
  pageData.host_id = pickNumber('host_id') ?? parseNumericFromPath(pathname, /\/hosts\/(\d+)/);
  pageData.target_id = pickNumber('target_id') ?? parseNumericFromPath(pathname, /\/targets\/(\d+)/);
  pageData.namespace = params.get('namespace') || undefined;
  pageData.env = params.get('env') || undefined;

  const selectedItems = (window as any).__AI_SELECTED_ITEMS__;

  const runtime: Record<string, any> = {
    scene,
    pageData,
    selectedItems: selectedItems && typeof selectedItems === 'object' ? selectedItems : undefined,
  };
  // 兼容后端当前平铺读取逻辑
  Object.assign(runtime, pageData);
  return runtime;
};

const GlobalAIAssistant: React.FC<GlobalAIAssistantProps> = ({ inlineTrigger = false }) => {
  const [open, setOpen] = React.useState(false);
  const [isClosing, setIsClosing] = React.useState(false);
  const [scope, setScope] = React.useState<'scene' | 'global'>('scene');
  const [tabKey, setTabKey] = React.useState<'chat' | 'command'>('chat');
  const [viewportWidth, setViewportWidth] = React.useState(() => window.innerWidth);

  // 使用 ref 存储宽度，避免频繁 setState 导致重渲染
  const drawerWidthRef = React.useRef<number>(getInitialDrawerWidth());

  // 用于触发重渲染的状态（仅在需要时更新）
  const [, forceUpdate] = React.useReducer((x) => x + 1, 0);

  const location = useLocation();
  const isMobile = viewportWidth < MOBILE_BREAKPOINT;
  const pageScene = React.useMemo(() => sceneFromPath(location.pathname), [location.pathname]);
  const currentScene = scope === 'global' ? 'global' : pageScene;
  const runtimeContext = React.useMemo(
    () => buildRuntimeContext(location.pathname, location.search, currentScene),
    [location.pathname, location.search, currentScene],
  );
  const resizingRef = React.useRef<{ startX: number; startWidth: number } | null>(null);
  const closeRafRef = React.useRef<number | null>(null);

  // 记住最后一次的 scene，避免切换时重新加载
  const lastSceneRef = React.useRef<string>(currentScene);
  const [activeScene, setActiveScene] = React.useState(currentScene);

  // 只在 scope 或页面变化时更新 scene
  React.useEffect(() => {
    if (open && currentScene !== lastSceneRef.current) {
      lastSceneRef.current = currentScene;
      setActiveScene(currentScene);
    }
  }, [open, currentScene]);

  React.useEffect(() => {
    const onResize = () => {
      const nextViewportWidth = window.innerWidth;
      setViewportWidth(nextViewportWidth);
      // 自动调整宽度到有效范围
      const newWidth = clampDrawerWidth(drawerWidthRef.current, nextViewportWidth);
      if (newWidth !== drawerWidthRef.current) {
        drawerWidthRef.current = newWidth;
        forceUpdate();
      }
    };
    window.addEventListener('resize', onResize);
    return () => window.removeEventListener('resize', onResize);
  }, []);

  React.useEffect(() => () => {
    if (closeRafRef.current !== null) {
      cancelAnimationFrame(closeRafRef.current);
    }
  }, []);

  const handleResizeStart = (event: React.MouseEvent<HTMLDivElement>) => {
    if (isMobile) return;
    event.preventDefault();
    const startWidth = drawerWidthRef.current;
    resizingRef.current = { startX: event.clientX, startWidth };

    const onMouseMove = (moveEvent: MouseEvent) => {
      const state = resizingRef.current;
      if (!state) return;

      const delta = state.startX - moveEvent.clientX;
      const newWidth = clampDrawerWidth(state.startWidth + delta, window.innerWidth);

      // 直接更新 DOM 宽度，避免 React 重渲染
      const wrapper = document.querySelector('.ai-assistant-drawer .ant-drawer-content-wrapper') as HTMLElement;
      if (wrapper) {
        wrapper.style.width = `${newWidth}px`;
      }
      drawerWidthRef.current = newWidth;
    };

    const onMouseUp = () => {
      localStorage.setItem(DRAWER_WIDTH_STORAGE_KEY, String(drawerWidthRef.current));
      resizingRef.current = null;
      document.body.style.userSelect = '';
      forceUpdate();
      window.removeEventListener('mousemove', onMouseMove);
      window.removeEventListener('mouseup', onMouseUp);
    };

    document.body.style.userSelect = 'none';
    window.addEventListener('mousemove', onMouseMove);
    window.addEventListener('mouseup', onMouseUp);
  };

  const tabItems = React.useMemo(() => ([
    {
      key: 'chat',
      label: '对话',
      children: (
        <div className="ai-assistant-tabpane-wrap">
          <MemoChatInterface className="ai-chat-interface" scene={activeScene} runtimeContext={runtimeContext} />
        </div>
      ),
    },
    {
      key: 'command',
      label: '命令中心',
      children: (
        <div className="ai-assistant-tabpane-wrap">
          <MemoCommandPanel scene={activeScene} />
        </div>
      ),
    },
  ]), [activeScene]);

  // 打开时更新 scene
  const handleOpen = () => {
    setIsClosing(false);
    setActiveScene(currentScene);
    lastSceneRef.current = currentScene;
    setOpen(true);
  };

  const handleClose = () => {
    if (isClosing) return;
    setIsClosing(true);
    // 延迟关闭，让动画先执行
    setTimeout(() => {
      setOpen(false);
      setIsClosing(false);
    }, 200); // 匹配 CSS 动画时长
  };

  return (
    <>
      <Button
        type={inlineTrigger ? 'default' : 'primary'}
        shape={inlineTrigger ? 'default' : 'circle'}
        size={inlineTrigger ? 'middle' : 'large'}
        icon={<MessageOutlined />}
        style={inlineTrigger ? undefined : { position: 'fixed', right: 28, bottom: 28, zIndex: 1000, boxShadow: '0 10px 30px rgba(0,0,0,0.2)' }}
        onClick={handleOpen}
      >
        {inlineTrigger ? 'AI助手' : null}
      </Button>
      <Drawer
        rootClassName={`ai-assistant-drawer${isClosing ? ' ai-assistant-drawer-closing' : ''}`}
        title={<Space><MessageOutlined /><Text>AI 助手</Text></Space>}
        open={open}
        onClose={handleClose}
        maskClosable={!isClosing}
        keyboard={!isClosing}
        afterOpenChange={(nextOpen) => {
          if (!nextOpen) {
            setIsClosing(false);
          }
        }}
        width={isMobile ? '100vw' : drawerWidthRef.current}
        styles={{
          body: {
            position: 'relative',
            padding: 12,
            height: 'calc(100vh - 56px)',
            overflow: 'hidden',
            display: 'flex',
            flexDirection: 'column'
          },
          wrapper: {
            willChange: 'transform',
          } as React.CSSProperties,
        }}
        destroyOnClose={false}
        extra={(
          <Segmented
            size="small"
            value={scope}
            options={[
              { label: '当前场景', value: 'scene' },
              { label: '全局', value: 'global' },
            ]}
            onChange={(v) => setScope(v as 'scene' | 'global')}
          />
        )}
      >
        {!isMobile ? (
          <div
            className="ai-assistant-resize-handle"
            onMouseDown={handleResizeStart}
            role="separator"
            aria-orientation="vertical"
            aria-label="调整 AI 助手宽度"
          />
        ) : null}
        <div className={`ai-assistant-layout${isClosing ? ' is-closing' : ''}`}>
          {/* <div className="ai-assistant-hero">
            <Text className="ai-assistant-hero-title">智能运维助手</Text>
            <Text type="secondary" className="ai-assistant-hero-subtitle">
              对话诊断、命令预览、确认执行与历史回放
            </Text>
          </div> */}
          <div className="ai-assistant-chat-wrap">
            <Tabs
              className="ai-assistant-tabs"
              activeKey={tabKey}
              onChange={(v) => setTabKey(v as 'chat' | 'command')}
              items={tabItems}
            />
          </div>
        </div>
      </Drawer>
    </>
  );
};

export default GlobalAIAssistant;
