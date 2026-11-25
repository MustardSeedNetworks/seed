/**
 * The fleet's one `<title>` format: "Label · Product", the page's navigation
 * label first (so a tab that truncates on the right keeps the part that tells
 * tabs apart), then the bare product name: "Devices · NIAC", "History · Stem",
 * "Surveys · Trellis", "Dashboard · Seed". A page with no label of its own (the
 * sign-in screen) is the product name alone. The separator is U+00B7, not a pipe:
 * screen readers announce "|" aloud.
 *
 * Call it from the route change, with the same translated label the PageHeader
 * title and the navigation use.
 */
export function documentTitle(label: string, product: string): string {
  return label === '' ? product : `${label} · ${product}`;
}
