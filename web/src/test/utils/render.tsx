import React from 'react';
import { render, type RenderOptions } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { ConfigProvider } from 'antd';

/**
 * All providers wrapper for testing components that need routing and antd.
 */
const AllProviders: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  return (
    <BrowserRouter>
      <ConfigProvider
        theme={{
          token: {
            colorPrimary: '#1890ff',
          },
        }}
      >
        {children}
      </ConfigProvider>
    </BrowserRouter>
  );
};

/**
 * Custom render function that includes all providers.
 * Use this instead of @testing-library/react's render.
 *
 * @example
 * ```tsx
 * import { renderWithProviders } from '../test/utils/render';
 *
 * it('renders correctly', () => {
 *   renderWithProviders(<MyComponent />);
 * });
 * ```
 */
export function renderWithProviders(
  ui: React.ReactElement,
  options?: Omit<RenderOptions, 'wrapper'>
) {
  return render(ui, { wrapper: AllProviders, ...options });
}

/**
 * Wrapper for testing components that only need antd ConfigProvider.
 */
const AntdProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  return (
    <ConfigProvider
      theme={{
        token: {
          colorPrimary: '#1890ff',
        },
      }}
    >
      {children}
    </ConfigProvider>
  );
};

/**
 * Render with antd provider only (no router).
 */
export function renderWithAntd(
  ui: React.ReactElement,
  options?: Omit<RenderOptions, 'wrapper'>
) {
  return render(ui, { wrapper: AntdProvider, ...options });
}

/**
 * Wrapper for testing components that only need router.
 */
const RouterProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  return <BrowserRouter>{children}</BrowserRouter>;
};

/**
 * Render with router provider only (no antd).
 */
export function renderWithRouter(
  ui: React.ReactElement,
  options?: Omit<RenderOptions, 'wrapper'>
) {
  return render(ui, { wrapper: RouterProvider, ...options });
}

/**
 * Re-export everything from @testing-library/react for convenience.
 */
export * from '@testing-library/react';
export { render };
