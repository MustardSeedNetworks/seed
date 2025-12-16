import type { Meta, StoryObj } from "@storybook/react-vite";
import {
  // Card header icons
  Activity,
  Cable,
  Globe,
  Network,
  Router,
  Search,
  ScanSearch,
  Server,
  Wifi,
  Gauge,
  HeartPulse,
  Settings,
  // Status icons
  CheckCircle,
  XCircle,
  AlertTriangle,
  AlertCircle,
  Info,
  // Device type icons
  Monitor,
  Smartphone,
  Printer,
  HardDrive,
  Laptop,
  Tablet,
  Tv,
  // Navigation/action icons
  ChevronDown,
  ChevronRight,
  ChevronUp,
  ChevronLeft,
  X,
  Menu,
  RefreshCw,
  Download,
  Upload,
  Play,
  Pause,
  RotateCcw,
  Loader,
  // UI icons
  Eye,
  EyeOff,
  Copy,
  Check,
  Trash2,
  Edit,
  Plus,
  Minus,
  Filter,
  // Network specific
  Unplug,
  PlugZap,
  Signal,
  SignalHigh,
  SignalLow,
  SignalMedium,
  SignalZero,
  Zap,
  // Service/protocol icons
  Terminal,
  FileText,
  Mail,
  Database,
  Container,
  ShieldOff,
  // System/misc
  Cpu,
  MemoryStick,
  HardDriveDownload,
  Clock,
  Timer,
  Calendar,
  User,
  Lock,
  Unlock,
  Key,
  Shield,
  ShieldCheck,
  ShieldAlert,
  // Layout
  LayoutDashboard,
  List,
  Grid3X3,
  Columns3,
  // Settings icons
  SlidersHorizontal,
  Palette,
} from "./Icons";
import { ICON_SIZES } from "./iconConfig";
import { spacing, button, section, icon as iconTheme, layout } from "../../styles/theme";

/**
 * Icon Library showcases all available icons re-exported from lucide-react.
 *
 * This centralized module provides:
 * - Consistent icon naming across the application
 * - Single source of truth for icon usage
 * - Easy migration if switching icon libraries
 * - Standardized size presets
 *
 * Always import icons from this module instead of directly from lucide-react.
 */
const meta: Meta = {
  title: "UI/Icons",
  parameters: {
    layout: "padded",
    docs: {
      description: {
        component:
          "Centralized icon library re-exporting Lucide React icons with consistent naming and size presets.",
      },
    },
  },
  tags: ["autodocs"],
};

export default meta;
type Story = StoryObj;

/**
 * All available icons organized by category.
 */
