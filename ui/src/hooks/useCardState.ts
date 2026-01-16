// biome-ignore-all lint/complexity/noExcessiveCognitiveComplexity: Complex component
/**
 * Card State Management Hook
 *
 * Manages the state for all network monitoring cards.
 * Extracted from App.tsx to improve maintainability and reduce component complexity.
 *
 * Handles:
 * - Card data state initialization
 * - SSE message handling for initial state
 * - Card update handling via SSE
 * - Link-up detection for auto-run tests
 */

import type React from "react";
import { useCallback, useEffect, useRef, useState } from "react";
import type { CableData } from "../components/cards/CableCard";
import type { DnsData } from "../components/cards/DnsCard";
import type { GatewayData } from "../components/cards/GatewayCard";
import type { LinkData } from "../components/cards/LinkCard";
import type { DhcpData } from "../components/cards/NetworkCard";
import type { TraceHopMessage } from "../components/cards/PathDiscoveryCard";
import type { PublicIpData } from "../components/cards/PublicIpCard";
import type { SwitchData, VlanData } from "../components/cards/SwitchCard";
import type { WiFiData } from "../components/cards/WiFiCard";
import { LogComponents, logger } from "../lib/logger";
import type { PipelineEvent, PipelineEventType } from "./usePipelineStatus";
import type { SseCardUpdate as CardUpdate, SseMessage as Message } from "./useSSE";

// Pipeline event types for routing WebSocket messages
const PIPELINE_EVENT_TYPES: PipelineEventType[] = [
  "pipeline_started",
  "phase_started",
  "phase_progress",
  "phase_completed",
  "phase_failed",
  "device_discovered",
  "device_updated",
  "pipeline_completed",
  "pipeline_failed",
  "pipeline_canceled",
];

function isPipelineEvent(type: string): type is PipelineEventType {
  return PIPELINE_EVENT_TYPES.includes(type as PipelineEventType);
}

/**
 * Centralized state for all network monitoring cards.
 * Each card can be null if not yet loaded or unavailable.
 */
export interface CardState {
  link: LinkData | null;
  cable: CableData | null;
  vlan: VlanData | null;
  switch: SwitchData | null;
  wifi: WiFiData | null;
  dhcp: DhcpData | null;
  dns: DnsData | null;
  gateway: GatewayData | null;
  publicip: PublicIpData | null;
}

const CARD_IDS = [
  "link",
  "cable",
  "vlan",
  "switch",
  "wifi",
  "dhcp",
  "dns",
  "gateway",
  "publicip",
] as const;
type CardId = (typeof CARD_IDS)[number];

