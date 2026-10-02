import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type { ReactNode } from 'react';
import { describe, expect, it, vi } from 'vitest';
import { useFocusTrap } from '../../hooks/useFocusTrap';
import { Button, IconButton } from './Button';
import { Tooltip } from './Tooltip';

function TrapDialog({ onEscape, children }: { onEscape: () => void; children: ReactNode }) {
  const ref = useFocusTrap<HTMLDivElement>({ isActive: true, onEscape });
  return (
    <div ref={ref} role="dialog" aria-modal="true">
      {children}
    </div>
  );
}

describe('Tooltip keyboard contract', () => {
  it('describes the actual focused trigger and dismisses with Escape', async () => {
    const user = userEvent.setup();
    render(
      <Tooltip text="Time spent resolving the hostname">
        <button type="button">DNS</button>
      </Tooltip>,
    );
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument();
    await user.tab();
    const trigger = screen.getByText('DNS');
    expect(trigger).toHaveFocus();
    expect(trigger).toHaveAccessibleDescription('Time spent resolving the hostname');
    expect(screen.getByRole('tooltip')).toBeVisible();
    await user.keyboard('{Escape}');
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument();
    expect(trigger).toHaveFocus();
  });

  it('dismisses a hover-only tooltip when Escape is pressed elsewhere', async () => {
    const user = userEvent.setup();
    render(
      <Tooltip text="Connection details">
        <button type="button">Status</button>
      </Tooltip>,
    );
    await user.hover(screen.getByRole('button', { name: 'Status' }));
    expect(screen.getByRole('tooltip')).toBeVisible();
    expect(screen.getByRole('button', { name: 'Status' })).not.toHaveFocus();
    await user.keyboard('{Escape}');
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument();
  });

  it('dismisses on activation so a newly opened drawer receives Escape', async () => {
    const user = userEvent.setup();
    const open = vi.fn();
    render(
      <Tooltip text="Help for this page">
        <button
          type="button"
          onClick={(event) => {
            event.currentTarget.focus();
            open();
          }}
        >
          Open help
        </button>
      </Tooltip>,
    );
    await user.hover(screen.getByRole('button', { name: 'Open help' }));
    await user.click(screen.getByRole('button', { name: 'Open help' }));
    expect(open).toHaveBeenCalledOnce();
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument();
  });

  // WebKit sends no mouseleave when a drawer opens over a still pointer, so a
  // hovered tooltip stays open beneath the dialog (#2893).
  it('leaves Escape to a modal dialog opened over a hovered tooltip', async () => {
    const user = userEvent.setup();
    const closeDialog = vi.fn();
    render(
      <>
        <Tooltip text="Select an Ethernet interface">
          <button type="button">Ethernet</button>
        </Tooltip>
        <TrapDialog onEscape={closeDialog}>
          <button type="button">Close help</button>
        </TrapDialog>
      </>,
    );
    await user.hover(screen.getByRole('button', { name: 'Ethernet' }));
    expect(screen.getByRole('tooltip')).toBeVisible();
    screen.getByRole('button', { name: 'Close help' }).focus();
    await user.keyboard('{Escape}');
    expect(closeDialog).toHaveBeenCalledOnce();
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument();
  });

  it('still takes Escape for a tooltip inside the open dialog', async () => {
    const user = userEvent.setup();
    const closeDialog = vi.fn();
    render(
      <TrapDialog onEscape={closeDialog}>
        <Tooltip text="Section contents">
          <button type="button">Contents</button>
        </Tooltip>
      </TrapDialog>,
    );
    await user.tab();
    expect(screen.getByRole('tooltip')).toBeVisible();
    await user.keyboard('{Escape}');
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument();
    expect(closeDialog).not.toHaveBeenCalled();
  });

  it('keeps focus and the accessible name when contextual text changes', async () => {
    const user = userEvent.setup();
    const { rerender } = render(
      <Tooltip text="More detail">
        <button type="button" aria-label="Settings">
          ⚙
        </button>
      </Tooltip>,
    );
    await user.tab();
    const trigger = screen.getByRole('button', { name: 'Settings' });
    rerender(
      <Tooltip>
        <button type="button" aria-label="Settings">
          ⚙
        </button>
      </Tooltip>,
    );
    expect(trigger).toHaveFocus();
    expect(screen.getByRole('button', { name: 'Settings' })).toBe(trigger);
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument();
  });

  it('makes a disabled action explanation reachable without enabling the action', async () => {
    const user = userEvent.setup();
    const action = vi.fn();
    render(
      <Tooltip text="An operator role is required">
        <button type="button" disabled={true} onClick={action}>
          Run test
        </button>
      </Tooltip>,
    );
    await user.tab();
    expect(document.activeElement).toHaveAccessibleDescription('An operator role is required');
    expect(screen.getByRole('tooltip')).toBeVisible();
    await user.keyboard('{Enter} ');
    expect(action).not.toHaveBeenCalled();
    expect(screen.getByRole('button', { name: 'Run test' })).toHaveAttribute(
      'aria-disabled',
      'true',
    );
  });
});

describe('Button tooltip forwarding', () => {
  it('uses the shared tooltip for a disabled Button title', async () => {
    const user = userEvent.setup();
    const action = vi.fn();
    render(
      <Button disabled={true} title="Requires an operator" onClick={action}>
        Scan
      </Button>,
    );
    await user.tab();
    expect(screen.getByRole('button', { name: 'Scan' })).toHaveFocus();
    expect(screen.getByRole('button', { name: 'Scan' })).toHaveAccessibleDescription(
      'Requires an operator',
    );
    expect(screen.getByRole('button', { name: 'Scan' })).not.toHaveAttribute('title');
    await user.keyboard('{Enter} ');
    expect(action).not.toHaveBeenCalled();
  });
  it('keeps the IconButton name independent from its description', async () => {
    const user = userEvent.setup();
    render(
      <IconButton
        icon={<span aria-hidden="true">+</span>}
        aria-label="Add target"
        title="Add a monitored device"
      />,
    );
    await user.tab();
    expect(screen.getByRole('button', { name: 'Add target' })).toHaveAccessibleDescription(
      'Add a monitored device',
    );
  });
});
