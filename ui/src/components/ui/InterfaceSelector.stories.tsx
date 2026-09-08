import type { Meta, StoryObj } from '@storybook/react-vite';
import { useState } from 'react';
import type { InterfaceInfo } from '../../types/generated/categorized-interfaces-response';
import { InterfaceSelector } from './InterfaceSelector';

const sampleInterfaces: InterfaceInfo[] = [
  {
    name: 'eth0',
    friendlyName: 'Primary Ethernet',
    type: 'ethernet',
    up: true,
    running: true,
    hardwareAddr: '00:1b:21:aa:bb:01',
    mtu: 1500,
    addresses: ['192.168.1.10/24'],
    speedDisplay: '1 Gb/s',
    chipsetVendor: 'Intel',
    chipsetModel: 'i225',
    hasTDR: true,
    hasDOM: true,
  },
  {
    name: 'eth1',
    description: 'Backup NIC',
    type: 'ethernet',
    up: false,
    running: false,
    hardwareAddr: '00:1b:21:aa:bb:02',
    mtu: 1500,
    addresses: [],
    speedDisplay: '1 Gb/s',
  },
  {
    name: 'wlan0',
    friendlyName: 'WiFi Adapter',
    type: 'wifi',
    up: true,
    running: true,
    hardwareAddr: '00:1b:21:aa:bb:03',
    mtu: 1500,
    addresses: ['192.168.1.11/24'],
  },
  {
    name: 'wlan1',
    type: 'wifi',
    up: false,
    running: false,
    hardwareAddr: '00:1b:21:aa:bb:04',
    mtu: 1500,
    addresses: [],
  },
];

const meta = {
  title: 'UI/InterfaceSelector',
  component: InterfaceSelector,
} satisfies Meta<typeof InterfaceSelector>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Ethernet: Story = {
  args: { interfaces: [], currentInterface: 'eth0', isWifi: false, onChange: () => {} },
  render: () => {
    const [current, setCurrent] = useState('eth0');
    return (
      <InterfaceSelector
        interfaces={sampleInterfaces}
        currentInterface={current}
        isWifi={false}
        onChange={setCurrent}
        recommendedEthernet="eth0"
      />
    );
  },
};

export const Wifi: Story = {
  args: { interfaces: [], currentInterface: 'wlan0', isWifi: true, onChange: () => {} },
  render: () => {
    const [current, setCurrent] = useState('wlan0');
    return (
      <InterfaceSelector
        interfaces={sampleInterfaces}
        currentInterface={current}
        isWifi={true}
        onChange={setCurrent}
        recommendedWifi="wlan0"
      />
    );
  },
};

export const Warning: Story = {
  args: { interfaces: [], currentInterface: 'eth0', isWifi: false, onChange: () => {} },
  render: () => (
    <InterfaceSelector
      interfaces={sampleInterfaces}
      currentInterface="eth1"
      isWifi={false}
      onChange={() => {}}
      warning="Selected interface is down."
      suggestedInterface="eth0"
      onAcceptSuggestion={() => {}}
    />
  ),
};
