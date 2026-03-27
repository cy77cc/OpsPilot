import React from 'react';
import { motion } from 'framer-motion';

/**
 * StaggerList Props
 */
export interface StaggerListProps {
  children: React.ReactNode;
  className?: string;
  staggerDelay?: number;
}

/**
 * StaggerItem Props
 */
export interface StaggerItemProps {
  children: React.ReactNode;
  className?: string;
}

/**
 * Stagger Animation Variants
 */
const containerVariants = {
  hidden: { opacity: 1 },
  visible: {
    opacity: 1,
    transition: {
      staggerChildren: 0.05, // 50ms stagger
    },
  },
};

const itemVariants = {
  hidden: {
    opacity: 0,
    y: 8,
  },
  visible: {
    opacity: 1,
    y: 0,
    transition: {
      duration: 0.2, // 200ms
      ease: [0.16, 1, 0.3, 1], // --motion-ease-out
    },
  },
};

/**
 * StaggerList Component
 *
 * 包裹子元素实现交错进入动画。
 * 子元素依次从 opacity 0, y: 8px 进入到 opacity 1, y: 0。
 *
 * @example
 * ```tsx
 * <StaggerList>
 *   <StaggerItem><Card>...</Card></StaggerItem>
 *   <StaggerItem><Card>...</Card></StaggerItem>
 * </StaggerList>
 * ```
 */
export const StaggerList: React.FC<StaggerListProps> = ({
  children,
  className,
  staggerDelay = 0.05,
}) => {
  const customContainerVariants = {
    ...containerVariants,
    visible: {
      ...containerVariants.visible,
      transition: {
        staggerChildren: staggerDelay,
      },
    },
  };

  return (
    <motion.div
      className={className}
      initial="hidden"
      animate="visible"
      variants={customContainerVariants}
    >
      {children}
    </motion.div>
  );
};

/**
 * StaggerItem Component
 *
 * 配合 StaggerList 使用，为每个子项提供动画。
 */
export const StaggerItem: React.FC<StaggerItemProps> = ({ children, className }) => {
  return (
    <motion.div className={className} variants={itemVariants}>
      {children}
    </motion.div>
  );
};

export default StaggerList;
