/**
 * AI Copilot 抽屉组件
 * 支持场景自动感知与手动切换
 */
import React, { lazy, Suspense } from 'react';
import { Drawer, Skeleton } from 'antd';
import { AISurfaceBoundary } from './AISurfaceBoundary';
import { useResizableDrawer } from './hooks/useResizableDrawer';
import type { SceneOption } from './hooks/useAutoScene';
import './AIAssistantDrawer.css';

const CopilotSurface = lazy(() => import('./CopilotSurface'));

interface AIAssistantDrawerProps {
  open: boolean;
  onClose: () => void;
  scene: string;
  selectValue?: string;
  onSceneChange?: (scene: string) => void;
  availableScenes?: SceneOption[];
  isAuto?: boolean;
}

/**
 * AI Copilot 抽屉组件
 * 使用 @ant-design/x 和 @ant-design/x-sdk 实现
 */
export function AIAssistantDrawer({
  open,
  onClose,
  scene,
  selectValue,
  onSceneChange,
  availableScenes = [{ key: 'global', label: '全局助手' }],
  isAuto = true,
}: AIAssistantDrawerProps) {
  const { width, isResizing, handleMouseDown } = useResizableDrawer();

  // 拖拽手柄
  const ResizeHandle = (
    <div
      className={`ai-drawer-resize-handle ${isResizing ? 'resizing' : ''}`}
      onMouseDown={handleMouseDown}
    />
  );

  return (
    <Drawer
      open={open}
      onClose={onClose}
      placement="right"
      width={width}
      closable={false}
      maskClosable
      rootClassName="ai-assistant-drawer"
      styles={{
        body: { padding: 0, display: 'flex', flexDirection: 'column', height: '100%' },
        wrapper: { transition: isResizing ? 'none' : undefined },
      }}
      title={null}
    >
      {ResizeHandle}
      <AISurfaceBoundary>
        <Suspense fallback={<div style={{ padding: 16 }}><Skeleton active paragraph={{ rows: 4 }} /></div>}>
          <CopilotSurface
            open={open}
            onClose={onClose}
            scene={scene}
            selectValue={selectValue}
            onSceneChange={onSceneChange}
            availableScenes={availableScenes}
            isAuto={isAuto}
          />
        </Suspense>
      </AISurfaceBoundary>
    </Drawer>
  );
}

export default AIAssistantDrawer;
