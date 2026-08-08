<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/state';
  import { AGENTS, EDGES, VIEWBOX } from '$lib/magnetic/topology';
  import { restNodes } from '$lib/magnetic/homes';
  import { initLivingGraph } from '$lib/magnetic/engine';
  import type { EngineHandle } from '$lib/magnetic/engine';
  import { startEventFeed, startMockFeed } from '$lib/magnetic/event-feed';
  import type { FeedHandle } from '$lib/magnetic/event-feed';

  // Rest homes are a pure function of the roster — stable per project.
  const HOMES = restNodes(AGENTS);
  const NODES = Object.entries(AGENTS);
  const EDGES_LIST = Object.entries(EDGES);

  let projectId = $derived(page.url.searchParams.get('project') ?? '');
  let root: HTMLElement;
  let engine: EngineHandle | null = null;
  let feed: FeedHandle | null = null;
  let mode = $state<'mock' | 'live'>('mock');
  let connected = $state(false);
  let feedError = $state<string | null>(null);
  let liveSupported = $derived(projectId.length > 0);

  function teardown() {
    feed?.close(); feed = null;
    engine?.destroy(); engine = null;
  }

  function start() {
    if (!engine) return;
    feed?.close(); feed = null;
    connected = false; feedError = null;
    if (mode === 'live' && liveSupported) {
      feed = startEventFeed(projectId, engine, {
        onConnect: () => { connected = true; },
        onError: (e) => { feedError = e.message; },
      });
    } else {
      mode = 'live' === mode && !liveSupported ? 'mock' : mode;
      feed = startMockFeed(engine);
      connected = true;
    }
  }

  onMount(() => {
    if (!root) return;
    engine = initLivingGraph(root);
    // default to live if a project is selected, else the demo mock.
    mode = liveSupported ? 'live' : 'mock';
    start();
    return teardown;
  });

  function switchMode(next: 'mock' | 'live') {
    if (next === mode) return;
    mode = next;
    start();
  }
</script>

<svelte:head><title>Live Magnetic Graph</title></svelte:head>

