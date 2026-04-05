import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Alert, Button, Card, Space, Table, Tag, Typography, message } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useParams } from 'react-router-dom';
import { Api } from '../../../api';
import type { Cluster } from '../../../api/modules/cluster';
import type { Phase3OperationResponse, Phase3SecurityAlert } from '../../../api/modules/cluster.phase3';

const { Text, Title } = Typography;

const stateColorMap: Record<string, string> = {
  success: 'green',
  warning: 'gold',
  pending: 'blue',
  error: 'red',
};

const ClusterSecurityCenterPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const clusterId = Number(id);

  const [cluster, setCluster] = useState<Cluster | null>(null);
  const [alerts, setAlerts] = useState<Phase3SecurityAlert[]>([]);
  const [loading, setLoading] = useState(false);
  const [containingId, setContainingId] = useState<number | null>(null);
  const [lastContainResult, setLastContainResult] = useState<Phase3OperationResponse | null>(null);

  const loadCluster = useCallback(async () => {
    if (!clusterId) return;
    try {
      const res = await Api.cluster.getClusterDetail(clusterId);
      setCluster(res.data);
    } catch {
      setCluster(null);
    }
  }, [clusterId]);

  const loadAlerts = useCallback(async () => {
    if (!clusterId) return;
    setLoading(true);
    try {
      const res = await Api.phase3.listSecurityAlerts(clusterId);
      setAlerts(res.data?.list || []);
    } catch (err) {
      setAlerts([]);
      message.error(err instanceof Error ? err.message : '加载安全告警失败');
    } finally {
      setLoading(false);
    }
  }, [clusterId]);

  useEffect(() => {
    loadCluster();
    loadAlerts();
  }, [loadCluster, loadAlerts]);

  const handleContain = useCallback(async (alertId: number) => {
    if (!clusterId) return;
    setContainingId(alertId);
    try {
      const res = await Api.phase3.containAlert(clusterId, alertId);
      setLastContainResult(res.data);
      if (res.data.ui_state === 'warning') {
        message.warning('Containment downgraded to suggest_only, manual action required.');
      } else if (res.data.ui_state === 'pending') {
        message.info('Containment requires approval.');
      } else if (res.data.ui_state === 'success') {
        message.success('Containment executed.');
      } else {
        message.error(res.data.message || 'Containment failed.');
      }
      await loadAlerts();
    } catch (err) {
      message.error(err instanceof Error ? err.message : 'Containment failed.');
    } finally {
      setContainingId(null);
    }
  }, [clusterId, loadAlerts]);

  const columns: ColumnsType<Phase3SecurityAlert> = useMemo(() => [
    {
      title: '告警 ID',
      dataIndex: 'id',
      width: 90,
    },
    {
      title: '命名空间',
      dataIndex: 'namespace',
      render: (value: string | undefined) => value || '-',
    },
    {
      title: '工作负载',
      dataIndex: 'workload',
      render: (value: string | undefined) => value || '-',
    },
    {
      title: '严重级别',
      dataIndex: 'severity',
      render: (value: string | undefined) => (
        <Tag color={value === 'critical' ? 'red' : value === 'high' ? 'volcano' : 'blue'}>{value || 'unknown'}</Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'dispose_status',
      render: (value: string | undefined) => value || 'pending',
    },
    {
      title: '操作',
      key: 'action',
      render: (_, record) => (
        <Button
          size="small"
          type="primary"
          loading={containingId === record.id}
          onClick={() => handleContain(record.id)}
        >
          Contain
        </Button>
      ),
    },
  ], [containingId, handleContain]);

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Card>
        <Title level={4} style={{ marginBottom: 8 }}>集群安全中心</Title>
        <Space size={8}>
          <Text type="secondary">Cluster</Text>
          <Text strong>{cluster?.name || `#${clusterId}`}</Text>
          <Tag color={cluster?.source === 'external_managed' ? 'gold' : 'green'}>
            {cluster?.source || 'unknown'}
          </Tag>
        </Space>
      </Card>

      {lastContainResult ? (
        <Alert
          type={lastContainResult.ui_state === 'error' ? 'error' : lastContainResult.ui_state === 'warning' ? 'warning' : 'success'}
          showIcon
          message={`Containment Result: ${lastContainResult.code}`}
          description={(
            <Space direction="vertical" size={4}>
              <Tag color={stateColorMap[lastContainResult.ui_state] || 'default'}>{lastContainResult.ui_state}</Tag>
              <Text data-testid="contain-mode">{(lastContainResult.result as any)?.mode || '-'}</Text>
              {(lastContainResult.result as any)?.mode === 'suggest_only' ? (
                <Text data-testid="manual-suggestion">external_managed containment is manual suggestion</Text>
              ) : null}
            </Space>
          )}
        />
      ) : null}

      <Card title="Runtime Security Alerts">
        <Table<Phase3SecurityAlert>
          rowKey="id"
          columns={columns}
          dataSource={alerts}
          loading={loading}
          pagination={false}
        />
      </Card>
    </Space>
  );
};

export default ClusterSecurityCenterPage;
