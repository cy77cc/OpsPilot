import React from 'react';
import type { FieldGuide } from './types';

type FieldGuideRowProps = {
  label: string;
  value: string;
};

const FieldGuideRow: React.FC<FieldGuideRowProps> = ({ label, value }) => (
  <div className="space-y-1">
    <div className="text-xs font-semibold tracking-wide text-emerald-700">{label}</div>
    <div className="text-sm leading-6 text-slate-700">{value}</div>
  </div>
);

type FieldGuideCardProps = {
  guide: FieldGuide;
};

const FieldGuideCard: React.FC<FieldGuideCardProps> = ({ guide }) => {
  const rows = [
    { label: '这里填什么', value: guide.whatToEnter },
    { label: '这个值是干嘛的', value: guide.purpose },
    { label: '推荐示例', value: guide.example },
    { label: '填错会怎样', value: guide.impact },
    { label: '什么时候必填', value: guide.whenRequired },
    { label: '格式要求', value: guide.formatNotes },
  ].filter((row): row is { label: string; value: string } => Boolean(row.value && row.value.trim()));

  return (
    <div
      data-testid="field-guide-card"
      className="rounded-xl border border-emerald-200 bg-emerald-50/80 p-3 shadow-sm"
    >
      <div className="space-y-3">
        {rows.map((row) => (
          <FieldGuideRow key={row.label} label={row.label} value={row.value} />
        ))}
      </div>
    </div>
  );
};

export default FieldGuideCard;
