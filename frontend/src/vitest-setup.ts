/**
 * Vitest setup — runs before each test file
 * Provides global jsdom globals only (localStorage).
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
