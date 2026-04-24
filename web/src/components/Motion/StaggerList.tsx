import React from 'react';

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
 * StaggerList Component
 */
export const StaggerList: React.FC<StaggerListProps> = ({
  children,
  className,
}) => {
  return (
    <div className={className}>
      {children}
    </div>
  );
};

/**
 * StaggerItem Component
 */
export const StaggerItem: React.FC<StaggerItemProps> = ({ children, className }) => {
  return (
    <div className={className}>
      {children}
    </div>
  );
};

export default StaggerList;
