import React from 'react';
import { Skeleton } from 'antd';
import LoadingSkeletonSection from './LoadingSkeletonSection';

export interface DetailSkeletonProps {
  summaryCards?: number;
  sections?: number;
}

const DetailSkeleton: React.FC<DetailSkeletonProps> = ({
  summaryCards = 3,
  sections = 3,
}) => {
  const safeSummaryCards = Math.max(0, summaryCards);
  const safeSections = Math.max(0, sections);

  return (
    <LoadingSkeletonSection testId="detail-skeleton" className="loading-skeleton-detail">
      <div className="loading-skeleton-surface">
        <Skeleton.Input active block className="!h-10 !w-72" />
        <div className="mt-4">
          <Skeleton active paragraph={{ rows: 2 }} />
        </div>
      </div>
      <div className="loading-skeleton-detail-summary">
        {Array.from({ length: safeSummaryCards }).map((_, index) => (
          <div key={index} data-testid="detail-skeleton-card" className="loading-skeleton-surface">
            <Skeleton active paragraph={{ rows: 2 }} />
          </div>
        ))}
      </div>
      <div className="loading-skeleton-detail-section">
        {Array.from({ length: safeSections }).map((_, index) => (
          <div key={index} className="loading-skeleton-surface">
            <Skeleton active paragraph={{ rows: 5 }} />
          </div>
        ))}
      </div>
    </LoadingSkeletonSection>
  );
};

export default DetailSkeleton;
