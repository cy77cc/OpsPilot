import React from 'react';
import { Skeleton } from 'antd';
import {
  CheckCircleFilled,
  ClockCircleFilled,
  KeyOutlined,
  RedoOutlined,
  WarningFilled,
} from '@ant-design/icons';
import type { CredentialStats } from '../../../../api/modules/hosts';
import { buildStatsCards } from '../viewModels';

interface Props {
  stats?: CredentialStats;
  loading?: boolean;
}

const iconMap = {
  key: KeyOutlined,
  safe: CheckCircleFilled,
  warning: ClockCircleFilled,
  danger: WarningFilled,
  recent: RedoOutlined,
} as const;

const Sparkline: React.FC<{ values: number[]; stroke: string }> = ({ values, stroke }) => {
  const max = Math.max(...values, 1);
  const width = 76;
  const height = 28;
  const points = values
    .map((value, index) => {
      const x = (index / Math.max(values.length - 1, 1)) * width;
      const y = height - (value / max) * (height - 4) - 2;
      return `${x},${y}`;
    })
    .join(' ');

  return (
    <svg width={width} height={height} viewBox={`0 0 ${width} ${height}`} aria-hidden="true">
      <polyline
        points={points}
        fill="none"
        stroke={stroke}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
};

export const CredentialStatsCards: React.FC<Props> = ({ stats, loading }) => {
  const cards = buildStatsCards(stats);

  return (
    <div className="grid grid-cols-1 gap-4 xl:grid-cols-5 md:grid-cols-2">
      {cards.map((card) => {
        const Icon = iconMap[card.icon];
        return (
          <div
            key={card.key}
            className="rounded-2xl border border-[#e8edf5] bg-white px-5 py-4 shadow-[0_8px_24px_rgba(15,23,42,0.04)]"
          >
            {loading ? (
              <Skeleton active paragraph={{ rows: 2 }} title={false} />
            ) : (
              <div className="flex items-center gap-4">
                <div
                  className="flex h-12 w-12 items-center justify-center rounded-full text-[22px]"
                  style={{ backgroundColor: card.soft, color: card.accent }}
                >
                  <Icon />
                </div>
                <div className="min-w-0 flex-1">
                  <div className="text-[14px] font-medium text-[#4b5565]">{card.title}</div>
                  <div className="mt-1 text-[18px] font-semibold leading-none text-[#111827] xl:text-[38px] xl:leading-[1.1]">
                    {card.value}
                  </div>
                  <div className="mt-2 text-[12px] text-[#6b7280]">{card.helper}</div>
                </div>
                <div className="hidden self-end xl:block">
                  <Sparkline values={card.sparkline} stroke={card.accent} />
                </div>
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
};
