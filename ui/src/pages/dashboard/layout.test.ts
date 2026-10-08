import { describe, expect, it } from 'vitest';
import {
  availableWidgets,
  DASHBOARD_WIDGETS,
  type DashboardWidgetId,
  DEFAULT_DASHBOARD,
  moveWidget,
  resolveLayout,
} from './layout';

describe('resolveLayout', () => {
  it.each([
    { name: 'never saved', widgets: [], saved: false, want: [...DEFAULT_DASHBOARD] },
    { name: 'saved empty', widgets: [], saved: true, want: [] },
    { name: 'saved order', widgets: ['dns', 'link'], saved: true, want: ['dns', 'link'] },
    {
      name: 'drops ids the catalog no longer has',
      widgets: ['retired', 'gateway'],
      saved: true,
      want: ['gateway'],
    },
  ])('$name', ({ widgets, saved, want }) => {
    expect(resolveLayout(widgets, saved)).toEqual(want);
  });
});

describe('moveWidget', () => {
  const layout: DashboardWidgetId[] = ['link', 'gateway', 'dns'];

  it.each([
    { name: 'up', index: 1, delta: -1 as const, want: ['gateway', 'link', 'dns'] },
    { name: 'down', index: 1, delta: 1 as const, want: ['link', 'dns', 'gateway'] },
    { name: 'first up is a no-op', index: 0, delta: -1 as const, want: layout },
    { name: 'last down is a no-op', index: 2, delta: 1 as const, want: layout },
  ])('$name', ({ index, delta, want }) => {
    expect(moveWidget(layout, index, delta)).toEqual(want);
    expect(layout).toEqual(['link', 'gateway', 'dns']);
  });
});

describe('availableWidgets', () => {
  it('lists what is not on the layout, in catalog order', () => {
    expect(availableWidgets(['dns', 'link'])).toEqual(
      DASHBOARD_WIDGETS.filter((w) => w !== 'dns' && w !== 'link'),
    );
    expect(availableWidgets([...DASHBOARD_WIDGETS])).toEqual([]);
  });
});
