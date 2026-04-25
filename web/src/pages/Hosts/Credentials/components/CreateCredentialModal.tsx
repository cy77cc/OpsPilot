import React, { useState } from 'react';
import { Modal, Form, Input, message } from 'antd';
import { hostApi } from '../../../../api/modules/hosts';

interface Props {
  open: boolean;
  onCancel: () => void;
  onSuccess: () => void;
}

export const CreateCredentialModal: React.FC<Props> = ({ open, onCancel, onSuccess }) => {
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);
      const res = await hostApi.createSSHKey({
        name: values.name,
        privateKey: values.privateKey,
        passphrase: values.passphrase,
      });
      if (res.success) {
        message.success('凭证创建成功');
        form.resetFields();
        onSuccess();
      }
    } catch (err: any) {
      if (err.message) {
        message.error(err.message);
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal
      title="创建凭证 (SSH 密钥)"
      open={open}
      onCancel={onCancel}
      onOk={handleSubmit}
      confirmLoading={loading}
      width={600}
      destroyOnHidden
    >
      <Form form={form} layout="vertical" className="mt-4">
        <Form.Item
          name="name"
          label="凭证名称"
          rules={[{ required: true, message: '请输入凭证名称' }]}
        >
          <Input placeholder="例如：prod-ssh-key" />
        </Form.Item>

        <Form.Item
          name="privateKey"
          label="私钥内容"
          rules={[{ required: true, message: '请输入私钥内容' }]}
          extra="为了您的服务器安全，私钥将被加密存储。"
        >
          <Input.TextArea rows={6} placeholder="-----BEGIN OPENSSH PRIVATE KEY-----..." style={{ fontFamily: 'monospace' }} />
        </Form.Item>

        <Form.Item
          name="passphrase"
          label="私钥密码 (Passphrase)"
          extra="如果生成密钥时没有设置密码，请留空"
        >
          <Input.Password placeholder="可选" />
        </Form.Item>
      </Form>
    </Modal>
  );
};
