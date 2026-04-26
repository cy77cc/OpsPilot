import React from 'react';
import { Row, Col } from 'antd';
import HostBasicInfoCard from '../components/HostBasicInfoCard';
import HostResourceSummaryCard from '../components/HostResourceSummaryCard';
import HostNetworkInfoCard from '../components/HostNetworkInfoCard';
import HostDiskInfoCard from '../components/HostDiskInfoCard';
import HostTrendChartCard from '../components/HostTrendChartCard';
import HostRecentAlertsCard from '../components/HostRecentAlertsCard';
import HostRecentOperationCard from '../components/HostRecentOperationCard';
import HostQuickActionsCard from '../components/HostQuickActionsCard';
import type { Host } from '../../../../api/modules/hosts';

interface OverviewTabProps {
  host: Host | null;
  loading: boolean;
  onAction: (action: string) => void;
  onTabChange: (key: string) => void;
}

const OverviewTab: React.FC<OverviewTabProps> = ({ host, loading, onAction, onTabChange }) => {
  return (
    <div className="flex flex-col gap-4 py-4">
      {/* Row 1: Basic Info & Resource Summary */}
      <Row gutter={16}>
        <Col span={12}>
          <HostBasicInfoCard host={host} loading={loading} />
        </Col>
        <Col span={12}>
          <HostResourceSummaryCard host={host} loading={loading} />
        </Col>
      </Row>

      {/* Row 2: Network, Disk, Trend */}
      <Row gutter={16}>
        <Col span={6}>
          <HostNetworkInfoCard host={host} loading={loading} />
        </Col>
        <Col span={6}>
          <HostDiskInfoCard 
            host={host} 
            loading={loading} 
            onViewDetails={() => onTabChange('disk')} 
          />
        </Col>
        <Col span={12}>
          <HostTrendChartCard loading={loading} />
        </Col>
      </Row>

      {/* Row 3: Alerts, Operations, Quick Actions */}
      <Row gutter={16}>
        <Col span={8}>
          <HostRecentAlertsCard 
            loading={loading} 
            onViewAll={() => onTabChange('alarm')} 
          />
        </Col>
        <Col span={8}>
          <HostRecentOperationCard 
            loading={loading} 
            onViewAll={() => onTabChange('logs')} 
          />
        </Col>
        <Col span={8}>
          <HostQuickActionsCard onAction={onAction} />
        </Col>
      </Row>
    </div>
  );
};

export default OverviewTab;
