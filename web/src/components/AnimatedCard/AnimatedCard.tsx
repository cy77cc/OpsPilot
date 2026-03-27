import { motion } from 'framer-motion';
import React from 'react';

interface AnimatedCardProps {
  children: React.ReactNode;
  className?: string;
  delay?: number;
  onClick?: () => void;
}

/**
 * 带动画效果的卡片组件
 *
 * 特性:
 * - Hover 时轻微上浮和阴影增强
 * - 点击时缩放反馈
 * - 入场动画
 */
const AnimatedCard: React.FC<AnimatedCardProps> = ({ children, className = '', delay = 0, onClick }) => {
  return (
    <motion.div
      className={className}
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{
        duration: 0.3,
        delay,
        ease: [0.4, 0, 0.2, 1],
      }}
      whileHover={{
        y: -4,
        boxShadow: '0 12px 24px -4px rgba(0, 0, 0, 0.12), 0 8px 16px -4px rgba(0, 0, 0, 0.08)',
        transition: { duration: 0.2 },
      }}
      whileTap={onClick ? { scale: 0.98 } : undefined}
      onClick={onClick}
    >
      {children}
    </motion.div>
  );
};

export default AnimatedCard;
