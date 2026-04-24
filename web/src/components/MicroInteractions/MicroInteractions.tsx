import React from 'react';
import './MicroInteractions.css';

/**
 * 按钮点击反馈组件
 */
export const ButtonFeedback: React.FC<{ children: React.ReactNode; onClick?: () => void }> = ({
  children,
  onClick,
}) => {
  return (
    <div className="button-feedback-container" onClick={onClick}>
      {children}
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
    <div className={`form-validation-feedback form-validation-${type}`}>
      <span className="form-validation-icon">
        {icons[type]}
      </span>
      <span className="form-validation-message">{message}</span>
    </div>
  );
};

/**
 * 成功提示
 */
export const SuccessCheckmark: React.FC = () => {
  return (
    <div className="success-checkmark">
      <svg viewBox="0 0 52 52" className="success-checkmark-svg">
        <circle
          className="success-checkmark-circle"
          cx="26"
          cy="26"
          r="25"
          fill="none"
          strokeDasharray="157"
          strokeDashoffset="0"
        />
        <path
          className="success-checkmark-check"
          fill="none"
          d="M14.1 27.2l7.1 7.2 16.7-16.8"
          strokeDasharray="40"
          strokeDashoffset="0"
        />
      </svg>
    </div>
  );
};

/**
 * 加载点 (静态)
 */
export const LoadingDots: React.FC = () => {
  return (
    <div className="loading-dots">
      {[0, 1, 2].map((index) => (
        <span
          key={index}
          className="loading-dot"
          style={{ opacity: 1 }}
        />
      ))}
    </div>
  );
};

/**
 * 脉冲 (静态)
 */
export const PulseAnimation: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  return (
    <div>
      {children}
    </div>
  );
};