export const AllIcons: Story = {
  render: () => (
    <div className={`${section.spacing.spacious} ${spacing.pad.default}`}>
      <IconCategory title="Card Header Icons">
        <IconItem icon={<Activity />} name="Activity" />
        <IconItem icon={<Cable />} name="Cable" />
        <IconItem icon={<Globe />} name="Globe" />
        <IconItem icon={<Network />} name="Network" />
        <IconItem icon={<Router />} name="Router" />
        <IconItem icon={<Search />} name="Search" />
        <IconItem icon={<ScanSearch />} name="ScanSearch" />
        <IconItem icon={<Server />} name="Server" />
        <IconItem icon={<Wifi />} name="Wifi" />
        <IconItem icon={<Gauge />} name="Gauge" />
        <IconItem icon={<HeartPulse />} name="HeartPulse" />
        <IconItem icon={<Settings />} name="Settings" />
      </IconCategory>

      <IconCategory title="Status Icons">
        <IconItem icon={<CheckCircle className="text-status-success" />} name="CheckCircle" />
        <IconItem icon={<XCircle className="text-status-error" />} name="XCircle" />
        <IconItem icon={<AlertTriangle className="text-status-warning" />} name="AlertTriangle" />
        <IconItem icon={<AlertCircle className="text-status-info" />} name="AlertCircle" />
        <IconItem icon={<Info className="text-brand-primary" />} name="Info" />
      </IconCategory>

      <IconCategory title="Device Type Icons">
        <IconItem icon={<Monitor />} name="Monitor" />
        <IconItem icon={<Smartphone />} name="Smartphone" />
        <IconItem icon={<Printer />} name="Printer" />
        <IconItem icon={<HardDrive />} name="HardDrive" />
        <IconItem icon={<Laptop />} name="Laptop" />
        <IconItem icon={<Tablet />} name="Tablet" />
        <IconItem icon={<Tv />} name="Tv" />
      </IconCategory>

      <IconCategory title="Navigation Icons">
        <IconItem icon={<ChevronDown />} name="ChevronDown" />
        <IconItem icon={<ChevronUp />} name="ChevronUp" />
        <IconItem icon={<ChevronLeft />} name="ChevronLeft" />
        <IconItem icon={<ChevronRight />} name="ChevronRight" />
        <IconItem icon={<X />} name="X" />
        <IconItem icon={<Menu />} name="Menu" />
      </IconCategory>

      <IconCategory title="Action Icons">
        <IconItem icon={<RefreshCw />} name="RefreshCw" />
        <IconItem icon={<Download />} name="Download" />
        <IconItem icon={<Upload />} name="Upload" />
        <IconItem icon={<Play />} name="Play" />
        <IconItem icon={<Pause />} name="Pause" />
        <IconItem icon={<RotateCcw />} name="RotateCcw" />
        <IconItem icon={<Loader className="animate-spin" />} name="Loader" />
      </IconCategory>

      <IconCategory title="UI Icons">
        <IconItem icon={<Eye />} name="Eye" />
        <IconItem icon={<EyeOff />} name="EyeOff" />
        <IconItem icon={<Copy />} name="Copy" />
        <IconItem icon={<Check />} name="Check" />
        <IconItem icon={<Trash2 />} name="Trash2" />
        <IconItem icon={<Edit />} name="Edit" />
        <IconItem icon={<Plus />} name="Plus" />
        <IconItem icon={<Minus />} name="Minus" />
        <IconItem icon={<Filter />} name="Filter" />
      </IconCategory>

      <IconCategory title="Network Icons">
        <IconItem icon={<Unplug />} name="Unplug" />
        <IconItem icon={<PlugZap />} name="PlugZap" />
        <IconItem icon={<Signal />} name="Signal" />
        <IconItem icon={<SignalHigh />} name="SignalHigh" />
        <IconItem icon={<SignalMedium />} name="SignalMedium" />
        <IconItem icon={<SignalLow />} name="SignalLow" />
        <IconItem icon={<SignalZero />} name="SignalZero" />
        <IconItem icon={<Zap />} name="Zap" />
      </IconCategory>

      <IconCategory title="Service/Protocol Icons">
        <IconItem icon={<Terminal />} name="Terminal" />
        <IconItem icon={<FileText />} name="FileText" />
        <IconItem icon={<Mail />} name="Mail" />
        <IconItem icon={<Database />} name="Database" />
        <IconItem icon={<Container />} name="Container" />
        <IconItem icon={<ShieldOff />} name="ShieldOff" />
      </IconCategory>

      <IconCategory title="System Icons">
        <IconItem icon={<Cpu />} name="Cpu" />
        <IconItem icon={<MemoryStick />} name="MemoryStick" />
        <IconItem icon={<HardDriveDownload />} name="HardDriveDownload" />
        <IconItem icon={<Clock />} name="Clock" />
        <IconItem icon={<Timer />} name="Timer" />
        <IconItem icon={<Calendar />} name="Calendar" />
        <IconItem icon={<User />} name="User" />
      </IconCategory>

      <IconCategory title="Security Icons">
        <IconItem icon={<Lock />} name="Lock" />
        <IconItem icon={<Unlock />} name="Unlock" />
        <IconItem icon={<Key />} name="Key" />
        <IconItem icon={<Shield />} name="Shield" />
        <IconItem icon={<ShieldCheck className="text-status-success" />} name="ShieldCheck" />
        <IconItem icon={<ShieldAlert className="text-status-warning" />} name="ShieldAlert" />
      </IconCategory>

      <IconCategory title="Layout Icons">
        <IconItem icon={<LayoutDashboard />} name="LayoutDashboard" />
        <IconItem icon={<List />} name="List" />
        <IconItem icon={<Grid3X3 />} name="Grid3X3" />
        <IconItem icon={<Columns3 />} name="Columns3" />
      </IconCategory>

      <IconCategory title="Settings Icons">
        <IconItem icon={<SlidersHorizontal />} name="SlidersHorizontal" />
        <IconItem icon={<Palette />} name="Palette" />
      </IconCategory>
    </div>
  ),
};

/**
 * Icon size presets demonstration.
 */
