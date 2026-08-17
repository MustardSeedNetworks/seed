import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { StatusRollup } from './StatusRollup';

describe('StatusRollup', () => {
  it('exposes its state so the theme gate and the page can both read it', () => {
    const { container } = render(<StatusRollup state="warn" headline="Loss above threshold" />);
    expect(container.querySelector('[data-state="warn"]')).not.toBeNull();
  });

  it('prints figures when it has data', () => {
    render(
      <StatusRollup
        state="ok"
        headline="Nothing is wrong right now"
        figures={[{ label: 'Frames', value: '1.2M' }]}
      />,
    );
    expect(screen.getByText('1.2M')).toBeInTheDocument();
  });

  /**
   * The one bug this component exists to prevent. A rollup that renders zeros
   * when its source failed says "nothing is wrong" at the moment nobody can
   * tell, so the unknown state must not be able to show a figure even when a
   * caller passes one.
   */
  it('refuses to print figures it does not have', () => {
    render(
      <StatusRollup
        state="unknown"
        headline="Health data is not arriving"
        figures={[
          { label: 'Frames', value: '0' },
          { label: 'Loss', value: '0%' },
        ]}
      />,
    );
    expect(screen.queryByText('0')).not.toBeInTheDocument();
    expect(screen.queryByText('0%')).not.toBeInTheDocument();
    expect(screen.getAllByText('—')).toHaveLength(2);
    // Labels survive: the reader still learns what is missing.
    expect(screen.getByText('Frames')).toBeInTheDocument();
  });

  it('never reports unknown as ok', () => {
    const { container } = render(
      <StatusRollup state="unknown" headline="Health data is not arriving" />,
    );
    expect(container.querySelector('[data-state="ok"]')).toBeNull();
    expect(screen.getByText(/not arriving/i)).toBeInTheDocument();
  });

  it('shows at most four figures, because more stops being a summary', () => {
    render(
      <StatusRollup
        state="ok"
        headline="Nothing is wrong right now"
        figures={[
          { label: 'A', value: '1' },
          { label: 'B', value: '2' },
          { label: 'C', value: '3' },
          { label: 'D', value: '4' },
          { label: 'E', value: '5' },
        ]}
      />,
    );
    expect(screen.queryByText('E')).not.toBeInTheDocument();
    expect(screen.getByText('D')).toBeInTheDocument();
  });
});
