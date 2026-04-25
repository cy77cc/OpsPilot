import React from 'react';
import { Route } from 'react-router-dom';
import type { WithAuth } from './routeGuards';
import { SettingsPage } from './pages';

interface AIRoutesProps {
  withAuth: WithAuth;
}

export function renderAIRoutes({ withAuth }: AIRoutesProps) {
  return (
    <>
      <Route path="/ai/settings/models" element={<SettingsPage defaultTab="ai" />} />
      <Route path="/ai/chat" element={<SettingsPage defaultTab="ai" />} />
      <Route path="/ai/models" element={<SettingsPage defaultTab="ai" />} />
      <Route path="/ai/usage" element={<SettingsPage defaultTab="ai" />} />
    </>
  );
}
