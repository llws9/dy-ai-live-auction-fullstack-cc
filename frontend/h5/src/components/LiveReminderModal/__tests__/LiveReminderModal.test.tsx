import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import LiveReminderModal from '../index';

jest.mock('../../../utils/trackEvent', () => ({
  trackEvent: jest.fn(),
}));

jest.mock('../../../utils/businessEvent', () => ({
  trackBusinessEvent: jest.fn(),
}));

describe('LiveReminderModal', () => {
  it('does not render an empty image src when stream avatar is missing', () => {
    render(
      <MemoryRouter>
        <LiveReminderModal
          isOpen
          onClose={() => {}}
          stream={{ id: 6, name: 'Demo 商家直播间', avatarUrl: '', statusText: '正在直播' }}
        />
      </MemoryRouter>,
    );

    expect(screen.queryByRole('img', { name: 'Demo 商家直播间' })).not.toBeInTheDocument();
    expect(screen.getByText('D')).toBeInTheDocument();
  });
});
