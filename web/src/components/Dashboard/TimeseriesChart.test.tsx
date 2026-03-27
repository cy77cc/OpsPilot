import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import TimeseriesChart from './TimeseriesChart';

vi.mock('@ant-design/charts', () => ({
  Line: ({ data }: { data: Array<any> }) => <div data-testid="line-chart">{data.length}</div>,
}));

describe('TimeseriesChart', () => {
  it('shows empty state when no data', () => {
    render(<TimeseriesChart title="CPU 使用率" series={[]} loading={false} />);
    expect(screen.getByText('暂无数据')).toBeInTheDocument();
  });

  it('renders line chart when data is available', () => {
    render(
      <TimeseriesChart
        title="内存使用率"
        series={[
          {
            hostId: 1,
            hostName: 'host-1',
            data: [
              { timestamp: '2026-03-04T00:00:00Z', value: 33 },
              { timestamp: '2026-03-04T00:01:00Z', value: 35 },
            ],
          },
        ]}
        loading={false}
      />,
    );

    expect(screen.getByTestId('line-chart')).toBeInTheDocument();
  });
});
