import React, { useEffect, useState } from 'react';
import { Alert, Button, Card, Descriptions, Empty, Space, Table, Tag, Typography, message } from 'antd';
import { Link, useParams } from 'react-router-dom';
import { Api } from '../../api';
import type { Alert as MonitorAlert } from '../../api/modules/monitoring';
import type { AlertHealApprovalTask, AlertHealJob } from '../../api/modules/aiAlertHeal';
import { normalizeHealStatus } from './monitorAlertHealStatus';

const retryBlockedStatuses = new Set(['pending', 'analyzing', 'auto_fixing', 'waiting_approval']);

function pickLatestJob(jobs: AlertHealJob[]): AlertHealJob | undefined {
  if (!jobs.length) {
    return undefined;
  }
  return [...jobs].sort((a, b) => new Date(b.updated_at || 0).getTime() - new Date(a.updated_at || 0).getTime())[0];
}

const AlertDetailPage: React.FC = () => {
  const { alertId = '' } = useParams<{ alertId: string }>();
  const [loading, setLoading] = useState(false);
  const [retrying, setRetrying] = useState(false);
  const [showTrace, setShowTrace] = useState(false);
  const [showApprovals, setShowApprovals] = useState(false);
  const [approvalsLoading, setApprovalsLoading] = useState(false);
  const [alertItem, setAlertItem] = useState<MonitorAlert | null>(null);
  const [latestJob, setLatestJob] = useState<AlertHealJob | null>(null);
  const [approvals, setApprovals] = useState<AlertHealApprovalTask[]>([]);

  const load = async () => {
    if (!alertId) {
      return;
    }
    setLoading(true);
    try {
      const [alertRes, jobsRes] = await Promise.all([
        Api.monitoring.getAlertList({ alertId, page: 1, pageSize: 1 }),
        Api.aiAlertHeal.listByAlert(alertId),
      ]);
      setAlertItem(alertRes.data?.list?.[0] || null);
      setLatestJob(pickLatestJob(jobsRes.data?.list || []) || null);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void load();
  }, [alertId]);

  const retryDisabled = !latestJob || (alertItem?.status === 'resolved') || retryBlockedStatuses.has(latestJob.status);

  const loadApprovals = async () => {
    if (!latestJob || latestJob.status !== 'waiting_approval') {
      setApprovals([]);
      return;
    }
    setApprovalsLoading(true);
    try {
      const response = await Api.aiAlertHeal.listGlobalPendingApprovals(1, 20);
      const list = response.data?.list || [];
      if (!latestJob.latest_run_id) {
        setApprovals(list);
        return;
      }
      setApprovals(list.filter((item) => item.run_id === latestJob.latest_run_id));
    } catch (error: any) {
      message.error(error?.message || '加载审批任务失败');
    } finally {
      setApprovalsLoading(false);
    }
  };

  const onToggleApprovals = async () => {
    if (showApprovals) {
      setShowApprovals(false);
      return;
    }
    setShowApprovals(true);
    await loadApprovals();
  };

  const onRetry = async () => {
    if (!latestJob || retryDisabled) {
      return;
    }
    setRetrying(true);
    try {
      await Api.aiAlertHeal.retryJob(latestJob.id);
      message.success('已触发重试');
      await load();
    } catch (error: any) {
      message.error(error?.message || '重试失败');
    } finally {
      setRetrying(false);
    }
  };

  const healStatus = normalizeHealStatus(latestJob?.status || '');

  return (
    <Space orientation="vertical" size={16} style={{ width: '100%' }}>
      <Card
        title={
          <Space>
            <Link to="/monitor/alerts">返回列表</Link>
            <Typography.Text>告警详情</Typography.Text>
          </Space>
        }
        extra={
          <Space>
            <Button loading={retrying} disabled={retryDisabled} onClick={() => void onRetry()}>
              手动重试
            </Button>
            {latestJob?.status === 'waiting_approval' ? (
              <Button onClick={() => void onToggleApprovals()}>查看审批</Button>
            ) : null}
            <Button onClick={() => setShowTrace((prev) => !prev)}>查看执行轨迹</Button>
          </Space>
        }
        loading={loading}
      >
        {alertItem ? (
          <Descriptions column={1} bordered>
            <Descriptions.Item label="告警消息">{alertItem.title || '-'}</Descriptions.Item>
            <Descriptions.Item label="级别">
              <Tag color={alertItem.severity === 'critical' ? 'error' : alertItem.severity === 'warning' ? 'warning' : 'blue'}>
                {alertItem.severity || '-'}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label="状态">{alertItem.status || '-'}</Descriptions.Item>
            <Descriptions.Item label="处理状态">
              <Tag color={healStatus.processingColor}>{healStatus.processing}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="自愈状态">
              <Tag color={healStatus.healingColor}>{healStatus.healing}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="最近处理">
              {latestJob?.updated_at ? new Date(latestJob.updated_at).toLocaleString() : '-'}
            </Descriptions.Item>
          </Descriptions>
        ) : (
          <Empty description="未找到告警" />
        )}
      </Card>

      {showApprovals ? (
        <Card title="待审批任务" loading={approvalsLoading}>
          {approvals.length > 0 ? (
            <Table<AlertHealApprovalTask>
              rowKey="approval_id"
              pagination={false}
              dataSource={approvals}
              columns={[
                { title: '审批ID', dataIndex: 'approval_id' },
                { title: '工具', dataIndex: 'tool_name', render: (value: string) => value || '-' },
                { title: '状态', dataIndex: 'status', render: (value: string) => value || '-' },
                {
                  title: '创建时间',
                  dataIndex: 'created_at',
                  render: (value?: string) => (value ? new Date(value).toLocaleString() : '-'),
                },
              ]}
            />
          ) : (
            <Empty description="暂无匹配审批任务" />
          )}
        </Card>
      ) : null}

      {showTrace ? (
        <Card title="执行轨迹">
          {latestJob?.latest_run_id || latestJob?.last_error ? (
            <Descriptions column={1} size="small" bordered>
              <Descriptions.Item label="latest_run_id">{latestJob?.latest_run_id || '-'}</Descriptions.Item>
              <Descriptions.Item label="last_error">{latestJob?.last_error || '-'}</Descriptions.Item>
            </Descriptions>
          ) : (
            <Alert type="info" showIcon message="暂无执行轨迹" />
          )}
        </Card>
      ) : null}
    </Space>
  );
};

export default AlertDetailPage;
