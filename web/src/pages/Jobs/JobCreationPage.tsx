import React, { useEffect, useState } from 'react';
import { Form, Input, Select, Button, Card, Row, Col, InputNumber, Switch, notification, Space } from 'antd';
import { SaveOutlined } from '@ant-design/icons';
import { useNavigate, useParams } from 'react-router-dom';
import { Api } from '../../api';

const { Option } = Select;
const { TextArea } = Input;

const JobCreationPage: React.FC = () => {
  const [form] = Form.useForm();
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const [isEditing, setIsEditing] = useState(false);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (id && id !== 'create') {
      setIsEditing(true);
      Api.tasks.getTaskDetail(id).then((res) => {
        const job = res.data;
        form.setFieldsValue({
          name: job.name,
          type: job.type,
          command: job.command,
          schedule: job.schedule,
          description: job.description,
          timeout: job.timeout || 300,
          priority: job.priority || 0,
          hostIds: job.hostIds || '1',
          enabled: job.status !== 'stopped',
        });
      });
    }
  }, [id]);

  const onFinish = async (values: any) => {
    setLoading(true);
    try {
      if (isEditing && id) {
        await Api.tasks.updateTask(id, {
          name: values.name,
          type: values.type,
          command: values.command,
          schedule: values.schedule,
          description: values.description,
          timeout: values.timeout,
          priority: values.priority,
          hostIds: values.hostIds,
          status: values.enabled ? 'pending' : 'stopped',
        });
      } else {
        await Api.tasks.createTask({
          name: values.name,
          type: values.type,
          command: values.command,
          schedule: values.schedule,
          description: values.description,
          timeout: values.timeout,
          priority: values.priority,
          hostIds: values.hostIds,
        });
      }

      notification.success({
        message: isEditing ? '更新成功' : '创建成功',
        description: isEditing ? '任务已成功更新' : '任务已成功创建',
      });
      navigate('/jobs');
    } catch (error) {
      notification.error({ message: '提交失败', description: (error as Error).message || '请重试' });
    } finally {
      setLoading(false);
    }
  };

  return (
    <Card style={{ background: '#16213e', border: '1px solid #2d3748' }} title={<span className="text-white text-lg">{isEditing ? '编辑任务' : '创建新任务'}</span>}>
      <Form
        form={form}
        name="job_form"
        layout="vertical"
        onFinish={onFinish}
        initialValues={{
          type: 'shell',
          schedule: '0 * * * *',
          timeout: 300,
          priority: 0,
          hostIds: '1',
          enabled: true,
        }}
      >
        <Card title="基本信息" style={{ marginBottom: 16, background: '#1a1a2e', border: '1px solid #2d3748' }}>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="name" label="任务名称" rules={[{ required: true, message: '请输入任务名称' }]}>
                <Input placeholder="输入任务名称" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="type" label="任务类型" rules={[{ required: true, message: '请选择任务类型' }]}>
                <Select placeholder="选择任务类型">
                  <Option value="shell">Shell脚本</Option>
                  <Option value="python">Python</Option>
                  <Option value="http">HTTP请求</Option>
                  <Option value="ansible">Ansible</Option>
                  <Option value="kubectl">Kubectl</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Form.Item name="description" label="任务描述">
            <TextArea rows={3} placeholder="输入任务描述" />
          </Form.Item>
        </Card>

        <Card title="执行配置" style={{ marginBottom: 16, background: '#1a1a2e', border: '1px solid #2d3748' }}>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="schedule" label="定时规则 (cron)" rules={[{ required: true, message: '请输入定时规则' }]}>
                <Input placeholder="例如: */5 * * * *" />
              </Form.Item>
              <Form.Item name="command" label="执行命令" rules={[{ required: true, message: '请输入执行命令' }]}>
                <TextArea rows={6} placeholder="输入要执行的命令或脚本内容" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="hostIds" label="执行节点ID" extra="多个ID用逗号分隔，如 1,2,3">
                <Input placeholder="默认 1" />
              </Form.Item>
              <Form.Item name="timeout" label="超时时间（秒）">
                <InputNumber min={1} max={86400} className="w-full" />
              </Form.Item>
              <Form.Item name="priority" label="优先级">
                <InputNumber min={0} max={100} className="w-full" />
              </Form.Item>
              <Form.Item name="enabled" label="启用状态" valuePropName="checked">
                <Switch checkedChildren="启用" unCheckedChildren="禁用" />
              </Form.Item>
            </Col>
          </Row>
        </Card>

        <Space style={{ display: 'flex', justifyContent: 'flex-end' }}>
          <Button size="large" onClick={() => navigate('/jobs')}>取消</Button>
          <Button type="primary" size="large" icon={<SaveOutlined />} htmlType="submit" loading={loading}>
            {isEditing ? '保存更改' : '创建任务'}
          </Button>
        </Space>
      </Form>
    </Card>
  );
};

export default JobCreationPage;
