import React from 'react';
import Copilot from './Copilot';
import type { SceneOption } from './hooks/useAutoScene';

interface CopilotSurfaceProps {
  open: boolean;
  onClose?: () => void;
  scene: string;
  selectValue?: string;
  onSceneChange?: (scene: string) => void;
  availableScenes?: SceneOption[];
  isAuto?: boolean;
}

/**
 * AI runtime boundary:
 * AppLayout -> AICopilotButton -> AIAssistantDrawer -> CopilotSurface -> Copilot
 * Keep this wrapper as the lazy-load seam so AI-rich rendering failures stay local.
 */
export default function CopilotSurface(props: CopilotSurfaceProps) {
  return <Copilot {...props} />;
}
