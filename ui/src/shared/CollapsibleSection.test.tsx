import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
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

describe('CollapsibleSection, covering seed usage', () => {
  it('takes a ReactNode title, names the region by it, and shows a count', () => {
    render(
      <CollapsibleSection
        title={<strong data-testid="rich">Server results</strong>}
        count={3}
        defaultOpen
      >
        <p>body</p>
      </CollapsibleSection>,
    );
    expect(screen.getByTestId('rich')).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Server results (3)' })).toBeTruthy();
    expect(screen.getByRole('region', { name: 'Server results (3)' })).toBeTruthy();
  });

  it('shows a count of zero', () => {
    render(
      <CollapsibleSection title="Items" count={0}>
        <p>body</p>
      </CollapsibleSection>,
    );
    expect(screen.getByText('(0)')).toBeTruthy();
  });

  it('shows a status indicator before the title, named by statusLabel', () => {
    render(
      <CollapsibleSection title="Checks" status="warning" statusLabel="Warning">
        <p>body</p>
      </CollapsibleSection>,
    );
    const indicator = screen.getByRole('img', { name: 'Warning' });
    expect(indicator.getAttribute('data-tone')).toBe('warning');
    expect(screen.getByRole('button').contains(indicator)).toBe(true);
  });

  it('is bordered by default and borderless when compact, for inside a Card', () => {
    const { rerender } = render(
      <CollapsibleSection title="A" data-testid="s">
        <p>x</p>
      </CollapsibleSection>,
    );
    expect(screen.getByTestId('s').className).toContain('border');
    rerender(
      <CollapsibleSection title="A" variant="compact" data-testid="s">
        <p>x</p>
      </CollapsibleSection>,
    );
    expect(screen.getByTestId('s').className).not.toContain('border');
  });

  describe('readOnlyReason', () => {
    const readOnly = (
      <CollapsibleSection title="SNMP" readOnlyReason="Operators only" defaultOpen>
        <input aria-label="Community" />
        <button type="button">Save</button>
      </CollapsibleSection>
    );

    it('shows the reason in the header and disables every control in the body', () => {
      render(readOnly);
      expect(screen.getByRole('button', { name: /SNMP/ }).textContent).toContain('Operators only');
      expect((screen.getByLabelText('Community') as HTMLInputElement).matches(':disabled')).toBe(
        true,
      );
      expect(screen.getByRole('button', { name: 'Save' }).matches(':disabled')).toBe(true);
    });

    it('keeps the header a working button, so a viewer can still open and read it', () => {
      render(
        <CollapsibleSection title="SNMP" readOnlyReason="Operators only">
          <input aria-label="Community" />
        </CollapsibleSection>,
      );
      const header = screen.getByRole('button', { name: /SNMP/ });
      expect(header.matches(':disabled')).toBe(false);
      fireEvent.click(header);
      expect(header.getAttribute('aria-expanded')).toBe('true');
      expect(screen.getByLabelText('Community')).toBeTruthy();
    });

    it('leaves a writable section enabled and without a fieldset', () => {
      render(
        <CollapsibleSection title="SNMP" defaultOpen>
          <input aria-label="Community" />
        </CollapsibleSection>,
      );
      expect(document.querySelector('fieldset')).toBeNull();
      expect(screen.getByLabelText('Community').matches(':disabled')).toBe(false);
    });
  });

  it('passes data-testid to the root, whether folded or open', () => {
    render(
      <CollapsibleSection title="A" data-testid="adv">
        <p>x</p>
      </CollapsibleSection>,
    );
    expect(screen.getByTestId('adv').contains(screen.getByRole('button'))).toBe(true);
  });
});

describe('CollapsibleSection, controlled', () => {
  it('is controlled by open, and reports the requested state without changing it itself', () => {
    const onOpenChange = vi.fn();
    const { rerender } = render(
      <CollapsibleSection title="Protocol" open={false} onOpenChange={onOpenChange}>
        <p>body</p>
      </CollapsibleSection>,
    );
    const button = screen.getByRole('button', { name: 'Protocol' });
    fireEvent.click(button);
    expect(onOpenChange).toHaveBeenCalledExactlyOnceWith(true);
    expect(button.getAttribute('aria-expanded')).toBe('false');
    rerender(
      <CollapsibleSection title="Protocol" open onOpenChange={onOpenChange}>
        <p>body</p>
      </CollapsibleSection>,
    );
    expect(button.getAttribute('aria-expanded')).toBe('true');
    fireEvent.click(button);
    expect(onOpenChange).toHaveBeenLastCalledWith(false);
  });

  it('reports changes when uncontrolled too', () => {
    const onOpenChange = vi.fn();
    render(
      <CollapsibleSection title="A" onOpenChange={onOpenChange}>
        <p>x</p>
      </CollapsibleSection>,
    );
    fireEvent.click(screen.getByRole('button', { name: 'A' }));
    expect(onOpenChange).toHaveBeenCalledExactlyOnceWith(true);
  });

  it('takes an id on the root', () => {
    render(
      <CollapsibleSection title="A" id="section-dhcp">
        <p>x</p>
      </CollapsibleSection>,
    );
    expect(document.getElementById('section-dhcp')?.contains(screen.getByRole('button'))).toBe(
      true,
    );
  });

  it('puts a required chip in the title without nesting a control in the button', () => {
    render(
      <CollapsibleSection
        title={
          <>
            Interfaces <span data-testid="chip">Required</span>
          </>
        }
      >
        <p>x</p>
      </CollapsibleSection>,
    );
    expect(screen.getByRole('button', { name: 'Interfaces Required' })).toBeTruthy();
  });
});
