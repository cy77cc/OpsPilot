import React from 'react';
import type { ThoughtStageDetailItem as ThoughtStageDetailItemData } from '../types';
import { ToolCard } from './ToolCard';

export function ThoughtChainDetailItem({ detail }: { detail: ThoughtStageDetailItemData }) {
  if (detail.kind === 'tool' || detail.tool || detail.params || detail.result) {
    return (
      <ToolCard
        tool={{
          id: detail.id,
          name: detail.tool || detail.label,
          status: detail.status === 'error' ? 'error' : detail.status === 'success' ? 'success' : 'running',
          summary: detail.content,
          params: detail.params,
          result: detail.result
            ? {
                ok: detail.result.ok !== false,
                data: detail.result.data,
                error: detail.result.error,
                latency_ms: detail.result.latency_ms,
              }
            : undefined,
          error: detail.result?.error,
        }}
      />
    );
  }

  return (
    <div style={{ marginTop: 8, fontSize: 12, lineHeight: 1.6 }}>
      <div style={{ fontWeight: 500 }}>{detail.label}</div>
      {detail.content ? <div>{detail.content}</div> : null}
    </div>
  );
}
