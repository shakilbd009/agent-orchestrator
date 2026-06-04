<script lang="ts">
  import { page } from '$app/state';
  import { onMount } from 'svelte';
  import { isClientPortalEnabled } from '$lib/client-portal/feature-flags';
  import { getReady } from '$lib/api/client';
  import type { ReadyResponse } from '$lib/api/orchestration';

  let { children } = $props();

  let enabled = $state(false);
  let readyState = $state<ReadyResponse | null>(null);
  let readyError = $state<string | null>(null);

  let currentProjectId = $derived(page.url.searchParams.get('project') ?? '');

  onMount(() => {
    enabled = isClientPortalEnabled();

    let interval: ReturnType<typeof setInterval>;

    (async () => {
      try {
        readyState = await getReady();
        readyError = null;
      } catch (e) {
        readyError = e instanceof Error ? e.message : String(e);
      }
      interval = setInterval(async () => {
        try {
          readyState = await getReady();
          readyError = null;
        } catch (e) {
          readyError = e instanceof Error ? e.message : String(e);
        }
      }, 30000);
    })();

    return () => clearInterval(interval);
  });

  const navItems = [
    { href: '/client-portal', label: 'Portfolio' },
    { href: '/client-portal/approvals', label: 'Approvals' },
    { href: '/client-portal/search', label: 'Search' },
  ];

  function getStatusColor(state: ReadyResponse | null, err: string | null): string {
    if (err) return 'var(--color-error)';
    if (!state) return 'var(--color-muted)';
    return state.status === 'ready' ? 'var(--color-success)' : 'var(--color-warn)';
  }

  function getStatusLabel(state: ReadyResponse | null, err: string | null): string {
    if (err) return `Ready check failed: ${err}`;
    if (!state) return 'Checking...';
    return state.status === 'ready' ? 'All systems ready' : `Degraded: ${Object.keys(state.subsystems ?? {}).join(', ')}`;
  }
</script>

<svelte:head>
  <title>Client Portal{currentProjectId ? ` — Project ${currentProjectId}` : ''}</title>
</svelte:head>

{#if !enabled}
  <div class="ff-gate">
    <div class="ff-gate-inner">
      <h1>Client Portal</h1>
      <p>
        <code>VITE_FF_ENABLE_CLIENT_PORTAL</code> is not enabled.
        Set it to <code>true</code> in your <code>.env</code> to access this capability.
      </p>
    </div>
  </div>
{:else}
  <div class="cp-layout">
    <header class="cp-header">
      <div class="cp-brand">
        <span class="brand-mark">◆</span>
        <span class="brand-name">Client Portal</span>
      </div>
      <nav class="cp-nav">
        {#each navItems as item}
          <a
            href="{item.href}{currentProjectId ? `?project=${currentProjectId}` : ''}"
            class="cp-nav-item"
            class:active={page.url.pathname === item.href || (item.href !== '/client-portal' && page.url.pathname.startsWith(item.href))}
          >
            {item.label}
          </a>
        {/each}
      </nav>
      <div class="cp-status">
        <span
          class="status-dot"
          style="background: {getStatusColor(readyState, readyError)}"
          title={getStatusLabel(readyState, readyError)}
        ></span>
        <span class="status-label">{getStatusLabel(readyState, readyError)}</span>
      </div>
    </header>

    <main class="cp-main">
      {@render children()}
    </main>
  </div>
{/if}

<style>
  :global(body) {
    font-family: system-ui, sans-serif;
    background: #0f0f0f;
    color: #e0e0e0;
  }

  .ff-gate {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 100vh;
    padding: 2rem;
  }

  .ff-gate-inner {
    max-width: 480px;
    text-align: center;
    background: #1a1a1a;
    border: 1px solid #333;
    border-radius: 8px;
    padding: 2rem;
  }

  .ff-gate h1 {
    font-size: 1.25rem;
    margin-bottom: 1rem;
  }

  .ff-gate p {
    color: #a0a0a0;
    line-height: 1.6;
  }

  code {
    background: #2a2a2a;
    padding: 0.15em 0.4em;
    border-radius: 4px;
    font-size: 0.9em;
  }

  .cp-layout {
    display: flex;
    flex-direction: column;
    min-height: 100vh;
  }

  .cp-header {
    display: flex;
    align-items: center;
    gap: 2rem;
    padding: 0.75rem 1.5rem;
    background: #1a1a1a;
    border-bottom: 1px solid #2a2a2a;
    position: sticky;
    top: 0;
    z-index: 100;
  }

  .cp-brand {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    flex-shrink: 0;
  }

  .brand-mark {
    color: #60a5fa;
    font-size: 1.1rem;
  }

  .brand-name {
    font-weight: 600;
    font-size: 0.95rem;
    color: #e0e0e0;
  }

  .cp-nav {
    display: flex;
    gap: 0.25rem;
    flex: 1;
  }

  .cp-nav-item {
    padding: 0.4rem 0.8rem;
    border-radius: 4px;
    text-decoration: none;
    color: #a0a0a0;
    font-size: 0.875rem;
    transition: background 0.15s, color 0.15s;
  }

  .cp-nav-item:hover {
    background: #2a2a2a;
    color: #e0e0e0;
  }

  .cp-nav-item.active {
    background: #2a2a2a;
    color: #fff;
    font-weight: 500;
  }

  .cp-status {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 0.8rem;
    color: #a0a0a0;
    flex-shrink: 0;
  }

  .status-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    flex-shrink: 0;
  }

  .status-label {
    max-width: 200px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .cp-main {
    flex: 1;
    padding: 1.5rem;
    max-width: 1100px;
    width: 100%;
    margin: 0 auto;
  }

  :global(:root) {
    --color-success: #4ade80;
    --color-warn: #fb923c;
    --color-error: #f87171;
    --color-muted: #666;
    --color-bg: #0f0f0f;
    --color-surface: #1a1a1a;
    --color-border: #2a2a2a;
    --color-border-hover: #3a3a3a;
    --color-text: #e0e0e0;
    --color-text-muted: #888;
    --color-text-subtle: #555;
    --font-mono: ui-monospace, 'Cascadia Code', 'Source Code Pro', Menlo, monospace;
  }

  :global(html) {
    background: var(--color-bg);
    color: var(--color-text);
    scroll-behavior: smooth;
  }
</style>