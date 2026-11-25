import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { Breadcrumbs } from './Breadcrumbs';

afterEach(cleanup);

const items = [
  { label: 'Home', href: '/' },
  { label: 'Devices', href: '/devices' },
  { label: 'Router' },
];

describe('Breadcrumbs', () => {
  it('is a named navigation landmark holding an ordered list', () => {
    render(<Breadcrumbs items={items} label="Breadcrumb" />);
    const nav = screen.getByRole('navigation', { name: 'Breadcrumb' });
    expect(nav.querySelectorAll('ol > li')).toHaveLength(3);
  });

  it('links every item but the last, which is the current page', () => {
    render(<Breadcrumbs items={items} label="Breadcrumb" />);
    expect(screen.getAllByRole('link').map((link) => link.getAttribute('href'))).toEqual([
      '/',
      '/devices',
    ]);
    expect(screen.getByText('Router').getAttribute('aria-current')).toBe('page');
  });

  it('treats a last item with an href as the current page too', () => {
    render(
      <Breadcrumbs
        items={[
          { label: 'Home', href: '/' },
          { label: 'Here', href: '/here' },
        ]}
        label="b"
      />,
    );
    expect(screen.getAllByRole('link')).toHaveLength(1);
  });

  it('hands link rendering to the caller so any router can supply its own Link', () => {
    render(
      <Breadcrumbs
        items={items}
        label="Breadcrumb"
        renderLink={({ href, className, children }) => (
          <a href={href} className={className} data-router-link="">
            {children}
          </a>
        )}
      />,
    );
    expect(document.querySelectorAll('[data-router-link]')).toHaveLength(2);
  });

  it('wraps and truncates so a long trail never leaves a 390px viewport', () => {
    render(<Breadcrumbs items={items} label="Breadcrumb" />);
    expect(document.querySelector('ol')?.className).toContain('flex-wrap');
    expect(screen.getByText('Router').className).toContain('truncate');
  });
});
