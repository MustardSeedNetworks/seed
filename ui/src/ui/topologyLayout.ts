/**
 * topologyLayout — where the topology map's nodes and edges go.
 *
 * Pure, O(n), and deterministic. The alternative, a force simulation, is what
 * makes a 250-node map stutter and what makes a coordinate impossible to
 * assert in a test; a discovered network also already has a hierarchy worth
 * showing, so the layout reads the role the sysinfo reconciler assigned
 * (internal/topology/device_role.go) and stacks core to edge.
 *
 * A tier wider than ROW_MAX wraps rather than stretching: 160 endpoints on one
 * line is 8000 units of drawing nobody can read once it is scaled to fit.
 *
 * Coordinates are SVG user units, not pixels — the page scales the viewBox.
 */

/** A node the caller wants drawn. `deviceType` is the reconciler's role. */
export interface GraphNodeInput {
  id: string;
  label: string;
  deviceType: string;
}

/** An edge. `learned` marks an FDB edge, which is drawn dashed. */
export interface GraphLinkInput {
  id: string;
  source: string;
  target: string;
  learned: boolean;
}

export interface PlacedNode extends GraphNodeInput {
  tier: number;
  x: number;
  y: number;
}

export interface PlacedLink extends GraphLinkInput {
  x1: number;
  y1: number;
  x2: number;
  y2: number;
}

export interface TopologyLayout {
  nodes: PlacedNode[];
  links: PlacedLink[];
  width: number;
  height: number;
}

/** Spacing chosen so a node's 28-unit hit area clears the WCAG 2.2 minimum
 * once the widest drawing (ROW_MAX columns) renders inside a desktop pane. */
const STEP_X = 48;
const STEP_Y = 76;
const MARGIN = 36;
const ROW_MAX = 20;

/** TIERS stacks the roles core-to-edge. Anything the reconciler could not
 * classify — and a role added later — falls to the edge tier rather than
 * vanishing, which is why the lookup has a default instead of an exhaustive
 * switch. */
const TIERS: readonly (readonly string[])[] = [
  ['router', 'firewall'],
  ['wireless-controller'],
  ['switch'],
  ['access-point'],
];
const EDGE_TIER = TIERS.length;

function tierOf(deviceType: string): number {
  const found = TIERS.findIndex((roles) => roles.includes(deviceType));
  return found === -1 ? EDGE_TIER : found;
}

/** orderWithin sorts a tier by the average x of the neighbours already placed
 * above it, which pulls a switch under its own router and keeps the edges from
 * crossing the whole drawing. A node with no placed neighbour sorts last, by
 * label then id so the result never depends on input order. */
function orderWithin(
  tier: GraphNodeInput[],
  neighbours: Map<string, string[]>,
  placed: Map<string, PlacedNode>,
): GraphNodeInput[] {
  const anchor = new Map<string, number>();
  for (const node of tier) {
    const xs = (neighbours.get(node.id) ?? [])
      .map((id) => placed.get(id)?.x)
      .filter((x): x is number => x !== undefined);
    if (xs.length > 0) {
      anchor.set(node.id, xs.reduce((sum, x) => sum + x, 0) / xs.length);
    }
  }
  return [...tier].sort((a, b) => {
    const ax = anchor.get(a.id);
    const bx = anchor.get(b.id);
    if (ax !== bx) {
      if (ax === undefined) return 1;
      if (bx === undefined) return -1;
      return ax - bx;
    }
    return a.label.localeCompare(b.label) || a.id.localeCompare(b.id);
  });
}

export function layoutTopology(
  nodes: readonly GraphNodeInput[],
  links: readonly GraphLinkInput[],
): TopologyLayout {
  const known = new Set(nodes.map((n) => n.id));
  const drawable = links.filter((l) => known.has(l.source) && known.has(l.target));

  const neighbours = new Map<string, string[]>();
  for (const link of drawable) {
    neighbours.set(link.source, [...(neighbours.get(link.source) ?? []), link.target]);
    neighbours.set(link.target, [...(neighbours.get(link.target) ?? []), link.source]);
  }

  const byTier = new Map<number, GraphNodeInput[]>();
  for (const node of nodes) {
    const tier = tierOf(node.deviceType);
    byTier.set(tier, [...(byTier.get(tier) ?? []), node]);
  }

  const placed = new Map<string, PlacedNode>();
  let row = 0;
  let widestRow = 0;

  for (let tier = 0; tier <= EDGE_TIER; tier++) {
    const members = byTier.get(tier);
    if (!members || members.length === 0) continue;

    const ordered = orderWithin(members, neighbours, placed);
    const perRow = Math.min(ordered.length, ROW_MAX);
    widestRow = Math.max(widestRow, perRow);

    ordered.forEach((node, index) => {
      const column = index % perRow;
      const rowIndex = row + Math.floor(index / perRow);
      // The last row of a wrapped tier is short; centring it on the full row
      // keeps the tier symmetrical instead of hanging off to the left.
      const inThisRow = Math.min(perRow, ordered.length - Math.floor(index / perRow) * perRow);
      const indent = ((perRow - inThisRow) * STEP_X) / 2;
      placed.set(node.id, {
        ...node,
        tier,
        x: MARGIN + indent + column * STEP_X,
        y: MARGIN + rowIndex * STEP_Y,
      });
    });

    row += Math.ceil(ordered.length / perRow);
  }

  const laidOut = [...placed.values()];
  return {
    nodes: laidOut,
    links: drawable.map((link) => {
      const from = placed.get(link.source);
      const to = placed.get(link.target);
      // Both ends are in `known`, so both are placed; the reads are narrowing.
      return {
        ...link,
        x1: from?.x ?? 0,
        y1: from?.y ?? 0,
        x2: to?.x ?? 0,
        y2: to?.y ?? 0,
      };
    }),
    width: MARGIN * 2 + Math.max(0, widestRow - 1) * STEP_X,
    height: MARGIN * 2 + Math.max(0, row - 1) * STEP_Y,
  };
}
