import { act, cleanup, fireEvent, render, screen, within } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { RailFooter, type RailFooterProps, type RailState, railStateTone } from './RailFooter';

beforeEach(() => {
  vi.useFakeTimers();
});

afterEach(() => {
  cleanup();
  vi.useRealTimers();
});

function footer(overrides: Partial<RailFooterProps> = {}): RailFooterProps {
  return {
    collapsed: false,
    help: { label: 'Open help', onClick: vi.fn() },
    settings: { label: 'Open settings', onClick: vi.fn() },
    theme: { label: 'Switch to dark theme', isDark: false, onClick: vi.fn() },
    mark: { src: 'mark.svg', name: 'Mustard Seed Networks' },
    collapse: {
      collapseLabel: 'Collapse sidebar',
      expandLabel: 'Expand sidebar',
      onToggle: vi.fn(),
    },
    ...overrides,
  };
}

const full = (): RailFooterProps =>
  footer({
    status: { state: 'connected', label: 'Connected', detail: 'Reaching the daemon' },
    productSlot: {
      label: 'Profile',
      value: 'Lab',
      icon: <svg data-testid="slot-icon" />,
      options: [
        { value: 'lab', label: 'Lab' },
        { value: 'site', label: 'Site' },
      ],
      selected: 'lab',
      onSelect: vi.fn(),
    },
    signOut: { label: 'Sign out', onClick: vi.fn() },
    version: { value: 'v0.285.3-31-gf4194afb-dirty', label: 'Version' },
  });

const ids = () =>
  [...screen.getByTestId('rail-footer').querySelectorAll('[data-testid]')]
    .map((element) => element.getAttribute('data-testid'))
    .filter((id) => id?.startsWith('rail-') && !id.endsWith('-menu'));

describe('RailFooter order', () => {
  it('is status, product slot, actions, version, mark, collapse', () => {
    render(<RailFooter {...full()} />);
    expect(ids()).toEqual([
      'rail-status',
      'rail-product-slot',
      'rail-help',
      'rail-settings',
      'rail-theme',
      'rail-sign-out',
      'rail-version',
      'rail-mark',
      'rail-collapse',
    ]);
  });

  it('keeps the same order when the optional parts are absent', () => {
    render(<RailFooter {...footer()} />);
    expect(ids()).toEqual([
      'rail-help',
      'rail-settings',
      'rail-theme',
      'rail-mark',
      'rail-collapse',
    ]);
  });
});

describe('RailFooter actions', () => {
  it('are icon-only buttons named by aria-label, with no visible text', () => {
    render(<RailFooter {...full()} />);
    for (const [id, name] of [
      ['rail-help', 'Open help'],
      ['rail-settings', 'Open settings'],
      ['rail-theme', 'Switch to dark theme'],
      ['rail-sign-out', 'Sign out'],
    ] as const) {
      const button = screen.getByTestId(id);
      expect(button.tagName).toBe('BUTTON');
      expect(button.getAttribute('aria-label')).toBe(name);
      expect(button.textContent).toBe('');
      expect(screen.getByRole('button', { name })).toBe(button);
    }
  });

  it('call their handlers', () => {
    const props = full();
    render(<RailFooter {...props} />);
    for (const id of ['rail-help', 'rail-settings', 'rail-theme', 'rail-sign-out']) {
      fireEvent.click(screen.getByTestId(id));
    }
    expect(props.help.onClick).toHaveBeenCalledOnce();
    expect(props.settings.onClick).toHaveBeenCalledOnce();
    expect(props.theme.onClick).toHaveBeenCalledOnce();
    expect(props.signOut?.onClick).toHaveBeenCalledOnce();
  });

  it('show a tooltip on keyboard focus', () => {
    render(<RailFooter {...full()} />);
    fireEvent.focus(screen.getByTestId('rail-help'));
    expect(screen.getByRole('tooltip', { name: 'Open help' })).toBeTruthy();
  });

  it('render no sign-out button for a product without auth', () => {
    render(<RailFooter {...footer()} />);
    expect(screen.queryByTestId('rail-sign-out')).toBeNull();
  });

  it('show the sun in the dark theme and the moon in the light one', () => {
    const { rerender } = render(<RailFooter {...footer()} />);
    const icon = () => screen.getByTestId('rail-theme').querySelector('svg')?.getAttribute('class');
    const light = icon();
    rerender(
      <RailFooter
        {...footer({ theme: { label: 'Switch to light theme', isDark: true, onClick: vi.fn() } })}
      />,
    );
    expect(icon()).not.toBe(light);
  });
});

