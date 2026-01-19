import type { Meta, StoryObj } from '@storybook/react-vite';
import type React from 'react';
import { button, cn, icon as iconTheme, layout, section, spacing } from '../../styles/theme';
import {
  // Card header icons
  Activity,
  AlertCircle,
  AlertTriangle,
  Cable,
  Calendar,
  Check,
  // Status icons
  CheckCircle,
  // Navigation/action icons
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  ChevronUp,
  Clock,
  Columns3,
  Container,
  Copy,
  // System/misc
  Cpu,
  Database,
  Download,
  Edit,
  // UI icons
  Eye,
  EyeOff,
  FileText,
  Filter,
  Gauge,
  Globe,
  Grid3X3,
  HardDrive,
  HardDriveDownload,
  HeartPulse,
  Info,
  Key,
  Laptop,
  // Layout
  LayoutDashboard,
  List,
  Loader,
  Lock,
  Mail,
  MemoryStick,
  Menu,
  Minus,
  // Device type icons
  Monitor,
  Network,
  Palette,
  Pause,
  Play,
  PlugZap,
  Plus,
  Printer,
  RefreshCw,
  RotateCcw,
  Router,
  ScanSearch,
  Search,
  Server,
  Settings,
  Shield,
  ShieldAlert,
  ShieldCheck,
  ShieldOff,
  Signal,
  SignalHigh,
  SignalLow,
  SignalMedium,
  SignalZero,
  // Settings icons
  SlidersHorizontal,
  Smartphone,
  Tablet,
  // Service/protocol icons
  Terminal,
  Timer,
  Trash2,
  Tv,
  Unlock,
  // Network specific
  Unplug,
  Upload,
  User,
  Wifi,
  X,
  XCircle,
  Zap,
} from './Icons';
import { ICON_SIZES } from './icon-config';

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
    <div class={cn(section.spacing.spacious, spacing.pad.default)}>
      <iconCategory title="Card Header Icons">
        <iconItem icon={<Activity />} name="Activity" />
        <iconItem icon={<Cable />} name="Cable" />
        <iconItem icon={<Globe />} name="Globe" />
        <iconItem icon={<Network />} name="Network" />
        <iconItem icon={<Router />} name="Router" />
        <iconItem icon={<Search />} name="Search" />
        <iconItem icon={<ScanSearch />} name="ScanSearch" />
        <iconItem icon={<Server />} name="Server" />
        <iconItem icon={<Wifi />} name="Wifi" />
        <iconItem icon={<Gauge />} name="Gauge" />
        <iconItem icon={<HeartPulse />} name="HeartPulse" />
        <iconItem icon={<Settings />} name="Settings" />
      </iconCategory>

      <iconCategory title="Status Icons">
        <iconItem icon={<CheckCircle class="text-status-success" />} name="CheckCircle" />
        <iconItem icon={<XCircle class="text-status-error" />} name="XCircle" />
        <iconItem icon={<AlertTriangle class="text-status-warning" />} name="AlertTriangle" />
        <iconItem icon={<AlertCircle class="text-status-info" />} name="AlertCircle" />
        <iconItem icon={<Info class="text-brand-primary" />} name="Info" />
      </iconCategory>

      <iconCategory title="Device Type Icons">
        <iconItem icon={<Monitor />} name="Monitor" />
        <iconItem icon={<Smartphone />} name="Smartphone" />
        <iconItem icon={<Printer />} name="Printer" />
        <iconItem icon={<HardDrive />} name="HardDrive" />
        <iconItem icon={<Laptop />} name="Laptop" />
        <iconItem icon={<Tablet />} name="Tablet" />
        <iconItem icon={<Tv />} name="Tv" />
      </iconCategory>

      <iconCategory title="Navigation Icons">
        <iconItem icon={<ChevronDown />} name="ChevronDown" />
        <iconItem icon={<ChevronUp />} name="ChevronUp" />
        <iconItem icon={<ChevronLeft />} name="ChevronLeft" />
        <iconItem icon={<ChevronRight />} name="ChevronRight" />
        <iconItem icon={<X />} name="X" />
        <iconItem icon={<Menu />} name="Menu" />
      </iconCategory>

      <iconCategory title="Action Icons">
        <iconItem icon={<RefreshCw />} name="RefreshCw" />
        <iconItem icon={<Download />} name="Download" />
        <iconItem icon={<Upload />} name="Upload" />
        <iconItem icon={<Play />} name="Play" />
        <iconItem icon={<Pause />} name="Pause" />
        <iconItem icon={<RotateCcw />} name="RotateCcw" />
        <iconItem icon={<Loader class="animate-spin" />} name="Loader" />
      </iconCategory>

      <iconCategory title="UI Icons">
        <iconItem icon={<Eye />} name="Eye" />
        <iconItem icon={<EyeOff />} name="EyeOff" />
        <iconItem icon={<Copy />} name="Copy" />
        <iconItem icon={<Check />} name="Check" />
        <iconItem icon={<Trash2 />} name="Trash2" />
        <iconItem icon={<Edit />} name="Edit" />
        <iconItem icon={<Plus />} name="Plus" />
        <iconItem icon={<Minus />} name="Minus" />
        <iconItem icon={<Filter />} name="Filter" />
      </iconCategory>

      <iconCategory title="Network Icons">
        <iconItem icon={<Unplug />} name="Unplug" />
        <iconItem icon={<PlugZap />} name="PlugZap" />
        <iconItem icon={<Signal />} name="Signal" />
        <iconItem icon={<SignalHigh />} name="SignalHigh" />
        <iconItem icon={<SignalMedium />} name="SignalMedium" />
        <iconItem icon={<SignalLow />} name="SignalLow" />
        <iconItem icon={<SignalZero />} name="SignalZero" />
        <iconItem icon={<Zap />} name="Zap" />
      </iconCategory>

      <iconCategory title="Service/Protocol Icons">
        <iconItem icon={<Terminal />} name="Terminal" />
        <iconItem icon={<FileText />} name="FileText" />
        <iconItem icon={<Mail />} name="Mail" />
        <iconItem icon={<Database />} name="Database" />
        <iconItem icon={<Container />} name="Container" />
        <iconItem icon={<ShieldOff />} name="ShieldOff" />
      </iconCategory>

      <iconCategory title="System Icons">
        <iconItem icon={<Cpu />} name="Cpu" />
        <iconItem icon={<MemoryStick />} name="MemoryStick" />
        <iconItem icon={<HardDriveDownload />} name="HardDriveDownload" />
        <iconItem icon={<Clock />} name="Clock" />
        <iconItem icon={<Timer />} name="Timer" />
        <iconItem icon={<Calendar />} name="Calendar" />
        <iconItem icon={<User />} name="User" />
      </iconCategory>

      <iconCategory title="Security Icons">
        <iconItem icon={<Lock />} name="Lock" />
        <iconItem icon={<Unlock />} name="Unlock" />
        <iconItem icon={<Key />} name="Key" />
        <iconItem icon={<Shield />} name="Shield" />
        <iconItem icon={<ShieldCheck class="text-status-success" />} name="ShieldCheck" />
        <iconItem icon={<ShieldAlert class="text-status-warning" />} name="ShieldAlert" />
      </iconCategory>

      <iconCategory title="Layout Icons">
        <iconItem icon={<LayoutDashboard />} name="LayoutDashboard" />
        <iconItem icon={<List />} name="List" />
        <iconItem icon={<Grid3X3 />} name="Grid3X3" />
        <iconItem icon={<Columns3 />} name="Columns3" />
      </iconCategory>

      <iconCategory title="Settings Icons">
        <iconItem icon={<SlidersHorizontal />} name="SlidersHorizontal" />
        <iconItem icon={<Palette />} name="Palette" />
      </iconCategory>
    </div>
  ),
};

