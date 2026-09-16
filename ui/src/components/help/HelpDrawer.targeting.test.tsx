/**
 * The page header's (?) opens this drawer on the page's own section
 * (seed#1943). The interesting case is the second visit: the reader opens
 * help on a page, browses to another section, closes, and clicks the same
 * page's (?) again — it must land back on that page's section, not on
 * wherever they wandered. The shell clears the target on close, which is
 * what makes the repeat open a state change the drawer can act on.
 */
import { cleanup, render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import i18next from 'i18next';
import { afterEach, describe, expect, it } from 'vitest';
import { must } from '../../test/must';
import { HelpDrawer } from './HelpDrawer';

// The pane's first heading is the section title; block headings follow it.
const heading = (): string =>
  must(
    within(screen.getByTestId('help-drawer-content')).getAllByRole('heading')[0],
    'first heading',
  ).textContent ?? '';

describe('HelpDrawer — section targeting', () => {
  it('opens on the requested section', () => {
    render(<HelpDrawer isOpen={true} onClose={() => {}} section="wifi" />);
    expect(heading()).toBe('Wi-Fi Status');
  });

  it('clears search when reopening for another page', async () => {
    const user = userEvent.setup();
    const { rerender } = render(<HelpDrawer isOpen={true} onClose={() => {}} section="network" />);
    await user.type(
      screen.getByPlaceholderText(i18next.t('help:modal.searchPlaceholder')),
      'glossary',
    );
    rerender(<HelpDrawer isOpen={false} onClose={() => {}} />);
    rerender(<HelpDrawer isOpen={true} onClose={() => {}} section="security" />);
    expect(screen.getByPlaceholderText(i18next.t('help:modal.searchPlaceholder'))).toHaveValue('');
    expect(heading()).toBe(i18next.t('help:sections.security'));
  });

  it('keeps the last-read section when help is opened without a target', async () => {
    const { rerender } = render(<HelpDrawer isOpen={true} onClose={() => {}} section="wifi" />);
    rerender(<HelpDrawer isOpen={false} onClose={() => {}} />);
    rerender(<HelpDrawer isOpen={true} onClose={() => {}} />);

    expect(heading()).toBe('Wi-Fi Status');
  });

  it('re-applies the same target after the reader browses away and reopens', async () => {
    const user = userEvent.setup();
    const { rerender } = render(<HelpDrawer isOpen={true} onClose={() => {}} section="wifi" />);

    await user.click(screen.getByRole('button', { name: 'Glossary' }));
    expect(heading()).toBe('Glossary');

    // Close clears the target, then the same (?) is clicked again.
    rerender(<HelpDrawer isOpen={false} onClose={() => {}} />);
    rerender(<HelpDrawer isOpen={true} onClose={() => {}} section="wifi" />);

    expect(heading()).toBe('Wi-Fi Status');
  });
});

describe('HelpDrawer — rendered content', () => {
  afterEach(async () => {
    cleanup();
    await i18next.changeLanguage('en');
  });

  it.each(['0.214.55', 'v0.214.55'])('shows one version prefix for %s', (version) => {
    render(<HelpDrawer isOpen={true} onClose={() => {}} version={version} />);
    expect(screen.getByText('v0.214.55')).toBeVisible();
  });

  it('renders and searches Spanish body text', async () => {
    await i18next.changeLanguage('es');
    const user = userEvent.setup();
    render(<HelpDrawer isOpen={true} onClose={() => {}} section="dns" />);
    expect(screen.getByTestId('help-drawer-content')).toHaveTextContent(
      i18next.t('help:content.dnsTests.description'),
    );
    await user.type(
      screen.getByPlaceholderText(i18next.t('help:modal.searchPlaceholder')),
      'resolución',
    );
    expect(screen.getByRole('button', { name: i18next.t('help:sections.dns') })).toBeVisible();
  });

  it.each([
    ['security', ['Autenticación de múltiples factores', 'Bluetooth']],
    ['network', ['Caché de vecinos', 'Bonjour']],
  ])('explains the current %s cards in Spanish', async (section, labels) => {
    await i18next.changeLanguage('es');
    render(<HelpDrawer isOpen={true} onClose={() => {}} section={section} />);
    for (const label of labels) {
      expect(screen.getByTestId('help-drawer-content')).toHaveTextContent(label);
    }
  });
});
