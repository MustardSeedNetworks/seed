import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useWebSocket, Message, CardUpdate } from "./useWebSocket";

// Mock WebSocket
class MockWebSocket {
  static instances: MockWebSocket[] = [];
  static OPEN = 1;
  static CLOSED = 3;

  url: string;
  readyState: number = 0;
  onopen: ((event: Event) => void) | null = null;
  onclose: ((event: CloseEvent) => void) | null = null;
  onerror: ((event: Event) => void) | null = null;
  onmessage: ((event: MessageEvent) => void) | null = null;
  closeWasCalled = false;
  sentMessages: string[] = [];

  constructor(url: string) {
    this.url = url;
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

  beforeEach(() => {
    originalWebSocket = global.WebSocket;
    global.WebSocket = MockWebSocket as unknown as typeof WebSocket;
    MockWebSocket.instances = [];
    vi.useFakeTimers();
  });

  afterEach(() => {
    global.WebSocket = originalWebSocket;
    vi.useRealTimers();
  });

  it("starts with disconnected status", () => {
    const { result } = renderHook(() => useWebSocket({ url: "/ws" }));

    // Initially it will be connecting since useEffect runs
    expect(["connecting", "disconnected"]).toContain(result.current.status);
  });

  it("connects to WebSocket on mount", () => {
    renderHook(() => useWebSocket({ url: "/ws", token: "test-token" }));

    expect(MockWebSocket.instances.length).toBe(1);
  });

  it("transitions to connected status on open", async () => {
    const { result } = renderHook(() =>
      useWebSocket({ url: "/ws", token: "test-token" }),
    );

    const ws = MockWebSocket.instances[0];
    act(() => {
      ws.simulateOpen();
    });

    expect(result.current.status).toBe("connected");
  });

  it("transitions to error status on WebSocket error", async () => {
    const { result } = renderHook(() =>
      useWebSocket({ url: "/ws", token: "test-token" }),
    );

    const ws = MockWebSocket.instances[0];
    act(() => {
      ws.simulateError();
    });

    expect(result.current.status).toBe("error");
  });

  it("transitions to disconnected status on close", async () => {
    const { result } = renderHook(() =>
      useWebSocket({ url: "/ws", token: "test-token" }),
    );

    const ws = MockWebSocket.instances[0];
    act(() => {
      ws.simulateOpen();
    });

    act(() => {
      ws.simulateClose();
    });

    expect(result.current.status).toBe("disconnected");
  });

  it("calls onMessage callback when receiving message", async () => {
    const onMessage = vi.fn();
    renderHook(() =>
      useWebSocket({ url: "/ws", token: "test-token", onMessage }),
    );

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
    renderHook(() =>
      useWebSocket({ url: "/ws", token: "test-token", onCardUpdate }),
    );

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
    const { result } = renderHook(() =>
      useWebSocket({ url: "/ws", token: "test-token" }),
    );

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
    const { result } = renderHook(() =>
      useWebSocket({ url: "/ws", token: "test-token" }),
    );

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

  it("cleans up WebSocket on unmount", () => {
    const { unmount } = renderHook(() =>
      useWebSocket({ url: "/ws", token: "test-token" }),
    );

    const ws = MockWebSocket.instances[0];
    act(() => {
      ws.simulateOpen();
    });

    unmount();

    expect(ws.closeWasCalled).toBe(true);
  });

  it("constructs correct WebSocket URL", () => {
    // Mock window.location
    Object.defineProperty(window, "location", {
      value: {
        protocol: "http:",
        host: "localhost:8080",
      },
      writable: true,
    });

    renderHook(() => useWebSocket({ url: "/ws/updates", token: "token" }));

    const ws = MockWebSocket.instances[0];
    expect(ws.url).toBe("ws://localhost:8080/ws/updates?token=token");
  });

  it("uses wss for https pages", () => {
    Object.defineProperty(window, "location", {
      value: {
        protocol: "https:",
        host: "example.com",
      },
      writable: true,
    });

    renderHook(() => useWebSocket({ url: "/ws", token: "token" }));

    const ws = MockWebSocket.instances[0];
    expect(ws.url).toBe("wss://example.com/ws?token=token");
  });

  it("uses provided URL directly if it starts with ws", () => {
    renderHook(() =>
      useWebSocket({ url: "ws://custom-server.com/socket", token: "token" }),
    );

    const ws = MockWebSocket.instances[0];
    expect(ws.url).toBe("ws://custom-server.com/socket?token=token");
  });

  it("attempts reconnection after close", async () => {
    renderHook(() =>
      useWebSocket({
        url: "/ws",
        token: "token",
        reconnectInterval: 1000,
        maxReconnectAttempts: 3,
      }),
    );

    const ws = MockWebSocket.instances[0];
    act(() => {
      ws.simulateOpen();
    });

    expect(MockWebSocket.instances.length).toBe(1);

    act(() => {
      ws.simulateClose();
    });

    // Fast forward to trigger reconnect
    act(() => {
      vi.advanceTimersByTime(1000);
    });

    // New WebSocket should be created
    expect(MockWebSocket.instances.length).toBe(2);
  });

  it("respects maxReconnectAttempts", async () => {
    renderHook(() =>
      useWebSocket({
        url: "/ws",
        token: "token",
        reconnectInterval: 100,
        maxReconnectAttempts: 2,
      }),
    );

    // First WebSocket
    let ws = MockWebSocket.instances[0];
    act(() => {
      ws.simulateClose();
    });

    // First reconnect attempt
    act(() => {
      vi.advanceTimersByTime(100);
    });
    ws = MockWebSocket.instances[MockWebSocket.instances.length - 1];
    act(() => {
      ws.simulateClose();
    });

    // Second reconnect attempt
    act(() => {
      vi.advanceTimersByTime(100);
    });
    ws = MockWebSocket.instances[MockWebSocket.instances.length - 1];
    act(() => {
      ws.simulateClose();
    });

    const currentCount = MockWebSocket.instances.length;

    // Should not reconnect anymore (max attempts reached)
    act(() => {
      vi.advanceTimersByTime(100);
    });

    expect(MockWebSocket.instances.length).toBe(currentCount);
  });
});
