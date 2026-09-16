import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import { Button, IconButton } from './Button';
import { Tooltip } from './tooltip';

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
