# OpsPilot 交互动画与切换效果升级实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 升级 OpsPilot 前端的页面切换和组件交互动画，引入“iOS/Jelly”风格的弹性反馈，提升整体产品的质感和响应感。

**Architecture:** 
1. 统一并增强 `PageTransition` 组件，使用 `framer-motion` 的 `Spring` 物理引擎实现带缩放和位移的弹性过渡。
2. 升级 `AnimatedButton` 和 `AnimatedCard` 基础组件，增加基于物理刚度（Stiffness）和阻尼（Damping）的点击/悬浮反馈。
3. 优化全局布局（AppLayout）和根包裹（ProtectedApp），确保动画层级正确且无冗余嵌套。

**Tech Stack:** React, framer-motion, antd

---

### Task 1: 升级核心 PageTransition 组件

**Files:**
- Modify: `web/src/components/Motion/PageTransition.tsx`
- Modify: `web/src/components/PageTransition/index.ts`

- [ ] **Step 1: 更新 PageTransition 实现 iOS/Jelly 弹性过渡**

```tsx
// web/src/components/Motion/PageTransition.tsx
import React from 'react';
import { useLocation } from 'react-router-dom';
import { AnimatePresence, motion } from 'framer-motion';

export interface PageTransitionProps {
  children: React.ReactNode;
}

const pageVariants = {
  initial: {
    opacity: 0,
    scale: 0.96,
    y: 10,
  },
  enter: {
    opacity: 1,
    scale: 1,
    y: 0,
    transition: {
      type: 'spring',
      stiffness: 300,
      damping: 30,
      mass: 1,
    },
  },
  exit: {
    opacity: 0,
    scale: 1.02,
    transition: {
      duration: 0.2,
      ease: 'easeIn',
    },
  },
};

export const PageTransition: React.FC<PageTransitionProps> = ({ children }) => {
  const location = useLocation();

  React.useEffect(() => {
    window.scrollTo(0, 0);
  }, [location.pathname]);

  return (
    <AnimatePresence mode="wait" initial={false}>
      <motion.div
        key={location.pathname}
        initial="initial"
        animate="enter"
        exit="exit"
        variants={pageVariants}
        style={{ width: '100%', height: '100%', position: 'relative' }}
      >
        {children}
      </motion.div>
    </AnimatePresence>
  );
};

export default PageTransition;
```

- [ ] **Step 2: 统一导出路径**

```typescript
// web/src/components/PageTransition/index.ts
export { PageTransition as default } from '../Motion/PageTransition';
```

- [ ] **Step 3: 提交更改**

```bash
git add web/src/components/Motion/PageTransition.tsx web/src/components/PageTransition/index.ts
git commit -m "feat(animation): upgrade PageTransition with spring physics and unify exports"
```

---

### Task 2: 增强基础交互组件 (Button & Card)

**Files:**
- Modify: `web/src/components/AnimatedButton/AnimatedButton.tsx`
- Modify: `web/src/components/AnimatedCard/AnimatedCard.tsx`

- [ ] **Step 1: 升级 AnimatedButton 弹性反馈**

```tsx
// web/src/components/AnimatedButton/AnimatedButton.tsx
import { motion } from 'framer-motion';
import React from 'react';

interface AnimatedButtonProps {
  children: React.ReactNode;
  className?: string;
  onClick?: () => void;
  type?: 'button' | 'submit' | 'reset';
  disabled?: boolean;
}

const AnimatedButton: React.FC<AnimatedButtonProps> = ({
  children,
  className = '',
  onClick,
  type = 'button',
  disabled = false,
}) => {
  return (
    <motion.button
      type={type}
      className={className}
      onClick={onClick}
      disabled={disabled}
      whileHover={!disabled ? { scale: 1.02 } : undefined}
      whileTap={!disabled ? { scale: 0.94 } : undefined}
      transition={{
        type: 'spring',
        stiffness: 400,
        damping: 17,
      }}
    >
      {children}
    </motion.button>
  );
};

export default AnimatedButton;
```

