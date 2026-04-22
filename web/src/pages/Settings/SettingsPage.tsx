import React, { useState } from 'react';
import { Card, Form, Input, Button, Switch, Select, Divider, Space, message, Tabs } from 'antd';
import { 
  SaveOutlined, 
  UserOutlined, 
  LockOutlined, 
  BellOutlined, 
  GlobalOutlined,
  RobotOutlined
} from '@ant-design/icons';
import { GuidedFormItem } from '../../components/FormGuidance';
import AIModelSettingsPage from './AIModelSettingsPage';

const { Option } = Select;

interface SettingsPageProps {
  defaultTab?: string;
}

const SettingsPage: React.FC<SettingsPageProps> = ({ defaultTab = 'basic' }) => {
  const [form] = Form.useForm();
  const [activeTab, setActiveTab] = useState(defaultTab);

  const handleSave = () => {
    message.success('设置已保存');
  };

  const basicSettings = (
    <div className="max-w-[600px] pt-4">
      <Form form={form} layout="vertical">
        <Divider><UserOutlined /> 个人信息</Divider>
        
        <GuidedFormItem label="用户名" name="username" initialValue="admin">
          <Input placeholder="请输入用户名" />
        </GuidedFormItem>
        
        <GuidedFormItem label="邮箱" name="email" initialValue="admin@company.com">
          <Input placeholder="请输入邮箱" />
        </GuidedFormItem>
        
        <GuidedFormItem label="手机号" name="phone" initialValue="138****8888">
          <Input placeholder="请输入手机号" />
        </GuidedFormItem>

        <Divider><LockOutlined /> 安全设置</Divider>
        
        <GuidedFormItem label="当前密码" name="currentPassword">
          <Input.Password placeholder="请输入当前密码" />
        </GuidedFormItem>
        
        <GuidedFormItem label="新密码" name="newPassword">
          <Input.Password placeholder="请输入新密码" />
        </GuidedFormItem>
        
        <GuidedFormItem label="确认密码" name="confirmPassword">
          <Input.Password placeholder="请确认新密码" />
        </GuidedFormItem>

        <Divider><BellOutlined /> 通知设置</Divider>
        
        <Form.Item label="邮件通知" name="emailNotify" valuePropName="checked" initialValue={true}>
          <Switch />
        </Form.Item>
        
        <Form.Item label="钉钉通知" name="dingtalkNotify" valuePropName="checked" initialValue={true}>
          <Switch />
        </Form.Item>
        
        <Form.Item label="短信通知" name="smsNotify" valuePropName="checked" initialValue={false}>
          <Switch />
        </Form.Item>

        <Divider><GlobalOutlined /> 系统偏好</Divider>
        
        <Form.Item label="语言" name="language" initialValue="zh-CN">
          <Select placeholder="选择语言">
            <Option value="zh-CN">简体中文</Option>
            <Option value="zh-TW">繁體中文</Option>
            <Option value="en-US">English</Option>
          </Select>
        </Form.Item>
        
        <Form.Item label="时区" name="timezone" initialValue="Asia/Shanghai">
          <Select placeholder="选择时区">
            <Option value="Asia/Shanghai">Asia/Shanghai (UTC+8)</Option>
            <Option value="America/New_York">America/New_York (UTC-5)</Option>
            <Option value="Europe/London">Europe/London (UTC+0)</Option>
          </Select>
        </Form.Item>
        
        <Form.Item label="主题" name="theme" initialValue="dark">
          <Select placeholder="选择主题">
            <Option value="dark">深色主题</Option>
            <Option value="light">浅色主题</Option>
            <Option value="auto">跟随系统</Option>
          </Select>
        </Form.Item>

        <Form.Item>
          <Space>
            <Button type="primary" icon={<SaveOutlined />} onClick={handleSave}>
              保存设置
            </Button>
            <Button onClick={() => form.resetFields()}>
              重置
            </Button>
          </Space>
        </Form.Item>
      </Form>
    </div>
  );

  const items = [
    {
      key: 'basic',
      label: (
        <Space>
          <GlobalOutlined />
          基础设置
        </Space>
      ),
      children: basicSettings,
    },
    {
      key: 'ai',
      label: (
        <Space>
          <RobotOutlined />
          AI 模型配置
        </Space>
      ),
      children: <AIModelSettingsPage />,
    },
  ];

  return (
    <div className="fade-in">
      <Card 
        style={{ background: '#16213e', border: '1px solid #2d3748' }}
        styles={{ body: { padding: '0 24px 24px 24px' } }}
      >
        <Tabs 
          activeKey={activeTab} 
          onChange={setActiveTab} 
          items={items}
          className="settings-tabs"
        />
      </Card>
    </div>
  );
};

export default SettingsPage;
