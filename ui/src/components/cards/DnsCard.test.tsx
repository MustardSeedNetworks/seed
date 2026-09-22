/**
 * DnsCard: whose resolvers the card is showing (#2690, slice 2).
 *
 * The card used to print the host's resolvers whatever interface was selected,
 * beside a gateway taken from the system default route. The list alone cannot
 * say which of three things happened, so these cover all three: the interface
 * has resolvers, it has none, or this host cannot attribute resolvers to an
 * interface at all.
 */

import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { DnsCard, type DnsData } from './DnsCard';

function dnsData(over: Partial<DnsData> = {}): DnsData {
  return {
    server: '10.0.0.1',
    servers: ['10.0.0.1'],
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
  };
}

describe('DnsCard resolver scope', () => {
  it('names the resolvers as the interface’s when they are scoped to it', () => {
    render(<DnsCard data={dnsData({ serverScope: 'interface' })} />);
    expect(screen.getByText('Resolvers for this interface')).toBeInTheDocument();
    expect(screen.getByText('10.0.0.1')).toBeInTheDocument();
  });

  it('says a system-wide list is system-wide instead of implying it is the link’s', () => {
    render(<DnsCard data={dnsData({ serverScope: 'system' })} />);
    expect(screen.getByText('DNS Servers')).toBeInTheDocument();
    expect(
      screen.getByText('Shown for the whole host, not for the selected interface.'),
    ).toBeInTheDocument();
  });

  /* An interface with no resolvers of its own shows no measurement, because
     one taken through another interface would describe that link. The card
     has to say that, or the reader sees an empty card and assumes a bug. */
  it('explains an interface with no resolvers instead of showing an empty list', () => {
    render(<DnsCard data={dnsData({ serverScope: 'interface', servers: [], forward: null })} />);
    expect(screen.getByText('No resolvers on this interface')).toBeInTheDocument();
    expect(screen.queryByText('Forward (A)')).not.toBeInTheDocument();
  });

  /* A payload from before the scoping carries no scope at all; it must read as
     the system-wide answer it is, not as an absence. */
  it('treats a payload without a scope as system-wide', () => {
    render(<DnsCard data={dnsData({ servers: [] })} />);
    expect(screen.queryByText('No resolvers on this interface')).not.toBeInTheDocument();
    expect(screen.getByText('DNS Servers')).toBeInTheDocument();
  });
});
