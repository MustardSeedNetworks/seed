/**
 * NetworkCard tests — the card is handed the DNS servers the IP config
 * response carries and has to put them on screen. It declared `dns: string[]`
 * in its props and rendered none of it, so a fixed backend still showed the
 * user nothing (#93).
 */

import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';

import { type DhcpData, NetworkCard } from './NetworkCard';

function makeData(overrides: Partial<DhcpData> = {}): DhcpData {
  return {
    interface: 'eth0',
    mac: '02:00:5e:10:00:00',
    vendor: 'Example Networks',
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

  // #123: the card leads with address and gateway; interface, MAC, vendor, mode
  // and the lease sit behind one Details control.
  describe('Details group', () => {
    it('keeps the secondary facts collapsed by default', () => {
      const { container } = render(<NetworkCard data={makeData()} />);
      const card = within(container);

      expect(card.getByText('192.0.2.10/24')).toBeInTheDocument();
      expect(card.getByText('Gateway')).toBeInTheDocument();
      expect(card.getByRole('button', { name: 'Details' })).toHaveAttribute(
        'aria-expanded',
        'false',
      );
      for (const label of ['Interface', 'MAC', 'Vendor', 'Mode', 'DHCP Server', 'Lease']) {
        expect(card.queryByText(label)).not.toBeInTheDocument();
      }
    });

    it('expands to every secondary fact and collapses again', async () => {
      const { container } = render(<NetworkCard data={makeData()} />);
      const card = within(container);
      const toggle = card.getByRole('button', { name: 'Details' });

      await userEvent.click(toggle);

      expect(toggle).toHaveAttribute('aria-expanded', 'true');
      const body = document.getElementById(toggle.getAttribute('aria-controls') ?? '');
      expect(body).not.toBeNull();
      const details = within(body as HTMLElement);
      expect(details.getByText('eth0')).toBeInTheDocument();
      expect(details.getByText('02:00:5e:10:00:00')).toBeInTheDocument();
      expect(details.getByText('Example Networks')).toBeInTheDocument();
      expect(details.getByText('DHCP')).toBeInTheDocument();
      expect(details.getByText('192.0.2.1')).toBeInTheDocument();
      expect(details.getByText('1h')).toBeInTheDocument();

      await userEvent.click(toggle);

      expect(toggle).toHaveAttribute('aria-expanded', 'false');
      expect(card.queryByText('Example Networks')).not.toBeInTheDocument();
    });

    it('lists only the facts the response carries', async () => {
      const { container } = render(
        <NetworkCard
          data={makeData({
            interface: undefined,
            vendor: undefined,
            ipv4: {
              address: '192.0.2.10',
              subnet: '24',
              gateway: null,
              dhcpServer: null,
              leaseTime: null,
            },
          })}
        />,
      );
      const card = within(container);

      await userEvent.click(card.getByRole('button', { name: 'Details' }));

      expect(card.getByText('MAC')).toBeInTheDocument();
      expect(card.getByText('Mode')).toBeInTheDocument();
      for (const label of ['Interface', 'Vendor', 'DHCP Server', 'Lease', 'Gateway']) {
        expect(card.queryByText(label)).not.toBeInTheDocument();
      }
    });
  });

  it('keeps DHCP timing behind its own control', async () => {
    const { container } = render(
      <NetworkCard
        data={makeData({ timing: { discover: 10, offer: 20, request: 30, total: 60 } })}
      />,
    );
    const card = within(container);

    expect(card.queryByText('Total')).not.toBeInTheDocument();
    await userEvent.click(card.getByRole('button', { name: 'DHCP Timing' }));
    expect(card.getByText('Total')).toBeInTheDocument();
  });

  // The fallback asked for `network.noIp`; the key is `network.noIP`, so an
  // interface with no address showed the raw key as its headline.
  it('says No IP when the interface has no address', () => {
    const { container } = render(<NetworkCard data={makeData({ ipv4: null })} />);

    expect(within(container).getByText('No IP')).toBeInTheDocument();
  });
});
