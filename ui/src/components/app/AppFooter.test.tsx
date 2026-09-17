/**
 * AppFooter.test.tsx — the footer's interpolated copy, asserted (UI-SEED-4).
 *
 * `footer.byCompany` and `footer.copyright` both carry `{{company}}`, and the
 * copyright carries `{{year}}` too. Nothing passed them, so every
 * authenticated page read "by {{company}}" verbatim, and the component
 * prefixed a second `©` onto a string that already had one. Rendering with the
 * real i18n instance is the only assertion that can see either.
 */
import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import i18n from '../../i18n';
import { AppFooter } from './AppFooter';

const year = String(new Date().getFullYear());

describe('AppFooter', () => {
  it('names the company instead of the placeholder', async () => {
    await i18n.changeLanguage('en');
    render(<AppFooter appVersion="0.214.63" />);

    expect(screen.getByText('by Mustard Seed Networks')).toBeInTheDocument();
    expect(screen.queryByText(/\{\{/)).toBeNull();
  });

  it('renders exactly one copyright line', async () => {
    await i18n.changeLanguage('en');
    render(<AppFooter appVersion="0.214.63" />);

    const notices = screen.getAllByText(/All rights reserved/);

    expect(notices).toHaveLength(1);
    expect(notices[0]).toHaveTextContent(`© ${year} Mustard Seed Networks. All rights reserved.`);
  });

  it('keeps the company verbatim in Spanish', async () => {
    await i18n.changeLanguage('es');
    render(<AppFooter appVersion="0.214.63" />);

    expect(screen.getByText('por Mustard Seed Networks')).toBeInTheDocument();
    expect(
      screen.getByText(`© ${year} Mustard Seed Networks. Todos los derechos reservados.`),
    ).toBeInTheDocument();

    await i18n.changeLanguage('en');
  });
});
