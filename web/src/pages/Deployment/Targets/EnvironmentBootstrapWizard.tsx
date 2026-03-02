import React, { useState, useEffect } from 'react';
import { Card, Button, Space, Select, Form, Alert, message, Row, Col } from 'antd';
import { ArrowLeftOutlined, PlayCircleOutlined } from '@ant-design/icons';
import { useNavigate, useParams } from 'react-router-dom';
import { Api } from '../../../api';
import BootstrapPhaseTracker from '../../../components/Deployment/BootstrapPhaseTracker';
import BootstrapLogViewer from '../../../components/Deployment/BootstrapLogViewer';

const EnvironmentBootstrapWizard: React.FC = () => {
  const navigate = useNavigate();
  const { targetId, jobId } = useParams<{ targetId: string; jobId?: string }>();
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [target, setTarget] = useState<any>(null);
  const [bootstrapJob, setBootstrapJob] = useState<any>(null);
  const [polling, setPolling] = useState(false);

  useEffect(() => {
    if (targetId) {
      loadTarget();
    }
    if (jobId) {
      loadBootstrapJob();
      startPolling();
    }
  }, [targetId, jobId]);

  const loadTarget = async () => {
    if (!targetId) return;
    try {
      const res = await Api.deployment.getTargetDetail(Number(targetId));
      setTarget(res.data);
    } catch (err) {
      message.error('加载目标信息失败');
    }
  };

  const loadBootstrapJob = async () => {
    if (!jobId) return;
    try {
      const res = await Api.deployment.getEnvironmentBootstrapJob(jobId);
      setBootstrapJob(res.data);

      // Stop polling if job is complete
      if (res.data.status === 'succeeded' || res.data.status === 'failed') {
        setPolling(false);
      }
    } catch (err) {
      message.error('加载初始化任务失败');
    }
  };

  const startPolling = () => {
    setPolling(true);
    const interval = setInterval(() => {
      loadBootstrapJob();
    }, 10000); // Poll every 10 seconds

    return () => clearInterval(interval);
  };

  const handleStart = async () => {
    if (!targetId) return;
    try {
      await form.validateFields();
      const values = form.getFieldsValue();
      setLoading(true);

      const res = await Api.deployment.startEnvironmentBootstrap({
        name: `bootstrap-${targetId}-${Date.now()}`,
        target_id: Number(targetId),
        runtime_type: target.runtime_type,
        package_version: values.package_version || 'latest',
      });

      message.success('环境初始化已启动');
      navigate(`/deployment/targets/${targetId}/bootstrap/${res.data.job_id}`);
    } catch (err) {
      message.error(err instanceof Error ? err.message : '启动失败');
    } finally {
      setLoading(false);
    }
  };

  // If we have a job ID, show progress tracking
  if (jobId && bootstrapJob) {
    const phases = [
      {
        name: 'Preflight Check',
        status: bootstrapJob.phases?.preflight || 'pending',
        startTime: bootstrapJob.phase_times?.preflight_start,
        endTime: bootstrapJob.phase_times?.preflight_end,
      },
      {
        name: 'Install Runtime',
        status: bootstrapJob.phases?.install || 'pending',
        startTime: bootstrapJob.phase_times?.install_start,
        endTime: bootstrapJob.phase_times?.install_end,
      },
      {
        name: 'Verify Installation',
        status: bootstrapJob.phases?.verify || 'pending',
        startTime: bootstrapJob.phase_times?.verify_start,
        endTime: bootstrapJob.phase_times?.verify_end,
      },
    ];

    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <Button icon={<ArrowLeftOutlined />} onClick={() => navigate(`/deployment/targets/${targetId}`)}>
              返回
            </Button>
            <div>
              <h1 className="text-2xl font-semibold text-gray-900">环境初始化</h1>
              <p className="text-sm text-gray-500 mt-1">Job ID: {jobId}</p>
            </div>
          </div>
        </div>

        <Row gutter={[16, 16]}>
          <Col xs={24} lg={12}>
            <BootstrapPhaseTracker phases={phases} currentPhase={bootstrapJob.current_phase} />
          </Col>
          <Col xs={24} lg={12}>
            <BootstrapLogViewer logs={bootstrapJob.logs || []} />
          </Col>
        </Row>

        {bootstrapJob.status === 'succeeded' && (
          <Alert
            message="初始化成功"
            description="环境初始化已完成，部署目标现在可以使用。"
            type="success"
            showIcon
          />
        )}

        {bootstrapJob.status === 'failed' && (
          <Alert
            message="初始化失败"
            description={`环境初始化失败: ${bootstrapJob.error_message || '未知错误'}`}
            type="error"
            showIcon
          />
        )}
      </div>
    );
  }

  // Otherwise, show the start form
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button icon={<ArrowLeftOutlined />} onClick={() => navigate(`/deployment/targets/${targetId}`)}>
            返回
          </Button>
          <div>
            <h1 className="text-2xl font-semibold text-gray-900">环境初始化</h1>
            <p className="text-sm text-gray-500 mt-1">为部署目标初始化运行时环境</p>
          </div>
        </div>
      </div>

      <Card title="选择运行时包">
        <Form form={form} layout="vertical">
          <Alert
            message="环境初始化说明"
            description={`将为 ${target?.name || '目标'} 安装 ${target?.runtime_type === 'k8s' ? 'Kubernetes' : 'Docker Compose'} 运行时组件和依赖。此过程可能需要几分钟时间。`}
            type="info"
            showIcon
            className="mb-4"
          />

          <Form.Item
            name="package_version"
            label="包版本"
            initialValue="latest"
          >
            <Select
              options={[
                { label: 'Latest (推荐)', value: 'latest' },
                { label: 'Stable', value: 'stable' },
                { label: 'LTS', value: 'lts' },
              ]}
            />
          </Form.Item>

          <Form.Item>
            <Space>
              <Button type="primary" icon={<PlayCircleOutlined />} onClick={handleStart} loading={loading}>
                开始初始化
              </Button>
              <Button onClick={() => navigate(`/deployment/targets/${targetId}`)}>
                取消
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
};

export default EnvironmentBootstrapWizard;
