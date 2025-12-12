const API_BASE = import.meta.env.VITE_API_BASE || "";

type SessionExpiredCallback = () => void;

let onSessionExpired: SessionExpiredCallback | null = null;

export function setSessionExpiredCallback(
  callback: SessionExpiredCallback,
): void {
  onSessionExpired = callback;
}

function getToken(): string | null {
  return localStorage.getItem("netscope-token");
}

function getAuthHeaders(): HeadersInit {
  const token = getToken();
  if (token) {
    return {
      Authorization: `Bearer ${token}`,
    };
  }
  return {};
}

async function handleResponse<T>(
  response: Response,
  isAuthEndpoint: boolean,
): Promise<T> {
  if (response.status === 401 && !isAuthEndpoint) {
    onSessionExpired?.();
    throw new Error("Session expired");
  }

  if (!response.ok) {
    throw new Error(`API error: ${response.status}`);
  }

  return response.json();
}

export const api = {
  async get<T>(endpoint: string): Promise<T> {
    const isAuthEndpoint = endpoint.includes("/api/auth/");
    const response = await fetch(`${API_BASE}${endpoint}`, {
      headers: getAuthHeaders(),
    });
    return handleResponse<T>(response, isAuthEndpoint);
  },

  async post<T>(endpoint: string, body?: unknown): Promise<T> {
    const isAuthEndpoint = endpoint.includes("/api/auth/");
    const headers: HeadersInit = {
      ...getAuthHeaders(),
      "Content-Type": "application/json",
    };

    const response = await fetch(`${API_BASE}${endpoint}`, {
      method: "POST",
      headers,
      body: body ? JSON.stringify(body) : undefined,
    });
    return handleResponse<T>(response, isAuthEndpoint);
  },

  async put<T>(endpoint: string, body?: unknown): Promise<T> {
    const isAuthEndpoint = endpoint.includes("/api/auth/");
    const headers: HeadersInit = {
      ...getAuthHeaders(),
      "Content-Type": "application/json",
    };

    const response = await fetch(`${API_BASE}${endpoint}`, {
      method: "PUT",
      headers,
      body: body ? JSON.stringify(body) : undefined,
    });
    return handleResponse<T>(response, isAuthEndpoint);
  },

  async delete<T>(endpoint: string): Promise<T> {
    const isAuthEndpoint = endpoint.includes("/api/auth/");
    const response = await fetch(`${API_BASE}${endpoint}`, {
      method: "DELETE",
      headers: getAuthHeaders(),
    });
    return handleResponse<T>(response, isAuthEndpoint);
  },

  // Raw fetch for cases where you need the Response object
  async fetch(endpoint: string, init?: RequestInit): Promise<Response> {
    const headers: HeadersInit = {
      ...getAuthHeaders(),
      ...init?.headers,
    };

    return fetch(`${API_BASE}${endpoint}`, {
      ...init,
      headers,
    });
  },
};
