import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import type { SNMPWalkResponse } from '../../types/generated/snmp-walk-response';
import { MibBrowser } from './MibBrowser';

const IF_DESCR: SNMPWalkResponse = {
  targetId: 't1',
  oid: '1.3.6.1.2.1.2.2.1.2',
  rows: [
    { oid: '1.3.6.1.2.1.2.2.1.2.1', name: 'ifDescr', instance: '1', value: 'Gi0/1' },
    { oid: '1.3.6.1.2.1.2.2.1.2.2', name: 'ifDescr', instance: '2', value: 'Gi0/2' },
    { oid: '1.3.6.1.4.1.99999.1.0', value: '0a:ff' },
  ],
  truncated: false,
};

async function walkFor(oid: string): Promise<void> {
  await userEvent.type(screen.getByTestId('mib-oid'), oid);
  await userEvent.click(screen.getByTestId('mib-walk'));
}

describe('MibBrowser', () => {
  it('walks the trimmed OID and lists each row by object and instance', async () => {
    const walk = vi.fn<(oid: string) => Promise<SNMPWalkResponse>>().mockResolvedValue(IF_DESCR);
    render(<MibBrowser walk={walk} />);

    await walkFor('  ifDescr ');

    expect(walk).toHaveBeenCalledWith('ifDescr');
    expect(await screen.findByTestId('mib-summary')).toHaveTextContent(
      '3 objects under 1.3.6.1.2.1.2.2.1.2',
    );
    const rows = screen.getByTestId('mib-rows').querySelectorAll('li');
    expect([...rows].map((li) => li.textContent)).toEqual([
      'ifDescr.1Gi0/1',
      'ifDescr.2Gi0/2',
      // An object the MIB database does not know keeps its numeric OID.
      '1.3.6.1.4.1.99999.1.00a:ff',
    ]);
    expect(screen.queryByTestId('mib-truncated')).toBeNull();
  });

  it('says when the walk stopped at its limit', async () => {
    const walk = vi
      .fn<(oid: string) => Promise<SNMPWalkResponse>>()
      .mockResolvedValue({ ...IF_DESCR, truncated: true });
    render(<MibBrowser walk={walk} />);

    await walkFor('ifTable');

    expect(await screen.findByTestId('mib-truncated')).toHaveTextContent(
      'Stopped at 3 objects. Walk a narrower subtree to see the rest.',
    );
  });

  it('says when the subtree is empty', async () => {
    const walk = vi
      .fn<(oid: string) => Promise<SNMPWalkResponse>>()
      .mockResolvedValue({ targetId: 't1', oid: '1.3.6.1.9', rows: [], truncated: false });
    render(<MibBrowser walk={walk} />);

    await walkFor('1.3.6.1.9');

    expect(await screen.findByTestId('mib-empty')).toHaveTextContent(
      'The device returned nothing under 1.3.6.1.9.',
    );
    expect(screen.queryByTestId('mib-rows')).toBeNull();
  });

  it("shows the server's reason when the walk fails, and clears it on the next walk", async () => {
    const walk = vi
      .fn<(oid: string) => Promise<SNMPWalkResponse>>()
      .mockRejectedValueOnce(new Error("'oid' must be a numeric OID or a known MIB object name"))
      .mockResolvedValueOnce(IF_DESCR);
    render(<MibBrowser walk={walk} />);

    await walkFor('notAnObject');
    expect(await screen.findByRole('alert')).toHaveTextContent(
      "The walk failed: 'oid' must be a numeric OID or a known MIB object name",
    );

    await userEvent.clear(screen.getByTestId('mib-oid'));
    await walkFor('ifDescr');
    expect(await screen.findByTestId('mib-summary')).toBeInTheDocument();
    expect(screen.queryByRole('alert')).toBeNull();
  });

  it('will not walk until an OID is entered, nor twice at once', async () => {
    let finish: (value: SNMPWalkResponse) => void = () => undefined;
    const walk = vi.fn<(oid: string) => Promise<SNMPWalkResponse>>().mockReturnValue(
      new Promise((resolve) => {
        finish = resolve;
      }),
    );
    render(<MibBrowser walk={walk} />);

    // Enter in an empty field must not walk the empty OID.
    await userEvent.type(screen.getByTestId('mib-oid'), '{Enter}');
    expect(walk).not.toHaveBeenCalled();
    expect(screen.getByTestId('mib-walk')).toBeDisabled();

    await walkFor('sysDescr.0');
    expect(screen.getByTestId('mib-walk')).toBeDisabled();
    expect(screen.getByTestId('mib-walk')).toHaveTextContent('Walking…');
    await userEvent.type(screen.getByTestId('mib-oid'), '{Enter}');
    expect(walk).toHaveBeenCalledTimes(1);

    finish(IF_DESCR);
    expect(await screen.findByTestId('mib-summary')).toBeInTheDocument();
    expect(screen.getByTestId('mib-walk')).toBeEnabled();
  });

  it('keeps the form disabled for a caller who may not walk', async () => {
    const walk = vi.fn<(oid: string) => Promise<SNMPWalkResponse>>();
    render(<MibBrowser walk={walk} disabledReason="Read-only" />);

    expect(screen.getByTestId('mib-oid')).toBeDisabled();
    // Tooltip keeps the button focusable to read its reason; it still does nothing.
    expect(screen.getByTestId('mib-walk')).toHaveAttribute('aria-disabled', 'true');
    await userEvent.click(screen.getByTestId('mib-walk'));
    expect(walk).not.toHaveBeenCalled();
  });
});
