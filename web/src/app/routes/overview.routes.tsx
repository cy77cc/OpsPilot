import React from 'react';
import { Route } from 'react-router-dom';
import type { WithAuth } from './routeGuards';
import { Dashboard } from './pages';

export function renderOverviewRoutes(withAuth: WithAuth) {
  return (
    <>
      <Route path="/" element={<Dashboard />} />
    </>
  );
}
