import React from 'react';
import { Skeleton } from 'antd';
import './LoadingSkeleton.css';

export type LoadingSkeletonType = 'card' | 'list' | 'table' | 'detail';

export interface LoadingSkeletonProps {
  type?: LoadingSkeletonType;
  count?: number;
  'data-testid'?: string;
}

/**
 * 加载骨架屏组件
 *
 * 提供多种预设布局:
 * - card: 卡片布局骨架屏
 * - list: 列表布局骨架屏
 * - table: 表格布局骨架屏
 * - detail: 详情页布局骨架屏
 */
const LoadingSkeleton: React.FC<LoadingSkeletonProps> = ({
  type = 'card',
  count = 3,
  'data-testid': dataTestId,
}) => {
  const safeCount = Math.max(0, count);

  if (type === 'card') {
    return (
      <div data-testid={dataTestId} className="loading-skeleton-grid loading-skeleton-card-grid">
        {Array.from({ length: safeCount }).map((_, index) => (
          <div key={index} className="loading-skeleton-card loading-skeleton-surface">
            <Skeleton.Avatar active size="large" shape="square" className="mb-4" />
            <Skeleton active paragraph={{ rows: 3 }} />
          </div>
        ))}
      </div>
    );
  }

  if (type === 'list') {
    return (
      <div data-testid={dataTestId} className="loading-skeleton-grid">
        {Array.from({ length: safeCount }).map((_, index) => (
          <div key={index} className="loading-skeleton-list-item loading-skeleton-surface">
            <Skeleton.Avatar active size="large" />
            <div className="flex-1">
              <Skeleton active paragraph={{ rows: 2 }} />
            </div>
          </div>
        ))}
      </div>
    );
  }

  if (type === 'table') {
    return (
      <div data-testid={dataTestId} className="loading-skeleton-table loading-skeleton-surface">
        <Skeleton.Input active size="large" block className="mb-4" />
        {Array.from({ length: safeCount }).map((_, index) => (
          <div key={index} className="loading-skeleton-table-row">
            <Skeleton active paragraph={{ rows: 1 }} />
          </div>
        ))}
      </div>
    );
  }

  if (type === 'detail') {
    return (
      <div data-testid={dataTestId} className="loading-skeleton-detail">
        <div className="loading-skeleton-detail-header loading-skeleton-surface">
          <Skeleton.Avatar active size={64} />
          <div className="flex-1">
            <Skeleton active paragraph={{ rows: 2 }} />
          </div>
        </div>
        <div className="loading-skeleton-detail-content loading-skeleton-surface">
          <Skeleton active paragraph={{ rows: 8 }} />
        </div>
      </div>
    );
  }

  return <Skeleton active />;
};

export default LoadingSkeleton;