export const Sizes: Story = {
  render: () => (
    <div className={`${section.spacing.comfortable} ${spacing.pad.default}`}>
      <h3 className="heading-3 text-text-primary">Icon Size Presets</h3>
      <div className={`flex items-end ${spacing.gap.spacious}`}>
        {(Object.keys(ICON_SIZES) as Array<keyof typeof ICON_SIZES>).map((size) => (
          <div key={size} className="text-center">
            <Activity className={`${ICON_SIZES[size]} text-brand-primary mx-auto`} />
            <p className={`body-small text-text-muted ${spacing.margin.top.inline}`}>{size}</p>
            <p className="caption text-text-muted">{ICON_SIZES[size]}</p>
          </div>
        ))}
      </div>
    </div>
  ),
  parameters: {
    docs: {
      description: {
        story:
          "Available size presets: xs (w-3 h-3), sm (w-4 h-4), md (w-5 h-5), lg (w-6 h-6), xl (w-8 h-8).",
      },
    },
  },
};

/**
 * Status icons with semantic colors.
 */
export const StatusIcons: Story = {
  render: () => (
    <div className={`${section.spacing.default} ${spacing.pad.default}`}>
      <h3 className={`heading-3 text-text-primary ${spacing.margin.bottom.content}`}>
        Status Icons with Semantic Colors
      </h3>
      <div className={`grid grid-cols-2 md:grid-cols-4 ${spacing.gap.comfortable}`}>
        <StatusExample
          icon={<CheckCircle className={`${iconTheme.size.lg} text-status-success`} />}
          label="Success"
          description="Operation completed"
        />
        <StatusExample
          icon={<AlertTriangle className={`${iconTheme.size.lg} text-status-warning`} />}
          label="Warning"
          description="Needs attention"
        />
        <StatusExample
          icon={<XCircle className={`${iconTheme.size.lg} text-status-error`} />}
          label="Error"
          description="Operation failed"
        />
        <StatusExample
          icon={<Info className={`${iconTheme.size.lg} text-status-info`} />}
          label="Info"
          description="Additional info"
        />
      </div>
    </div>
  ),
};

/**
 * Network status indicators.
 */
export const NetworkStatus: Story = {
  render: () => (
    <div className={`${section.spacing.default} ${spacing.pad.default}`}>
      <h3 className={`heading-3 text-text-primary ${spacing.margin.bottom.content}`}>
        Network Status Indicators
      </h3>
      <div className={`flex ${spacing.gap.spacious}`}>
        <div className="text-center">
          <div
            className={`${iconTheme.size["2xl"]} rounded-full bg-status-success/10 flex items-center justify-center ${spacing.margin.bottom.inline}`}
          >
            <Wifi className={`${iconTheme.size.lg} text-status-success`} />
          </div>
          <p className="body-small">Connected</p>
        </div>
        <div className="text-center">
          <div
            className={`${iconTheme.size["2xl"]} rounded-full bg-status-warning/10 flex items-center justify-center ${spacing.margin.bottom.inline}`}
          >
            <SignalLow className={`${iconTheme.size.lg} text-status-warning`} />
          </div>
          <p className="body-small">Weak Signal</p>
        </div>
        <div className="text-center">
          <div
            className={`${iconTheme.size["2xl"]} rounded-full bg-status-error/10 flex items-center justify-center ${spacing.margin.bottom.inline}`}
          >
            <Unplug className={`${iconTheme.size.lg} text-status-error`} />
          </div>
          <p className="body-small">Disconnected</p>
        </div>
        <div className="text-center">
          <div
            className={`${iconTheme.size["2xl"]} rounded-full bg-status-info/10 flex items-center justify-center ${spacing.margin.bottom.inline}`}
          >
            <Loader className={`${iconTheme.size.lg} text-status-info animate-spin`} />
          </div>
          <p className="body-small">Connecting</p>
        </div>
      </div>
    </div>
  ),
};

/**
 * Device type icons for network discovery.
 */
export const DeviceTypes: Story = {
  render: () => (
    <div className={`${section.spacing.default} ${spacing.pad.default}`}>
      <h3 className={`heading-3 text-text-primary ${spacing.margin.bottom.content}`}>
        Device Type Icons
      </h3>
      <div className={`grid grid-cols-3 md:grid-cols-7 ${spacing.gap.comfortable}`}>
        <DeviceExample icon={<Monitor />} name="Desktop" />
        <DeviceExample icon={<Laptop />} name="Laptop" />
        <DeviceExample icon={<Smartphone />} name="Phone" />
        <DeviceExample icon={<Tablet />} name="Tablet" />
        <DeviceExample icon={<Printer />} name="Printer" />
        <DeviceExample icon={<Tv />} name="TV" />
        <DeviceExample icon={<HardDrive />} name="NAS" />
      </div>
    </div>
  ),
};

/**
 * Button icons with different states.
 */
