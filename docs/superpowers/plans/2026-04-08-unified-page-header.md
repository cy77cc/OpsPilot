# 统一 PageHeader 组件实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 创建统一的 PageHeader 组件，为所有二级页面提供一致的面包屑导航和返回按钮体验。

**Architecture:** 创建 PageHeader 目录包含三个子组件：BreadcrumbNav（可点击面包屑）、BackButton（智能返回按钮）、PageHeader（主组件组合两者）。使用 TDD 方式开发，先写测试再实现。

**Tech Stack:** React, TypeScript, Ant Design (Breadcrumb, Button), React Router (useNavigate, Link), Vitest, Tailwind CSS

---

## 文件结构

```
web/src/components/PageHeader/
├── BreadcrumbNav.tsx          # 可点击面包屑子组件
├── BreadcrumbNav.test.tsx     # 面包屑测试
├── BackButton.tsx             # 智能返回按钮子组件
├── BackButton.test.tsx        # 返回按钮测试
├── PageHeader.tsx             # 主组件
├── PageHeader.test.tsx        # 主组件测试
└── index.ts                   # 导出
```

---

### Task 1: 创建 BreadcrumbNav 组件

**Files:**
- Create: `web/src/components/PageHeader/BreadcrumbNav.tsx`
- Create: `web/src/components/PageHeader/BreadcrumbNav.test.tsx`

- [ ] **Step 1: 编写 BreadcrumbNav 测试文件**

创建 `web/src/components/PageHeader/BreadcrumbNav.test.tsx`：

```tsx
import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { BreadcrumbNav } from './BreadcrumbNav';
import { renderWithProviders } from '../../test/utils/render';

// Mock navigate to track clicks
const mockNavigate = vi.fn();
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

describe('BreadcrumbNav', () => {
  beforeEach(() => {
    mockNavigate.mockClear();
  });

  it('renders breadcrumb items with correct titles', () => {
    renderWithProviders(
      <BreadcrumbNav
        items={[
          { title: '主机管理', path: '/hosts' },
          { title: '主机详情' }
        ]}
      />
    );

    expect(screen.getByText('主机管理')).toBeInTheDocument();
    expect(screen.getByText('主机详情')).toBeInTheDocument();
  });

  it('renders clickable links for items with path', () => {
    renderWithProviders(
      <BreadcrumbNav
        items={[
          { title: '主机管理', path: '/hosts' },
          { title: '主机详情' }
        ]}
      />
    );

    // Item with path should be a link
    const link = screen.getByRole('link', { name: '主机管理' });
    expect(link).toHaveAttribute('href', '/hosts');
  });

  it('renders non-clickable text for items without path', () => {
    renderWithProviders(
      <BreadcrumbNav
        items={[
          { title: '主机管理', path: '/hosts' },
          { title: '主机详情' }
        ]}
      />
    );

    // Last item without path should not be a link
    const lastItem = screen.getByText('主机详情');
    expect(lastItem.closest('a')).toBeNull();
  });

  it('supports multiple levels with paths', () => {
    renderWithProviders(
      <BreadcrumbNav
        items={[
          { title: '首页', path: '/' },
          { title: '主机管理', path: '/hosts' },
          { title: '集群详情', path: '/hosts/1' },
          { title: '编辑' }
        ]}
      />
    );

    expect(screen.getByRole('link', { name: '首页' })).toHaveAttribute('href', '/');
    expect(screen.getByRole('link', { name: '主机管理' })).toHaveAttribute('href', '/hosts');
    expect(screen.getByRole('link', { name: '集群详情' })).toHaveAttribute('href', '/hosts/1');
    expect(screen.getByText('编辑').closest('a')).toBeNull();
  });

  it('applies custom className', () => {
    renderWithProviders(
      <BreadcrumbNav
        items={[
          { title: '主机管理', path: '/hosts' },
          { title: '主机详情' }
        ]}
        className="custom-class"
      />
    );

    const breadcrumb = screen.getByRole('navigation');
    expect(breadcrumb).toHaveClass('custom-class');
  });
});
```

- [ ] **Step 2: 运行测试验证失败**

Run: `cd web && npm test -- --run src/components/PageHeader/BreadcrumbNav.test.tsx`
Expected: FAIL with "Cannot find module './BreadcrumbNav'"

