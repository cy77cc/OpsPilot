import React from 'react';
import type { ThoughtStageItem } from '../types';
import { ThoughtChainDetailItem } from './ThoughtChainDetailItem';

export function ThoughtChainStageCard({ stage }: { stage: ThoughtStageItem }) {
  return (
    <div>
      {stage.content ? (
        <div style={{ marginBottom: stage.details?.length ? 8 : 0, whiteSpace: 'pre-wrap' }}>
          {stage.content}
        </div>
      ) : null}
      {(stage.details || []).map((detail) => (
        <ThoughtChainDetailItem key={detail.id} detail={detail} />
      ))}
    </div>
  );
}
