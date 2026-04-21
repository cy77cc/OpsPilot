import React from 'react';

interface SparklesIconProps extends React.SVGProps<SVGSVGElement> {
  active?: boolean;
}

const SparklesIcon: React.FC<SparklesIconProps> = ({ className, active = true, style, ...props }) => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    width="16"
    height="16"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth="2"
    strokeLinecap="round"
    strokeLinejoin="round"
    role="img"
    aria-label="AI 辅助图标"
    focusable="false"
    className={className}
    style={{
      animation: 'pulse 2s infinite',
      filter: active ? 'none' : 'grayscale(1)',
      opacity: active ? 1 : 0.5,
      ...style,
    }}
    {...props}
  >
    <path d="m12 3-1.912 5.813a2 2 0 0 1-1.275 1.275L3 12l5.813 1.912a2 2 0 0 1 1.275 1.275L12 21l1.912-5.813a2 2 0 0 1 1.275-1.275L21 12l-5.813-1.912a2 2 0 0 1-1.275-1.275L12 3Z" />
    <path d="M5 3v4" />
    <path d="M19 17v4" />
    <path d="M3 5h4" />
    <path d="M17 19h4" />
  </svg>
);

export default SparklesIcon;
