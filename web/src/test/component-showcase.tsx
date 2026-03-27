/**
 * Component Showcase - Phase 2.1 基础组件测试
 *
 * 测试以下组件的样式重构：
 * - Button (圆角、阴影、动画)
 * - Input (focus 状态、error 状态)
 * - Card (圆角、阴影、hover 效果)
 * - Tag (色彩方案)
 * - Modal (圆角、阴影、背景模糊)
 */

import { useState } from 'react';
import { Button, Card, Input, Tag, Modal, Space, Divider, Form } from 'antd';
import {
  SearchOutlined,
  PlusOutlined,
  DeleteOutlined,
  EditOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  ExclamationCircleOutlined,
  InfoCircleOutlined
} from '@ant-design/icons';

export const ComponentShowcase = () => {
  const [modalVisible, setModalVisible] = useState(false);
  const [form] = Form.useForm();

  return (
    <div className="p-8 space-y-8 bg-gray-50 min-h-screen">
      <div className="max-w-7xl mx-auto">
        <h1 className="text-3xl font-semibold text-gray-900 mb-2">组件展示</h1>
        <p className="text-gray-500 mb-8">Phase 2.1 基础组件样式重构验证</p>

        {/* 2.1.1 Button 组件 */}
        <Card title="2.1.1 Button 组件 - 圆角、阴影、动画" className="mb-6">
          <Space direction="vertical" size="large" className="w-full">
            <div>
              <h3 className="text-sm font-medium text-gray-700 mb-3">按钮类型</h3>
              <Space wrap>
                <Button type="primary">Primary Button</Button>
                <Button>Default Button</Button>
                <Button type="dashed">Dashed Button</Button>
                <Button type="text">Text Button</Button>
                <Button type="link">Link Button</Button>
              </Space>
            </div>

            <div>
              <h3 className="text-sm font-medium text-gray-700 mb-3">按钮尺寸</h3>
              <Space wrap>
                <Button type="primary" size="large">Large Button</Button>
                <Button type="primary">Default Button</Button>
                <Button type="primary" size="small">Small Button</Button>
              </Space>
            </div>

            <div>
              <h3 className="text-sm font-medium text-gray-700 mb-3">带图标的按钮</h3>
              <Space wrap>
                <Button type="primary" icon={<PlusOutlined />}>新建</Button>
                <Button icon={<SearchOutlined />}>搜索</Button>
                <Button icon={<EditOutlined />}>编辑</Button>
                <Button danger icon={<DeleteOutlined />}>删除</Button>
      </Space>
            </div>

            <div>
              <h3 className="text-sm font-medium text-gray-700 mb-3">按钮状态</h3>
              <Space wrap>
                <Button type="primary" loading>Loading</Button>
                <Button type="primary" disabled>Disabled</Button>
                <Button danger>Danger</Button>
                <Button type="primary" ghost>Ghost</Button>
              </Space>
            </div>

            <div className="bg-gray-100 p-4 rounded-lg">
              <p className="text-sm text-gray-600">
                ✓ 圆角: 8px (borderRadius)<br/>
                ✓ 高度: 40px (default), 48px (large), 32px (small)<br/>
                ✓ 字重: 500 (medium)<br/>
                ✓ 阴影: 主按钮有微妙阴影<br/>
                ✓ 动画: hover 和 active 状态有过渡效果
              </p>
            </div>
          </Space>
        </Card>

        {/* 2.1.2 Input 组件 */}
        <Card title="2.1.2 Input 组件 - Focus 状态、Error 状态" className="mb-6">
          <Space direction="vertical" size="large" className="w-full">
            <div>
              <h3 className="text-sm font-medium text-gray-700 mb-3">输入框类型</h3>
              <Space direction="vertical" className="w-full max-w-md">
                <Input placeholder="默认输入框" />
                <Input placeholder="带前缀图标" prefix={<SearchOutlined />} />
                <Input placeholder="带后缀图标" suffix={<InfoCircleOutlined />} />
                <Input.Password placeholder="密码输入框" />
                <Input.TextArea placeholder="文本域" rows={3} />
              </Space>
            </div>

            <div>
              <h3 className="text-sm font-medium text-gray-700 mb-3">输入框尺寸</h3>
              <Space direction="vertical" className="w-full max-w-md">
                <Input size="large" placeholder="Large Input" />
                <Input placeholder="Default Input" />
                <Input size="small" placeholder="Small Input" />
              </Space>
            </div>

            <div>
              <h3 className="text-sm font-medium text-gray-700 mb-3">输入框状态</h3>
              <Form form={form} layout="vertical" className="max-w-md">
                <Form.Item label="正常状态">
                  <Input placeholder="输入内容" />
                </Form.Item>
                <Form.Item
                  label="错误状态"
                  validateStatus="error"
                  help="请输入正确的内容"
                >
                  <Input placeholder="错误输入" />
                </Form.Item>
                <Form.Item
                  label="警告状态"
                  validateStatus="warning"
                  help="建议修改此内容"
                >
                  <Input placeholder="警告输入" />
                </Form.Item>
                <Form.Item label="禁用状态">
                  <Input placeholder="禁用输入" disabled />
                </Form.Item>
              </Form>
            </div>

            <div className="bg-gray-100 p-4 rounded-lg">
              <p className="text-sm text-gray-600">
                ✓ 圆角: 8px (borderRadius)<br/>
                ✓ 高度: 40px (default), 48px (large), 32px (small)<br/>
                ✓ 边框: 1px solid #dee2e6<br/>
                ✓ Focus: 2px outline in primary-500 (#6366f1)<br/>
                ✓ Error: 红色边框和错误提示
              </p>
            </div>
          </Space>
        </Card>

        {/* 2.1.3 Card 组件 */}
        <Card title="2.1.3 Card 组件 - 圆角、阴影、Hover 效果" className="mb-6">
          <Space direction="vertical" size="large" className="w-full">
            <div>
              <h3 className="text-sm font-medium text-gray-700 mb-3">卡片类型</h3>
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <Card>
                  <p className="text-gray-700">默认卡片</p>
                  <p className="text-sm text-gray-500 mt-2">这是一个默认样式的卡片</p>
                </Card>
                <Card size="small">
                  <p className="text-gray-700">小尺寸卡片</p>
                  <p className="text-sm text-gray-500 mt-2">使用 size="small"</p>
                </Card>
                <Card hoverable>
                  <p className="text-gray-700">可悬停卡片</p>
                  <p className="text-sm text-gray-500 mt-2">鼠标悬停有效果</p>
                </Card>
              </div>
            </div>

            <div>
              <h3 className="text-sm font-medium text-gray-700 mb-3">带标题的卡片</h3>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <Card title="卡片标题" extra={<Button type="link">更多</Button>}>
                  <p className="text-gray-700">卡片内容区域</p>
                  <p className="text-sm text-gray-500 mt-2">这里是卡片的主要内容</p>
                </Card>
                <Card
                  title="操作卡片"
                  extra={<Button type="primary" size="small">操作</Button>}
                  actions={[
                    <Button type="text" key="edit">编辑</Button>,
                    <Button type="text" key="delete" danger>删除</Button>,
                  ]}
                >
                  <p className="text-gray-700">带操作按钮的卡片</p>
                </Card>
              </div>
            </div>

            <div className="bg-gray-100 p-4 rounded-lg">
              <p className="text-sm text-gray-600">
                ✓ 圆角: 12px (borderRadiusLG)<br/>
                ✓ 阴影: 0 1px 3px rgba(0,0,0,0.08)<br/>
                ✓ 背景: 白色<br/>
                ✓ 边框: 1px solid #e9ecef<br/>
                ✓ Hover: hoverable 卡片有阴影提升效果
              </p>
            </div>
          </Space>
        </Card>

        {/* 2.1.4 Tag 组件 */}
        <Card title="2.1.4 Tag 组件 - 色彩方案" className="mb-6">
          <Space direction="vertical" size="large" className="w-full">
            <div>
              <h3 className="text-sm font-medium text-gray-700 mb-3">语义色标签</h3>
              <Space wrap>
                <Tag color="success" icon={<CheckCircleOutlined />}>Success</Tag>
                <Tag color="processing" icon={<InfoCircleOutlined />}>Processing</Tag>
                <Tag color="warning" icon={<ExclamationCircleOutlined />}>Warning</Tag>
                <Tag color="error" icon={<CloseCircleOutlined />}>Error</Tag>
                <Tag color="default">Default</Tag>
              </Space>
            </div>

            <div>
              <h3 className="text-sm font-medium text-gray-700 mb-3">预设颜色</h3>
              <Space wrap>
                <Tag color="blue">Blue</Tag>
                <Tag color="green">Green</Tag>
                <Tag color="red">Red</Tag>
                <Tag color="orange">Orange</Tag>
                <Tag color="purple">Purple</Tag>
                <Tag color="cyan">Cyan</Tag>
                <Tag color="magenta">Magenta</Tag>
              </Space>
            </div>

            <div>
              <h3 className="text-sm font-medium text-gray-700 mb-3">可关闭标签</h3>
              <Space wrap>
                <Tag closable>可关闭</Tag>
                <Tag closable color="blue">蓝色标签</Tag>
                <Tag closable color="success">成功标签</Tag>
              </Space>
            </div>

            <div>
              <h3 className="text-sm font-medium text-gray-700 mb-3">状态标签示例</h3>
              <Space wrap>
                <Tag color="success">运行中</Tag>
                <Tag color="processing">部署中</Tag>
                <Tag color="warning">警告</Tag>
                <Tag color="error">失败</Tag>
                <Tag color="default">已停止</Tag>
              </Space>
            </div>

            <div className="bg-gray-100 p-4 rounded-lg">
              <p className="text-sm text-gray-600">
                ✓ 圆角: 4px (borderRadiusSM)<br/>
                ✓ 默认背景: #f8f9fa (Gray 100)<br/>
                ✓ 默认文字: #495057 (Gray 700)<br/>
                ✓ 语义色: success (#10b981), warning (#f59e0b), error (#ef4444)
              </p>
            </div>
          </Space>
        </Card>

        {/* 2.1.5 Modal 组件 */}
        <Card title="2.1.5 Modal 组件 - 圆角、阴影、背景模糊" className="mb-6">
          <Space direction="vertical" size="large" className="w-full">
            <div>
              <h3 className="text-sm font-medium text-gray-700 mb-3">对话框示例</h3>
              <Space wrap>
                <Button type="primary" onClick={() => setModalVisible(true)}>
                  打开对话框
                </Button>
                <Button onClick={() => Modal.info({ title: '信息', content: '这是一个信息提示' })}>
                  信息提示
                </Button>
                <Button onClick={() => Modal.success({ title: '成功', content: '操作成功完成' })}>
                  成功提示
                </Button>
                <Button onClick={() => Modal.warning({ title: '警告', content: '请注意此操作' })}>
                  警告提示
                </Button>
                <Button onClick={() => Modal.error({ title: '错误', content: '操作失败' })}>
                  错误提示
                </Button>
                <Button
                  danger
                  onClick={() => Modal.confirm({
                    title: '确认删除',
                    content: '删除后无法恢复，确定要删除吗？',
                    okText: '确定',
                    cancelText: '取消'
                  })}
                >
                  确认对话框
                </Button>
              </Space>
            </div>

            <div className="bg-gray-100 p-4 rounded-lg">
              <p className="text-sm text-gray-600">
                ✓ 圆角: 12px (borderRadiusLG)<br/>
                ✓ 阴影: 0 20px 25px -5px rgba(0,0,0,0.1) (2xl)<br/>
                ✓ 背景: 白色<br/>
                ✓ 遮罩: rgba(0,0,0,0.45)<br/>
                ✓ 动画: 淡入淡出效果
              </p>
            </div>
          </Space>
        </Card>

        {/* Modal 实例 */}
        <Modal
          title="示例对话框"
          open={modalVisible}
          onOk={() => setModalVisible(false)}
          onCancel={() => setModalVisible(false)}
          okText="确定"
          cancelText="取消"
        >
          <p className="text-gray-700">这是一个示例对话框</p>
          <p className="text-sm text-gray-500 mt-2">
            对话框使用了 12px 圆角和大阴影效果，背景有半透明遮罩。
          </p>
          <Divider />
          <Form layout="vertical">
            <Form.Item label="输入示例">
              <Input placeholder="在对话框中输入内容" />
            </Form.Item>
            <Form.Item label="选择示例">
              <Space>
                <Tag color="blue">选项 1</Tag>
                <Tag color="green">选项 2</Tag>
                <Tag color="orange">选项 3</Tag>
              </Space>
            </Form.Item>
          </Form>
        </Modal>

        {/* 总结 */}
        <Card className="bg-primary-50 border-primary-200">
          <h3 className="text-lg font-semibold text-primary-700 mb-3">Phase 2.1 验证总结</h3>
          <div className="space-y-2 text-sm text-gray-700">
            <p>✓ Button: 圆角 8px, 高度 40px, 字重 500, 有阴影和动画效果</p>
            <p>✓ Input: 圆角 8px, 高度 40px, focus 状态有 primary 色 outline, error 状态正确显示</p>
            <p>✓ Card: 圆角 12px, 有阴影, hoverable 卡片有提升效果</p>
            <p>✓ Tag: 圆角 4px, 语义色正确应用, 默认样式符合设计规范</p>
            <p>✓ Modal: 圆角 12px, 大阴影, 遮罩半透明, 动画流畅</p>
          </div>
        </Card>
      </div>
    </div>
  );
};
