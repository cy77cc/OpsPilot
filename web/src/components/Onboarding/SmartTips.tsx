import React, { useState, useEffect } from 'react';
import { Popover, Typography, Tag, Space, Button } from 'antd';
import { QuestionCircleOutlined, BulbOutlined, CheckCircleOutlined } from '@ant-design/icons';

const { Paragraph } = Typography;

// 智能提示数据结构
export interface SmartTip {
  id: string;
  title: string;
  content: string;
  type: 'info' | 'warning' | 'success' | 'error';
  action?: {
    label: string;
    handler: () => void;
  };
}

// 智能提示属性
interface SmartTipsProps {
  context: string;
  entityId?: string;
  position?: 'top' | 'bottom' | 'left' | 'right';
  className?: string;
}

// 智能提示组件
const SmartTips: React.FC<SmartTipsProps> = ({ 
  context, 
  position = 'top', 
  className 
}) => {
  const [tips, setTips] = useState<SmartTip[]>([]);

  // 加载智能提示
  const loadSmartTips = async () => {
    try {
      // 这里应该调用后端API获取智能提示
      // 暂时使用模拟数据
      const mockTips: SmartTip[] = getMockTips(context);
      setTips(mockTips);
    } catch (error) {
      console.error('加载智能提示失败:', error);
      setTips([]);
    }
  };

  // 获取模拟提示数据
  const getMockTips = (context: string): SmartTip[] => {
    const tipsMap: Record<string, SmartTip[]> = {
      'hosts': [
        {
          id: 'tip-1',
          title: '主机监控',
          content: '建议定期检查主机状态，确保系统稳定运行。',
          type: 'info'
        },
        {
          id: 'tip-2',
          title: '性能优化',
          content: '根据主机负载情况，适时调整资源配置以提高性能。',
          type: 'warning',
          action: {
            label: '查看详情',
            handler: () => console.log('查看性能详情')
          }
        }
      ],
      'tasks': [
        {
          id: 'tip-3',
          title: '任务管理',
          content: '合理安排任务调度，避免高峰期集中执行导致系统负载过高。',
          type: 'info'
        }
      ],
      'kubernetes': [
        {
          id: 'tip-4',
          title: '集群健康',
          content: '定期检查集群状态，确保所有节点正常运行。',
          type: 'info'
        },
        {
          id: 'tip-5',
          title: '资源管理',
          content: '合理配置Pod资源限制，避免资源浪费和不足。',
          type: 'warning'
        }
      ],
      'monitoring': [
        {
          id: 'tip-6',
          title: '告警设置',
          content: '根据业务需求，设置合理的告警阈值，及时发现和处理问题。',
          type: 'info'
        }
      ],
      'default': [
        {
          id: 'tip-7',
          title: '新手提示',
          content: '如果您有任何问题，可以随时咨询AI助手获取帮助。',
          type: 'success'
        }
      ]
    };

    return tipsMap[context] || tipsMap['default'];
  };

  // 初始化加载提示
  useEffect(() => {
    loadSmartTips();
  }, [context]);

  // 获取提示类型对应的样式
  const getTipTypeConfig = (type: SmartTip['type']) => {
    const configs = {
      info: { color: '#1890ff', icon: <QuestionCircleOutlined /> },
      warning: { color: '#faad14', icon: <BulbOutlined /> },
      success: { color: '#52c41a', icon: <CheckCircleOutlined /> },
      error: { color: '#f5222d', icon: <QuestionCircleOutlined /> },
    };
    return configs[type];
  };

  if (tips.length === 0) {
    return null;
  }

  return (
    <Space 
      direction="vertical" 
      size="small" 
      className={`smart-tips ${className || ''}`}
      style={{ width: '100%' }}
    >
      {tips.map((tip) => {
        const config = getTipTypeConfig(tip.type);
        return (
          <Popover
            key={tip.id}
            placement={position}
            title={
              <Space>
                <Tag color={config.color} icon={config.icon}>{tip.title}</Tag>
              </Space>
            }
            content={
              <div>
                <Paragraph>{tip.content}</Paragraph>
                {tip.action && (
                  <Button 
                    size="small" 
                    type="link" 
                    onClick={tip.action.handler}
                  >
                    {tip.action.label}
                  </Button>
                )}
              </div>
            }
            trigger="hover"
          >
            <Tag 
              color={config.color} 
              icon={config.icon} 
              style={{ cursor: 'pointer' }}
            >
              {tip.title}
            </Tag>
          </Popover>
        );
      })}
    </Space>
  );
};

export default SmartTips;