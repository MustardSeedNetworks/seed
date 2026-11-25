import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { Button, ButtonLink } from './Button';

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

describe('Button xs', () => {
  it('is a 24px target, the WCAG 2.5.8 floor, with the small type', () => {
    render(<Button size="xs">Edit</Button>);
    const { className } = screen.getByRole('button', { name: 'Edit' });
    expect(className).toContain('min-h-6');
    expect(className).toContain('px-2');
    expect(className).toContain('text-xs');
  });

  it('is a 24px square when icon-only', () => {
    render(<Button size="xs" iconOnly icon={<svg />} aria-label="Edit" />);
    const { className } = screen.getByRole('button', { name: 'Edit' });
    expect(className).toContain('h-6');
    expect(className).toContain('w-6');
  });
});

describe('ButtonLink', () => {
  it('is an anchor with the Button look, variant marker and icon', () => {
    render(
      <ButtonLink href="/surveys" variant="primary" icon={<svg data-testid="icon" />}>
        Surveys
      </ButtonLink>,
    );
    const link = screen.getByRole('link', { name: 'Surveys' });
    expect(link.getAttribute('href')).toBe('/surveys');
    expect(link.getAttribute('data-variant')).toBe('primary');
    expect(link.className).toContain('bg-brand-primary');
    expect(link.className).toContain('min-h-10');
    expect(link.className).toContain('no-underline');
    expect(screen.getByTestId('icon').parentElement?.getAttribute('aria-hidden')).toBe('true');
  });

  it('has exactly the classes of the Button of the same variant and size', () => {
    render(
      <>
        <Button variant="destructive" size="sm">
          Act
        </Button>
        <ButtonLink href="/x" variant="destructive" size="sm">
          Go
        </ButtonLink>
      </>,
    );
    const button = screen.getByRole('button', { name: 'Act' }).className;
    const link = screen.getByRole('link', { name: 'Go' }).className.replace(' no-underline', '');
    expect(link).toBe(button);
  });

  it('renders a router link through renderLink with the same look and the content', () => {
    render(
      <ButtonLink
        variant="secondary"
        size="xs"
        renderLink={(props) => <a href="/routed" data-router="" {...props} />}
      >
        Routed
      </ButtonLink>,
    );
    const link = screen.getByRole('link', { name: 'Routed' });
    expect(link.getAttribute('data-router')).toBe('');
    expect(link.getAttribute('data-variant')).toBe('secondary');
    expect(link.className).toContain('min-h-6');
    expect(link.className).toContain('border-surface-border');
  });

  it('needs an aria-label when icon-only, and passes it through renderLink', () => {
    render(
      <ButtonLink
        iconOnly
        icon={<svg />}
        aria-label="Open survey"
        renderLink={(props) => <a href="/s" {...props} />}
      />,
    );
    const link = screen.getByRole('link', { name: 'Open survey' });
    expect(link.className).toContain('h-10');
    expect(link.className).toContain('w-10');
  });

  it('when disabled is not navigable: no href, out of the tab order, aria-disabled', () => {
    const onClick = vi.fn();
    render(
      <ButtonLink href="/x" disabled onClick={onClick}>
        Locked
      </ButtonLink>,
    );
    const link = screen.getByRole('link', { name: 'Locked' });
    expect(link.hasAttribute('href')).toBe(false);
    expect(link.getAttribute('aria-disabled')).toBe('true');
    expect(link.getAttribute('tabindex')).toBe('-1');
    expect(link.className).toContain('aria-disabled:cursor-not-allowed');
    fireEvent.click(link);
    expect(onClick).not.toHaveBeenCalled();
  });

  it('swallows the click of a disabled routed link', () => {
    const onClick = vi.fn();
    render(
      <ButtonLink
        disabled
        renderLink={(props) => (
          <a href="/x" {...props} data-testid="routed">
            {props.children}
          </a>
        )}
      >
        Locked
      </ButtonLink>,
    );
    const routed = screen.getByTestId('routed');
    routed.addEventListener('click', onClick);
    expect(fireEvent.click(routed)).toBe(false);
    expect(routed.getAttribute('aria-disabled')).toBe('true');
  });

  it('forwards anchor attributes and calls onClick when enabled', () => {
    const onClick = vi.fn();
    render(
      <ButtonLink href="/x" target="_blank" rel="noreferrer" onClick={onClick}>
        Out
      </ButtonLink>,
    );
    const link = screen.getByRole('link', { name: 'Out' });
    expect(link.getAttribute('target')).toBe('_blank');
    fireEvent.click(link);
    expect(onClick).toHaveBeenCalledOnce();
  });
});
