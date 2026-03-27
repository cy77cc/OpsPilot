import React from 'react';
import './Kbd.css';

interface KbdProps {
  children: React.ReactNode;
  className?: string;
}

/**
 * Kbd 组件 - 用于显示键盘快捷键
 *
 * 使用示例:
 * <Kbd>⌘K</Kbd>
 * <Kbd>Ctrl+K</Kbd>
 * <Kbd>g h</Kbd>
 */
const Kbd: React.FC<KbdProps> = ({ children, className = '' }) => {
  return <kbd className={`kbd ${className}`}>{children}</kbd>;
};

export default Kbd;