export const ButtonIcons: Story = {
  render: () => (
    <div className={`${section.spacing.default} ${spacing.pad.default}`}>
      <h3 className={`heading-3 text-text-primary ${spacing.margin.bottom.content}`}>
        Button Icons
      </h3>
      <div className={`flex flex-wrap ${spacing.gap.default}`}>
        <button
          className={`${layout.inline.default} ${button.size.md} bg-brand-primary text-text-inverse rounded-lg hover:bg-brand-primary/90`}
        >
          <Play className={iconTheme.size.sm} />
          Run Test
        </button>
        <button
          className={`${layout.inline.default} ${button.size.md} bg-surface-raised border border-surface-border rounded-lg hover:bg-surface-hover`}
        >
          <RefreshCw className={iconTheme.size.sm} />
          Refresh
        </button>
        <button
          className={`${layout.inline.default} ${button.size.md} bg-status-success/10 text-status-success border border-status-success/20 rounded-lg hover:bg-status-success/20`}
        >
          <Download className={iconTheme.size.sm} />
          Export
        </button>
        <button
          className={`${layout.inline.default} ${button.size.md} bg-status-error/10 text-status-error border border-status-error/20 rounded-lg hover:bg-status-error/20`}
        >
          <Trash2 className={iconTheme.size.sm} />
          Delete
        </button>
      </div>
    </div>
  ),
};

/**
 * Card header icon examples.
 */
export const CardHeaders: Story = {
  render: () => (
    <div className={`${section.spacing.default} ${spacing.pad.default}`}>
      <h3 className={`heading-3 text-text-primary ${spacing.margin.bottom.content}`}>
        Card Header Icons
      </h3>
      <div className={`grid grid-cols-2 md:grid-cols-4 ${spacing.gap.comfortable}`}>
        <CardHeaderExample icon={<Activity />} title="Link Status" />
        <CardHeaderExample icon={<Wifi />} title="WiFi" />
        <CardHeaderExample icon={<Cable />} title="Cable Test" />
        <CardHeaderExample icon={<Network />} title="Network" />
        <CardHeaderExample icon={<Server />} title="Gateway" />
        <CardHeaderExample icon={<Gauge />} title="Performance" />
        <CardHeaderExample icon={<ScanSearch />} title="Discovery" />
        <CardHeaderExample icon={<HeartPulse />} title="Health Checks" />
      </div>
    </div>
  ),
};

// Helper components for stories
function IconCategory({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div>
      <h4
        className={`heading-4 text-text-primary ${spacing.margin.bottom.heading} border-b border-surface-border ${spacing.margin.bottom.inline}`}
      >
        {title}
      </h4>
      <div className={`grid grid-cols-4 md:grid-cols-6 lg:grid-cols-12 ${spacing.gap.comfortable}`}>
        {children}
      </div>
    </div>
  );
}

function IconItem({ icon, name }: { icon: React.ReactNode; name: string }) {
  return (
    <div
      className={`flex flex-col items-center text-center ${spacing.pad.sm} rounded-lg hover:bg-surface-hover`}
    >
      <span className={`text-text-primary ${iconTheme.size.md}`}>{icon}</span>
      <span className={`caption text-text-muted ${spacing.margin.top.inline} truncate w-full`}>
        {name}
      </span>
    </div>
  );
}

function StatusExample({
  icon,
  label,
  description,
}: {
  icon: React.ReactNode;
  label: string;
  description: string;
}) {
  return (
    <div
      className={`flex items-center ${spacing.gap.default} ${spacing.pad.sm} rounded-lg bg-surface-raised border border-surface-border`}
    >
      {icon}
      <div>
        <p className="body-small font-medium text-text-primary">{label}</p>
        <p className="caption text-text-muted">{description}</p>
      </div>
    </div>
  );
}

function DeviceExample({ icon, name }: { icon: React.ReactNode; name: string }) {
  return (
    <div
      className={`flex flex-col items-center ${spacing.pad.sm} rounded-lg bg-surface-raised border border-surface-border`}
    >
      <span className={`${iconTheme.size.xl} text-text-secondary`}>{icon}</span>
      <span className={`body-small text-text-muted ${spacing.margin.top.inline}`}>{name}</span>
    </div>
  );
}

function CardHeaderExample({ icon, title }: { icon: React.ReactNode; title: string }) {
  return (
    <div
      className={`${spacing.pad.default} rounded-lg bg-surface-raised border border-surface-border`}
    >
      <div className={`flex items-center ${spacing.gap.compact}`}>
        <span className={`${iconTheme.size.md} text-brand-primary`}>{icon}</span>
        <span className="body font-medium text-text-primary">{title}</span>
      </div>
    </div>
  );
}
