import React, { useState } from 'react';
import { Button, Card, Form, Input, Select, Space, message } from 'antd';
import { Api } from '../../api';

type ChannelTestForm = {
  provider: string;
  target?: string;
  configJson?: string;
};

const ChannelsConfigPage: React.FC = () => {
  const [form] = Form.useForm<ChannelTestForm>();
  const [submitting, setSubmitting] = useState(false);

  const handleTestSend = async () => {
    const values = await form.validateFields();
    setSubmitting(true);
    try {
      await Api.monitoring.testAlertChannel(values);
      message.success('测试发送成功');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Card title="通知渠道配置">
      <Form
        form={form}
        layout="vertical"
        initialValues={{
          provider: 'webhook',
          target: '',
          configJson: '{}',
        }}
      >
        <Form.Item label="Provider" name="provider" rules={[{ required: true, message: '请选择 provider' }]}>
          <Select
            options={[
              { label: 'Webhook', value: 'webhook' },
              { label: 'Log', value: 'log' },
              { label: 'Email', value: 'email' },
            ]}
          />
        </Form.Item>
        <Form.Item label="目标地址" name="target">
          <Input placeholder="https://example.com/hook" />
        </Form.Item>
        <Form.Item label="配置 JSON" name="configJson">
          <Input.TextArea rows={4} />
        </Form.Item>
        <Space>
          <Button type="primary" onClick={handleTestSend} loading={submitting}>
            测试发送
          </Button>
        </Space>
      </Form>
    </Card>
  );
};

export default ChannelsConfigPage;
