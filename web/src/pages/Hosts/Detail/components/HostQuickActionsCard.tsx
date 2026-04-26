import React from 'react';
import { Card, Row, Col, Button, Modal, message } from 'antd';
import {
  ConsoleSqlOutlined,
  FileSearchOutlined,
  CodeOutlined,
  ReloadOutlined,
  PoweroffOutlined,
  AppstoreAddOutlined,
  SettingOutlined,
  EllipsisOutlined,
} from '@ant-design/icons';

interface HostQuickActionsCardProps {
  onAction: (action: string) => void;
}

const HostQuickActionsCard: React.FC<HostQuickActionsCardProps> = ({ onAction }) => {
  const actions = [
    { key: 'terminal', label: '连接终端', icon: <ConsoleSqlOutlined /> },
    { key: 'files', label: '文件管理', icon: <FileSearchOutlined /> },
    { key: 'exec', label: '执行命令', icon: <CodeOutlined /> },
    { key: 'reboot', label: '重启主机', icon: <ReloadOutlined />, danger: true },
    { key: 'shutdown', label: '关机', icon: <PoweroffOutlined />, danger: true },
    { key: 'install', label: '安装软件', icon: <AppstoreAddOutlined /> },
    { key: 'config', label: '配置管理', icon: <SettingOutlined /> },
    { key: 'more', label: '更多操作', icon: <EllipsisOutlined /> },
  ];

  const handleAction = (key: string, label: string, isDanger?: boolean) => {
    if (isDanger) {
      Modal.confirm({
        title: `确认执行 ${label}`,
        content: `确定要对当前主机执行 ${label} 操作吗？`,
        okText: '确定',
        cancelText: '取消',
        okButtonProps: { danger: true },
        onOk: () => {
          onAction(key);
          message.success(`${label} 指令已发送`);
        },
      });
    } else {
      onAction(key);
    }
  };

  return (
    <Card title="快捷操作" className="h-full">
      <Row gutter={[12, 12]}>
        {actions.map((action) => (
          <Col span={6} key={action.key}>
            <Button
              className="w-full h-16 flex flex-col items-center justify-center p-2"
              icon={React.cloneElement(action.icon as React.ReactElement, { style: { fontSize: '20px', marginBottom: '4px' } })}
              onClick={() => handleAction(action.key, action.label, action.danger)}
              danger={action.danger}
            >
              <span className="text-xs">{action.label}</span>
            </Button>
          </Col>
        ))}
      </Row>
    </Card>
  );
};

export default HostQuickActionsCard;