<section class="mag-page">
  <header class="mag-bar">
    <div class="mag-title">
      <h2>Magnetic Clusters</h2>
      <span class="mag-sub">live agent activity → emergent graph motion</span>
    </div>
    <div class="mag-controls">
      <span class="dot" class:on={connected} title={feedError ?? (connected ? 'feed live' : 'connecting…')}></span>
      <div class="seg">
        <button class:on={mode === 'mock'} onclick={() => switchMode('mock')}>Demo</button>
        <button class:on={mode === 'live'} disabled={!liveSupported}
          title={liveSupported ? 'Drive from live SSE' : 'select a project (?project=)'} onclick={() => switchMode('live')}>Live</button>
      </div>
      <label class="rm"><input type="checkbox" data-rm /> reduced motion</label>
      <button class="play" data-action="play">Pause</button>
    </div>
  </header>

  {#if mode === 'mock'}
    <p class="banner">Demo mode — the graph is driven by a scripted SSE fixture (no backend required). Select a project and switch to <b>Live</b> to drive it from the real <code>/projects/:id/events/stream</code> SSE feed.</p>
  {:else if feedError}
    <p class="banner warn">Live feed error: {feedError}</p>
  {/if}

  <div class="mag-graph" bind:this={root}>
    <svg class="map" viewBox={`0 0 ${VIEWBOX.w} ${VIEWBOX.h}`} preserveAspectRatio="xMidYMid meet" role="img" aria-label="Live magnetic agent graph">
      <defs>
        <marker id="arrow-stub" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse"><path d="M0,0 L10,5 L0,10 z" class="mk-dim" /></marker>
        <marker id="arrow-now-stub" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="8" markerHeight="8" orient="auto-start-reverse"><path d="M0,0 L10,5 L0,10 z" class="mk-on" /></marker>
      </defs>

      {#each EDGES_LIST as [id, e] (id)}
        <path class="edge" class:rw={e.kind === 'rw'} data-edge={id} data-kind={e.kind} />
        <text class="elabel" data-elabel={id}>{e.label}</text>
      {/each}

      {#each NODES as [id, a] (id)}
        <g class="node" data-node={id} transform={`translate(${HOMES[id].x},${HOMES[id].y})`} style={`--role: var(--h-${a.hue})`} role="img" aria-label={`${a.role}: ${a.sub}`}>
          <circle class="ring" r="34" />
          <text class="role" y="-2">{a.role}</text>
          <text class="sub" y="14">{a.sub}</text>
        </g>
      {/each}
    </svg>

    <aside class="detail empty" data-detail><span>Wiring the live feed…</span></aside>
    <div class="caption" data-caption><span class="k">Settled.</span> <span class="c">Agents rest at home until work activates.</span></div>
    <div class="counter" data-count>0 ACTIVE</div>
    <div class="progress"><div class="fill" data-fill style="width:0"></div></div>
    <div class="summary" data-summary hidden>
      <h3>Knowledge map</h3>
      <p>Reduced-motion view. Cards accumulate as evidence of completed work; edges show the handoff graph that produced them.</p>
    </div>
  </div>
</section>

<style>
  :global(:root) {
    --h-user: #93c5fd; --h-intake: #60a5fa; --h-architect: #a78bfa; --h-builder: #4ade80;
    --h-security: #f87171; --h-qa: #fbbf24; --h-devops: #fb923c; --h-human: #e879f9; --h-prod: #34d399;
  }

  .mag-page { max-width: 1280px; margin: 0 auto; }

  .mag-bar {
    display: flex; align-items: center; justify-content: space-between; gap: 1rem;
    margin-bottom: 0.75rem; flex-wrap: wrap;
  }
  .mag-title h2 { margin: 0; font-size: 1.1rem; }
  .mag-sub { color: #888; font-size: 0.78rem; }

  .mag-controls { display: flex; align-items: center; gap: 0.6rem; font-size: 0.8rem; }
  .dot { width: 9px; height: 9px; border-radius: 50%; background: #555; }
  .dot.on { background: var(--color-success); box-shadow: 0 0 8px var(--color-success); }
  .seg { display: inline-flex; border: 1px solid #333; border-radius: 6px; overflow: hidden; }
  .seg button { background: #1a1a1a; color: #aaa; border: 0; padding: 0.3rem 0.7rem; cursor: pointer; font-size: 0.78rem; }
  .seg button.on { background: #2a4a2a; color: #fff; }
  .seg button:disabled { opacity: 0.4; cursor: not-allowed; }
  .rm { color: #888; display: inline-flex; align-items: center; gap: 0.3rem; cursor: pointer; }
  .play { background: #1a1a1a; color: #ddd; border: 1px solid #333; border-radius: 6px; padding: 0.3rem 0.8rem; cursor: pointer; font-size: 0.78rem; }

  .banner { background: #1a2a1a; border: 1px solid #2a4a2a; color: #bfe; padding: 0.5rem 0.8rem; border-radius: 6px; font-size: 0.8rem; margin: 0 0 0.75rem; }
  .banner.warn { background: #2a1a1a; border-color: #4a2a2a; color: #fcc; }
  .banner code { background: #000; padding: 0.05em 0.35em; border-radius: 3px; }

  .mag-graph {
    position: relative; background: #0b0d10; border: 1px solid #1d2024; border-radius: 10px; overflow: hidden;
  }
  .mag-graph svg.map { width: 100%; height: clamp(360px, 62vh, 760px); display: block; }

  /* The engine toggles element classes (.on/.now/.lit/.wait/.fresh) and mounts
     cards from JS, so all svg-internal element styles are :global under .mag-graph
     (Svelte would otherwise strip them as "unused" in the production build). */
  .mag-graph :global(.edge) { fill: none; stroke: #2a2f36; stroke-width: 1.5; transition: stroke 0.3s, stroke-width 0.3s; }
  .mag-graph :global(.edge.lit) { stroke: #3a4250; }
  .mag-graph :global(.edge.now) { stroke: #7dd3fc; stroke-width: 2.4; filter: drop-shadow(0 0 4px rgba(125,211,252,0.5)); }
  .mag-graph :global(.edge.rw.now) { stroke-dasharray: 6 5; }
  .mag-graph :global(.mk-dim) { fill: #3a4250; }
  .mag-graph :global(.mk-on) { fill: #7dd3fc; }
  .mag-graph :global(.elabel) { fill: #6b7280; font-size: 12px; opacity: 0; transition: opacity 0.3s; pointer-events: none; }
  .mag-graph :global(.elabel.on) { opacity: 1; }

  .mag-graph :global(.node) { cursor: pointer; }
  .mag-graph :global(.node .ring) { fill: #131721; stroke: var(--role, #2c3340); stroke-width: 2; transition: stroke 0.25s, filter 0.25s; }
  .mag-graph :global(.node .role) { fill: #cbd5e1; font-size: 11px; font-weight: 600; text-anchor: middle; pointer-events: none; }
  .mag-graph :global(.node .sub) { fill: #6b7280; font-size: 9px; text-anchor: middle; pointer-events: none; }
  .mag-graph :global(.node.on .ring) { fill: #0f1a14; stroke: var(--role); filter: drop-shadow(0 0 10px var(--role)); }
  .mag-graph :global(.node.on .role) { fill: #fff; }
  .mag-graph :global(.node.wait .ring) { stroke: var(--color-warn); }

  .mag-graph :global(.card rect) { fill: rgba(20,24,30,0.92); stroke: #555; stroke-width: 1.5; }
  .mag-graph :global(.card .cat) { fill: #93a1b5; font-size: 9px; font-weight: 700; letter-spacing: 0.5px; }
  .mag-graph :global(.card .ln) { fill: #d6dde6; font-size: 11px; }
  .mag-graph :global(.card.fresh rect) { stroke-width: 2.5; filter: drop-shadow(0 0 8px rgba(125,211,252,0.6)); }
  .mag-graph :global(.card[data-hue="builder"] rect) { stroke: var(--h-builder); }
  .mag-graph :global(.card[data-hue="intake"] rect) { stroke: var(--h-intake); }
  .mag-graph :global(.card[data-hue="architect"] rect) { stroke: var(--h-architect); }
  .mag-graph :global(.card[data-hue="security"] rect) { stroke: var(--h-security); }
  .mag-graph :global(.card[data-hue="qa"] rect) { stroke: var(--h-qa); }
  .mag-graph :global(.card[data-hue="devops"] rect) { stroke: var(--h-devops); }
  .mag-graph :global(.card[data-hue="prod"] rect) { stroke: var(--h-prod); }

  .detail {
    position: absolute; top: 0.75rem; left: 0.75rem; width: 250px; max-height: 60%; overflow-y: auto;
    background: rgba(15,17,21,0.92); border: 1px solid #222; border-radius: 8px; padding: 0.6rem 0.7rem;
    font-size: 0.78rem; color: #aab; backdrop-filter: blur(4px);
  }
  .detail.empty { color: #667; font-style: italic; }
  .detail :global(.who) { display: flex; flex-direction: column; margin-bottom: 0.4rem; border-left: 3px solid var(--role, #444); padding-left: 0.5rem; }
  .detail :global(.name) { color: #e6edf3; font-weight: 600; }
  .detail :global(.role-sub) { color: #778; font-size: 0.72rem; }
  .detail :global(.obj) { color: #bcc; }
  .detail :global(.chips) { display: flex; flex-wrap: wrap; gap: 0.25rem; margin-top: 0.35rem; }
  .detail :global(.chip) { background: #1d2024; color: #9ab; padding: 0.05rem 0.4rem; border-radius: 4px; font-size: 0.68rem; }
  .detail :global(.chip.state) { background: #243; color: #7e7; }

  .caption {
    position: absolute; bottom: 0.75rem; left: 0.75rem; right: 0.75rem;
    background: rgba(15,17,21,0.9); border: 1px solid #222; border-radius: 8px; padding: 0.5rem 0.8rem;
    font-size: 0.82rem; color: #ccd; backdrop-filter: blur(4px);
  }
  .caption :global(.k) { color: #7dd3fc; font-weight: 600; }

  .counter { position: absolute; top: 0.75rem; right: 0.75rem; background: rgba(15,17,21,0.9); border: 1px solid #222; border-radius: 6px; padding: 0.25rem 0.6rem; font-size: 0.72rem; color: #7e7; font-weight: 600; }
  .progress { position: absolute; top: 0; left: 0; right: 0; height: 2px; background: #1a1d22; }
  .progress .fill { height: 100%; background: linear-gradient(90deg, var(--h-architect), var(--h-builder)); transition: width 0.4s; }

  .summary { position: absolute; top: 50%; left: 50%; transform: translate(-50%, -50%); width: 80%; max-width: 520px; background: rgba(15,17,21,0.95); border: 1px solid #222; border-radius: 8px; padding: 1rem; color: #aab; text-align: center; }
</style>
