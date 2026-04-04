import React from 'react';
import { Skeleton } from 'antd';

export interface TableSkeletonProps {
  toolbar?: boolean;
  rows?: number;
  columns?: number;
}

const TableSkeleton: React.FC<TableSkeletonProps> = ({
  toolbar = true,
  rows = 8,
  columns = 6,
}) => {
  const safeRows = Math.max(0, rows);
  const safeColumns = Math.max(1, columns);
  const gridTemplateColumns = `repeat(${safeColumns}, minmax(0, 1fr))`;

  return (
    <section data-testid="table-skeleton" aria-busy="true" className="loading-skeleton-grid">
      {toolbar ? (
        <div className="loading-skeleton-surface loading-skeleton-toolbar">
          <Skeleton.Input active block className="!h-10 !w-full" />
        </div>
      ) : null}
      <div className="loading-skeleton-surface loading-skeleton-table">
        <div className="loading-skeleton-table-row-grid" style={{ gridTemplateColumns }}>
          {Array.from({ length: safeColumns }).map((_, index) => (
            <Skeleton.Input
              key={`header-${index}`}
              active
              block
              className="!h-4 !w-full"
            />
          ))}
        </div>
        <div className="loading-skeleton-grid">
          {Array.from({ length: safeRows }).map((_, rowIndex) => (
            <div key={rowIndex} data-testid="table-skeleton-row" className="loading-skeleton-table-row">
              <div className="loading-skeleton-table-row-grid" style={{ gridTemplateColumns }}>
                {Array.from({ length: safeColumns }).map((_, columnIndex) => (
                  <Skeleton.Input
                    key={`row-${rowIndex}-col-${columnIndex}`}
                    active
                    block
                    className={columnIndex === 0 ? '!h-4 !w-4/5' : '!h-4 !w-full'}
                  />
                ))}
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
};

export default TableSkeleton;
