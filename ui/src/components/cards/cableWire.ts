/**
 * T568A/B Ethernet wire-pair colors — a DOMAIN palette, not UI theme colors.
 *
 * These deliberately use raw Tailwind palette classes because each value
 * represents the PHYSICAL insulation color of a conductor in an 8P8C cable: a
 * green wire must render green, a blue wire blue, a brown wire brown, etc.
 * Mapping them onto the brand/semantic tokens would be wrong (it would, e.g.,
 * turn the blue pair green). They are therefore intentionally exempt from the
 * design-token rule, and this file is allowlisted from the raw-palette lint
 * check. Keep it limited to literal wire colors.
 *
 * Keys are lowercased pair labels for case-insensitive lookup.
 */
export const wireColorMap: Record<string, string> = {
  'white/orange': 'bg-orange-100 border-orange-400',
  orange: 'bg-orange-500',
  'white/green': 'bg-green-100 border-green-400',
  green: 'bg-green-500',
  'white/blue': 'bg-blue-100 border-blue-400',
  blue: 'bg-blue-500',
  'white/brown': 'bg-amber-100 border-amber-600',
  brown: 'bg-amber-700',
};
