import { readFileSync } from 'fs';
import { join } from 'path';

const colorsCss = readFileSync(join(__dirname, '..', 'colors.css'), 'utf8');

const phase2Tokens = [
  '--page-gradient-profile',
  '--surface-glass',
  '--surface-muted',
  '--chip-bg',
  '--chip-border',
  '--avatar-bg',
  '--avatar-border',
  '--avatar-shadow',
  '--icon-tile-bg',
  '--card-border-accent',
  '--item-subtle-bg',
  '--danger-bg',
  '--danger-border',
  '--danger-text',
  '--skeleton-bg',
  '--skeleton-wave',
  '--focus-ring',
];

describe('phase 2 theme tokens', () => {
  it('defines every phase 2 token in the dark/default theme block', () => {
    const darkBlock =
      colorsCss.match(/:root\[data-theme="dark"\],[\s\S]*?\n\}/)?.[0] ?? '';

    for (const token of phase2Tokens) {
      expect(darkBlock).toContain(token);
    }
  });

  it('defines every phase 2 token in the light theme block', () => {
    const lightBlock =
      colorsCss.match(/:root\[data-theme="light"\] \{[\s\S]*?\n\}/)?.[0] ?? '';

    for (const token of phase2Tokens) {
      expect(lightBlock).toContain(token);
    }
  });
});
