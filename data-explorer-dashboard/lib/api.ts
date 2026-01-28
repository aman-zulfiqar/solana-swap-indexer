// lib/api.ts
const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "/api";
const API_KEY = process.env.NEXT_PUBLIC_API_KEY || "";

type ErrorResponse = { error: string; code: number };

// Generic fetch wrapper
async function fetchAPI<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
  // Ensure endpoint starts with /v1
  const normalizedEndpoint = endpoint.startsWith("/v1") ? endpoint : `/v1${endpoint}`;
  
  const res = await fetch(`${API_BASE_URL}${normalizedEndpoint}`, {
    headers: {
      "Content-Type": "application/json",
      "X-API-Key": API_KEY,
      ...(options.headers || {}),
    },
    ...options,
  });

  if (!res.ok) {
    const err: ErrorResponse = await res.json().catch(() => ({
      error: "Unknown error",
      code: res.status,
    }));
    throw new Error(`${err.code}: ${err.error}`);
  }

  return res.json();
}

// -------- Health --------
export function getHealth() {
    return fetchAPI<{ ok: boolean }>("/health");
  }

// -------- Echo (test) --------
export function postEcho(message: string) {
  return fetchAPI<{ echo: string }>("/echo", {
    method: "POST",
    body: JSON.stringify({ message }),
  });
}

// -------- Swaps --------
export function getRecentSwaps() {
    return fetchAPI<{ items: any[] }>("/swaps/recent");
  }

// -------- Prices --------
export function getPrice(token: string) {
  return fetchAPI<{ token: string; price: number }>(`/prices/${token}`);
}

// -------- AI Ask --------
export function askAI(question: string) {
  return fetchAPI<any>("/ai/ask", {
    method: "POST",
    body: JSON.stringify({ question }),
  });
}

// -------- Feature Flags --------
export type Flag = { key: string; value: any };

export function getFlagsList() {
  return fetchAPI<{ items: Flag[] }>("/flags");
}

export function createFlag(flag: Flag) {
  return fetchAPI<Flag>("/flags", {
    method: "POST",
    body: JSON.stringify(flag),
  });
}

export function getFlag(key: string) {
  return fetchAPI<Flag>(`/flags/${key}`);
}

export function updateFlag(key: string, value: any) {
  return fetchAPI<Flag>(`/flags/${key}`, {
    method: "PUT",
    body: JSON.stringify({ value }),
  });
}

export function deleteFlag(key: string) {
  return fetchAPI<{ success: boolean }>(`/flags/${key}`, {
    method: "DELETE",
  });
}
