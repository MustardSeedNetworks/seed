import type { Meta, StoryObj } from '@storybook/react-vite';
import type { Profile } from '../../types/profile';
import { ProfileEditor } from './ProfileEditor';

const sampleProfile = {
  id: 'default',
  name: 'Default',
  description: 'Default profile',
  config: { notes: 'Initial profile setup.' },
  isDefault: true,
  createdAt: new Date().toISOString(),
  updatedAt: new Date().toISOString(),
  // biome-ignore lint/style/useNamingConvention: ProfileEditor reads snake_case from API
  is_default: true,
} as Profile;

const meta = {
  title: 'Profiles/ProfileEditor',
  component: ProfileEditor,
  parameters: { layout: 'fullscreen' },
} satisfies Meta<typeof ProfileEditor>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Create: Story = {
  args: {
    profile: null,
    onSave: async () => {},
    onCancel: () => {},
    isLoading: false,
  },
};

export const Edit: Story = {
  args: {
    profile: sampleProfile,
    onSave: async () => {},
    onCancel: () => {},
    isLoading: false,
  },
};
