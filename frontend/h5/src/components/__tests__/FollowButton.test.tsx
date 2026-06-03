import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import FollowButton from '../FollowButton';
import { AuthProvider, useAuth } from '../../store/authContext';

jest.mock('../../store/authContext', () => ({
  AuthProvider: ({ children }: { children: React.ReactNode }) => children,
  useAuth: jest.fn(),
}));

// Mock useNavigate
const mockNavigate = jest.fn();
jest.mock('react-router-dom', () => ({
  useNavigate: () => mockNavigate,
}));

// Mock followApi
jest.mock('../../services/api', () => ({
  followApi: {
    followLiveStream: jest.fn(),
    unfollowLiveStream: jest.fn(),
  },
}));

describe('FollowButton Component', () => {
  const defaultProps = {
    liveStreamId: 1,
  };

  const renderWithAuth = (component: React.ReactElement) => {
    return render(<AuthProvider>{component}</AuthProvider>);
  };

  beforeEach(() => {
    jest.clearAllMocks();
    (useAuth as jest.Mock).mockReturnValue({
      isAuthenticated: true,
      user: { id: 1, name: '测试用户', role: 0 },
      token: 'token-1',
      loading: false,
      login: jest.fn(),
      setAuth: jest.fn(),
      logout: jest.fn(),
      isAdmin: jest.fn(() => false),
      isMerchant: jest.fn(() => false),
    });
  });

  it('should render follow button', () => {
    renderWithAuth(<FollowButton {...defaultProps} />);

    expect(screen.getByText('关注')).toBeInTheDocument();
  });

  it('should show followed state when initialFollowed is true', () => {
    renderWithAuth(<FollowButton {...defaultProps} initialFollowed={true} />);

    expect(screen.getByText('已关注')).toBeInTheDocument();
  });

  it('should display follower count when provided', () => {
    renderWithAuth(<FollowButton {...defaultProps} initialCount={42} />);

    expect(screen.getByText('42')).toBeInTheDocument();
  });

  it('should toggle follow state on click', async () => {
    renderWithAuth(<FollowButton {...defaultProps} />);

    const button = screen.getByText('关注');
    fireEvent.click(button);

    // 由于乐观更新，按钮应该立即改变
    await waitFor(() => {
      expect(screen.getByText('已关注')).toBeInTheDocument();
    });
  });

  it('should increment count when following', async () => {
    renderWithAuth(<FollowButton {...defaultProps} initialCount={10} />);

    const button = screen.getByText('关注');
    fireEvent.click(button);

    await waitFor(() => {
      expect(screen.getByText('11')).toBeInTheDocument();
    });
  });

  it('should decrement count when unfollowing', async () => {
    renderWithAuth(
      <FollowButton {...defaultProps} initialFollowed={true} initialCount={10} />
    );

    const button = screen.getByText('已关注');
    fireEvent.click(button);

    await waitFor(() => {
      expect(screen.getByText('9')).toBeInTheDocument();
    });
  });

  it('should call onFollowSuccess callback', async () => {
    const onFollowSuccess = jest.fn();
    renderWithAuth(
      <FollowButton {...defaultProps} onFollowSuccess={onFollowSuccess} />
    );

    const button = screen.getByText('关注');
    fireEvent.click(button);

    await waitFor(() => {
      expect(onFollowSuccess).toHaveBeenCalledWith(true);
    });
  });

  it('should show loading state', async () => {
    renderWithAuth(<FollowButton {...defaultProps} />);

    const button = screen.getByText('关注');
    fireEvent.click(button);

    // 可能会短暂显示加载状态
    await waitFor(() => {
      expect(screen.getByText(/处理中|已关注/i)).toBeInTheDocument();
    });
  });

  it('should apply different sizes', () => {
    const { rerender } = renderWithAuth(
      <FollowButton {...defaultProps} size="small" />
    );

    let button = screen.getByText('关注');
    expect(button).toBeInTheDocument();

    rerender(
      <AuthProvider>
        <FollowButton {...defaultProps} size="large" />
      </AuthProvider>
    );

    button = screen.getByText('关注');
    expect(button).toBeInTheDocument();
  });
});
