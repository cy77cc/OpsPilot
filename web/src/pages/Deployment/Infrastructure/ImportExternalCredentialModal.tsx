import React, { useState } from 'react';
import { Modal, Form, Input, Select, Upload, Button, Radio, message } from 'antd';
import { UploadOutlined } from '@ant-design/icons';
import { Api } from '../../../api';

interface ImportExternalCredentialModalProps {
  visible: boolean;
  onCancel: () => void;
  onSuccess: () => void;
}

const ImportExternalCredentialModal: React.FC<ImportExternalCredentialModalProps> = ({
  visible,
  onCancel,
  onSuccess,
}) => {
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [authMethod, setAuthMethod] = useState<'kubeconfig' | 'cert'>('kubeconfig');

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);
      await Api.deployment.importExternalCredential({
        name: values.name,
        runtime_type: values.runtime_type || 'k8s',
        auth_method: authMethod,
        endpoint: values.endpoint,
        kubeconfig: values.kubeconfig,
        ca_cert: values.ca_cert,
        cert: values.cert,
        key: values.key,
        token: values.token,
      });
      message.success('外部凭证导入成功');
      form.resetFields();
      onSuccess();
    } catch (err) {
      message.error(err instanceof Error ? err.message : '导入失败');
    } finally {
      setLoading(false);
    }
  };

  const handleFileUpload = (field: string) => {
    return {
      beforeUpload: (file: File) => {
        const reader = new FileReader();
        reader.onload = (e) => {
          form.setFieldValue(field, e.target?.result as string);
        };
        reader.readAsText(file);
        return false;
      },
      showUploadList: false,
    };
  };

  return (
    <Modal
      title="导入外部凭证"
      open={visible}
      onCancel={onCancel}
      onOk={handleSubmit}
      confirmLoading={loading}
      width={700}
    >
      <Form form={form} layout="vertical">
        <Form.Item
          name="name"
          label="凭证名称"
          rules={[{ required: true, message: '请输入凭证名称' }]}
        >
          <Input placeholder="例如: production-k8s-cluster" />
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

        <Form.Item label="认证方式">
          <Radio.Group value={authMethod} onChange={(e) => setAuthMethod(e.target.value)}>
            <Radio value="kubeconfig">Kubeconfig 文件</Radio>
            <Radio value="cert">证书认证</Radio>
          </Radio.Group>
        </Form.Item>

        {authMethod === 'kubeconfig' ? (
          <Form.Item
            name="kubeconfig"
            label="Kubeconfig 内容"
            rules={[{ required: true, message: '请上传或粘贴 kubeconfig 内容' }]}
          >
            <Input.TextArea
              rows={10}
              placeholder="粘贴 kubeconfig 内容或使用下方按钮上传文件"
            />
          </Form.Item>
        ) : (
          <>
            <Form.Item
              name="endpoint"
              label="API Server Endpoint"
              rules={[{ required: true, message: '请输入 API Server 地址' }]}
            >
              <Input placeholder="https://kubernetes.example.com:6443" />
            </Form.Item>

            <Form.Item
              name="ca_cert"
              label="CA 证书"
              rules={[{ required: true, message: '请输入 CA 证书' }]}
            >
              <Input.TextArea rows={4} placeholder="-----BEGIN CERTIFICATE-----" />
            </Form.Item>

            <Form.Item
              name="cert"
              label="客户端证书"
              rules={[{ required: true, message: '请输入客户端证书' }]}
            >
              <Input.TextArea rows={4} placeholder="-----BEGIN CERTIFICATE-----" />
            </Form.Item>

            <Form.Item
              name="key"
              label="客户端私钥"
              rules={[{ required: true, message: '请输入客户端私钥' }]}
            >
              <Input.TextArea rows={4} placeholder="-----BEGIN RSA PRIVATE KEY-----" />
            </Form.Item>

            <Form.Item name="token" label="Bearer Token (可选)">
              <Input.Password placeholder="可选的 Bearer Token" />
            </Form.Item>
          </>
        )}

        {authMethod === 'kubeconfig' && (
          <Form.Item>
            <Upload {...handleFileUpload('kubeconfig')}>
              <Button icon={<UploadOutlined />}>上传 Kubeconfig 文件</Button>
            </Upload>
          </Form.Item>
        )}
      </Form>

      <div className="mt-4 p-4 bg-yellow-50 rounded-lg">
        <p className="text-sm text-yellow-800">
          <strong>安全提示:</strong> 所有敏感数据将使用 AES 加密后存储。
          请确保导入的凭证具有适当的权限范围。
        </p>
      </div>
    </Modal>
  );
};

export default ImportExternalCredentialModal;
