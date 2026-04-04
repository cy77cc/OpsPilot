import React from 'react';
import './LoadingSkeleton.css';

export interface LoadingSkeletonSectionProps {
  testId: string;
  className?: string;
  children: React.ReactNode;
}

const LoadingSkeletonSection: React.FC<LoadingSkeletonSectionProps> = ({
  testId,
  className,
  children,
}) => (
  <section data-testid={testId} aria-busy="true" className={className}>
    {children}
  </section>
);

export default LoadingSkeletonSection;
