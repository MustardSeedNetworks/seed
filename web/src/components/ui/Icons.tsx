/**
 * Icons Module
 *
 * Purpose: Centralized icon library re-exporting Lucide React icons with consistent naming.
 * Provides a single source of truth for icon usage across the application, making it easy
 * to maintain icon consistency and switch icon libraries if needed.
 *
 * Icon Categories:
 * - Card header icons: Activity, Cable, Globe, Network, Router, etc.
 * - Status icons: CheckCircle, XCircle, AlertTriangle, AlertCircle, Info
 * - Device type icons: Monitor, Smartphone, Printer, HardDrive, Laptop, Tablet, Tv
 * - Navigation/action icons: ChevronDown/Up/Left/Right, X, Menu, RefreshCw, Play, Pause
 * - UI icons: Eye, EyeOff, Copy, Check, More, Maximize, Minimize, Zap
 * - Specialized icons: Mail, Lock, Terminal, Cloud, Shield, Database, Container, Help
 *
 * Usage:
 * ```typescript
 * import { CheckCircle, AlertTriangle, Router, Eye } from '../ui/Icons';
 *
 * // Use in components
 * <CheckCircle className={iconTokens.size.md} />
 * <AlertTriangle className={iconTokens.size.lg} />
 * ```
 *
 * Best Practice: Always import from this module instead of directly from lucide-react.
 * This makes future icon library changes centralized and easier to manage.
 *
 * Dependencies: lucide-react library
 * Note: Some icons are aliased (e.g., Loader2 as Loader) for consistency
 */

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
  Loader2 as Loader,
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
  Eye as EyeOpen,
  // Path/route icons
  Route,
  // External link
  ExternalLink,
} from "lucide-react";

// Icon size presets are in iconConfig.ts - import from there for non-component needs
