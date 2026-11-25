import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { FormField } from './FormField';
import { Input } from './Input';
import { Textarea } from './Textarea';

afterEach(cleanup);

describe('Input and Textarea', () => {
  it('render native controls themed from tokens', () => {
    render(
      <>
        <Input aria-label="Name" />
        <Textarea aria-label="Notes" />
      </>,
    );
    const input = screen.getByRole('textbox', { name: 'Name' });
    expect(input.tagName).toBe('INPUT');
    expect(input.className).toContain('bg-surface-raised');
    expect(input.className).toContain('border-surface-border');
    expect(input.className).toContain('min-w-0');
    expect(screen.getByRole('textbox', { name: 'Notes' }).tagName).toBe('TEXTAREA');
  });

  it('take the control props FormField supplies, and flag an error', () => {
    render(
      <FormField label="Name" error="Required">
        {(control) => <Input {...control} />}
      </FormField>,
    );
    const input = screen.getByRole('textbox', { name: 'Name' });
    expect(input.getAttribute('aria-invalid')).toBe('true');
    expect(input.className).toContain('aria-[invalid=true]:border-status-error');
  });

  it('pass typing through', () => {
    const onChange = vi.fn();
    render(<Input aria-label="Name" onChange={onChange} />);
    fireEvent.change(screen.getByRole('textbox'), { target: { value: 'x' } });
    expect(onChange).toHaveBeenCalledOnce();
  });

  it('defaults the textarea to four rows', () => {
    render(<Textarea aria-label="Notes" />);
    expect(screen.getByRole('textbox').getAttribute('rows')).toBe('4');
  });
});
