import React from 'react';
import { ConfirmationPanel } from './ConfirmationPanel';
import type { ConfirmationRequest } from '../types';

export function ApprovalConfirmationPanel({ confirmation }: { confirmation: ConfirmationRequest }) {
  return <ConfirmationPanel confirmation={confirmation} />;
}
