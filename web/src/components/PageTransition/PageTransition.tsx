import { motion, AnimatePresence } from 'framer-motion';
import { useLocation } from 'react-router-dom';
import React from 'react';

interface PageTransitionProps {
  children: React.ReactNode;
}

/**
 * 页面切换动画组件
 *
 * 使用方式:
 * <PageTransition>
 *   <YourPageComponent />
 * </PageTransition>
 */
const PageTransition: React.FC<PageTransitionProps> = ({ children }) => {
  const location = useLocation();

  const pageVariants = {
    initial: {
      opacity: 0,
      y: 20,
    },
    animate: {
      opacity: 1,
      y: 0,
      transition: {
        duration: 0.3,
        ease: [0.4, 0, 0.2, 1], // easeInOut
      },
    },
    exit: {
      opacity: 0,
      y: -20,
      transition: {
        duration: 0.2,
        ease: [0.4, 0, 1, 1], // easeIn
      },
    },
  };

  return (
    <AnimatePresence mode="wait">
      <motion.div
        key={location.pathname}
        initial="initial"
        animate="animate"
        exit="exit"
        variants={pageVariants}
      >
        {children}
      </motion.div>
    </AnimatePresence>
  );
};

export default PageTransition;
