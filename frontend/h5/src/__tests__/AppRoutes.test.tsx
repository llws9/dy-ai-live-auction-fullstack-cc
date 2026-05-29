import { render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom';
import { LegacyAuctionRedirect, LegacyResultRedirect } from '../routes/legacyRedirects';

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
});
