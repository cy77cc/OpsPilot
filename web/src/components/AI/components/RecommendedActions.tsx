import React from 'react';
import { RecommendationCard } from './RecommendationCard';
import type { EmbeddedRecommendation } from '../types';

export function RecommendedActions({
  recommendations,
  onSelect,
}: {
  recommendations: EmbeddedRecommendation[];
  onSelect: (prompt: string) => void;
}) {
  return <RecommendationCard recommendations={recommendations} onSelect={onSelect} />;
}
