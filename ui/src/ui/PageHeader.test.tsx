/**
 * The header is only an entry point to help now — the drawer owns the content
 * (seed#1943). A page with no drawer section must therefore show no (?), so
 * the button never promises help that isn't there.
 */
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import { PageHeader } from './PageHeader';

describe('PageHeader — help entry point', () => {
  it('shows no help button when the page has no drawer section', () => {
    render(<PageHeader title="Reports" />);

    expect(screen.queryByTestId('page-header-help-button')).toBeNull();
  });

  it('delegates the click to the shell instead of rendering its own panel', async () => {
    const onHelp = vi.fn();
    render(<PageHeader title="Reports" onHelp={onHelp} />);

    await userEvent.click(screen.getByTestId('page-header-help-button'));

    expect(onHelp).toHaveBeenCalledOnce();
    expect(screen.queryByRole('dialog')).toBeNull();
  });
});
