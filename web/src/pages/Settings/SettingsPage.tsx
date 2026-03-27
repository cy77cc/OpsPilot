import React from 'react';
import { Card, Form, Input, Button, Switch, Select, Divider, Space, message, Alert } from 'antd';
import { SaveOutlined, UserOutlined, LockOutlined, BellOutlined, GlobalOutlined } from '@ant-design/icons';

const { Option } = Select;

const SettingsPage: React.FC = () => {
  const [form] = Form.useForm();

  const handleSave = () => {
    message.success('设置已保存');
  };

  return (
    <div className="fade-in">
      <Card 
        style={{ background: '#16213e', border: '1px solid #2d3748' }}
        title={<span className="text-white text-lg">系统设置</span>}
      >
        <Alert
          type="info"
          showIcon
          className="mb-4"
          message="用户、角色、权限管理已迁移到侧边导航“访问治理”菜单。"
        />
        <Form form={form} layout="vertical" style={{ maxWidth: 600 }}>
          <Divider><UserOutlined /> 个人信息</Divider>
          
          <Form.Item label="用户名" name="username" initialValue="admin">
            <Input placeholder="请输入用户名" />
          </Form.Item>
          
          <Form.Item label="邮箱" name="email" initialValue="admin@company.com">
            <Input placeholder="请输入邮箱" />
          </Form.Item>
          
          <Form.Item label="手机号" name="phone" initialValue="138****8888">
            <Input placeholder="请输入手机号" />
          </Form.Item>

          <Divider><LockOutlined /> 安全设置</Divider>
          
          <Form.Item label="当前密码" name="currentPassword">
            <Input.Password placeholder="请输入当前密码" />
          </Form.Item>
          
          <Form.Item label="新密码" name="newPassword">
            <Input.Password placeholder="请输入新密码" />
          </Form.Item>
          
          <Form.Item label="确认密码" name="confirmPassword">
            <Input.Password placeholder="请确认新密码" />
          </Form.Item>

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
      </Card>
    </div>
  );
};

export default SettingsPage;
