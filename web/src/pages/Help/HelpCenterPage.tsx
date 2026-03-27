import React from 'react';
import { Card, Col, Input, Row, Space, Tag, Typography, Anchor, Divider } from 'antd';
import { BookOutlined, SearchOutlined } from '@ant-design/icons';
import HelpCenter from '../../components/Onboarding/HelpCenter';
import { helpDocuments } from '../../content/helpDocs';

const { Title, Text, Paragraph } = Typography;

const HelpCenterPage: React.FC = () => {
  const [query, setQuery] = React.useState('');

  const categories = React.useMemo(() => [...new Set(helpDocuments.map((item) => item.category))], []);

  const filtered = React.useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return helpDocuments;
    return helpDocuments.filter((item) => {
      return (
        item.title.toLowerCase().includes(q) ||
        item.content.toLowerCase().includes(q) ||
        item.tags.some((tag) => tag.toLowerCase().includes(q)) ||
        item.category.toLowerCase().includes(q)
      );
    });
  }, [query]);

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Card>
        <Space direction="vertical" size={6} style={{ width: '100%' }}>
          <Space>
            <BookOutlined style={{ color: 'var(--color-brand-400)' }} />
            <Title level={4} style={{ margin: 0 }}>
              帮助中心
            </Title>
          </Space>
          <Text type="secondary">面向运维工程师 / SRE 的操作手册、FAQ 与 AI 提问模板。</Text>
        </Space>
      </Card>

      <Row gutter={[16, 16]}>
        <Col xs={24} xl={17}>
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            <Card>
              <Input
                allowClear
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder="搜索功能、操作步骤、排障关键词"
                prefix={<SearchOutlined />}
              />
              <Divider style={{ margin: '12px 0' }} />
              <Space wrap>
                {categories.map((category) => (
                  <Tag key={category} color="blue">
                    {category}
                  </Tag>
                ))}
              </Space>
            </Card>

            <Card title="快速帮助卡片">
              <Space direction="vertical" size={12} style={{ width: '100%' }}>
                {filtered.map((doc) => (
                  <div key={doc.id} id={`help-${doc.id}`}>
                    <Space>
                      <Text strong>{doc.title}</Text>
                      <Tag>{doc.category}</Tag>
                      <Tag color={doc.difficulty === 'beginner' ? 'green' : doc.difficulty === 'intermediate' ? 'blue' : 'orange'}>
                        {doc.difficulty === 'beginner' ? '新手' : doc.difficulty === 'intermediate' ? '中级' : '高级'}
                      </Tag>
                    </Space>
                    <Paragraph style={{ marginTop: 4, marginBottom: 8, whiteSpace: 'pre-wrap' }}>{doc.content}</Paragraph>
                    <Space size={[4, 4]} wrap>
                      {doc.tags.map((tag) => (
                        <Tag key={`${doc.id}-${tag}`}>{tag}</Tag>
                      ))}
                    </Space>
                    <Divider style={{ margin: '12px 0' }} />
                  </div>
                ))}
              </Space>
            </Card>

            <HelpCenter />
          </Space>
        </Col>

        <Col xs={24} xl={7}>
          <Card title="文档目录" style={{ position: 'sticky', top: 88 }}>
            <Anchor
              items={filtered.map((doc) => ({
                key: doc.id,
                href: `#help-${doc.id}`,
                title: doc.title,
              }))}
            />
            <Divider />
            <Space direction="vertical" size={6}>
              <a href="/docs/user/help-center-manual.md" target="_blank" rel="noopener noreferrer">
                完整帮助文档
              </a>
              <a href="/docs/user/ops-faq-100.md" target="_blank" rel="noopener noreferrer">
                运维值班 FAQ 100 题
              </a>
              <a href="/docs/ai/help-knowledge-base.md" target="_blank" rel="noopener noreferrer">
                AI 帮助知识库
              </a>
            </Space>
          </Card>
        </Col>
      </Row>
    </Space>
  );
};

export default HelpCenterPage;