describe('RailFooter version', () => {
  it('shows "v<semver>" and keeps the commit and dirty flag out of the visible line', () => {
    render(<RailFooter {...full()} />);
    const version = screen.getByTestId('rail-version');
    expect(version.textContent).toBe('Version v0.285.3');
    expect(version.textContent).not.toMatch(/dirty|gf4194afb/);
  });

  it('puts the build detail in a tooltip, reachable by keyboard', () => {
    render(<RailFooter {...full()} />);
    const version = screen.getByTestId('rail-version');
    expect(version.getAttribute('tabindex')).toBe('0');
    fireEvent.focus(version);
    const tip = screen.getByRole('tooltip', { name: 'v0.285.3-31-gf4194afb-dirty' });
    expect(tip.className).toContain('visible');
    expect(tip.className).not.toContain('invisible');
  });

  it('is not focusable and has no tooltip for a clean release', () => {
    render(<RailFooter {...footer({ version: { value: 'v1.2.3', label: 'Version' } })} />);
    expect(screen.getByTestId('rail-version').hasAttribute('tabindex')).toBe(false);
    expect(screen.queryByRole('tooltip', { hidden: true, name: /v1\.2\.3/ })).toBeNull();
  });

  it('truncates rather than widening the rail', () => {
    render(<RailFooter {...full()} />);
    expect(screen.getByTestId('rail-version').className).toContain('truncate');
  });
});

describe('RailFooter status, mark and collapse', () => {
  it('shows the status as a pill with its label', () => {
    render(<RailFooter {...full()} />);
    const status = screen.getByRole('status');
    expect(within(status).getByText('Connected')).toBeTruthy();
  });

  it.each([
    ['connected', 'success'],
    ['reconnecting', 'info'],
    ['disconnected', 'error'],
    ['degraded', 'warning'],
  ] as const satisfies readonly (readonly [RailState, string])[])(
    'state %s is drawn in the %s tone, collapsed or not',
    (state, tone) => {
      expect(railStateTone[state]).toBe(tone);
      const status = { state, label: state, detail: `${state} detail` };
      const { rerender } = render(<RailFooter {...full()} status={status} />);
      expect(screen.getByTestId('rail-status').getAttribute('data-state')).toBe(state);
      rerender(<RailFooter {...full()} status={status} collapsed />);
      expect(screen.getByRole('img', { name: state }).getAttribute('data-tone')).toBe(tone);
    },
  );

  it('gives the status a tooltip and a screen-reader line with its detail', () => {
    render(<RailFooter {...full()} />);
    expect(within(screen.getByRole('status')).getByText('Reaching the daemon')).toBeTruthy();
    fireEvent.mouseEnter(screen.getByText('Connected'));
    expect(screen.getByRole('tooltip', { name: 'Reaching the daemon' })).toBeTruthy();
  });

  it('shows the wordmark when expanded', () => {
    render(<RailFooter {...full()} />);
    expect(within(screen.getByTestId('rail-mark')).getByText('Mustard Seed Networks')).toBeTruthy();
  });

  it('toggles from the collapse control, named for what it will do', () => {
    const props = full();
    const { rerender } = render(<RailFooter {...props} />);
    const button = screen.getByRole('button', { name: 'Collapse sidebar' });
    expect(button.getAttribute('aria-expanded')).toBe('true');
    fireEvent.click(button);
    expect(props.collapse.onToggle).toHaveBeenCalledOnce();
    rerender(<RailFooter {...props} collapsed />);
    expect(
      screen.getByRole('button', { name: 'Expand sidebar' }).getAttribute('aria-expanded'),
    ).toBe('false');
  });
});

