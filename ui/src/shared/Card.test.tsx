import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { Card, CardDivider, CardRow, CardValue } from './Card';

afterEach(cleanup);

describe('Card', () => {
  it('takes its radius and border from the shared panel class, and is flat', () => {
    render(<Card data-testid="card">body</Card>);
    const { className } = screen.getByTestId('card');
    expect(className).toContain('panel');
    expect(className).not.toMatch(/rounded|shadow|border|bg-/);
  });

  it('renders the title as an h2, never the page h1', () => {
    render(
      <Card title="Link" headerAction={<span>pill</span>}>
        body
      </Card>,
    );
    expect(screen.getByRole('heading', { level: 2, name: 'Link' })).toBeTruthy();
    expect(screen.queryByRole('heading', { level: 1 })).toBeNull();
    expect(screen.getByText('pill')).toBeTruthy();
  });

  it('has no header row without a title or action', () => {
    render(<Card data-testid="card">body</Card>);
    expect(screen.getByTestId('card').querySelector('h2')).toBeNull();
  });

  it('can shrink inside a flex or grid parent so it never widens the page', () => {
    render(<Card data-testid="card">body</Card>);
    expect(screen.getByTestId('card').className).toContain('min-w-0');
  });
});

describe('Card header', () => {
  it('renders title, subtitle, a decorative icon, the action and a named status indicator', () => {
    render(
      <Card
        title="Link status"
        subtitle="eth0"
        icon={<svg data-testid="icon" />}
        headerAction={<button type="button">Details</button>}
        status="success"
        statusLabel="Success"
      >
        body
      </Card>,
    );
    expect(screen.getByRole('heading', { name: 'Link status' })).toBeTruthy();
    expect(screen.getByText('eth0')).toBeTruthy();
    expect(screen.getByTestId('icon').parentElement?.getAttribute('aria-hidden')).toBe('true');
    expect(screen.getByRole('button', { name: 'Details' })).toBeTruthy();
    expect(screen.getByRole('img', { name: 'Success' }).getAttribute('data-tone')).toBe('success');
  });

  it('requires a statusLabel with a status, by type', () => {
    // @ts-expect-error a status without its label would be an unnamed image
    render(<Card status="success">x</Card>);
    expect(screen.getByRole('img', { name: '' })).toBeTruthy();
  });

  it.each([2, 3, 4] as const)('renders the title at heading level %i', (level) => {
    render(
      <Card title="T" headingLevel={level}>
        x
      </Card>,
    );
    expect(screen.getByRole('heading', { level, name: 'T' })).toBeTruthy();
  });

  it('draws a header for a status alone and for a subtitle alone', () => {
    const { rerender } = render(
      <Card status="warning" statusLabel="Warning" data-testid="c">
        x
      </Card>,
    );
    expect(screen.getByRole('img', { name: 'Warning' })).toBeTruthy();
    rerender(
      <Card subtitle="only" data-testid="c">
        x
      </Card>,
    );
    expect(screen.getByText('only')).toBeTruthy();
  });
});

