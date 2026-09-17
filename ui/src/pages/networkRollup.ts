import type { DnsData } from '../components/cards/DnsCard';
import type { GatewayData } from '../components/cards/GatewayCard';
import type { RollupState } from '../ui/StatusRollup';

/**
 * The Network band, derived from the measurements it displays.
 *
 * It used to ask only whether a card had arrived, so a gateway reporting 0/3
 * packets and 100 % loss still rendered "Up GATEWAY" over its own numbers
 * (#2690). A band that contradicts the card under it is worse than no band:
 * the reader in a hurry stops at the band.
 *
 * "Not measured" is kept apart from "did not answer" for the same reason the
 * rollup has an `unknown` state at all — an unprivileged daemon that never
 * sent an echo request has learned nothing about the link, and printing
 * "Down" there is the mirror image of the defect.
 */

export interface NetworkRollupInput {
  gateway: GatewayData | null;
  dns: DnsData | null;
  loading: boolean;
}

/* The keys are spelled out rather than typed `string` so `t()` still checks
   them against the pages namespace; a bare string widens the key union away
   and a renamed key would then survive to runtime as raw text. */
type HeadlineKey =
  | 'network.rollupProbing'
  | 'network.rollupNoGateway'
  | 'network.rollupGatewayUnmeasured'
  | 'network.rollupGatewayUnreachable'
  | 'network.rollupGatewayLossy'
  | 'network.rollupNoDns'
  | 'network.rollupHealthy';

type BodyKey =
  | 'network.rollupNoGatewayBody'
  | 'network.rollupGatewayUnmeasuredBody'
  | 'network.rollupGatewayUnreachableBody'
  | 'network.rollupGatewayLossyBody'
  | 'network.rollupNoDnsBody';

type FigureKey =
  | 'network.figureUp'
  | 'network.figureDown'
  | 'network.figureLossy'
  | 'network.figureNone'
  | 'network.figureUnknown';

export interface NetworkRollup {
  state: RollupState;
  headlineKey: HeadlineKey;
  /** Absent when the headline needs no second sentence. */
  bodyKey?: BodyKey;
  gatewayFigureKey: FigureKey;
  dnsFigureKey: FigureKey;
}

type GatewayVerdict = 'absent' | 'unmeasured' | 'unreachable' | 'lossy' | 'answering';

function gatewayVerdict(gateway: GatewayData | null): GatewayVerdict {
  if (!gateway || gateway.gateway === '') {
    return 'absent';
  }
  if (gateway.sent === 0) {
    return 'unmeasured';
  }
  if (gateway.received === 0) {
    return 'unreachable';
  }
  return gateway.lossPercent > 0 ? 'lossy' : 'answering';
}

function dnsAnswered(dns: DnsData | null): boolean {
  return Boolean(dns?.forward && dns.forward.status !== 'error');
}

const GATEWAY_FIGURE: Record<GatewayVerdict, FigureKey> = {
  absent: 'network.figureNone',
  unmeasured: 'network.figureUnknown',
  unreachable: 'network.figureDown',
  lossy: 'network.figureLossy',
  answering: 'network.figureUp',
};

export function networkRollup({ gateway, dns, loading }: NetworkRollupInput): NetworkRollup {
  const verdict = gatewayVerdict(gateway);
  const gatewayFigureKey = GATEWAY_FIGURE[verdict];
  const dnsFigureKey: FigureKey = !dns
    ? 'network.figureNone'
    : dnsAnswered(dns)
      ? 'network.figureUp'
      : 'network.figureDown';

  if (loading) {
    return {
      state: 'unknown',
      headlineKey: 'network.rollupProbing',
      gatewayFigureKey,
      dnsFigureKey,
    };
  }

  switch (verdict) {
    case 'absent':
      return {
        state: 'crit',
        headlineKey: 'network.rollupNoGateway',
        bodyKey: 'network.rollupNoGatewayBody',
        gatewayFigureKey,
        dnsFigureKey,
      };
    case 'unmeasured':
      return {
        state: 'unknown',
        headlineKey: 'network.rollupGatewayUnmeasured',
        bodyKey: 'network.rollupGatewayUnmeasuredBody',
        gatewayFigureKey,
        dnsFigureKey,
      };
    case 'unreachable':
      return {
        state: 'crit',
        headlineKey: 'network.rollupGatewayUnreachable',
        bodyKey: 'network.rollupGatewayUnreachableBody',
        gatewayFigureKey,
        dnsFigureKey,
      };
    default:
      break;
  }

  if (!dnsAnswered(dns)) {
    return {
      state: 'warn',
      headlineKey: 'network.rollupNoDns',
      bodyKey: 'network.rollupNoDnsBody',
      gatewayFigureKey,
      dnsFigureKey,
    };
  }

  if (verdict === 'lossy') {
    return {
      state: 'warn',
      headlineKey: 'network.rollupGatewayLossy',
      bodyKey: 'network.rollupGatewayLossyBody',
      gatewayFigureKey,
      dnsFigureKey,
    };
  }

  return {
    state: 'ok',
    headlineKey: 'network.rollupHealthy',
    gatewayFigureKey,
    dnsFigureKey,
  };
}
