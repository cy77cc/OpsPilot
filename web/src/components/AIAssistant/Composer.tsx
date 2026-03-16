import React, { useState } from 'react';
import { Button, Input, Space } from 'antd';

interface ComposerProps {
  loading?: boolean;
  onSubmit: (message: string) => Promise<void> | void;
}

const Composer: React.FC<ComposerProps> = ({ loading, onSubmit }) => {
  const [value, setValue] = useState('');

  return (
    <Space.Compact style={{ width: '100%' }}>
      <Input
        value={value}
        onChange={(event) => setValue(event.target.value)}
        onPressEnter={() => {
          if (!value.trim()) return;
          void onSubmit(value.trim());
          setValue('');
        }}
        placeholder="询问平台问题，或请求诊断故障"
      />
      <Button
        type="primary"
        aria-label="发送消息"
        loading={loading}
        onClick={() => {
          if (!value.trim()) return;
          void onSubmit(value.trim());
          setValue('');
        }}
      >
        发送
      </Button>
    </Space.Compact>
  );
};

export default Composer;