/**
 * Icon size presets demonstration.
 */
export const Sizes: Story = {
  render: () => (
    <div class={cn(section.spacing.comfortable, spacing.pad.default)}>
      <h3 class="heading-3 text-text-primary">Icon Size Presets</h3>
      <div class={cn('flex items-end', spacing.gap.spacious)}>
        {(Object.keys(ICON_SIZES) as Array<keyof typeof ICON_SIZES>).map((size) => (
          <div key={size} class="text-center">
            <Activity class={cn(ICON_SIZES[size], 'text-brand-primary mx-auto')} />
            <p class={cn('body-small text-text-muted', spacing.margin.top.inline)}>{size}</p>
            <p class="caption text-text-muted">{ICON_SIZES[size]}</p>
          </div>
        ))}
      </div>
    </div>
  ),
  parameters: {
    docs: {
      description: {
        story:
          'Available size presets: xs (w-3 h-3), sm (w-4 h-4), md (w-5 h-5), lg (w-6 h-6), xl (w-8 h-8).',
      },
    },
  },
};

/**
 * Status icons with semantic colors.
 */
export const StatusIcons: Story = {
  render: () => (
    <div class={cn(section.spacing.default, spacing.pad.default)}>
      <h3 class={cn('heading-3 text-text-primary', spacing.margin.bottom.content)}>
        Status Icons with Semantic Colors
      </h3>
      <div class={cn('grid grid-cols-2 md:grid-cols-4', spacing.gap.comfortable)}>
        <statusExample
          icon={<CheckCircle class={cn(iconTheme.size.lg, 'text-status-success')} />}
          label="Success"
          description="Operation completed"
        />
        <statusExample
          icon={<AlertTriangle class={cn(iconTheme.size.lg, 'text-status-warning')} />}
          label="Warning"
          description="Needs attention"
        />
        <statusExample
          icon={<XCircle class={cn(iconTheme.size.lg, 'text-status-error')} />}
          label="Error"
          description="Operation failed"
        />
        <statusExample
          icon={<Info class={cn(iconTheme.size.lg, 'text-status-info')} />}
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
    <div class={cn(section.spacing.default, spacing.pad.default)}>
      <h3 class={cn('heading-3 text-text-primary', spacing.margin.bottom.content)}>
        Network Status Indicators
      </h3>
      <div class={cn('flex', spacing.gap.spacious)}>
        <div class="text-center">
          <div
            class={cn(
              iconTheme.size['2xl'],
              'rounded-full bg-status-success/10 flex items-center justify-center',
              spacing.margin.bottom.inline,
            )}
          >
            <Wifi class={cn(iconTheme.size.lg, 'text-status-success')} />
          </div>
          <p class="body-small">Connected</p>
        </div>
        <div class="text-center">
          <div
            class={cn(
              iconTheme.size['2xl'],
              'rounded-full bg-status-warning/10 flex items-center justify-center',
              spacing.margin.bottom.inline,
            )}
          >
            <SignalLow class={cn(iconTheme.size.lg, 'text-status-warning')} />
          </div>
          <p class="body-small">Weak Signal</p>
        </div>
        <div class="text-center">
          <div
            class={cn(
              iconTheme.size['2xl'],
              'rounded-full bg-status-error/10 flex items-center justify-center',
              spacing.margin.bottom.inline,
            )}
          >
            <Unplug class={cn(iconTheme.size.lg, 'text-status-error')} />
          </div>
          <p class="body-small">Disconnected</p>
        </div>
        <div class="text-center">
          <div
            class={cn(
              iconTheme.size['2xl'],
              'rounded-full bg-status-info/10 flex items-center justify-center',
              spacing.margin.bottom.inline,
            )}
          >
            <Loader class={cn(iconTheme.size.lg, 'text-status-info animate-spin')} />
          </div>
          <p class="body-small">Connecting</p>
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
    <div class={cn(section.spacing.default, spacing.pad.default)}>
      <h3 class={cn('heading-3 text-text-primary', spacing.margin.bottom.content)}>
        Device Type Icons
      </h3>
      <div class={cn('grid grid-cols-3 md:grid-cols-7', spacing.gap.comfortable)}>
        <deviceExample icon={<Monitor />} name="Desktop" />
        <deviceExample icon={<Laptop />} name="Laptop" />
        <deviceExample icon={<Smartphone />} name="Phone" />
        <deviceExample icon={<Tablet />} name="Tablet" />
        <deviceExample icon={<Printer />} name="Printer" />
        <deviceExample icon={<Tv />} name="TV" />
        <deviceExample icon={<HardDrive />} name="NAS" />
      </div>
    </div>
  ),
};

