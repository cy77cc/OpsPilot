import React from 'react';
import { Skeleton } from 'antd';

export interface FormSkeletonProps {
  title?: boolean;
  groups?: number;
  actions?: boolean;
}

const FormSkeleton: React.FC<FormSkeletonProps> = ({
  title = true,
  groups = 3,
  actions = true,
}) => {
  const safeGroups = Math.max(0, groups);

  return (
    <section data-testid="form-skeleton" aria-busy="true" className="loading-skeleton-form">
      {title ? (
        <div className="loading-skeleton-surface">
          <Skeleton.Input active block className="!h-9 !w-64" />
          <div className="mt-4">
            <Skeleton active paragraph={{ rows: 1 }} />
          </div>
        </div>
      ) : null}

      {Array.from({ length: safeGroups }).map((_, index) => (
        <div key={index} data-testid="form-skeleton-group" className="loading-skeleton-surface loading-skeleton-form-group">
          <Skeleton.Input active block className="!h-5 !w-40" />
          <Skeleton.Input active block className="!h-10 !w-full" />
          <Skeleton.Input active block className="!h-10 !w-full" />
        </div>
      ))}

      {actions ? (
        <div data-testid="form-skeleton-actions" className="loading-skeleton-form-actions">
          <Skeleton.Button active block className="!w-28" />
          <Skeleton.Button active block className="!w-32" />
        </div>
      ) : null}
    </section>
  );
};

export default FormSkeleton;
