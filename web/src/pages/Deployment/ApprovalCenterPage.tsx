import React, { useState, useEffect } from 'react';
import { Card, Tabs, Table, Button, Space, Tag, message, Modal, Input } from 'antd';
import {
  ReloadOutlined,
  CheckOutlined,
  CloseOutlined,
  ClockCircleOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { Api } from '../../api';
import type { DeployRelease } from '../../api/modules/deployment';

const { TextArea } = Input;

const ApprovalCenterPage: React.FC = () => {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const [releases, setReleases] = useState<DeployRelease[]>([]);
  const [activeTab, setActiveTab] = useState('pending');
  const [commentModalVisible, setCommentModalVisible] = useState(false);
  const [currentRelease, setCurrentRelease] = useState<DeployRelease | null>(null);
  const [actionType, setActionType] = useState<'approve' | 'reject'>('approve');
  const [comment, setComment] = useState('');

  const load = async () => {
    setLoading(true);
    try {
      const res = await Api.deployment.getReleases();
      setReleases(res.data.list || []);
    } catch (err) {
      message.error('加载审批列表失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const handleApprove = (release: DeployRelease) => {
    setCurrentRelease(release);
    setActionType('approve');
    setCommentModalVisible(true);
  };

  const handleReject = (release: DeployRelease) => {
    setCurrentRelease(release);
    setActionType('reject');
    setCommentModalVisible(true);
  };

  const handleSubmitAction = async () => {
    if (!currentRelease) return;
    try {
      if (actionType === 'approve') {
        await Api.deployment.approveRelease(currentRelease.id, { comment });
        message.success('审批通过');
      } else {
        await Api.deployment.rejectRelease(currentRelease.id, { comment });
        message.success('已拒绝');
      }
      setCommentModalVisible(false);
      setComment('');
      setCurrentRelease(null);
      load();
    } catch (err) {
      message.error(err instanceof Error ? err.message : '操作失败');
    }
  };

  const columns = [
    {
      title: 'Release ID',
      dataIndex: 'id',
      key: 'id',
      render: (id: number) => (
        <a onClick={() => navigate(`/deployment/${id}`)}>#{id}</a>
      ),
    },
    {
      title: '服务',
      dataIndex: 'service_name',
      key: 'service_name',
    },
    {
      title: '目标',
      dataIndex: 'target_name',
      key: 'target_name',
    },
    {
      title: '环境',
      dataIndex: 'environment',
      key: 'environment',
      render: (env: string) => (
        <Tag color={env === 'production' ? 'red' : env === 'staging' ? 'orange' : 'blue'}>
          {env}
        </Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'state',
      key: 'state',
      render: (state: string) => {
        const configs: Record<string, { icon: React.ReactNode; color: string; text: string }> = {
          pending_approval: { icon: <ClockCircleOutlined />, color: 'orange', text: '待审批' },
          approved: { icon: <CheckCircleOutlined />, color: 'blue', text: '已批准' },
          rejected: { icon: <CloseCircleOutlined />, color: 'default', text: '已拒绝' },
        };
        const config = configs[state] || { icon: null, color: 'default', text: state };
        return (
          <Tag color={config.color} icon={config.icon}>
            {config.text}
          </Tag>
        );
      },
    },
    {
      title: '请求人',
      dataIndex: 'requester',
      key: 'requester',
      render: (requester: string) => requester || '-',
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (time: string) => new Date(time).toLocaleString(),
    },
    {
      title: '操作',
      key: 'actions',
      render: (_: any, record: DeployRelease) => {
        if (record.state === 'pending_approval') {
          return (
            <Space>
              <Button
                type="primary"
                size="small"
                icon={<CheckOutlined />}
                onClick={() => handleApprove(record)}
              >
                批准
              </Button>
              <Button
                danger
                size="small"
                icon={<CloseOutlined />}
                onClick={() => handleReject(record)}
              >
                拒绝
              </Button>
            </Space>
          );
        }
        return '-';
      },
    },
  ];

  const pendingReleases = releases.filter((r) => r.state === 'pending_approval');
  const approvedReleases = releases.filter((r) => r.state === 'approved' || r.state === 'applied' || r.state === 'applying');
  const rejectedReleases = releases.filter((r) => r.state === 'rejected');

  return (
    <div className="space-y-6">
      {/* Page header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">审批中心</h1>
          <p className="text-sm text-gray-500 mt-1">管理发布审批请求</p>
        </div>
        <Button icon={<ReloadOutlined />} onClick={load} loading={loading}>
          刷新
        </Button>
      </div>

      {/* Tabs */}
      <Card>
        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          items={[
            {
              key: 'pending',
              label: (
                <span>
                  <ClockCircleOutlined /> 待审批 ({pendingReleases.length})
                </span>
              ),
              children: (
                <Table
                  dataSource={pendingReleases}
                  columns={columns}
                  rowKey="id"
                  loading={loading}
                  pagination={{ pageSize: 10 }}
                />
              ),
            },
            {
              key: 'approved',
              label: (
                <span>
                  <CheckCircleOutlined /> 已批准 ({approvedReleases.length})
                </span>
              ),
              children: (
                <Table
                  dataSource={approvedReleases}
                  columns={columns.filter((c) => c.key !== 'actions')}
                  rowKey="id"
                  loading={loading}
                  pagination={{ pageSize: 10 }}
                />
              ),
            },
            {
              key: 'rejected',
              label: (
                <span>
                  <CloseCircleOutlined /> 已拒绝 ({rejectedReleases.length})
                </span>
              ),
              children: (
                <Table
                  dataSource={rejectedReleases}
                  columns={columns.filter((c) => c.key !== 'actions')}
                  rowKey="id"
                  loading={loading}
                  pagination={{ pageSize: 10 }}
                />
              ),
            },
          ]}
        />
      </Card>

      {/* Comment modal */}
      <Modal
        title={actionType === 'approve' ? '审批通过' : '拒绝发布'}
        open={commentModalVisible}
        onOk={handleSubmitAction}
        onCancel={() => {
          setCommentModalVisible(false);
          setComment('');
          setCurrentRelease(null);
        }}
        okText={actionType === 'approve' ? '批准' : '拒绝'}
        okButtonProps={{ danger: actionType === 'reject' }}
      >
        <div className="space-y-4">
          <div>
            <div className="text-sm text-gray-600 mb-2">Release ID: #{currentRelease?.id}</div>
            <div className="text-sm text-gray-600 mb-2">服务: {currentRelease?.service_name}</div>
            <div className="text-sm text-gray-600 mb-2">目标: {currentRelease?.target_name}</div>
          </div>
          <div>
            <div className="text-sm font-semibold mb-2">备注 (可选):</div>
            <TextArea
              rows={4}
              value={comment}
              onChange={(e) => setComment(e.target.value)}
              placeholder="输入审批备注..."
            />
          </div>
        </div>
      </Modal>
    </div>
  );
};

export default ApprovalCenterPage;
