import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { FormField } from './FormField';

afterEach(cleanup);

describe('FormField', () => {
  it('labels the control it is given through a generated id', () => {
    render(<FormField label="Name">{(control) => <input {...control} />}</FormField>);
    expect(screen.getByLabelText('Name').tagName).toBe('INPUT');
  });

  it('links the hint and the error to the control, and marks it invalid', () => {
    render(
      <FormField label="Name" hint="Shown in the rail" error="Required">
        {(control) => <input {...control} />}
      </FormField>,
    );
    const input = screen.getByLabelText('Name');
    const described = (input.getAttribute('aria-describedby') ?? '').split(' ');
    expect(described.map((id) => document.getElementById(id)?.textContent)).toEqual([
      'Shown in the rail',
      'Required',
    ]);
    expect(input.getAttribute('aria-invalid')).toBe('true');
    expect(screen.getByRole('alert').textContent).toBe('Required');
  });

  it('leaves the control clean when there is no hint, error or requirement', () => {
    render(<FormField label="Name">{(control) => <input {...control} />}</FormField>);
    const input = screen.getByLabelText('Name');
    expect(input.getAttribute('aria-describedby')).toBeNull();
    expect(input.getAttribute('aria-invalid')).toBeNull();
    expect(input.getAttribute('aria-required')).toBeNull();
  });

  it('marks a required control for assistive technology and hides the asterisk', () => {
    render(
      <FormField label="Name" required>
        {(control) => <input {...control} />}
      </FormField>,
    );
    expect(screen.getByRole('textbox', { name: 'Name' }).getAttribute('aria-required')).toBe(
      'true',
    );
    expect(screen.getByText('*').getAttribute('aria-hidden')).toBe('true');
  });

  it('caps its width: md by default, lg when asked, never full width', () => {
    const { container, rerender } = render(
      <FormField label="A">{(c) => <input {...c} />}</FormField>,
    );
    expect(container.firstElementChild?.className).toContain('max-w-md');
    rerender(
      <FormField label="A" size="lg">
        {(c) => <input {...c} />}
      </FormField>,
    );
    expect(container.firstElementChild?.className).toContain('max-w-2xl');
    expect(container.firstElementChild?.className).toContain('w-full');
  });
});
