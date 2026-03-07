import React from 'react';
import { Alert, Button } from 'antd';

interface AISurfaceBoundaryProps {
  children: React.ReactNode;
}

interface AISurfaceBoundaryState {
  hasError: boolean;
}

class AISurfaceErrorBoundary extends React.Component<AISurfaceBoundaryProps, AISurfaceBoundaryState> {
  state: AISurfaceBoundaryState = { hasError: false };

  static getDerivedStateFromError(): AISurfaceBoundaryState {
    return { hasError: true };
  }

  componentDidCatch(error: Error) {
    console.error('AI surface failed to render', error);
  }

  private handleRetry = () => {
    this.setState({ hasError: false });
  };

  render() {
    if (this.state.hasError) {
      return (
        <div style={{ padding: 16 }}>
          <Alert
            type="warning"
            showIcon
            title="AI 助手暂时不可用"
            description="你仍然可以继续使用当前页面，其它功能不受影响。"
            action={(
              <Button size="small" onClick={this.handleRetry}>
                重试
              </Button>
            )}
          />
        </div>
      );
    }

    return this.props.children;
  }
}

export function AISurfaceBoundary({ children }: AISurfaceBoundaryProps) {
  return <AISurfaceErrorBoundary>{children}</AISurfaceErrorBoundary>;
}

export default AISurfaceBoundary;
