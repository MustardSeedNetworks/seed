/**
 * AppContext — shared dashboard state for the routed pages.
 *
 * The Seed pages (Link, Network, Path, Wi-Fi, Security, Performance,
 * Reports, Logs) all consume slices of the same backing state owned by
 * the top-level App component (cards, loading flag, interface
 * selection, etc.). Rather than thread props through every page, App
 * exposes the snapshot via this context.
 *
 * App.tsx still owns the hooks and fetchers; pages only read.
 */
import { createContext, useContext } from 'react';
import type { NetworkDiscoveryData } from '../components/cards/NetworkDiscoveryCard';
import type { TraceHopMessage } from '../components/cards/PathDiscoveryCard';
import type { ChannelGraphResponse } from '../components/cards/WiFiChannelGraph';
import type { CardState } from '../hooks/useCardState';
import type { CardSettings, DisplayOptions } from '../types/settings';

export interface AppContextValue {
  cards: CardState;
  loading: boolean;
  isWifi: boolean;
  currentInterface: string;
  cardSettings: CardSettings;
  displayOptions: DisplayOptions;
  networkDiscovery: NetworkDiscoveryData | null;
  triggerDeviceScan: () => Promise<void>;
  registerTraceHopHandler: (handler: (msg: TraceHopMessage) => void) => () => void;
  channelGraphData: ChannelGraphResponse | null;
  channelGraphLoading: boolean;
  appVersion: string;
}

export const AppContext = createContext<AppContextValue | null>(null);

export function useAppContext(): AppContextValue {
  const ctx = useContext(AppContext);
  if (!ctx) {
    throw new Error('useAppContext must be used inside <AppContext.Provider>');
  }
  return ctx;
}
