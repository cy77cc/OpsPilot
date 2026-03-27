import { motion, AnimatePresence } from 'framer-motion';
import React, { useState } from 'react';

interface CollapsibleProps {
  children: React.ReactNode;
  trigger: React.ReactNode;
  defaultOpen?: boolean;
  className?: string;
}

/**
 * 可折叠组件
 *
 * 特性:
 * - 平滑的展开/折叠动画
 * - 高度自适应
 * - 旋转箭头指示器
 */
const Collapsible: React.FC<CollapsibleProps> = ({
  children,
  trigger,
  defaultOpen = false,
  className = '',
}) => {
  const [isOpen, setIsOpen] = useState(defaultOpen);

  return (
    <div className={className}>
      <motion.div
        onClick={() => setIsOpen(!isOpen)}
        className="cursor-pointer"
        whileHover={{ backgroundColor: 'rgba(0, 0, 0, 0.02)' }}
        transition={{ duration: 0.2 }}
      >
        {trigger}
      </motion.div>

      <AnimatePresence initial={false}>
        {isOpen && (
          <motion.div
            initial={{ height: 0, opacity: 0 }}
            animate={{
              height: 'auto',
              opacity: 1,
              transition: {
                height: {
                  duration: 0.3,
                  ease: [0.4, 0, 0.2, 1],
                },
                opacity: {
                  duration: 0.25,
                  delay: 0.05,
                },
              },
            }}
            exit={{
              height: 0,
              opacity: 0,
              transition: {
                height: {
                  duration: 0.25,
                  ease: [0.4, 0, 1, 1],
                },
                opacity: {
                  duration: 0.2,
                },
              },
            }}
            style={{ overflow: 'hidden' }}
          >
            {children}
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
};

export default Collapsible;
