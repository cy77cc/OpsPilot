import React from 'react';
import { describe, expect, it } from 'vitest';
import { renderObservabilityRoutes } from './observability.routes';

function collectRoutePaths(node: React.ReactNode): string[] {
  const paths: string[] = [];

  const visit = (child: React.ReactNode) => {
    if (!React.isValidElement(child)) {
      return;
    }
    const props = child.props as { path?: string; children?: React.ReactNode };
    if (props.path) {
      paths.push(props.path);
    }
    if (props.children) {
      React.Children.forEach(props.children, visit);
    }
  };

  React.Children.forEach(node as any, visit);
  return paths;
}

describe('renderObservabilityRoutes', () => {
  it('registers monitor config routes', () => {
    const withAuthStub = (_resource: string, _action: string, element: React.ReactElement) => element;
    const routes = renderObservabilityRoutes(withAuthStub);
    const paths = collectRoutePaths(routes);

    expect(paths).toEqual(expect.arrayContaining([
      '/observability/monitor',
      'rules',
      'channels',
      'routing',
      'deliveries',
    ]));
  });
});