/**
 * Button icons with different states.
 */
export const ButtonIcons: Story = {
  render: () => (
    <div class={cn(section.spacing.default, spacing.pad.default)}>
      <h3 class={cn('heading-3 text-text-primary', spacing.margin.bottom.content)}>Button Icons</h3>
      <div class={cn('flex flex-wrap', spacing.gap.default)}>
        <button
          type="button"
          class={cn(
            layout.inline.default,
            button.size.md,
            'bg-brand-primary text-text-inverse rounded-lg hover:bg-brand-primary/90',
          )}
        >
          <Play class={iconTheme.size.sm} />
          Run Test
        </button>
        <button
          type="button"
          class={cn(
            layout.inline.default,
            button.size.md,
            'bg-surface-raised border border-surface-border rounded-lg hover:bg-surface-hover',
          )}
        >
          <RefreshCw class={iconTheme.size.sm} />
          Refresh
        </button>
        <button
          type="button"
          class={cn(
            layout.inline.default,
            button.size.md,
            'bg-status-success/10 text-status-success border border-status-success/20 rounded-lg hover:bg-status-success/20',
          )}
        >
          <Download class={iconTheme.size.sm} />
          Export
        </button>
        <button
          type="button"
          class={cn(
            layout.inline.default,
            button.size.md,
            'bg-status-error/10 text-status-error border border-status-error/20 rounded-lg hover:bg-status-error/20',
          )}
        >
          <Trash2 class={iconTheme.size.sm} />
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
    <div class={cn(section.spacing.default, spacing.pad.default)}>
      <h3 class={cn('heading-3 text-text-primary', spacing.margin.bottom.content)}>
        Card Header Icons
      </h3>
      <div class={cn('grid grid-cols-2 md:grid-cols-4', spacing.gap.comfortable)}>
        <cardHeaderExample icon={<Activity />} title="Link Status" />
        <cardHeaderExample icon={<Wifi />} title="WiFi" />
        <cardHeaderExample icon={<Cable />} title="Cable Test" />
        <cardHeaderExample icon={<Network />} title="Network" />
        <cardHeaderExample icon={<Server />} title="Gateway" />
        <cardHeaderExample icon={<Gauge />} title="Performance" />
        <cardHeaderExample icon={<ScanSearch />} title="Discovery" />
        <cardHeaderExample icon={<HeartPulse />} title="Health Checks" />
      </div>
    </div>
  ),
};

