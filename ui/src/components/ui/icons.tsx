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

import {
  Activity,
  AlertCircle,
  AlertTriangle,
  ArrowUpDown,
  BookOpen,
  Cable,
  Calendar,
  Check,
  CheckCircle,
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  ChevronUp,
  Clock,
  Columns3,
  Container,
  Copy,
  Cpu,
  Database,
  Download,
  Edit,
  ExternalLink,
  Eye,
  EyeOff,
  FileText,
  Filter,
  Gauge,
  Globe,
  Grid3X3,
  HardDrive,
  HardDriveDownload,
  Heart,
  HeartPulse,
  Info,
  Key,
  Laptop,
  LayoutDashboard,
  Lightbulb,
  List,
  Loader2,
  Lock,
  Mail,
  Maximize2,
  MemoryStick,
  Menu,
  Minimize2,
  Minus,
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
  Route,
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
  SlidersHorizontal,
  Smartphone,
  SortAsc,
  SortDesc,
  Tablet,
  Terminal,
  Timer,
  Trash2,
  Tv,
  Unlock,
  Unplug,
  Upload,
  User,
  Wifi,
  X,
  XCircle,
  Zap,
} from "lucide-react";

export {
  // Card header icons
  Activity,
  AlertCircle,
  AlertTriangle,
  ArrowUpDown,
  // Help/documentation icons
  BookOpen,
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
  // External link
  ExternalLink,
  // UI icons
  Eye,
  Eye as EyeOpen,
  EyeOff,
  FileText,
  Filter,
  Gauge,
  Globe,
  Grid3X3,
  HardDrive,
  HardDriveDownload,
  Heart,
  HeartPulse,
  Info,
  Key,
  Laptop,
  // Layout
  LayoutDashboard,
  Lightbulb,
  List,
  Loader2 as Loader,
  Lock,
  Mail,
  Maximize2,
  MemoryStick,
  Menu,
  Minimize2,
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
  // Path/route icons
  Route,
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
  SortAsc,
  SortDesc,
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
};

// Icon size presets are in icon-config.ts - import from there for non-component needs
