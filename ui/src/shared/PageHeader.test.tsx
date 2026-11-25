import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { PageHeader, type PageHeaderProps } from './PageHeader';

afterEach(cleanup);

const base: PageHeaderProps = {
  kicker: 'Test module',
  title: 'Reflector',
  breadcrumbs: [{ label: 'Home', href: '/' }, { label: 'Reflector' }],
  breadcrumbsLabel: 'Breadcrumb',
};

describe('PageHeader', () => {
  it('renders the kicker, breadcrumbs and a single h1 title', () => {
    render(<PageHeader {...base} />);
    expect(screen.getByTestId('page-header-eyebrow').textContent).toBe('Test module');
    expect(screen.getByRole('navigation', { name: 'Breadcrumb' })).toBeTruthy();
    const headings = screen.getAllByRole('heading', { level: 1 });
    expect(headings).toHaveLength(1);
    expect(headings[0]?.textContent).toBe('Reflector');
  });

  it('keeps the fleet page-title scale and the three-line 76px measure', () => {
    render(<PageHeader {...base} subtitle="Bounces frames back" />);
    const title = screen.getByTestId('page-header-title').className;
    expect(title).toContain('text-xl');
    expect(title).toContain('sm:text-2xl');
    expect(title).toContain('leading-tight');
    expect(screen.getByTestId('page-header-eyebrow').className).toContain('kicker');
    expect(screen.getByTestId('page-header-description').className).toContain('text-sm');
  });

  it('puts the subtitle and readout in only when given', () => {
    render(<PageHeader {...base} />);
    expect(screen.queryByTestId('page-header-description')).toBeNull();
    expect(screen.queryByTestId('page-header-secondary')).toBeNull();
  });

  it('shows the readout and actions in a group that wraps under the title at 390px', () => {
    render(
      <PageHeader {...base} readout="64 targets" actions={<button type="button">Start</button>} />,
    );
    expect(screen.getByTestId('page-header-secondary').textContent).toBe('64 targets');
    const group = screen.getByRole('button', { name: 'Start' }).parentElement;
    expect(group?.className).toContain('flex-wrap');
    expect(group?.parentElement?.className).toContain('flex-wrap');
  });

  it('renders the help button only when asked, inside the title row, and calls back', () => {
    const onClick = vi.fn();
    const { rerender } = render(<PageHeader {...base} />);
    expect(screen.queryByTestId('page-header-help-button')).toBeNull();

    rerender(<PageHeader {...base} help={{ label: 'Help for Reflector', onClick }} />);
    const button = screen.getByRole('button', { name: 'Help for Reflector' });
    expect(button.parentElement).toBe(screen.getByTestId('page-header-title').parentElement);
    expect(button.className).toContain('shrink-0');
    fireEvent.click(button);
    expect(onClick).toHaveBeenCalledOnce();
  });

  it('exposes the title as a programmatic focus target for route changes', () => {
    render(<PageHeader {...base} />);
    expect(screen.getByTestId('page-header-title').getAttribute('tabindex')).toBe('-1');
  });

  it('forwards a router link renderer to the breadcrumbs', () => {
    render(
      <PageHeader
        {...base}
        renderBreadcrumbLink={({ href, className, children }) => (
          <a href={href} className={className} data-router-link="">
            {children}
          </a>
        )}
      />,
    );
    expect(document.querySelectorAll('[data-router-link]')).toHaveLength(1);
  });
});
