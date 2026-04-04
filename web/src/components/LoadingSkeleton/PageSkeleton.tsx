import React from 'react';
import { Skeleton } from 'antd';

const PageSkeleton: React.FC = () => (
  <section data-testid="page-skeleton" aria-busy="true" className="space-y-6">
    <div className="space-y-3">
      <Skeleton.Input active block className="!h-10 !w-60" />
      <Skeleton.Input active block className="!h-6 !w-80" />
    </div>
    <div className="grid grid-cols-1 gap-4 xl:grid-cols-3">
      <Skeleton active paragraph={{ rows: 4 }} />
      <Skeleton active paragraph={{ rows: 4 }} />
      <Skeleton active paragraph={{ rows: 4 }} />
    </div>
    <div className="rounded-2xl border border-gray-200 bg-white p-5">
      <Skeleton active paragraph={{ rows: 10 }} />
    </div>
  </section>
);

export default PageSkeleton;
