import { cleanup, fireEvent, render, screen, within } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { frameOf, table } from './DataTableTestUtils';

afterEach(cleanup);

describe('DataTable layout', () => {
  it('contains horizontal overflow inside the table frame at 390px', () => {
    render(table({ 'data-testid': 'dt' }));
    const frame = screen.getByTestId('dt');
    expect(frame.className).toContain('panel');
    expect(frame.className).toContain('min-w-0');
    expect(frame.className).toContain('overflow-hidden');
    const scroller = screen.getByRole('table').parentElement;
    expect(scroller?.className).toContain('overflow-x-auto');
    expect(scroller?.getAttribute('data-phone-width-exempt')).toBe('table-scroll');
  });

  describe('sticky header', () => {
    it('pins the header and bounds the scroller when maxHeight is set', () => {
      render(table({ maxHeight: 300 }));
      const scroller = screen.getByRole('table').parentElement;
      expect(scroller?.style.maxHeight).toBe('300px');
      expect(scroller?.className).toContain('overflow-y-auto');
      expect(screen.getAllByRole('rowgroup')[0]?.className).toContain('sticky');
    });

    it('does not pin or bound anything by default', () => {
      render(table());
      expect(screen.getAllByRole('rowgroup')[0]?.className).not.toContain('sticky');
      expect(screen.getByRole('table').parentElement?.style.maxHeight).toBe('');
    });
  });

  describe('virtualization', () => {
    const big = Array.from({ length: 500 }, (_, i) => ({ id: `${i}`, name: `d${i}`, ports: i }));
    const dataRows = () =>
      screen
        .getAllByRole('row')
        .filter((r) => r.hasAttribute('aria-rowindex') && r.getAttribute('aria-rowindex') !== '1');

    it('mounts only a window of rows and no pager', () => {
      render(table({ rows: big, virtualize: {}, maxHeight: 440 }));
      expect(dataRows().length).toBeLessThan(30);
      expect(screen.getByRole('table').getAttribute('aria-rowcount')).toBe('501');
      expect(screen.queryByRole('navigation', { name: 'Pages' })).toBeNull();
      expect(screen.getByText('d0')).toBeTruthy();
      expect(screen.queryByText('d400')).toBeNull();
    });

    it('slides the window with the scroll offset and pads the rest with spacers', () => {
      render(table({ rows: big, virtualize: {}, maxHeight: 440 }));
      const scroller = screen.getByRole('table').parentElement as HTMLElement;
      scroller.scrollTop = 44 * 300;
      fireEvent.scroll(scroller);
      expect(screen.queryByText('d0')).toBeNull();
      expect(screen.getByText('d300')).toBeTruthy();
      const [first] = dataRows();
      expect(first?.getAttribute('aria-rowindex')).toBe(String(295 + 2));
      const spacers = screen.getByRole('table').querySelectorAll('td[aria-hidden="true"]');
      expect(spacers).toHaveLength(2);
    });

    it('stays a plain paged table below the threshold', () => {
      render(table({ virtualize: { threshold: 100 }, pageSize: 2 }));
      expect(screen.getByRole('navigation', { name: 'Pages' })).toBeTruthy();
      expect(screen.getByRole('table').getAttribute('aria-rowcount')).toBeNull();
    });
  });
});

describe('DataTable containment and keyboard access', () => {
  it('makes the scroll frame a containing block, so sr-only text cannot widen the page', () => {
    render(table({ rowActions: (d) => <button type="button">{`Edit ${d.name}`}</button> }));
    expect(frameOf().className).toContain('relative');
    expect(frameOf().className).toContain('overflow-x-auto');
    const actionsHeader = screen.getByRole('columnheader', { name: 'Actions' });
    expect(actionsHeader.className).toContain('relative');
    expect(actionsHeader.querySelector('.sr-only')).toBeTruthy();
    expect(screen.getByRole('table').querySelector('caption')?.className).toContain('sr-only');
  });

  describe('when the frame scrolls', () => {
    function stubOverflow() {
      vi.spyOn(HTMLElement.prototype, 'scrollWidth', 'get').mockReturnValue(900);
      vi.spyOn(HTMLElement.prototype, 'clientWidth', 'get').mockReturnValue(358);
    }

    it('is a focusable, labelled group', () => {
      stubOverflow();
      render(table());
      const frame = screen.getByRole('group', { name: 'Devices' });
      expect(frame).toBe(frameOf());
      expect(frame.getAttribute('tabindex')).toBe('0');
      expect(frame.className).toContain('focus-visible:outline');
      vi.restoreAllMocks();
    });

    it('is focusable even when it also holds buttons', () => {
      stubOverflow();
      render(table({ rowActions: () => <button type="button">Edit</button> }));
      expect(frameOf().getAttribute('tabindex')).toBe('0');
      vi.restoreAllMocks();
    });
  });

  it('leaves the frame out of the tab order and unlabelled while nothing overflows', () => {
    render(table());
    expect(frameOf().hasAttribute('tabindex')).toBe(false);
    expect(screen.queryByRole('group')).toBeNull();
  });

  it('keeps sort buttons, checkboxes and expand toggles at least 24px', () => {
    render(
      table({
        selection: { selectedKeys: new Set(), onChange: () => {} },
        expandable: {
          render: () => 'x',
          columnLabel: 'Details',
          toggleLabel: (d) => `Show ${d.name}`,
        },
      }),
    );
    for (const button of within(screen.getByRole('columnheader', { name: /Name/ })).getAllByRole(
      'button',
    )) {
      expect(button.className).toContain('min-h-6');
      expect(button.className).toContain('min-w-6');
    }
    for (const box of screen.getAllByRole('checkbox')) {
      expect(box.className).toContain('h-6');
      expect(box.className).toContain('w-6');
    }
    expect(screen.getByRole('button', { name: 'Show core' }).className).toContain('h-6');
  });
});
