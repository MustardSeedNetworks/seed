import type { Meta, StoryObj } from '@storybook/react-vite';
import { useState } from 'react';
import type { SubnetConfig } from '../../../../types/settings';
import { SubnetManager } from './SubnetManager';

const meta = {
  title: 'Settings/SubnetManager',
  component: SubnetManager,
} satisfies Meta<typeof SubnetManager>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: () => {
    const [subnets, setSubnets] = useState<SubnetConfig[]>([
      { cidr: '192.168.1.0/24', name: 'Office LAN', enabled: true },
      { cidr: '10.0.0.0/24', name: 'Lab', enabled: false },
    ]);
    const [newSubnetCidr, setNewSubnetCidr] = useState('');
    const [newSubnetName, setNewSubnetName] = useState('');
    const [subnetError, setSubnetError] = useState<string | null>(null);

    return (
      <SubnetManager
        subnets={subnets}
        subnetsStatus="saved"
        newSubnetCidr={newSubnetCidr}
        setNewSubnetCidr={setNewSubnetCidr}
        newSubnetName={newSubnetName}
        setNewSubnetName={setNewSubnetName}
        subnetError={subnetError}
        setSubnetError={setSubnetError}
        addSubnet={() => {
          if (!newSubnetCidr.trim()) {
            setSubnetError('CIDR required');
            return;
          }
          setSubnets((prev) => [
            ...prev,
            {
              cidr: newSubnetCidr.trim(),
              name: newSubnetName.trim(),
              enabled: true,
            },
          ]);
          setNewSubnetCidr('');
          setNewSubnetName('');
          setSubnetError(null);
        }}
        toggleSubnet={(cidr, enabled) =>
          setSubnets((prev) => prev.map((s) => (s.cidr === cidr ? { ...s, enabled } : s)))
        }
        deleteSubnet={(cidr) => setSubnets((prev) => prev.filter((s) => s.cidr !== cidr))}
      />
    );
  },
};
