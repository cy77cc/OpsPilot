# 统一 PageHeader 组件设计

## 背景

当前系统的二级页面（详情页、编辑页、创建页等）存在以下问题：

1. **返回操作不一致** - 每个页面单独实现返回按钮，样式和行为各异
2. **路径硬编码** - 所有返回按钮使用 `navigate('/固定路径')`，而非浏览器历史
3. **面包屑导航受限** - 只有首页可点击，中间层级无法跳转
4. **代码重复** - 56 个 Page 组件中大量重复的面包屑和返回按钮代码

## 目标

创建统一的 `PageHeader` 组件，为所有二级页面提供一致的导航体验。

## 设计

### 组件 Props

```typescript
interface PageHeaderProps {
  // 面包屑配置 - 所有层级可点击
  breadcrumbItems: Array<{ title: string; path?: string }>;
  
  // 页面标题
  title: string | ReactNode;
  
  // 返回配置（可选，一级页面不需要）
  backButton?: {
    fallbackPath: string;      // 无历史时回退路径
    parentTitle: string;       // 动态显示的上级名称
  };
  
  // 右侧操作按钮区域（可选）
  extra?: ReactNode;
  
  // 其他可选配置
  loading?: boolean;           // 标题加载状态
  className?: string;
}
```

### 组件布局

```
┌─────────────────────────────────────────────────────────────┐
│ 面包屑：主机管理 > 集群详情                                    │
├─────────────────────────────────────────────────────────────┤
│ ← 返回主机列表    主机名称-001               [刷新][编辑][终端]│
│                   (标题)                      (操作按钮区)   │
└─────────────────────────────────────────────────────────────┘
```

布局说明：
- 第一行：面包屑导航，带 `path` 的层级可点击跳转
- 第二行：返回按钮（左侧）+ 标题（中间偏左）+ 操作按钮（右侧）

### 返回按钮行为

优先使用浏览器历史返回，无历史时回退预设父级：

```typescript
const handleBack = () => {
  if (window.history.length > 1 && document.referrer.includes(window.location.origin)) {
    navigate(-1);
  } else {
    navigate(fallbackPath);
  }
};
```

显示文字动态显示上级页面名称，如 "返回主机管理"、"返回集群列表"。

### 面包屑导航

所有带 `path` 的层级都可点击：

```tsx
<Breadcrumb
  items={breadcrumbItems.map((item, index) => ({
    title: item.path 
      ? <Link to={item.path}>{item.title}</Link>
      : <span>{item.title}</span>  // 最后一项不可点击
  }))}
/>
```

### 文件组织

```
web/src/components/PageHeader/
├── PageHeader.tsx          # 主组件
├── PageHeader.test.tsx     # 测试文件
├── BackButton.tsx          # 返回按钮子组件
├── BreadcrumbNav.tsx       # 可点击面包屑子组件
└── index.ts                # 导出
```

## 使用示例

### 详情页（带返回按钮）

```tsx
// HostDetailPage.tsx
<PageHeader
  breadcrumbItems={[
    { title: '主机管理', path: '/hosts' },
    { title: host?.name || '主机详情' }
  ]}
  title={host?.name || '主机详情'}
  backButton={{
    fallbackPath: '/hosts',
    parentTitle: '主机管理'
  }}
  extra={
    <Space>
      <Button icon={<ReloadOutlined />} onClick={load}>刷新</Button>
      <Button icon={<EditOutlined />} onClick={openEditModal}>编辑</Button>
    </Space>
  }
/>
<Card>...内容...</Card>
```

### 一级页面（无返回按钮）

```tsx
// HostListPage.tsx
<PageHeader
  breadcrumbItems={[{ title: '主机管理' }]}
  title="主机管理"
/>
```

### 创建/编辑页

```tsx
// HostEditPage.tsx
<PageHeader
  breadcrumbItems={[
    { title: '主机管理', path: '/hosts' },
    { title: host?.name || '新主机', path: `/hosts/${id}` },
    { title: '编辑' }
  ]}
  title={host?.name ? `编辑 ${host.name}` : '创建主机'}
  backButton={{
    fallbackPath: `/hosts/${id}`,
    parentTitle: host?.name || '主机详情'
  }}
/>
```

## 适用范围

所有二级页面：
- 详情页（HostDetailPage, ClusterDetailPage, ServiceDetailPage 等）
- 编辑页
- 创建页
- 设置页
- 向导页（Wizard）
- 终端页等子功能页

## 迁移计划

1. 创建 PageHeader 组件及子组件
2. 编写单元测试
3. 逐页替换现有面包屑和返回按钮实现
4. 移除页面中的重复代码

## 技术要点

- 使用 `react-router-dom` 的 `useNavigate` hook
- 面包屑使用 Ant Design 的 `Breadcrumb` 组件
- 返回按钮使用 Ant Design 的 `Button` 组件 + `ArrowLeftOutlined` 图标
- 样式继承现有项目风格，使用 Tailwind CSS 辅助布局