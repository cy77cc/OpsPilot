/**
 * AI Copilot 抽屉组件
 * 支持场景自动感知与手动切换
 */
import React, { lazy, Suspense } from 'react';
import { Drawer, Skeleton } from 'antd';
import { AISurfaceBoundary } from './AISurfaceBoundary';
import { useResizableDrawer } from './hooks/useResizableDrawer';
import type { SceneOption } from './hooks/useAutoScene';

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

  return (
    <Drawer
      open={open}
      onClose={onClose}
      placement="right"
      size={width}
      closable={false}
      maskClosable
      styles={{
        body: { padding: 0, display: 'flex', flexDirection: 'column', height: '100%' },
        wrapper: { transition: isResizing ? 'none' : undefined },
      }}
      title={null}
    >
      {/* 拖拽手柄 */}
      <div
        style={{
          position: 'absolute',
          left: 0,
          top: 0,
          bottom: 0,
          width: 4,
          cursor: 'ew-resize',
          background: isResizing ? '#1890ff' : 'transparent',
          transition: 'background 0.2s',
        }}
        onMouseDown={handleMouseDown}
      />
      <Suspense fallback={<div style={{ padding: 16 }}><Skeleton active paragraph={{ rows: 4 }} /></div>}>
        <AISurfaceBoundary>
          <CopilotSurface
            open={open}
            onClose={onClose}
            scene={scene}
            selectValue={selectValue}
            onSceneChange={onSceneChange}
            availableScenes={availableScenes}
            isAuto={isAuto}
          />
        </AISurfaceBoundary>
      </Suspense>
    </Drawer>
  );
}

export default AIAssistantDrawer;
