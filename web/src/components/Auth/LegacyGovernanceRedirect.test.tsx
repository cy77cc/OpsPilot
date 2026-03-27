import { render, waitFor } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import LegacyGovernanceRedirect from './LegacyGovernanceRedirect';

const mockApi = vi.hoisted(() => ({
  rbac: {
    recordMigrationEvent: vi.fn().mockResolvedValue({ data: { accepted: true } }),
  },
}));

vi.mock('../../api', () => ({ Api: mockApi }));

describe('LegacyGovernanceRedirect', () => {
  it('records legacy redirect event before navigating', async () => {
    render(
      <MemoryRouter initialEntries={['/settings/users']}>
        <Routes>
          <Route path="/settings/users" element={<LegacyGovernanceRedirect to="/governance/users" />} />
          <Route path="/governance/users" element={<div>governance users page</div>} />
        </Routes>
      </MemoryRouter>,
    );

    await waitFor(() => {
      expect(mockApi.rbac.recordMigrationEvent).toHaveBeenCalledWith(
        expect.objectContaining({
          eventType: 'legacy_redirect',
          fromPath: '/settings/users',
          toPath: '/governance/users',
          status: 'redirected',
        }),
      );
    });
  });
});
