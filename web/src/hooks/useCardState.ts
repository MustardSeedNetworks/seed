/**
 * Card State Management Hook
 *
 * Manages the state for all network monitoring cards.
 * Extracted from App.tsx to improve maintainability and reduce component complexity.
 *
 * Handles:
 * - Card data state initialization
 * - WebSocket message handling for initial state
 * - Card update handling via WebSocket
 * - Link-up detection for auto-run tests
 */

import { useState, useCallback, useRef, useEffect } from "react";
import { logger, LogComponents } from "../lib/logger";
import type { Message, CardUpdate } from "./useWebSocket";
import type {
  LinkData,
  SwitchData,
  DHCPData,
  DNSData,
  VLANData,
  GatewayData,
  WiFiData,
  CableData,
  PublicIPData,
} from "../components/cards";
import type { PipelineEvent, PipelineEventType } from "./usePipelineStatus";

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
  vlan: VLANData | null;
  switch: SwitchData | null;
  wifi: WiFiData | null;
  dhcp: DHCPData | null;
  dns: DNSData | null;
  gateway: GatewayData | null;
  publicip: PublicIPData | null;
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
  return (
    typeof value === "string" && (CARD_IDS as readonly string[]).includes(value)
  );
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
}: UseCardStateProps) {
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
          Type?: string;
          Timestamp?: string;
          RunID?: string;
          Payload?: unknown;
        };

        // Validate the nested event structure
        if (!rawEvent || typeof rawEvent.Type !== "string") {
          logger.warn(
            LogComponents.WEBSOCKET,
            "Invalid pipeline event structure",
            { payload: message.payload }
          );
          return;
        }

        // Check if it's a valid pipeline event type
        if (!isPipelineEvent(rawEvent.Type)) {
          logger.warn(LogComponents.WEBSOCKET, "Unknown pipeline event type", {
            type: rawEvent.Type,
          });
          return;
        }

        const pipelineEvent: PipelineEvent = {
          type: rawEvent.Type,
          timestamp: rawEvent.Timestamp || new Date().toISOString(),
          runId: rawEvent.RunID || "",
          payload: rawEvent.Payload,
        };

        // Dispatch to the pipeline event handler stored on window
        const handler = (
          window as unknown as {
            __pipelineEventHandler?: (event: PipelineEvent) => void;
          }
        ).__pipelineEventHandler;

        if (handler) {
          handler(pipelineEvent);
        }
        return;
      }

      if (message.type === "initial_state") {
        setLoading(false);
        if (!isPlainObject(message.payload)) {
          logger.warn(
            LogComponents.WEBSOCKET,
            "Invalid initial_state payload",
            {
              payload: message.payload,
            }
          );
          return;
        }

        const payload = message.payload;
        if (typeof payload.interface === "string" && payload.interface) {
          setCurrentInterface(payload.interface);
        }

        // Only auto-set WiFi mode if user hasn't manually selected
        if (
          typeof payload.isWireless === "boolean" &&
          !userSetWifiModeRef.current
        ) {
          setIsWifi(payload.isWireless);
        }

        if (isPlainObject(payload.cards)) {
          const updates: Partial<CardState> = {};
          for (const [key, value] of Object.entries(payload.cards)) {
            if (!isCardId(key)) continue;

            const normalized =
              value === null ? null : isPlainObject(value) ? value : undefined;
            if (normalized === undefined) continue;

            switch (key) {
              case "link":
                updates.link = normalized as CardState["link"];
                // Initialize prevLinkUpRef on first load
                if (
                  normalized &&
                  typeof (normalized as { linkUp?: boolean }).linkUp ===
                    "boolean"
                ) {
                  const linkUp = (normalized as { linkUp: boolean }).linkUp;
                  prevLinkUpRef.current = linkUp;

                  // Trigger initial auto-run if link is up on page load
                  if (linkUp && !initialAutoRunDoneRef.current) {
                    initialAutoRunDoneRef.current = true;
                    logger.info(
                      LogComponents.NETWORK,
                      "Link up on initial load, triggering auto-run tests"
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
            }
          }

          if (Object.keys(updates).length > 0) {
            setCards((prev) => ({ ...prev, ...updates }));
          }
        }
      }
    },
    [setCurrentInterface, setIsWifi, userSetWifiModeRef]
  );

  const handleCardUpdate = useCallback((update: CardUpdate) => {
    if (!update || typeof update !== "object") {
      return;
    }

    const { cardId, data } = update as { cardId?: unknown; data?: unknown };

    if (!isCardId(cardId)) {
      logger.warn(
        LogComponents.WEBSOCKET,
        "Ignoring card_update for unknown cardId",
        { cardId }
      );
      return;
    }

    if (data === undefined || (data !== null && !isPlainObject(data))) {
      logger.warn(
        LogComponents.WEBSOCKET,
        "Ignoring card_update with invalid data",
        {
          cardId,
          data,
        }
      );
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
        logger.info(
          LogComponents.NETWORK,
          "Link up detected, triggering auto-run tests"
        );
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
    return () => {
      timeoutIdsRef.current.forEach((id) => clearTimeout(id));
      timeoutIdsRef.current.clear();
    };
  }, []);

  return {
    cards,
    loading,
    setCards,
    setLoading,
    handleMessage,
    handleCardUpdate,
    prevLinkUpRef,
  };
}
