import type { Meta, StoryObj } from "@storybook/react-vite";
import { SwitchCard } from "./SwitchCard";

/**
 * SwitchCard displays switch/network device information learned via Layer 2 discovery protocols.
 *
 * Features:
 * - Protocol detection: LLDP, CDP, EDP, FDP
 * - Switch identification: name, management IP, system description
 * - Port information: port ID and description from switch
 * - VLAN support: native VLAN, tagged VLANs, voice VLAN
 * - Dual data display: combines switch info and VLAN data
 * - Status determination: success if switch/VLAN info available
 * - Compact layout with protocol badge
 *
 * This story demonstrates various switch discovery scenarios.
 */
const meta = {
  title: "Cards/SwitchCard",
  component: SwitchCard,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
  decorators: [
    (Story) => (
      <div style={{ width: "380px" }}>
        <Story />
      </div>
    ),
  ],
} satisfies Meta<typeof SwitchCard>;

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Cisco switch discovered via CDP.
 * Shows comprehensive Cisco switch information with VLANs.
 */
export const CiscoSwitch: Story = {
  args: {
    data: {
      protocol: "cdp",
      switchName: "CORE-SW-01",
      portId: "GigabitEthernet1/0/24",
      portDescription: "Server VLAN",
      managementIp: "10.0.1.1",
      systemDescription: "Cisco IOS Software, Catalyst 9300 Series",
    },
    vlanData: {
      nativeVlan: 1,
      taggedVlans: [10, 20, 30, 100],
      voiceVlan: 200,
      configured: {
        enabled: false,
        id: 0,
      },
    },
    loading: false,
  },
};

/**
 * Generic switch via LLDP.
 * Shows LLDP-discovered switch with basic information.
 */
export const GenericLLDP: Story = {
  args: {
    data: {
      protocol: "lldp",
      switchName: "access-switch-02",
      portId: "eth1/12",
      portDescription: "Workstation Port",
      managementIp: "192.168.1.250",
      systemDescription: "Linux 5.15.0-generic",
    },
    vlanData: {
      nativeVlan: 10,
      taggedVlans: [],
      voiceVlan: null,
      configured: {
        enabled: false,
        id: 0,
      },
    },
    loading: false,
  },
};

/**
 * Extreme Networks switch via EDP.
 * Shows EDP protocol discovery.
 */
export const ExtremeEDP: Story = {
  args: {
    data: {
      protocol: "edp",
      switchName: "ExtremeSwitch-Core",
      portId: "1:25",
      portDescription: "Uplink Port",
      managementIp: "10.20.1.1",
      systemDescription: "ExtremeXOS 30.7.1.4",
    },
    vlanData: {
      nativeVlan: 1,
      taggedVlans: [10, 20, 30, 40, 50],
      voiceVlan: null,
      configured: {
        enabled: true,
        id: 30,
      },
    },
    loading: false,
  },
};

/**
 * Voice VLAN configuration.
 * Shows switch port with voice VLAN for IP phones.
 */
export const WithVoiceVLAN: Story = {
  args: {
    data: {
      protocol: "cdp",
      switchName: "ACCESS-SW-FLOOR2",
      portId: "FastEthernet0/12",
      portDescription: "Desk Phone Port",
      managementIp: "192.168.2.250",
      systemDescription: "Cisco Catalyst 2960",
    },
    vlanData: {
      nativeVlan: 10,
      taggedVlans: [],
      voiceVlan: 100,
      configured: {
        enabled: false,
        id: 0,
      },
    },
    loading: false,
  },
};

/**
 * Trunk port with multiple VLANs.
 * Shows port configured for VLAN trunking.
 */
export const TrunkPort: Story = {
  args: {
    data: {
      protocol: "lldp",
      switchName: "DIST-SW-01",
      portId: "TenGigE0/1/0",
      portDescription: "Trunk to Access Layer",
      managementIp: "10.0.0.5",
      systemDescription: "HP ProCurve Switch 5400zl",
    },
    vlanData: {
      nativeVlan: 1,
      taggedVlans: [10, 20, 30, 40, 50, 100, 200, 300],
      voiceVlan: null,
      configured: {
        enabled: false,
        id: 0,
      },
    },
    loading: false,
  },
};

/**
 * Access port (untagged only).
 * Shows simple access port configuration.
 */
export const AccessPort: Story = {
  args: {
    data: {
      protocol: "lldp",
      switchName: "access-switch-03",
      portId: "gi0/5",
      portDescription: "User Port",
      managementIp: "192.168.1.251",
      systemDescription: null,
    },
    vlanData: {
      nativeVlan: 20,
      taggedVlans: [],
      voiceVlan: null,
      configured: {
        enabled: false,
        id: 0,
      },
    },
    loading: false,
  },
};

/**
 * Switch with configured VLAN tag.
 * Shows active VLAN configuration on the interface.
 */
export const WithConfiguredVLAN: Story = {
  args: {
    data: {
      protocol: "cdp",
      switchName: "EDGE-SW-01",
      portId: "GigabitEthernet0/8",
      portDescription: "Guest Network",
      managementIp: "172.16.1.1",
      systemDescription: "Cisco IOS",
    },
    vlanData: {
      nativeVlan: 1,
      taggedVlans: [10, 20, 99],
      voiceVlan: null,
      configured: {
        enabled: true,
        id: 99,
      },
    },
    loading: false,
  },
};

/**
 * Switch without management IP.
 * Shows discovery info without management address.
 */
export const NoManagementIP: Story = {
  args: {
    data: {
      protocol: "lldp",
      switchName: "unmanaged-switch-01",
      portId: "Port 5",
      portDescription: null,
      managementIp: null,
      systemDescription: "Netgear GS308",
    },
    vlanData: null,
    loading: false,
  },
};

/**
 * Minimal switch information.
 * Shows discovery with only basic details available.
 */
export const MinimalInfo: Story = {
  args: {
    data: {
      protocol: "lldp",
      switchName: "switch-minimal",
      portId: "eth0",
      portDescription: null,
      managementIp: null,
      systemDescription: null,
    },
    vlanData: null,
    loading: false,
  },
};

/**
 * No discovery frames received.
 * Shows waiting state when no LLDP/CDP frames detected.
 */
export const NoDiscovery: Story = {
  args: {
    data: null,
    vlanData: null,
    loading: false,
  },
};

/**
 * Listening for discovery frames.
 * Shows loading state during discovery process.
 */
export const Listening: Story = {
  args: {
    data: null,
    vlanData: null,
    loading: true,
  },
};

/**
 * VLAN-only information.
 * Shows VLAN data without switch discovery.
 */
export const VLANOnly: Story = {
  args: {
    data: null,
    vlanData: {
      nativeVlan: 10,
      taggedVlans: [20, 30],
      voiceVlan: 100,
      configured: {
        enabled: true,
        id: 20,
      },
    },
    loading: false,
  },
};

/**
 * Foundry/Brocade FDP.
 * Shows Foundry Discovery Protocol information.
 */
export const FoundryFDP: Story = {
  args: {
    data: {
      protocol: "fdp",
      switchName: "BROCADE-ICX-01",
      portId: "1/1/24",
      portDescription: "Server Connection",
      managementIp: "10.10.1.1",
      systemDescription: "Brocade ICX 7450",
    },
    vlanData: {
      nativeVlan: 1,
      taggedVlans: [10, 20, 30],
      voiceVlan: null,
      configured: {
        enabled: false,
        id: 0,
      },
    },
    loading: false,
  },
};
