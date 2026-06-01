/**
 * Type adapter: present a Svelte 5 `+page.svelte` (or any `.svelte` file) module
 * in the shape `@testing-library/svelte` v4's `render()` type signature expects.
 *
 * Why this exists:
 * - Svelte 5 `.svelte` files export a `Component` — a *callable* interface,
 *   NOT a class constructor.
 * - `@testing-library/svelte` v4's `render()` parameter type is
 *   `Constructor<SvelteComponent>` — a class.
 * - At runtime, the library's svelte5 subpath calls
 *   `createClassComponent({ component: C, ... })` from `svelte/legacy`, which
 *   then calls Svelte 5 `mount(C, ...)` — and `mount` requires a function.
 * - The library's `render` does `Component.default || Component` to extract
 *   the default export from whatever you pass it, so we can hand it the whole
 *   module and the runtime path gets the function.
 *
 * The static types, however, are still a mismatch. `asClassComponent` from
 * `svelte/legacy` returns a class that satisfies the type signature, but its
 * returned class is *not* a valid input to Svelte 5 `mount` (which the library
 * calls internally), so using it breaks the runtime.
 *
 * Resolution: this adapter accepts a Svelte component module, re-exposes its
 * `.default` (the Component function) in a way that satisfies the library's
 * declared parameter type at the call site, *without* `any`. We assert
 * `unknown` only at the structural bridge between the Svelte 5 callable
 * interface and the Svelte 4 class type — no type checks anywhere else in
 * the test are weakened.
 *
 * Scope: test code only. Lives under `src/test-utils/`.
 */
import type { Component, SvelteComponent } from 'svelte';

/**
 * Minimal shape of a Svelte 5 component module — the SvelteKit/Vite default
 * export of `+page.svelte` is a `Component` callable; we don't need any other
 * fields for testing-library's `render` to be happy at runtime.
 */
export interface SvelteComponentModule<P extends Record<string, unknown> = Record<string, unknown>> {
  default: Component<P>;
}

/**
 * Return value shaped to satisfy `@testing-library/svelte` v4's
 * `Constructor<SvelteComponent>` parameter type. The runtime value is the
 * original Svelte 5 Component function (the library does
 * `Component.default || Component`, so either the module or the function
 * works; we use the function).
 */
export interface Svelte5ComponentClass<P extends Record<string, unknown>> {
  new (options: {
    target: Element | Document | ShadowRoot;
    props?: P;
    anchor?: Element;
    context?: Map<unknown, unknown>;
    intro?: boolean;
    hydrate?: boolean;
    recover?: boolean;
  }): SvelteComponent<P>;
}

/**
 * Adapt a Svelte 5 component module into a class-shaped value for the
 * testing-library type signature, while preserving the underlying Component
 * function so Svelte 5 `mount` works at runtime.
 *
 * Usage:
 *   const Page = wrapSvelte5Component(await import('./+page.svelte'));
 *   render(Page);
 */
export function wrapSvelte5Component<P extends Record<string, unknown>>(
  module: SvelteComponentModule<P>,
): Svelte5ComponentClass<P> {
  // The structural bridge: at the boundary between the Svelte 5 callable
  // `Component<P>` interface and the Svelte 4 `Constructor<SvelteComponent>`
  // shape, we re-state the function as a class constructor. The library's
  // runtime path is what actually runs the component, and it accepts the
  // function via `Component.default || Component` extraction in `pure.js`.
  //
  // The cast is `unknown` at this single boundary, not `any`, so no other
  // type checks in the codebase lose precision.
  return module.default as unknown as Svelte5ComponentClass<P>;
}
