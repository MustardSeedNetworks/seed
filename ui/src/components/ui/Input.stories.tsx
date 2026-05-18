import type { Meta, StoryFn, StoryObj } from '@storybook/react-vite';
import { Lock, Mail, Search } from 'lucide-react';
import type React from 'react';
import { useState } from 'react';
import {
  Checkbox,
  FormGroup,
  FormSection,
  Input,
  SearchInput,
  Select,
  Textarea,
  Toggle,
} from './Input';

const meta: Meta<typeof Input> = {
  title: 'UI/Input',
  component: Input,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component:
          'Form primitives: text input, textarea, select, checkbox, toggle, and search input. All share the same focus ring and color tokens.',
      },
    },
  },
  tags: ['autodocs'],
  argTypes: {
    label: { control: 'text' },
    placeholder: { control: 'text' },
    error: { control: 'text' },
    hint: { control: 'text' },
    disabled: { control: 'boolean' },
  },
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

export const Default: Story = {
  args: {
    label: 'Server hostname',
    placeholder: 'example.local',
  },
};

export const WithHint: Story = {
  args: {
    label: 'IP address',
    placeholder: '192.168.1.1',
    hint: 'Enter the IPv4 address of the gateway.',
  },
};

export const WithError: Story = {
  args: {
    label: 'Email',
    placeholder: 'you@example.com',
    error: 'Email is required.',
    defaultValue: '',
  },
};

export const WithLeftIcon: Story = {
  args: {
    label: 'Email',
    placeholder: 'you@example.com',
    leftIcon: <Mail class="h-4 w-4" />,
  },
};

export const Disabled: Story = {
  args: {
    label: 'Locked field',
    defaultValue: 'cannot edit',
    disabled: true,
    leftIcon: <Lock class="h-4 w-4" />,
  },
};

export const TextareaExample: Story = {
  name: 'Textarea',
  render: () => (
    <Textarea
      label="Notes"
      placeholder="Describe the network topology…"
      hint="Markdown supported."
    />
  ),
};

export const SelectExample: Story = {
  name: 'Select',
  render: () => (
    <Select
      label="Test protocol"
      placeholder="Choose protocol"
      options={[
        { value: 'tcp', label: 'TCP' },
        { value: 'udp', label: 'UDP' },
        { value: 'icmp', label: 'ICMP' },
      ]}
    />
  ),
};

export const CheckboxExample: Story = {
  name: 'Checkbox',
  render: () => {
    const [checked, setChecked] = useState(false);
    return (
      <Checkbox
        label="Enable deep packet inspection"
        description="Inspects TCP/UDP headers in addition to flow stats."
        checked={checked}
        onChange={(e) => setChecked(e.target.checked)}
      />
    );
  },
};

export const ToggleExample: Story = {
  name: 'Toggle',
  render: () => {
    const [enabled, setEnabled] = useState(true);
    return (
      <Toggle
        label="Auto-refresh dashboard"
        description="Polls the server every 5 seconds."
        checked={enabled}
        onChange={(e) => setEnabled(e.target.checked)}
      />
    );
  },
};

export const SearchInputExample: Story = {
  name: 'SearchInput',
  render: () => {
    const [query, setQuery] = useState('');
    return (
      <SearchInput
        label="Find host"
        placeholder="Search by hostname or IP…"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        onClear={() => setQuery('')}
        leftIcon={<Search class="h-4 w-4" />}
      />
    );
  },
};

export const FormLayout: Story = {
  name: 'Form layout',
  render: () => (
    <FormSection
      title="Network profile"
      description="DHCP, DNS, and ARP settings for this interface."
    >
      <FormGroup>
        <Input label="Profile name" placeholder="Office gateway" />
        <Input label="Gateway IP" placeholder="192.168.1.1" hint="IPv4 only." />
        <Select
          label="Mode"
          options={[
            { value: 'dhcp', label: 'DHCP' },
            { value: 'static', label: 'Static' },
          ]}
        />
      </FormGroup>
    </FormSection>
  ),
};
