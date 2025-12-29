/**
 * useWebSocket.test.ts - WebSocket Hook Tests
 *
 * Purpose: Comprehensive test suite for the useWebSocket hook covering connection
 * management, reconnection logic, message handling, and error scenarios.
 *
 * Key Test Areas:
 * - Connection initialization: proper WebSocket creation and setup
 * - Auto-reconnection: exponential backoff with retry attempts
 * - Message handling: parsing and processing incoming messages
 * - CardUpdate events: proper event dispatch on card data updates
 * - Connection status: tracking connected/disconnected/error states
 * - Auth token passing: token sent via Sec-WebSocket-Protocol header
 * - Error handling: graceful error state and recovery
 * - Cleanup: proper cleanup on unmount
 * - Message type validation: proper Message and CardUpdate type handling
 *
 * Test Framework: Vitest with React Testing Library hooks
 * Mocks: WebSocket class with MockWebSocket implementation
 *
 * Usage:
 * ```bash
 * npm test -- useWebSocket.test.ts
 * ```
 *
 * Dependencies: vitest, @testing-library/react
 */

import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { type CardUpdate, type Message, useWebSocket } from "./useWebSocket";

// Mock WebSocket
class MockWebSocket {
  static instances: MockWebSocket[] = [];
  static CONNECTING = 0;
  static OPEN = 1;
  static CLOSING = 2;
  static CLOSED = 3;

  url: string;
  protocols?: string | string[];
  readyState: number = MockWebSocket.CONNECTING;
  onopen: ((event: Event) => void) | null = null;
  onclose: ((event: CloseEvent) => void) | null = null;
  onerror: ((event: Event) => void) | null = null;
  onmessage: ((event: MessageEvent) => void) | null = null;
  closeWasCalled = false;
  sentMessages: string[] = [];

  constructor(url: string, protocols?: string | string[]) {
    this.url = url;
    this.protocols = protocols;
    MockWebSocket.instances.push(this);
  }

  send(data: string) {
    this.sentMessages.push(data);
  }

  close() {
    this.closeWasCalled = true;
    this.readyState = MockWebSocket.CLOSED;
    if (this.onclose) {
      this.onclose(new CloseEvent("close"));
    }
  }

  // Test helpers
  simulateOpen() {
    this.readyState = MockWebSocket.OPEN;
    if (this.onopen) {
      this.onopen(new Event("open"));
    }
  }

  simulateClose(code = 1000, reason = "") {
    this.readyState = MockWebSocket.CLOSED;
    if (this.onclose) {
      this.onclose({ code, reason, wasClean: true } as CloseEvent);
    }
  }

  simulateError() {
    if (this.onerror) {
      this.onerror(new Event("error"));
    }
  }

  simulateMessage(data: object) {
    if (this.onmessage) {
      this.onmessage({ data: JSON.stringify(data) } as MessageEvent);
    }
  }
}

