/**
 * theme_layout.ts — layout-related tokens: sizing constraints, icon sizing,
 * border-radius/border tokens, and flex/grid layout patterns. Re-exported
 * through theme.ts.
 */

/**
 * Sizing tokens - for consistent heights/widths
 * Avoids arbitrary values like h-[500px] or w-[28rem]
 */
export const sizing = {
  // Modal/drawer heights (viewport-relative)
  height: {
    modal: 'max-h-modal', // 85vh - defined in @theme
    drawer: 'h-full', // Full height for side drawers
    panel: 'max-h-[70vh]', // Shorter panels
  },

  // Fixed widths for drawers, panels
  width: {
    drawer: 'w-80', // 320px - standard drawer
    drawerWide: 'w-96', // 384px - wide drawer
    panel: 'w-72', // 288px - side panels
    dropdown: 'w-64', // 256px - dropdown menus
  },

  // Min/max constraints
  minHeight: {
    card: 'min-h-[120px]', // Minimum card height
    section: 'min-h-[200px]', // Minimum section height
  },
} as const;

/**
 * Icon sizes - standardized icon dimensions
 * Use these instead of arbitrary w-4 h-4, w-5 h-5, etc.
 */
export const icon = {
  // Size tokens (width and height combined)
  size: {
    xs: 'w-3 h-3', // 12px - inline with small text
    sm: 'w-4 h-4', // 16px - most common, inline with body text
    md: 'w-5 h-5', // 20px - buttons, list items
    lg: 'w-6 h-6', // 24px - card headers, prominent icons
    xl: 'w-8 h-8', // 32px - empty states, feature icons
    '2xl': 'w-12 h-12', // 48px - hero sections, large features
  },

  // Common icon + text patterns
  inline: 'inline-flex items-center gap-1.5', // Icon with inline text
  button: 'inline-flex items-center gap-2', // Icon in button
  leading: 'flex items-center gap-2', // Icon leading text
} as const;

/**
 * Border radius tokens - consistent rounding across components
 */
export const radius = {
  none: 'rounded-none',
  sm: 'rounded-sm', // 2px - subtle rounding
  default: 'rounded', // 4px - inputs, small elements
  md: 'rounded-md', // 6px - buttons
  lg: 'rounded-lg', // 8px - cards, panels
  xl: 'rounded-xl', // 12px - modals, large containers
  full: 'rounded-full', // Pills, badges, avatars
} as const;

/**
 * Border tokens - consistent border styling
 */
export const border = {
  // Border widths
  width: {
    none: 'border-0',
    default: 'border', // 1px
    thick: 'border-2', // 2px
  },

  // Border colors (using design tokens)
  color: {
    default: 'border-surface-border',
    focus: 'border-brand-primary',
    error: 'border-status-error',
    success: 'border-status-success',
    warning: 'border-status-warning',
  },

  // Common combinations
  card: 'border border-surface-border',
  input: 'border border-surface-border focus:border-brand-primary',
  divider: 'border-t border-surface-border',
} as const;

/**
 * Layout tokens - common flex/grid patterns
 * Use these instead of repeating flex items-center gap-2 everywhere
 */
export const layout = {
  // Flex utilities
  flex: {
    center: 'flex items-center justify-center',
    between: 'flex items-center justify-between',
    start: 'flex items-center justify-start',
    end: 'flex items-center justify-end',
    col: 'flex flex-col',
    colCenter: 'flex flex-col items-center justify-center',
    wrap: 'flex flex-wrap',
  },

  // Grid layouts
  grid: {
    // Responsive card grids - cards stretch to fill available space.
    // auto-fit rather than the fixed 1/2/3/4 columns: those snapped at the
    // viewport breakpoints regardless of how much room the content column
    // actually had, so a three-card page left a dead fourth column at 1440px
    // and every card was squeezed to 232px at 1280px, where card headers
    // painted outside their own border. A 16rem floor keeps four columns at
    // 1440px, drops to three at 1280px before anything is squeezed, and the
    // 1fr share means the last card in a row ends where the column does.
    // grid-flow-dense so a card that spans two columns does not strand the
    // single column beside it: without it the first row of /network ended
    // three cards wide with a dead fourth column at 1440px.
    cards: 'grid grid-flow-dense grid-cols-[repeat(auto-fit,minmax(16rem,1fr))] gap-6',
    cardsWide: 'grid grid-cols-1 lg:grid-cols-3 gap-6',
    // Form layouts
    form2col: 'grid grid-cols-2 gap-2',
    form4col: 'grid grid-cols-2 md:grid-cols-4 gap-4',
    // Settings/data layouts
    data2col: 'grid grid-cols-2 gap-x-4 gap-y-2',
    data3col: 'grid grid-cols-3 gap-2',
  },

  // Inline patterns (horizontal lists, tags, buttons)
  inline: {
    tight: 'flex items-center gap-1',
    default: 'flex items-center gap-2',
    comfortable: 'flex items-center gap-3',
    spacious: 'flex items-center gap-4',
    wrap: 'flex flex-wrap items-center gap-2',
  },

  // Stack patterns (vertical lists)
  stack: {
    tight: 'flex flex-col gap-1',
    default: 'flex flex-col gap-2',
    comfortable: 'flex flex-col gap-3',
    spacious: 'flex flex-col gap-4',
  },
} as const;