describe('Card tone and padding', () => {
  it.each([
    ['success', 'border-l-status-success'],
    ['warning', 'border-l-status-warning'],
    ['error', 'border-l-status-error'],
    ['info', 'border-l-status-info'],
  ] as const)('tone %s draws a left bar in %s', (tone, bar) => {
    render(
      <Card tone={tone} data-testid="c">
        x
      </Card>,
    );
    expect(screen.getByTestId('c').className).toContain(bar);
  });

  it('draws no bar for an unknown tone', () => {
    render(
      <Card tone="unknown" data-testid="c">
        x
      </Card>,
    );
    expect(screen.getByTestId('c').className).not.toContain('border-l');
  });

  it('is flat: no shadow, gradient or blur', () => {
    render(
      <Card tone="error" onClick={() => {}} data-testid="c">
        x
      </Card>,
    );
    expect(screen.getByTestId('c').className).not.toMatch(/shadow|gradient|backdrop|blur/);
  });

  it.each([
    ['sm', 'p-3'],
    ['md', 'p-4'],
    ['lg', 'p-6'],
  ] as const)('padding %s is %s', (padding, cls) => {
    render(
      <Card padding={padding} data-testid="c">
        x
      </Card>,
    );
    expect(screen.getByTestId('c').className).toContain(cls);
  });

  it('padding none leaves the body to its content but keeps the header inset', () => {
    render(
      <Card padding="none" title="T" data-testid="c">
        <div data-testid="body">body</div>
      </Card>,
    );
    expect(screen.getByTestId('c').className).not.toMatch(/(^|\s)p-\d/);
    expect(screen.getByRole('heading').closest('div[class*="px-4"]')).toBeTruthy();
  });

  it('never emits two utilities for one property: the header margin is exclusive with its inset', () => {
    const { rerender } = render(
      <Card title="T" data-testid="c">
        x
      </Card>,
    );
    const header = () => screen.getByRole('heading').closest('div[class*="flex-wrap"]');
    expect(header()?.className).toContain('mb-3');
    expect(header()?.className).not.toContain('px-4');
    rerender(
      <Card title="T" padding="none" data-testid="c">
        x
      </Card>,
    );
    expect(header()?.className).not.toContain('mb-');
    expect(header()?.className).toContain('px-4');
  });
});

describe('Card body states', () => {
  it('loading shows skeletons and aria-busy in place of the body', () => {
    render(
      <Card loading data-testid="c">
        <p>real</p>
      </Card>,
    );
    expect(screen.getByTestId('c').getAttribute('aria-busy')).toBe('true');
    expect(screen.queryByText('real')).toBeNull();
    expect(screen.getByTestId('c').querySelectorAll('[aria-hidden="true"]').length).toBeGreaterThan(
      0,
    );
  });

  it('error shows the message as an alert in place of the body', () => {
    render(<Card error="Probe failed">real</Card>);
    expect(screen.getByRole('alert').textContent).toBe('Probe failed');
    expect(screen.queryByText('real')).toBeNull();
  });

  it('empty shows a neutral message in place of the body', () => {
    render(<Card empty="No data yet">real</Card>);
    expect(screen.getByText('No data yet').className).toContain('text-text-secondary');
    expect(screen.queryByText('real')).toBeNull();
  });

  it('prefers loading, then error, then empty', () => {
    const { rerender } = render(
      <Card loading error="e" empty="m" data-testid="c">
        real
      </Card>,
    );
    expect(screen.queryByRole('alert')).toBeNull();
    rerender(
      <Card error="e" empty="m">
        real
      </Card>,
    );
    expect(screen.getByRole('alert')).toBeTruthy();
    expect(screen.queryByText('m')).toBeNull();
  });
});

