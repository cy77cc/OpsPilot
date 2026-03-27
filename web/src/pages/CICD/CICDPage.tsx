import React from 'react';
import { Button, Card, Col, Form, Input, InputNumber, Row, Select, Space, Table, Tabs, Tag, message } from 'antd';
import { Api } from '../../api';
import type { ServiceItem } from '../../api/modules/services';
import type { DeployTarget } from '../../api/modules/deployment';
import type { CIRunRecord, ReleaseApproval, ReleaseRecord, ServiceCIConfig, TimelineEvent } from '../../api/modules/cicd';

const triggerModes = ['manual', 'source-event', 'both'];
const releaseStrategies = ['rolling', 'blue-green', 'canary'];

const CICDPage: React.FC = () => {
  const [loading, setLoading] = React.useState(false);
  const [services, setServices] = React.useState<ServiceItem[]>([]);
  const [targets, setTargets] = React.useState<DeployTarget[]>([]);
  const [ciConfig, setCiConfig] = React.useState<ServiceCIConfig | null>(null);
  const [ciRuns, setCiRuns] = React.useState<CIRunRecord[]>([]);
  const [releases, setReleases] = React.useState<ReleaseRecord[]>([]);
  const [approvals, setApprovals] = React.useState<ReleaseApproval[]>([]);
  const [timeline, setTimeline] = React.useState<TimelineEvent[]>([]);

  const [serviceId, setServiceId] = React.useState<number>();
  const [deploymentId, setDeploymentId] = React.useState<number>();

  const [ciForm] = Form.useForm();
  const [cdForm] = Form.useForm();
  const [releaseForm] = Form.useForm();
  const [rollbackForm] = Form.useForm();

  const loadBase = React.useCallback(async () => {
    const [svcRes, targetRes] = await Promise.all([
      Api.services.getList({ page: 1, pageSize: 500 }),
      Api.deployment.getTargets(),
    ]);
    const svcList = svcRes.data.list || [];
    setServices(svcList);
    setTargets(targetRes.data.list || []);
    if (!serviceId && svcList.length > 0) setServiceId(Number(svcList[0].id));
    if (!deploymentId && (targetRes.data.list || []).length > 0) setDeploymentId(Number(targetRes.data.list[0].id));
  }, [serviceId, deploymentId]);

  const loadDetail = React.useCallback(async () => {
    if (!serviceId) return;
    setLoading(true);
    try {
      const reqs: Promise<any>[] = [
        Api.cicd.listCIRuns(serviceId),
        Api.cicd.listReleases({ service_id: serviceId }),
        Api.cicd.getServiceTimeline(serviceId),
      ];
      reqs.unshift(Api.cicd.getServiceCIConfig(serviceId));
      const [ciCfgRes, ciRunRes, releaseRes, timelineRes] = await Promise.all(reqs);
      setCiConfig(ciCfgRes.data || null);
      setCiRuns(ciRunRes.data.list || []);
      setReleases(releaseRes.data.list || []);
      setTimeline(timelineRes.data.list || []);
      if (releaseRes.data.list?.[0]?.id) {
        const ar = await Api.cicd.listApprovals(releaseRes.data.list[0].id);
        setApprovals(ar.data.list || []);
      } else {
        setApprovals([]);
      }
      if (ciCfgRes.data) {
        ciForm.setFieldsValue({
          repo_url: ciCfgRes.data.repo_url,
          branch: ciCfgRes.data.branch,
          build_steps: (ciCfgRes.data.build_steps || []).join('\n'),
          artifact_target: ciCfgRes.data.artifact_target,
          trigger_mode: ciCfgRes.data.trigger_mode,
        });
      }
    } catch {
      setCiConfig(null);
      setCiRuns([]);
      setReleases([]);
      setTimeline([]);
      setApprovals([]);
    } finally {
      setLoading(false);
    }
  }, [serviceId, ciForm]);

  React.useEffect(() => {
    void loadBase();
  }, [loadBase]);

  React.useEffect(() => {
    void loadDetail();
  }, [loadDetail]);

  const saveCIConfig = async () => {
    if (!serviceId) return;
    const v = await ciForm.validateFields();
    await Api.cicd.putServiceCIConfig(serviceId, {
      repo_url: String(v.repo_url),
      branch: String(v.branch || 'main'),
      build_steps: String(v.build_steps || '').split('\n').map((s: string) => s.trim()).filter(Boolean),
      artifact_target: String(v.artifact_target),
      trigger_mode: v.trigger_mode,
    });
    message.success('CI 配置已保存');
    await loadDetail();
  };

  const deleteCIConfig = async () => {
    if (!serviceId) return;
    await Api.cicd.deleteServiceCIConfig(serviceId);
    message.success('CI 配置已删除');
    setCiConfig(null);
    ciForm.resetFields();
    await loadDetail();
  };

  const triggerRun = async (trigger_type: 'manual' | 'source-event') => {
    if (!serviceId) return;
    await Api.cicd.triggerCIRun(serviceId, { trigger_type });
    message.success(`CI 已触发: ${trigger_type}`);
    await loadDetail();
  };

  const saveCDConfig = async () => {
    if (!deploymentId) return;
    const selectedTarget = targets.find((t) => Number(t.id) === Number(deploymentId));
    const runtimeType = (selectedTarget?.runtime_type || selectedTarget?.target_type || 'k8s') as 'k8s' | 'compose';
    const v = await cdForm.validateFields();
    let strategyConfig = {};
    try {
      strategyConfig = v.strategy_config ? JSON.parse(v.strategy_config) : {};
    } catch {
      message.error('策略配置必须是 JSON 对象');
      return;
    }
    if (runtimeType === 'compose' && v.strategy === 'canary') {
      message.error('Compose 运行时不支持 canary 策略');
      return;
    }
    await Api.cicd.putDeploymentCDConfig(deploymentId, {
      env: String(v.env || 'staging'),
      runtime_type: runtimeType,
      strategy: v.strategy,
      strategy_config: strategyConfig,
      approval_required: Boolean(v.approval_required),
    });
    message.success('CD 配置已保存');
  };

  const triggerRelease = async () => {
    const v = await releaseForm.validateFields();
    const selectedTarget = targets.find((t) => Number(t.id) === Number(v.deployment_id));
    const runtimeType = (selectedTarget?.runtime_type || selectedTarget?.target_type || 'k8s') as 'k8s' | 'compose';
    await Api.cicd.triggerRelease({
      service_id: Number(v.service_id),
      deployment_id: Number(v.deployment_id),
      env: String(v.env),
      runtime_type: runtimeType,
      version: String(v.version),
    });
    message.success('发布已触发');
    setServiceId(Number(v.service_id));
    await loadDetail();
  };

  const approve = async (id: number, decision: 'approve' | 'reject') => {
    if (decision === 'approve') {
      await Api.cicd.approveRelease(id, 'approved in UI');
      message.success('已审批通过');
    } else {
      await Api.cicd.rejectRelease(id, 'rejected in UI');
      message.success('已拒绝');
    }
    await loadDetail();
  };

  const rollback = async () => {
    const v = await rollbackForm.validateFields();
    await Api.cicd.rollbackRelease(Number(v.release_id), {
      target_version: String(v.target_version),
      comment: String(v.comment || ''),
    });
    message.success('回滚已提交');
    await loadDetail();
  };

  const serviceOptions = services.map((s) => ({ value: Number(s.id), label: `${s.name} (#${s.id})` }));
  const targetOptions = targets.map((t) => ({ value: Number(t.id), label: `${t.name} [${t.env}]` }));

  return (
    <div className="space-y-4">
      <Card title="CI/CD 管理" extra={<Button onClick={() => void loadDetail()} loading={loading}>刷新</Button>}>
        <Row gutter={12}>
          <Col span={8}>
            <Select style={{ width: '100%' }} value={serviceId} options={serviceOptions} onChange={(v) => setServiceId(Number(v))} placeholder="选择服务" />
          </Col>
          <Col span={8}>
            <Select style={{ width: '100%' }} value={deploymentId} options={targetOptions} onChange={(v) => setDeploymentId(Number(v))} placeholder="选择部署目标" />
          </Col>
        </Row>
      </Card>

      <Tabs items={[
        {
          key: 'service-ci',
          label: '服务 CI 配置',
          children: (
            <Card>
              <Form form={ciForm} layout="vertical" initialValues={{ branch: 'main', trigger_mode: 'manual' }}>
                <Form.Item name="repo_url" label="仓库地址" rules={[{ required: true }]}><Input /></Form.Item>
                <Form.Item name="branch" label="分支"><Input /></Form.Item>
                <Form.Item name="build_steps" label="构建步骤（每行一个）"><Input.TextArea rows={5} /></Form.Item>
                <Form.Item name="artifact_target" label="产物目标（镜像仓库）" rules={[{ required: true }]}><Input /></Form.Item>
                <Form.Item name="trigger_mode" label="触发模式" rules={[{ required: true }]}><Select options={triggerModes.map((x) => ({ value: x }))} /></Form.Item>
                <Space>
                  <Button type="primary" onClick={() => void saveCIConfig()} disabled={!serviceId}>保存 CI 配置</Button>
                  <Button danger onClick={() => void deleteCIConfig()} disabled={!serviceId}>删除 CI 配置</Button>
                  <Button onClick={() => void triggerRun('manual')} disabled={!serviceId}>手动触发</Button>
                  <Button onClick={() => void triggerRun('source-event')} disabled={!serviceId}>事件触发</Button>
                </Space>
              </Form>
              <Table rowKey="id" dataSource={ciRuns} pagination={false} style={{ marginTop: 16 }} columns={[
                { title: 'Run ID', dataIndex: 'id' },
                { title: '触发类型', dataIndex: 'trigger_type' },
                { title: '状态', dataIndex: 'status', render: (v) => <Tag>{v}</Tag> },
                { title: '触发人', dataIndex: 'triggered_by' },
                { title: '时间', dataIndex: 'triggered_at' },
              ]} />
            </Card>
          ),
        },
        {
          key: 'deployment-cd',
          label: '部署 CD 配置与发布',
          children: (
            <Card>
              <Form form={cdForm} layout="vertical" initialValues={{ env: 'staging', strategy: 'rolling', approval_required: false, strategy_config: '{}' }}>
                <Row gutter={12}>
                  <Col span={6}><Form.Item name="env" label="环境" rules={[{ required: true }]}><Input /></Form.Item></Col>
                  <Col span={6}><Form.Item name="strategy" label="发布策略" rules={[{ required: true }]}><Select options={releaseStrategies.map((x) => ({ value: x }))} /></Form.Item></Col>
                  <Col span={6}><Form.Item name="approval_required" label="需要审批"><Select options={[{ value: true, label: '是' }, { value: false, label: '否' }]} /></Form.Item></Col>
                </Row>
                <Form.Item name="strategy_config" label="策略配置(JSON)"><Input.TextArea rows={4} /></Form.Item>
                <Button type="primary" onClick={() => void saveCDConfig()} disabled={!deploymentId}>保存 CD 配置</Button>
              </Form>

              <Form form={releaseForm} layout="vertical" style={{ marginTop: 16 }} initialValues={{ env: 'staging' }}>
                <Row gutter={12}>
                  <Col span={6}><Form.Item name="service_id" label="服务" rules={[{ required: true }]}><Select options={serviceOptions} /></Form.Item></Col>
                  <Col span={6}><Form.Item name="deployment_id" label="部署目标" rules={[{ required: true }]}><Select options={targetOptions} /></Form.Item></Col>
                  <Col span={4}><Form.Item name="env" label="环境" rules={[{ required: true }]}><Input /></Form.Item></Col>
                  <Col span={6}><Form.Item name="version" label="版本" rules={[{ required: true }]}><Input placeholder="v1.0.0" /></Form.Item></Col>
                </Row>
                <Button type="primary" onClick={() => void triggerRelease()}>触发发布</Button>
              </Form>

              <Table rowKey="id" dataSource={releases} pagination={false} style={{ marginTop: 16 }} columns={[
                { title: 'Release ID', dataIndex: 'id' },
                { title: 'Unified ID', dataIndex: 'unified_release_id', render: (v) => v || '-' },
                { title: '版本', dataIndex: 'version' },
                { title: '状态', dataIndex: 'status', render: (v) => <Tag color={v === 'pending_approval' ? 'orange' : v === 'applied' || v === 'succeeded' ? 'green' : v === 'failed' ? 'red' : 'blue'}>{v}</Tag> },
                { title: '来源', dataIndex: 'trigger_source', render: (v) => <Tag color={v === 'ci' ? 'purple' : 'geekblue'}>{v || 'manual'}</Tag> },
                { title: '策略', dataIndex: 'strategy' },
                {
                  title: '操作',
                  render: (_, row) => (
                    <Space>
                      <Button data-testid={`approve-release-${row.id}`} size="small" onClick={() => void approve(row.id, 'approve')} disabled={row.status !== 'pending_approval'}>通过</Button>
                      <Button data-testid={`reject-release-${row.id}`} size="small" danger onClick={() => void approve(row.id, 'reject')} disabled={row.status !== 'pending_approval'}>拒绝</Button>
                    </Space>
                  ),
                },
              ]} />
            </Card>
          ),
        },
        {
          key: 'audit',
          label: '审批与审计时间线',
          children: (
            <Card>
              <Form form={rollbackForm} layout="inline" initialValues={{ target_version: 'v0.0.0' }}>
                <Form.Item name="release_id" label="Release ID" rules={[{ required: true }]}><InputNumber min={1} /></Form.Item>
                <Form.Item name="target_version" label="回滚到版本" rules={[{ required: true }]}><Input /></Form.Item>
                <Form.Item name="comment" label="说明"><Input /></Form.Item>
                <Button onClick={() => void rollback()}>执行回滚</Button>
              </Form>
              <Table rowKey="id" dataSource={approvals} pagination={false} style={{ marginTop: 16 }} columns={[
                { title: '审批ID', dataIndex: 'id' },
                { title: 'Release ID', dataIndex: 'release_id' },
                { title: '审批人', dataIndex: 'approver_id' },
                { title: '决策', dataIndex: 'decision' },
                { title: '备注', dataIndex: 'comment' },
                { title: '时间', dataIndex: 'created_at' },
              ]} />
              <Table rowKey="id" dataSource={timeline} pagination={{ pageSize: 10 }} style={{ marginTop: 16 }} columns={[
                { title: '事件ID', dataIndex: 'id' },
                { title: '类型', dataIndex: 'event_type' },
                { title: 'Release', dataIndex: 'release_id' },
                { title: '操作人', dataIndex: 'actor_id' },
                { title: '时间', dataIndex: 'created_at' },
              ]} />
            </Card>
          ),
        },
      ]} />

      <Card title="当前服务 CI 摘要">
        {ciConfig ? <pre>{JSON.stringify(ciConfig, null, 2)}</pre> : '暂无 CI 配置'}
      </Card>
    </div>
  );
};

export default CICDPage;
