import { render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom';
import App from '../App';
import { LegacyAuctionRedirect, LegacyResultRedirect } from '../routes/legacyRedirects';

jest.mock('../pages/Login', () => ({
  __esModule: true,
  default: () => <div>登录页</div>,
}));

jest.mock('../pages/Home', () => ({
  __esModule: true,
  default: () => <div>首页</div>,
}));

jest.mock('../components/DemoConsole', () => {
  const { useDemo } = jest.requireActual('../store/demoContext');

  return {
    __esModule: true,
    default: () => {
      useDemo();
      return <div data-testid="demo-console-mounted">Demo Console</div>;
    },
  };
});

function LocationProbe() {
  const location = useLocation();

  return <div data-testid="location">{location.pathname}{location.search}</div>;
}

describe('App route closure', () => {
  it('redirects legacy auction detail routes to the retained product detail page', () => {
    render(
      <MemoryRouter initialEntries={['/auction/42']}>
        <Routes>
          <Route path="/auction/:id" element={<LegacyAuctionRedirect />} />
          <Route path="/detail" element={<LocationProbe />} />
        </Routes>
      </MemoryRouter>,
    );

    expect(screen.getByTestId('location')).toHaveTextContent('/detail?id=42');
  });

  it('redirects legacy result path params to the retained result query route', () => {
    render(
      <MemoryRouter initialEntries={['/result/42']}>
        <Routes>
          <Route path="/result/:id" element={<LegacyResultRedirect />} />
          <Route path="/result" element={<LocationProbe />} />
        </Routes>
      </MemoryRouter>,
    );

    expect(screen.getByTestId('location')).toHaveTextContent('/result?id=42');
  });

  it('mounts DemoConsole globally inside DemoProvider', async () => {
    render(
      <MemoryRouter initialEntries={['/']}>
        <App />
      </MemoryRouter>,
    );

    expect(await screen.findByTestId('demo-console-mounted')).toBeInTheDocument();
  });
});
