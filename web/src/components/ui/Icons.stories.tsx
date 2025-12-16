import type { Meta, StoryObj } from '@storybook/react-vite';
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
  ICON_SIZES,
} from './Icons';

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
  title: 'UI/Icons',
  parameters: {
    layout: 'padded',
    docs: {
      description: {
        component:
          'Centralized icon library re-exporting Lucide React icons with consistent naming and size presets.',
      },
    },
  },
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj;

/**
 * All available icons organized by category.
 */
export const AllIcons: Story = {
  render: () => (
    <div className="space-y-8 p-4">
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
    <div className="space-y-6 p-4">
      <h3 className="heading-3 text-text-primary">Icon Size Presets</h3>
      <div className="flex items-end gap-8">
        {(Object.keys(ICON_SIZES) as Array<keyof typeof ICON_SIZES>).map((size) => (
          <div key={size} className="text-center">
            <Activity className={`${ICON_SIZES[size]} text-brand-primary mx-auto`} />
            <p className="body-small text-text-muted mt-2">{size}</p>
            <p className="caption text-text-muted">{ICON_SIZES[size]}</p>
          </div>
        ))}
      </div>
    </div>
  ),
  parameters: {
    docs: {
      description: {
        story: 'Available size presets: xs (w-3 h-3), sm (w-4 h-4), md (w-5 h-5), lg (w-6 h-6), xl (w-8 h-8).',
      },
    },
  },
};

/**
 * Status icons with semantic colors.
 */
export const StatusIcons: Story = {
  render: () => (
    <div className="space-y-4 p-4">
      <h3 className="heading-3 text-text-primary mb-4">Status Icons with Semantic Colors</h3>
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <StatusExample
          icon={<CheckCircle className="w-6 h-6 text-status-success" />}
          label="Success"
          description="Operation completed"
        />
        <StatusExample
          icon={<AlertTriangle className="w-6 h-6 text-status-warning" />}
          label="Warning"
          description="Needs attention"
        />
        <StatusExample
          icon={<XCircle className="w-6 h-6 text-status-error" />}
          label="Error"
          description="Operation failed"
        />
        <StatusExample
          icon={<Info className="w-6 h-6 text-status-info" />}
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
    <div className="space-y-4 p-4">
      <h3 className="heading-3 text-text-primary mb-4">Network Status Indicators</h3>
      <div className="flex gap-6">
        <div className="text-center">
          <div className="w-12 h-12 rounded-full bg-status-success/10 flex items-center justify-center mb-2">
            <Wifi className="w-6 h-6 text-status-success" />
          </div>
          <p className="body-small">Connected</p>
        </div>
        <div className="text-center">
          <div className="w-12 h-12 rounded-full bg-status-warning/10 flex items-center justify-center mb-2">
            <SignalLow className="w-6 h-6 text-status-warning" />
          </div>
          <p className="body-small">Weak Signal</p>
        </div>
        <div className="text-center">
          <div className="w-12 h-12 rounded-full bg-status-error/10 flex items-center justify-center mb-2">
            <Unplug className="w-6 h-6 text-status-error" />
          </div>
          <p className="body-small">Disconnected</p>
        </div>
        <div className="text-center">
          <div className="w-12 h-12 rounded-full bg-status-info/10 flex items-center justify-center mb-2">
            <Loader className="w-6 h-6 text-status-info animate-spin" />
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
    <div className="space-y-4 p-4">
      <h3 className="heading-3 text-text-primary mb-4">Device Type Icons</h3>
      <div className="grid grid-cols-3 md:grid-cols-7 gap-4">
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
    <div className="space-y-4 p-4">
      <h3 className="heading-3 text-text-primary mb-4">Button Icons</h3>
      <div className="flex flex-wrap gap-3">
        <button className="flex items-center gap-2 px-4 py-2 bg-brand-primary text-text-inverse rounded-lg hover:bg-brand-primary/90">
          <Play className="w-4 h-4" />
          Run Test
        </button>
        <button className="flex items-center gap-2 px-4 py-2 bg-surface-raised border border-surface-border rounded-lg hover:bg-surface-hover">
          <RefreshCw className="w-4 h-4" />
          Refresh
        </button>
        <button className="flex items-center gap-2 px-4 py-2 bg-status-success/10 text-status-success border border-status-success/20 rounded-lg hover:bg-status-success/20">
          <Download className="w-4 h-4" />
          Export
        </button>
        <button className="flex items-center gap-2 px-4 py-2 bg-status-error/10 text-status-error border border-status-error/20 rounded-lg hover:bg-status-error/20">
          <Trash2 className="w-4 h-4" />
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
    <div className="space-y-4 p-4">
      <h3 className="heading-3 text-text-primary mb-4">Card Header Icons</h3>
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
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
      <h4 className="heading-4 text-text-primary mb-3 border-b border-surface-border pb-2">{title}</h4>
      <div className="grid grid-cols-4 md:grid-cols-6 lg:grid-cols-12 gap-4">{children}</div>
    </div>
  );
}

function IconItem({ icon, name }: { icon: React.ReactNode; name: string }) {
  return (
    <div className="flex flex-col items-center text-center p-2 rounded-lg hover:bg-surface-hover">
      <span className="text-text-primary w-5 h-5">{icon}</span>
      <span className="caption text-text-muted mt-1 truncate w-full">{name}</span>
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
    <div className="flex items-center gap-3 p-3 rounded-lg bg-surface-raised border border-surface-border">
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
    <div className="flex flex-col items-center p-3 rounded-lg bg-surface-raised border border-surface-border">
      <span className="w-8 h-8 text-text-secondary">{icon}</span>
      <span className="body-small text-text-muted mt-2">{name}</span>
    </div>
  );
}

function CardHeaderExample({ icon, title }: { icon: React.ReactNode; title: string }) {
  return (
    <div className="p-4 rounded-lg bg-surface-raised border border-surface-border">
      <div className="flex items-center gap-2">
        <span className="w-5 h-5 text-brand-primary">{icon}</span>
        <span className="body font-medium text-text-primary">{title}</span>
      </div>
    </div>
  );
}
