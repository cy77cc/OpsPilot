# OpsPilot 前端交互动画与切换效果升级设计文档

## 1. 背景与目标
目前 OpsPilot 前端在页面切换和组件交互上表现较为生硬，缺乏物理反馈。本项目旨在通过引入“iOS/Jelly”风格的弹性动画，提升系统的灵活性、响应感和整体质感。

## 2. 核心动画参数 (Spring Physics)
我们将使用 `framer-motion` 的弹簧动力学（Spring Physics）替代传统的持续时间（Duration）动画。

| 场景 | Stiffness (刚度) | Damping (阻尼) | Mass (质量) | 说明 |
| :--- | :--- | :--- | :--- | :--- |
| **页面切换** | 300 | 30 | 1 | 稳重平滑，避免眩晕感 |
| **基础交互 (点击/悬浮)** | 400 | 17 | 1 | 快速响应，富有弹性 |
| **列表错开加载 (Stagger)** | - | - | - | 延迟间隔: 0.05s |

## 3. 详细设计方案

### 3.1 页面切换 (Page Transitions)
*   **统一组件**：清理 `web/src/components/PageTransition` 和 `web/src/components/Motion/PageTransition` 的冗余，统一使用 `web/src/components/Motion/PageTransition.tsx` 作为全局唯一组件。
*   **动画路径**：
    *   **Initial**: `opacity: 0, scale: 0.96, y: 10`
    *   **Animate**: `opacity: 1, scale: 1, y: 0`
    *   **Exit**: `opacity: 0, scale: 1.02` (略微放大淡出，营造深度感)
*   **优化**：在 `ProtectedApp.tsx` 中确保只包裹一层 `PageTransition`，避免嵌套动画导致的性能问题和视觉混乱。

### 3.2 基础组件反馈 (Micro-interactions)
*   **AnimatedButton**：
    *   `whileHover`: `scale: 1.02`
    *   `whileTap`: `scale: 0.94`
    *   增加对 `type="primary"` 等不同类型的视觉增强支持。
*   **AnimatedCard**：
    *   `whileHover`: `y: -4, scale: 1.01, boxShadow: "0 10px 20px rgba(0,0,0,0.1)"`
    *   `whileTap`: `scale: 0.98`
*   **StaggerList**：
    *   确保列表子项在挂载时自动应用 `variants`，实现从下往上的错开平滑滑入。

### 3.3 导航增强
*   **侧边栏 (Sidebar)**：菜单项图标在收起/展开时应用 `whileHover` 缩放效果。
*   **当前项标注**：使用 `layoutId` 实现激活项背景块在不同菜单间的平滑滑动。

## 4. 实施文件列表
*   `web/src/components/Motion/PageTransition.tsx` (更新核心逻辑)
*   `web/src/components/AnimatedButton/AnimatedButton.tsx` (更新参数)
*   `web/src/components/AnimatedCard/AnimatedCard.tsx` (更新参数)
*   `web/src/app/layout/AppLayout.tsx` (优化侧边栏与布局动画)
*   `web/src/ProtectedApp.tsx` (清理冗余包裹)

## 5. 验收标准
*   页面切换无瞬间闪烁或生硬跳变。
*   所有可点击元素在操作时均有明显的物理位移或缩放反馈。
*   列表加载时呈现自然、有序的出现节奏。
