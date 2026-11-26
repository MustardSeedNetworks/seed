import { act, cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { FormField } from './FormField';
import { FormSection } from './FormSection';
import { HelpIcon } from './HelpIcon';

afterEach(cleanup);

describe('HelpIcon', () => {
  it('is a named button, with the text in a tooltip that describes it', () => {
    render(<HelpIcon label="Help: Timeout" text="How long to wait" />);
    const button = screen.getByRole('button', { name: 'Help: Timeout' });
    const tooltip = screen.getByRole('tooltip', { hidden: true });
    expect(button.getAttribute('aria-describedby')).toContain(tooltip.id);
    expect(tooltip.textContent).toBe('How long to wait');
  });

  it('is a 24px target', () => {
    render(<HelpIcon label="Help" text="x" />);
    const { className } = screen.getByRole('button');
    expect(className).toContain('h-6');
    expect(className).toContain('w-6');
  });

  it('takes focus on a tap, which is what opens the tooltip where click does not focus', () => {
    render(<HelpIcon label="Help" text="x" />);
    const button = screen.getByRole('button');
    expect(document.activeElement).not.toBe(button);
    fireEvent.click(button);
    expect(document.activeElement).toBe(button);
    expect(screen.getByRole('tooltip').className).not.toContain('invisible');
  });

  it('opens on keyboard focus and closes on Escape', () => {
    render(<HelpIcon label="Help" text="x" />);
    act(() => {
      screen.getByRole('button').focus();
    });
    expect(screen.getByRole('tooltip').className).not.toContain('invisible');
    fireEvent.keyDown(screen.getByRole('button'), { key: 'Escape' });
    expect(screen.getByRole('tooltip', { hidden: true }).className).toContain('invisible');
  });

  it('does not toggle a control it sits beside, nor submit a form', () => {
    const onChange = vi.fn();
    const onSubmit = vi.fn((event: { preventDefault: () => void }) => event.preventDefault());
    render(
      <form onSubmit={onSubmit}>
        <label>
          Enable
          <input type="checkbox" onChange={onChange} />
          <HelpIcon label="Help" text="x" />
        </label>
      </form>,
    );
    fireEvent.click(screen.getByRole('button', { name: 'Help' }));
    expect(onChange).not.toHaveBeenCalled();
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it('sits beside a FormField label, outside it', () => {
    render(
      <FormField label="Timeout" labelAccessory={<HelpIcon label="Help: Timeout" text="ms" />}>
        {(control) => <input {...control} />}
      </FormField>,
    );
    expect(screen.getByLabelText('Timeout').tagName).toBe('INPUT');
    expect(screen.getByRole('button', { name: 'Help: Timeout' }).closest('label')).toBeNull();
  });
});

describe('FormSection', () => {
  it('is a group named by its h3 title, with help beside it and a description under it', () => {
    render(
      <FormSection
        title="Timing"
        description="How the test paces frames"
        help={<HelpIcon label="Help: Timing" text="x" />}
      >
        <input aria-label="Duration" />
      </FormSection>,
    );
    const group = screen.getByRole('region', { name: 'Timing' });
    expect(group.contains(screen.getByRole('heading', { level: 3, name: 'Timing' }))).toBe(true);
    expect(group.contains(screen.getByRole('button', { name: 'Help: Timing' }))).toBe(true);
    expect(screen.getByText('How the test paces frames')).toBeTruthy();
    expect(group.contains(screen.getByLabelText('Duration'))).toBe(true);
  });

  it('needs no help or description', () => {
    render(
      <FormSection title="Frame">
        <p>x</p>
      </FormSection>,
    );
    expect(screen.queryByRole('button')).toBeNull();
  });
});
