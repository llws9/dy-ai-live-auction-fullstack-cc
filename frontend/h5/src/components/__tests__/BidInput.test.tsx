import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import BidInput from '../BidInput';
import { AuthProvider } from '../../store/authContext';

// Mock useNavigate
const mockNavigate = jest.fn();
jest.mock('react-router-dom', () => ({
  useNavigate: () => mockNavigate,
}));

// Mock bidApi
jest.mock('../../services/api', () => ({
  bidApi: {
    placeBid: jest.fn(),
  },
}));

describe('BidInput Component', () => {
  const defaultProps = {
    auctionId: 1,
    currentPrice: 100,
    minIncrement: 10,
  };

  const renderWithAuth = (component: React.ReactElement) => {
    return render(<AuthProvider>{component}</AuthProvider>);
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('should render bid input with current price', () => {
    renderWithAuth(<BidInput {...defaultProps} />);

    expect(screen.getByText(/当前价格/i)).toBeInTheDocument();
    expect(screen.getByText('¥100.00')).toBeInTheDocument();
  });

  it('should show minimum bid amount', () => {
    renderWithAuth(<BidInput {...defaultProps} />);

    expect(screen.getByText(/最小出价/i)).toBeInTheDocument();
    expect(screen.getByPlaceholderText(/最低出价/i)).toBeInTheDocument();
  });

  it('should validate minimum bid amount', async () => {
    renderWithAuth(<BidInput {...defaultProps} />);

    const input = screen.getByPlaceholderText(/最低出价/i);
    fireEvent.change(input, { target: { value: '50' } });
    fireEvent.blur(input);

    await waitFor(() => {
      expect(screen.getByText(/不能低于/i)).toBeInTheDocument();
    });
  });

  it('should validate decimal places', async () => {
    renderWithAuth(<BidInput {...defaultProps} />);

    const input = screen.getByPlaceholderText(/最低出价/i);
    fireEvent.change(input, { target: { value: '110.123' } });
    fireEvent.blur(input);

    await waitFor(() => {
      expect(screen.getByText(/小数点后2位/i)).toBeInTheDocument();
    });
  });

  it('should have quick bid buttons', () => {
    renderWithAuth(<BidInput {...defaultProps} />);

    expect(screen.getByText('最低价')).toBeInTheDocument();
    expect(screen.getByText('+¥10')).toBeInTheDocument();
    expect(screen.getByText('+¥50')).toBeInTheDocument();
  });

  it('should update amount when quick bid button clicked', () => {
    renderWithAuth(<BidInput {...defaultProps} />);

    const input = screen.getByPlaceholderText(/最低出价/i) as HTMLInputElement;
    const quickBidButton = screen.getByText('+¥10');

    fireEvent.click(quickBidButton);

    expect(input.value).toBe('120.00');
  });

  it('should disable submit button when there is an error', async () => {
    renderWithAuth(<BidInput {...defaultProps} />);

    const input = screen.getByPlaceholderText(/最低出价/i);
    fireEvent.change(input, { target: { value: '50' } });
    fireEvent.blur(input);

    await waitFor(() => {
      const submitButton = screen.getByText('立即出价');
      expect(submitButton).toBeDisabled();
    });
  });

  it('should show login prompt for unauthenticated users', () => {
    renderWithAuth(<BidInput {...defaultProps} />);

    const loginPrompt = screen.getByText(/登录后才能出价/i);
    expect(loginPrompt).toBeInTheDocument();
  });
});
