import { readFileSync } from 'fs';
import { join } from 'path';

const css = readFileSync(join(__dirname, '..', 'Profile.module.css'), 'utf8');

describe('Profile phase 2 theme tokens', () => {
  it('uses profile atmosphere and semantic surface tokens', () => {
    expect(css).toContain('background: var(--page-gradient-profile);');
    expect(css).toContain('border: 2px solid var(--avatar-border);');
    expect(css).toContain('background: var(--avatar-bg);');
    expect(css).toContain('box-shadow: var(--avatar-shadow);');
    expect(css).toContain('background: var(--chip-bg);');
    expect(css).toContain('border: 1px solid var(--card-border-accent);');
    expect(css).toContain('background: var(--item-subtle-bg);');
    expect(css).toContain('background: var(--icon-tile-bg);');
    expect(css).toContain('color: var(--danger-text);');
  });

  it('does not keep dark-only values in theme-sensitive Profile blocks', () => {
    expect(css).not.toContain('linear-gradient(180deg, #242424 0%, #171717 42%, #101010 100%)');
    expect(css).not.toContain('background: rgba(44, 44, 44, 0.82);');
    expect(css).not.toContain('background: rgba(26, 26, 26, 0.64);');
    expect(css).not.toContain('color: #f87171;');
  });

  it('keeps primary CTA text inverse during link interaction states', () => {
    expect(css).toMatch(
      /\.primaryAuctionCta:hover,\s*\.primaryAuctionCta:focus-visible,\s*\.primaryAuctionCta:active\s*\{[\s\S]*?color: var\(--text-inverse\);[\s\S]*?\}/,
    );
    expect(css).toMatch(/\.primaryAuctionCta strong\s*\{[\s\S]*?color: var\(--text-inverse\);[\s\S]*?\}/);
  });

  it('keeps metric badge count text pure white for contrast', () => {
    expect(css).toMatch(/\.metricBadge\s*\{[\s\S]*?--touchpoint-badge-text: #ffffff;[\s\S]*?\}/);
    expect(css).toMatch(/\.metricCard \.metricBadge\s*\{[\s\S]*?color: #ffffff;[\s\S]*?\}/);
  });

  it('places footprint status badge at top-left with profile theme tokens', () => {
    const badgeCss = css.match(/\.footprintStatusBadge\s*\{[\s\S]*?\n\}/)?.[0] ?? '';

    expect(badgeCss).toContain('left: 6px;');
    expect(badgeCss).not.toContain('right: 6px;');
    expect(badgeCss).toContain('border: 1px solid var(--card-border-accent);');
    expect(badgeCss).toContain('background: color-mix(in srgb, var(--bg-surface) 84%, var(--text-brand));');
    expect(badgeCss).toContain('color: var(--text-brand);');
  });
});
