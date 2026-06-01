import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import Login from '../index';

const mockNavigate = jest.fn();
const mockSetAuth = jest.fn();

jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useNavigate: () => mockNavigate,
}));

jest.mock('../../../store/authContext', () => ({
  useAuth: () => ({
    setAuth: mockSetAuth,
  }),
}));

describe('Login migration', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    localStorage.clear();
  });

  it('logs in with phone and redirects through H5 auth context', async () => {
    const loginSuccessListener = jest.fn();
    window.addEventListener('login-success', loginSuccessListener);

    const fetchMock = jest.fn().mockResolvedValue({
      ok: true,
      headers: { get: jest.fn(() => 'application/json') },
      json: async () => ({
        data: {
          token: 'token-1',
          user: { id: 7, email: 'buyer@example.com', name: '林见山', role: 0 },
        },
      }),
    } as Response);
    global.fetch = fetchMock;

    render(
      <MemoryRouter
        initialEntries={['/login?redirect=%2Fdetail%3Fid%3D42']}
        future={{ v7_relativeSplatPath: true, v7_startTransition: true }}
      >
        <Login />
      </MemoryRouter>
    );

    fireEvent.change(screen.getByLabelText('手机号'), { target: { value: '13800138000' } });
    fireEvent.change(screen.getByLabelText('密码'), { target: { value: 'secret123' } });
    fireEvent.click(screen.getByRole('button', { name: '登录' }));

    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/auth/login',
      expect.objectContaining({
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ phone: '13800138000', password: 'secret123' }),
      })
    );

    await waitFor(() => expect(mockSetAuth).toHaveBeenCalledWith('token-1', expect.objectContaining({ id: 7 })));
    expect(loginSuccessListener).toHaveBeenCalledTimes(1);
    expect(mockNavigate).toHaveBeenCalledWith('/detail?id=42');

    window.removeEventListener('login-success', loginSuccessListener);
  });

  it('shows validation error before requesting login when phone is empty', () => {
    const fetchMock = jest.fn();
    global.fetch = fetchMock;

    render(
      <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <Login />
      </MemoryRouter>
    );

    fireEvent.change(screen.getByLabelText('密码'), { target: { value: 'secret123' } });
    fireEvent.click(screen.getByRole('button', { name: '登录' }));

    expect(screen.getByText('请输入手机号')).toBeInTheDocument();
    expect(fetchMock).not.toHaveBeenCalled();
  });
});