- [ ] **Step 3: 创建 BreadcrumbNav 组件目录**

Run: `mkdir -p web/src/components/PageHeader`

- [ ] **Step 4: 实现 BreadcrumbNav 组件**

创建 `web/src/components/PageHeader/BreadcrumbNav.tsx`：

```tsx
import React from 'react';
import { Breadcrumb } from 'antd';
import { Link } from 'react-router-dom';

export interface BreadcrumbItem {
  title: string;
  path?: string;
}

export interface BreadcrumbNavProps {
  items: BreadcrumbItem[];
  className?: string;
}

export const BreadcrumbNav: React.FC<BreadcrumbNavProps> = ({ items, className }) => {
  const breadcrumbItems = items.map((item) => ({
    title: item.path ? (
      <Link to={item.path}>{item.title}</Link>
    ) : (
      <span className="text-gray-900 font-medium">{item.title}</span>
    ),
  }));

  return (
    <Breadcrumb
      className={className}
      items={breadcrumbItems}
      separator="/"
    />
  );
};
```

- [ ] **Step 5: 运行测试验证通过**

Run: `cd web && npm test -- --run src/components/PageHeader/BreadcrumbNav.test.tsx`
Expected: PASS (5 tests)

- [ ] **Step 6: 提交**

```bash
git add web/src/components/PageHeader/BreadcrumbNav.tsx web/src/components/PageHeader/BreadcrumbNav.test.tsx
git commit -m "feat: add BreadcrumbNav component with clickable navigation"
```

---

### Task 2: 创建 BackButton 组件

**Files:**
- Create: `web/src/components/PageHeader/BackButton.tsx`
- Create: `web/src/components/PageHeader/BackButton.test.tsx`

- [ ] **Step 1: 编写 BackButton 测试文件**

创建 `web/src/components/PageHeader/BackButton.test.tsx`：

```tsx
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { BackButton } from './BackButton';
import { renderWithProviders } from '../../test/utils/render';

const mockNavigate = vi.fn();
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

describe('BackButton', () => {
  const user = userEvent.setup();

  beforeEach(() => {
    mockNavigate.mockClear();
    // Reset window.history and document.referrer mocks
    vi.stubGlobal('history', { length: 2 });
    vi.stubGlobal('document', {
      referrer: 'http://localhost:3000/hosts',
      location: { origin: 'http://localhost:3000' },
    });
  });

  it('renders with dynamic parent title', () => {
    renderWithProviders(
      <BackButton fallbackPath="/hosts" parentTitle="主机管理" />
    );

    expect(screen.getByRole('button', { name: '返回主机管理' })).toBeInTheDocument();
  });

  it('uses browser history when available', async () => {
    renderWithProviders(
      <BackButton fallbackPath="/hosts" parentTitle="主机管理" />
    );

    await user.click(screen.getByRole('button'));

    expect(mockNavigate).toHaveBeenCalledWith(-1);
    expect(mockNavigate).not.toHaveBeenCalledWith('/hosts');
  });

  it('falls back to preset path when no history', async () => {
    // Mock no history (external entry)
    vi.stubGlobal('history', { length: 1 });
    vi.stubGlobal('document', {
      referrer: 'https://external-site.com',
      location: { origin: 'http://localhost:3000' },
    });

    renderWithProviders(
      <BackButton fallbackPath="/hosts" parentTitle="主机管理" />
    );

    await user.click(screen.getByRole('button'));

    expect(mockNavigate).toHaveBeenCalledWith('/hosts');
  });

  it('falls back when history exists but referrer is external', async () => {
    // History exists but came from external site
    vi.stubGlobal('history', { length: 5 });
    vi.stubGlobal('document', {
      referrer: 'https://google.com/search',
      location: { origin: 'http://localhost:3000' },
    });

    renderWithProviders(
      <BackButton fallbackPath="/hosts" parentTitle="主机管理" />
    );

    await user.click(screen.getByRole('button'));

    expect(mockNavigate).toHaveBeenCalledWith('/hosts');
  });

  it('applies custom className', () => {
    renderWithProviders(
      <BackButton
        fallbackPath="/hosts"
        parentTitle="主机管理"
        className="custom-class"
      />
    );

    const button = screen.getByRole('button');
    expect(button).toHaveClass('custom-class');
  });

  it('does not render when disabled prop is true', () => {
    renderWithProviders(
      <BackButton fallbackPath="/hosts" parentTitle="主机管理" disabled />
    );

    const button = screen.getByRole('button');
    expect(button).toBeDisabled();
  });
});
```

