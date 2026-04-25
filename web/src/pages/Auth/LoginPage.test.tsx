import React from 'react';
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import LoginPage from './LoginPage';

const loginMock = vi.hoisted(() => vi.fn());

vi.mock('../../components/Auth/AuthContext', () => ({
  useAuth: () => ({
    login: loginMock,
  }),
}));

describe('LoginPage error sanitization', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    sessionStorage.clear();
  });

  afterEach(() => {
    cleanup();
  });

  it('shows normalized error text instead of backend error details', async () => {
    const backendMessage = 'SQLSTATE[42000]: syntax error near users.password_hash';
    loginMock.mockRejectedValueOnce(new Error(backendMessage));

    render(
      <MemoryRouter initialEntries={['/login']}>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
        </Routes>
      </MemoryRouter>
    );

    fireEvent.change(screen.getByLabelText('用户名'), { target: { value: 'admin' } });
    fireEvent.change(screen.getByLabelText('密码'), { target: { value: 'bad-password' } });
    fireEvent.click(screen.getByRole('button', { name: /登\s*录/ }));

    await waitFor(() => {
      expect(screen.getByText('登录失败，请检查用户名或密码后重试')).toBeInTheDocument();
    });
    expect(screen.queryByText(backendMessage)).not.toBeInTheDocument();
  });

  it('prefers redirectAfterLogin from sessionStorage after successful login', async () => {
    loginMock.mockResolvedValueOnce(undefined);
    sessionStorage.setItem('redirectAfterLogin', '/observability/alerts?tab=open');

    render(
      <MemoryRouter initialEntries={[{ pathname: '/login', state: { from: '/legacy-fallback' } }]}>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/observability/alerts" element={<div>alerts-page</div>} />
          <Route path="/legacy-fallback" element={<div>legacy-fallback-page</div>} />
        </Routes>
      </MemoryRouter>
    );

    fireEvent.change(screen.getByLabelText('用户名'), { target: { value: 'admin' } });
    fireEvent.change(screen.getByLabelText('密码'), { target: { value: 'secret' } });
    fireEvent.click(screen.getByRole('button', { name: /登\s*录/ }));

    await waitFor(() => {
      expect(screen.getByText('alerts-page')).toBeInTheDocument();
    });

    expect(screen.queryByText('legacy-fallback-page')).not.toBeInTheDocument();
    expect(sessionStorage.getItem('redirectAfterLogin')).toBeNull();
  });
});
