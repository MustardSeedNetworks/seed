/**
 * The dashboard's widget catalog and the edits a layout goes through
 * (UI-SEED-22). A layout is an ordered list of widget ids; the server stores
 * it per user and checks only its shape, so the catalog is the UI's.
 */

export const DASHBOARD_WIDGETS = [
  'link',
  'network',
  'gateway',
  'dns',
  'publicIp',
  'switch',
  'neighbours',
  'driverStats',
] as const;

export type DashboardWidgetId = (typeof DASHBOARD_WIDGETS)[number];

/** What a user sees before they save a layout of their own. */
export const DEFAULT_DASHBOARD: readonly DashboardWidgetId[] = [
  'link',
  'gateway',
  'dns',
  'network',
];

export function isDashboardWidget(id: string): id is DashboardWidgetId {
  return DASHBOARD_WIDGETS.some((w) => w === id);
}

/**
 * The widgets to show for a stored layout: the default when the user never
 * saved one, otherwise their ids in order, skipping any the catalog no longer
 * has.
 */
export function resolveLayout(widgets: readonly string[], saved: boolean): DashboardWidgetId[] {
  return saved ? widgets.filter(isDashboardWidget) : [...DEFAULT_DASHBOARD];
}

/** Moves the widget at index one place up (-1) or down (+1); a no-op at either end. */
export function moveWidget(
  layout: readonly DashboardWidgetId[],
  index: number,
  delta: -1 | 1,
): DashboardWidgetId[] {
  const target = index + delta;
  const next = [...layout];
  const moving = next[index];
  const displaced = next[target];
  if (moving === undefined || displaced === undefined) {
    return next;
  }
  next[target] = moving;
  next[index] = displaced;
  return next;
}

/** The catalog widgets not already on the layout, in catalog order. */
export function availableWidgets(layout: readonly DashboardWidgetId[]): DashboardWidgetId[] {
  return DASHBOARD_WIDGETS.filter((w) => !layout.includes(w));
}
