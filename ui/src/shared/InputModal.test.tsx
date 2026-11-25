import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { InputModal } from './InputModal';

afterEach(cleanup);

const modal = (props: Partial<React.ComponentProps<typeof InputModal>> = {}) => (
  <InputModal
    open
    title="Rename profile"
    label="Name"
    submitLabel="Rename"
    cancelLabel="Cancel"
    closeLabel="Close"
    initialValue="lab"
    onSubmit={() => {}}
    onCancel={() => {}}
    {...props}
  />
);

describe('InputModal', () => {
  it('labels the field, starts with the initial value and focuses it', () => {
    render(modal());
    const input = screen.getByRole('textbox', { name: 'Name' }) as HTMLInputElement;
    expect(input.value).toBe('lab');
    expect(document.activeElement).toBe(input);
  });

  it('submits the edited value from the submit button and from Enter', () => {
    const onSubmit = vi.fn();
    render(modal({ onSubmit }));
    fireEvent.change(screen.getByRole('textbox'), { target: { value: 'prod' } });
    fireEvent.click(screen.getByRole('button', { name: 'Rename' }));
    fireEvent.submit(screen.getByRole('textbox').closest('form') as HTMLFormElement);
    expect(onSubmit).toHaveBeenNthCalledWith(1, 'prod');
    expect(onSubmit).toHaveBeenCalledTimes(2);
  });

  it("shows the caller's error against the field and stays open", () => {
    render(modal({ error: 'Name taken' }));
    expect(screen.getByRole('alert').textContent).toBe('Name taken');
    expect(screen.getByRole('textbox').getAttribute('aria-invalid')).toBe('true');
    expect(screen.getByRole('dialog')).toBeTruthy();
  });

  it('cancels from the cancel button', () => {
    const onCancel = vi.fn();
    render(modal({ onCancel }));
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(onCancel).toHaveBeenCalledOnce();
  });
});
