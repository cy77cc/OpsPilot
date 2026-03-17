import React from 'react';
import { Alert } from 'antd';

interface AISurfaceBoundaryProps {
  children: React.ReactNode;
}

interface AISurfaceBoundaryState {
  hasError: boolean;
}

export class AISurfaceBoundary extends React.Component<
  AISurfaceBoundaryProps,
  AISurfaceBoundaryState
> {
  state: AISurfaceBoundaryState = {
    hasError: false,
  };

  static getDerivedStateFromError(): AISurfaceBoundaryState {
    return { hasError: true };
  }

  componentDidCatch(error: Error) {
    if (import.meta.env.DEV) {
      // Keep the failure local to the AI surface without crashing the shell.
      console.error('AI surface failed to render', error);
    }
  }

  render() {
    if (this.state.hasError) {
      return (
        <div data-testid="ai-surface-fallback" style={{ padding: 16 }}>
          <Alert
            type="error"
            showIcon
            message="AI Copilot unavailable"
            description="The assistant surface failed to load. The rest of the app is still available."
          />
        </div>
      );
    }

    return this.props.children;
  }
}
