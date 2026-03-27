import React from 'react';
import { Card, Descriptions, Tag, Space, Alert } from 'antd';
import { ClockCircleOutlined, UserOutlined, FileTextOutlined } from '@ant-design/icons';

interface ApprovalRequest {
  ticket_id?: string;
  requester?: string;
  requester_email?: string;
  reason?: string;
  created_at: string;
  approved_by?: string;
  approved_at?: string;
  rejected_by?: string;
  rejected_at?: string;
  comment?: string;
}

interface ApprovalWorkflowProps {
  approval: ApprovalRequest;
  status: 'pending' | 'approved' | 'rejected';
}

const ApprovalWorkflow: React.FC<ApprovalWorkflowProps> = ({ approval, status }) => {
  return (
    <Card title="审批信息">
      <Descriptions column={1} bordered>
        {approval.ticket_id && (
          <Descriptions.Item label="审批单号">
            <Space>
              <FileTextOutlined />
              {approval.ticket_id}
            </Space>
          </Descriptions.Item>
        )}
        <Descriptions.Item label="请求人">
          <Space>
            <UserOutlined />
            {approval.requester || '未知'}
            {approval.requester_email && (
              <span className="text-xs text-gray-500">({approval.requester_email})</span>
            )}
          </Space>
        </Descriptions.Item>
        <Descriptions.Item label="请求时间">
          <Space>
            <ClockCircleOutlined />
            {new Date(approval.created_at).toLocaleString()}
          </Space>
        </Descriptions.Item>
        {approval.reason && (
          <Descriptions.Item label="申请理由">{approval.reason}</Descriptions.Item>
        )}
        <Descriptions.Item label="审批状态">
          {status === 'pending' && (
            <Tag color="orange" icon={<ClockCircleOutlined />}>
              待审批
            </Tag>
          )}
          {status === 'approved' && (
            <Tag color="success">已批准</Tag>
          )}
          {status === 'rejected' && (
            <Tag color="error">已拒绝</Tag>
          )}
        </Descriptions.Item>
        {approval.approved_by && (
          <>
            <Descriptions.Item label="批准人">{approval.approved_by}</Descriptions.Item>
            <Descriptions.Item label="批准时间">
              {approval.approved_at && new Date(approval.approved_at).toLocaleString()}
            </Descriptions.Item>
          </>
        )}
        {approval.rejected_by && (
          <>
            <Descriptions.Item label="拒绝人">{approval.rejected_by}</Descriptions.Item>
            <Descriptions.Item label="拒绝时间">
              {approval.rejected_at && new Date(approval.rejected_at).toLocaleString()}
            </Descriptions.Item>
          </>
        )}
        {approval.comment && (
          <Descriptions.Item label="审批备注">{approval.comment}</Descriptions.Item>
        )}
      </Descriptions>

      {status === 'pending' && (
        <Alert
          message="等待审批"
          description="此发布需要审批后才能执行。请相关人员审核并批准或拒绝。"
          type="warning"
          showIcon
          className="mt-4"
        />
      )}
    </Card>
  );
};

export default ApprovalWorkflow;
