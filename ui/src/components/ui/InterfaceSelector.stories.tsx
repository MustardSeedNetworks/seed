import type { Meta, StoryObj } from '@storybook/react-vite';
import { useState } from 'react';
import { InterfaceSelector, type NetworkInterface } from './InterfaceSelector';

const sampleInterfaces: NetworkInterface[] = [
  {
    name: 'eth0',
    friendlyName: 'Primary Ethernet',
    type: 'ethernet',
    up: true,
    speedDisplay: '1 Gb/s',
    chipsetVendor: 'Intel',
    chipsetModel: 'i225',
    hasTdr: true,
    hasDom: true,
  },
  {
    name: 'eth1',
    description: 'Backup NIC',
    type: 'ethernet',
    up: false,
    speedDisplay: '1 Gb/s',
  },
  {
    name: 'wlan0',
    friendlyName: 'WiFi Adapter',
    type: 'wifi',
    up: true,
    signalStrength: -48,
  },
  {
    name: 'wlan1',
    type: 'wifi',
    up: false,
    signalStrength: -90,
  },
];

const meta = {
  title: 'UI/InterfaceSelector',
  component: InterfaceSelector,
} satisfies Meta<typeof InterfaceSelector>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Ethernet: Story = {
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
  render: () => {
    const [current, setCurrent] = useState('wlan0');
    return (
      <InterfaceSelector
        interfaces={sampleInterfaces}
        currentInterface={current}
        isWifi
        onChange={setCurrent}
        recommendedWifi="wlan0"
      />
    );
  },
};

export const Warning: Story = {
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
