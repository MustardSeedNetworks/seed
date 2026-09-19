/**
 * AppFooter.test.tsx — the footer's interpolated copy, asserted (UI-SEED-4).
 *
 * `footer.copyright` carries `{{company}}` and `{{year}}`. Nothing passed
 * them, so every authenticated page read the placeholders verbatim, and the
 * component prefixed a second `©` onto a string that already had one.
 * Rendering with the real i18n instance is the only assertion that can see it.
 *
 * UI-SEED-20 collapsed the four-column panel to one line and dropped the
 * product name, the `by {{company}}` line and the version, all of which the
 * rail already carries. The copyright is the remaining interpolated string,
 * so it is what both locales are read through now.
 */
import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import i18n from '../../i18n';
import { AppFooter } from './AppFooter';

const year = String(new Date().getFullYear());

describe('AppFooter', () => {
  it('names the company instead of the placeholder', async () => {
    await i18n.changeLanguage('en');
    render(<AppFooter />);

    expect(screen.getByText(new RegExp(`${year} Mustard Seed Networks`))).toBeInTheDocument();
    expect(screen.queryByText(/\{\{/)).toBeNull();
  });

  it('renders exactly one copyright line', async () => {
    await i18n.changeLanguage('en');
    render(<AppFooter />);

    const notices = screen.getAllByText(/All rights reserved/);

    expect(notices).toHaveLength(1);
    expect(notices[0]).toHaveTextContent(`© ${year} Mustard Seed Networks. All rights reserved.`);
  });

  it('keeps the company verbatim in Spanish', async () => {
    await i18n.changeLanguage('es');
    render(<AppFooter />);

    expect(
      screen.getByText(`© ${year} Mustard Seed Networks. Todos los derechos reservados.`),
    ).toBeInTheDocument();

    await i18n.changeLanguage('en');
  });
});
