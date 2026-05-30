/**
 * Vitest setup — runs before each test file
 * Provides global jsdom globals only (localStorage, EventSource).
 * SvelteKit virtual module mocks ($app/navigation, $lib/api/client) must
 * be declared in each test file at top level so Vitest hoists them correctly.
 */
import { vi } from 'vitest';

// ---------------------------------------------------------------------------
// Mock localStorage — used by $lib/api/client getHeaders()
// ---------------------------------------------------------------------------
const localStorageMock = {
  getItem: vi.fn().mockReturnValue(null),
  setItem: vi.fn(),
  removeItem: vi.fn(),
  clear: vi.fn(),
};
Object.defineProperty(global, 'localStorage', {
  value: localStorageMock,
  writable: true,
  configurable: true,
});

// ---------------------------------------------------------------------------
// EventSource — not defined in jsdom; SSE uses it in createSSEConnection
// ---------------------------------------------------------------------------
class MockEventSource {
  url: string;
  readyState = 0; // CONNECTING
  static CONNECTING = 0;
  static OPEN = 1;
  static CLOSED = 2;

  onopen: (() => void) | null = null;
  onmessage: ((event: { data: string }) => void) | null = null;
  onerror: ((event: unknown) => void) | null = null;

  constructor(url: string) {
    this.url = url;
    setTimeout(() => this.onopen?.(), 0);
  }

  addEventListener(event: string, handler: (e: unknown) => void) {
    if (event === 'open') this.onopen = handler as () => void;
    if (event === 'message') this.onmessage = handler as (e: { data: string }) => void;
    if (event === 'error') this.onerror = handler as (e: unknown) => void;
  }

  close() {
    this.readyState = 2;
  }
}
Object.defineProperty(global, 'EventSource', {
  value: MockEventSource,
  writable: true,
  configurable: true,
});
