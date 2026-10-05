/**
 * Search work skipped by the React Compiler rather than by useMemo
 * (UI-SEED-41 slice 9, #3066).
 *
 * Compiled, the drawer re-rendered with unchanged props and query keeps the
 * filtered section list it cached, so the section bodies are not flattened
 * to search text again. Fails when the compiler is removed from the vitest
 * config.
 */

import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import i18next from 'i18next';
import { describe, expect, it, vi } from 'vitest';

import { HelpDrawer } from './HelpDrawer';
import { sectionSearchText } from './helpModel';

vi.mock('./helpModel', { spy: true });

describe('compiled help drawer skips work on an unchanged re-render', () => {
  it('does not re-run the search', async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    const { rerender } = render(<HelpDrawer isOpen={true} onClose={onClose} />);
    await user.type(screen.getByPlaceholderText(i18next.t('help:modal.searchPlaceholder')), 'dns');
    const searched = vi.mocked(sectionSearchText).mock.calls.length;
    rerender(<HelpDrawer isOpen={true} onClose={onClose} />);
    expect(searched).toBeGreaterThan(0);
    expect(sectionSearchText).toHaveBeenCalledTimes(searched);
  });
});
