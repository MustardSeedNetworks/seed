/**
 * The layout is pure and deterministic on purpose: a force simulation is what
 * makes a 250-node map freeze and an E2E assertion flaky, and it cannot be
 * tested without animation frames. These lock the three properties the page
 * depends on — every node lands somewhere, the role tiers stack in one order,
 * and the same input always produces the same coordinates.
 */
import { describe, expect, it } from 'vitest';
import { type GraphLinkInput, type GraphNodeInput, layoutTopology } from './topologyLayout';

function n(id: string, deviceType: string): GraphNodeInput {
  return { id, label: id, deviceType };
}

function l(id: string, source: string, target: string, learned = false): GraphLinkInput {
  return { id, source, target, learned };
}

describe('layoutTopology', () => {
  it('places every node, including one no link reaches', () => {
    const laid = layoutTopology(
      [n('r1', 'router'), n('s1', 'switch'), n('orphan', 'server')],
      [l('e1', 'r1', 's1')],
    );

    expect(laid.nodes.map((p) => p.id).sort((a, b) => a.localeCompare(b))).toEqual([
      'orphan',
      'r1',
      's1',
    ]);
    expect(laid.nodes.every((p) => Number.isFinite(p.x) && Number.isFinite(p.y))).toBe(true);
  });

  it('stacks the role tiers core-to-edge, so a router never sits below its access point', () => {
    const laid = layoutTopology(
      [
        n('ap', 'access-point'),
        n('host', 'unknown'),
        n('rtr', 'router'),
        n('wlc', 'wireless-controller'),
        n('sw', 'switch'),
      ],
      [],
    );
    const y = (id: string): number => {
      const found = laid.nodes.find((p) => p.id === id);
      if (!found) throw new Error(`${id} was not placed`);
      return found.y;
    };

    expect(y('rtr')).toBeLessThan(y('wlc'));
    expect(y('wlc')).toBeLessThan(y('sw'));
    expect(y('sw')).toBeLessThan(y('ap'));
    expect(y('ap')).toBeLessThan(y('host'));
  });

  it('groups a firewall with the routers rather than inventing a tier for it', () => {
    const laid = layoutTopology([n('fw', 'firewall'), n('rtr', 'router')], []);
    const tiers = new Set(laid.nodes.map((p) => p.tier));

    expect(tiers.size).toBe(1);
  });

  it('is deterministic — the same input twice gives the same coordinates', () => {
    const nodes = [n('b', 'switch'), n('a', 'switch'), n('r', 'router')];
    const links = [l('e1', 'r', 'a'), l('e2', 'r', 'b')];

    expect(layoutTopology(nodes, links)).toEqual(layoutTopology(nodes, links));
  });

  it('wraps a wide tier instead of growing without bound, keeping the drawing roughly square', () => {
    const many = Array.from({ length: 160 }, (_, i) => n(`h${i}`, 'unknown'));
    const laid = layoutTopology(many, []);

    // 160 endpoints on one row would be several thousand units wide.
    expect(laid.width).toBeLessThan(1200);
    expect(new Set(laid.nodes.map((p) => p.y)).size).toBeGreaterThan(1);
  });

  it('drops a link whose far end is not a known node, and keeps the ones that are', () => {
    const laid = layoutTopology(
      [n('r1', 'router'), n('s1', 'switch')],
      [l('kept', 'r1', 's1'), l('dangling', 'r1', 'never-polled')],
    );

    expect(laid.links.map((e) => e.id)).toEqual(['kept']);
  });

  it('carries the learned flag through, because FDB edges are drawn dashed', () => {
    const laid = layoutTopology(
      [n('r1', 'router'), n('s1', 'switch')],
      [l('fdb', 'r1', 's1', true)],
    );

    expect(laid.links[0]?.learned).toBe(true);
  });

  it('gives a 250-node graph coordinates quickly enough to render on every refresh', () => {
    const nodes = [
      n('core', 'router'),
      ...Array.from({ length: 20 }, (_, i) => n(`sw${i}`, 'switch')),
      ...Array.from({ length: 60 }, (_, i) => n(`ap${i}`, 'access-point')),
      ...Array.from({ length: 169 }, (_, i) => n(`h${i}`, 'unknown')),
    ];
    const links = nodes.slice(1).map((node, i) => l(`e${i}`, 'core', node.id));

    const started = performance.now();
    const laid = layoutTopology(nodes, links);

    expect(laid.nodes).toHaveLength(250);
    expect(performance.now() - started).toBeLessThan(250);
  });
});
