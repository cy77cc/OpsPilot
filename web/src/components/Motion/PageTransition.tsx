import React from 'react';
import { useLocation } from 'react-router-dom';
import { AnimatePresence, motion } from 'framer-motion';

/**
 * PageTransition Props
 */
export interface PageTransitionProps {
  children: React.ReactNode;
}

/**
 * Page Transition Animation Variants
 *
 * 使用缩放淡入效果（scale + opacity）
 * 退出: opacity 1 → 0, scale 1 → 0.98
 * 进入: opacity 0 → 1, scale 0.98 → 1
 */
const pageVariants = {
  initial: {
    opacity: 0,
    scale: 0.98,
  },
  enter: {
    opacity: 1,
    scale: 1,
    transition: {
      duration: 0.2,
      ease: [0.16, 1, 0.3, 1], // --motion-ease-out
    },
  },
  exit: {
    opacity: 0,
    scale: 0.98,
    transition: {
      duration: 0.2,
      ease: [0.16, 1, 0.3, 1], // --motion-ease-out
    },
  },
};

/**
 * PageTransition Component
 *
 * 包裹页面内容，提供路由切换时的过渡动画。
 * 使用 location.pathname 作为 key 确保每次路由变化触发动画。
 *
 * @example
 * ```tsx
 * <PageTransition>
 *   <Routes>...</Routes>
 * </PageTransition>
 * ```
 */
export const PageTransition: React.FC<PageTransitionProps> = ({ children }) => {
  const location = useLocation();

  React.useEffect(() => {
    // 滚动重置到顶部
    window.scrollTo(0, 0);
  }, [location.pathname]);

  return (
    <AnimatePresence mode="wait">
      <motion.div
        key={location.pathname}
        initial="initial"
        animate="enter"
        exit="exit"
        variants={pageVariants}
        style={{ width: '100%', height: '100%' }}
      >
        {children}
      </motion.div>
    </AnimatePresence>
  );
};

export default PageTransition;
