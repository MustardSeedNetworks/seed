/**
 * The page header's (?) opens this drawer on the page's own section
 * (seed#1943). The interesting case is the second visit: the reader opens
 * help on a page, browses to another section, closes, and clicks the same
 * page's (?) again — it must land back on that page's section, not on
 * wherever they wandered. The shell clears the target on close, which is
 * what makes the repeat open a state change the drawer can act on.
 */
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import { HelpDrawer } from './HelpDrawer';

// The pane's first heading is the section title; block headings follow it.
const heading = (): string =>
  within(screen.getByTestId('help-drawer-content')).getAllByRole('heading')[0].textContent ?? '';

describe('HelpDrawer — section targeting', () => {
  it('opens on the requested section', () => {
    render(<HelpDrawer isOpen={true} onClose={() => {}} section="wifi" />);
    expect(heading()).toBe('WiFi Status');
  });

  it('keeps the last-read section when help is opened without a target', async () => {
    const { rerender } = render(<HelpDrawer isOpen={true} onClose={() => {}} section="wifi" />);
    rerender(<HelpDrawer isOpen={false} onClose={() => {}} />);
    rerender(<HelpDrawer isOpen={true} onClose={() => {}} />);

    expect(heading()).toBe('WiFi Status');
  });

  it('re-applies the same target after the reader browses away and reopens', async () => {
    const user = userEvent.setup();
    const { rerender } = render(<HelpDrawer isOpen={true} onClose={() => {}} section="wifi" />);

    await user.click(screen.getByRole('button', { name: 'Glossary' }));
    expect(heading()).toBe('Glossary');

    // Close clears the target, then the same (?) is clicked again.
    rerender(<HelpDrawer isOpen={false} onClose={() => {}} />);
    rerender(<HelpDrawer isOpen={true} onClose={() => {}} section="wifi" />);

    expect(heading()).toBe('WiFi Status');
  });
});
