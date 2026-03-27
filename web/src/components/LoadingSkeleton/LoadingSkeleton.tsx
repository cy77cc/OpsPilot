import React from 'react';
import { Skeleton } from 'antd';
import './LoadingSkeleton.css';

interface LoadingSkeletonProps {
  type?: 'card' | 'list' | 'table' | 'detail';
  count?: number;
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
const LoadingSkeleton: React.FC<LoadingSkeletonProps> = ({ type = 'card', count = 3 }) => {
  if (type === 'card') {
    return (
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
        {Array.from({ length: count }).map((_, index) => (
          <div key={index} className="loading-skeleton-card">
            <Skeleton.Avatar active size="large" shape="square" className="mb-4" />
            <Skeleton active paragraph={{ rows: 3 }} />
          </div>
        ))}
      </div>
    );
  }

  if (type === 'list') {
    return (
      <div className="space-y-4">
        {Array.from({ length: count }).map((_, index) => (
          <div key={index} className="loading-skeleton-list-item">
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
      <div className="loading-skeleton-table">
        <Skeleton.Input active size="large" block className="mb-4" />
        {Array.from({ length: count }).map((_, index) => (
          <div key={index} className="loading-skeleton-table-row">
            <Skeleton active paragraph={{ rows: 1 }} />
          </div>
        ))}
      </div>
    );
  }

  if (type === 'detail') {
    return (
      <div className="space-y-6">
        <div className="loading-skeleton-detail-header">
          <Skeleton.Avatar active size={64} />
          <div className="flex-1">
            <Skeleton active paragraph={{ rows: 2 }} />
          </div>
        </div>
        <div className="loading-skeleton-detail-content">
          <Skeleton active paragraph={{ rows: 8 }} />
        </div>
      </div>
    );
  }

  return <Skeleton active />;
};

export default LoadingSkeleton;
