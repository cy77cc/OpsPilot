import { motion } from 'framer-motion';
import React, { useState } from 'react';
import './MicroInteractions.css';

/**
 * 按钮点击反馈组件
 */
export const ButtonFeedback: React.FC<{ children: React.ReactNode; onClick?: () => void }> = ({
  children,
  onClick,
}) => {
  const [ripples, setRipples] = useState<Array<{ x: number; y: number; id: number }>>([]);

  const handleClick = (e: React.MouseEvent<HTMLDivElement>) => {
    const rect = e.currentTarget.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const y = e.clientY - rect.top;
    const id = Date.now();

    setRipples([...ripples, { x, y, id }]);

    setTimeout(() => {
      setRipples((prev) => prev.filter((ripple) => ripple.id !== id));
    }, 600);

    onClick?.();
  };

  return (
    <div className="button-feedback-container" onClick={handleClick}>
      {children}
      {ripples.map((ripple) => (
        <motion.span
          key={ripple.id}
          className="button-feedback-ripple"
          style={{
            left: ripple.x,
            top: ripple.y,
          }}
          initial={{ scale: 0, opacity: 0.5 }}
          animate={{ scale: 4, opacity: 0 }}
          transition={{ duration: 0.6, ease: 'easeOut' }}
        />
      ))}
    </div>
  );
};

/**
 * 表单验证反馈组件
 */
export const FormValidationFeedback: React.FC<{
  type: 'success' | 'error' | 'warning';
  message: string;
}> = ({ type, message }) => {
  const icons = {
    success: '✓',
    error: '✕',
    warning: '!',
  };

  return (
    <motion.div
      className={`form-validation-feedback form-validation-${type}`}
      initial={{ opacity: 0, x: -10 }}
      animate={{ opacity: 1, x: 0 }}
      exit={{ opacity: 0, x: 10 }}
      transition={{ duration: 0.2 }}
    >
      <motion.span
        className="form-validation-icon"
        initial={{ scale: 0 }}
        animate={{ scale: 1 }}
        transition={{ type: 'spring', stiffness: 500, damping: 15 }}
      >
        {icons[type]}
      </motion.span>
      <span className="form-validation-message">{message}</span>
    </motion.div>
  );
};

/**
 * 成功提示动画
 */
export const SuccessCheckmark: React.FC = () => {
  return (
    <motion.div
      className="success-checkmark"
      initial={{ scale: 0 }}
      animate={{ scale: 1 }}
      transition={{ type: 'spring', stiffness: 260, damping: 20 }}
    >
      <motion.svg viewBox="0 0 52 52" className="success-checkmark-svg">
        <motion.circle
          className="success-checkmark-circle"
          cx="26"
          cy="26"
          r="25"
          fill="none"
          initial={{ pathLength: 0 }}
          animate={{ pathLength: 1 }}
          transition={{ duration: 0.6, ease: 'easeInOut' }}
        />
        <motion.path
          className="success-checkmark-check"
          fill="none"
          d="M14.1 27.2l7.1 7.2 16.7-16.8"
          initial={{ pathLength: 0 }}
          animate={{ pathLength: 1 }}
          transition={{ duration: 0.4, delay: 0.2, ease: 'easeInOut' }}
        />
      </motion.svg>
    </motion.div>
  );
};

/**
 * 加载点动画
 */
export const LoadingDots: React.FC = () => {
  return (
    <div className="loading-dots">
      {[0, 1, 2].map((index) => (
        <motion.span
          key={index}
          className="loading-dot"
          animate={{
            y: [0, -10, 0],
            opacity: [0.5, 1, 0.5],
          }}
          transition={{
            duration: 0.6,
            repeat: Infinity,
            delay: index * 0.15,
            ease: 'easeInOut',
          }}
        />
      ))}
    </div>
  );
};

/**
 * 脉冲动画
 */
export const PulseAnimation: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  return (
    <motion.div
      animate={{
        scale: [1, 1.05, 1],
        opacity: [1, 0.8, 1],
      }}
      transition={{
        duration: 2,
        repeat: Infinity,
        ease: 'easeInOut',
      }}
    >
      {children}
    </motion.div>
  );
};
