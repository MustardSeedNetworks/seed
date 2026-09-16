/**
 * NetworkCard tests — the card is handed the DNS servers the IP config
 * response carries and has to put them on screen. It declared `dns: string[]`
 * in its props and rendered none of it, so a fixed backend still showed the
 * user nothing (#93).
 */

import { render, screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { type DhcpData, NetworkCard } from './NetworkCard';

function makeData(overrides: Partial<DhcpData> = {}): DhcpData {
  return {
    mac: '02:00:5e:10:00:00',
    mode: 'dhcp',
    ipv4: {
      address: '192.0.2.10',
      subnet: '24',
      gateway: '192.0.2.1',
      dhcpServer: '192.0.2.1',
      leaseTime: 3600,
    },
    ipv6: [],
    dns: [],
    timing: null,
    ...overrides,
  };
}

describe('NetworkCard', () => {
  it('renders every DNS server it is given', () => {
    // Scoped to the card: Tooltip also renders each value into a portal bubble
    // on document.body, which an unscoped query would match a second time.
    const { container } = render(
      <NetworkCard data={makeData({ dns: ['192.0.2.53', '198.51.100.53'] })} />,
    );

    expect(within(container).getByText('192.0.2.53')).toBeInTheDocument();
    expect(within(container).getByText('198.51.100.53')).toBeInTheDocument();
  });

  it('omits the DNS section when the response carries no servers', () => {
    render(<NetworkCard data={makeData({ dns: [] })} />);

    expect(screen.queryByText('DNS')).not.toBeInTheDocument();
  });
});
