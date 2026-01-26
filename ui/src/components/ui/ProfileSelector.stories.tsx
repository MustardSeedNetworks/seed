import type { Meta, StoryObj } from '@storybook/react-vite';
import { useState } from 'react';
import type { Profile } from '../../types/profile';
import { ProfileSelector } from './ProfileSelector';

const profiles: Profile[] = [
  {
    id: 'default',
    name: 'Default',
    description: 'Default profile',
    config: {},
    isDefault: true,
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
  },
  {
    id: 'client-01',
    name: 'Acme HQ',
    description: 'Primary office profile',
    config: {},
    isDefault: false,
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
  },
];

const meta = {
  title: 'UI/ProfileSelector',
  component: ProfileSelector,
} satisfies Meta<typeof ProfileSelector>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: () => {
    const [active, setActive] = useState<Profile | null>(profiles[0]);
    return (
      <ProfileSelector
        profiles={profiles}
        activeProfile={active}
        onSwitch={async (id) => {
          const next = profiles.find((p) => p.id === id) ?? null;
          setActive(next);
          return true;
        }}
        onManageClick={() => {}}
      />
    );
  },
};

export const Loading: Story = {
  render: () => (
    <ProfileSelector
      profiles={profiles}
      activeProfile={profiles[0]}
      onSwitch={async () => true}
      loading
    />
  ),
};

export const Disabled: Story = {
  render: () => (
    <ProfileSelector
      profiles={profiles}
      activeProfile={profiles[0]}
      onSwitch={async () => true}
      disabled
    />
  ),
};
