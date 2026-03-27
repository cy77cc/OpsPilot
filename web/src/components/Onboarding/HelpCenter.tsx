import React, { useState } from 'react';
import { Card, Typography, Collapse, Space, Button, Input, Tag } from 'antd';
import { QuestionCircleOutlined, SearchOutlined, BookOutlined, MessageOutlined } from '@ant-design/icons';
import { helpDocuments } from '../../content/helpDocs';
import type { HelpDocument } from '../../content/helpDocs';

const { Text, Paragraph } = Typography;
const { Panel } = Collapse;
const { Search } = Input;

// 帮助中心属性
interface HelpCenterProps {
  onAskAI?: () => void;
  className?: string;
}

// 帮助中心组件
const HelpCenter: React.FC<HelpCenterProps> = ({ 
  onAskAI, 
  className 
}) => {
  const [searchValue, setSearchValue] = useState('');
  const [activeCategory, setActiveCategory] = useState<string | null>(null);

  // 获取所有分类
  const categories = [...new Set(helpDocuments.map(doc => doc.category))];

  // 过滤文档
  const filteredDocuments = helpDocuments.filter(doc => {
    const matchesSearch = doc.title.toLowerCase().includes(searchValue.toLowerCase()) || 
                        doc.content.toLowerCase().includes(searchValue.toLowerCase()) ||
                        doc.tags.some(tag => tag.toLowerCase().includes(searchValue.toLowerCase()));
    const matchesCategory = !activeCategory || doc.category === activeCategory;
    return matchesSearch && matchesCategory;
  });

  // 获取难度对应的颜色
  const getDifficultyColor = (difficulty: HelpDocument['difficulty']) => {
    const colors = {
      beginner: 'green',
      intermediate: 'blue',
      advanced: 'orange'
    };
    return colors[difficulty];
  };

  // 获取难度对应的文本
  const getDifficultyText = (difficulty: HelpDocument['difficulty']) => {
    const texts = {
      beginner: '新手',
      intermediate: '中级',
      advanced: '高级'
    };
    return texts[difficulty];
  };

  return (
    <Card 
      title={
        <Space>
          <QuestionCircleOutlined />
          <Text strong>帮助中心</Text>
        </Space>
      }
      className={`help-center ${className || ''}`}
    >
      <Space direction="vertical" style={{ width: '100%', marginBottom: '16px' }}>
        {/* 搜索框 */}
        <Search
          placeholder="搜索帮助文档"
          allowClear
          enterButton={<SearchOutlined />}
          value={searchValue}
          onChange={(e) => setSearchValue(e.target.value)}
          style={{ width: '100%', maxWidth: 600, marginBottom: '16px' }}
        />

        {/* 分类标签 */}
        <Space wrap style={{ marginBottom: '16px' }}>
          <Tag 
            color={!activeCategory ? 'blue' : 'default'}
            onClick={() => setActiveCategory(null)}
          >
            全部
          </Tag>
          {categories.map(category => (
            <Tag
              key={category}
              color={activeCategory === category ? 'blue' : 'default'}
              onClick={() => setActiveCategory(category)}
            >
              {category}
            </Tag>
          ))}
        </Space>

        {/* 文档列表 */}
        <Collapse
          defaultActiveKey={[]}
          style={{ width: '100%' }}
        >
          {filteredDocuments.map(doc => (
            <Panel
              key={doc.id}
              header={
                <Space style={{ width: '100%', justifyContent: 'space-between' }}>
                  <Text strong>{doc.title}</Text>
                  <Space>
                    <Tag color={getDifficultyColor(doc.difficulty)}>
                      {getDifficultyText(doc.difficulty)}
                    </Tag>
                    <Tag>{doc.category}</Tag>
                  </Space>
                </Space>
              }
            >
              <Paragraph>
                {doc.content.split('\n').map((line, index) => (
                  <div key={index}>{line}</div>
                ))}
              </Paragraph>
              <Space>
                {doc.tags.map(tag => (
                  <Tag key={tag}>{tag}</Tag>
                ))}
              </Space>
            </Panel>
          ))}
        </Collapse>

        {/* 空状态 */}
        {filteredDocuments.length === 0 && (
          <div style={{ padding: '32px', textAlign: 'center' }}>
            <Text type="secondary">未找到相关帮助文档</Text>
          </div>
        )}

        {/* 底部操作 */}
        <Space style={{ marginTop: '16px', justifyContent: 'center' }}>
          <Button 
            type="primary" 
            icon={<MessageOutlined />} 
            onClick={onAskAI}
          >
            咨询AI助手
          </Button>
          <Button 
            icon={<BookOutlined />}
            onClick={() => window.open('/docs/user/help-center-manual.md', '_blank', 'noopener,noreferrer')}
          >
            查看完整文档
          </Button>
        </Space>
      </Space>
    </Card>
  );
};

export default HelpCenter;
