/**
 * Pins the seam between the discovery API and the props SwitchCard renders.
 *
 * SwitchCard's stories hand-write their props, and the Go decoder tests assert
 * on Go structs, so nothing previously covered the step in between: a backend
 * field rename or a protocol-case change would leave every Go test and every
 * story passing while the card rendered blank (#486).
 *
 * The payloads below are the shape `GET /api/v1/security/discovery` actually
 * emits -- see internal/api/handlers_discovery.go's DiscoveryNeighborInfo and
 * the mapping in internal/discovery/enumerate/manager.go's GetNeighbors.
 */
import { renderHook } from '@testing-library/react';
import type React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { SwitchData } from '../components/cards/SwitchCard';
import { useNetworkFetchers } from './useNetworkFetchers';

type CardState = Parameters<typeof useNetworkFetchers>[0] extends {
  setCards: React.Dispatch<React.SetStateAction<infer S>>;
}
  ? S
  : never;

function ref<T>(value: T): React.MutableRefObject<T> {
  return { current: value };
}

/**
 * Renders the hook, runs fetchDiscoveryData against a stubbed response, and
 * returns whatever it wrote into the `switch` card slot.
 */
async function switchCardFor(body: unknown): Promise<SwitchData | null> {
  let cards = { switch: null } as CardState;

  vi.stubGlobal(
    'fetch',
    vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(body),
    }),
  );

  const { result } = renderHook(() =>
    useNetworkFetchers({
      currentInterfaceRef: ref('eth0'),
      setCards: (update) => {
        cards = typeof update === 'function' ? update(cards) : update;
      },
      setCurrentInterface: vi.fn(),
      setInterfaces: vi.fn(),
      setAppVersion: vi.fn(),
      setNetworkDiscovery: vi.fn(),
      setIsWifi: vi.fn(),
      userSetWifiModeRef: ref(false),
      networkDiscoveryAbortRef: ref<AbortController | null>(null),
      prevLinkUpRef: ref<boolean | null>(null),
    }),
  );

  await result.current.fetchDiscoveryData();

  return cards.switch;
}

/** One neighbour as the Go handler serialises it. */
function neighbor(overrides: Record<string, unknown> = {}): Record<string, unknown> {
  return {
    protocol: 'lldp',
    chassisId: '00:11:22:33:44:55',
    portId: 'GigabitEthernet0/1',
    portDescription: 'Uplink to core',
    systemName: 'access-sw-01',
    systemDescription: 'Cisco IOS Software, C2960',
    capabilities: ['Bridge', 'Router'],
    managementAddress: '10.0.0.2',
    ttl: 120,
    lastSeen: '2026-08-27T10:00:00Z',
    sourceMAC: '00:11:22:33:44:55',
    ...overrides,
  };
}

describe('fetchDiscoveryData -> SwitchCard props', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('maps every field SwitchCard renders', async () => {
    const data = await switchCardFor({ neighbors: [neighbor()] });

    expect(data).toEqual({
      protocol: 'lldp',
      switchName: 'access-sw-01',
      portId: 'GigabitEthernet0/1',
      portDescription: 'Uplink to core',
      managementIp: '10.0.0.2',
      systemDescription: 'Cisco IOS Software, C2960',
    } satisfies SwitchData);
  });

  // The Go constants are lowercase, but the handler's doc comment claimed
  // uppercase for a long time. Both must land on a real protocol, never
  // 'unknown' -- 'unknown' renders as the literal label "Unknown".
  it.each([
    ['lldp', 'lldp'],
    ['cdp', 'cdp'],
    ['edp', 'edp'],
    ['LLDP', 'lldp'],
    ['CDP', 'cdp'],
  ] as const)('normalises protocol %s to %s', async (wire, expected) => {
    const data = await switchCardFor({ neighbors: [neighbor({ protocol: wire })] });

    expect(data?.protocol).toBe(expected);
  });

  it('falls back to unknown for a protocol SwitchCard has no label for', async () => {
    const data = await switchCardFor({ neighbors: [neighbor({ protocol: 'mndp' })] });

    expect(data?.protocol).toBe('unknown');
  });

  // CDP and EDP neighbours carry no systemName of their own: the Go mapping
  // copies DeviceID into both systemName and chassisId. If that ever stops,
  // the card falls back to chassisId rather than rendering nameless.
  it('falls back to chassisId when systemName is absent', async () => {
    const data = await switchCardFor({
      neighbors: [neighbor({ protocol: 'cdp', systemName: undefined, chassisId: 'core-sw-01' })],
    });

    expect(data?.switchName).toBe('core-sw-01');
  });

  // `omitempty` drops these fields entirely rather than sending "". The card
  // hides each row on null, so the mapping must produce null, not undefined.
  it('maps omitted optional fields to null', async () => {
    const data = await switchCardFor({
      neighbors: [
        {
          protocol: 'edp',
          chassisId: 'summit-1',
          portId: '1:1',
          ttl: 120,
          lastSeen: '2026-08-27T10:00:00Z',
          sourceMAC: '00:04:96:00:00:01',
        },
      ],
    });

    expect(data).toEqual({
      protocol: 'edp',
      switchName: 'summit-1',
      portId: '1:1',
      portDescription: null,
      managementIp: null,
      systemDescription: null,
    } satisfies SwitchData);
  });

  it('clears the card when no neighbours were discovered', async () => {
    expect(await switchCardFor({ neighbors: [] })).toBeNull();
  });

  it('clears the card when the response has no neighbours array', async () => {
    expect(await switchCardFor({})).toBeNull();
  });
});