function isPlainObject(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function isCardId(value: unknown): value is CardId {
  return typeof value === "string" && (CARD_IDS as readonly string[]).includes(value);
}

interface UseCardStateProps {
  setCurrentInterface: (name: string) => void;
  setIsWifi: (wifi: boolean) => void;
  userSetWifiModeRef: React.MutableRefObject<boolean>;
}

/**
 *
 */
export function useCardState({
  setCurrentInterface,
  setIsWifi,
  userSetWifiModeRef,
}: UseCardStateProps): {
  cards: CardState;
  loading: boolean;
  setCards: React.Dispatch<React.SetStateAction<CardState>>;
  setLoading: React.Dispatch<React.SetStateAction<boolean>>;
  handleMessage: (message: Message) => void;
  handleCardUpdate: (update: CardUpdate) => void;
  prevLinkUpRef: React.MutableRefObject<boolean | null>;
  registerTraceHopHandler: (handler: (msg: TraceHopMessage) => void) => () => void;
} {
  const [cards, setCards] = useState<CardState>({
    link: null,
    cable: null,
    vlan: null,
    switch: null,
    wifi: null,
    dhcp: null,
    dns: null,
    gateway: null,
    publicip: null,
  });
  const [loading, setLoading] = useState(true);

  // Track previous link state to detect link-up transitions for auto-run
  const prevLinkUpRef = useRef<boolean | null>(null);
  // Track if we've triggered initial auto-run on page load
  const initialAutoRunDoneRef = useRef(false);
  // Track setTimeout IDs for cleanup on unmount (fixes #851)
  const timeoutIdsRef = useRef<Set<ReturnType<typeof setTimeout>>>(new Set());

  const handleMessage = useCallback(
    (message: Message) => {
      // Route pipeline events to the pipeline status hook
      // Backend sends: { type: "pipeline", payload: PipelineEvent }
      // PipelineEvent has: { Type: "pipeline_started", Timestamp, RunID, Payload }
      if (message.type === "pipeline") {
        const rawEvent = message.payload as {
          type?: string;
          timestamp?: string;
          runId?: string;
          payload?: unknown;
        };

        // Validate the nested event structure
        if (!rawEvent || typeof rawEvent.type !== "string") {
          logger.warn(LogComponents.Websocket, "Invalid pipeline event structure", {
            payload: message.payload,
          });
          return;
        }

        // Check if it's a valid pipeline event type
        if (!isPipelineEvent(rawEvent.type)) {
          logger.warn(LogComponents.Websocket, "Unknown pipeline event type", {
            type: rawEvent.type,
          });
          return;
        }

        const pipelineEvent: PipelineEvent = {
          type: rawEvent.type,
          timestamp: rawEvent.timestamp || new Date().toISOString(),
          runId: rawEvent.runId || "",
          payload: rawEvent.payload,
        };

        // Dispatch to the pipeline event handler stored on window
        // Fixes #934: Validate handler is a function before calling
        const handler = (
          window as unknown as {
            __pipelineEventHandler?: (event: PipelineEvent) => void;
          }
        ).__pipelineEventHandler;

        // Fixes #966: Wrap in try-catch to prevent handler errors from crashing WebSocket
        if (typeof handler === "function") {
          try {
            handler(pipelineEvent);
          } catch (err) {
            logger.error(LogComponents.Websocket, "Pipeline event handler threw exception", {
              error: err,
              eventType: pipelineEvent.type,
            });
          }
        }
        return;
      }

      // Route traceHop events to the path discovery component
      if (message.type === "traceHop") {
        const traceHopMessage = message.payload as TraceHopMessage;
        const handler = (
          window as unknown as {
            __traceHopHandler?: (msg: TraceHopMessage) => void;
          }
        ).__traceHopHandler;

        if (typeof handler === "function") {
          try {
            handler(traceHopMessage);
          } catch (err) {
            logger.error(LogComponents.Websocket, "TraceHop handler threw exception", {
              error: err,
              target: traceHopMessage.target,
            });
          }
        }
        return;
      }

      if (message.type === "initial_state") {
        setLoading(false);
        if (!isPlainObject(message.payload)) {
          logger.warn(LogComponents.Websocket, "Invalid initial_state payload", {
            payload: message.payload,
          });
          return;
        }

        const { interface: iface, isWireless, cards: payloadCards } = message.payload;
        if (typeof iface === "string" && iface) {
          setCurrentInterface(iface);
        }

        // Only auto-set WiFi mode if user hasn't manually selected
        if (typeof isWireless === "boolean" && !userSetWifiModeRef.current) {
          setIsWifi(isWireless);
        }

        if (isPlainObject(payloadCards)) {
          const updates: Partial<CardState> = {};
          for (const [key, value] of Object.entries(payloadCards)) {
            if (!isCardId(key)) {
              continue;
            }

            // Normalize value: null stays null, plain objects stay, others become undefined
            let normalized: Record<string, unknown> | null | undefined;
            if (value === null) {
              normalized = null;
            } else if (isPlainObject(value)) {
              normalized = value;
            } else {
              normalized = undefined;
            }
            if (normalized === undefined) {
              continue;
            }

            switch (key) {
              case "link":
                updates.link = normalized as CardState["link"];
                // Initialize prevLinkUpRef on first load
                if (
                  normalized &&
                  typeof (normalized as { linkUp?: boolean }).linkUp === "boolean"
                ) {
                  const { linkUp } = normalized as { linkUp: boolean };
                  prevLinkUpRef.current = linkUp;

                  // Trigger initial auto-run if link is up on page load
                  if (linkUp && !initialAutoRunDoneRef.current) {
                    initialAutoRunDoneRef.current = true;
                    logger.info(
                      LogComponents.Network,
                      "Link up on initial load, triggering auto-run tests",
                    );
                    // Track timeout for cleanup on unmount (fixes #851)
                    const timeoutId = setTimeout(() => {
                      timeoutIdsRef.current.delete(timeoutId);
                      window.dispatchEvent(new CustomEvent("runAllTests"));
                    }, 2000);
                    timeoutIdsRef.current.add(timeoutId);
                  }
                }
                break;
              case "cable":
                updates.cable = normalized as CardState["cable"];
                break;
              case "vlan":
                updates.vlan = normalized as CardState["vlan"];
                break;
              case "switch":
                updates.switch = normalized as CardState["switch"];
                break;
              case "wifi":
                updates.wifi = normalized as CardState["wifi"];
                break;
              case "dhcp":
                updates.dhcp = normalized as CardState["dhcp"];
                break;
              case "dns":
                updates.dns = normalized as CardState["dns"];
                break;
              case "gateway":
                updates.gateway = normalized as CardState["gateway"];
                break;
              case "publicip":
                updates.publicip = normalized as CardState["publicip"];
                break;
              default:
                // Unknown card ID - log for debugging
                break;
            }
          }

          if (Object.keys(updates).length > 0) {
            setCards((prev) => ({ ...prev, ...updates }));
          }
        }
      }
    },
    [setCurrentInterface, setIsWifi, userSetWifiModeRef],
  );

  const handleCardUpdate = useCallback((update: CardUpdate) => {
    if (!update || typeof update !== "object") {
      return;
    }

    const { cardId, data } = update as { cardId?: unknown; data?: unknown };

    if (!isCardId(cardId)) {
      logger.warn(LogComponents.Websocket, "Ignoring card_update for unknown cardId", { cardId });
      return;
    }

    if (data === undefined || (data !== null && !isPlainObject(data))) {
      logger.warn(LogComponents.Websocket, "Ignoring card_update with invalid data", {
        cardId,
        data,
      });
      return;
    }

    // Detect link-up transition for auto-run tests
    if (cardId === "link" && data && typeof data === "object") {
      const linkData = data as { linkUp?: boolean };
      const newLinkUp = linkData.linkUp === true;
      const wasDown = prevLinkUpRef.current === false;

      // Update previous state
      if (typeof linkData.linkUp === "boolean") {
        prevLinkUpRef.current = linkData.linkUp;
      }

      // Trigger auto-run when link transitions from down to up
      if (newLinkUp && wasDown) {
        logger.info(LogComponents.Network, "Link up detected, triggering auto-run tests");
        // Small delay to let link stabilize before running tests
        // Track timeout for cleanup on unmount (fixes #851)
        const timeoutId = setTimeout(() => {
          timeoutIdsRef.current.delete(timeoutId);
          window.dispatchEvent(new CustomEvent("runAllTests"));
        }, 1500);
        timeoutIdsRef.current.add(timeoutId);
      }
    }

    setCards((prev) => ({
      ...prev,
      [cardId]: data as CardState[typeof cardId],
    }));
  }, []);

  // Cleanup timeouts on unmount (fixes #851)
  useEffect(() => {
    // Copy ref value for cleanup function (fixes react-hooks/exhaustive-deps warning)
    const timeoutIds = timeoutIdsRef.current;
    return (): void => {
      for (const id of timeoutIds) {
        clearTimeout(id);
      }
      timeoutIds.clear();
    };
  }, []);

  // Register handler for traceHop WebSocket messages
  const registerTraceHopHandler = useCallback(
    (handler: (msg: TraceHopMessage) => void): (() => void) => {
      (
        window as unknown as {
          __traceHopHandler?: (msg: TraceHopMessage) => void;
        }
      ).__traceHopHandler = handler;

      // Return cleanup function
      return () => {
        const win = window as unknown as {
          __traceHopHandler?: (msg: TraceHopMessage) => void;
        };
        if (win.__traceHopHandler === handler) {
          win.__traceHopHandler = undefined;
        }
      };
    },
    [],
  );

  return {
    cards,
    loading,
    setCards,
    setLoading,
    handleMessage,
    handleCardUpdate,
    prevLinkUpRef,
    registerTraceHopHandler,
  };
}
