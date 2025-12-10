import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import App from "./App";
import { SettingsProvider } from "./contexts/SettingsContext";
import { ReactNode } from "react";

// Mock localStorage
const mockLocalStorage = (() => {
  let store: Record<string, string> = {};
  return {
    getItem: vi.fn((key: string) => store[key] || null),
    setItem: vi.fn((key: string, value: string) => {
      store[key] = value;
    }),
    removeItem: vi.fn((key: string) => {
      delete store[key];
    }),
    clear: () => {
      store = {};
    },
    _getStore: () => store,
  };
})();

Object.defineProperty(window, "localStorage", {
  value: mockLocalStorage,
});

// Mock fetch
const mockFetch = vi.fn();
global.fetch = mockFetch;

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
  }

  simulateOpen() {
    this.readyState = MockWebSocket.OPEN;
    if (this.onopen) {
      this.onopen(new Event("open"));
    }
  }

  simulateClose() {
    this.readyState = MockWebSocket.CLOSED;
    if (this.onclose) {
      this.onclose({ code: 1000, reason: "", wasClean: true } as CloseEvent);
    }
  }

  simulateMessage(data: object) {
    if (this.onmessage) {
      this.onmessage({ data: JSON.stringify(data) } as MessageEvent);
    }
  }
}

// Wrapper with SettingsProvider
function createWrapper() {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <SettingsProvider>{children}</SettingsProvider>;
  };
}

function renderWithProviders(ui: React.ReactElement) {
  return render(ui, { wrapper: createWrapper() });
}