describe('RailFooter collapsed', () => {
  it('is icons only: no wordmark, no slot text, no visible version label, and a dot for the status', () => {
    render(<RailFooter {...full()} collapsed />);
    expect(screen.queryByText('Mustard Seed Networks')).toBeNull();
    expect(screen.getByTestId('rail-footer').getAttribute('data-collapsed')).toBe('true');
    const slot = screen.getByTestId('rail-product-slot');
    expect(slot.textContent).toBe('');
    expect(slot.getAttribute('aria-label')).toBe('Profile: Lab');
    expect(screen.getByRole('img', { name: 'Connected' }).getAttribute('data-tone')).toBe(
      'success',
    );
  });

  it('keeps every action reachable by name and stacks them in a column', () => {
    render(<RailFooter {...full()} collapsed />);
    for (const name of ['Open help', 'Open settings', 'Switch to dark theme', 'Sign out']) {
      expect(screen.getByRole('button', { name })).toBeTruthy();
    }
    expect(screen.getByTestId('rail-help').parentElement?.parentElement?.className).toContain(
      'flex-col',
    );
    expect(screen.getByTestId('rail-footer').className).toContain('px-2');
  });

  it('gives the collapsed slot a tooltip with its label and value', () => {
    render(<RailFooter {...full()} collapsed />);
    fireEvent.focus(screen.getByTestId('rail-product-slot'));
    expect(screen.getByRole('tooltip', { name: 'Profile: Lab' })).toBeTruthy();
  });
});

describe('RailFooter in the 390px drawer', () => {
  it('is expanded in the same order and has no collapse control, even when asked to collapse', () => {
    render(<RailFooter {...full()} variant="drawer" collapsed />);
    expect(screen.queryByTestId('rail-collapse')).toBeNull();
    expect(ids()).toEqual([
      'rail-status',
      'rail-product-slot',
      'rail-help',
      'rail-settings',
      'rail-theme',
      'rail-sign-out',
      'rail-version',
      'rail-mark',
    ]);
    expect(screen.getByText('Mustard Seed Networks')).toBeTruthy();
  });

  it('cannot widen the drawer: every row is min-w-0, the actions share the row, long text truncates', () => {
    render(<RailFooter {...full()} variant="drawer" />);
    expect(screen.getByTestId('rail-footer').className).toContain('min-w-0');
    expect(screen.getByTestId('rail-help').className).toContain('grow');
    expect(screen.getByTestId('rail-help').className).toContain('min-w-10');
    expect(screen.getByTestId('rail-product-slot').className).toContain('min-w-0');
    expect(screen.getByText('Mustard Seed Networks').className).toContain('truncate');
  });

  it('keeps every action target at least 40px tall', () => {
    render(<RailFooter {...full()} variant="drawer" />);
    expect(screen.getByTestId('rail-help').className).toContain('h-10');
    expect(screen.getByTestId('rail-product-slot').className).toContain('min-h-11');
  });
});

describe('RailFooter product slot as a button', () => {
  it('is one plain button that shows label and value', () => {
    const onClick = vi.fn();
    render(
      <RailFooter
        {...footer({ productSlot: { label: 'Session', value: 'default', icon: <svg />, onClick } })}
      />,
    );
    const slot = screen.getByTestId('rail-product-slot');
    expect(slot.getAttribute('aria-haspopup')).toBeNull();
    expect(slot.textContent).toBe('Sessiondefault');
    fireEvent.click(slot);
    expect(onClick).toHaveBeenCalledOnce();
  });
});

