import React, { useEffect, useState } from 'react';
import { Button, Card, Form, Input, Space, Typography, message } from 'antd';
import { useNavigate, useParams } from 'react-router-dom';
import { Api } from '../../api';

const { Title, Paragraph } = Typography;

const ServiceDeployPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [form] = Form.useForm();
  const [previewYAML, setPreviewYAML] = useState('');
  const [previewing, setPreviewing] = useState(false);
  const [deploying, setDeploying] = useState(false);

  useEffect(() => {
    form.setFieldsValue({ env: 'staging' });
  }, [form]);

  const preview = async () => {
    if (!id) return;
    setPreviewing(true);
    try {
      const values = form.getFieldsValue(true);
      const resp = await Api.services.deployPreview(id, {
        env: values.env,
        namespace: values.namespace,
        cluster_id: values.cluster_id ? Number(values.cluster_id) : undefined,
      });
      setPreviewYAML(resp.data.resolved_yaml || '');
    } catch (err) {
      message.error(err instanceof Error ? err.message : '预览失败');
    } finally {
      setPreviewing(false);
    }
  };

  const deploy = async () => {
    if (!id) return;
    setDeploying(true);
    try {
      const values = form.getFieldsValue(true);
      const resp = await Api.services.deploy(id, {
        env: values.env,
        namespace: values.namespace,
        cluster_id: values.cluster_id ? Number(values.cluster_id) : undefined,
      });
      message.success(`部署已触发: #${resp.data.unified_release_id || resp.data.release_record_id}`);
      navigate(`/services/${id}`);
    } catch (err) {
      message.error(err instanceof Error ? err.message : '部署失败');
    } finally {
      setDeploying(false);
    }
  };

  return (
    <div className="p-6 space-y-4">
      <Title level={3}>服务部署</Title>
      <Paragraph className="text-gray-500">统一从服务管理入口进行部署预览与执行。</Paragraph>

      <Card>
        <Form form={form} layout="vertical">
          <Space size={16} wrap>
            <Form.Item label="环境" name="env"><Input style={{ width: 180 }} /></Form.Item>
            <Form.Item label="Cluster ID" name="cluster_id"><Input style={{ width: 180 }} /></Form.Item>
            <Form.Item label="Namespace" name="namespace"><Input style={{ width: 240 }} /></Form.Item>
          </Space>
        </Form>

        <Space className="mb-3">
          <Button onClick={preview} loading={previewing}>预览渲染</Button>
          <Button type="primary" onClick={deploy} loading={deploying}>确认部署</Button>
        </Space>

        <Input.TextArea
          value={previewYAML}
          readOnly
          autoSize={{ minRows: 10, maxRows: 22 }}
          placeholder="预览渲染结果将显示在这里"
        />
      </Card>
    </div>
  );
};

export default ServiceDeployPage;
