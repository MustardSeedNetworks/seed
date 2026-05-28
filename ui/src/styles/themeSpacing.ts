/**
 * theme_spacing.ts — semantic spacing tokens (CSS utility classes defined in
 * index.css). Re-exported through theme.ts so callers can continue to
 * `import { spacing } from '../styles/theme'`.
 */

// SPACING SCALE — based on 4px grid.
// Use these semantic spacing utilities for consistency.

export const spacing = {
  // Raw Tailwind values (for reference)
  values: {
    xs: '1', // 4px
    sm: '2', // 8px
    default: '3', // 12px
    md: '4', // 16px
    lg: '6', // 24px
    xl: '8', // 32px
    '2xl': '12', // 48px
  },

  // Semantic CSS utility classes (preferred - use these)
  stack: {
    xs: 'stack-xs', // 4px vertical
    sm: 'stack-sm', // 8px vertical
    default: 'stack', // 12px vertical
    lg: 'stack-lg', // 16px vertical
    xl: 'stack-xl', // 24px vertical
  },

  section: {
    default: 'section-gap', // 24px between sections
    lg: 'section-gap-lg', // 32px for page-level
  },

  gap: {
    tight: 'gap-tight', // 4px
    compact: 'gap-compact', // 8px
    default: 'gap-default', // 12px
    comfortable: 'gap-comfortable', // 16px
    spacious: 'gap-spacious', // 24px
  },

  inline: {
    xs: 'inline-gap-xs', // 4px
    sm: 'inline-gap-sm', // 6px
    default: 'inline-gap', // 8px
    lg: 'inline-gap-lg', // 12px
  },

  pad: {
    xs: 'pad-xs', // 8px
    sm: 'pad-sm', // 12px
    default: 'pad', // 16px
    lg: 'pad-lg', // 24px
    xl: 'pad-xl', // 32px
  },

  // Chip/pill padding (for tags, badges, small interactive elements)
  chip: {
    sm: 'chip-pad', // px-3 py-1
    md: 'chip-pad-md', // px-3 py-1.5
    lg: 'chip-pad-lg', // px-3 py-2
  },

  // Tab button padding
  tab: 'tab-pad', // py-2.5 px-3

  // Semantic margin utilities (CSS classes from index.css)
  margin: {
    bottom: {
      section: 'mb-section', // 24px - between major sections
      sectionLg: 'mb-section-lg', // 32px - large section gaps
      heading: 'mb-heading', // 12px - after headings
      content: 'mb-content', // 16px - after content blocks
      inline: 'mb-2', // 8px - inline content (small bottom margins)
      tight: 'mb-tight', // 4px - very tight, for labels
    },
    top: {
      section: 'mt-section', // 32px - before major sections
      content: 'mt-content', // 16px - content separation
      heading: 'mt-heading', // 12px - before headings/after inline
      inline: 'mt-inline', // 8px - inline content
      tight: 'mt-tight', // 4px - very tight, for form fields
    },
    left: {
      tight: 'ml-tight', // 4px - minimal left margin
      inline: 'ml-inline', // 8px - inline content
      content: 'ml-content', // 16px - content indentation
      spacious: 'ml-spacious', // 24px - large indentation (lists)
    },
    right: {
      inline: 'mr-2', // 8px - inline content (e.g. side-anchored tooltips)
    },
  },

  // Padding utilities for dividers/sections
  padding: {
    top: {
      heading: 'pt-heading', // 12px - section divider top
      section: 'pt-section', // 16px - section divider with more space
      tight: 'pt-tight', // 4px - minimal top padding for subtle spacing
    },
    bottom: {
      inline: 'pb-inline', // 8px - inline bottom padding
      tight: 'pb-tight', // 4px - minimal bottom padding
    },
    right: {
      icon: 'pr-icon', // 40px - space for right-positioned icon
      tight: 'pr-tight', // 32px - slightly smaller right padding
    },
  },

  // Centered content padding (for loading states, empty states)
  centered: 'py-centered', // 48px vertical

  // Compact action button padding (remove, delete buttons)
  actionBtn: 'action-btn-pad', // 4px horizontal

  // Micro spacing (for fine-grained adjustments)
  micro: {
    gap: 'gap-micro', // 2px - very tight badge/icon spacing
    mt: 'mt-micro', // 2px - minimal top margin for icon alignment
    mtNeg: '-mt-tight', // -4px - negative margin for tight visual alignment
    pb: 'pb-micro', // 2px - minimal bottom padding
    pbCompact: 'pb-compact', // 4px - compact bottom padding
    pbCompactMd: 'pb-compact-md', // 6px - medium compact bottom padding
    mtCompactMd: 'mt-compact-md', // 6px - medium compact top margin
  },

  // Badge/status indicator padding
  badge: {
    xs: 'p-badge-xs', // 2px - extra small badge padding
    sm: 'p-badge-sm', // 4px - small badge padding
    padXs: 'badge-pad-xs', // px-2 py-0.5 - compact status badge
  },

  // Keyboard key styling
  kbd: 'kbd-pad', // px-1 py-0.5 - keyboard key padding

  // Left padding for indentation
  indent: 'pl-indent', // 20px - nested content indentation

  // Compact vertical padding (for list items, table cells)
  compact: {
    py: 'py-compact', // 4px - compact list item
    pyMd: 'py-compact-md', // 6px - slightly larger compact
  },

  // Row/cell padding
  row: {
    py: 'py-row', // 8px - table cell/row padding
    pyLg: 'py-row-lg', // 12px - larger row padding
  },

  // Cell horizontal padding
  cell: {
    px: 'px-cell', // 8px - table cell horizontal padding
  },

  // Icon button padding
  iconBtn: {
    sm: 'p-icon-btn', // 4px - compact icon button
    md: 'p-icon-btn-md', // 6px - medium icon button
  },

  // Main content layout padding
  mainPadding: {
    y: 'main-padding-y', // py-4 sm:py-6
    x: 'content-padding-x', // px-4 sm:px-6 lg:px-8
  },

  // Header padding (responsive)
  headerPadding: {
    y: 'header-padding-y', // py-2 sm:py-3
  },

  // Drawer/panel content padding
  drawerPad: 'drawer-content-pad', // px-4 sm:px-5 pb-10 pt-4

  // Table cell padding
  tableCell: {
    empty: 'table-cell-empty', // px-2 py-4 - empty state cells
  },
} as const;
