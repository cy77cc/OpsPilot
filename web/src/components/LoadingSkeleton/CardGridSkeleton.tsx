import React from 'react';
import { Skeleton } from 'antd';

export interface CardGridSkeletonProps {
  cards?: number;
  columns?: number;
}

const CardGridSkeleton: React.FC<CardGridSkeletonProps> = ({ cards = 6, columns = 3 }) => {
  const safeCards = Math.max(0, cards);
  const safeColumns = Math.max(1, columns);
  const gridTemplateColumns = `repeat(${safeColumns}, minmax(0, 1fr))`;

  return (
    <section data-testid="card-grid-skeleton" aria-busy="true" className="loading-skeleton-grid">
      <div className="loading-skeleton-grid" style={{ gridTemplateColumns }}>
        {Array.from({ length: safeCards }).map((_, index) => (
          <article
            key={index}
            data-testid="card-grid-skeleton-card"
            className="loading-skeleton-surface loading-skeleton-card"
          >
            <Skeleton.Avatar active size="large" shape="square" />
            <Skeleton active paragraph={{ rows: 3 }} />
          </article>
        ))}
      </div>
    </section>
  );
};

export default CardGridSkeleton;