- [ ] **Step 2: 运行测试验证失败**

Run: `cd web && npm test -- --run src/components/PageHeader/BackButton.test.tsx`
Expected: FAIL with "Cannot find module './BackButton'"

- [ ] **Step 3: 实现 BackButton 组件**

创建 `web/src/components/PageHeader/BackButton.tsx`：

```tsx
import React from 'react';
import { Button } from 'antd';
import { ArrowLeftOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';

export interface BackButtonProps {
  fallbackPath: string;
  parentTitle: string;
  className?: string;
  disabled?: boolean;
}

export const BackButton: React.FC<BackButtonProps> = ({
  fallbackPath,
  parentTitle,
  className,
  disabled = false,
}) => {
  const navigate = useNavigate();

  const handleBack = () => {
    // Check if there's browser history from this site
    const hasHistory = window.history.length > 1;
    const isFromThisSite = document.referrer.includes(window.location.origin);

    if (hasHistory && isFromThisSite) {
      navigate(-1);
    } else {
      navigate(fallbackPath);
    }
  };

  return (
    <Button
      icon={<ArrowLeftOutlined />}
      onClick={handleBack}
      className={className}
      disabled={disabled}
    >
      返回{parentTitle}
    </Button>
  );
};
```

- [ ] **Step 4: 运行测试验证通过**

Run: `cd web && npm test -- --run src/components/PageHeader/BackButton.test.tsx`
Expected: PASS (6 tests)

- [ ] **Step 5: 提交**

```bash
git add web/src/components/PageHeader/BackButton.tsx web/src/components/PageHeader/BackButton.test.tsx
git commit -m "feat: add BackButton component with smart history fallback"
```

---

### Task 3: 创建 PageHeader 主组件

**Files:**
- Create: `web/src/components/PageHeader/PageHeader.tsx`
- Create: `web/src/components/PageHeader/PageHeader.test.tsx`
- Create: `web/src/components/PageHeader/index.ts`

- [ ] **Step 1: 编写 PageHeader 测试文件**

创建 `web/src/components/PageHeader/PageHeader.test.tsx`：

```tsx
import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { PageHeader } from './PageHeader';
import { renderWithProviders } from '../../test/utils/render';
import { Button } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';

const mockNavigate = vi.fn();
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

describe('PageHeader', () => {
  const user = userEvent.setup();

  beforeEach(() => {
    mockNavigate.mockClear();
    vi.stubGlobal('history', { length: 2 });
    vi.stubGlobal('document', {
      referrer: 'http://localhost:3000/hosts',
      location: { origin: 'http://localhost:3000' },
    });
  });

  it('renders breadcrumb and title', () => {
    renderWithProviders(
      <PageHeader
        breadcrumbItems={[
          { title: '主机管理', path: '/hosts' },
          { title: '主机详情' }
        ]}
        title="主机名称"
      />
    );

    expect(screen.getByText('主机管理')).toBeInTheDocument();
    expect(screen.getByText('主机详情')).toBeInTheDocument();
    expect(screen.getByText('主机名称')).toBeInTheDocument();
  });

  it('renders back button when backButton prop provided', () => {
    renderWithProviders(
      <PageHeader
        breadcrumbItems={[
          { title: '主机管理', path: '/hosts' },
          { title: '主机详情' }
        ]}
        title="主机名称"
        backButton={{
          fallbackPath: '/hosts',
          parentTitle: '主机管理',
        }}
      />
    );

    expect(screen.getByRole('button', { name: '返回主机管理' })).toBeInTheDocument();
  });

  it('does not render back button when backButton not provided', () => {
    renderWithProviders(
      <PageHeader
        breadcrumbItems={[{ title: '主机管理' }]}
        title="主机管理"
      />
    );

    expect(screen.queryByRole('button', { name: /返回/ })).toBeNull();
  });

  it('renders extra action buttons on the right', () => {
    renderWithProviders(
      <PageHeader
        breadcrumbItems={[
          { title: '主机管理', path: '/hosts' },
          { title: '主机详情' }
        ]}
        title="主机名称"
        backButton={{
          fallbackPath: '/hosts',
          parentTitle: '主机管理',
        }}
        extra={
          <Button icon={<ReloadOutlined />}>刷新</Button>
        }
      />
    );

    expect(screen.getByRole('button', { name: '刷新' })).toBeInTheDocument();
  });

  it('back button navigates correctly on click', async () => {
    renderWithProviders(
      <PageHeader
        breadcrumbItems={[
          { title: '主机管理', path: '/hosts' },
          { title: '主机详情' }
        ]}
        title="主机名称"
        backButton={{
          fallbackPath: '/hosts',
          parentTitle: '主机管理',
        }}
      />
    );

    await user.click(screen.getByRole('button', { name: '返回主机管理' }));

    expect(mockNavigate).toHaveBeenCalledWith(-1);
  });

  it('renders title as ReactNode', () => {
    renderWithProviders(
      <PageHeader
        breadcrumbItems={[{ title: '主机管理' }]}
        title={<span className="custom-title">自定义标题</span>}
      />
    );

    expect(screen.getByText('自定义标题')).toBeInTheDocument();
    expect(screen.getByText('自定义标题')).toHaveClass('custom-title');
  });

  it('shows loading skeleton when loading prop is true', () => {
    renderWithProviders(
      <PageHeader
        breadcrumbItems={[
          { title: '主机管理', path: '/hosts' },
          { title: '主机详情' }
        ]}
        title="主机名称"
        loading
      />
    );

    // Title area should show skeleton instead of actual title
    const skeleton = screen.getByTestId('page-header-skeleton');
    expect(skeleton).toBeInTheDocument();
  });

  it('applies custom className to container', () => {
    renderWithProviders(
      <PageHeader
        breadcrumbItems={[{ title: '主机管理' }]}
        title="主机管理"
        className="custom-container"
      />
    );

    const container = screen.getByTestId('page-header');
    expect(container).toHaveClass('custom-container');
  });
});
```

