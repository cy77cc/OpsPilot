import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, waitFor } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { NotificationProvider } from '../../contexts/NotificationContext';
import { NotificationPanel } from '../../components/Notification';

// Mock window.innerWidth
const setViewportWidth = (width: number) => {
  Object.defineProperty(window, 'innerWidth', {
    writable: true,
    configurable: true,
    value: width,
  });
  window.dispatchEvent(new Event('resize'));
};

// Mock notification API
vi.mock('../../api/modules/notification', () => ({
  notificationApi: {
    getNotifications: vi.fn().mockResolvedValue({
      data: { list: [], total: 0 },
    }),
    getUnreadCount: vi.fn().mockResolvedValue({
      data: {
        total: 0,
        by_type: { alert: 0, task: 0, system: 0, approval: 0 },
        by_severity: { critical: 0, warning: 0, info: 0 },
      },
    }),
    markAsRead: vi.fn().mockResolvedValue({ success: true }),
    dismiss: vi.fn().mockResolvedValue({ success: true }),
    confirm: vi.fn().mockResolvedValue({ success: true }),
    markAllAsRead: vi.fn().mockResolvedValue({ success: true }),
  },
}));

// Mock browser notification
vi.mock('../../utils/browserNotification', () => ({
  notify: vi.fn(),
  isBrowserNotificationEnabled: vi.fn(() => false),
}));

const renderWithProviders = (component: React.ReactNode) => {
  return render(
    <BrowserRouter>
      <NotificationProvider>
        {component}
      </NotificationProvider>
    </BrowserRouter>
  );
};

describe('NotificationPanel Responsive', () => {
  const originalInnerWidth = window.innerWidth;

  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    setViewportWidth(originalInnerWidth);
  });

  it('renders correctly on desktop (1024px)', async () => {
    setViewportWidth(1024);
    renderWithProviders(<NotificationPanel />);

    await waitFor(() => {
      const panel = document.querySelector('.notification-panel');
      expect(panel).toBeInTheDocument();
    });
  });

  it('renders correctly on tablet (768px)', async () => {
    setViewportWidth(768);
    renderWithProviders(<NotificationPanel />);

    await waitFor(() => {
      const panel = document.querySelector('.notification-panel');
      expect(panel).toBeInTheDocument();
    });
  });

  it('renders correctly on mobile (375px)', async () => {
    setViewportWidth(375);
    renderWithProviders(<NotificationPanel />);

    await waitFor(() => {
      const panel = document.querySelector('.notification-panel');
      expect(panel).toBeInTheDocument();
    });
  });

  it('adjusts tab layout on small screens', async () => {
    setViewportWidth(375);
    renderWithProviders(<NotificationPanel />);

    await waitFor(() => {
      const tabs = document.querySelector('.notification-panel-tabs');
      expect(tabs).toBeInTheDocument();
    });
  });
});
