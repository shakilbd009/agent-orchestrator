/**
 * Unit tests for the Svelte 5 → testing-library/svelte v4 type adapter.
 *
 * The adapter's job is purely a *type-level* bridge:
 *   - A Svelte 5 `.svelte` file's default export is a `Component` (callable).
 *   - `@testing-library/svelte` v4's `render()` parameter type is
 *     `Constructor<SvelteComponent>`.
 *   - The library's runtime path (`svelte5` subpath) extracts
 *     `Component.default || Component` and hands it to Svelte 5 `mount()`,
 *     which accepts a function.
 *
 * So the adapter takes a module, returns the underlying Component function
 * re-typed as a class constructor, and the actual render path is exercised
 * end-to-end by `webhooks.test.ts` (which renders the real `+page.svelte`).
 *
 * These tests verify the structural contract of the adapter itself:
 *   1. It accepts a SvelteComponentModule and returns a callable value.
 *   2. The returned value passes through the library's
 *      `Component.default || Component` extraction unchanged.
 *   3. The adapter does not modify the underlying Component identity.
 */
import { describe, it, expect } from 'vitest';
import { wrapSvelte5Component } from './svelte5-class-component';
import type { Component } from 'svelte';

/** A trivial Svelte 5 Component shape for the test. */
const fakeComponent = (() => {}) as unknown as Component<Record<string, unknown>>;

/** A trivial Svelte 5 component module. */
const fakeModule = { default: fakeComponent };

describe('wrapSvelte5Component', () => {
  it('returns a callable value (the underlying Component function)', () => {
    const Klass = wrapSvelte5Component(fakeModule);
    expect(typeof Klass).toBe('function');
  });

  it('returns the Component that the library can mount (identity preserved)', () => {
    const Klass = wrapSvelte5Component(fakeModule);
    // The library does `Component.default || Component` to extract the
    // constructor; simulate that to confirm the adapter is shape-compatible
    // with the runtime path.
    const ComponentConstructor = (Klass as unknown as { default?: unknown }).default ?? Klass;
    expect(ComponentConstructor).toBe(fakeComponent);
  });

  it('exposes the module default through the returned value (so the library extraction path is happy)', () => {
    // The adapter's whole purpose is to bridge types; the test for webhooks
    // (which actually renders a real +page.svelte) is the integration check
    // that proves the type contract is satisfied end-to-end.
    const Klass = wrapSvelte5Component(fakeModule);
    // Adapter returns the function form; the library falls through
    // `Component.default || Component` to reach the function. Identity is
    // preserved.
    expect(Klass).toBe(fakeComponent);
  });
});
