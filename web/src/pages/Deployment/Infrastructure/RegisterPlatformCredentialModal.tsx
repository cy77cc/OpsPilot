import React, { useState, useEffect } from 'react';
import { Modal, Form, Input, Select, message } from 'antd';
import { Api } from '../../../api';

interface RegisterPlatformCredentialModalProps {
  visible: boolean;
  onCancel: () => void;
  onSuccess: () => void;
}

const RegisterPlatformCredentialModal: React.FC<RegisterPlatformCredentialModalProps> = ({
  visible,
  onCancel,
  onSuccess,
}) => {
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [clusters, setClusters] = useState<Array<{ id: number; name: string }>>([]);

  useEffect(() => {
    if (visible) {
      loadClusters();
    }
  }, [visible]);

  const loadClusters = async () => {
    try {
      const res = await Api.cluster.getClusters();
      setClusters(res.data.list || []);
    } catch (err) {
      message.error('加载集群列表失败');
    }
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);
      await Api.deployment.registerPlatformCredential({
        cluster_id: values.cluster_id,
        name: values.name,
        runtime_type: values.runtime_type || 'k8s',
      });
      message.success('平台凭证注册成功');
      form.resetFields();
      onSuccess();
    } catch (err) {
      message.error(err instanceof Error ? err.message : '注册失败');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal
      title="注册平台凭证"
      open={visible}
      onCancel={onCancel}
      onOk={handleSubmit}
      confirmLoading={loading}
      width={600}
    >
      <Form form={form} layout="vertical">
        <Form.Item
          name="cluster_id"
          label="选择集群"
          rules={[{ required: true, message: '请选择集群' }]}
        >
          <Select
            placeholder="选择平台托管的集群"
            showSearch
            filterOption={(input, option) =>
              (option?.label ?? '').toLowerCase().includes(input.toLowerCase())
            }
            options={clusters.map((c) => ({ label: c.name, value: c.id }))}
          />
        </Form.Item>

        <Form.Item
          name="name"
          label="凭证名称"
          tooltip="可选，默认使用集群名称"
        >
          <Input placeholder="留空则使用集群名称" />
        </Form.Item>

        <Form.Item
          name="runtime_type"
          label="运行时类型"
          initialValue="k8s"
        >
          <Select
            options={[
              { label: 'Kubernetes', value: 'k8s' },
              { label: 'Docker Compose', value: 'compose' },
            ]}
          />
        </Form.Item>
      </Form>

      <div className="mt-4 p-4 bg-blue-50 rounded-lg">
        <p className="text-sm text-blue-800">
          <strong>说明:</strong> 平台凭证将自动从选定的集群中提取 kubeconfig 和认证信息。
          系统会加密存储所有敏感数据。
        </p>
      </div>
    </Modal>
  );
};

export default RegisterPlatformCredentialModal;
