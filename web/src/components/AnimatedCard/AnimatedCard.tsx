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
