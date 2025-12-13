// Re-export commonly used Lucide icons for consistency across the app
// This provides a single source of truth for icon usage

export {
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
  SortAsc,
  SortDesc,
  ArrowUpDown,
  // Network specific
  Unplug,
  PlugZap,
  Signal,
  SignalHigh,
  SignalLow,
  SignalMedium,
  SignalZero,
  Zap,
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
} from "lucide-react";

// Icon size presets for consistency
export const ICON_SIZES = {
  xs: "w-3 h-3",
  sm: "w-4 h-4",
  md: "w-5 h-5",
  lg: "w-6 h-6",
  xl: "w-8 h-8",
} as const;

export type IconSize = keyof typeof ICON_SIZES;
