import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { Button } from './Button';

afterEach(cleanup);

describe('Button', () => {
  it('defaults to a secondary button that does not submit a form', () => {
    render(<Button>Cancel</Button>);
    const button = screen.getByRole('button', { name: 'Cancel' });
    expect(button.getAttribute('type')).toBe('button');
    expect(button.getAttribute('data-variant')).toBe('secondary');
  });

  it('exposes the variant so one-primary-per-page is queryable', () => {
    render(
      <>
        <Button variant="primary">Save</Button>
        <Button>Cancel</Button>
        <Button variant="destructive">Delete</Button>
      </>,
    );
    expect(document.querySelectorAll('[data-variant="primary"]')).toHaveLength(1);
    expect(document.querySelectorAll('[data-variant="destructive"]')).toHaveLength(1);
  });

  it('styles each variant from tokens: brand fill, raised face, filled danger', () => {
    render(
      <>
        <Button variant="primary">P</Button>
        <Button variant="secondary">S</Button>
        <Button variant="destructive">D</Button>
      </>,
    );
    expect(screen.getByText('P').className).toContain('bg-brand-primary');
    expect(screen.getByText('S').className).toContain('bg-surface-raised');
    expect(screen.getByText('D').className).toContain('bg-status-error');
    expect(screen.getByText('D').className).toContain('text-on-danger');
  });

  it('is visibly disabled, not merely faded, and does not fire', () => {
    const onClick = vi.fn();
    render(
      <Button variant="primary" disabled onClick={onClick}>
        Save
      </Button>,
    );
    const button = screen.getByRole('button', { name: 'Save' }) as HTMLButtonElement;
    expect(button.disabled).toBe(true);
    expect(button.className).toContain('disabled:bg-surface-sunken');
    expect(button.className).toContain('disabled:text-text-disabled');
    expect(button.className).toContain('disabled:cursor-not-allowed');
    expect(button.className).not.toContain('opacity');
    fireEvent.click(button);
    expect(onClick).not.toHaveBeenCalled();
  });

  it('calls onClick and hides a decorative icon', () => {
    const onClick = vi.fn();
    render(
      <Button onClick={onClick} icon={<svg data-testid="icon" />}>
        Go
      </Button>,
    );
    fireEvent.click(screen.getByRole('button', { name: 'Go' }));
    expect(onClick).toHaveBeenCalledOnce();
    expect(screen.getByTestId('icon').parentElement?.getAttribute('aria-hidden')).toBe('true');
  });

  it('styles ghost as borderless with no fill until hovered', () => {
    render(<Button variant="ghost">More</Button>);
    const { className } = screen.getByRole('button', { name: 'More' });
    expect(className).toContain('border-transparent');
    expect(className).toContain('bg-transparent');
    expect(className).toContain('hover:bg-surface-hover');
  });

  describe('loading', () => {
    it('shows a spinner, disables the button and marks it busy, keeping the label', () => {
      const onClick = vi.fn();
      render(
        <Button variant="primary" loading onClick={onClick} icon={<svg data-testid="icon" />}>
          Save
        </Button>,
      );
      const button = screen.getByRole('button', { name: 'Save' }) as HTMLButtonElement;
      expect(button.disabled).toBe(true);
      expect(button.getAttribute('aria-busy')).toBe('true');
      expect(button.querySelector('svg.lucide-loader-circle')).toBeTruthy();
      expect(screen.queryByTestId('icon')).toBeNull();
      fireEvent.click(button);
      expect(onClick).not.toHaveBeenCalled();
    });

    it('is not busy when not loading', () => {
      render(<Button>Go</Button>);
      expect(screen.getByRole('button').getAttribute('aria-busy')).toBeNull();
    });
  });

  describe('icon only', () => {
    it('is a square button named by its aria-label, with the icon hidden', () => {
      render(<Button iconOnly icon={<svg data-testid="icon" />} aria-label="Delete device" />);
      const button = screen.getByRole('button', { name: 'Delete device' });
      expect(button.className).toContain('h-10');
      expect(button.className).toContain('w-10');
      expect(button.textContent).toBe('');
      expect(screen.getByTestId('icon').parentElement?.getAttribute('aria-hidden')).toBe('true');
    });

    it('is smaller at size sm', () => {
      render(<Button iconOnly size="sm" icon={<svg />} aria-label="Close" />);
      expect(screen.getByRole('button', { name: 'Close' }).className).toContain('h-8');
    });

    it('cannot be written without an aria-label', () => {
      // @ts-expect-error an icon-only button must carry an aria-label
      const missing = <Button iconOnly icon={<svg />} />;
      expect(missing).toBeTruthy();
    });
  });
});