// Helper components for stories
function _iconCategory({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}): React.JSX.Element {
  return (
    <div>
      <h4
        class={cn(
          'heading-4 text-text-primary border-b border-surface-border',
          spacing.margin.bottom.heading,
          spacing.margin.bottom.inline,
        )}
      >
        {title}
      </h4>
      <div class={cn('grid grid-cols-4 md:grid-cols-6 lg:grid-cols-12', spacing.gap.comfortable)}>
        {children}
      </div>
    </div>
  );
}

function _iconItem({ icon, name }: { icon: React.ReactNode; name: string }): React.JSX.Element {
  return (
    <div
      class={cn(
        'flex flex-col items-center text-center rounded-lg hover:bg-surface-hover',
        spacing.pad.sm,
      )}
    >
      <span class={cn('text-text-primary', iconTheme.size.md)}>{icon}</span>
      <span class={cn('caption text-text-muted truncate w-full', spacing.margin.top.inline)}>
        {name}
      </span>
    </div>
  );
}

function _statusExample({
  icon,
  label,
  description,
}: {
  icon: React.ReactNode;
  label: string;
  description: string;
}): React.JSX.Element {
  return (
    <div
      class={cn(
        'flex items-center rounded-lg bg-surface-raised border border-surface-border',
        spacing.gap.default,
        spacing.pad.sm,
      )}
    >
      {icon}
      <div>
        <p class="body-small font-medium text-text-primary">{label}</p>
        <p class="caption text-text-muted">{description}</p>
      </div>
    </div>
  );
}

function _deviceExample({
  icon,
  name,
}: {
  icon: React.ReactNode;
  name: string;
}): React.JSX.Element {
  return (
    <div
      class={cn(
        'flex flex-col items-center rounded-lg bg-surface-raised border border-surface-border',
        spacing.pad.sm,
      )}
    >
      <span class={cn(iconTheme.size.xl, 'text-text-secondary')}>{icon}</span>
      <span class={cn('body-small text-text-muted', spacing.margin.top.inline)}>{name}</span>
    </div>
  );
}

function _cardHeaderExample({
  icon,
  title,
}: {
  icon: React.ReactNode;
  title: string;
}): React.JSX.Element {
  return (
    <div
      class={cn('rounded-lg bg-surface-raised border border-surface-border', spacing.pad.default)}
    >
      <div class={cn('flex items-center', spacing.gap.compact)}>
        <span class={cn(iconTheme.size.md, 'text-brand-primary')}>{icon}</span>
        <span class="body font-medium text-text-primary">{title}</span>
      </div>
    </div>
  );
}
