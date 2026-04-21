import React from 'react';

interface SparklesIconProps extends React.HTMLAttributes<HTMLSpanElement> {
  active?: boolean;
}

const SparklesIcon: React.FC<SparklesIconProps> = ({ className, active = true, ...props }) => (
  <span 
    role="img" 
    aria-label="sparkles" 
    className={className} 
    style={{ 
      fontSize: '16px',
      filter: active ? 'none' : 'grayscale(1)',
      opacity: active ? 1 : 0.5,
      ...props.style 
    }}
    {...props}
  >
    ✨
  </span>
);

export default SparklesIcon;
