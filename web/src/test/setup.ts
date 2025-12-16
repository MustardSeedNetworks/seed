/**
 * Test Setup and Utilities
 *
 * Purpose: Shared test configuration and mock utilities for Vitest.
 * Provides mock implementations of browser APIs (localStorage, fetch, WebSocket, etc.)
 * and common test helpers used across the test suite.
 *
 * Key Utilities:
 * - createMockLocalStorage(): Mock localStorage with spy functions
 * - Mock implementations: localStorage, fetch, WebSocket, matchMedia
 * - Test setup: beforeEach/afterEach hooks for test isolation
 * - Environment mocks: Simulates browser APIs for testing
 *
 * Usage:
 * ```typescript
 * import { createMockLocalStorage } from '../test/setup';
 *
 * beforeEach(() => {
 *   const localStorage = createMockLocalStorage();
 *   Object.defineProperty(window, 'localStorage', { value: localStorage });
 * });
 * ```
 *
 * Dependencies: vitest, @testing-library/jest-dom
 * Applied In: All test files via vitest configuration
 */

import "@testing-library/jest-dom";
import { vi, beforeEach, afterEach } from "vitest";

// ============================================================
// Mock react-i18next
// ============================================================
vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => {
      // Return common translations for tests
      const translations: Record<string, string> = {
        // Common namespace
        "app.title": "The Seed",
        "app.tagline": "Network Diagnostics by Mustard Seed Networks",
        "buttons.login": "Login",
        "buttons.logout": "Logout",
        "status.loggingIn": "Logging in...",
        "labels.username": "Username",
        "labels.password": "Password",
        "login.defaultCredentials": "Default: admin / seed",
        "status.error": "Error",
        "status.noDataAvailable": "No data available",
        "accessibility.openHelp": "Open help",
        "accessibility.openSettings": "Open settings",
        "accessibility.switchToLightMode": "Switch to light mode",
        "accessibility.switchToDarkMode": "Switch to dark mode",
        "accessibility.selectInterface": "Select network interface",
        // Cards namespace
        "system.title": "System Health",
        "system.cpu": "CPU",
        "system.memory": "Memory",
        "system.disk": "Disk",
        "system.uptime": "Uptime",
        "system.load1m": "Load (1m)",
        "system.goroutines": "Goroutines",
        "system.processMem": "Process Memory",
      };
      return translations[key] || key;
    },
    i18n: {
      language: "en",
      changeLanguage: vi.fn(),
    },
  }),
  Trans: ({ children }: { children: React.ReactNode }) => children,
  initReactI18next: { type: "3rdParty", init: vi.fn() },
}));

// ============================================================
// Mock localStorage
// ============================================================
export interface MockLocalStorage {
  getItem: ReturnType<typeof vi.fn>;
  setItem: ReturnType<typeof vi.fn>;
  removeItem: ReturnType<typeof vi.fn>;
  clear: () => void;
  _store: Record<string, string>;
}

export function createMockLocalStorage(): MockLocalStorage {
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
    get _store() {
      return store;
    },
  };
}

const mockLocalStorage = createMockLocalStorage();
Object.defineProperty(window, "localStorage", { value: mockLocalStorage });

// Export for use in tests
export { mockLocalStorage };

// ============================================================
// Mock fetch
// ============================================================
export const mockFetch = vi.fn();
global.fetch = mockFetch;

// Helper to create standard API responses
export function createMockResponse<T>(data: T, ok = true, status = 200) {
  return Promise.resolve({
    ok,
    status,
    json: () => Promise.resolve(data),
    text: () => Promise.resolve(JSON.stringify(data)),
    headers: new Headers(),
  });
}

// Helper to create error responses
export function createMockErrorResponse(status = 500, message = "Error") {
  return Promise.resolve({
    ok: false,
    status,
    json: () => Promise.resolve({ error: message }),
    text: () => Promise.resolve(message),
    headers: new Headers(),
  });
}

// ============================================================
// Mock WebSocket
// ============================================================
export class MockWebSocket {
  static instances: MockWebSocket[] = [];
  static CONNECTING = 0;
  static OPEN = 1;
  static CLOSING = 2;
  static CLOSED = 3;

  url: string;
  readyState: number = MockWebSocket.CONNECTING;
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

  static resetInstances() {
    MockWebSocket.instances = [];
  }
}

// ============================================================
// Mock window.location
// ============================================================
export function mockWindowLocation(overrides: Partial<Location> = {}) {
  const defaultLocation = {
    protocol: "http:",
    host: "localhost:8080",
    hostname: "localhost",
    port: "8080",
    pathname: "/",
    search: "",
    hash: "",
    href: "http://localhost:8080/",
    origin: "http://localhost:8080",
    ...overrides,
  };

  Object.defineProperty(window, "location", {
    value: defaultLocation,
    writable: true,
  });
}

// ============================================================
// Test lifecycle hooks
// ============================================================
beforeEach(() => {
  vi.clearAllMocks();
  mockLocalStorage.clear();
  mockFetch.mockReset();
  MockWebSocket.resetInstances();
});

afterEach(() => {
  vi.restoreAllMocks();
});

// ============================================================
// Common test data factories
// ============================================================

// Auth token factory
export function createMockAuthToken(expiresInSeconds = 3600): {
  token: string;
  expiry: number;
} {
  return {
    token: `test-token-${Date.now()}`,
    expiry: Math.floor(Date.now() / 1000) + expiresInSeconds,
  };
}

// Settings thresholds factory
export function createMockThresholds() {
  return {
    dns: { good: 50, warning: 100 },
    gateway: { good: 20, warning: 50 },
    link: { good: 1000, warning: 100 },
    wifi: { good: -50, warning: -70 },
  };
}

// Network interface factory
export function createMockInterface(
  name: string,
  type: "ethernet" | "wifi" | "loopback" = "ethernet",
  up = true,
  options?: {
    friendlyName?: string;
    description?: string;
    speedDisplay?: string;
    chipsetVendor?: string;
    chipsetModel?: string;
    hasTDR?: boolean;
    hasDOM?: boolean;
    score?: number;
  }
) {
  return {
    name,
    type,
    up,
    friendlyName: options?.friendlyName,
    description: options?.description,
    speedDisplay: options?.speedDisplay,
    chipsetVendor: options?.chipsetVendor,
    chipsetModel: options?.chipsetModel,
    hasTDR: options?.hasTDR ?? false,
    hasDOM: options?.hasDOM ?? false,
    score: options?.score ?? 0,
  };
}
