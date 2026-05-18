import type { Meta, StoryFn, StoryObj } from '@storybook/react-vite';
import type React from 'react';
import { useState } from 'react';
import { Combobox } from './Combobox';

interface Interface {
  name: string;
  speedMbps: number;
}

const interfaces: Interface[] = [
  { name: 'eth0', speedMbps: 1000 },
  { name: 'eth1', speedMbps: 1000 },
  { name: 'wlan0', speedMbps: 866 },
  { name: 'lo', speedMbps: 0 },
];

const meta: Meta<typeof Combobox<Interface>> = {
  title: 'UI/Combobox',
  component: Combobox<Interface>,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component:
          'Generic typeahead select built on cmdk. Same look-and-feel as the command palette so navigation feels consistent.',
      },
    },
  },
  tags: ['autodocs'],
  decorators: [
    (StoryComponent: StoryFn): React.ReactElement => (
      <div class="w-[360px]">
        <StoryComponent />
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

export const WithOptions: Story = {
  render: () => {
    const [value, setValue] = useState<Interface | null>(null);
    return (
      <Combobox<Interface>
        value={value}
        onChange={setValue}
        options={interfaces}
        getKey={(o) => o.name}
        getLabel={(o) =>
          o.speedMbps > 0 ? `${o.name} (${o.speedMbps} Mbps)` : `${o.name} (loopback)`
        }
        placeholder="Select interface…"
        ariaLabel="Network interface selector"
      />
    );
  },
};

export const Preselected: Story = {
  render: () => {
    const [initialValue] = interfaces;
    if (!initialValue) {
      return <div>No interfaces available</div>;
    }
    const [value, setValue] = useState<Interface | null>(initialValue);
    return (
      <Combobox<Interface>
        value={value}
        onChange={setValue}
        options={interfaces}
        getKey={(o) => o.name}
        getLabel={(o) => `${o.name} (${o.speedMbps} Mbps)`}
        ariaLabel="Network interface selector"
      />
    );
  },
};

export const Empty: Story = {
  render: () => {
    const [value, setValue] = useState<Interface | null>(null);
    return (
      <Combobox<Interface>
        value={value}
        onChange={setValue}
        options={[]}
        getKey={(o) => o.name}
        getLabel={(o) => o.name}
        placeholder="No interfaces detected"
        emptyText="No network interfaces available."
        ariaLabel="Network interface selector"
      />
    );
  },
};

export const Disabled: Story = {
  render: () => {
    const [value, setValue] = useState<Interface | null>(null);
    return (
      <Combobox<Interface>
        value={value}
        onChange={setValue}
        options={interfaces}
        getKey={(o) => o.name}
        getLabel={(o) => o.name}
        placeholder="Locked"
        ariaLabel="Network interface selector"
        disabled={true}
      />
    );
  },
};
