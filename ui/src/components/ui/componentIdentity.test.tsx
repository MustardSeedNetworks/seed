/**
 * Render work skipped by the React Compiler rather than by memo()
 * (UI-SEED-41 slice 6, #3066).
 *
 * A compiled component re-rendered with unchanged props returns the
 * elements and derived values it cached, so its children and formatters
 * are not called again. Each case fails when the compiler is removed from
 * the vitest config, and when @babel/core is 8: compiler 1.0.0 cannot
 * lower a defaulted destructured prop (`disabled = false`) there and
 * skips the whole component.
 */

import { render } from '@testing-library/react';
import type { ReactNode } from 'react';
import { describe, expect, it, vi } from 'vitest';

import { InterfaceSelector } from './InterfaceSelector';
import { Slider } from './Slider';
import { HealthScoreBadge } from './Sparkline';

const { tooltipRenders } = vi.hoisted(() => ({ tooltipRenders: vi.fn() }));
vi.mock('./Tooltip', () => ({
  Tooltip: ({ children }: { children: ReactNode }): ReactNode => {
    tooltipRenders();
    return children;
  },
}));

describe('compiled components skip work on an unchanged re-render', () => {
  it('Slider formats its value once', () => {
    const formatValue = vi.fn((v: number) => `${v} ms`);
    const onChange = vi.fn();
    const { rerender } = render(
      <Slider value={5} onChange={onChange} min={0} max={10} step={1} formatValue={formatValue} />,
    );
    rerender(
      <Slider value={5} onChange={onChange} min={0} max={10} step={1} formatValue={formatValue} />,
    );
    expect(formatValue).toHaveBeenCalledTimes(1);
  });

  it('HealthScoreBadge does not re-render its tooltip', () => {
    tooltipRenders.mockClear();
    const { rerender } = render(<HealthScoreBadge score={92} />);
    const first = tooltipRenders.mock.calls.length;
    rerender(<HealthScoreBadge score={92} />);
    expect(first).toBeGreaterThan(0);
    expect(tooltipRenders).toHaveBeenCalledTimes(first);
  });

  it('InterfaceSelector does not re-render its tooltips', () => {
    tooltipRenders.mockClear();
    const onChange = vi.fn();
    const onAccept = vi.fn();
    const props = {
      interfaces: [],
      currentInterface: 'eth0',
      isWifi: false,
      onChange,
      warning: 'eth0 is down',
      suggestedInterface: 'eth1',
      onAcceptSuggestion: onAccept,
    };
    const { rerender } = render(<InterfaceSelector {...props} />);
    const first = tooltipRenders.mock.calls.length;
    rerender(<InterfaceSelector {...props} />);
    expect(first).toBeGreaterThan(0);
    expect(tooltipRenders).toHaveBeenCalledTimes(first);
  });
});
