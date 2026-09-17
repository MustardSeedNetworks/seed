import { describe, expect, it } from 'vitest';
import type { DnsData } from '../components/cards/DnsCard';
import type { GatewayData } from '../components/cards/GatewayCard';
import { networkRollup } from './networkRollup';

/**
 * The band derives from the measurements it displays (#2690).
 *
 * The defect these cases pin: the band asked only whether a gateway card had
 * arrived, so a card reporting 0/3 packets and 100 % loss rendered "ALL CLEAR,
 * Up GATEWAY" over its own numbers. Every row below is a state the card can
 * actually be in, and the assertion is on the state the band may claim.
 */

function gatewayCard(over: Partial<GatewayData> = {}): GatewayData {
  return {
    gateway: '192.168.1.1',
    reachable: true,
    sent: 3,
    received: 3,
    lossPercent: 0,
    minTime: 1,
    maxTime: 2,
    avgTime: 1.5,
    lastTime: 1.5,
    status: 'success',
    ...over,
  };
}

function dnsCard(over: Partial<DnsData> = {}): DnsData {
  return {
    server: '192.168.1.1',
    testHostname: 'example.com',
    forward: {
      result: '93.184.216.34',
      time: 12,
      timeMs: 12,
      status: 'success',
      resolved: ['93.184.216.34'],
    },
    reverse: null,
    ...over,
  } as DnsData;
}

describe('networkRollup', () => {
  it('is unknown while the probes are still running', () => {
    expect(networkRollup({ gateway: null, dns: null, loading: true }).state).toBe('unknown');
  });

  it('is critical when no gateway card has arrived', () => {
    const rollup = networkRollup({ gateway: null, dns: dnsCard(), loading: false });
    expect(rollup.state).toBe('crit');
    expect(rollup.headlineKey).toBe('network.rollupNoGateway');
  });

  it('is critical when the active interface has no gateway of its own', () => {
    const rollup = networkRollup({
      gateway: gatewayCard({ gateway: '', sent: 0, received: 0, reachable: false }),
      dns: dnsCard(),
      loading: false,
    });
    expect(rollup.state).toBe('crit');
    expect(rollup.headlineKey).toBe('network.rollupNoGateway');
  });

  it('is critical when every packet to the gateway was lost', () => {
    const rollup = networkRollup({
      gateway: gatewayCard({ reachable: false, received: 0, lossPercent: 100, status: 'error' }),
      dns: dnsCard(),
      loading: false,
    });
    expect(rollup.state).toBe('crit');
    expect(rollup.headlineKey).toBe('network.rollupGatewayUnreachable');
    expect(rollup.gatewayFigureKey).toBe('network.figureDown');
  });

  it('is unknown, not healthy, when the gateway was never measured', () => {
    const rollup = networkRollup({
      gateway: gatewayCard({ reachable: false, sent: 0, received: 0, status: 'unknown' }),
      dns: dnsCard(),
      loading: false,
    });
    expect(rollup.state).toBe('unknown');
    expect(rollup.headlineKey).toBe('network.rollupGatewayUnmeasured');
    expect(rollup.gatewayFigureKey).toBe('network.figureUnknown');
  });

  it('warns on partial loss rather than calling the link healthy', () => {
    const rollup = networkRollup({
      gateway: gatewayCard({ received: 2, lossPercent: 33.3, status: 'warning' }),
      dns: dnsCard(),
      loading: false,
    });
    expect(rollup.state).toBe('warn');
    expect(rollup.headlineKey).toBe('network.rollupGatewayLossy');
  });

  it('warns when the gateway answers but no resolver did', () => {
    const rollup = networkRollup({ gateway: gatewayCard(), dns: null, loading: false });
    expect(rollup.state).toBe('warn');
    expect(rollup.headlineKey).toBe('network.rollupNoDns');
    expect(rollup.dnsFigureKey).toBe('network.figureNone');
  });

  it('warns when the resolver was asked and failed', () => {
    const rollup = networkRollup({
      gateway: gatewayCard(),
      dns: dnsCard({
        forward: { result: '', time: 0, timeMs: 0, status: 'error', error: 'timeout' },
      } as Partial<DnsData>),
      loading: false,
    });
    expect(rollup.state).toBe('warn');
    expect(rollup.headlineKey).toBe('network.rollupNoDns');
    expect(rollup.dnsFigureKey).toBe('network.figureDown');
  });

  it('is healthy only when both were measured and both answered', () => {
    const rollup = networkRollup({ gateway: gatewayCard(), dns: dnsCard(), loading: false });
    expect(rollup.state).toBe('ok');
    expect(rollup.headlineKey).toBe('network.rollupHealthy');
    expect(rollup.gatewayFigureKey).toBe('network.figureUp');
    expect(rollup.dnsFigureKey).toBe('network.figureUp');
  });
});