describe('RailFooter product slot as a menu', () => {
  const open = () => {
    const props = full();
    render(<RailFooter {...props} />);
    const trigger = screen.getByTestId('rail-product-slot');
    fireEvent.click(trigger);
    return { props, trigger };
  };

  it('is a menu button that opens a radio menu with the current option checked and focused', () => {
    const { trigger } = open();
    expect(trigger.getAttribute('aria-haspopup')).toBe('menu');
    expect(trigger.getAttribute('aria-expanded')).toBe('true');
    const items = screen.getAllByRole('menuitemradio');
    expect(items.map((item) => item.textContent)).toEqual(['Lab', 'Site']);
    expect(items.map((item) => item.getAttribute('aria-checked'))).toEqual(['true', 'false']);
    expect(document.activeElement).toBe(items[0]);
    expect(screen.getByRole('menu', { name: 'Profile' })).toBeTruthy();
  });

  it('moves with the arrow keys, wrapping, and with Home and End', () => {
    open();
    const menu = screen.getByRole('menu');
    const items = screen.getAllByRole('menuitemradio');
    fireEvent.keyDown(menu, { key: 'ArrowDown' });
    expect(document.activeElement).toBe(items[1]);
    fireEvent.keyDown(menu, { key: 'ArrowDown' });
    expect(document.activeElement).toBe(items[0]);
    fireEvent.keyDown(menu, { key: 'ArrowUp' });
    expect(document.activeElement).toBe(items[1]);
    fireEvent.keyDown(menu, { key: 'Home' });
    expect(document.activeElement).toBe(items[0]);
    fireEvent.keyDown(menu, { key: 'End' });
    expect(document.activeElement).toBe(items[1]);
  });

  it('chooses an option, closes and returns focus to the trigger', () => {
    const { props, trigger } = open();
    fireEvent.click(screen.getAllByRole('menuitemradio')[1] as HTMLElement);
    expect(
      props.productSlot && 'onSelect' in props.productSlot && props.productSlot.onSelect,
    ).toHaveBeenCalledWith('site');
    expect(screen.queryByRole('menu')).toBeNull();
    expect(document.activeElement).toBe(trigger);
  });

  it('closes on Escape, returning focus, without reaching a dialog behind it', () => {
    const { trigger } = open();
    const behind = vi.fn();
    document.addEventListener('keydown', behind);
    fireEvent.keyDown(screen.getByRole('menu'), { key: 'Escape' });
    document.removeEventListener('keydown', behind);
    expect(screen.queryByRole('menu')).toBeNull();
    expect(document.activeElement).toBe(trigger);
    expect(behind).not.toHaveBeenCalled();
  });

  it('closes on a click elsewhere and on Tab', () => {
    open();
    fireEvent.mouseDown(document.body);
    expect(screen.queryByRole('menu')).toBeNull();
    fireEvent.click(screen.getByTestId('rail-product-slot'));
    fireEvent.keyDown(screen.getByRole('menu'), { key: 'Tab' });
    expect(screen.queryByRole('menu')).toBeNull();
  });

  it('opens from the keyboard with ArrowDown on the trigger', () => {
    render(<RailFooter {...full()} />);
    fireEvent.keyDown(screen.getByTestId('rail-product-slot'), { key: 'ArrowDown' });
    expect(screen.getByRole('menu')).toBeTruthy();
  });

  it('opens beside the rail when collapsed, above the trigger when expanded', () => {
    const props = full();
    const { rerender } = render(<RailFooter {...props} />);
    fireEvent.click(screen.getByTestId('rail-product-slot'));
    expect(screen.getByRole('menu').className).toContain('bottom-full');
    rerender(<RailFooter {...props} collapsed />);
    expect(screen.getByRole('menu').className).toContain('left-full');
  });

  it('flushes pending timers without a stray update', () => {
    open();
    act(() => {
      vi.runAllTimers();
    });
    expect(screen.getByRole('menu')).toBeTruthy();
  });
});

describe('RailFooter touch targets', () => {
  const px = (classes: string[], prefix: string) => {
    const token = classes.find((name) => name.startsWith(`${prefix}-`));
    return token ? Number(token.slice(prefix.length + 1)) * 4 : 0;
  };
  const actionIds = ['rail-help', 'rail-settings', 'rail-theme', 'rail-sign-out', 'rail-collapse'];

  it.each([
    ['collapsed rail', { collapsed: true }],
    ['expanded rail', { collapsed: false }],
    ['drawer', { collapsed: false, variant: 'drawer' as const }],
  ])('keeps every action at least 24px high and 40px for the icon buttons: %s', (_, props) => {
    render(<RailFooter {...full()} {...props} />);
    for (const id of actionIds) {
      const button = screen.queryByTestId(id);
      if (!button) {
        expect(id).toBe('rail-collapse');
        continue;
      }
      const classes = [...button.classList];
      expect(px(classes, 'h'), id).toBeGreaterThanOrEqual(24);
      if (id !== 'rail-collapse') {
        expect(px(classes, 'h'), id).toBe(40);
        expect(px(classes, 'min-h'), id).toBe(40);
        expect(px(classes, 'min-w'), id).toBe(40);
        expect(classes, id).toContain('shrink-0');
      }
    }
  });

  it('does not use flex-1, whose zero basis collapses h-10 in the collapsed column', () => {
    render(<RailFooter {...full()} collapsed />);
    for (const id of actionIds) {
      expect([...screen.getByTestId(id).classList], id).not.toContain('flex-1');
    }
  });
});
