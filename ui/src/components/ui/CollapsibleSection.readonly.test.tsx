/**
 * CollapsibleSection read-only mode (#1254).
 *
 * The owner's 2026-09-04 decision is that a viewer reads every settings
 * section and mutates none. The header must therefore stay clickable while
 * every control inside the body goes read-only — the reason a blanket
 * `RequireRole` wrapper, and an outer disabled fieldset, are both wrong.
 */

import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';

import { CollapsibleSection } from './CollapsibleSection';

const REASON = 'Read-only — operator role required to change these settings.';

async function openSection(name: RegExp): Promise<void> {
  await userEvent.click(screen.getByRole('button', { name }));
}

describe('CollapsibleSection — readOnlyReason', () => {
  it('disables every control in the body and says why', async () => {
    render(
      <CollapsibleSection title="Link" readOnlyReason={REASON}>
        <input aria-label="mode" />
        <button type="button">Save</button>
      </CollapsibleSection>,
    );

    expect(screen.getByText(REASON)).toBeInTheDocument();
    await openSection(/link/i);

    expect(screen.getByLabelText('mode')).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Save' })).toBeDisabled();
  });

  it('leaves the header clickable so the section is still readable', async () => {
    render(
      <CollapsibleSection title="Link" readOnlyReason={REASON}>
        <p>current mode: auto</p>
      </CollapsibleSection>,
    );

    await openSection(/link/i);

    expect(screen.getByText('current mode: auto')).toBeInTheDocument();
  });

  // The read-only body carries the same classes as the writable one, so the
  // section does not silently change shape by role. `border-0` on the fieldset
  // was the live trap: tailwind-merge drops the body's own `border-t` for it.
  it('styles the read-only body the same as the writable one', async () => {
    const { unmount } = render(
      <CollapsibleSection title="Link" readOnlyReason={REASON}>
        <p>body</p>
      </CollapsibleSection>,
    );
    await openSection(/link/i);
    const readOnlyClasses = screen.getByText('body').parentElement?.className ?? '';
    unmount();

    render(
      <CollapsibleSection title="Link">
        <p>body</p>
      </CollapsibleSection>,
    );
    await openSection(/link/i);
    const writableClasses = screen.getByText('body').parentElement?.className ?? '';

    expect(writableClasses).not.toBe('');
    for (const cls of writableClasses.split(' ')) {
      expect(readOnlyClasses.split(' ')).toContain(cls);
    }
  });

  it('leaves controls usable when no reason is given', async () => {
    render(
      <CollapsibleSection title="Link">
        <button type="button">Save</button>
      </CollapsibleSection>,
    );

    await openSection(/link/i);

    expect(screen.getByRole('button', { name: 'Save' })).not.toBeDisabled();
    expect(screen.queryByText(REASON)).not.toBeInTheDocument();
  });
});
