import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import HealthCard from './HealthCard';

describe('HealthCard', () => {
  it('renders totals and health percentage', () => {
    render(
      <HealthCard
        title="主机健康"
        data={{ total: 10, healthy: 9, degraded: 1, unhealthy: 0, offline: 0 }}
      />,
    );

    expect(screen.getByText('主机健康')).toBeInTheDocument();
    expect(screen.getByText('10')).toBeInTheDocument();
    expect(screen.getByText('90% 健康')).toBeInTheDocument();
  });

  it('calls onClick when card is clicked', () => {
    const onClick = vi.fn();
    render(
      <HealthCard
        title="服务健康"
        data={{ total: 5, healthy: 4, degraded: 1, unhealthy: 0, offline: 0 }}
        onClick={onClick}
      />,
    );

    fireEvent.click(screen.getByText('服务健康'));
    expect(onClick).toHaveBeenCalledTimes(1);
  });
});