describe("useWebSocket", () => {
  let originalWebSocket: typeof WebSocket;
  let originalWindowWebSocket: typeof WebSocket;

  beforeEach(() => {
    originalWebSocket = globalThis.WebSocket;
    originalWindowWebSocket = window.WebSocket;
    // biome-ignore lint/style/useNamingConvention: WebSocket is the browser global API name
    (globalThis as unknown as { WebSocket: typeof WebSocket }).WebSocket =
      MockWebSocket as unknown as typeof WebSocket;
    // biome-ignore lint/style/useNamingConvention: WebSocket is the browser global API name
    (window as unknown as { WebSocket: typeof WebSocket }).WebSocket =
      MockWebSocket as unknown as typeof WebSocket;
    MockWebSocket.instances = [];
    vi.useFakeTimers();
  });

  afterEach(() => {
    // biome-ignore lint/style/useNamingConvention: WebSocket is the browser global API name
    (globalThis as unknown as { WebSocket: typeof WebSocket }).WebSocket = originalWebSocket;
    // biome-ignore lint/style/useNamingConvention: WebSocket is the browser global API name
    (window as unknown as { WebSocket: typeof WebSocket }).WebSocket = originalWindowWebSocket;
    vi.useRealTimers();
  });

  async function flushEffects() {
    await act(async () => {
      await Promise.resolve();
    });
  }

  it("starts with disconnected status", () => {
    const { result } = renderHook(() => useWebSocket({ url: "/ws", token: "test-token" }));

    // Initially it will be connecting since useEffect runs
    expect(["connecting", "disconnected"]).toContain(result.current.status);
  });

  it("connects to WebSocket on mount", async () => {
    renderHook(() => useWebSocket({ url: "/ws", token: "test-token" }));
    await flushEffects();

    expect(MockWebSocket.instances.length).toBe(1);
  });

  it("transitions to connected status on open", async () => {
    const { result } = renderHook(() => useWebSocket({ url: "/ws", token: "test-token" }));
    await flushEffects();

    const ws = MockWebSocket.instances[0];
    act(() => {
      ws.simulateOpen();
    });

    expect(result.current.status).toBe("connected");
  });

  it("transitions to error status on WebSocket error", async () => {
    const { result } = renderHook(() => useWebSocket({ url: "/ws", token: "test-token" }));
    await flushEffects();

    const ws = MockWebSocket.instances[0];
    act(() => {
      ws.simulateError();
    });

    expect(result.current.status).toBe("error");
  });

  it("transitions to disconnected status on abnormal close", async () => {
    const { result } = renderHook(() => useWebSocket({ url: "/ws", token: "test-token" }));
    await flushEffects();

    const ws = MockWebSocket.instances[0];
    act(() => {
      ws.simulateOpen();
    });

    // Abnormal close (code 1006) transitions to "disconnected" before retry
    act(() => {
      ws.simulateClose(1006, "Connection lost");
    });

    expect(result.current.status).toBe("disconnected");
  });

  it("transitions to error status on normal close (non-retryable)", async () => {
    const { result } = renderHook(() => useWebSocket({ url: "/ws", token: "test-token" }));
    await flushEffects();

    const ws = MockWebSocket.instances[0];
    act(() => {
      ws.simulateOpen();
    });

    // Normal close (code 1000) is non-retryable, transitions to "error"
    act(() => {
      ws.simulateClose(1000, "Normal closure");
    });

    expect(result.current.status).toBe("error");
  });

  it("calls onMessage callback when receiving message", async () => {
    const onMessage = vi.fn();
    renderHook(() => useWebSocket({ url: "/ws", token: "test-token", onMessage }));
    await flushEffects();

    const ws = MockWebSocket.instances[0];
    act(() => {
      ws.simulateOpen();
    });

    const testMessage: Message = { type: "test", payload: { data: "value" } };
    act(() => {
      ws.simulateMessage(testMessage);
    });

    expect(onMessage).toHaveBeenCalledWith(testMessage);
  });

  it("calls onCardUpdate for card_update messages", async () => {
    const onCardUpdate = vi.fn();
    renderHook(() => useWebSocket({ url: "/ws", token: "test-token", onCardUpdate }));
    await flushEffects();

    const ws = MockWebSocket.instances[0];
    act(() => {
      ws.simulateOpen();
    });

    const cardUpdate: CardUpdate = {
      cardId: "test-card",
      data: { value: 123 },
    };
    act(() => {
      ws.simulateMessage({ type: "card_update", payload: cardUpdate });
    });

    expect(onCardUpdate).toHaveBeenCalledWith(cardUpdate);
  });

  it("send function sends message when connected", async () => {
    const { result } = renderHook(() => useWebSocket({ url: "/ws", token: "test-token" }));
    await flushEffects();

    const ws = MockWebSocket.instances[0];
    act(() => {
      ws.simulateOpen();
    });

    const message: Message = { type: "subscribe", payload: { topic: "test" } };
    act(() => {
      result.current.send(message);
    });

    expect(ws.sentMessages).toHaveLength(1);
    expect(JSON.parse(ws.sentMessages[0])).toEqual(message);
  });

  it("reconnect function reconnects the WebSocket", async () => {
    const { result } = renderHook(() => useWebSocket({ url: "/ws", token: "test-token" }));
    await flushEffects();

    const initialWs = MockWebSocket.instances[0];
    act(() => {
      initialWs.simulateOpen();
    });

    expect(MockWebSocket.instances.length).toBe(1);

    act(() => {
      result.current.reconnect();
    });

    // New WebSocket should be created
    expect(MockWebSocket.instances.length).toBe(2);
  });

  it("cleans up WebSocket on unmount", async () => {
    const { unmount } = renderHook(() => useWebSocket({ url: "/ws", token: "test-token" }));
    await flushEffects();

    const ws = MockWebSocket.instances[0];
    act(() => {
      ws.simulateOpen();
    });

    unmount();

    expect(ws.closeWasCalled).toBe(true);
  });

  it("constructs correct WebSocket URL", async () => {
    // Mock window.location
    Object.defineProperty(window, "location", {
      value: {
        protocol: "http:",
        host: "localhost:8080",
      },
      writable: true,
    });

    renderHook(() => useWebSocket({ url: "/ws/updates", token: "test-token" }));
    await flushEffects();

    const ws = MockWebSocket.instances[0];
    expect(ws.url).toBe("ws://localhost:8080/ws/updates");
    // Since fix #660: No protocols - uses httpOnly cookie authentication
    expect(ws.protocols).toBeUndefined();
  });

  it("uses wss for https pages", async () => {
    Object.defineProperty(window, "location", {
      value: {
        protocol: "https:",
        host: "example.com",
      },
      writable: true,
    });

    renderHook(() => useWebSocket({ url: "/ws", token: "test-token" }));
    await flushEffects();

    const ws = MockWebSocket.instances[0];
    expect(ws.url).toBe("wss://example.com/ws");
    // Since fix #660: No protocols - uses httpOnly cookie authentication
    expect(ws.protocols).toBeUndefined();
  });

  it("uses provided URL directly if it starts with ws", async () => {
    renderHook(() =>
      useWebSocket({
        url: "ws://custom-server.com/socket",
        token: "test-token",
      }),
    );
    await flushEffects();

    const ws = MockWebSocket.instances[0];
    expect(ws.url).toBe("ws://custom-server.com/socket");
    // Since fix #660: No protocols - uses httpOnly cookie authentication
    expect(ws.protocols).toBeUndefined();
  });

  it("attempts reconnection after close", async () => {
    const randomSpy = vi.spyOn(Math, "random").mockReturnValue(0);
    renderHook(() =>
      useWebSocket({
        url: "/ws",
        token: "test-token",
        reconnectInterval: 1000,
        maxReconnectAttempts: 3,
      }),
    );
    await flushEffects();

    const ws = MockWebSocket.instances[0];
    expect(MockWebSocket.instances.length).toBe(1);

    // Use code 1006 (abnormal closure) to trigger reconnection
    act(() => {
      ws.simulateClose(1006, "Connection lost");
    });

    // Fast forward to trigger reconnect
    act(() => {
      vi.advanceTimersByTime(1000);
    });

    // New WebSocket should be created (at least one reconnect)
    expect(MockWebSocket.instances.length).toBeGreaterThanOrEqual(2);
    randomSpy.mockRestore();
  });

  it("continues reconnecting after maxReconnectAttempts", async () => {
    const randomSpy = vi.spyOn(Math, "random").mockReturnValue(0);
    renderHook(() =>
      useWebSocket({
        url: "/ws",
        token: "test-token",
        reconnectInterval: 100,
        maxReconnectAttempts: 2,
      }),
    );
    await flushEffects();

    // First connection closes (use code 1006 for abnormal closure to trigger retry)
    act(() => {
      const lastInstance = MockWebSocket.instances.at(-1);
      if (lastInstance) lastInstance.simulateClose(1006, "Connection lost");
    });
    act(() => {
      vi.advanceTimersByTime(100);
    });

    // Second close -> reconnect after 200ms
    act(() => {
      const lastInstance = MockWebSocket.instances.at(-1);
      if (lastInstance) lastInstance.simulateClose(1006, "Connection lost");
    });
    act(() => {
      vi.advanceTimersByTime(200);
    });

    // Third close (exceeds maxReconnectAttempts) -> should still reconnect after capped delay (200ms)
    act(() => {
      const lastInstance = MockWebSocket.instances.at(-1);
      if (lastInstance) lastInstance.simulateClose(1006, "Connection lost");
    });
    act(() => {
      vi.advanceTimersByTime(200);
    });

    expect(MockWebSocket.instances.length).toBeGreaterThanOrEqual(4);
    randomSpy.mockRestore();
  });

  it("handles multiple messages in one frame", async () => {
    const onMessage = vi.fn();
    renderHook(() => useWebSocket({ url: "/ws", token: "test-token", onMessage }));
    await flushEffects();

    const ws = MockWebSocket.instances[0];
    act(() => {
      ws.simulateOpen();
    });

    // Simulate multiple JSON messages separated by newlines
    const multiMessage = '{"type":"message1","payload":{}}\n{"type":"message2","payload":{}}';
    act(() => {
      if (ws.onmessage) {
        ws.onmessage({ data: multiMessage } as MessageEvent);
      }
    });

    expect(onMessage).toHaveBeenCalledTimes(2);
    expect(onMessage).toHaveBeenCalledWith({
      type: "message1",
      payload: {},
    });
    expect(onMessage).toHaveBeenCalledWith({
      type: "message2",
      payload: {},
    });
  });

  it("handles invalid JSON message gracefully", async () => {
    const onMessage = vi.fn();
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {
      // intentionally empty - suppress console.error output in tests
    });

    renderHook(() => useWebSocket({ url: "/ws", token: "test-token", onMessage }));
    await flushEffects();

    const ws = MockWebSocket.instances[0];
    act(() => {
      ws.simulateOpen();
    });

    // Send invalid JSON
    act(() => {
      if (ws.onmessage) {
        ws.onmessage({ data: "not valid json" } as MessageEvent);
      }
    });

    expect(onMessage).not.toHaveBeenCalled();
    expect(consoleSpy).toHaveBeenCalled();

    consoleSpy.mockRestore();
  });

  it("send warns when not connected", async () => {
    const consoleSpy = vi.spyOn(console, "warn").mockImplementation(() => {
      // intentionally empty - suppress console.warn output in tests
    });

    // Without token, WebSocket is never created, so send() should warn
    const { result } = renderHook(() => useWebSocket({ url: "/ws" }));
    await flushEffects();

    // Don't open the connection
    const message: Message = { type: "test", payload: {} };
    act(() => {
      result.current.send(message);
    });

    // Logger formats with color codes and prefix, check message is included
    expect(consoleSpy).toHaveBeenCalled();
    const calls = consoleSpy.mock.calls;
    const hasWarning = calls.some((args) =>
      args.some((arg) => String(arg).includes("WebSocket not connected, cannot send message")),
    );
    expect(hasWarning).toBe(true);

    consoleSpy.mockRestore();
  });

  it("reconnect resets attempt counter", async () => {
    const randomSpy = vi.spyOn(Math, "random").mockReturnValue(0);
    const { result } = renderHook(() =>
      useWebSocket({
        url: "/ws",
        token: "test-token",
        reconnectInterval: 100,
        maxReconnectAttempts: 2,
      }),
    );
    await flushEffects();

    const ws = MockWebSocket.instances[0];
    act(() => {
      ws.simulateOpen();
    });

    // Close and exhaust reconnect attempts (use code 1006 for abnormal closure)
    act(() => {
      ws.simulateClose(1006, "Connection lost");
    });
    act(() => {
      vi.advanceTimersByTime(100);
    });
    const ws2 = MockWebSocket.instances[MockWebSocket.instances.length - 1];
    act(() => {
      ws2.simulateClose(1006, "Connection lost");
    });
    act(() => {
      vi.advanceTimersByTime(100);
    });

    const countBeforeManualReconnect = MockWebSocket.instances.length;

    // Manual reconnect should reset counter and allow reconnection
    act(() => {
      result.current.reconnect();
    });

    // Should have created a new WebSocket
    expect(MockWebSocket.instances.length).toBe(countBeforeManualReconnect + 1);
    randomSpy.mockRestore();
  });

  it("resets reconnect attempts on successful connection", async () => {
    const randomSpy = vi.spyOn(Math, "random").mockReturnValue(0);
    renderHook(() =>
      useWebSocket({
        url: "/ws",
        token: "test-token",
        reconnectInterval: 100,
        maxReconnectAttempts: 3,
      }),
    );
    await flushEffects();

    // First connection
    let ws = MockWebSocket.instances[0];
    act(() => {
      ws.simulateOpen();
    });

    // Close to trigger reconnect (use code 1006 for abnormal closure)
    act(() => {
      ws.simulateClose(1006, "Connection lost");
    });
    act(() => {
      vi.advanceTimersByTime(100);
    });

    // Reconnected WebSocket opens successfully
    ws = MockWebSocket.instances[MockWebSocket.instances.length - 1];
    act(() => {
      ws.simulateOpen();
    });

    // Close again - should still be able to reconnect (counter was reset)
    act(() => {
      ws.simulateClose(1006, "Connection lost");
    });
    act(() => {
      vi.advanceTimersByTime(100);
    });

    // Should have created another WebSocket
    expect(MockWebSocket.instances.length).toBeGreaterThanOrEqual(3);
    randomSpy.mockRestore();
  });
});
