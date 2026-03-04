import React from 'react';
import { Button, Segmented, Space, Switch, Typography } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import type { TimeRange } from '../../api/modules/dashboard';

interface TimeRangeSelectorProps {
  value: TimeRange;
  autoRefresh: boolean;
  loading?: boolean;
  onChange: (timeRange: TimeRange) => void;
  onRefresh: () => void;
  onAutoRefreshChange: (enabled: boolean) => void;
}

const TimeRangeSelector: React.FC<TimeRangeSelectorProps> = ({
  value,
  autoRefresh,
  loading,
  onChange,
  onRefresh,
  onAutoRefreshChange,
}) => {
  return (
    <Space size={12} wrap>
      <Segmented
        options={[
          { label: '1h', value: '1h' },
          { label: '6h', value: '6h' },
          { label: '24h', value: '24h' },
        ]}
        value={value}
        onChange={(v) => onChange(v as TimeRange)}
      />
      <Button icon={<ReloadOutlined />} onClick={onRefresh} loading={loading}>
        刷新
      </Button>
      <Space size={6}>
        <Typography.Text type="secondary">自动刷新</Typography.Text>
        <Switch checked={autoRefresh} onChange={onAutoRefreshChange} />
      </Space>
    </Space>
  );
};

export default TimeRangeSelector;