describe('Card interaction and live region', () => {
  it('becomes clickable through a real button, not a role on a section', () => {
    const onClick = vi.fn();
    render(
      <Card onClick={onClick} title="Open me" data-testid="c">
        x
      </Card>,
    );
    const button = screen.getByRole('button', { name: 'Open me' });
    expect(button.tagName).toBe('BUTTON');
    expect(button.getAttribute('type')).toBe('button');
    fireEvent.click(button);
    expect(onClick).toHaveBeenCalledOnce();
    const card = screen.getByTestId('c');
    expect(card.tagName).toBe('DIV');
    expect(card.hasAttribute('role')).toBe(false);
    expect(card.hasAttribute('tabindex')).toBe(false);
    expect(card.querySelector('[role="button"]')).toBeNull();
  });

  it('names the button by aria-label when given, else by the title', () => {
    const { rerender } = render(
      <Card onClick={() => {}} title="Title" aria-label="Open the device">
        x
      </Card>,
    );
    expect(screen.getByRole('button', { name: 'Open the device' })).toBeTruthy();
    rerender(
      <Card onClick={() => {}} title="Title">
        x
      </Card>,
    );
    expect(screen.getByRole('button', { name: 'Title' })).toBeTruthy();
  });

  it('keeps headings and rows valid: nothing is nested inside the button', () => {
    render(
      <Card onClick={() => {}} title="Open me">
        <CardRow label="A" value="1" />
      </Card>,
    );
    const button = screen.getByRole('button', { name: 'Open me' });
    expect(button.children).toHaveLength(0);
    expect(screen.getByRole('heading', { name: 'Open me' })).toBeTruthy();
  });

  it('does not steal Enter from a control inside it', () => {
    const onClick = vi.fn();
    render(
      <Card onClick={onClick} headerAction={<button type="button">Inner</button>}>
        x
      </Card>,
    );
    fireEvent.keyDown(screen.getByRole('button', { name: 'Inner' }), { key: 'Enter' });
    expect(onClick).not.toHaveBeenCalled();
  });

  it('is not a button without onClick', () => {
    render(<Card data-testid="c">x</Card>);
    expect(screen.queryByRole('button')).toBeNull();
    expect(screen.getByTestId('c').hasAttribute('tabindex')).toBe(false);
  });

  it('announces changes politely when it is a live region, and takes an aria-label and id', () => {
    render(
      <Card liveRegion aria-label="Link" id="link-card" data-testid="c">
        x
      </Card>,
    );
    const card = screen.getByTestId('c');
    expect(card.getAttribute('aria-live')).toBe('polite');
    expect(card.getAttribute('aria-atomic')).toBe('true');
    expect(card.getAttribute('aria-label')).toBe('Link');
    expect(card.id).toBe('link-card');
  });
});

describe('CardRow, CardValue and CardDivider', () => {
  it('lays out a label and a value, truncating with the full text on hover', () => {
    render(<CardRow label="Gateway" value="192.168.100.254" />);
    const value = screen.getByTestId('card-row-value');
    expect(value.className).toContain('truncate');
    expect(value.getAttribute('title')).toBe('192.168.100.254');
    expect(screen.getByText('Gateway').className).toContain('shrink-0');
  });

  it('wraps a long value on request, with no title', () => {
    render(<CardRow label="Path" value="/a/very/long/path" wrap />);
    const value = screen.getByTestId('card-row-value');
    expect(value.className).toContain('break-all');
    expect(value.className).not.toContain('truncate');
    expect(value.hasAttribute('title')).toBe(false);
  });

  it('shows a status as glyph and the -strong text colour, never colour alone', () => {
    render(<CardRow label="Link" value="Down" status="error" />);
    const value = screen.getByTestId('card-row-value');
    expect(value.className).toContain('text-status-error-strong');
    expect(value.querySelector('svg')?.getAttribute('aria-hidden')).toBe('true');
  });

  it('takes a ReactNode value, a mono figure and left alignment', () => {
    render(<CardRow label="Speed" value={<b data-testid="rich">1G</b>} mono align="left" />);
    const value = screen.getByTestId('card-row-value');
    expect(screen.getByTestId('rich')).toBeTruthy();
    expect(value.className).toContain('figure');
    expect(value.className).toContain('justify-start');
    expect(value.hasAttribute('title')).toBe(false);
  });

  it('renders a value with a unit, size, status and wrap', () => {
    const { rerender } = render(<CardValue label="Latency" value={12} unit="ms" size="lg" mono />);
    expect(screen.getByText('Latency')).toBeTruthy();
    expect(screen.getByText('ms')).toBeTruthy();
    const value = screen.getByTestId('card-value');
    expect(value.className).toContain('text-xl');
    expect(value.className).toContain('figure');
    rerender(<CardValue value="Up" status="success" allowWrap />);
    const status = screen.getByTestId('card-value');
    expect(status.className).toContain('text-status-success-strong');
    expect(status.className).toContain('break-all');
    expect(status.querySelector('svg')).toBeTruthy();
  });

  it('draws a hairline divider that keeps its own spacing', () => {
    const { container } = render(<CardDivider className="my-1" />);
    expect(container.querySelector('hr')?.className).toContain('border-hairline');
  });
});