- [ ] **Step 2: 升级 AnimatedCard 悬浮与点击反馈**

```tsx
// web/src/components/AnimatedCard/AnimatedCard.tsx
import { motion } from 'framer-motion';
import React from 'react';

interface AnimatedCardProps {
  children: React.ReactNode;
  className?: string;
  onClick?: () => void;
}

const AnimatedCard: React.FC<AnimatedCardProps> = ({
  children,
  className = '',
  onClick,
}) => {
  return (
    <motion.div
      className={`bg-white rounded-xl shadow-sm border border-gray-100 p-4 cursor-pointer ${className}`}
      onClick={onClick}
      whileHover={{ 
        y: -4, 
        scale: 1.01, 
        boxShadow: "0 10px 25px -5px rgba(0, 0, 0, 0.1), 0 8px 10px -6px rgba(0, 0, 0, 0.1)" 
      }}
      whileTap={{ scale: 0.98 }}
      transition={{
        type: 'spring',
        stiffness: 400,
        damping: 17,
      }}
    >
      {children}
    </motion.div>
  );
};

export default AnimatedCard;
```

- [ ] **Step 3: 提交更改**

```bash
git add web/src/components/AnimatedButton/AnimatedButton.tsx web/src/components/AnimatedCard/AnimatedCard.tsx
git commit -m "feat(animation): enhance AnimatedButton and AnimatedCard with jelly spring effects"
```

---

### Task 3: 优化布局动画集成 (AppLayout & ProtectedApp)

**Files:**
- Modify: `web/src/app/layout/AppLayout.tsx`
- Modify: `web/src/ProtectedApp.tsx`

- [ ] **Step 1: 清理 ProtectedApp 中的冗余 PageTransition**

```tsx
// web/src/ProtectedApp.tsx
// ... (imports remain the same)

export default function ProtectedApp() {
  const { user } = useAuth();
  const governanceMenuEnabled = import.meta.env.VITE_FEATURE_GOVERNANCE_MENU !== 'false';
  const withAuth = createWithAuth();

  return (
    <PermissionProvider>
      <NotificationProvider userId={user?.id}>
        <AppLayout>
          {/* 移除此处的 PageTransition，交给 AppLayout 内部统一处理或保持一层结构 */}
          <Suspense fallback={<PageSkeleton />}>
            <ProtectedRoutes withAuth={withAuth} governanceMenuEnabled={governanceMenuEnabled} />
          </Suspense>
        </AppLayout>
      </NotificationProvider>
    </PermissionProvider>
  );
}
```

- [ ] **Step 2: 增强 AppLayout 内部动画 (侧边栏与激活项)**

```tsx
// web/src/app/layout/AppLayout.tsx 关键部分修改示例
// 在渲染菜单项时，可以考虑增加 layoutId 动画（可选，视具体重构工作量而定）
// 至少确保 PageTransition 正确包裹 Content 区域
// ... (此处描述逻辑更新，实施时需定位到 Content 渲染处)
```

- [ ] **Step 3: 提交更改**

```bash
git add web/src/ProtectedApp.tsx web/src/app/layout/AppLayout.tsx
git commit -m "refactor(layout): unify animation hierarchy and clean up redundant transitions"
```

---

### Task 4: 清理旧组件与验证

**Files:**
- Delete: `web/src/components/PageTransition/PageTransition.tsx`

- [ ] **Step 1: 删除已废弃的旧版 PageTransition 文件**

Run: `rm web/src/components/PageTransition/PageTransition.tsx`

- [ ] **Step 2: 运行 Lint 和单元测试确保无破坏性改动**

Run: `npm run lint` (in web directory)
Run: `npm run test` (in web directory)

- [ ] **Step 3: 提交清理**

```bash
git add .
git commit -m "cleanup: remove redundant PageTransition implementation"
```
