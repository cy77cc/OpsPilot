import { motion } from 'framer-motion';
import React from 'react';
import './LoadingSpinner.css';

interface LoadingSpinnerProps {
  size?: 'small' | 'medium' | 'large';
  text?: string;
  fullScreen?: boolean;
}

/**
 * 加载动画组件
 *
 * 特性:
 * - 平滑的旋转动画
 * - 多种尺寸
 * - 可选文字提示
 * - 全屏模式
 */
const LoadingSpinner: React.FC<LoadingSpinnerProps> = ({
  size = 'medium',
  text,
  fullScreen = false,
}) => {
  const sizeMap = {
    small: 24,
    medium: 40,
    large: 64,
  };

  const spinnerSize = sizeMap[size];

  const spinner = (
    <div className={`loading-spinner-container ${fullScreen ? 'loading-spinner-fullscreen' : ''}`}>
      <motion.div
        className="loading-spinner"
        style={{
          width: spinnerSize,
          height: spinnerSize,
        }}
        animate={{ rotate: 360 }}
        transition={{
          duration: 1,
          repeat: Infinity,
          ease: 'linear',
        }}
      >
        <svg viewBox="0 0 50 50" className="loading-spinner-svg">
          <circle
            className="loading-spinner-circle"
            cx="25"
            cy="25"
            r="20"
            fill="none"
            strokeWidth="4"
          />
        </svg>
      </motion.div>
      {text && (
        <motion.p
          className="loading-spinner-text"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ delay: 0.2 }}
        >
          {text}
        </motion.p>
      )}
    </div>
  );

  return spinner;
};

export default LoadingSpinner;
