/**
 * Feedback & Layout Components Showcase - Phase 2.3
 *
 * 测试以下组件的样式重构：
 * - Notification (左侧色条、动画)
 * - Message (样式)
 * - Progress (色彩、动画)
 * - Spin (样式)
 * - Layout (侧边栏、顶部导航)
 */

import { useState } from 'react';
import {
  Card,
  Space,
  Button,
  notification,
  message,
  Progress,
  Spin,
  Layout,
  Menu,
  Avatar,
  Dropdown,
  Breadcrumb,
  Divider
} from 'antd';
import {
  CheckCircleOutlined,
  InfoCircleOutlined,
  WarningOutlined,
  CloseCircleOutlined,
  LoadingOutlined,
  UserOutlined,
  HomeOutlined,
  AppstoreOutlined,
  SettingOutlined,
  BellOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined
} from '@ant-design/icons';

const { Header, Sider, Content } = Layout;

export const FeedbackLayoutShowcase = () => {
  const [loading, setLoading] = useState(false);
  const [collapsed, setCollapsed] = useState(false);
  const [messageApi, contextHolder] = message.useMessage();
  const [notificationApi, notificationContextHolder] = notification.useNotification();

  // Notification 示例
  const openNotification = (type: 'success' | 'info' | 'warning' | 'error') => {
    const config = {
      success: {
        message: '操作成功',
        description: '您的操作已成功完成，系统已保存所有更改。',
        icon: <CheckCircleOutlined style={{ color: '#10b981' }} />
      },
      info: {
        message: '系统通知',
        description: '有新的系统更新可用，建议您尽快更新以获得最佳体验。',
        icon: <InfoCircleOutlined style={{ color: '#3b82f6' }} />
      },
      warning: {
        message: '警告提示',
        description: '检测到异常活动，请检查您的账户安全设置。',
        icon: <WarningOutlined style={{ color: '#f59e0b' }} />
      },
      error: {
        message: '操作失败',
        description: '操作失败，请稍后重试或联系系统管理员。',
        icon: <CloseCircleOutlined style={{ color: '#ef4444' }} />
      }
    };

    notificationApi[type]({
      ...config[type],
      placement: 'topRight',
      duration: 4.5
    });
  };

  // Message 示例
  const showMessage = (type: 'success' | 'info' | 'warning' | 'error' | 'loading') => {
    const messages = {
      success: '操作成功！',
      info: '这是一条信息提示',
      warning: '请注意此操作',
      error: '操作失败，请重试',
      loading: '正在处理中...'
    };

    messageApi[type](messages[type]);
  };

  // 模拟加载
  const handleLoad = () => {
    setLoading(true);
    setTimeout(() => setLoading(false), 3000);
  };

  return (
    <div className="space-y-8">
      {contextHolder}
      {notificationContextHolder}

      <div className="p-8 bg-gray-50">
        <div className="max-w-7xl mx-auto space-y-8">
          <div>
            <h1 className="text-3xl font-semibold text-gray-900 mb-2">反馈与布局组件</h1>
            <p className="text-gray-500 mb-8">Phase 2.3 反馈组件和布局组件样式重构验证</p>
          </div>

          {/* 2.3.1 Notification 组件 */}
          <Card title="2.3.1 Notification 组件 - 左侧色条、动画" className="mb-6">
            <Space direction="vertical" size="large" className="w-full">
              <div>
                <h3 className="text-sm font-medium text-gray-700 mb-3">通知类型</h3>
                <Space wrap>
                  <Button
                    type="primary"
                    icon={<CheckCircleOutlined />}
                    onClick={() => openNotification('success')}
                  >
                    成功通知
                  </Button>
                  <Button
                    icon={<InfoCircleOutlined />}
                    onClick={() => openNotification('info')}
                  >
                    信息通知
                  </Button>
                  <Button
                    icon={<WarningOutlined />}
                    onClick={() => openNotification('warning')}
                  >
                    警告通知
                  </Button>
                  <Button
                    danger
                    icon={<CloseCircleOutlined />}
                    onClick={() => openNotification('error')}
                  >
                    错误通知
                  </Button>
                </Space>
              </div>

              <div className="bg-gray-100 p-4 rounded-lg">
                <p className="text-sm text-gray-600">
                  ✓ 宽度: 384px<br/>
                  ✓ 圆角: 8px (md)<br/>
                  ✓ 背景: 白色<br/>
                  ✓ 阴影: 中等阴影效果<br/>
                  ✓ 图标: 使用语义色 (success, info, warning, error)<br/>
                  ✓ 动画: 从右侧滑入，淡出效果<br/>
                  ✓ 位置: topRight (右上角)
                </p>
              </div>
            </Space>
          </Card>

          {/* 2.3.2 Message 组件 */}
          <Card title="2.3.2 Message 组件 - 样式" className="mb-6">
            <Space direction="vertical" size="large" className="w-full">
              <div>
                <h3 className="text-sm font-medium text-gray-700 mb-3">消息类型</h3>
                <Space wrap>
                  <Button
                    type="primary"
                    icon={<CheckCircleOutlined />}
                    onClick={() => showMessage('success')}
                  >
                    成功消息
                  </Button>
                  <Button
                    icon={<InfoCircleOutlined />}
                    onClick={() => showMessage('info')}
                  >
                    信息消息
                  </Button>
                  <Button
                    icon={<WarningOutlined />}
                    onClick={() => showMessage('warning')}
                  >
                    警告消息
                  </Button>
                  <Button
                    danger
                    icon={<CloseCircleOutlined />}
                    onClick={() => showMessage('error')}
                  >
                    错误消息
                  </Button>
                  <Button
                    icon={<LoadingOutlined />}
                    onClick={() => showMessage('loading')}
                  >
                    加载消息
                  </Button>
                </Space>
              </div>

              <div className="bg-gray-100 p-4 rounded-lg">
                <p className="text-sm text-gray-600">
                  ✓ 背景: 白色<br/>
                  ✓ 圆角: 8px (md)<br/>
                  ✓ 阴影: 中等阴影<br/>
                  ✓ 图标: 使用语义色<br/>
                  ✓ 动画: 从顶部滑入，淡出效果<br/>
                  ✓ 位置: top (顶部居中)<br/>
                  ✓ 持续时间: 3秒自动关闭
                </p>
              </div>
            </Space>
          </Card>

          {/* 2.3.3 Progress 组件 */}
          <Card title="2.3.3 Progress 组件 - 色彩、动画" className="mb-6">
            <Space direction="vertical" size="large" className="w-full">
              <div>
                <h3 className="text-sm font-medium text-gray-700 mb-3">进度条类型</h3>
                <Space direction="vertical" className="w-full">
                  <div>
                    <p className="text-sm text-gray-600 mb-2">默认进度条</p>
                    <Progress percent={30} />
                  </div>
                  <div>
                    <p className="text-sm text-gray-600 mb-2">成功状态</p>
                    <Progress percent={100} status="success" />
                  </div>
                  <div>
                    <p className="text-sm text-gray-600 mb-2">异常状态</p>
                    <Progress percent={70} status="exception" />
                  </div>
                  <div>
                    <p className="text-sm text-gray-600 mb-2">进行中</p>
                    <Progress percent={50} status="active" />
                  </div>
                </Space>
              </div>

              <div>
                <h3 className="text-sm font-medium text-gray-700 mb-3">圆形进度条</h3>
                <Space size="large">
                  <Progress type="circle" percent={75} />
                  <Progress type="circle" percent={100} status="success" />
                  <Progress type="circle" percent={70} status="exception" />
                </Space>
              </div>

              <div>
                <h3 className="text-sm font-medium text-gray-700 mb-3">仪表盘进度条</h3>
                <Space size="large">
                  <Progress type="dashboard" percent={75} />
                  <Progress type="dashboard" percent={100} status="success" />
                </Space>
              </div>

              <div>
                <h3 className="text-sm font-medium text-gray-700 mb-3">小型进度条</h3>
                <Space direction="vertical" className="w-full">
                  <Progress percent={30} size="small" />
                  <Progress percent={50} size="small" status="active" />
                  <Progress percent={100} size="small" status="success" />
                </Space>
              </div>

              <div className="bg-gray-100 p-4 rounded-lg">
                <p className="text-sm text-gray-600">
                  ✓ 默认颜色: #6366f1 (Primary 500)<br/>
                  ✓ 剩余颜色: #e9ecef (Gray 200)<br/>
                  ✓ 成功色: #10b981 (Success)<br/>
                  ✓ 异常色: #ef4444 (Error)<br/>
                  ✓ 动画: active 状态有流动动画<br/>
                  ✓ 圆角: 进度条有圆角效果
                </p>
              </div>
            </Space>
          </Card>

          {/* 2.3.4 Spin 组件 */}
          <Card title="2.3.4 Spin 组件 - 样式" className="mb-6">
            <Space direction="vertical" size="large" className="w-full">
              <div>
                <h3 className="text-sm font-medium text-gray-700 mb-3">加载指示器</h3>
                <Space size="large" align="center">
                  <Spin size="small" />
                  <Spin />
                  <Spin size="large" />
                </Space>
              </div>

              <div>
                <h3 className="text-sm font-medium text-gray-700 mb-3">自定义图标</h3>
                <Spin indicator={<LoadingOutlined style={{ fontSize: 24 }} spin />} />
              </div>

              <div>
                <h3 className="text-sm font-medium text-gray-700 mb-3">容器加载</h3>
                <Space direction="vertical" className="w-full">
                  <Button onClick={handleLoad}>切换加载状态</Button>
                  <Card>
                    <Spin spinning={loading} tip="加载中...">
                      <div className="p-8">
                        <p className="text-gray-700">这是一段示例内容</p>
                        <p className="text-sm text-gray-500 mt-2">
                          当加载状态为 true 时，会显示加载遮罩层
                        </p>
                      </div>
                    </Spin>
                  </Card>
                </Space>
              </div>

              <div className="bg-gray-100 p-4 rounded-lg">
                <p className="text-sm text-gray-600">
                  ✓ 主色: #6366f1 (Primary 500)<br/>
                  ✓ 尺寸: small (16px), default (32px), large (40px)<br/>
                  ✓ 动画: 流畅的旋转动画<br/>
                  ✓ 遮罩: 半透明白色背景<br/>
                  ✓ 提示文字: 可自定义加载提示
                </p>
              </div>
            </Space>
          </Card>

          {/* 2.3.5 Layout 组件 */}
          <Card title="2.3.5 Layout 组件 - 侧边栏、顶部导航" className="mb-6">
            <Space direction="vertical" size="large" className="w-full">
              <div>
                <h3 className="text-sm font-medium text-gray-700 mb-3">布局示例</h3>
                <div className="border border-gray-200 rounded-lg overflow-hidden">
                  <Layout style={{ minHeight: '400px' }}>
                    <Sider
                      collapsible
                      collapsed={collapsed}
                      onCollapse={setCollapsed}
                      theme="light"
                      width={240}
                    >
                      <div className="h-16 flex items-center justify-center border-b border-gray-200">
                        <span className="text-lg font-semibold text-primary-600">
                          {collapsed ? 'OP' : 'OpsPilot'}
                        </span>
                      </div>
                      <Menu
                        mode="inline"
                        defaultSelectedKeys={['1']}
                        items={[
                          {
                            key: '1',
                            icon: <HomeOutlined />,
                            label: '主控台'
                          },
                          {
                            key: '2',
                            icon: <AppstoreOutlined />,
                            label: '服务管理'
                          },
                          {
                            key: '3',
                            icon: <SettingOutlined />,
                            label: '系统设置'
                          }
                        ]}
                      />
                    </Sider>
                    <Layout>
                      <Header className="bg-white border-b border-gray-200 px-6 flex items-center justify-between">
                        <div className="flex items-center space-x-4">
                          <Button
                            type="text"
                            icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
                            onClick={() => setCollapsed(!collapsed)}
                          />
                          <Breadcrumb
                            items={[
                              { title: '首页' },
                              { title: '主控台' }
                            ]}
                          />
                        </div>
                        <Space>
                          <Button type="text" icon={<BellOutlined />} />
                          <Dropdown
                            menu={{
                              items: [
                                { key: '1', label: '个人中心' },
                                { key: '2', label: '退出登录' }
                              ]
                            }}
                          >
                            <Avatar icon={<UserOutlined />} className="bg-primary-500 cursor-pointer" />
                          </Dropdown>
                        </Space>
                      </Header>
                      <Content className="m-6">
                        <Card>
                          <h3 className="text-lg font-medium text-gray-900 mb-2">内容区域</h3>
                          <p className="text-gray-600">
                            这是主要内容区域，展示了完整的布局结构。
                          </p>
                          <Divider />
                          <p className="text-sm text-gray-500">
                            侧边栏可以折叠，顶部导航包含面包屑、通知和用户菜单。
                          </p>
                        </Card>
                      </Content>
                    </Layout>
                  </Layout>
                </div>
              </div>

              <div className="bg-gray-100 p-4 rounded-lg">
                <p className="text-sm text-gray-600">
                  ✓ Header 背景: 白色<br/>
                  ✓ Header 高度: 64px<br/>
                  ✓ Header 内边距: 0 32px<br/>
                  ✓ Sider 背景: 白色 (light 主题)<br/>
                  ✓ Sider 宽度: 240px (展开), 80px (折叠)<br/>
                  ✓ Body 背景: #fafbfc (Gray 50)<br/>
                  ✓ 边框: #e9ecef (Gray 200)<br/>
                  ✓ 菜单: 选中项背景 #eef2ff (Primary 50)
                </p>
              </div>
            </Space>
          </Card>

          {/* 总结 */}
          <Card className="bg-primary-50 border-primary-200">
            <h3 className="text-lg font-semibold text-primary-700 mb-3">Phase 2.3 验证总结</h3>
            <div className="space-y-2 text-sm text-gray-700">
              <p>✓ Notification: 宽度 384px, 圆角 8px, 从右侧滑入动画</p>
              <p>✓ Message: 白色背景, 圆角 8px, 从顶部滑入动画</p>
              <p>✓ Progress: 主色 #6366f1, 剩余色 #e9ecef, active 状态有流动动画</p>
              <p>✓ Spin: 主色 #6366f1, 流畅旋转动画, 支持容器加载</p>
              <p>✓ Layout: Header 64px, Sider 240px, 白色主题, 可折叠侧边栏</p>
            </div>
          </Card>
        </div>
      </div>
    </div>
  );
};
