import React from 'react';
import { Route } from 'react-router-dom';
import type { WithAuth } from './routeGuards';
import {
  Dashboard,
  TasksPage,
  JobListPage,
  JobCreationPage,
  ExecutionHistoryPage,
  JobCalendarPage,
} from './pages';

export function renderWorkloadRoutes(withAuth: WithAuth) {
  return (
  <>
    <Route path="/" element={<Dashboard />} />
    <Route path="/tasks" element={withAuth('task', 'read', <TasksPage />)} />
    <Route path="/tasks/create" element={withAuth('task', 'write', <TasksPage />)} />
    <Route path="/tasks/:id" element={withAuth('task', 'read', <TasksPage />)} />
    <Route path="/jobs" element={withAuth('task', 'read', <JobListPage />)} />
    <Route path="/jobs/create" element={withAuth('task', 'write', <JobCreationPage />)} />
    <Route path="/jobs/:id/edit" element={withAuth('task', 'write', <JobCreationPage />)} />
    <Route path="/jobs/:jobId/history" element={withAuth('task', 'read', <ExecutionHistoryPage />)} />
    <Route path="/jobs/calendar" element={withAuth('task', 'read', <JobCalendarPage />)} />
  </>
  );
}
