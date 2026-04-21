import React from 'react';
import { Alert, Button, Card, Form, Input, Typography } from 'antd';
import { Link, useNavigate } from 'react-router-dom';
import { useAuth } from '../../components/Auth/AuthContext';
import { GuidedFormItem } from '../../components/FormGuidance';

const { Title, Text } = Typography;
const REGISTER_ERROR_TEXT = '注册失败，请稍后重试';

const RegisterPage: React.FC = () => {
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);
  const { register } = useAuth();
  const navigate = useNavigate();

  const onFinish = async (values: { username: string; name?: string; email: string; password: string }) => {
    try {
      setLoading(true);
      setError(null);
      await register(values);
      navigate('/', { replace: true });
    } catch {
      setError(REGISTER_ERROR_TEXT);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-slate-100 px-4">
      <Card className="w-full max-w-md shadow-lg">
        <Title level={3}>注册 OpsPilot</Title>
        <Text type="secondary">创建账号并自动登录系统</Text>
        <Form layout="vertical" className="mt-6" onFinish={onFinish}>
          {error && <Alert type="error" message={error} className="mb-4" />}
          <GuidedFormItem name="username" label="用户名" rules={[{ required: true, message: '请输入用户名' }]}>
            <Input placeholder="ops_user" />
          </GuidedFormItem>
          <GuidedFormItem name="name" label="显示名称">
            <Input placeholder="运维同学" />
          </GuidedFormItem>
          <GuidedFormItem
            name="email"
            label="邮箱"
            rules={[{ required: true, message: '请输入邮箱' }, { type: 'email', message: '邮箱格式不正确' }]}
          >
            <Input placeholder="ops@example.com" />
          </GuidedFormItem>
          <GuidedFormItem
            name="password"
            label="密码"
            rules={[{ required: true, message: '请输入密码' }, { min: 6, message: '至少 6 位字符' }]}
          >
            <Input.Password placeholder="******" />
          </GuidedFormItem>
          <Button type="primary" htmlType="submit" loading={loading} block>
            注册并登录
          </Button>
          <div className="mt-4 text-center">
            <Text type="secondary">
              已有账号？<Link to="/login">去登录</Link>
            </Text>
          </div>
        </Form>
      </Card>
    </div>
  );
};

export default RegisterPage;
