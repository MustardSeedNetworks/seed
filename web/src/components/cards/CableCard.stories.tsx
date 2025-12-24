import type { Meta, StoryObj } from "@storybook/react-vite";
import { CableCard } from "./CableCard";

/**
 * CableCard displays Ethernet cable test results using Time Domain Reflectometry (TDR).
 *
 * Features:
 * - Cable condition detection: OK, Open circuit, Short circuit, Impedance mismatch
 * - Cable length measurement in meters
 * - Fault detection and listing
 * - Color-coded status: green (ok), red (open/short), yellow (impedance), gray (unknown)
 * - Graceful handling of unsupported NICs
 * - TDR diagnostics for physical layer troubleshooting
 *
 * This story demonstrates various cable test results.
 */
const meta = {
  title: "Cards/CableCard",
  component: CableCard,
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
} satisfies Meta<typeof CableCard>;

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Cable test showing healthy cable.
 * Green status with measured length.
 */
export const CableOK: Story = {
  args: {
    data: {
      supported: true,
      length: 15.2,
      status: "ok",
      faults: [],
    },
    loading: false,
    unitSystem: "sae", // Display in feet
  },
};

/**
 * Short cable in good condition.
 * Shows shorter measured length.
 */
export const ShortCableOK: Story = {
  args: {
    data: {
      supported: true,
      length: 2.8,
      status: "ok",
      faults: [],
    },
    loading: false,
  },
};

/**
 * Long cable run still within spec.
 * Shows longer measured length approaching limits.
 */
export const LongCableOK: Story = {
  args: {
    data: {
      supported: true,
      length: 87.5,
      status: "ok",
      faults: [],
    },
    loading: false,
  },
};

/**
 * Open circuit detected.
 * Red status indicating cable is disconnected or broken.
 */
export const OpenCircuit: Story = {
  args: {
    data: {
      supported: true,
      length: 12.4,
      status: "open",
      faults: ["Open circuit detected at 12.4m"],
    },
    loading: false,
  },
};

/**
 * Short circuit detected.
 * Red status indicating cable has shorted conductors.
 */
export const ShortCircuit: Story = {
  args: {
    data: {
      supported: true,
      length: 8.7,
      status: "short",
      faults: ["Short circuit detected at 8.7m"],
    },
    loading: false,
  },
};

/**
 * Impedance mismatch detected.
 * Yellow warning status for cable quality issues.
 */
export const ImpedanceMismatch: Story = {
  args: {
    data: {
      supported: true,
      length: 22.1,
      status: "impedance_mismatch",
      faults: ["Impedance mismatch detected", "Possible cable damage or bend"],
    },
    loading: false,
  },
};

/**
 * Multiple faults detected.
 * Shows cable with several issues.
 */
export const MultipleFaults: Story = {
  args: {
    data: {
      supported: true,
      length: 45.3,
      status: "open",
      faults: [
        "Open circuit at 45.3m",
        "Impedance variations detected",
        "Possible water damage",
      ],
    },
    loading: false,
  },
};

/**
 * Unknown cable status.
 * Gray status when test results are inconclusive.
 */
export const UnknownStatus: Story = {
  args: {
    data: {
      supported: true,
      length: null,
      status: "unknown",
      faults: ["Unable to determine cable status"],
    },
    loading: false,
  },
};

/**
 * TDR not supported by NIC.
 * Shows appropriate message for unsupported hardware.
 */
export const NotSupported: Story = {
  args: {
    data: {
      supported: false,
      length: null,
      status: "unknown",
      faults: [],
    },
    loading: false,
  },
};

/**
 * Testing in progress.
 * Shows loading state during cable test.
 */
export const Testing: Story = {
  args: {
    data: null,
    loading: true,
  },
};

/**
 * No data available.
 * Initial state before any test has been run.
 */
export const NoData: Story = {
  args: {
    data: null,
    loading: false,
  },
};

/**
 * Very short cable (patch cable).
 * Shows sub-meter measurement.
 */
export const PatchCable: Story = {
  args: {
    data: {
      supported: true,
      length: 0.5,
      status: "ok",
      faults: [],
    },
    loading: false,
  },
};

/**
 * Cable at maximum recommended length.
 * Shows cable near 100m Ethernet limit.
 */
export const MaxLengthCable: Story = {
  args: {
    data: {
      supported: true,
      length: 98.2,
      status: "ok",
      faults: [],
    },
    loading: false,
    unitSystem: "sae",
  },
};

/**
 * Cable displayed in metric units.
 * Shows same cable with metric unit system.
 */
export const CableMetricUnits: Story = {
  args: {
    data: {
      supported: true,
      length: 15.2,
      status: "ok",
      faults: [],
    },
    loading: false,
    unitSystem: "metric", // Display in meters
  },
};
