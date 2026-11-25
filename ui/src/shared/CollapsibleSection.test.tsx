import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { CollapsibleSection } from './CollapsibleSection';

afterEach(cleanup);

describe('CollapsibleSection', () => {
  it('is closed by default, with the title as a button that controls the region', () => {
    render(
      <CollapsibleSection title="Advanced">
        <p>body</p>
      </CollapsibleSection>,
    );
    const button = screen.getByRole('button', { name: 'Advanced' });
    expect(button.getAttribute('aria-expanded')).toBe('false');
    expect(
      document.getElementById(button.getAttribute('aria-controls') ?? '')?.hasAttribute('hidden'),
    ).toBe(true);
  });

  it('opens and closes on click', () => {
    render(
      <CollapsibleSection title="Advanced">
        <p>body</p>
      </CollapsibleSection>,
    );
    const button = screen.getByRole('button', { name: 'Advanced' });
    fireEvent.click(button);
    expect(button.getAttribute('aria-expanded')).toBe('true');
    expect(screen.getByText('body').closest('section')?.hasAttribute('hidden')).toBe(false);
    fireEvent.click(button);
    expect(button.getAttribute('aria-expanded')).toBe('false');
  });

  it('can start open', () => {
    render(
      <CollapsibleSection title="Advanced" defaultOpen>
        <p>body</p>
      </CollapsibleSection>,
    );
    expect(screen.getByRole('button').getAttribute('aria-expanded')).toBe('true');
  });
});
