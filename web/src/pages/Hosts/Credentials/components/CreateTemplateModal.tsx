import React, { useState, useEffect } from 'react';
import { Modal, Form, Input, Select, InputNumber, message } from 'antd';
import { hostApi } from '../../../../api/modules/hosts';
import type { SSHKeyItem } from '../../../../api/modules/hosts';

interface Props {
  open: boolean;
  onCancel: () => void;
  onSuccess: () => void;
}

export const CreateTemplateModal: React.FC<Props> = ({ open, onCancel, onSuccess }) => {
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [sshKeys, setSshKeys] = useState<SSHKeyItem[]>([]);
  const authType = Form.useWatch('authType', form);

  useEffect(() => {
    if (open) {
      hostApi.listSSHKeys().then(res => {
        if (res.success) setSshKeys(res.data);
      });
    }
  }, [open]);

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);
      const res = await hostApi.createCredentialTemplate({
        name: values.name,
        authType: values.authType,
        sshUser: values.sshUser,
        port: values.port,
        password: values.password,
        sshKeyId: values.sshKeyId ? Number(values.sshKeyId) : undefined,
        description: values.description,
      });
      if (res.success) {
        message.success('模板创建成功');
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
      title="新建认证预设模板"
      open={open}
      onCancel={onCancel}
      onOk={handleSubmit}
      confirmLoading={loading}
      width={600}
      destroyOnHidden
    >
      <Form
        form={form}
        layout="vertical"
        initialValues={{ authType: 'password', sshUser: 'root', port: 22 }}
        className="mt-4"
      >
        <Form.Item
          name="name"
          label="模板名称"
          rules={[{ required: true, message: '请输入模板名称' }]}
        >
          <Input placeholder="例如：上海生产环境默认认证" />
        </Form.Item>

        <div className="grid grid-cols-2 gap-4">
          <Form.Item
            name="authType"
            label="认证方式"
            rules={[{ required: true }]}
          >
            <Select>
              <Select.Option value="password">用户名密码</Select.Option>
              <Select.Option value="key">SSH 密钥</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item
            name="sshUser"
            label="登录用户名"
            rules={[{ required: true }]}
          >
            <Input placeholder="root" />
          </Form.Item>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <Form.Item
            name="port"
            label="端口"
            rules={[{ required: true }]}
          >
            <InputNumber className="w-full" placeholder="22" />
          </Form.Item>
          {authType === 'password' ? (
            <Form.Item
              name="password"
              label="密码"
              rules={[{ required: true, message: '请输入密码' }]}
            >
              <Input.Password placeholder="登录密码" />
            </Form.Item>
          ) : (
            <Form.Item
              name="sshKeyId"
              label="选择密钥"
              rules={[{ required: true, message: '请选择一个密钥' }]}
            >
              <Select placeholder="请选择凭证库中的密钥">
                {sshKeys.map(key => (
                  <Select.Option key={key.id} value={key.id}>{key.name}</Select.Option>
                ))}
              </Select>
            </Form.Item>
          )}
        </div>

        <Form.Item name="description" label="备注说明">
          <Input.TextArea placeholder="可选" />
        </Form.Item>
      </Form>
    </Modal>
  );
};
