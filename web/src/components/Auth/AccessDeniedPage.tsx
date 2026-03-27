import React from 'react';
import { Result, Button } from 'antd';
import { useNavigate } from 'react-router-dom';

const AccessDeniedPage: React.FC<{ compact?: boolean }> = ({ compact = false }) => {
  const navigate = useNavigate();

  return (
    <Result
      status="403"
      title="403"
      subTitle="您没有权限访问该资源，请联系管理员申请权限。"
      extra={
        compact ? null : (
          <Button type="primary" onClick={() => navigate('/')}>
            返回首页
          </Button>
        )
      }
    />
  );
};

export default AccessDeniedPage;