describe("App", () => {
  let originalWebSocket: typeof WebSocket;

  beforeEach(() => {
    mockLocalStorage.clear();
    vi.clearAllMocks();
    originalWebSocket = global.WebSocket;
    global.WebSocket = MockWebSocket as unknown as typeof WebSocket;
    MockWebSocket.instances = [];

    // Default API mocks
    mockFetch.mockImplementation((url: string) => {
      if (url.includes("/api/settings")) {
        return Promise.resolve({
          ok: true,
          json: () =>
            Promise.resolve({
              thresholds: {
                dns: { good: 50, warning: 100 },
                gateway: { good: 20, warning: 50 },
                link: { good: 1000, warning: 100 },
                wifi: { good: -50, warning: -70 },
              },
            }),
        });
      }
      // Default response for other endpoints
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve({}),
      });
    });
  });

  afterEach(() => {
    global.WebSocket = originalWebSocket;
    vi.restoreAllMocks();
  });

  describe("unauthenticated state", () => {
    it("renders login form when not authenticated", () => {
      renderWithProviders(<App />);

      expect(screen.getByText("NetScope")).toBeInTheDocument();
      expect(screen.getByText("Network Diagnostic Tool")).toBeInTheDocument();
      expect(screen.getByPlaceholderText("admin")).toBeInTheDocument();
      expect(screen.getByPlaceholderText("••••••••")).toBeInTheDocument();
      expect(
        screen.getByRole("button", { name: /login/i }),
      ).toBeInTheDocument();
    });

    it("shows default credentials hint", () => {
      renderWithProviders(<App />);

      expect(
        screen.getByText(/Default: admin \/ netscope/i),
      ).toBeInTheDocument();
    });

    it("handles login form submission", async () => {
      mockFetch.mockImplementation((url: string) => {
        if (url.includes("/api/auth/login")) {
          return Promise.resolve({
            ok: true,
            json: () =>
              Promise.resolve({
                token: "test-token",
                expires: Math.floor(Date.now() / 1000) + 3600,
              }),
          });
        }
        if (url.includes("/api/settings")) {
          return Promise.resolve({
            ok: true,
            json: () => Promise.resolve({ thresholds: {} }),
          });
        }
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({}),
        });
      });

      renderWithProviders(<App />);

      const usernameInput = screen.getByPlaceholderText("admin");
      const passwordInput = screen.getByPlaceholderText("••••••••");
      const loginButton = screen.getByRole("button", { name: /login/i });

      fireEvent.change(usernameInput, { target: { value: "admin" } });
      fireEvent.change(passwordInput, { target: { value: "netscope" } });
      fireEvent.click(loginButton);

      await waitFor(() => {
        expect(mockFetch).toHaveBeenCalledWith(
          expect.stringContaining("/api/auth/login"),
          expect.any(Object),
        );
      });
    });

    it("shows error message on login failure", async () => {
      mockFetch.mockImplementation((url: string) => {
        if (url.includes("/api/auth/login")) {
          return Promise.resolve({
            ok: false,
          });
        }
        if (url.includes("/api/settings")) {
          return Promise.resolve({
            ok: true,
            json: () => Promise.resolve({ thresholds: {} }),
          });
        }
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({}),
        });
      });

      renderWithProviders(<App />);

      const usernameInput = screen.getByPlaceholderText("admin");
      const passwordInput = screen.getByPlaceholderText("••••••••");
      const loginButton = screen.getByRole("button", { name: /login/i });

      fireEvent.change(usernameInput, { target: { value: "admin" } });
      fireEvent.change(passwordInput, { target: { value: "wrong" } });
      fireEvent.click(loginButton);

      await waitFor(() => {
        expect(screen.getByRole("alert")).toBeInTheDocument();
      });
    });

    it("disables login button while loading", async () => {
      let resolveLogin: (value: unknown) => void;
      mockFetch.mockImplementation((url: string) => {
        if (url.includes("/api/auth/login")) {
          return new Promise((resolve) => {
            resolveLogin = resolve;
          });
        }
        if (url.includes("/api/settings")) {
          return Promise.resolve({
            ok: true,
            json: () => Promise.resolve({ thresholds: {} }),
          });
        }
        return Promise.resolve({ ok: true, json: () => Promise.resolve({}) });
      });

      renderWithProviders(<App />);

      const usernameInput = screen.getByPlaceholderText("admin");
      const passwordInput = screen.getByPlaceholderText("••••••••");
      const loginButton = screen.getByRole("button", { name: /login/i });

      fireEvent.change(usernameInput, { target: { value: "admin" } });
      fireEvent.change(passwordInput, { target: { value: "password" } });
      fireEvent.click(loginButton);

      await waitFor(() => {
        expect(
          screen.getByRole("button", { name: /logging in/i }),
        ).toBeDisabled();
      });

      // Cleanup - resolve the pending promise
      resolveLogin!({
        ok: false,
      });
    });
  });

  describe("authenticated state", () => {
    beforeEach(() => {
      // Set up authenticated state
      const futureExpiry = Math.floor(Date.now() / 1000) + 3600;
      mockLocalStorage.setItem("netscope_token", "test-token");
      mockLocalStorage.setItem("netscope_token_expiry", String(futureExpiry));
      mockLocalStorage.setItem("netscope_username", "admin");
    });

    it("renders main dashboard when authenticated", async () => {
      renderWithProviders(<App />);

      await waitFor(() => {
        expect(screen.getByText("NetScope")).toBeInTheDocument();
      });

      // Should show logout button(s) - desktop and mobile versions may both render
      const logoutButtons = screen.getAllByRole("button", { name: /logout/i });
      expect(logoutButtons.length).toBeGreaterThan(0);
    });

    it("renders interface selector", async () => {
      mockFetch.mockImplementation((url: string) => {
        if (url.includes("/api/interfaces")) {
          return Promise.resolve({
            ok: true,
            json: () =>
              Promise.resolve([
                { name: "eth0", type: "ethernet", up: true },
                { name: "wlan0", type: "wifi", up: true },
              ]),
          });
        }
        if (url.includes("/api/settings")) {
          return Promise.resolve({
            ok: true,
            json: () => Promise.resolve({ thresholds: {} }),
          });
        }
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({}),
        });
      });

      renderWithProviders(<App />);

      await waitFor(() => {
        const select = screen.getByRole("combobox", {
          name: /select network interface/i,
        });
        expect(select).toBeInTheDocument();
      });
    });

    it("renders theme toggle button", async () => {
      renderWithProviders(<App />);

      await waitFor(() => {
        const themeButton = screen.getByRole("button", {
          name: /switch to (light|dark) mode/i,
        });
        expect(themeButton).toBeInTheDocument();
      });
    });

    it("renders settings button", async () => {
      renderWithProviders(<App />);

      await waitFor(() => {
        const settingsButton = screen.getByRole("button", {
          name: /open settings/i,
        });
        expect(settingsButton).toBeInTheDocument();
      });
    });

    it("renders help button", async () => {
      renderWithProviders(<App />);

      await waitFor(() => {
        const helpButton = screen.getByRole("button", { name: /open help/i });
        expect(helpButton).toBeInTheDocument();
      });
    });

    // Note: Version info and logout tests are excluded due to complexity
    // with WebSocket state and timing. Core functionality is tested above.
    // Additional integration tests can be added with a proper E2E framework.
  });
});

// Note: ConnectionStatus tests are covered in the useWebSocket.test.ts hook tests
// Full integration testing with WebSocket messages would trigger HealthCheckCard
// which requires proper pingResults initialization (see #221 for that fix)

describe("LoginForm input validation", () => {
  beforeEach(() => {
    mockLocalStorage.clear();
    vi.clearAllMocks();

    mockFetch.mockImplementation((url: string) => {
      if (url.includes("/api/settings")) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({ thresholds: {} }),
        });
      }
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve({}),
      });
    });
  });

  it("username and password inputs are required", () => {
    renderWithProviders(<App />);

    const usernameInput = screen.getByPlaceholderText("admin");
    const passwordInput = screen.getByPlaceholderText("••••••••");

    expect(usernameInput).toBeRequired();
    expect(passwordInput).toBeRequired();
  });

  it("password input has type password", () => {
    renderWithProviders(<App />);

    const passwordInput = screen.getByPlaceholderText("••••••••");
    expect(passwordInput).toHaveAttribute("type", "password");
  });

  it("shows form labels", () => {
    renderWithProviders(<App />);

    expect(screen.getByText("Username")).toBeInTheDocument();
    expect(screen.getByText("Password")).toBeInTheDocument();
  });
});