- [ ] **Step 2: 运行测试验证失败**

Run: `cd web && npm test -- --run src/components/PageHeader/PageHeader.test.tsx`
Expected: FAIL with "Cannot find module './PageHeader'"

- [ ] **Step 3: 实现 PageHeader 主组件**

创建 `web/src/components/PageHeader/PageHeader.tsx`：

```tsx
import React from 'react';
import { Space, Skeleton } from 'antd';
import { BreadcrumbNav, BreadcrumbItem } from './BreadcrumbNav';
import { BackButton } from './BackButton';

export interface PageHeaderProps {
  breadcrumbItems: BreadcrumbItem[];
  title: React.ReactNode;
  backButton?: {
    fallbackPath: string;
    parentTitle: string;
  };
  extra?: React.ReactNode;
  loading?: boolean;
  className?: string;
}

export const PageHeader: React.FC<PageHeaderProps> = ({
  breadcrumbItems,
  title,
  backButton,
  extra,
  loading = false,
  className,
}) => {
  return (
    <div data-testid="page-header" className={`mb-4 ${className || ''}`}>
      {/* Breadcrumb row */}
      <BreadcrumbNav items={breadcrumbItems} className="mb-2" />

      {/* Title row with back button and extra */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          {backButton && (
            <BackButton
              fallbackPath={backButton.fallbackPath}
              parentTitle={backButton.parentTitle}
            />
          )}
          <div className="text-xl font-semibold text-gray-900">
            {loading ? (
              <Skeleton.Input
                data-testid="page-header-skeleton"
                active
                size="small"
                style={{ width: 200 }}
              />
            ) : (
              title
            )}
          </div>
        </div>
        {extra && <Space>{extra}</Space>}
      </div>
    </div>
  );
};
```

- [ ] **Step 4: 创建 index.ts 导出文件**

创建 `web/src/components/PageHeader/index.ts`：

```tsx
export { PageHeader } from './PageHeader';
export type { PageHeaderProps } from './PageHeader';

export { BreadcrumbNav } from './BreadcrumbNav';
export type { BreadcrumbNavProps, BreadcrumbItem } from './BreadcrumbNav';

export { BackButton } from './BackButton';
export type { BackButtonProps } from './BackButton';
```

- [ ] **Step 5: 运行测试验证通过**

Run: `cd web && npm test -- --run src/components/PageHeader/PageHeader.test.tsx`
Expected: PASS (8 tests)

