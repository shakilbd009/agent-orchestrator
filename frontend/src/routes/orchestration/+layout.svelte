<script lang="ts">
  import { page } from '$app/state';
  import { onMount } from 'svelte';
  import { isOrchestrationEnabled } from '$lib/orchestration/feature-flags';
  import { getReady } from '$lib/api/client';
  import type { ReadyResponse } from '$lib/api/orchestration';
  import type { EventEnvelope } from '$lib/api/orchestration';

  let { children } = $props();

  let enabled = $state(false);
  let readyState = $state<ReadyResponse | null>(null);
  let readyError = $state<string | null>(null);
  let recentEvents = $state<EventEnvelope[]>([]);

  let currentProjectId = $derived(page.url.searchParams.get('project') ?? '');
  let eventCount = $state(0);

  let sseCleanup: (() => void) | null = null;

  // Wire SSE when project changes
  $effect(() => {
    const pid = currentProjectId;
    if (!pid || !enabled) {
      sseCleanup?.();
      sseCleanup = null;
      return;
    }

    // Reset cleanup - each run gets its own
    sseCleanup = null;

    import('$lib/api/client').then((mod) => {
      const conn = mod.createSSEConnection({
        projectId: pid,
        onEvent(envelope) {
          eventCount++;
          recentEvents = [envelope, ...recentEvents].slice(0, 50);
        },
        onError(err) {
          console.error('SSE error:', err);
        },
      });
      sseCleanup = conn.close;
    });

    return () => {
      sseCleanup?.();
      sseCleanup = null;
    };
  });

  onMount(() => {
    enabled = isOrchestrationEnabled();

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
    { href: '/orchestration', label: 'Projects' },
    { href: '/orchestration/board', label: 'Board' },
    { href: '/orchestration/decomposition', label: 'Decomposition' },
    { href: '/orchestration/gates', label: 'Gates' },
    { href: '/orchestration/webhooks', label: 'Webhooks' },
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
  <title>Orchestration{currentProjectId ? ` — Project ${currentProjectId}` : ''}</title>
</svelte:head>

{#if !enabled}
  <div class="ff-gate">
    <div class="ff-gate-inner">
      <h1>Platform Orchestration</h1>
      <p>
        <code>VITE_FF_ENABLE_PLATFORM_ORCHESTRATION</code> is not enabled.
        Set it to <code>true</code> in your <code>.env</code> to access this capability.
      </p>
    </div>
  </div>
{:else}
  <div class="orch-layout">
    <header class="orch-header">
      <nav class="orch-nav">
        {#each navItems as item}
          <a
            href="{item.href}{currentProjectId ? `?project=${currentProjectId}` : ''}"
            class="orch-nav-item"
            class:active={page.url.pathname === item.href || (item.href !== '/orchestration' && page.url.pathname.startsWith(item.href))}
          >
            {item.label}
          </a>
        {/each}
      </nav>

      <div class="orch-status">
        <span
          class="status-dot"
          style="background: {getStatusColor(readyState, readyError)}"
          title={getStatusLabel(readyState, readyError)}
        ></span>
        <span class="status-label">{getStatusLabel(readyState, readyError)}</span>
        {#if currentProjectId && eventCount > 0}
          <span class="event-badge" title="SSE events received this session">{eventCount} events</span>
        {/if}
      </div>
    </header>

    <main class="orch-main">
      {@render children()}
    </main>

    {#if recentEvents.length > 0}
      <aside class="event-panel">
        <h3>Recent Events <button class="clear-btn" onclick={() => { recentEvents = []; eventCount = 0; }}>clear</button></h3>
        <ul class="event-list">
          {#each recentEvents as event}
            <li class="event-item">
              <span class="event-topic">{event.topic}</span>
              <span class="event-time">{new Date(event.timestamp).toLocaleTimeString()}</span>
            </li>
          {/each}
        </ul>
      </aside>
    {/if}
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

  .orch-layout {
    display: flex;
    flex-direction: column;
    min-height: 100vh;
  }

  .orch-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0.75rem 1.5rem;
    background: #1a1a1a;
    border-bottom: 1px solid #2a2a2a;
    position: sticky;
    top: 0;
    z-index: 100;
  }

  .orch-nav {
    display: flex;
    gap: 0.25rem;
  }

  .orch-nav-item {
    padding: 0.4rem 0.8rem;
    border-radius: 4px;
    text-decoration: none;
    color: #a0a0a0;
    font-size: 0.875rem;
    transition: background 0.15s, color 0.15s;
  }

  .orch-nav-item:hover {
    background: #2a2a2a;
    color: #e0e0e0;
  }

  .orch-nav-item.active {
    background: #2a2a2a;
    color: #fff;
    font-weight: 500;
  }

  .orch-status {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 0.8rem;
    color: #a0a0a0;
  }

  .status-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    flex-shrink: 0;
  }

  .status-label {
    max-width: 240px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .event-badge {
    background: #2a2a2a;
    padding: 0.15em 0.5em;
    border-radius: 4px;
    font-size: 0.75em;
    color: #888;
  }

  .orch-main {
    flex: 1;
    padding: 1.5rem;
    max-width: 1200px;
    width: 100%;
    margin: 0 auto;
  }

  .event-panel {
    position: fixed;
    bottom: 1rem;
    right: 1rem;
    width: 300px;
    max-height: 320px;
    overflow-y: auto;
    background: #1a1a1a;
    border: 1px solid #333;
    border-radius: 8px;
    padding: 0.75rem;
    z-index: 200;
    box-shadow: 0 4px 24px rgba(0,0,0,0.5);
  }

  .event-panel h3 {
    font-size: 0.8rem;
    color: #888;
    margin: 0 0 0.5rem 0;
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .clear-btn {
    background: none;
    border: none;
    color: #666;
    cursor: pointer;
    font-size: 0.7rem;
    padding: 0;
  }

  .clear-btn:hover {
    color: #aaa;
  }

  .event-list {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }

  :global(:root) {
    --color-success: #4ade80;
    --color-warn: #fb923c;
    --color-error: #f87171;
    --color-muted: #666;
    --font-mono: ui-monospace, 'Cascadia Code', 'Source Code Pro', Menlo, monospace;
  }

  .event-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    font-size: 0.75rem;
    padding: 0.25rem 0;
    border-bottom: 1px solid #1e1e1e;
    animation: event-slide-in 0.2s ease-out;
  }

  @keyframes event-slide-in {
    from { opacity: 0; transform: translateY(-4px); }
    to { opacity: 1; transform: translateY(0); }
  }

  .event-topic {
    color: #93c5fd;
    font-family: var(--font-mono);
    font-size: 0.7rem;
  }

  .event-time {
    color: #555;
    font-size: 0.65rem;
    font-family: var(--font-mono);
  }
</style>