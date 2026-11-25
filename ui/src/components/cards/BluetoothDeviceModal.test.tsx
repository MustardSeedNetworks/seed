/**
 * BluetoothDeviceModal tests — the device table is the shared DataTable with a
 * search box above it and the decoded detail of the clicked row below it.
 */

import { fireEvent, render, screen, within } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import i18n from '../../i18n';
import type { BluetoothDevice } from '../../types/generated/bluetooth-scan-response';
import { BluetoothDeviceModal } from './BluetoothDeviceModal';

function device(overrides: Partial<BluetoothDevice>): BluetoothDevice {
  return {
    id: overrides.address ?? 'id',
    address: 'AA:BB:CC:00:00:01',
    name: '',
    alias: '',
    vendor: '',
    isConnected: false,
    type: 'BLE',
    deviceClass: '',
    appearance: 0,
    rssi: -60,
    txPower: 0,
    estDistanceM: 2,
    isConnectable: true,
    isAuthorized: false,
    isTrusted: false,
    isPaired: false,
    isBlocked: false,
    firstSeen: '2026-01-01T00:00:00Z',
    lastSeen: '2026-01-01T00:00:00Z',
    ...overrides,
  };
}

const devices = [
  device({ address: 'AA:BB:CC:00:00:01', name: 'Keyboard', companyName: 'Logitech' }),
  device({
    address: 'AA:BB:CC:00:00:02',
    name: 'Headset',
    companyName: 'Bose',
    appearanceLabel: 'Headphones',
  }),
];

beforeEach(async () => {
  await i18n.changeLanguage('en');
});

describe('BluetoothDeviceModal', () => {
  it('filters the rows from the search box', () => {
    render(<BluetoothDeviceModal isOpen={true} onClose={vi.fn()} devices={devices} stats={null} />);

    expect(screen.getByText('Keyboard')).toBeInTheDocument();
    fireEvent.change(screen.getByRole('searchbox'), { target: { value: 'bose' } });
    expect(screen.queryByText('Keyboard')).not.toBeInTheDocument();
    expect(screen.getByText('Headset')).toBeInTheDocument();
  });

  it('shows the decoded detail of the clicked row and hides it on a second click', () => {
    render(<BluetoothDeviceModal isOpen={true} onClose={vi.fn()} devices={devices} stats={null} />);
    expect(screen.queryByTestId('bluetooth-device-detail')).not.toBeInTheDocument();

    const row = screen.getByText('Headset').closest('tr');
    if (!row) {
      throw new Error('no row for Headset');
    }
    fireEvent.click(row);
    const detail = screen.getByTestId('bluetooth-device-detail');
    expect(within(detail).getByText('Headphones')).toBeInTheDocument();
    expect(row).toHaveAttribute('aria-selected', 'true');

    fireEvent.click(row);
    expect(screen.queryByTestId('bluetooth-device-detail')).not.toBeInTheDocument();
  });
});
