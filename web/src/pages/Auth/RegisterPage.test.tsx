import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import RegisterPage from './RegisterPage';

const registerMock = vi.hoisted(() => vi.fn());

vi.mock('../../components/Auth/AuthContext', () => ({
  useAuth: () => ({
    register: registerMock,
  }),
}));

describe('RegisterPage error sanitization', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('shows normalized error text instead of backend error details', async () => {
    const backendMessage = 'duplicate key value violates unique constraint users_email_key';
    registerMock.mockRejectedValueOnce(new Error(backendMessage));

    render(
      <MemoryRouter initialEntries={['/register']}>
        <Routes>
          <Route path="/register" element={<RegisterPage />} />
        </Routes>
      </MemoryRouter>
    );

    fireEvent.change(screen.getByLabelText('用户名'), { target: { value: 'ops_user' } });
    fireEvent.change(screen.getByLabelText('邮箱'), { target: { value: 'ops@example.com' } });
    fireEvent.change(screen.getByLabelText('密码'), { target: { value: 'password123' } });
    fireEvent.click(screen.getByRole('button', { name: '注册并登录' }));

    await waitFor(() => {
      expect(screen.getByText('注册失败，请稍后重试')).toBeInTheDocument();
    });
    expect(screen.queryByText(backendMessage)).not.toBeInTheDocument();
  });
});
