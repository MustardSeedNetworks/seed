import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { useState } from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { RadioGroup, type RadioGroupProps } from './RadioGroup';

afterEach(cleanup);

const options = [
  { value: 'reflector', label: 'Reflector', description: 'Answers tests' },
  { value: 'master', label: 'Test Master' },
  { value: 'off', label: 'Off', disabled: true },
] as const;

function group(overrides: Partial<RadioGroupProps> = {}) {
  return (
    <RadioGroup
      label="Role"
      name="role"
      options={options}
      value="reflector"
      onValueChange={() => {}}
      {...overrides}
    />
  );
}

function Harness() {
  const [value, setValue] = useState('reflector');
  return group({ value, onValueChange: setValue });
}

describe('RadioGroup', () => {
  it('is a fieldset named by its legend, holding native radios of one name', () => {
    render(group());
    const fieldset = screen.getByRole('group', { name: 'Role' });
    expect(fieldset.tagName).toBe('FIELDSET');
    const radios = screen.getAllByRole('radio') as HTMLInputElement[];
    expect(radios.map((radio) => radio.tagName)).toEqual(['INPUT', 'INPUT', 'INPUT']);
    expect(new Set(radios.map((radio) => radio.name))).toEqual(new Set(['role']));
    expect(radios.map((radio) => radio.type)).toEqual(['radio', 'radio', 'radio']);
  });

  it('names each radio by its label and links the description', () => {
    render(group());
    const radio = screen.getByRole('radio', { name: 'Reflector' });
    const description = screen.getByText('Answers tests');
    expect(radio.getAttribute('aria-describedby')).toBe(description.id);
    expect(
      screen.getByRole('radio', { name: 'Test Master' }).hasAttribute('aria-describedby'),
    ).toBe(false);
  });

  it('checks the option matching value, and only that one', () => {
    render(group({ value: 'master' }));
    expect(
      (screen.getAllByRole('radio') as HTMLInputElement[]).map((radio) => radio.checked),
    ).toEqual([false, true, false]);
  });

  it('reports the chosen value, from the radio or its label', () => {
    const onValueChange = vi.fn();
    render(group({ onValueChange }));
    fireEvent.click(screen.getByRole('radio', { name: 'Test Master' }));
    fireEvent.click(screen.getByText('Test Master'));
    expect(onValueChange).toHaveBeenCalledWith('master');
  });

  it('is controlled: it follows the value it is given', () => {
    render(<Harness />);
    fireEvent.click(screen.getByRole('radio', { name: 'Test Master' }));
    expect((screen.getByRole('radio', { name: 'Test Master' }) as HTMLInputElement).checked).toBe(
      true,
    );
    expect((screen.getByRole('radio', { name: 'Reflector' }) as HTMLInputElement).checked).toBe(
      false,
    );
  });

  it('keeps a disabled option out of reach, and disables all with disabled', () => {
    const { rerender } = render(group());
    const off = screen.getByRole('radio', { name: 'Off' }) as HTMLInputElement;
    expect(off.disabled).toBe(true);
    rerender(group({ disabled: true }));
    expect(
      (screen.getAllByRole('radio') as HTMLInputElement[]).every((radio) => radio.disabled),
    ).toBe(true);
  });

  it('keeps a 24px target and a label that is part of it', () => {
    render(group());
    const radio = screen.getByRole('radio', { name: 'Reflector' });
    expect(radio.className).toContain('h-6');
    expect(radio.className).toContain('w-6');
    expect(screen.getByText('Reflector').tagName).toBe('LABEL');
    expect(screen.getByText('Reflector').getAttribute('for')).toBe(radio.id);
  });

  it('lays out in a column by default and a wrapping row when horizontal', () => {
    const { rerender } = render(group({ 'data-testid': 'opts' }));
    expect(screen.getByTestId('opts').className).toContain('flex-col');
    rerender(group({ 'data-testid': 'opts', orientation: 'horizontal' }));
    expect(screen.getByTestId('opts').className).toContain('flex-wrap');
  });

  it('attaches hint and error to the group, announces the error and flags every radio', () => {
    render(group({ hint: 'Pick one', error: 'A role is required', id: 'r' }));
    const fieldset = screen.getByRole('group', { name: 'Role' });
    expect(fieldset.getAttribute('aria-describedby')).toBe('r-hint r-error');
    expect(screen.getByRole('alert').textContent).toBe('A role is required');
    for (const radio of screen.getAllByRole('radio')) {
      expect(radio.getAttribute('aria-invalid')).toBe('true');
    }
  });

  it('has no alert and no aria-invalid without an error', () => {
    render(group());
    expect(screen.queryByRole('alert')).toBeNull();
    expect(screen.getAllByRole('radio')[0]?.hasAttribute('aria-invalid')).toBe(false);
  });

  it('marks required on the legend and on each radio', () => {
    render(group({ required: true }));
    expect(
      screen.getByRole('group').querySelector('legend [aria-hidden="true"]')?.textContent,
    ).toBe('*');
    expect((screen.getAllByRole('radio')[0] as HTMLInputElement).required).toBe(true);
  });

  it('puts data-testid on each option for specs', () => {
    render(
      group({
        options: [
          { value: 'a', label: 'A', 'data-testid': 'opt-a' },
          { value: 'b', label: 'B', 'data-testid': 'opt-b' },
        ],
      }),
    );
    expect(screen.getByTestId('opt-a').getAttribute('type')).toBe('radio');
  });

  it('is reached by Tab as the checked radio (native roving), in group order', () => {
    render(group());
    const radios = screen.getAllByRole('radio') as HTMLInputElement[];
    radios[0]?.focus();
    expect(document.activeElement).toBe(radios[0]);
    expect(radios.every((radio) => radio.tabIndex === 0 || radio.tabIndex === -1)).toBe(true);
  });
});