- [ ] **Step 6: 运行所有 PageHeader 测试**

Run: `cd web && npm test -- --run src/components/PageHeader`
Expected: PASS (all tests)

- [ ] **Step 7: 提交**

```bash
git add web/src/components/PageHeader/
git commit -m "feat: add PageHeader component with breadcrumb, back button, and action area"
```

---

### Task 4: 示例迁移 - HostDetailPage

**Files:**
- Modify: `web/src/pages/Hosts/HostDetailPage.tsx`

- [ ] **Step 1: 读取 HostDetailPage 当前实现**

Run: Read file to understand current structure (lines 220-240)

- [ ] **Step 2: 在 HostDetailPage 中导入 PageHeader**

修改 `web/src/pages/Hosts/HostDetailPage.tsx`，在 import 部分添加：

```tsx
import { PageHeader } from '../../components/PageHeader';
```

移除不再需要的导入：
```tsx
// 移除: ArrowLeftOutlined (如果仅用于返回按钮)
// 移除: Breadcrumb (如果仅用于页面面包屑)
// 移除: Link (如果仅用于面包屑跳转)
```

- [ ] **Step 3: 替换面包屑和返回按钮**

找到原有的面包屑和 Card title 部分（约 line 220-230），替换为：

**原代码（删除）：**
```tsx
<Breadcrumb
  className="mb-4"
  items={[
    { title: <Link to="/hosts">主机管理</Link> },
    { title: host?.name || id },
  ]}
/>

<Card
  title={<Space><Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/deployment/infrastructure/hosts')}>返回</Button><span>{host?.name || '主机详情'}</span></Space>}
  extra={...}
>
```

**新代码：**
```tsx
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
      <Button icon={<ReloadOutlined />} onClick={() => void load()} loading={loading && !isInitialLoading}>刷新</Button>
      <Button icon={<EditOutlined />} onClick={openEditModal}>编辑主机</Button>
      <Button onClick={() => navigate(`/hosts/${id}/terminal`)}>终端</Button>
      <Button onClick={() => void runHealthCheck()}>健康检查</Button>
      <Button onClick={() => runAction('restart', true)}>重启</Button>
      <Button danger onClick={() => runAction('shutdown', true)}>关机</Button>
    </Space>
  }
/>

<Card>
```

- [ ] **Step 4: 验证页面正常运行**

Run: `cd web && npm run dev`
手动检查：浏览器访问主机详情页，确认面包屑可点击、返回按钮正常工作

- [ ] **Step 5: 提交**

```bash
git add web/src/pages/Hosts/HostDetailPage.tsx
git commit -m "refactor: use PageHeader component in HostDetailPage"
```

---

### Task 5: 最终验证和清理

- [ ] **Step 1: 运行完整测试套件**

Run: `cd web && npm test -- --run`
Expected: All tests PASS

- [ ] **Step 2: TypeScript 类型检查**

Run: `cd web && npm run typecheck` (or `tsc --noEmit`)
Expected: No type errors

- [ ] **Step 3: ESLint 检查**

Run: `cd web && npm run lint`
Expected: No errors (warnings acceptable)

- [ ] **Step 4: 最终提交（如有遗漏）**

```bash
git status
# 如果有未提交的更改，提交它们
git add -A
git commit -m "chore: final cleanup for PageHeader implementation"
```

---

## Spec Coverage 检查

| Spec 要求 | 覆盖任务 |
|----------|---------|
| 组件 Props 定义 | Task 3 |
| 面包屑所有层级可点击 | Task 1 |
| 返回按钮浏览器历史优先 | Task 2 |
| 无历史时回退预设父级 | Task 2 |
| 动态显示上级页面名称 | Task 2 |
| 右侧操作按钮区域 | Task 3 |
| loading 状态支持 | Task 3 |
| className 支持 | Task 1, 2, 3 |
| 示例迁移 | Task 4 |

## Placeholder 扫描结果

无 TBD、TODO、模糊描述。所有步骤包含完整代码。

## 类型一致性检查

- `BreadcrumbItem` 在 BreadcrumbNav.tsx 和 PageHeader.tsx 中使用一致
- `BackButtonProps.fallbackPath: string` 与 PageHeader 中使用一致
- `BackButtonProps.parentTitle: string` 与 PageHeader 中使用一致