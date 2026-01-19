/**
 * SNMPSettings Storybook Stories
 *
 * Demonstrates the SNMP configuration component for managing community strings
 * (v1/v2c) and v3 credentials with authentication and privacy settings.
 *
 * Variants:
 * - Default (empty): No credentials configured
 * - With community strings: v1/v2c community strings only
 * - With v3 credentials: SNMPv3 users with auth/priv
 * - Mixed: Both v2c communities and v3 credentials
 * - All security levels: noAuthNoPriv, authNoPriv, authPriv
 */

import type { Meta, StoryFn, StoryObj } from '@storybook/react-vite';
import type React from 'react';
import { useState } from 'react';
import type { SaveStatus, SNMPSettings as SnmpSettingsType } from '../../../types/settings';
import { SNMPSettings } from './SnmpSettings';

const defaultSettings: SnmpSettingsType = {
  communities: ['public'],
  v3Credentials: [],
  timeout: 5000,
  retries: 2,
  port: 161,
};

const meta: Meta<typeof SNMPSettings> = {
  title: 'Settings/SnmpSettings',
  component: SNMPSettings,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component:
          'SNMP configuration panel for managing v1/v2c community strings and v3 credentials. Supports multiple authentication and privacy protocols with expandable credential forms.',
      },
    },
  },
  tags: ['autodocs'],
  argTypes: {
    snmpStatus: {
      control: 'select',
      options: ['idle', 'saving', 'saved', 'error'],
      description: 'Auto-save status indicator',
    },
  },
  decorators: [
    (StoryComponent: StoryFn): React.ReactElement => (
      <div class="w-[500px] max-h-[700px] overflow-y-auto">
        <StoryComponent />
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Default configuration - single "public" community
 */
export const Default: Story = {
  args: {
    snmpSettings: defaultSettings,
    setSnmpSettings: (): void => {
      // intentionally empty
    },
    snmpStatus: 'idle',
  },
};

/**
 * Multiple community strings configured
 */
export const MultipleCommunities: Story = {
  args: {
    snmpSettings: {
      ...defaultSettings,
      communities: ['public', 'private', 'monitoring'],
    },
    setSnmpSettings: (): void => {
      // intentionally empty
    },
    snmpStatus: 'idle',
  },
};

/**
 * Single v3 credential - no auth no priv
 */
export const V3NoAuthNoPriv: Story = {
  args: {
    snmpSettings: {
      ...defaultSettings,
      v3Credentials: [
        {
          id: '1',
          name: 'Basic User',
          username: 'basicuser',
          authProtocol: '',
          authPassword: '',
          privProtocol: '',
          privPassword: '',
          contextName: '',
          securityLevel: 'noAuthNoPriv',
        },
      ],
    },
    setSnmpSettings: (): void => {
      // intentionally empty
    },
    snmpStatus: 'idle',
  },
};

/**
 * Single v3 credential - auth no priv
 */
export const V3AuthNoPriv: Story = {
  args: {
    snmpSettings: {
      ...defaultSettings,
      v3Credentials: [
        {
          id: '1',
          name: 'Auth User',
          username: 'authuser',
          authProtocol: 'SHA',
          authPassword: 'authpass123',
          privProtocol: '',
          privPassword: '',
          contextName: '',
          securityLevel: 'authNoPriv',
        },
      ],
    },
    setSnmpSettings: (): void => {
      // intentionally empty
    },
    snmpStatus: 'idle',
  },
};

/**
 * Single v3 credential - auth and priv
 */
export const V3AuthPriv: Story = {
  args: {
    snmpSettings: {
      ...defaultSettings,
      v3Credentials: [
        {
          id: '1',
          name: 'Secure User',
          username: 'secureuser',
          authProtocol: 'SHA256',
          authPassword: 'authpass123',
          privProtocol: 'AES',
          privPassword: 'privpass456',
          contextName: 'production',
          securityLevel: 'authPriv',
        },
      ],
    },
    setSnmpSettings: (): void => {
      // intentionally empty
    },
    snmpStatus: 'idle',
  },
};

/**
 * Multiple v3 credentials with different security levels
 */
export const MultipleV3Credentials: Story = {
  args: {
    snmpSettings: {
      ...defaultSettings,
      v3Credentials: [
        {
          id: '1',
          name: 'Read Only',
          username: 'readonly',
          authProtocol: '',
          authPassword: '',
          privProtocol: '',
          privPassword: '',
          contextName: '',
          securityLevel: 'noAuthNoPriv',
        },
        {
          id: '2',
          name: 'Monitoring',
          username: 'monitor',
          authProtocol: 'SHA',
          authPassword: 'monitorpass',
          privProtocol: '',
          privPassword: '',
          contextName: '',
          securityLevel: 'authNoPriv',
        },
        {
          id: '3',
          name: 'Admin',
          username: 'admin',
          authProtocol: 'SHA256',
          authPassword: 'adminauth',
          privProtocol: 'AES256',
          privPassword: 'adminpriv',
          contextName: 'admin',
          securityLevel: 'authPriv',
        },
      ],
    },
    setSnmpSettings: (): void => {
      // intentionally empty
    },
    snmpStatus: 'idle',
  },
};

/**
 * Mixed configuration - both v2c and v3
 */
export const MixedConfiguration: Story = {
  args: {
    snmpSettings: {
      communities: ['public', 'monitoring'],
      v3Credentials: [
        {
          id: '1',
          name: 'v3 User',
          username: 'snmpuser',
          authProtocol: 'SHA',
          authPassword: 'authpass',
          privProtocol: 'AES',
          privPassword: 'privpass',
          contextName: '',
          securityLevel: 'authPriv',
        },
      ],
      timeout: 5000,
      retries: 2,
      port: 161,
    },
    setSnmpSettings: (): void => {
      // intentionally empty
    },
    snmpStatus: 'idle',
  },
};

/**
 * Custom port and timeout
 */
export const CustomPortTimeout: Story = {
  args: {
    snmpSettings: {
      ...defaultSettings,
      port: 1161,
      timeout: 10000,
      retries: 5,
    },
    setSnmpSettings: (): void => {
      // intentionally empty
    },
    snmpStatus: 'idle',
  },
};

/**
 * Empty configuration - no credentials
 */
export const Empty: Story = {
  args: {
    snmpSettings: {
      communities: [],
      v3Credentials: [],
      timeout: 5000,
      retries: 2,
      port: 161,
    },
    setSnmpSettings: (): void => {
      // intentionally empty
    },
    snmpStatus: 'idle',
  },
};

/**
 * Saving state
 */
export const Saving: Story = {
  args: {
    snmpSettings: {
      ...defaultSettings,
      communities: ['public', 'private'],
    },
    setSnmpSettings: (): void => {
      // intentionally empty
    },
    snmpStatus: 'saving',
  },
};

/**
 * Interactive SNMP settings - fully functional CRUD
 */
export const Interactive: Story = {
  render: function interactiveStory() {
    const [snmpSettings, setSnmpSettings] = useState<SnmpSettingsType>(defaultSettings);
    const [status, setStatus] = useState<SaveStatus>('idle');

    const handleSetSnmpSettings = (updater: React.SetStateAction<SnmpSettingsType>): void => {
      setSnmpSettings(updater);
      setStatus('saving');

      setTimeout(() => {
        setStatus('saved');
        setTimeout(() => {
          setStatus('idle');
        }, 2000);
      }, 800);
    };

    return (
      <SNMPSettings
        snmpSettings={snmpSettings}
        setSnmpSettings={handleSetSnmpSettings}
        snmpStatus={status}
      />
    );
  },
};
